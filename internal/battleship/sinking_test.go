package battleship

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Kirill555dg/mini-games/internal/engine"
)

func TestSinkingRevealsOnlyDestroyedShipAndProvenWater(t *testing.T) {
	g := New(DefaultSize)
	for player := 0; player < 2; player++ {
		if err := g.Apply(player, engine.Action{Kind: "deploy", Placements: classicDraft()}); err != nil {
			t.Fatal(err)
		}
	}
	for cell := 0; cell < classicFleet[0]; cell++ {
		if err := g.Apply(0, engine.Action{Kind: "move", Cell: cell}); err != nil {
			t.Fatal(err)
		}
		if cell < classicFleet[0]-1 {
			view := g.View(0)
			if len(view.Moves[len(view.Moves)-1].Sunk) != 0 || view.Enemy[4] != CellWater {
				t.Fatal("partial hit revealed the ship's extent or surrounding water")
			}
		}
	}
	view := g.View(0)
	last := view.Moves[len(view.Moves)-1]
	if !slices.Equal(last.Sunk, []int{0, 1, 2, 3}) || view.Turn != 0 || view.Phase != engine.Playing {
		t.Fatal("sinking did not retain the turn and reveal the destroyed ship")
	}
	for _, cell := range last.Sunk {
		if view.Enemy[cell] != CellSunk || g.View(1).Own[cell] != CellSunk || g.View(-1).Enemy[cell] != CellSunk {
			t.Fatal("players and spectator disagree about the sunk ship")
		}
	}
	water := slices.Clone(last.Water)
	slices.Sort(water)
	if !slices.Equal(water, []int{4, 10, 11, 12, 13, 14}) {
		t.Fatal("sinking inferred the wrong adjacent water")
	}
	for _, cell := range water {
		if view.Enemy[cell] != CellClear || g.View(1).Own[cell] != CellClear || g.View(-1).Enemy[cell] != CellClear {
			t.Fatal("deduced water differs between viewers")
		}
	}
	for cell, ship := range g.boards[1] {
		if ship == CellShip && !slices.Contains(last.Sunk, cell) && view.Enemy[cell] != CellWater {
			t.Fatal("living ship leaked")
		}
	}
	before := g.View(0)
	for _, cell := range append(last.Sunk, last.Water...) {
		if g.Apply(0, engine.Action{Kind: "move", Cell: cell}) == nil || !reflect.DeepEqual(before, g.View(0)) {
			t.Fatal("resolved target accepted a shot or changed the turn")
		}
	}
	view.Moves[len(view.Moves)-1].Sunk[0] = 99
	view.Moves[len(view.Moves)-1].Water[0] = 99
	clone := g.Clone().(*Game)
	clone.moves[len(clone.moves)-1].Sunk[0] = 98
	clone.moves[len(clone.moves)-1].Water[0] = 98
	if !reflect.DeepEqual(before, g.View(0)) {
		t.Fatal("snapshot or clone aliases public reveals")
	}
}

func TestSingleDeckCornerAndPreviouslyMissedWater(t *testing.T) {
	g := prepare(t)
	for player := 0; player < 2; player++ {
		if err := g.Apply(player, engine.Action{Kind: "ready"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, shot := range []struct{ player, cell int }{{0, 62}, {1, 62}, {0, 63}} {
		if err := g.Apply(shot.player, engine.Action{Kind: "move", Cell: shot.cell}); err != nil {
			t.Fatal(err)
		}
	}
	view := g.View(0)
	last := view.Moves[len(view.Moves)-1]
	if !slices.Equal(last.Sunk, []int{63}) || len(last.Water) != 2 || view.Enemy[62] != CellMiss || view.Enemy[54] != CellClear || view.Enemy[55] != CellClear {
		t.Fatal("corner sinking wrapped the field or rewrote an actual miss")
	}
}
