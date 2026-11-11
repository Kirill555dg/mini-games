package history

import (
	"encoding/json"
	"github.com/Kirill555dg/mini-games/internal/engine"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func sample() Record {
	return Record{FinishedAt: time.Now(), Code: "1234", Game: "tictactoe", Size: 3, Players: []string{"Один", "Два"}, Winner: 0, State: engine.State{Width: 3, Height: 3, Phase: engine.Finished, Winner: 0, Board: make([]int, 9)}}
}
func TestBoundedAtomicHistoryAndRestart(t *testing.T) {
	p := filepath.Join(t.TempDir(), "stats.json")
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < Limit+5; n++ {
		if s.Append(sample()) != nil {
			t.Fatal("append")
		}
	}
	s, err = Open(p)
	if err != nil {
		t.Fatal(err)
	}
	records := s.List()
	if len(records) != Limit || records[0].ID != 6 || records[Limit-1].ID != 105 {
		t.Fatal("wrong retained history")
	}
	i, _ := os.Stat(p)
	if i.Mode().Perm() != 0600 {
		t.Fatal("history is not restricted")
	}
	records[0].Players[0] = "Changed"
	if s.List()[0].Players[0] == "Changed" {
		t.Fatal("list aliases store")
	}
}
func TestRetiredMatchFormatsStartEmpty(t *testing.T) {
	for version := 1; version < fileVersion; version++ {
		p := filepath.Join(t.TempDir(), "stats.json")
		data, _ := json.Marshal(map[string]any{"version": version, "serial": 8, "games": []any{map[string]any{"retired": true}}})
		if err := os.WriteFile(p, data, 0600); err != nil {
			t.Fatal(err)
		}
		s, err := Open(p)
		if err != nil || len(s.List()) != 0 {
			t.Fatal("retired history was not discarded")
		}
		if err := s.Append(sample()); err != nil {
			t.Fatal(err)
		}
		data, _ = os.ReadFile(p)
		var current file
		if json.Unmarshal(data, &current) != nil || current.Version != fileVersion || len(current.Games) != 1 || current.Games[0].ID != 1 {
			t.Fatal("new history did not replace retired format")
		}
	}
}
func TestCorruptCurrentHistoryIsPreserved(t *testing.T) {
	p := filepath.Join(t.TempDir(), "stats.json")
	r := sample()
	r.ID = 1
	r.State.Phase = engine.Playing
	data, _ := json.Marshal(file{fileVersion, 1, []Record{r}})
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p); err == nil {
		t.Fatal("corrupt terminal state accepted")
	}
	after, _ := os.ReadFile(p)
	if string(after) != string(data) {
		t.Fatal("corrupt file overwritten")
	}
}
