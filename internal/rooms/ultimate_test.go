package rooms

import (
	"reflect"
	"testing"
)

func TestNestedBoardStateIsSharedAndMovesAreAuthorized(t *testing.T) {
	h := newTest(t, "")
	v := command(t, h, 1, Command{Action: ActionCreate, Game: "ultimate", Name: "Первый"})
	command(t, h, 2, Command{Action: ActionEnter, Room: v.Room.ID, Name: "Второй"})
	command(t, h, 3, Command{Action: ActionEnter, Room: v.Room.ID, Name: "Зритель", Watch: true})
	command(t, h, 1, Command{Action: ActionReady, Room: v.Room.ID, Round: v.Round})
	command(t, h, 2, Command{Action: ActionReady, Room: v.Room.ID, Round: v.Round})
	moved := command(t, h, 1, Command{Action: ActionMove, Room: v.Room.ID, Round: v.Round, Kind: "move", Cell: 0})
	for _, viewer := range []byte{2, 3} {
		view, err := h.View(v.Room.ID, key(viewer))
		if err != nil || !reflect.DeepEqual(moved.State, view.State) {
			t.Fatal("players and spectator disagree")
		}
	}
	for _, viewer := range []byte{2, 3} {
		if _, err := h.Do(key(viewer), Command{Action: ActionMove, Room: v.Room.ID, Round: v.Round, Kind: "move", Cell: 3}); err == nil {
			t.Fatal("accepted wrong board or spectator move")
		}
	}
	state, err := h.View(v.Room.ID, key(1))
	if err != nil || !reflect.DeepEqual(moved.State, state.State) {
		t.Fatal("rejected action changed room")
	}
	command(t, h, 2, Command{Action: ActionMove, Room: v.Room.ID, Round: v.Round, Kind: "move", Cell: 1})
}
