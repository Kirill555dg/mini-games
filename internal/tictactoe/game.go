// Package tictactoe implements the short classic 3x3 game.
package tictactoe

import "github.com/Kirill555dg/mini-games/internal/engine"

const Size = 3

type Game struct{ state engine.State }

func New() *Game {
	return &Game{engine.State{Width: Size, Height: Size, WinLength: Size, Phase: engine.Playing, Winner: engine.NoWinner, Board: make([]int, Size*Size)}}
}

func (g *Game) View(_ int) engine.State { return engine.Copy(g.state) }
func (g *Game) Clone() engine.Game      { return &Game{engine.Copy(g.state)} }

func (g *Game) Apply(player int, a engine.Action) error {
	s := &g.state
	if a.Kind != "move" || player != s.Turn || s.Phase != engine.Playing || a.Cell < 0 || a.Cell >= len(s.Board) || s.Board[a.Cell] != 0 {
		return engine.ErrMove
	}
	s.Board[a.Cell] = player + 1
	s.Moves = append(s.Moves, engine.Move{Player: player, Cell: a.Cell})
	s.Line = engine.WinningLine(s.Board, s.Width, s.Height, s.WinLength, player+1)
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
