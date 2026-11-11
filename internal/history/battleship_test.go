package history

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Kirill555dg/mini-games/internal/engine"
)

func TestCompactBattleRecordSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	state := engine.State{Width: 8, Height: 8, Phase: engine.Finished, Winner: 0, Own: make([]int, 64), Enemy: make([]int, 64), Moves: []engine.Move{{Player: 0, Cell: 63, Hit: true}}}
	state.Enemy[63] = 2
	if store.Append(Record{FinishedAt: time.Now().UTC(), Code: "ABC123", Game: "battleship", Size: 8, Players: []string{"Первый", "Второй"}, Winner: 0, State: state}) != nil {
		t.Fatal("compact record fixture")
	}
	resumed, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	records := resumed.List()
	if len(records) != 1 || records[0].State.Width != 8 || records[0].State.Enemy[63] != 2 || records[0].Size != 8 {
		t.Fatal("compact battle history changed")
	}
}
