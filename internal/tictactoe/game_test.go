package tictactoe

import (
	"github.com/Kirill555dg/mini-games/internal/engine"
	"testing"
)

func TestClassicVictoryAndDraw(t *testing.T) {
	g := New()
	if g.View(-1).Width != Size || g.View(-1).WinLength != Size {
		t.Fatal("wrong classic geometry")
	}
	for i, cell := range []int{0, 3, 1, 4, 2} {
		if err := g.Apply(i%2, engine.Action{Kind: "move", Cell: cell}); err != nil {
			t.Fatal(err)
		}
	}
	if g.View(-1).Winner != 0 || g.View(-1).Phase != engine.Finished {
		t.Fatal("expected X victory")
	}
	if g.Apply(1, engine.Action{Kind: "move", Cell: 8}) == nil {
		t.Fatal("move after finish")
	}
	g = New()
	for i, cell := range []int{0, 1, 2, 4, 3, 5, 7, 6, 8} {
		if err := g.Apply(i%2, engine.Action{Kind: "move", Cell: cell}); err != nil {
			t.Fatal(err)
		}
	}
	if g.View(-1).Winner != engine.Draw || g.View(-1).Phase != engine.Finished {
		t.Fatal("expected a full-board draw")
	}
}
func TestRejectedMoveAndClone(t *testing.T) {
	g := New()
	if g.Apply(1, engine.Action{Kind: "move", Cell: 0}) == nil {
		t.Fatal("out of turn")
	}
	clone := g.Clone()
	_ = clone.Apply(0, engine.Action{Kind: "move", Cell: 0})
	if g.View(0).Board[0] != 0 {
		t.Fatal("clone aliases board")
	}
	v := g.View(0)
	v.Board[0] = 2
	if g.View(0).Board[0] != 0 {
		t.Fatal("snapshot aliases board")
	}
}
