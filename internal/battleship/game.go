// Package battleship keeps fleet placement private until the match has finished.
package battleship

import (
	"crypto/rand"
	"github.com/Kirill555dg/mini-games/internal/engine"
	"math/big"
)

const Size = 8

var fleet = [...]int{4, 3, 2, 2, 1}

type Game struct {
	boards       [2][Size * Size]int
	shots        [2][Size * Size]int
	placed       [2]int
	ready        [2]bool
	turn, winner int
	moves        []engine.Move
}

func New() *Game { return &Game{winner: engine.NoWinner} }
func (g *Game) Clone() engine.Game {
	c := *g
	c.moves = append([]engine.Move(nil), g.moves...)
	return &c
}
func (g *Game) phase() string {
	if g.winner != engine.NoWinner {
		return engine.Finished
	}
	if !g.ready[0] || !g.ready[1] {
		return engine.Setup
	}
	return engine.Playing
}

func (g *Game) View(viewer int) engine.State {
	s := engine.State{Width: Size, Height: Size, Phase: g.phase(), Turn: g.turn, Winner: g.winner, Ready: []bool{g.ready[0], g.ready[1]}, Moves: append([]engine.Move(nil), g.moves...)}
	if viewer < 0 || viewer > 1 {
		// Spectators receive only fired shots, never an unfinished fleet.
		s.Own = append([]int(nil), g.shots[1][:]...)
		s.Enemy = append([]int(nil), g.shots[0][:]...)
		return s
	}
	s.Own = append([]int(nil), g.boards[viewer][:]...)
	for i, v := range g.shots[1-viewer] {
		if v != 0 {
			s.Own[i] = v
		}
	}
	s.Enemy = append([]int(nil), g.shots[viewer][:]...)
	if !g.ready[viewer] {
		s.Fleet = append([]int(nil), fleet[g.placed[viewer]:]...)
	}
	return s
}

func (g *Game) place(player, cell int, vertical bool) bool {
	if cell < 0 || cell >= Size*Size || g.placed[player] >= len(fleet) {
		return false
	}
	length := fleet[g.placed[player]]
	r, c := cell/Size, cell%Size
	dr, dc := 0, 1
	if vertical {
		dr, dc = 1, 0
	}
	if r+(length-1)*dr >= Size || c+(length-1)*dc >= Size {
		return false
	}
	for n := 0; n < length; n++ {
		rr, cc := r+n*dr, c+n*dc
		for y := rr - 1; y <= rr+1; y++ {
			for x := cc - 1; x <= cc+1; x++ {
				if y >= 0 && y < Size && x >= 0 && x < Size && g.boards[player][y*Size+x] != 0 {
					return false
				}
			}
		}
	}
	for n := 0; n < length; n++ {
		g.boards[player][(r+n*dr)*Size+c+n*dc] = 1
	}
	g.placed[player]++
	return true
}

func (g *Game) shuffle(player int) error {
	for attempt := 0; attempt < 100; attempt++ {
		g.boards[player] = [Size * Size]int{}
		g.placed[player] = 0
		for n := 0; n < 1000 && g.placed[player] < len(fleet); n++ {
			x, err := rand.Int(rand.Reader, big.NewInt(Size*Size*2))
			if err != nil {
				return err
			}
			g.place(player, int(x.Int64())/2, x.Int64()%2 == 1)
		}
		if g.placed[player] == len(fleet) {
			return nil
		}
	}
	return engine.ErrMove
}

func (g *Game) Apply(player int, a engine.Action) error {
	if player < 0 || player > 1 || g.phase() == engine.Finished {
		return engine.ErrMove
	}
	if g.phase() == engine.Setup {
		if g.ready[player] {
			return engine.ErrMove
		}
		switch a.Kind {
		case "place":
			if !g.place(player, a.Cell, a.Vertical) {
				return engine.ErrMove
			}
		case "clear":
			g.boards[player] = [Size * Size]int{}
			g.placed[player] = 0
		case "shuffle":
			return g.shuffle(player)
		case "ready":
			if g.placed[player] != len(fleet) {
				return engine.ErrMove
			}
			g.ready[player] = true
		default:
			return engine.ErrMove
		}
		return nil
	}
	if a.Kind != "move" || player != g.turn || a.Cell < 0 || a.Cell >= Size*Size || g.shots[player][a.Cell] != 0 {
		return engine.ErrMove
	}
	hit := g.boards[1-player][a.Cell] != 0
	g.shots[player][a.Cell] = -1
	if hit {
		g.shots[player][a.Cell] = 2
	}
	g.moves = append(g.moves, engine.Move{Player: player, Cell: a.Cell, Hit: hit})
	remaining := false
	for cell, v := range g.boards[1-player] {
		if v != 0 && g.shots[player][cell] != 2 {
			remaining = true
			break
		}
	}
	if !remaining {
		g.winner = player
	} else if !hit {
		g.turn = 1 - player
	}
	return nil
}
