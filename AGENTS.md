# Backend development rules

- Keep `cmd/arcade` limited to configuration, wiring and process lifecycle.
- Keep packages flat under `internal`. Name each package after its responsibility. No `utils`, generic service layers, or interfaces without a consumer.
- `engine` owns the small game contract. Each game has its own package; it cannot import HTTP, rooms or persistence. `games` is the only game registry and source of public rules and capabilities.
- `rooms` owns membership, capacity, turns across rounds, presence and subscriptions. `httpapi` owns cookies, origins, request limits and transport. `history` owns versioned bounded persistence.
- The server validates every action. Never rely on a disabled button or client-supplied outcome. Copy state before mutation so failed persistence can roll back an entire action.
- Produce snapshots for the requesting viewer. Hidden information must never enter an opponent's response, spectator response, event, log or unfinished history.
- Admit rooms and spectators against a memory budget, without a fixed product cap. Bound connections, concurrency, payloads, history, telemetry and deadlines. Prefer a 503 over exceeding the resource budget. Never hold the room mutex while writing to a client.
- Store only hashes of opaque browser identities in memory. Do not log cookies, request bodies or session state. The game has no dependency on any other service.
- Make constants and enums belong to their domain. Derive frontend options from the game catalog rather than duplicating game rules.
- Prefer clear concrete types, explicit errors, composition and narrow interfaces. Apply SOLID where it reduces coupling; do not create a framework to satisfy an acronym.
- Run `gofmt`, `go vet ./...`, `go test -race ./...`. Test behavior: legal/illegal moves, private information, races, expiry, new room continuation, recovery and persistence failures.
- Go wire types and domain constants own the API contract. Regenerate OpenAPI and frontend types together; do not patch generated files or maintain a second schema by hand.
- Build stripped static Linux releases on the development machine or CI. VPS installation verifies and switches a prepared release; it never compiles or installs a toolchain.

References: [Google Go style](https://google.github.io/styleguide/go/), [Go module layout](https://go.dev/doc/modules/layout). Use only the directories this application needs; `golang-standards/project-layout` is a community reference, not an official standard.

- This is a public MIT-licensed mini-game project. Keep infrastructure integrations, machine addresses and unrelated deployment configuration outside this repository. Original code and SVG only; preserve dependency license notices in distributed assets.
- The game may fail under load. Resource isolation is more important than accepting a request: keep hard memory/CPU limits, bounded concurrency, visitors, telemetry and history.
