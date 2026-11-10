// Package availability retains bounded, anonymous application self-observations.
package availability

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
)

const (
	PeriodDays     = 30
	SampleInterval = time.Minute
	maxSampleGap   = SampleInterval + SampleInterval/2
	MaxBytes       = 32 * 1024
	Filename       = "status.json"
	fileVersion    = 1
	dayDuration    = 24 * time.Hour
	StateOK        = "ok"
	StateDegraded  = "degraded"
	StateDown      = "down"
	StateUnknown   = "unknown"
)

func States() []string { return []string{StateOK, StateDegraded, StateDown, StateUnknown} }

// Day describes observations, not an externally verified availability guarantee.
// UnknownSeconds includes every elapsed second not covered by consecutive samples.
type Day struct {
	Date            string `json:"date"`
	State           string `json:"state"`
	ObservedSeconds int64  `json:"observedSeconds"`
	UnknownSeconds  int64  `json:"unknownSeconds"`
	Requests        uint64 `json:"requests"`
	ServerErrors    uint64 `json:"serverErrors"`
}
type Snapshot struct {
	PeriodDays            int       `json:"periodDays"`
	SampleIntervalSeconds int64     `json:"sampleIntervalSeconds"`
	ObservedSince         time.Time `json:"observedSince"`
	Days                  []Day     `json:"days"`
}
type observedDay struct {
	Date            string `json:"date"`
	ObservedSeconds int64  `json:"observedSeconds"`
	ObservedUntil   int64  `json:"observedUntil"`
	Requests        uint64 `json:"requests"`
	ServerErrors    uint64 `json:"serverErrors"`
}
type persisted struct {
	Version       int           `json:"version"`
	ObservedSince time.Time     `json:"observedSince"`
	Days          []observedDay `json:"days"`
}
type Store struct {
	mu            sync.Mutex
	path          string
	now           func() time.Time
	last          time.Time
	observedSince time.Time
	days          []observedDay
}

