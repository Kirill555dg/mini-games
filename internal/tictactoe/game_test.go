package tictactoe

import (
	"fmt"
	"github.com/Kirill555dg/mini-games/internal/engine"
	"testing"
)

func TestBoardSizesAndWinningLines(t *testing.T) {
	for size := 3; size <= 7; size++ {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			g := New(size)
			k := size
			if k > 5 {
				k = 5
			}
			if g.View(-1).WinLength != k {
				t.Fatal("wrong victory length")
			}
			for _, d := range [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}} {
				b := make([]int, size*size)
				col := 0
				if d[1] < 0 {
					col = size - 1
				}
				for n := 0; n < k; n++ {
					b[n*d[0]*size+col+n*d[1]] = 1
				}
				if len(WinningLine(b, size, size, k, 1)) != k {
					t.Fatal("missed winning direction")
				}
			}
			b := make([]int, size*size)
			for n := 0; n < k; n++ {
				b[size-1+n] = 1
			}
			if len(WinningLine(b, size, size, k, 1)) != 0 {
				t.Fatal("line wrapped across row")
			}
			for n := 0; n < k; n++ {
				if g.Apply(0, engine.Action{Kind: "move", Cell: n}) != nil {
					t.Fatal("legal X move")
				}
				if n < k-1 && g.Apply(1, engine.Action{Kind: "move", Cell: size + n}) != nil {
					t.Fatal("legal O move")
				}
			}
			if g.View(-1).Winner != 0 || g.View(-1).Phase != engine.Finished {
				t.Fatal("expected X victory")
			}
			if g.Apply(1, engine.Action{Kind: "move", Cell: size*size - 1}) == nil {
				t.Fatal("move after finish")
			}
		})
	}
}
func TestRejectedMoveAndClone(t *testing.T) {
	g := New(3)
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
