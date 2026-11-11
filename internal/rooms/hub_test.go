package rooms

import (
	"github.com/Kirill555dg/mini-games/internal/engine"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func newTest(t *testing.T, path string) *Hub {
	t.Helper()
	h, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	return h
}
func key(n byte) Key { var k Key; k[0] = n; return k }
func command(t *testing.T, h *Hub, k byte, c Command) View {
	t.Helper()
	v, err := h.Do(key(k), c)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func pair(t *testing.T, h *Hub) View {
	t.Helper()
	v := command(t, h, 1, Command{Action: "create", Name: "Одинаковое имя", Game: "tictactoe", Size: 3, Players: 2})
	command(t, h, 2, Command{Action: "enter", Room: v.Room.ID, Name: "Одинаковое имя"})
	command(t, h, 1, Command{Action: "ready", Room: v.Room.ID, Round: 1})
	return command(t, h, 2, Command{Action: "ready", Room: v.Room.ID, Round: 1})
}
func win(t *testing.T, h *Hub, v View) View {
	for i, c := range []int{0, 3, 1, 4, 2} {
		v = command(t, h, byte(i%2+1), Command{Action: "move", Room: v.Room.ID, Round: v.Round, Cell: c})
	}
	return v
}
func TestNewPartyHasNewCodeAndRequiresSetup(t *testing.T) {
	h := newTest(t, "")
	v := win(t, h, pair(t, h))
	id := v.Room.ID
	v = command(t, h, 1, Command{Action: "enter", Room: id, Name: "Обновлённое имя"})
	if v.Round != 1 || !v.Room.Finished {
		t.Fatal("reenter reset completed game")
	}
	side := 1
	next := command(t, h, 1, Command{Action: "create", From: id, Name: "Первый", Game: "ultimate", Players: 2, Seat: &side})
	if next.Room.ID == id || next.Room.Started || next.Room.Game != "ultimate" || next.Role != 1 || len(next.State.Moves) != 0 {
		t.Fatal("new party did not create a clean configurable room")
	}
	old, err := h.View(id, key(2))
	if err != nil || old.Room.NextRoom != next.Room.ID || !old.Room.Finished {
		t.Fatal("participants do not see continuation")
	}
	command(t, h, 2, Command{Action: "enter", Room: next.Room.ID, Name: "Второй"})
	if _, err := h.Do(key(2), Command{Action: "move", Room: next.Room.ID, Round: 1, Cell: 0}); err == nil {
		t.Fatal("move before readiness")
	}
	command(t, h, 1, Command{Action: "ready", Room: next.Room.ID, Round: 1})
	command(t, h, 2, Command{Action: "ready", Room: next.Room.ID, Round: 1})
	if _, err := h.Do(key(2), Command{Action: "move", Room: next.Room.ID, Round: 2, Cell: 0}); err == nil {
		t.Fatal("stale round accepted")
	}
}
func TestFinishedRoomsLeaveLobbyButKeepTheirInvitation(t *testing.T) {
	h := newTest(t, "")
	v := win(t, h, pair(t, h))
	if lobby := h.Lobby(); lobby.TotalRooms != 0 || len(lobby.Rooms) != 0 {
		t.Fatal("completed games still advertised as open rooms")
	}
	old, err := h.View(v.Room.ID, key(2))
	if err != nil || !old.Room.Finished {
		t.Fatal("removing a lobby entry invalidated the room's invitation")
	}
}
func TestSwapNeedsConsentAndClearsReadiness(t *testing.T) {
	h := newTest(t, "")
	v := command(t, h, 1, Command{Action: "create", Name: "Один"})
	command(t, h, 2, Command{Action: "enter", Room: v.Room.ID, Name: "Два"})
	command(t, h, 1, Command{Action: "ready", Room: v.Room.ID, Round: 1})
	side := 1
	v = command(t, h, 1, Command{Action: "seat", Room: v.Room.ID, Round: 1, Seat: &side})
	if v.Role != 0 || v.Room.Swap == nil || v.Room.Seats[0].Ready {
		t.Fatal("swap occurred without consent or retained readiness")
	}
	if _, err := h.Do(key(1), Command{Action: "accept-swap", Room: v.Room.ID, Round: 1}); err == nil {
		t.Fatal("requester accepted own swap")
	}
	v = command(t, h, 2, Command{Action: "accept-swap", Room: v.Room.ID, Round: 1})
	if v.Role != 0 || v.Room.Seats[1].Name != "Один" || v.Room.Swap != nil {
		t.Fatal("consensual swap failed")
	}
}

func TestCannotCreateSecondPlayingRoomByPayload(t *testing.T) {
	h := newTest(t, "")
	v := pair(t, h)
	for _, c := range []Command{{Action: "create", Name: "Один", Watch: true}, {Action: "create", Name: "Один", Room: v.Room.ID}} {
		if _, err := h.Do(key(1), c); err == nil {
			t.Fatal("second active membership accepted")
		}
	}
	if len(h.Lobby().Rooms) != 1 {
		t.Fatal("rejected create left a room")
	}
}
func TestSpectatorPermissionsAndCapacity(t *testing.T) {
	h := newTest(t, "")
	v := pair(t, h)
	spect := command(t, h, 3, Command{Action: "enter", Room: v.Room.ID, Name: "Зритель", Watch: true})
	if spect.Role != -1 || spect.Room.Spectators != 1 {
		t.Fatal("spectator state")
	}
	for _, action := range []string{"move", "again"} {
		if _, err := h.Do(key(3), Command{Action: action, Room: v.Room.ID, Round: 1, Cell: 0}); err == nil {
			t.Fatal("spectator action accepted")
		}
	}
	if _, err := h.Do(key(4), Command{Action: "enter", Room: v.Room.ID, Name: "Третий"}); err == nil {
		t.Fatal("full room accepted player")
	}
	command(t, h, 4, Command{Action: "create", Name: "Вторая"})
	command(t, h, 5, Command{Action: "create", Name: "Третья"})
	if len(h.Lobby().Rooms) != 3 {
		t.Fatal("obsolete two-room cap")
	}
	h.mu.Lock()
	h.rooms[0].memberCapacity = StateBudgetBytes / memberBytes
	h.mu.Unlock()
	if _, err := h.Do(key(6), Command{Action: "create", Name: "Новая"}); err == nil {
		t.Fatal("memory budget ignored")
	}

}
func TestFourPlayersAndNoStartBeforeFull(t *testing.T) {
	h := newTest(t, "")
	v := command(t, h, 1, Command{Action: "create", Name: "Первый", Game: "dotsboxes", Players: 4})
	for n := 2; n <= 4; n++ {
		if v.Room.Started {
			t.Fatal("started without selected players")
		}
		v = command(t, h, byte(n), Command{Action: "enter", Room: v.Room.ID, Name: "Игрок"})
	}
	for n := 1; n <= 4; n++ {
		v = command(t, h, byte(n), Command{Action: "ready", Room: v.Room.ID, Round: 1})
	}
	if !v.Room.Started || len(v.Room.Seats) != 4 {
		t.Fatal("four-player room failed")
	}
	command(t, h, 1, Command{Action: "move", Room: v.Room.ID, Round: 1, Cell: 0})
}
func TestDisconnectGraceAndForfeit(t *testing.T) {
	h := newTest(t, "")
	now := time.Now()
	h.mu.Lock()
	h.now = func() time.Time { return now }
	h.mu.Unlock()
	v := pair(t, h)
	sub, err := h.Subscribe(v.Room.ID, key(1))
	if err != nil {
		t.Fatal(err)
	}
	h.mu.Lock()
	now = now.Add(GracePeriod - time.Second)
	h.mu.Unlock()
	if len(h.Lobby().Rooms) != 1 || len(h.History()) != 0 {
		t.Fatal("expired early")
	}
	h.mu.Lock()
	now = now.Add(time.Second)
	h.mu.Unlock()
	lobby := h.Lobby()
	if len(lobby.Rooms) != 0 || len(h.History()) != 1 || h.History()[0].Winner != 0 {
		t.Fatal("timeout must award surviving player")
	}
	h.Unsubscribe(sub)
	h.mu.Lock()
	now = now.Add(GracePeriod)
	h.mu.Unlock()
	if len(h.Lobby().Rooms) != 0 {
		t.Fatal("empty room not removed")
	}
	if h.timer != nil {
		t.Fatal("timer survives last room")
	}
}
func TestPersistenceFailureRollsBackWinningMove(t *testing.T) {
	h := newTest(t, filepath.Join(t.TempDir(), "missing", "stats.json"))
	v := pair(t, h)
	for i, c := range []int{0, 3, 1, 4} {
		command(t, h, byte(i%2+1), Command{Action: "move", Room: v.Room.ID, Round: 1, Cell: c})
	}
	if _, err := h.Do(key(1), Command{Action: "move", Room: v.Room.ID, Round: 1, Cell: 2}); err == nil {
		t.Fatal("write failure hidden")
	}
	view, err := h.View(v.Room.ID, key(1))
	if err != nil {
		t.Fatal(err)
	}
	if view.State.Board[2] != 0 || view.State.Winner != engine.NoWinner || len(view.State.Moves) != 4 || view.State.Turn != 0 {
		t.Fatal("winning move did not roll back")
	}
}
func TestLeaveRecordsOnceAndPrivateNotifications(t *testing.T) {
	h := newTest(t, "")
	v := pair(t, h)
	other := command(t, h, 3, Command{Action: "create", Name: "Другой"})
	s, err := h.Subscribe(other.Room.ID, key(3))
	if err != nil {
		t.Fatal(err)
	}
	defer h.Unsubscribe(s)
	<-s.Change
	command(t, h, 1, Command{Action: "leave", Room: v.Room.ID})
	if len(h.History()) != 1 || h.History()[0].Winner != 1 {
		t.Fatal("forfeit record")
	}
	select {
	case <-s.Change:
		t.Fatal("unrelated room notified")
	default:
	}
	command(t, h, 2, Command{Action: "leave", Room: v.Room.ID})
	if len(h.History()) != 1 {
		t.Fatal("duplicate result")
	}
}
func TestConcurrentViewsAndBoundedSubscriptions(t *testing.T) {
	h := newTest(t, "")
	v := pair(t, h)
	var wg sync.WaitGroup
	for n := 0; n < 20; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				h.Lobby()
				_, _ = h.View(v.Room.ID, key(1))
				h.History()
			}
		}()
	}
	wg.Wait()
	for n := 0; n < MaxEventStreams; n++ {
		if _, err := h.Subscribe("", Key{}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.Subscribe("", Key{}); err == nil {
		t.Fatal("subscription limit exceeded")
	}
}
