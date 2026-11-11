package ultimate

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/Kirill555dg/mini-games/internal/engine"
)

func move(t *testing.T, g *Game, cell int) {
	t.Helper()
	if err := g.Apply(g.state.Turn, engine.Action{Kind: "move", Cell: cell}); err != nil {
		t.Fatal(err)
	}
}

func TestForcedBoardZeroValidationAndSnapshotIsolation(t *testing.T) {
	g := New()
	if len(g.state.Board) != 81 || len(g.state.Regions) != 9 || g.state.ActiveBoard != nil {
		t.Fatal("initial board")
	}
	move(t, g, 0)
	wire, _ := json.Marshal(g.View(-1))
	if !strings.Contains(string(wire), `"activeBoard":0`) || !strings.Contains(string(wire), `"nextBoard":0`) {
		t.Fatal("board zero was omitted")
	}
	before := g.View(-1)
	for _, action := range []engine.Action{{Kind: "move", Cell: 3}, {Kind: "move", Cell: 0}, {Kind: "move", Cell: -1}, {Kind: "move", Cell: 81}, {Kind: "ready", Cell: 1}} {
		if g.Apply(1, action) == nil || !reflect.DeepEqual(before, g.View(-1)) {
			t.Fatal("invalid action changed state")
		}
	}
	if g.Apply(0, engine.Action{Kind: "move", Cell: 1}) == nil {
		t.Fatal("out of turn")
	}
	snapshot := g.View(-1)
	snapshot.Board[0], snapshot.Regions[0] = 2, 2
	*snapshot.ActiveBoard, *snapshot.Moves[0].NextBoard = 7, 8
	if !reflect.DeepEqual(before, g.View(-1)) {
		t.Fatal("snapshot aliases mutable state")
	}
	clone := g.Clone().(*Game)
	move(t, clone, 1)
	if !reflect.DeepEqual(before, g.View(-1)) || *clone.state.ActiveBoard != 1 {
		t.Fatal("clone or next destination")
	}
}

func almostWon(g *Game, region int) {
	g.state.Board[globalCell(region, 0)] = 1
	g.state.Board[globalCell(region, 1)] = 1
	g.state.ActiveBoard = &region
}

func almostDrawn(g *Game, region int) {
	for cell, value := range []int{1, 2, 1, 1, 2, 2, 2, 1} {
		g.state.Board[globalCell(region, cell)] = value
	}
	g.state.ActiveBoard = &region
}

func TestLocalWinAndClosedDestination(t *testing.T) {
	for _, result := range []int{1, engine.Draw} {
		g := New()
		almostWon(g, 4)
		g.state.Regions[2] = result
		move(t, g, globalCell(4, 2))
		if g.state.Regions[4] != 1 || g.state.ActiveBoard != nil || g.state.Moves[0].RegionResult != 1 || g.state.Moves[0].NextBoard != nil {
			t.Fatal("closed destination did not release next player")
		}
		before := g.View(-1)
		if g.Apply(1, engine.Action{Kind: "move", Cell: globalCell(4, 3)}) == nil || !reflect.DeepEqual(before, g.View(-1)) {
			t.Fatal("won board accepted another move")
		}
		move(t, g, globalCell(0, 3))
		if g.state.ActiveBoard == nil || *g.state.ActiveBoard != 3 {
			t.Fatal("free choice did not restore normal sending rule")
		}
	}
	g := New()
	almostWon(g, 2)
	move(t, g, globalCell(2, 2))
	if g.state.ActiveBoard != nil {
		t.Fatal("sending to board won by this move must allow free choice")
	}
}

func TestDrawnBoardIsNeutralAndClosed(t *testing.T) {
	g := New()
	g.state.Regions[0], g.state.Regions[4] = 1, 1
	almostDrawn(g, 8)
	move(t, g, globalCell(8, 8))
	if g.state.Regions[8] != engine.Draw || g.state.Phase != engine.Playing || g.state.Winner != engine.NoWinner || g.state.ActiveBoard != nil {
		t.Fatal("drawn local board changed global result")
	}
	if g.state.Moves[0].RegionResult != engine.Draw {
		t.Fatal("draw closure missing from public replay")
	}
	if g.Apply(1, engine.Action{Kind: "move", Cell: globalCell(8, 0)}) == nil {
		t.Fatal("drawn board accepted play")
	}
}

func TestGlobalWinAndExhaustedBoards(t *testing.T) {
	g := New()
	g.state.Regions[0], g.state.Regions[1] = 1, 1
	almostWon(g, 2)
	move(t, g, globalCell(2, 2))
	if g.state.Phase != engine.Finished || g.state.Winner != 0 || !reflect.DeepEqual(g.state.RegionLine, []int{0, 1, 2}) || g.state.ActiveBoard != nil {
		t.Fatal("global victory")
	}
	snapshot := g.View(-1)
	snapshot.RegionLine[0] = 8
	if g.state.RegionLine[0] != 0 {
		t.Fatal("winning regions alias")
	}
	if g.Apply(1, engine.Action{Kind: "move", Cell: globalCell(3, 0)}) == nil {
		t.Fatal("move after global victory")
	}
	g = New()
	copy(g.state.Regions, []int{1, 2, 1, 1, 2, 2, 2, 1, 0})
	almostDrawn(g, 8)
	move(t, g, globalCell(8, 8))
	if g.state.Phase != engine.Finished || g.state.Winner != engine.Draw || len(g.state.RegionLine) != 0 {
		t.Fatal("all closed boards without winning row must draw")
	}
}

func TestLegalGamesTerminateAndAnnotateReplay(t *testing.T) {
	for seed := int64(0); seed < 64; seed++ {
		g := New()
		random := rand.New(rand.NewSource(seed))
		for steps := 0; g.state.Phase != engine.Finished; steps++ {
			if steps >= CellCount {
				t.Fatal("game exceeded board capacity")
			}
			legal := []int{}
			for cell, value := range g.state.Board {
				region := regionOf(cell)
				if value == Open && g.state.Regions[region] == Open && (g.state.ActiveBoard == nil || *g.state.ActiveBoard == region) {
					legal = append(legal, cell)
				}
			}
			if len(legal) == 0 {
				t.Fatal("playing game has no legal moves")
			}
			move(t, g, legal[random.Intn(len(legal))])
			recorded := g.state.Moves[len(g.state.Moves)-1]
			if !reflect.DeepEqual(recorded.NextBoard, g.state.ActiveBoard) {
				t.Fatal("replay destination differs from live state")
			}
			if recorded.RegionResult != Open && recorded.RegionResult != g.state.Regions[regionOf(recorded.Cell)] {
				t.Fatal("replay local result differs")
			}
		}
		if g.state.Winner == engine.NoWinner {
			t.Fatal("finished without outcome")
		}
	}
}
