package connectfour

import (
	"github.com/Kirill555dg/mini-games/internal/engine"
	"testing"
)

func TestGravityAndWin(t *testing.T) {
	g := New()
	for _, c := range []int{0, 1, 0, 1, 0, 1, 0} {
		s := g.View(-1)
		if g.Apply(s.Turn, engine.Action{Kind: "move", Cell: c}) != nil {
			t.Fatal("legal move")
		}
	}
	s := g.View(-1)
	if s.Winner != 0 || len(s.Line) != 4 || s.Board[5*Width] != 1 {
		t.Fatal("gravity or victory")
	}
}
func TestFullColumn(t *testing.T) {
	g := New()
	for n := 0; n < Height; n++ {
		if g.Apply(n%2, engine.Action{Kind: "move", Cell: 0}) != nil {
			t.Fatal("legal column fill")
		}
	}
	if g.Apply(0, engine.Action{Kind: "move", Cell: 0}) == nil {
		t.Fatal("full column accepted")
	}
}