func Open(path string) (*Store, error) { return openAt(path, time.Now().UTC()) }
func openAt(path string, now time.Time) (*Store, error) {
	s := &Store{path: path, now: time.Now, last: now, observedSince: now, days: []observedDay{}}
	if path == "" {
		return s, nil
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, errors.New("cannot open application status history")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	var saved persisted
	if err != nil || len(data) > MaxBytes || json.Unmarshal(data, &saved) != nil || saved.Version != fileVersion || saved.ObservedSince.IsZero() || len(saved.Days) > PeriodDays {
		return nil, errors.New("invalid application status history; existing file preserved")
	}
	previous := ""
	for _, day := range saved.Days {
		date, parseErr := time.Parse(time.DateOnly, day.Date)
		if parseErr != nil || day.Date <= previous || day.ObservedSeconds < 0 || day.ObservedSeconds > int64(dayDuration/time.Second) || day.ServerErrors > day.Requests {
			return nil, errors.New("invalid application status day; existing file preserved")
		}
		if day.ObservedUntil != 0 && (day.ObservedUntil < date.Unix() || day.ObservedUntil > date.Add(dayDuration).Unix()) {
			return nil, errors.New("invalid application observation time; existing file preserved")
		}
		if day.ObservedSeconds > 0 && (day.ObservedUntil == 0 || day.ObservedSeconds > day.ObservedUntil-date.Unix()) {
			return nil, errors.New("invalid application observation coverage; existing file preserved")
		}
		previous = day.Date
	}
	s.observedSince = saved.ObservedSince
	s.days = saved.Days
	// Deliberately start a new observation interval. Time while stopped is unknown.
	return s, nil
}

func (s *Store) day(now time.Time) *observedDay {
	date := now.UTC().Format(time.DateOnly)
	index := sort.Search(len(s.days), func(i int) bool { return s.days[i].Date >= date })
	if index < len(s.days) && s.days[index].Date == date {
		return &s.days[index]
	}
	s.days = append(s.days, observedDay{})
	copy(s.days[index+1:], s.days[index:])
	s.days[index] = observedDay{Date: date}
	return &s.days[index]
}
func (s *Store) trim(now time.Time) {
	cutoff := now.UTC().Truncate(dayDuration).AddDate(0, 0, 1-PeriodDays).Format(time.DateOnly)
	first := sort.Search(len(s.days), func(i int) bool { return s.days[i].Date >= cutoff })
	s.days = s.days[first:]
	if len(s.days) > PeriodDays {
		s.days = s.days[len(s.days)-PeriodDays:]
	}
}
func (s *Store) Request(status int) {
	if status < 100 || status > 599 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	day := s.day(now)
	day.Requests++
	if status >= 500 {
		day.ServerErrors++
	}
	s.trim(now)
}

// Pulse only covers a short interval between consecutive samples in this process.
// A restart, a long scheduling gap or a clock jump never fabricates healthy time.
func (s *Store) Pulse() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	elapsed := now.Sub(s.last)
	if elapsed > 0 && elapsed <= maxSampleGap {
		end := now.Truncate(time.Second)
		for start := s.last.Truncate(time.Second); start.Before(end); {
			dayEnd := start.Truncate(dayDuration).Add(dayDuration)
			segmentEnd := end
			if dayEnd.Before(segmentEnd) {
				segmentEnd = dayEnd
			}
			day := s.day(start)
			segmentStart := start.Unix()
			if day.ObservedUntil > segmentStart {
				segmentStart = day.ObservedUntil
			}
			if segmentEnd.Unix() > segmentStart {
				day.ObservedSeconds += segmentEnd.Unix() - segmentStart
				day.ObservedUntil = segmentEnd.Unix()
			}
			start = segmentEnd
		}
	}
	if now.After(s.last) {
		s.last = now
	}
	s.trim(now)
	return s.flush()
}
func (s *Store) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now().UTC()
	today := now.Truncate(dayDuration)
	out := Snapshot{PeriodDays: PeriodDays, SampleIntervalSeconds: int64(SampleInterval / time.Second), ObservedSince: s.observedSince, Days: make([]Day, 0, PeriodDays)}
	for n := 1 - PeriodDays; n <= 0; n++ {
		date := today.AddDate(0, 0, n)
		elapsed := int64(dayDuration / time.Second)
		if n == 0 {
			elapsed = int64(now.Sub(today) / time.Second)
		}
		item := Day{Date: date.Format(time.DateOnly), State: StateUnknown, UnknownSeconds: elapsed}
		for _, saved := range s.days {
			if saved.Date != item.Date {
				continue
			}
			item.ObservedSeconds = min(saved.ObservedSeconds, elapsed)
			item.UnknownSeconds = elapsed - item.ObservedSeconds
			item.Requests, item.ServerErrors = saved.Requests, saved.ServerErrors
			switch {
			case saved.Requests > 0 && saved.ServerErrors == saved.Requests:
				item.State = StateDown
			case saved.ServerErrors > 0:
				item.State = StateDegraded
			case item.ObservedSeconds > 0 || saved.Requests > 0:
				item.State = StateOK
			}
			break
		}
		out.Days = append(out.Days, item)
	}
	return out
}
func (s *Store) flush() error {
	if s.path == "" {
		return nil
	}
	data, err := json.Marshal(persisted{fileVersion, s.observedSince, s.days})
	if err != nil || len(data) > MaxBytes {
		return errors.New("application status history exceeds limit")
	}
	// One fixed pending file bounds storage even when a process is killed mid-write.
	pending := filepath.Join(filepath.Dir(s.path), "."+filepath.Base(s.path)+".pending")
	f, err := os.OpenFile(pending, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return errors.New("cannot create application status history")
	}
	defer os.Remove(pending)
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.New("cannot write application status history")
	}
	if os.Rename(pending, s.path) != nil {
		return errors.New("cannot replace application status history")
	}
	return nil
}
func (s *Store) Close() error { return s.Pulse() }
