// Package httpapi implements the same-origin JSON and event-stream transport.
package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Kirill555dg/mini-games/internal/rooms"
	"github.com/Kirill555dg/mini-games/internal/telemetry"
	"io"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const Cookie = "__Secure-game_session"

type ErrorResponse struct {
	Error string `json:"error"`
}
type VisitResponse struct {
	OK bool `json:"ok"`
}
type Status struct {
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"startedAt"`
	UptimeSeconds int64     `json:"uptimeSeconds"`
	Version       string    `json:"version"`
}

type Server struct {
	hub                     *rooms.Hub
	origin, version, cookie string
	secure                  bool
	started                 time.Time
	counters                *telemetry.Counters
	slots                   chan struct{}
	AccessLog               bool
}

func New(h *rooms.Hub, origin, version string, dev bool) *Server {
	cookie := Cookie
	if dev {
		cookie = "game_session"
	}
	counters, _ := telemetry.New("")
	return &Server{hub: h, origin: origin, version: version, cookie: cookie, secure: !dev, started: time.Now().UTC(), counters: counters, slots: make(chan struct{}, MaxFiniteRequests)}
}

func (s *Server) SetCounters(c *telemetry.Counters) { s.counters = c }

type response struct {
	http.ResponseWriter
	status int
}

func (w *response) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *response) WriteHeader(status int)      { w.status = status; w.ResponseWriter.WriteHeader(status) }
func (s *Server) key(r *http.Request) rooms.Key {
	c, err := r.Cookie(s.cookie)
	if err != nil || len(c.Value) != 43 {
		return rooms.Key{}
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil || len(raw) != 32 {
		return rooms.Key{}
	}
	return sha256.Sum256([]byte(c.Value))
}
func (s *Server) identity(w http.ResponseWriter, r *http.Request) (rooms.Key, error) {
	if k := s.key(r); k != [32]byte{} {
		return k, nil
	}
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return rooms.Key{}, err
	}
	v := base64.RawURLEncoding.EncodeToString(b[:])
	http.SetCookie(w, &http.Cookie{Name: s.cookie, Value: v, Path: "/game-api/", Secure: s.secure, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 86400})
	return sha256.Sum256([]byte(v)), nil
}
func write(w http.ResponseWriter, status int, value any) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(WriteTimeout))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, err error) {
	var e *rooms.Error
	if errors.As(err, &e) {
		write(w, e.Status, map[string]string{"error": e.Message})
	} else {
		write(w, 503, map[string]string{"error": "Служба игры временно недоступна. Повторите позже."})
	}
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/game-api/events" {
		select {
		case s.slots <- struct{}{}:
			defer func() { <-s.slots }()
		default:
			s.counters.Request(http.StatusServiceUnavailable)
			write(w, 503, map[string]string{"error": "Игре не хватает выделенных ресурсов. Повторите позже."})
			return
		}
	}
	if r.URL.Path == "/internal/metrics" {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		write(w, 200, map[string]any{"application": s.counters.Snapshot(), "rooms": s.hub.Stats(), "heapBytes": m.HeapAlloc, "runtimeBytes": m.Sys, "goroutines": runtime.NumGoroutine(), "uptimeSeconds": int64(time.Since(s.started).Seconds()), "version": s.version})
		return
	}
	start := time.Now()
	tracked := &response{ResponseWriter: w, status: 200}
	w = tracked
	defer func() {
		s.counters.Request(tracked.status)
		if s.AccessLog {
			route := "other"
			switch r.URL.Path {
			case "/game-api/action":
				route = "action"
			case "/game-api/events":
				route = "events"
			case "/game-api/lobby":
				route = "lobby"
			case "/game-api/state":
				route = "state"
			case "/game-api/history":
				route = "history"
			case "/game-api/status":
				route = "status"
			case "/game-api/visit":
				route = "visit"
			}
			log.Printf("request route=%s status=%d duration_ms=%d", route, tracked.status, time.Since(start).Milliseconds())
		}
	}()

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if origin := r.Header.Get("Origin"); origin != "" && origin != s.origin {
		write(w, 403, map[string]string{"error": "Запрос с другого сайта отклонён."})
		return
	}
	if r.URL.Path == "/game-api/action" {
		s.action(w, r)
		return
	}
	if r.URL.Path == "/game-api/visit" {
		if r.Method != http.MethodPost || r.Header.Get("Origin") != s.origin {
			write(w, 403, map[string]string{"error": "Запрос отклонён."})
			return
		}
		c := http.NewResponseController(w)
		_ = c.SetReadDeadline(time.Now().Add(ReadTimeout))
		defer c.SetReadDeadline(time.Time{})
		if _, err := io.Copy(io.Discard, http.MaxBytesReader(w, r.Body, 64)); err != nil {
			write(w, 400, ErrorResponse{"Invalid visit request"})
			return
		}
		k, err := s.identity(w, r)
		if err != nil {
			fail(w, err)
			return
		}
		s.counters.Visit(k)
		write(w, 200, VisitResponse{true})
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		write(w, 405, map[string]string{"error": "Нужен GET."})
		return
	}
	switch r.URL.Path {
	case "/game-api/lobby":
		write(w, 200, s.hub.Lobby())
	case "/game-api/history":
		write(w, 200, s.hub.History())
	case "/game-api/status":
		write(w, 200, Status{"ok", s.started, int64(time.Since(s.started).Seconds()), s.version})
	case "/game-api/state":
		v, err := s.hub.View(r.URL.Query().Get("room"), s.key(r))
		if err != nil {
			fail(w, err)
			return
		}
		write(w, 200, v)
	case "/game-api/events":
		s.events(w, r)
	default:
		write(w, 404, map[string]string{"error": "Такого адреса нет."})
	}
}
func (s *Server) action(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		write(w, 405, map[string]string{"error": "Нужен POST."})
		return
	}
	if r.Header.Get("Origin") != s.origin || strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]) != "application/json" {
		write(w, 403, map[string]string{"error": "Запрос отклонён."})
		return
	}
	c := http.NewResponseController(w)
	_ = c.SetReadDeadline(time.Now().Add(ReadTimeout))
	defer c.SetReadDeadline(time.Time{})
	var in rooms.Command
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBody))
	d.DisallowUnknownFields()
	if d.Decode(&in) != nil || d.Decode(&struct{}{}) != io.EOF {
		write(w, 400, map[string]string{"error": "Неверная команда."})
		return
	}
	k, err := s.identity(w, r)
	if err != nil {
		fail(w, err)
		return
	}
	v, err := s.hub.Do(k, in)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, 200, v)
}
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	sub, err := s.hub.Subscribe(r.URL.Query().Get("room"), s.key(r))
	if err != nil {
		fail(w, err)
		return
	}
	defer s.hub.Unsubscribe(sub)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("X-Accel-Buffering", "no")
	c := http.NewResponseController(w)
	send := func() error {
		value := s.hub.Event(sub)
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_ = c.SetWriteDeadline(time.Now().Add(WriteTimeout))
		if _, err = fmt.Fprintf(w, "event: state\ndata: %s\n\n", data); err != nil {
			return err
		}
		return c.Flush()
	}
	if send() != nil {
		return
	}
	ticker := time.NewTicker(EventHeartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case _, ok := <-sub.Change:
			if !ok || send() != nil {
				return
			}
		case <-ticker.C:
			if send() != nil {
				return
			}
		}
	}
}
