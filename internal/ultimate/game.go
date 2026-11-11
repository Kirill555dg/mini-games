// Package ultimate implements nine-board tic-tac-toe with closed local boards.
package ultimate

import "github.com/Kirill555dg/mini-games/internal/engine"

const (
	Side        = 3
	Width       = Side * Side
	CellCount   = Width * Width
	RegionCount = Side * Side
	Players     = 2
	Open        = 0
)

type Game struct{ state engine.State }

func New() *Game {
	return &Game{engine.State{Width: Width, Height: Width, WinLength: Side,
		RegionSize: Side, Regions: make([]int, RegionCount), Board: make([]int, CellCount),
		Phase: engine.Playing, Winner: engine.NoWinner}}
}

func (g *Game) View(_ int) engine.State { return engine.Copy(g.state) }
func (g *Game) Clone() engine.Game      { return &Game{engine.Copy(g.state)} }

func regionOf(cell int) int { return (cell/Width/Side)*Side + (cell % Width / Side) }
func localOf(cell int) int  { return (cell/Width%Side)*Side + cell%Side }
func globalCell(region, cell int) int {
	return (region/Side*Side+cell/Side)*Width + region%Side*Side + cell%Side
}

func (g *Game) Apply(player int, action engine.Action) error {
	s := &g.state
	if action.Kind != "move" || player != s.Turn || s.Phase != engine.Playing ||
		action.Cell < 0 || action.Cell >= CellCount || s.Board[action.Cell] != Open {
		return engine.ErrMove
	}
	region, local := regionOf(action.Cell), localOf(action.Cell)
	if s.Regions[region] != Open || s.ActiveBoard != nil && *s.ActiveBoard != region {
		return engine.ErrMove
	}
	mark := player + 1
	s.Board[action.Cell] = mark
	cells := make([]int, RegionCount)
	full := true
	for index := range cells {
		cells[index] = s.Board[globalCell(region, index)]
		full = full && cells[index] != Open
	}
	result := Open
	if len(engine.WinningLine(cells, Side, Side, Side, mark)) > 0 {
		result = mark
	} else if full {
		result = engine.Draw
	}
	s.Regions[region] = result
	s.RegionLine = engine.WinningLine(s.Regions, Side, Side, Side, mark)
	allClosed := true
	for _, value := range s.Regions {
		allClosed = allClosed && value != Open
	}
	s.ActiveBoard = nil
	if len(s.RegionLine) > 0 {
		s.Winner, s.Phase = player, engine.Finished
	} else if allClosed {
		s.Winner, s.Phase = engine.Draw, engine.Finished
	} else {
		s.Turn = Players - 1 - player
		if s.Regions[local] == Open {
			s.ActiveBoard = &local
		}
	}
	s.Moves = append(s.Moves, engine.Move{Player: player, Cell: action.Cell,
		RegionResult: result, NextBoard: s.ActiveBoard})
	return nil
}
