package httpapi

import "time"

const (
	MaxBody           = 4096
	MaxFiniteRequests = 16
	MaxConnections    = 160
	MaxHeaderBytes    = 8192
	ReadTimeout       = 5 * time.Second
	WriteTimeout      = 5 * time.Second
	IdleTimeout       = 30 * time.Second
	EventHeartbeat    = 15 * time.Second
)
