// Package battleship keeps fleet placement private until the match has finished.
package battleship

import (
	"crypto/rand"
	"math/big"

	"github.com/Kirill555dg/mini-games/internal/engine"
)

const (
	DefaultSize       = 10
	CompactSize       = 8
	ShipGap           = 1
	CellWater         = 0
	CellShip          = 1
	CellHit           = 2
	CellMiss          = -1
	maxCells          = DefaultSize * DefaultSize
	shuffleAttempts   = 100
	placementAttempts = 1000
)

var classicFleet = [...]int{4, 3, 3, 2, 2, 2, 1, 1, 1, 1}
var compactFleet = [...]int{4, 3, 2, 2, 1}

func Sizes() []int { return []int{DefaultSize, CompactSize} }
func Cells() map[string]int {
	return map[string]int{"water": CellWater, "ship": CellShip, "hit": CellHit, "miss": CellMiss}
}

type Game struct {
	size         int
	boards       [2][maxCells]int
	shots        [2][maxCells]int
	placements   [2][]engine.Placement
	ready        [2]bool
	turn, winner int
	moves        []engine.Move
}

// New receives a size validated against Sizes by the game registry.
func New(size int) *Game {
	if size == 0 {
		size = DefaultSize
	}
	return &Game{size: size, winner: engine.NoWinner}
}
func (g *Game) fleet() []int {
	if g.size == CompactSize {
		return compactFleet[:]
	}
	return classicFleet[:]
}
func (g *Game) Clone() engine.Game {
	c := *g
	c.moves = append([]engine.Move(nil), g.moves...)
	for p := range c.placements {
		c.placements[p] = append([]engine.Placement(nil), g.placements[p]...)
	}
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
	cells := g.size * g.size
	s := engine.State{Width: g.size, Height: g.size, Phase: g.phase(), Turn: g.turn, Winner: g.winner, Ready: []bool{g.ready[0], g.ready[1]}, Moves: append([]engine.Move(nil), g.moves...)}
	if viewer < 0 || viewer > 1 {
		// Spectators receive only fired shots, never fleets or private drafts.
		s.Own = append([]int(nil), g.shots[1][:cells]...)
		s.Enemy = append([]int(nil), g.shots[0][:cells]...)
		return s
	}
	s.Own = append([]int(nil), g.boards[viewer][:cells]...)
	for i, v := range g.shots[1-viewer][:cells] {
		if v != CellWater {
			s.Own[i] = v
		}
	}
	s.Enemy = append([]int(nil), g.shots[viewer][:cells]...)
	if s.Phase == engine.Setup {
		// The full fleet gives stable ship IDs even after a ship is moved or removed.
		s.FleetLengths = append([]int(nil), g.fleet()...)
		s.FleetGap = ShipGap
		s.Placements = append([]engine.Placement(nil), g.placements[viewer]...)
		if !g.ready[viewer] {
			placed := [len(classicFleet)]bool{}
			for _, ship := range g.placements[viewer] {
				placed[ship.Ship] = true
			}
			for ship, length := range g.fleet() {
				if !placed[ship] {
					s.Fleet = append(s.Fleet, length)
				}
			}
		}
	}
	return s
}

func (g *Game) validate(placements []engine.Placement, complete bool) ([maxCells]int, bool) {
	var board [maxCells]int
	fleet := g.fleet()
	if len(placements) > len(fleet) || (complete && len(placements) != len(fleet)) {
		return board, false
	}
	var seen [len(classicFleet)]bool
	for _, ship := range placements {
		if ship.Ship < 0 || ship.Ship >= len(fleet) || seen[ship.Ship] || ship.Cell < 0 || ship.Cell >= g.size*g.size {
			return board, false
		}
		seen[ship.Ship] = true
		length := fleet[ship.Ship]
		r, c := ship.Cell/g.size, ship.Cell%g.size
		dr, dc := 0, 1
		if ship.Vertical {
			dr, dc = 1, 0
		}
		if r+(length-1)*dr >= g.size || c+(length-1)*dc >= g.size {
			return board, false
		}
		for n := 0; n < length; n++ {
			rr, cc := r+n*dr, c+n*dc
			for y := rr - ShipGap; y <= rr+ShipGap; y++ {
				for x := cc - ShipGap; x <= cc+ShipGap; x++ {
					if y >= 0 && y < g.size && x >= 0 && x < g.size && board[y*g.size+x] != CellWater {
						return board, false
					}
				}
			}
		}
		for n := 0; n < length; n++ {
			board[(r+n*dr)*g.size+c+n*dc] = CellShip
		}
	}
	return board, true
}
func (g *Game) draft(player int, placements []engine.Placement, complete bool) bool {
	board, valid := g.validate(placements, complete)
	if !valid {
		return false
	}
	g.boards[player] = board
	g.placements[player] = append([]engine.Placement(nil), placements...)
	return true
}
func (g *Game) place(player, cell int, vertical bool) bool {
	used := [len(classicFleet)]bool{}
	for _, ship := range g.placements[player] {
		used[ship.Ship] = true
	}
	for ship := range g.fleet() {
		if !used[ship] {
			placements := append(append([]engine.Placement(nil), g.placements[player]...), engine.Placement{Ship: ship, Cell: cell, Vertical: vertical})
			return g.draft(player, placements, false)
		}
	}
	return false
}
func (g *Game) shuffle(player int) error {
	for attempt := 0; attempt < shuffleAttempts; attempt++ {
		trial := *g
		trial.placements[player] = nil
		trial.boards[player] = [maxCells]int{}
		for n := 0; n < placementAttempts && len(trial.placements[player]) < len(g.fleet()); n++ {
			x, err := rand.Int(rand.Reader, big.NewInt(int64(g.size*g.size*2)))
			if err != nil {
				return err
			}
			trial.place(player, int(x.Int64())/2, x.Int64()%2 == 1)
		}
		if len(trial.placements[player]) == len(g.fleet()) {
			g.boards[player] = trial.boards[player]
			g.placements[player] = trial.placements[player]
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
		case "fleet", "deploy":
			deploy := a.Kind == "deploy"
			if !g.draft(player, a.Placements, deploy) {
				return engine.ErrMove
			}
			g.ready[player] = deploy
		case "place":
			if !g.place(player, a.Cell, a.Vertical) {
				return engine.ErrMove
			}
		case "clear":
			g.boards[player] = [maxCells]int{}
			g.placements[player] = nil
		case "shuffle":
			return g.shuffle(player)
		case "ready":
			if _, valid := g.validate(g.placements[player], true); !valid {
				return engine.ErrMove
			}
			g.ready[player] = true
		default:
			return engine.ErrMove
		}
		return nil
	}
	if a.Kind != "move" || player != g.turn || a.Cell < 0 || a.Cell >= g.size*g.size || g.shots[player][a.Cell] != CellWater {
		return engine.ErrMove
	}
	hit := g.boards[1-player][a.Cell] != CellWater
	g.shots[player][a.Cell] = CellMiss
	if hit {
		g.shots[player][a.Cell] = CellHit
	}
	g.moves = append(g.moves, engine.Move{Player: player, Cell: a.Cell, Hit: hit})
	remaining := false
	for cell, v := range g.boards[1-player][:g.size*g.size] {
		if v != CellWater && g.shots[player][cell] != CellHit {
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
