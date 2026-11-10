# Frontend development rules

- Use TypeScript, Preact, native HTML controls and CSS. Bun owns installation, code generation, checks and builds. Dependencies must justify their runtime and maintenance cost. Build tooling stays on the development machine.
- Follow FSD dependency direction: `app -> pages -> widgets -> features -> entities -> shared`. Add a layer only when it has a real responsibility. Use descriptive module names, never generic `index.js`. Use absolute `@/` imports; no cross-slice imports at the same layer. `check-boundaries.ts` enforces this.
- `app` wires navigation and lifecycle; pages compose screens; widgets render independent areas; features implement user actions; entities describe game/room state; shared owns transport and small UI primitives.
- Keep source readable. Bundle, minify and fingerprint production JS/CSS. Serve all assets locally; no CDN, web fonts or third-party trackers. Keep first-load size and interaction latency small.
- Consume the generated OpenAPI client types through the named API module. Regenerate rather than duplicating routes, DTOs or domain constraints. Constants belong to their owning module; persist only display preferences and the display name in local storage.
- Derive game options, rules, capacity and victory conditions from the server catalog/state. Never reconstruct authorization or hidden game state in the browser.
- Make create, join, watch, play and review direct. Rooms have URLs. Keep join actions next to their code. A playing room has one explicit exit; explain forfeiture before confirming it.
- Use semantic native controls, visible labels, keyboard focus, readable contrast, reduced-motion support and useful live status. Every icon action needs an accessible name.
- Fit boards to width and available viewport height. Verify phone, tablet, desktop and short landscape screens. Touch interaction must not require hover or drag.
- Preserve input after failure. Show busy, disconnected, stale, empty and error states. Prevent duplicate actions and stale async responses after navigation.
- Use SSE with bounded reconnect and polling fallback. Stop streams/timers on navigation; restore a room after refresh. Never log or export browser identities.
- Keep styles based on named tokens and component responsibility. Use consistent spacing, one primary action per task and native Dialog only for bounded confirmation.
- Run `bun run check` (Biome, FSD boundaries, TypeScript and behavior tests), then `bun run build`. Validate the production build in a browser with independent player contexts. Cover full games, spectator permissions, reconnect, new room continuation, history and responsive layout. Keep screenshots outside Git.

- Game interactions come first. Use visible buttons/segmented choices for short option sets, direct room entry on the home screen, and game-specific side selection in the room. Native select is not the default for every setting.
- Keep SVG artwork and icons in public assets, with source/license attribution for downloaded icons. Do not embed path data in TSX. Original game artwork remains MIT.
- Move review belongs only to completed match history. A live room focuses on the current board and next action.
- Every drag interaction needs an equivalent tap and keyboard flow. Preserve drafts across refreshes from the server and failures; validate all placements server-side.

References: [FSD layers](https://feature-sliced.design/docs/reference/layers), [Anthropic frontend-design](https://github.com/anthropics/skills/tree/main/skills/frontend-design), [Vercel Web Interface Guidelines](https://github.com/vercel-labs/web-interface-guidelines). Design for a friendly multiplayer game site, with consistent interactions, accessible controls and local assets.
