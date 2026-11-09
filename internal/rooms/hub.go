// Package rooms owns rooms, browser membership and bounded live subscriptions.
package rooms

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/Kirill555dg/mini-games/internal/engine"
	"github.com/Kirill555dg/mini-games/internal/games"
	"github.com/Kirill555dg/mini-games/internal/history"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

type Key [32]byte
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string                 { return e.Message }
func problem(status int, message string) error { return &Error{status, message} }

type member struct {
	name  string
	seen  time.Time
	ready bool
}
type room struct {
	id, game       string
	size           int
	round          uint64
	players        []Key
	members        map[Key]*member
	memberCapacity int
	engine         engine.Game
	started        bool
	winner         int
	reason         string
	nextRoom       string
	swap           *Swap
}
type Subscription struct {
	Room   string
	Key    Key
	Change chan struct{}
}
type Hub struct {
	mu       sync.Mutex
	rooms    []*room
	subs     map[*Subscription]struct{}
	history  *history.Store
	instance string
	revision uint64
	now      func() time.Time
	timer    *time.Timer
	closed   bool
	window   time.Time
	actions  int
}
type Seat struct {
	Name     string     `json:"name"`
	Taken    bool       `json:"taken"`
	Online   bool       `json:"online"`
	Ready    bool       `json:"ready"`
	ReturnBy *time.Time `json:"returnBy,omitempty"`
}
type Swap struct {
	From int `json:"from"`
	To   int `json:"to"`
}
type Summary struct {
	ID         string `json:"id"`
	Game       string `json:"game"`
	Size       int    `json:"size"`
	Seats      []Seat `json:"seats"`
	Spectators int    `json:"spectators"`
	Started    bool   `json:"started"`
	Finished   bool   `json:"finished"`
	NextRoom   string `json:"nextRoom,omitempty"`
	Swap       *Swap  `json:"swap,omitempty"`
}
type View struct {
	Instance string       `json:"instance"`
	Revision uint64       `json:"revision"`
	Room     Summary      `json:"room"`
	Round    uint64       `json:"round"`
	Role     int          `json:"role"`
	Name     string       `json:"name"`
	State    engine.State `json:"state"`
	Reason   string       `json:"reason,omitempty"`
}
type Lobby struct {
	Instance   string        `json:"instance"`
	Revision   uint64        `json:"revision"`
	TotalRooms int           `json:"totalRooms"`
	Rooms      []Summary     `json:"rooms"`
	Games      []engine.Spec `json:"games"`
}
type Command struct {
	Action   string `json:"action"`
	From     string `json:"from,omitempty"`
	Seat     *int   `json:"seat,omitempty"`
	Room     string `json:"room,omitempty"`
	Name     string `json:"name,omitempty"`
	Game     string `json:"game,omitempty"`
	Size     int    `json:"size,omitempty"`
	Players  int    `json:"players,omitempty"`
	Watch    bool   `json:"watch,omitempty"`
	Round    uint64 `json:"round,omitempty"`
	Cell     int    `json:"cell,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Vertical bool   `json:"vertical,omitempty"`
}

func New(path string) (*Hub, error) {
	s, err := history.Open(path)
	if err != nil {
		return nil, err
	}
	var id [8]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	return &Hub{history: s, instance: hex.EncodeToString(id[:]), revision: 1, subs: make(map[*Subscription]struct{}), now: time.Now}, nil
}
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	if h.timer != nil {
		h.timer.Stop()
	}
	for s := range h.subs {
		close(s.Change)
		delete(h.subs, s)
	}
}
func (h *Hub) find(id string) *room {
	for _, r := range h.rooms {
		if r.id == NormalizeCode(id) {
			return r
		}
	}
	return nil
}
func (r *room) role(k Key) int {
	if k == [32]byte{} {
		return -1
	}
	for i, p := range r.players {
		if p == k {
			return i
		}
	}
	return -1
}
func (h *Hub) streaming(id string, k Key) bool {
	for s := range h.subs {
		if s.Room == id && s.Key == k {
			return true
		}
	}
	return false
}
func (h *Hub) notify(id string) {
	h.revision++
	for s := range h.subs {
		if s.Room == "" || s.Room == id {
			select {
			case s.Change <- struct{}{}:
			default:
			}
		}
	}
}
func (h *Hub) schedule() {
	if h.closed || len(h.rooms) == 0 || h.timer != nil {
		return
	}
	h.timer = time.AfterFunc(ExpireInterval, func() { h.mu.Lock(); defer h.mu.Unlock(); h.timer = nil; h.expire(); h.schedule() })
}
func validName(s string) bool {
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) < 1 || utf8.RuneCountInString(s) > MaxNameRunes {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}
func (h *Hub) summary(r *room) Summary {
	s := Summary{ID: r.id, Game: r.game, Size: r.size, Seats: make([]Seat, len(r.players)), Started: r.started, Finished: r.winner != engine.NoWinner, NextRoom: r.nextRoom, Swap: r.swap}
	for i, k := range r.players {
		if m := r.members[k]; m != nil {
			online := h.streaming(r.id, k) || h.now().Sub(m.seen) < PresenceWindow
			s.Seats[i] = Seat{Name: m.name, Taken: true, Online: online, Ready: m.ready}
			if !online {
				deadline := m.seen.Add(GracePeriod)
				s.Seats[i].ReturnBy = &deadline
			}
		}
	}
	for k := range r.members {
		if r.role(k) < 0 {
			s.Spectators++
		}
	}
	return s
}
func (h *Hub) view(r *room, k Key) View {
	actor := r.role(k)
	s := r.engine.View(actor)
	if r.winner != engine.NoWinner {
		s.Winner = r.winner
		s.Phase = engine.Finished
	}
	v := View{Instance: h.instance, Revision: h.revision, Room: h.summary(r), Round: r.round, Role: actor, State: s, Reason: r.reason}
	if m := r.members[k]; m != nil {
		v.Name = m.name
	}
	return v
}
func (h *Hub) lobby() Lobby {
	out := Lobby{Instance: h.instance, Revision: h.revision, TotalRooms: len(h.rooms), Rooms: []Summary{}, Games: games.Catalog()}
	for _, r := range h.rooms {
		if len(out.Rooms) >= LobbyPageSize {
			break
		}
		out.Rooms = append(out.Rooms, h.summary(r))
	}
	return out
}
func (h *Hub) Lobby() Lobby              { h.mu.Lock(); defer h.mu.Unlock(); h.expire(); return h.lobby() }
func (h *Hub) History() []history.Record { h.mu.Lock(); defer h.mu.Unlock(); return h.history.List() }
func (h *Hub) View(id string, k Key) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expire()
	r := h.find(id)
	if r == nil {
		return View{}, problem(404, "Комната закрыта. Вернитесь к списку игр.")
	}
	if m := r.members[k]; m != nil {
		m.seen = h.now()
	}
	return h.view(r, k), nil
}

func (h *Hub) finish(r *room, winner int, reason string) error {
	if r.winner != engine.NoWinner {
		return nil
	}
	if r.started {
		names := make([]string, len(r.players))
		for i, k := range r.players {
			if m := r.members[k]; m != nil {
				names[i] = m.name
			} else {
				names[i] = "Игрок"
			}
		}
		s := r.engine.View(-1)
		s.Winner = winner
		s.Phase = engine.Finished
		if err := h.history.Append(history.Record{FinishedAt: h.now().UTC(), Code: r.id, Game: r.game, Size: r.size, Players: names, Winner: winner, Reason: reason, State: s}); err != nil {
			return problem(503, "Не удалось сохранить результат. Повторите действие.")
		}
	}
	r.winner = winner
	r.reason = reason
	return nil
}

func (h *Hub) expire() {
	kept := h.rooms[:0]
	for _, r := range h.rooms {
		expired := map[Key]bool{}
		for k, m := range r.members {
			if !h.streaming(r.id, k) && h.now().Sub(m.seen) >= GracePeriod {
				expired[k] = true
			}
		}
		if r.started && r.winner == engine.NoWinner {
			losers := 0
			survivor := -1
			alive := 0
			for p, k := range r.players {
				if expired[k] {
					losers++
				} else if k != [32]byte{} {
					alive++
					survivor = p
				}
			}
			if losers > 0 {
				winner := engine.Draw
				if alive == 1 {
					winner = survivor
				}
				if h.finish(r, winner, "timeout") != nil {
					kept = append(kept, r)
					continue
				}
			}
		}
		for k := range expired {
			if p := r.role(k); p >= 0 {
				r.players[p] = Key{}
				r.swap = nil
				r.clearReady()
			}
			delete(r.members, k)
		}
		if len(expired) > 0 {
			r.compactMembers()
		}
		occupied := false
		for _, k := range r.players {
			occupied = occupied || k != [32]byte{}
		}
		if occupied {
			kept = append(kept, r)
		}
		if len(expired) > 0 || !occupied {
			h.notify(r.id)
		}
	}
	clear(h.rooms[len(kept):])
	h.rooms = kept
	if len(h.rooms) == 0 && h.timer != nil {
		h.timer.Stop()
		h.timer = nil
	}
}

func (h *Hub) Subscribe(id string, k Key) (*Subscription, error) {
	if id != "" {
		id = NormalizeCode(id)
		if id == "" {
			return nil, problem(400, "Неверный код комнаты.")
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expire()
	if h.closed || len(h.subs) >= MaxEventStreams || !h.admit(streamBytes) {
		return nil, problem(503, "Подключений слишком много. Включён резервный опрос.")
	}
	if id != "" {
		r := h.find(id)
		if r == nil {
			return nil, problem(404, "Комната закрыта.")
		}
		if m := r.members[k]; m != nil {
			m.seen = h.now()
		}
	}
	s := &Subscription{Room: id, Key: k, Change: make(chan struct{}, 1)}
	h.subs[s] = struct{}{}
	h.notify(id)
	return s, nil
}
func (h *Hub) Unsubscribe(s *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[s]; !ok {
		return
	}
	delete(h.subs, s)
	if r := h.find(s.Room); r != nil {
		if m := r.members[s.Key]; m != nil {
			m.seen = h.now()
		}
	}
	h.notify(s.Room)
}
func (h *Hub) Event(s *Subscription) any {
	if s.Room == "" {
		return h.Lobby()
	}
	v, err := h.View(s.Room, s.Key)
	if err != nil {
		return nil
	}
	return v
}

func (h *Hub) Do(k Key, c Command) (View, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.expire()
	if k == [32]byte{} {
		return View{}, problem(403, "Не удалось открыть сессию.")
	}
	if h.now().Sub(h.window) >= time.Second {
		h.window = h.now()
		h.actions = 0
	}
	h.actions++
	if h.actions > MaxActionsPerSecond {
		return View{}, problem(429, "Слишком много действий. Подождите секунду.")
	}
	if c.Action == ActionCreate || c.Action == ActionEnter {
		return h.enter(k, c)
	}
	r := h.find(c.Room)
	if r == nil {
		return View{}, problem(404, "Комната уже закрыта.")
	}
	m := r.members[k]
	if m == nil {
		return View{}, problem(403, "Сначала войдите в комнату.")
	}
	m.seen = h.now()
	actor := r.role(k)
	if c.Action == ActionLeave {
		if actor >= 0 {
			winner := engine.Draw
			others := 0
			for p, pk := range r.players {
				if p != actor && pk != [32]byte{} {
					winner = p
					others++
				}
			}
			if others != 1 {
				winner = engine.Draw
			}
			if r.started && h.finish(r, winner, "left") != nil {
				return View{}, problem(503, "Не удалось сохранить выход. Повторите действие.")
			}
			r.players[actor] = Key{}
			r.swap = nil
			r.clearReady()
		}
		delete(r.members, k)
		if len(r.members)*2 < r.memberCapacity {
			r.compactMembers()
		}
		v := h.view(r, k)
		h.expire()
		h.notify(r.id)
		return v, nil
	}
	if actor < 0 {
		return View{}, problem(403, "Зрители не могут менять игру.")
	}
	if c.Round != r.round {
		return View{}, problem(409, "Партия уже обновилась. Повторите действие.")
	}
	switch c.Action {
	case ActionReady:
		if r.swap != nil {
			return View{}, problem(409, "Сначала ответьте на предложение обмена сторонами.")
		}
		if r.started || r.winner != engine.NoWinner {
			return View{}, problem(409, "Подготовка партии уже завершена.")
		}
		m.ready = !m.ready
		full := true
		for _, pk := range r.players {
			full = full && pk != Key{} && r.members[pk] != nil && r.members[pk].ready
		}
		if full {
			r.started = true
			r.swap = nil
		}
	case ActionSeat:
		if r.started || r.winner != engine.NoWinner || c.Seat == nil || *c.Seat < 0 || *c.Seat >= len(r.players) {
			return View{}, problem(409, "Сторону можно выбрать перед началом партии.")
		}
		target := *c.Seat
		if target != actor {
			if r.players[target] == (Key{}) {
				r.players[target] = k
				r.players[actor] = Key{}
				r.swap = nil
			} else {
				r.swap = &Swap{From: actor, To: target}
			}
			r.clearReady()
		}
	case ActionAcceptSwap:
		if r.started || r.swap == nil || actor != r.swap.To {
			return View{}, problem(409, "Нет запроса на обмен сторонами.")
		}
		a, b := r.swap.From, r.swap.To
		r.players[a], r.players[b] = r.players[b], r.players[a]
		r.swap = nil
		r.clearReady()
	case ActionCancelSwap:
		if r.started || r.swap == nil || (actor != r.swap.From && actor != r.swap.To) {
			return View{}, problem(409, "Нет запроса на обмен сторонами.")
		}
		r.swap = nil
	case ActionMove:
		if !r.started || r.winner != engine.NoWinner {
			return View{}, problem(409, "Ждём игроков или новую партию.")
		}
		next := r.engine.Clone()
		kind := c.Kind
		if kind == "" {
			kind = "move"
		}
		if err := next.Apply(actor, engine.Action{Kind: kind, Cell: c.Cell, Vertical: c.Vertical}); err != nil {
			return View{}, problem(409, err.Error())
		}
		old := r.engine
		r.engine = next
		if s := next.View(-1); s.Phase == engine.Finished {
			if err := h.finish(r, s.Winner, ""); err != nil {
				r.engine = old
				return View{}, err
			}
		}
	default:
		return View{}, problem(400, "Неизвестное действие.")
	}
	h.notify(r.id)
	return h.view(r, k), nil
}
func (r *room) clearReady() {
	for _, m := range r.members {
		m.ready = false
	}
}

// NormalizeCode accepts pasted formatting while keeping one canonical identifier.
func NormalizeCode(code string) string {
	code = strings.ToUpper(strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' {
			return -1
		}
		return r
	}, strings.TrimSpace(code)))
	if len(code) != CodeLength {
		return ""
	}
	for _, r := range code {
		if !strings.ContainsRune(CodeAlphabet, r) {
			return ""
		}
	}
	return code
}
func (h *Hub) admit(extra int) bool {
	used := len(h.rooms)*roomBytes + len(h.subs)*streamBytes
	for _, r := range h.rooms {
		used += max(len(r.members), r.memberCapacity) * memberBytes
	}
	return used+extra <= StateBudgetBytes
}
func (h *Hub) enter(k Key, c Command) (View, error) {
	if c.Action == ActionCreate {
		c.Watch = false
		c.Room = ""
	}
	c.Name = strings.TrimSpace(c.Name)
	if !validName(c.Name) {
		return View{}, problem(400, "Имя: от 1 до 24 символов без управляющих знаков.")
	}
	r := h.find(c.Room)
	if c.Action == ActionEnter && r == nil {
		return View{}, problem(404, "Комната уже закрыта.")
	}
	if !c.Watch {
		for _, other := range h.rooms {
			if other != r && other.role(k) >= 0 && other.winner == engine.NoWinner {
				return View{}, problem(409, "Сначала выйдите из своей текущей комнаты.")
			}
		}
	}
	extra := 0
	if r == nil || r.members[k] == nil {
		extra = memberBytes
	}
	if c.Action == ActionCreate {
		extra += roomBytes
	}
	if !h.admit(extra) {
		return View{}, problem(503, "Игре не хватает выделенных ресурсов. Повторите позже.")
	}
	var previous *room
	if c.Action == ActionCreate {
		if c.From != "" {
			previous = h.find(c.From)
			if previous == nil || previous.role(k) < 0 || previous.winner == engine.NoWinner {
				return View{}, problem(409, "Новая партия доступна игрокам завершённой комнаты.")
			}
			if previous.nextRoom != "" && h.find(previous.nextRoom) != nil {
				return View{}, problem(409, "Новая комната уже создана. Нажмите «Продолжить».")
			}
		}
		if c.Game == "" {
			c.Game = "tictactoe"
		}
		if c.Size == 0 {
			c.Size = 3
		}
		if c.Players == 0 {
			c.Players = 2
		}
		g, err := games.New(c.Game, c.Size, c.Players)
		if err != nil {
			return View{}, problem(400, err.Error())
		}
		if c.Seat != nil && (*c.Seat < 0 || *c.Seat >= c.Players) {
			return View{}, problem(400, "Выберите доступную сторону.")
		}
		code := ""
		for attempt := 0; attempt < CodeAttempts; attempt++ {
			candidate := make([]byte, CodeLength)
			for i := range candidate {
				n, err := rand.Int(rand.Reader, big.NewInt(int64(len(CodeAlphabet))))
				if err != nil {
					return View{}, problem(503, "Не удалось создать комнату.")
				}
				candidate[i] = CodeAlphabet[n.Int64()]
			}
			if h.find(string(candidate)) == nil {
				code = string(candidate)
				break
			}
		}
		if code == "" {
			return View{}, problem(503, "Не удалось создать код комнаты.")
		}
		r = &room{id: code, game: c.Game, size: c.Size, round: 1, players: make([]Key, c.Players), members: make(map[Key]*member), engine: g, winner: engine.NoWinner}
		h.rooms = append(h.rooms, r)
	}
	actor := r.role(k)
	if actor >= 0 && c.Watch {
		return View{}, problem(409, "Вы уже игрок в этой комнате.")
	}
	if !c.Watch && actor < 0 {
		if r.winner != engine.NoWinner {
			return View{}, problem(409, "Партия завершена. Можно смотреть или перейти в новую комнату.")
		}
		if c.Seat != nil {
			if *c.Seat < 0 || *c.Seat >= len(r.players) || r.players[*c.Seat] != (Key{}) {
				return View{}, problem(409, "Выбранная сторона занята.")
			}
			actor = *c.Seat
		} else {
			for i, pk := range r.players {
				if pk == (Key{}) {
					actor = i
					break
				}
			}
		}
		if actor < 0 {
			return View{}, problem(409, "Все места заняты. Можно войти зрителем.")
		}
		r.players[actor] = k
		r.swap = nil
		r.clearReady()
	}
	if r.members[k] == nil {
		r.members[k] = &member{}
	}
	r.memberCapacity = max(r.memberCapacity, len(r.members))
	r.members[k].name = c.Name
	r.members[k].seen = h.now()
	if previous != nil {
		previous.nextRoom = r.id
		h.notify(previous.id)
	}
	h.notify(r.id)
	h.schedule()
	return h.view(r, k), nil
}

// Stats exposes counts only; names and browser identities never enter telemetry.
func (h *Hub) Stats() map[string]int {
	h.mu.Lock()
	defer h.mu.Unlock()
	players, watchers := 0, 0
	for _, r := range h.rooms {
		for k := range r.members {
			if r.role(k) >= 0 {
				players++
			} else {
				watchers++
			}
		}
	}
	return map[string]int{"rooms": len(h.rooms), "players": players, "spectators": watchers, "streams": len(h.subs)}
}

func (r *room) compactMembers() {
	compact := make(map[Key]*member, len(r.members))
	for k, m := range r.members {
		compact[k] = m
	}
	r.members = compact
	r.memberCapacity = len(compact)
}
