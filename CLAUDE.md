# JobSeek

Job application tracker. Go API (flat `package main`: `main.go`, `api.go`, `db.go`, pure-Go SQLite, no CGO) + Svelte 5 / TypeScript / Tailwind v4 / shadcn-svelte frontend in `web/` (Vite).

**End goal:** a single binary (frontend embedded in the Go server) released via GitHub Releases, so installing it gives the whole app. Not built yet. Don't break the path: frontend builds to `web/dist`, no CGO, frontend only uses relative `/api` URLs.

## JobTweak (AI resume tailoring)
Second tab, **Tweak** (`#/tweak`; old `#/jobtweak` links still work). Code names keep the `JobTweak` prefix. Files: `resume.go` (model, storage, proposals), `resume_api.go` (REST), `mcp.go` (MCP tools at `/mcp`), `agent.go` (chat: runs the user's own `claude` CLI), web `JobTweakView` / `ResumeEditor` / `ResumeDocument` / `ChatPanel`.
- **AI edits follow the user’s request.** `update_resume` applies requested changes directly with revision checks. For reviewable changes, `propose_edit` supports item edits and `replace_document` (including section reordering); only the UI accept endpoint applies a pending proposal. Preserve document formatting and stable IDs.
- **Version AI sessions.** Claude CLI snapshots system prompts when a session starts. Bump `agentToolContractVersion` when tool contracts change; prompt/instruction changes are hashed automatically. Resume only sessions whose stored version matches, so old restrictions cannot persist after an upgrade.
- **Local only.** The server binds to 127.0.0.1 and `localOnly` rejects non-local Host/Origin (DNS rebinding, cross-site POSTs). There's no login, so don't loosen this without adding real auth. `/mcp` also needs the bearer token from the `settings` table.
- **The agent gets no built-in tools** (`--tools ""`, only `mcp__jobseek__*`, `dontAsk`) because job descriptions are untrusted text (prompt injection). The MCP token goes in a 0600 temp file, the message via stdin, never on argv.
- **Auth.** Never offer "Sign in with Claude" or touch Claude credentials: Anthropic doesn't allow third-party apps to. We spawn the user's own unmodified, signed-in `claude`. Shipping this to other people needs a licensing check first.
- Items (bullets, skills lines, summary) have stable IDs so suggestions target an item, not a position. Accepting returns 409 if the item's text changed since it was suggested.
- Schema changes: `CREATE TABLE IF NOT EXISTS` won't alter existing DBs, so add new columns in `migrate()` (`db.go`).
- Tests for agent code use a fake `claude` script (see `agent_test.go`); never call the real CLI in tests.

## Commands (use bun, not npm)
- `make dev`: Go API on :8080 + Vite dev server (proxies `/api`)
- `bun run test`: gofmt check + `go vet` + `go test` + svelte-check + Vitest
- `bun run build`: frontend build + `go build`

## Hooks (Husky)
- pre-commit: `bun run test`. pre-push: `bun run build` only.
- Never use `--no-verify`. Fix the failure instead.

## Testing rules
- Every feature or bugfix ships with tests in the same change.
- Backend: table-driven Go tests using `httptest` against `app.routes()` and a temp-file SQLite DB (see `api_test.go`).
- Frontend: Vitest + Testing Library, files next to the code as `*.test.ts`. Mock `$lib/api` for component tests.
- Test important behavior only (CRUD, validation, inline edit save/revert/rollback, filters, error states). Don't test styling or library internals.
- If you change the API, update `api.go`/`db.go`, `web/src/lib/{api,types}.ts`, and the tests on both sides together.
- `PUT /api/jobs/{id}` replaces the whole job, so every frontend update must send every field (use `toInput` from `$lib/job`).

## Design language
Near-monochrome ink on soft paper (light) / deep ink (dark). Theme is `<html data-theme="light|dark">`: `index.html` sets it before first paint from the saved choice (else the system), `ThemeSwitch` (Light/Dark/System, `$lib/theme`) changes it. Use the `dark:` variant or `:root[data-theme="dark"]` tokens, never `prefers-color-scheme` directly. One accent, daylight blue, for actions and "live" things.
- **Tokens only.** Colours, radii, shadows, easings and fonts all live in `web/src/app.css` (Tailwind v4 `@theme`, no config file). No hex, `bg-white`, or Tailwind palette colours (`amber-500`…) in components. Add a token plus its `@theme` entry instead.
- **Colours:** neutrals `canvas` (page) / `surface` (cards) / `sunken` (wells inside cards) / `wash` (tracks and chips on the canvas) / `line` / `ink`, `ink-muted`, `ink-faint`. The accent is `primary` (shadcn's `accent` is only the hover surface). Semantic `success`, `danger`, `warning` with `-soft` variants for tints.
- **Status = small coloured marks, never coloured boxes:** `bg-mark-<status>` dots via `StatusBadge`.
- **Paper:** the resume document always uses `paper*` tokens (white page, dark ink) in every theme, so the PDF is correct even in dark mode.
- **Fonts, one job each:** `font-display` (Funnel Display) headings h1–h3; `font-sans` (Funnel Sans) body; `numeral` (Fraunces, SOFT axis) big numbers and the one large title per screen; `hand` (Gaegu) short status lines and labels only. Self-hosted via Fontsource (works offline in the single binary).
- **Shape:** `rounded-chip` (pill) < `rounded-control` (16px) < `rounded-card` (24px) < `rounded-sheet` (32px, dialogs and side sheets). Elevation with `shadow-card` / `shadow-lift` / `shadow-press`, not borders. Lists use hairlines between rows (`rows` utility, table rows), never boxes per row.
- **Buttons** are textured (`tb tb-primary` / `tb tb-secondary`) via the shared `Button` variants; `ghost` stays quiet.
- **Glass** (`glass glass-sky`) is reserved for controls floating over the sky (`Hero`: sky gradient with small twinkling dots). Not a general surface.
- **Cursor:** a base rule gives every clickable element `cursor: pointer` (and disabled ones `not-allowed`). Don't add per-component cursor classes.
- **Motion:** use `ease-spring` / `ease-out-soft` and `duration-(--duration-fast|base|slow)`. Animations must have a `prefers-reduced-motion` fallback (a global rule covers CSS; JS-driven motion checks it, see `Hero`).
- White text on the sky must stay at or above 4.5:1 contrast. Re-check if you change `--sky-*`.
- Shared pieces: `Segmented` (tabs and toggles, optional icons), `Hero` (sky), `ThemeSwitch`, `StatusBadge`, `IconButton`, `ErrorBanner`.

## Shared-component rule
Every UI pattern lives in exactly one component, so a change there applies everywhere.
1. Before changing a component, grep for all its usages (`rg "ComponentName" web/src`).
2. Make the change in the shared component or token, never at one call site. If a call site needs a variant, add a prop.
3. If a pattern appears a second time, extract it into `web/src/lib/components/` instead of copying markup (e.g. `ErrorBanner`, `StatusBadge`).
4. After changing, check every usage still looks right and its tests pass.
