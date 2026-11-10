package availability

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func at(t *testing.T, path string, now *time.Time) *Store {
	t.Helper()
	s, err := openAt(path, *now)
	if err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return *now }
	return s
}
func lastDay(s *Store) Day { days := s.Snapshot().Days; return days[len(days)-1] }
func TestObservationsSplitAtMidnightAndDoNotFillRestart(t *testing.T) {
	now := time.Date(2026, 1, 2, 23, 59, 30, 0, time.UTC)
	path := filepath.Join(t.TempDir(), Filename)
	s := at(t, path, &now)
	first := s.observedSince
	now = now.Add(SampleInterval)
	if s.Pulse() != nil {
		t.Fatal("sampling failed")
	}
	value := s.Snapshot()
	yesterday := value.Days[len(value.Days)-2]
	today := lastDay(s)
	if yesterday.ObservedSeconds != 30 || yesterday.UnknownSeconds != 86400-30 || today.ObservedSeconds != 30 || today.UnknownSeconds != 0 {
		t.Fatal("midnight observation lost or invented")
	}
	if value.Days[0].State != StateUnknown || value.Days[0].UnknownSeconds != 86400 {
		t.Fatal("unobserved past must stay unknown")
	}
	now = now.Add(5 * time.Minute)
	resumed := at(t, path, &now)
	if !resumed.observedSince.Equal(first) || lastDay(resumed).ObservedSeconds != 30 || lastDay(resumed).UnknownSeconds != 300 {
		t.Fatal("restart filled unobserved time")
	}
	now = now.Add(SampleInterval)
	if resumed.Pulse() != nil || lastDay(resumed).ObservedSeconds != 90 || lastDay(resumed).UnknownSeconds != 300 {
		t.Fatal("new samples must preserve restart gap")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > MaxBytes {
		t.Fatal("persistence")
	}
	for _, value := range [][]byte{[]byte("cookie"), []byte("player"), []byte("hostname")} {
		if bytes.Contains(data, value) {
			t.Fatal("unrelated data in status history")
		}
	}
}
func TestLongSchedulingGapAndClockAdjustmentRemainUnknown(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := at(t, "", &now)
	now = now.Add(5 * time.Minute)
	if s.Pulse() != nil || lastDay(s).ObservedSeconds != 0 {
		t.Fatal("long scheduling gap fabricated uptime")
	}
	now = now.Add(SampleInterval)
	if s.Pulse() != nil || lastDay(s).ObservedSeconds != 60 {
		t.Fatal("valid observation missing")
	}
	now = now.Add(-2 * time.Minute)
	if s.Pulse() != nil || lastDay(s).ObservedSeconds != 60 {
		t.Fatal("clock rollback changed coverage")
	}
	now = now.Add(3 * time.Minute)
	if s.Pulse() != nil || lastDay(s).ObservedSeconds != 120 {
		t.Fatal("clock recovery duplicated or lost observation")
	}
}
func TestHealthCountsAndThirtyDayRetention(t *testing.T) {
	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), Filename)
	s := at(t, path, &now)
	if lastDay(s).State != StateUnknown {
		t.Fatal("no observation shown as healthy")
	}
	now = now.Add(SampleInterval)
	if s.Pulse() != nil || lastDay(s).State != StateOK {
		t.Fatal("sample not shown")
	}
	s.Request(503)
	if lastDay(s).State != StateDown {
		t.Fatal("failed API observation not shown")
	}
	s.Request(200)
	if lastDay(s).State != StateDegraded || lastDay(s).Requests != 2 || lastDay(s).ServerErrors != 1 {
		t.Fatal("partial API failure not shown")
	}
	for n := 0; n < PeriodDays+10; n++ {
		now = now.AddDate(0, 0, 1)
		s.Request(200)
		now = now.Add(SampleInterval)
		if s.Pulse() != nil {
			t.Fatal("retention persistence")
		}
	}
	if len(s.days) != PeriodDays || len(s.Snapshot().Days) != PeriodDays {
		t.Fatal("retention exceeded")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > MaxBytes {
		t.Fatal("bounded persistence")
	}
	resumed := at(t, path, &now)
	if len(resumed.days) != PeriodDays || lastDay(resumed).Requests != 1 {
		t.Fatal("retained counters changed after restart")
	}
}
func TestInvalidStatusFilesArePreserved(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := [][]byte{[]byte("not json"), bytes.Repeat([]byte(" "), MaxBytes+1)}
	base := persisted{Version: fileVersion, ObservedSince: now, Days: []observedDay{{Date: "2026-01-01", Requests: 1}}}
	for _, mutate := range []func(*persisted){
		func(v *persisted) { v.Version++ },
		func(v *persisted) { v.ObservedSince = time.Time{} },
		func(v *persisted) { v.Days[0].ObservedSeconds = 86401 },
		func(v *persisted) { v.Days[0].ObservedSeconds = 60 },
		func(v *persisted) { v.Days[0].ServerErrors = 2 },
		func(v *persisted) { v.Days[0].ObservedUntil = now.AddDate(0, 0, 1).Unix() },
		func(v *persisted) { v.Days[0].Date = "invalid" },
		func(v *persisted) { v.Days = append(v.Days, v.Days[0]) },
	} {
		value := base
		value.Days = append([]observedDay(nil), base.Days...)
		mutate(&value)
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, data)
	}
	for _, data := range cases {
		path := filepath.Join(t.TempDir(), Filename)
		if os.WriteFile(path, data, 0600) != nil {
			t.Fatal("fixture")
		}
		if _, err := openAt(path, now); err == nil {
			t.Fatal("invalid status history accepted")
		}
		unchanged, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(unchanged, data) {
			t.Fatal("invalid file changed")
		}
	}
}

func TestPendingFileIsBoundedAndCannotRedirectWrites(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	s := at(t, path, &now)
	s.Request(200)
	if s.Pulse() != nil {
		t.Fatal("initial persistence")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(dir, "."+Filename+".pending")
	if os.Symlink(path, pending) != nil {
		t.Fatal("pending fixture")
	}
	now = now.Add(SampleInterval)
	if s.Pulse() == nil {
		t.Fatal("pending symlink accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed write modified committed history")
	}
	if os.Remove(pending) != nil {
		t.Fatal("remove fixture")
	}
	if os.WriteFile(pending, []byte("interrupted previous write"), 0600) != nil {
		t.Fatal("interruption fixture")
	}
	if s.Pulse() != nil {
		t.Fatal("pending file not replaced")
	}
	if _, err := os.Stat(pending); !os.IsNotExist(err) {
		t.Fatal("pending file retained after success")
	}
	if _, err := openAt(path, now); err != nil {
		t.Fatal("new history unreadable")
	}
}
func TestConcurrentRequestsAndSnapshots(t *testing.T) {
	s, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	const workers, perWorker = 8, 100
	for n := 0; n < workers; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for k := 0; k < perWorker; k++ {
				s.Request(200)
				s.Snapshot()
				s.Pulse()
			}
		}()
	}
	wg.Wait()
	if lastDay(s).Requests != workers*perWorker {
		t.Fatal("lost concurrent request counts")
	}
}
