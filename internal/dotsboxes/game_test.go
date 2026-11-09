package dotsboxes

import (
	"github.com/Kirill555dg/mini-games/internal/engine"
	"testing"
)

func TestBoxKeepsTurnAndMultiplayer(t *testing.T) {
	g := New(4)
	for _, cell := range []int{0, 4, horizontal, horizontal + 1} {
		s := g.View(-1)
		if g.Apply(s.Turn, engine.Action{Kind: "move", Cell: cell}) != nil {
			t.Fatal("legal edge")
		}
	}
	s := g.View(-1)
	if s.Scores[3] != 1 || s.Turn != 3 || s.Boxes[0] != 4 {
		t.Fatal("closed box must score and retain turn")
	}
	for cell := 0; cell < Edges; cell++ {
		s = g.View(-1)
		if s.Board[cell] == 0 {
			if g.Apply(s.Turn, engine.Action{Kind: "move", Cell: cell}) != nil {
				t.Fatal("remaining edge")
			}
		}
	}
	s = g.View(-1)
	sum := 0
	for _, n := range s.Scores {
		sum += n
	}
	if sum != Size*Size || s.Phase != engine.Finished || s.Winner == engine.NoWinner {
		t.Fatal("all boxes must be scored")
	}
}
