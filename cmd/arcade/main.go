// Command arcade serves a small bounded mini-game service behind Nginx.
package main

import (
	"context"
	"errors"
	"flag"
	"github.com/Kirill555dg/mini-games/internal/httpapi"
	"github.com/Kirill555dg/mini-games/internal/rooms"
	"github.com/Kirill555dg/mini-games/internal/telemetry"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version = "development"

func main() {
	socket := flag.String("socket", "/run/mini-games/game.sock", "Production Unix socket")
	history := flag.String("history", "/var/lib/mini-games/stats.json", "Completed public match history")
	listen := flag.String("listen", "", "Local development loopback listener")
	web := flag.String("web", "", "Static production build for local development")
	metrics := flag.String("metrics", "", "Optional bounded anonymous daily counters file")
	accessLog := flag.Bool("access-log", false, "Log safe route/status/timing metadata; no visitor information")
	flag.Parse()
	origin := "https://" + os.Getenv("DOMAIN")
	dev := *listen != ""
	if dev {
		host, _, err := net.SplitHostPort(*listen)
		if err != nil || host != "127.0.0.1" {
			log.Fatal("Development listener must use 127.0.0.1")
		}
		origin = "http://" + *listen
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (!dev && (u.Port() != "" || u.Hostname() != os.Getenv("DOMAIN"))) {
		log.Fatal("DOMAIN must be a hostname")
	}
	h, err := rooms.New(*history)
	if err != nil {
		log.Fatal("Cannot load match history; existing file preserved")
	}
	defer h.Close()
	api := httpapi.New(h, origin, version, dev)
	counters, err := telemetry.New(*metrics)
	if err != nil {
		log.Fatal("Cannot load anonymous counters; existing file preserved")
	}
	defer counters.Close()
	api.SetCounters(counters)
	api.AccessLog = *accessLog
	var handler http.Handler = api
	if dev && *web != "" {
		mux := http.NewServeMux()
		mux.Handle("/game-api/", api)
		mux.Handle("/internal/metrics", api)
		mux.Handle("/", http.FileServer(http.Dir(*web)))
		handler = mux
	}
	var listener net.Listener
	if dev {
		listener, err = net.Listen("tcp", *listen)
	} else {
		listener, err = net.Listen("unix", *socket)
	}
	if err != nil {
		log.Fatal("Cannot open game listener")
	}
	defer listener.Close()
	if !dev {
		if os.Chmod(*socket, 0660) != nil {
			log.Fatal("Cannot restrict game socket")
		}
	}
	listener = httpapi.LimitListener(listener, httpapi.MaxConnections)
	server := &http.Server{Handler: handler, ReadTimeout: httpapi.ReadTimeout, ReadHeaderTimeout: httpapi.ReadTimeout, IdleTimeout: httpapi.IdleTimeout, MaxHeaderBytes: httpapi.MaxHeaderBytes}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		h.Close()
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(c)
	}()
	log.Print("Mini-game service ready; access logging disabled")
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("Game server stopped unexpectedly")
	}
}
