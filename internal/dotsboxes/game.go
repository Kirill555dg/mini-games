// Package dotsboxes implements a shared edge board for two to four players.
package dotsboxes

import "github.com/Kirill555dg/mini-games/internal/engine"

const Size = 4
const horizontal = (Size + 1) * Size
const Edges = 2 * horizontal

type Game struct{ state engine.State }

func New(players int) *Game {
	return &Game{engine.State{Width: Size, Height: Size, Phase: engine.Playing, Winner: engine.NoWinner, Board: make([]int, Edges), Boxes: make([]int, Size*Size), Scores: make([]int, players)}}
}
func (g *Game) View(_ int) engine.State { return engine.Copy(g.state) }
func (g *Game) Clone() engine.Game      { return &Game{engine.Copy(g.state)} }
func (g *Game) Apply(player int, a engine.Action) error {
	s := &g.state
	if a.Kind != "move" || player != s.Turn || s.Phase != engine.Playing || a.Cell < 0 || a.Cell >= Edges || s.Board[a.Cell] != 0 {
		return engine.ErrMove
	}
	s.Board[a.Cell] = player + 1
	s.Moves = append(s.Moves, engine.Move{Player: player, Cell: a.Cell})
	won := false
	for r := 0; r < Size; r++ {
		for c := 0; c < Size; c++ {
			i := r*Size + c
			if s.Boxes[i] != 0 {
				continue
			}
			if s.Board[r*Size+c] > 0 && s.Board[(r+1)*Size+c] > 0 && s.Board[horizontal+r*(Size+1)+c] > 0 && s.Board[horizontal+r*(Size+1)+c+1] > 0 {
				s.Boxes[i] = player + 1
				s.Scores[player]++
				s.Moves[len(s.Moves)-1].Boxes = append(s.Moves[len(s.Moves)-1].Boxes, i)
				won = true
			}
		}
	}
	if len(s.Moves) == Edges {
		s.Phase = engine.Finished
		s.Winner = engine.Draw
		highest := -1
		for p, n := range s.Scores {
			if n > highest {
				highest = n
				s.Winner = p
			} else if n == highest {
				s.Winner = engine.Draw
			}
		}
	} else if !won {
		s.Turn = (player + 1) % len(s.Scores)
	}
	return nil
}
