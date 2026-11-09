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
func TestLegacyMigrationAndCorruptPreservation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "stats.json")
	legacy := `{"version":1,"serial":1,"games":[{"id":1,"code":"1234","x":"Один","o":"Два","result":"X","reason":"left","moves":[{"mark":"X","cell":0}]}]}`
	if os.WriteFile(p, []byte(legacy), 0600) != nil {
		t.Fatal("fixture")
	}
	s, err := Open(p)
	if err != nil || len(s.List()) != 1 || s.List()[0].Size != 3 {
		t.Fatal("legacy migration failed")
	}
	before, _ := os.ReadFile(p)
	if string(before) != legacy {
		t.Fatal("read rewrote history")
	}
	if s.Append(sample()) != nil {
		t.Fatal("append after migration")
	}
	var f file
	data, _ := os.ReadFile(p)
	if json.Unmarshal(data, &f) != nil || f.Version != 2 || len(f.Games) != 2 {
		t.Fatal("migration was not persisted on next result")
	}
	f.Games[0].State.Phase = engine.Playing
	data, _ = json.Marshal(f)
	_ = os.WriteFile(p, data, 0600)
	if _, err := Open(p); err == nil {
		t.Fatal("corrupt terminal state accepted")
	}
	after, _ := os.ReadFile(p)
	if string(after) != string(data) {
		t.Fatal("corrupt file overwritten")
	}
}
