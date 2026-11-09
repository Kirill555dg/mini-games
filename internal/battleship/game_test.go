package battleship

import (
	"github.com/Kirill555dg/mini-games/internal/engine"
	"testing"
)

func prepare(t *testing.T) *Game {
	t.Helper()
	g := New()
	for p := 0; p < 2; p++ {
		for _, cell := range []int{0, 16, 32, 48, 63} {
			if g.Apply(p, engine.Action{Kind: "place", Cell: cell}) != nil {
				t.Fatal("valid fleet placement")
			}
		}
	}
	return g
}
func TestPrivateFleetAndShotLifecycle(t *testing.T) {
	g := prepare(t)
	for _, viewer := range []int{-1, 0, 1} {
		s := g.View(viewer)
		for _, v := range s.Enemy {
			if v != 0 {
				t.Fatal("enemy fleet leaked")
			}
		}
		if viewer < 0 {
			for _, v := range s.Own {
				if v != 0 {
					t.Fatal("spectator fleet leaked")
				}
			}
		}
	}
	for p := 0; p < 2; p++ {
		if g.Apply(p, engine.Action{Kind: "ready"}) != nil {
			t.Fatal("ready")
		}
	}
	if g.Apply(1, engine.Action{Kind: "move", Cell: 0}) == nil {
		t.Fatal("wrong turn")
	}
	for i, v := range g.boards[1] {
		if v != 0 {
			if g.Apply(0, engine.Action{Kind: "move", Cell: i}) != nil {
				t.Fatal("hit")
			}
		}
	}
	if g.View(0).Winner != 0 || g.View(-1).Phase != engine.Finished {
		t.Fatal("fleet destruction must win")
	}
}
func TestPlacementAdjacencyAndRandomFleet(t *testing.T) {
	g := New()
	if g.Apply(0, engine.Action{Kind: "place", Cell: 0}) != nil {
		t.Fatal("first ship")
	}
	for _, cell := range []int{4, 8, 9, -1, 63} {
		if g.Apply(0, engine.Action{Kind: "place", Cell: cell}) == nil {
			t.Fatal("invalid or touching ship accepted")
		}
	}
	if g.Apply(0, engine.Action{Kind: "ready"}) == nil {
		t.Fatal("incomplete fleet accepted")
	}
	for n := 0; n < 20; n++ {
		if g.Apply(0, engine.Action{Kind: "shuffle"}) != nil {
			t.Fatal("random placement failed")
		}
		sum := 0
		for _, v := range g.boards[0] {
			sum += v
		}
		if sum != 12 || g.placed[0] != len(fleet) {
			t.Fatal("wrong fleet")
		}
	}
	clone := g.Clone()
	_ = clone.Apply(0, engine.Action{Kind: "clear"})
	if g.placed[0] != len(fleet) {
		t.Fatal("clone aliases fleet")
	}
}
