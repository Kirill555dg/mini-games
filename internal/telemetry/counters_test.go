package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAnonymousCountersAreBoundedAndSurviveRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	c, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < visitorLimit+10; n++ {
		var key [32]byte
		key[0] = byte(n)
		key[1] = byte(n >> 8)
		c.Visit(key)
	}
	if len(c.visitors) > visitorLimit {
		t.Fatal("visitor cache exceeded budget")
	}
	c.Request(200)
	c.Request(503)
	s := c.Snapshot()
	if s["requests"] != uint64(2) || s["serverErrors"] != uint64(1) {
		t.Fatal("request counters")
	}
	if c.Close() != nil {
		t.Fatal("persistence")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var days []Day
	if json.Unmarshal(data, &days) != nil || len(days) != 1 || days[0].PageViews != visitorLimit+10 {
		t.Fatal("aggregate persistence")
	}
	resumed, err := New(path)
	if err != nil || len(resumed.days) != 1 || len(resumed.visitors) != 0 {
		t.Fatal("only aggregate counters should persist")
	}
	c.days = make([]Day, 14)
	c.days[13].Date = time.Now().UTC().AddDate(0, 0, -1).Format(time.DateOnly)
	c.Visit([32]byte{255})
	if len(c.days) > 14 {
		t.Fatal("daily retention exceeded")
	}
}
func TestOversizedCountersAreRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	if os.WriteFile(path, make([]byte, 16*1024+1), 0600) != nil {
		t.Fatal("fixture")
	}
	if _, err := New(path); err == nil {
		t.Fatal("oversized file accepted")
	}
}
