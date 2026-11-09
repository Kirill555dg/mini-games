// Package telemetry keeps bounded, anonymous application counters.
package telemetry

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const visitorLimit = 4096
const visitTTL = 30 * time.Minute

type Day struct {
	Date      string `json:"date"`
	PageViews uint64 `json:"pageViews"`
	Visits    uint64 `json:"visits"`
}
type bucket struct {
	second           int64
	requests, errors uint64
}
type Counters struct {
	mu               sync.Mutex
	ring             [60]bucket
	requests, errors uint64
	visitors         map[[32]byte]time.Time
	days             []Day
	path             string
	saved            time.Time
}

func New(path string) (*Counters, error) {
	c := &Counters{path: path, visitors: make(map[[32]byte]time.Time), days: []Day{}}
	if path == "" {
		return c, nil
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 16*1024+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 16*1024 || json.Unmarshal(data, &c.days) != nil || len(c.days) > 14 {
		return nil, os.ErrInvalid
	}
	return c, nil
}
func (c *Counters) Request(status int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now().Unix()
	b := &c.ring[now%60]
	if b.second != now {
		*b = bucket{second: now}
	}
	b.requests++
	c.requests++
	if status >= 500 {
		b.errors++
		c.errors++
	}
}
func (c *Counters) Visit(key [32]byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	date := now.UTC().Format("2006-01-02")
	if len(c.days) == 0 || c.days[len(c.days)-1].Date != date {
		c.days = append(c.days, Day{Date: date})
		if len(c.days) > 14 {
			c.days = c.days[len(c.days)-14:]
		}
	}
	day := &c.days[len(c.days)-1]
	day.PageViews++
	if last, ok := c.visitors[key]; !ok || now.Sub(last) >= visitTTL {
		day.Visits++
		if len(c.visitors) >= visitorLimit {
			for k, t := range c.visitors {
				if now.Sub(t) >= visitTTL {
					delete(c.visitors, k)
				}
			}
		}
		if len(c.visitors) >= visitorLimit {
			for k := range c.visitors {
				delete(c.visitors, k)
				break
			}
		}
	}
	c.visitors[key] = now
	if now.Sub(c.saved) >= time.Minute {
		_ = c.flush()
		c.saved = now
	}
}
func (c *Counters) Snapshot() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	requests, errors, active := uint64(0), uint64(0), 0
	for _, b := range c.ring {
		if now.Unix()-b.second < 60 {
			requests += b.requests
			errors += b.errors
		}
	}
	for _, seen := range c.visitors {
		if now.Sub(seen) < visitTTL {
			active++
		}
	}
	return map[string]any{"requests": c.requests, "serverErrors": c.errors, "rps": float64(requests) / 60, "errorsLastMinute": errors, "activeBrowsers30m": active, "days": append([]Day{}, c.days...)}
}
func (c *Counters) flush() error {
	if c.path == "" {
		return nil
	}
	data, err := json.Marshal(c.days)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(c.path), ".metrics-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), c.path)
}
func (c *Counters) Close() error { c.mu.Lock(); defer c.mu.Unlock(); return c.flush() }
