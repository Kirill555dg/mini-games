// Package engine defines the transport-independent contract implemented by games.
package engine

import "errors"

const (
	NoWinner = -1
	Draw     = -2
	Playing  = "playing"
	Setup    = "setup"
	Finished = "finished"
)

var ErrMove = errors.New("Этот ход недоступен. Обновите поле и попробуйте снова.")

type Action struct {
	Kind       string      `json:"kind"`
	Cell       int         `json:"cell"`
	Vertical   bool        `json:"vertical,omitempty"`
	Placements []Placement `json:"placements,omitempty"`
}

// Placement identifies a ship in the fleet supplied by the owning game.
type Placement struct {
	Ship     int  `json:"ship"`
	Cell     int  `json:"cell"`
	Vertical bool `json:"vertical"`
}

type Move struct {
	Player int   `json:"player"`
	Cell   int   `json:"cell"`
	Hit    bool  `json:"hit,omitempty"`
	Boxes  []int `json:"boxes,omitempty"`
}

// State contains only information the viewer is allowed to receive.
// Empty board cells are zero; player marks are their one-based seat numbers.
type State struct {
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	Phase        string      `json:"phase"`
	Turn         int         `json:"turn"`
	Winner       int         `json:"winner"`
	WinLength    int         `json:"winLength,omitempty"`
	Board        []int       `json:"board,omitempty"`
	Line         []int       `json:"line,omitempty"`
	Moves        []Move      `json:"moves,omitempty"`
	Scores       []int       `json:"scores,omitempty"`
	Boxes        []int       `json:"boxes,omitempty"`
	Own          []int       `json:"own,omitempty"`
	Enemy        []int       `json:"enemy,omitempty"`
	Ready        []bool      `json:"ready,omitempty"`
	Fleet        []int       `json:"fleet,omitempty"`
	FleetLengths []int       `json:"fleetLengths,omitempty"`
	FleetGap     int         `json:"fleetGap,omitempty"`
	Placements   []Placement `json:"placements,omitempty"`
}

type Game interface {
	Apply(player int, action Action) error
	View(viewer int) State
	Clone() Game
}

type Translation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Rules       string `json:"rules"`
}
type Spec struct {
	ID           string                 `json:"id"`
	Translations map[string]Translation `json:"translations"`
	Sides        map[string][]string    `json:"sides"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Rules        string                 `json:"rules"`
	MinPlayers   int                    `json:"minPlayers"`
	MaxPlayers   int                    `json:"maxPlayers"`
	Sizes        []int                  `json:"sizes,omitempty"`
}

// DefaultSize is the first advertised board size; fixed-size games omit sizes.
func (s Spec) DefaultSize() int {
	if len(s.Sizes) > 0 {
		return s.Sizes[0]
	}
	return 0
}

// Copy prevents snapshots and history from aliasing a live game's mutable slices.
func Copy(s State) State {
	s.Board = append([]int(nil), s.Board...)
	s.Line = append([]int(nil), s.Line...)
	s.Moves = append([]Move(nil), s.Moves...)
	for i := range s.Moves {
		s.Moves[i].Boxes = append([]int(nil), s.Moves[i].Boxes...)
	}
	s.Scores = append([]int(nil), s.Scores...)
	s.Boxes = append([]int(nil), s.Boxes...)
	s.Own = append([]int(nil), s.Own...)
	s.Enemy = append([]int(nil), s.Enemy...)
	s.Ready = append([]bool(nil), s.Ready...)
	s.Fleet = append([]int(nil), s.Fleet...)
	s.FleetLengths = append([]int(nil), s.FleetLengths...)
	s.Placements = append([]Placement(nil), s.Placements...)
	return s
}
