package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/Kirill555dg/mini-games/internal/rooms"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testAPI(t *testing.T) (*Server, *rooms.Hub) {
	h, err := rooms.New("")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	return New(h, "https://example.test", "test", false), h
}
func TestSameOriginValidationAndPrivateCookie(t *testing.T) {
	s, _ := testAPI(t)
	for _, test := range []struct {
		body, origin string
		status       int
	}{{`{"action":"create","name":"Игрок"}`, "https://evil.test", 403}, {`{"action":"create","name":"Игрок","extra":1}`, "https://example.test", 400}, {strings.Repeat(" ", MaxBody+1), "https://example.test", 400}} {
		r := httptest.NewRequest("POST", "/game-api/action", strings.NewReader(test.body))
		r.Header.Set("Origin", test.origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != test.status {
			t.Fatalf("expected %d, got %d", test.status, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/game-api/action", strings.NewReader(`{"action":"create","name":"Игрок"}`))
	r.Header.Set("Origin", "https://example.test")
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal("create failed")
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].Path != "/game-api/" {
		t.Fatal("private cookie policy")
	}
	if bytes.Contains(w.Body.Bytes(), []byte(cookies[0].Value)) {
		t.Fatal("session leaked in JSON")
	}
}

type slowWriter struct {
	header  http.Header
	entered chan struct{}
	release chan struct{}
}

func (w *slowWriter) Header() http.Header         { return w.header }
func (w *slowWriter) WriteHeader(int)             {}
func (w *slowWriter) Write(p []byte) (int, error) { close(w.entered); <-w.release; return len(p), nil }
func TestSlowReceiverDoesNotHoldRoomMutex(t *testing.T) {
	s, h := testAPI(t)
	w := &slowWriter{http.Header{}, make(chan struct{}), make(chan struct{})}
	done := make(chan struct{})
	go func() { s.ServeHTTP(w, httptest.NewRequest("GET", "/game-api/lobby", nil)); close(done) }()
	<-w.entered
	read := make(chan struct{})
	go func() { h.Lobby(); close(read) }()
	select {
	case <-read:
	case <-time.After(time.Second):
		close(w.release)
		t.Fatal("network write holds room mutex")
	}
	close(w.release)
	<-done
}
func TestUptimeAndNoHistoryInLobby(t *testing.T) {
	s, _ := testAPI(t)
	for _, path := range []string{"lobby", "status"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "/game-api/"+path, nil))
		var v map[string]any
		if json.Unmarshal(w.Body.Bytes(), &v) != nil {
			t.Fatal("invalid JSON")
		}
		if path == "status" {
			if v["status"] != "ok" || v["startedAt"] == nil || v["uptimeSeconds"] == nil {
				t.Fatal("missing real uptime")
			}
		} else if _, ok := v["history"]; ok {
			t.Fatal("history bundled into lobby")
		}
	}
}

func TestOverloadFailsFastAndVisitBodyIsBounded(t *testing.T) {
	s, _ := testAPI(t)
	for n := 0; n < MaxFiniteRequests; n++ {
		s.slots <- struct{}{}
	}
	for _, path := range []string{"/game-api/lobby", "/internal/metrics"} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 503 {
			t.Fatal("request bypassed admission")
		}
	}
	for n := 0; n < MaxFiniteRequests; n++ {
		<-s.slots
	}
	r := httptest.NewRequest("POST", "/game-api/visit", strings.NewReader(strings.Repeat("x", 65)))
	r.Header.Set("Origin", "https://example.test")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("unbounded visit body")
	}
}
