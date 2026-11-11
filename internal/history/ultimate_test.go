package history

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Kirill555dg/mini-games/internal/engine"
	"github.com/Kirill555dg/mini-games/internal/ultimate"
)

func TestNestedMatchRoundTripAndDetachedReplay(t *testing.T) {
	game := ultimate.New()
	state := game.View(-1)
	for state.Phase != engine.Finished {
		found := false
		for cell, mark := range state.Board {
			region := (cell/state.Width/state.RegionSize)*(state.Width/state.RegionSize) + cell%state.Width/state.RegionSize
			if mark == 0 && state.Regions[region] == 0 && (state.ActiveBoard == nil || *state.ActiveBoard == region) {
				if err := game.Apply(state.Turn, engine.Action{Kind: "move", Cell: cell}); err != nil {
					t.Fatal(err)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatal("game without legal continuation")
		}
		state = game.View(-1)
	}
	path := filepath.Join(t.TempDir(), "stats.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Append(Record{FinishedAt: time.Now().UTC(), Code: "NINE09", Game: "ultimate", Players: []string{"Первый", "Второй"}, Winner: state.Winner, State: state}); err != nil {
		t.Fatal(err)
	}
	resumed, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	record := resumed.List()[0]
	if !reflect.DeepEqual(record.State, state) {
		t.Fatal("nested history changed after restart")
	}
	if record.State.Moves[0].NextBoard == nil || *record.State.Moves[0].NextBoard != 0 {
		t.Fatal("first destination lost")
	}
	*record.State.Moves[0].NextBoard = 8
	record.State.Regions[0] = 99
	if !reflect.DeepEqual(resumed.List()[0].State, state) {
		t.Fatal("history exposes mutable replay state")
	}
}
