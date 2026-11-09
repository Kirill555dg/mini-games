// Package connectfour implements gravity-based four-in-a-row.
package connectfour

import (
	"github.com/Kirill555dg/mini-games/internal/engine"
	"github.com/Kirill555dg/mini-games/internal/tictactoe"
)

const Width, Height, WinLength = 7, 6, 4

type Game struct{ state engine.State }

func New() *Game {
	return &Game{engine.State{Width: Width, Height: Height, WinLength: WinLength, Phase: engine.Playing, Winner: engine.NoWinner, Board: make([]int, Width*Height)}}
}
func (g *Game) View(_ int) engine.State { return engine.Copy(g.state) }
func (g *Game) Clone() engine.Game      { return &Game{engine.Copy(g.state)} }
func (g *Game) Apply(player int, a engine.Action) error {
	s := &g.state
	if a.Kind != "move" || player != s.Turn || s.Phase != engine.Playing || a.Cell < 0 || a.Cell >= Width {
		return engine.ErrMove
	}
	cell := -1
	for r := Height - 1; r >= 0; r-- {
		i := r*Width + a.Cell
		if s.Board[i] == 0 {
			cell = i
			break
		}
	}
	if cell < 0 {
		return engine.ErrMove
	}
	s.Board[cell] = player + 1
	s.Moves = append(s.Moves, engine.Move{Player: player, Cell: cell})
	s.Line = tictactoe.WinningLine(s.Board, Width, Height, WinLength, player+1)
	if len(s.Line) > 0 {
		s.Winner = player
		s.Phase = engine.Finished
	} else if len(s.Moves) == len(s.Board) {
		s.Winner = engine.Draw
		s.Phase = engine.Finished
	} else {
		s.Turn = 1 - player
	}
	return nil
}
