// Package tictactoe implements square line-making games.
package tictactoe

import "github.com/Kirill555dg/mini-games/internal/engine"

type Game struct{ state engine.State }

func New(size int) *Game {
	k := 5
	if size < k {
		k = size
	}
	return &Game{engine.State{Width: size, Height: size, WinLength: k, Phase: engine.Playing, Winner: engine.NoWinner, Board: make([]int, size*size)}}
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
	s.Line = WinningLine(s.Board, s.Width, s.Height, s.WinLength, player+1)
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

// WinningLine uses coordinates so horizontal and diagonal lines cannot wrap.
func WinningLine(board []int, width, height, length, mark int) []int {
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			if board[row*width+col] != mark {
				continue
			}
			for _, d := range [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}} {
				er, ec := row+(length-1)*d[0], col+(length-1)*d[1]
				if er < 0 || er >= height || ec < 0 || ec >= width {
					continue
				}
				line := make([]int, 0, length)
				for n := 0; n < length; n++ {
					i := (row+n*d[0])*width + col + n*d[1]
					if board[i] != mark {
						break
					}
					line = append(line, i)
				}
				if len(line) == length {
					return line
				}
			}
		}
	}
	return nil
}
