package rooms

import (
	"reflect"
	"testing"

	"github.com/Kirill555dg/mini-games/internal/battleship"
	"github.com/Kirill555dg/mini-games/internal/engine"
)

func TestBattleDefaultsAndDraftAreViewerSpecific(t *testing.T) {
	h := newTest(t, "")
	v := command(t, h, 1, Command{Action: ActionCreate, Game: "battleship", Name: "Первый"})
	if v.Room.Size != battleship.DefaultSize || v.State.Width != battleship.DefaultSize || len(v.Room.Seats) != 2 {
		t.Fatal("catalog default not applied")
	}
	command(t, h, 2, Command{Action: ActionEnter, Room: v.Room.ID, Name: "Второй"})
	command(t, h, 3, Command{Action: ActionEnter, Room: v.Room.ID, Name: "Зритель", Watch: true})
	command(t, h, 1, Command{Action: ActionReady, Room: v.Room.ID, Round: v.Round})
	v = command(t, h, 2, Command{Action: ActionReady, Room: v.Room.ID, Round: v.Round})
	draft := []engine.Placement{{Ship: 0, Cell: 0}}
	owner := command(t, h, 1, Command{Action: ActionMove, Room: v.Room.ID, Round: v.Round, Kind: "fleet", Placements: draft})
	if len(owner.State.Placements) != 1 || len(owner.State.FleetLengths) != 10 || len(owner.State.Fleet) != 9 {
		t.Fatal("draft not forwarded")
	}
	for _, viewer := range []byte{2, 3} {
		other, err := h.View(v.Room.ID, key(viewer))
		if err != nil || len(other.State.Placements) != 0 {
			t.Fatal("fleet draft leaked")
		}
		for _, value := range other.State.Enemy {
			if value != 0 {
				t.Fatal("fleet cells leaked")
			}
		}
	}
	if _, err := h.Do(key(1), Command{Action: ActionMove, Room: v.Room.ID, Round: v.Round, Kind: "deploy", Placements: draft}); err == nil {
		t.Fatal("partial fleet deployed")
	}
	after, err := h.View(v.Room.ID, key(1))
	if err != nil || !reflect.DeepEqual(owner.State, after.State) {
		t.Fatal("failed deploy mutated room")
	}
	seat := 1
	if _, err := h.Do(key(1), Command{Action: ActionSeat, Room: v.Room.ID, Round: v.Round, Seat: &seat}); err == nil {
		t.Fatal("seat changed after private fleet placement")
	}
}
