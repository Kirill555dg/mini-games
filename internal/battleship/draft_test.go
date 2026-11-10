package battleship

import (
	"reflect"
	"testing"

	"github.com/Kirill555dg/mini-games/internal/engine"
)

func classicDraft() []engine.Placement {
	cells := []int{0, 6, 20, 25, 40, 44, 48, 60, 64, 68}
	ships := make([]engine.Placement, len(cells))
	for ship, cell := range cells {
		ships[ship] = engine.Placement{Ship: ship, Cell: cell}
	}
	return ships
}
func TestClassicDraftCanBeEditedAndIsPrivate(t *testing.T) {
	g := New(DefaultSize)
	draft := []engine.Placement{{Ship: 0, Cell: 0}, {Ship: 2, Cell: 91}}
	if err := g.Apply(0, engine.Action{Kind: "fleet", Placements: draft}); err != nil {
		t.Fatal(err)
	}
	draft[0].Cell = 99
	view := g.View(0)
	if len(view.Own) != 100 || len(view.FleetLengths) != len(classicFleet) || len(view.Fleet) != len(classicFleet)-len(draft) || len(view.Placements) != 2 || view.Placements[0].Cell != 0 {
		t.Fatal("incomplete owner draft")
	}
	view.Placements[0].Cell = 99
	view.FleetLengths[0] = 99
	if g.View(0).Placements[0].Cell != 0 || g.View(0).FleetLengths[0] != classicFleet[0] {
		t.Fatal("view aliases game")
	}
	for _, viewer := range []int{-1, 1} {
		state := g.View(viewer)
		if len(state.Placements) != 0 {
			t.Fatal("private draft leaked")
		}
		for _, value := range state.Enemy {
			if value != 0 {
				t.Fatal("private fleet leaked")
			}
		}
		if viewer < 0 && (len(state.Fleet) != 0 || len(state.FleetLengths) != 0 || state.FleetGap != 0) {
			t.Fatal("private setup data in spectator state")
		}
	}
	before := g.View(0)
	invalid := [][]engine.Placement{
		{{Ship: 0, Cell: 98}},
		{{Ship: 0, Cell: 90, Vertical: true}},
		{{Ship: -1, Cell: 0}},
		{{Ship: len(classicFleet), Cell: 0}},
		{{Ship: 0, Cell: -1}},
		{{Ship: 0, Cell: 100}},
		{{Ship: 0, Cell: 0}, {Ship: 0, Cell: 50}},
		{{Ship: 0, Cell: 0}, {Ship: 1, Cell: 4}},
		{{Ship: 0, Cell: 0}, {Ship: 1, Cell: 11}},
	}
	for _, ships := range invalid {
		if g.Apply(0, engine.Action{Kind: "fleet", Placements: ships}) == nil {
			t.Fatalf("invalid placement accepted: %+v", ships)
		}
		if !reflect.DeepEqual(before, g.View(0)) {
			t.Fatal("invalid draft changed prior state")
		}
	}
	if g.Apply(0, engine.Action{Kind: "deploy", Placements: before.Placements}) == nil || !reflect.DeepEqual(before, g.View(0)) {
		t.Fatal("partial deployment changed state")
	}
	moved := []engine.Placement{{Ship: 2, Cell: 80, Vertical: false}}
	clone := g.Clone()
	if clone.Apply(0, engine.Action{Kind: "fleet", Placements: moved}) != nil || !reflect.DeepEqual(before, g.View(0)) {
		t.Fatal("clone aliases draft")
	}
	if g.Apply(0, engine.Action{Kind: "fleet", Placements: nil}) != nil || len(g.View(0).Placements) != 0 {
		t.Fatal("removal did not save empty draft")
	}
}
func TestCompleteClassicDeploymentAndMatch(t *testing.T) {
	g := New(DefaultSize)
	for player := 0; player < 2; player++ {
		if g.Apply(player, engine.Action{Kind: "deploy", Placements: classicDraft()}) != nil {
			t.Fatal("classic deployment failed")
		}
		if g.Apply(player, engine.Action{Kind: "fleet"}) == nil {
			t.Fatal("ready fleet changed")
		}
	}
	if g.View(0).Phase != engine.Playing || len(g.View(0).Placements) != 0 || len(g.View(-1).Placements) != 0 {
		t.Fatal("deployment phase or privacy")
	}
	for player := 0; player < 2; player++ {
		if g.Apply(player, engine.Action{Kind: "move", Cell: 99}) != nil {
			t.Fatal("corner miss failed")
		}
	}
	hits := 0
	for cell, value := range g.boards[1] {
		if value == 0 {
			continue
		}
		if g.Apply(0, engine.Action{Kind: "move", Cell: cell}) != nil {
			t.Fatal("classic shot failed")
		}
		hits++
	}
	if hits != 20 || g.View(-1).Phase != engine.Finished || g.View(-1).Winner != 0 {
		t.Fatal("wrong classic fleet result")
	}
	if g.Apply(0, engine.Action{Kind: "move", Cell: 99}) == nil {
		t.Fatal("finished match accepted a shot")
	}
}
func TestRandomFleetsFitBothModes(t *testing.T) {
	for _, size := range Sizes() {
		g := New(size)
		for n := 0; n < 5; n++ {
			if err := g.Apply(0, engine.Action{Kind: "shuffle"}); err != nil {
				t.Fatal(err)
			}
			if _, valid := g.validate(g.placements[0], true); !valid {
				t.Fatal("invalid random fleet")
			}
			cells := 0
			for _, value := range g.View(0).Own {
				cells += value
			}
			want := 20
			if size == CompactSize {
				want = 12
			}
			if cells != want {
				t.Fatal("wrong fleet size")
			}
		}
	}
}

func TestLegacyFleetIsRemainingWhileFullLengthsStayStable(t *testing.T) {
	g := New(CompactSize)
	for placed, cell := range []int{0, 16, 32, 48, 63} {
		if !reflect.DeepEqual(g.View(0).Fleet, compactFleet[placed:]) {
			t.Fatal("legacy remaining fleet changed")
		}
		if !reflect.DeepEqual(g.View(0).FleetLengths, compactFleet[:]) {
			t.Fatal("full fleet lengths changed during placement")
		}
		if g.Apply(0, engine.Action{Kind: "place", Cell: cell}) != nil {
			t.Fatal("legacy placement failed")
		}
	}
	if len(g.View(0).Fleet) != 0 || len(g.View(0).FleetLengths) != len(compactFleet) {
		t.Fatal("complete draft does not enable legacy readiness")
	}
	if g.Apply(0, engine.Action{Kind: "ready"}) != nil || len(g.View(0).Fleet) != 0 {
		t.Fatal("legacy readiness failed")
	}
	if len(g.View(0).FleetLengths) != len(compactFleet) {
		t.Fatal("ready owner lost full setup fleet")
	}
	if g.Apply(1, engine.Action{Kind: "shuffle"}) != nil || g.Apply(1, engine.Action{Kind: "ready"}) != nil {
		t.Fatal("second fleet ready")
	}
	for _, viewer := range []int{-1, 0, 1} {
		state := g.View(viewer)
		if len(state.Fleet) != 0 || len(state.FleetLengths) != 0 || len(state.Placements) != 0 || state.FleetGap != 0 {
			t.Fatal("setup metadata remained after play started")
		}
	}
}
