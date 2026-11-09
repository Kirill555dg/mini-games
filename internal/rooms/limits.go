package rooms

import "time"

// Admission is based on a memory budget, not a product limit on room counts.
const StateBudgetBytes = 16 * 1024 * 1024
const roomBytes = 16 * 1024
const memberBytes = 512
const streamBytes = 64 * 1024
const CodeLength = 6
const CodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const MaxEventStreams = 128
const GracePeriod = 90 * time.Second
const ExpireInterval = 5 * time.Second
const MaxActionsPerSecond = 60
const LobbyPageSize = 32
const MaxNameRunes = 24
const PresenceWindow = 25 * time.Second
const CodeAttempts = 32
