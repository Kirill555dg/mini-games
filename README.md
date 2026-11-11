# Mini-games

A small multiplayer mini-game site. Go's standard library on the server, Preact and TypeScript in the browser. Russian and English, light and dark themes, phone/tablet/desktop layouts. Original SVG illustrations and an MIT license.

- Tic-tac-toe: a quick classic 3×3 round.
- Nine boards: nested tic-tac-toe with forced destinations and closed local boards. [Rules and design research](docs/game-design.md)
- Sea battle: the full classic 10×10 fleet, plus a quick 8×8 mode. Drag, rotate and move ships, or randomize placement.
- Four in a row: a gravity-based two-player game.
- Dots and boxes: two to four players, captured boxes and extra turns.

Create a room and share its six-character invitation. Invite links can suggest a side or open spectator mode. Players select sides, agree to swaps and confirm readiness before starting. A new game creates a new room; everyone in the previous room sees a continuation link. Names are remembered in the browser. Spectators receive public state only.

## Develop locally

Install Go 1.22 or newer, Bun 1.2.21 or newer, and Python 3 for release packaging. Development needs no container or database.

```sh
cd web
bun install --frozen-lockfile
bun run check
bun run build
cd ..
go run ./cmd/arcade -listen 127.0.0.1:8181 -history /tmp/mini-games-history.json -web ./web/dist
```

Open [the local site](http://127.0.0.1:8181). Use independent browser profiles to test different players. Plain HTTP and the local cookie name are enabled only with an explicit loopback development listener.

## Structure and rules

`cmd/arcade` wires the process. `internal/rooms` owns membership, preparation and subscriptions; `internal/httpapi` owns transport; `internal/history` owns completed public records; `internal/availability` owns bounded status observations. Each game has its own flat package behind the small `engine.Game` interface. [Backend postulates](AGENTS.md)

The frontend follows FSD: app, pages, widgets, features, entities, shared. Modules use descriptive filenames and `@/` imports. Bun builds and minifies all assets; no runtime CDN or web fonts. Biome checks formatting, code and accessibility, TypeScript checks the generated API, and a boundary check enforces FSD. [Frontend postulates](web/AGENTS.md)

## API generation

The Go wire types own the public contract. Generate OpenAPI and the client after changing a wire type:

```sh
go run ./cmd/apigen
cd web
bun run generate
bun run check
```

[OpenAPI](api/openapi.json) generates the `openapi-fetch` client's types and shared code/name/protocol constants. Networking, cancellation and SSE recovery are handwritten transport policies; endpoint request/response types are generated. Generated files are committed and checked for drift during a release build.

## Resources, privacy and history

There is no product limit of two rooms or a fixed number of spectators. Admission is constrained by a conservative 16 MiB state budget, bounded HTTP concurrency, connections and event streams. Extra viewers use polling when the stream budget is full. Exhausted capacity returns an error. Room lists are bounded summaries; invitations locate rooms directly.

Use service-manager limits in production: a low CPU quota, hard memory limit, no swap allocation and low scheduling priority. Application admission supplements these operating-system limits. The public status page shows current application uptime, response time and 30 UTC days of application self-observations. A sample runs once per minute. Only short consecutive sample intervals count as observed running time; restarts, long gaps and time before collection remain unknown. This is not an external availability guarantee. Daily API health is green with no observed server errors, yellow with some, and red when every observed request failed with a server error; missing coverage is separate from that health classification.

Browser identities are opaque private cookies; only hashes remain in memory. Fleet placement never appears in opponents' snapshots, spectator events, history or logs. The history retains the last 100 completed games with names and public moves, using atomic replacement. A lost connection has 90 seconds to return. Leaving a two-player match forfeits it; leaving a larger match abandons it.

Sea battle drafts use stable ship IDs from the server-supplied full fleet. `fleet` validates and atomically replaces a partial draft; `deploy` validates the complete fleet and confirms readiness in one action. Invalid positions leave the previous draft unchanged. Only the owning player receives draft placements, fleet lengths and their required separation. Side swaps are locked before placement starts.

Optional `-access-log` records only a fixed route category, status and timing. It does not record IP addresses, request URLs, cookies, names or bodies. Optional `-metrics FILE` retains up to 14 days of anonymous page-view and visit counters; the bounded browser cache lasts 30 minutes and is never persisted. The private rolling HTTP counters reset on process restart. Public daily health counts and sampled running time survive restarts in a separate, atomically replaced `status.json` beside the configured match-history file, limited to 30 days and 32 KiB. An empty `-history` path keeps both histories in memory. `/internal/metrics` is for a private Unix socket consumer and must not be routed publicly.

## Build and release

Commit reviewed source, then build on your development machine. Go cross-compiles static Linux binaries from macOS; the server needs no toolchain.

```sh
python3 tools/build-release.py --version v0.3.0
gh release create v0.3.0 dist/mini-games-linux-amd64.tar.gz dist/mini-games-linux-arm64.tar.gz dist/SHA256SUMS --title v0.3.0 --generate-notes
```

Each archive includes the stripped binary, minified local static assets, source revision, file hashes and license notices. Pin the archive SHA256 in the consuming deployment before installing. Hosting integrations and machine-specific configuration belong in the consuming project.

Match history is disposable: retired formats are reset instead of migrated. This release starts a new match history. Application availability is stored separately and retains its rolling 30-day window across updates and restarts.

Production runs `mini-games` with `DOMAIN` set and its Unix socket accessible only to the web server. Serve the packaged static directory and route `/game-api/` to that socket. Never expose `/internal/metrics` on the public web server. The default paths are `/run/mini-games/game.sock` and `/var/lib/mini-games/stats.json`; `-socket` and `-history` can override them.

## License

[MIT](LICENSE). Application code, text, game illustrations and the logo are original. The game rules are implemented independently. Downloaded Lucide icons retain their upstream license and revision in [the icon assets](web/public/icons/SOURCE.txt). [Third-party notices](THIRD-PARTY-NOTICES.txt) cover icons and runtime dependencies; build tooling is listed in the locked frontend manifest.
