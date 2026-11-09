# JobSeek

Job application tracker. Go API (flat `package main`: `main.go`, `api.go`, `db.go`, pure-Go SQLite, no CGO) + Svelte 5 / TypeScript / Tailwind v4 / shadcn-svelte frontend in `web/` (Vite).

**End goal:** a single binary (frontend embedded in the Go server) released via GitHub Releases, so installing it gives the whole app. Not built yet. Don't break the path: frontend builds to `web/dist`, no CGO, frontend only uses relative `/api` URLs.

## JobTweak (AI resume tailoring)
Second tab (`#/jobtweak`). Files: `resume.go` (model, storage, proposals), `resume_api.go` (REST), `mcp.go` (MCP tools at `/mcp`), `agent.go` (chat: runs the user's own `claude` CLI), web `JobTweakView` / `ResumeEditor` / `ResumeDocument` / `ChatPanel`.
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

## Design language (Notion-style)
- All colors, radius and fonts come from tokens in `web/src/app.css`. No hardcoded hex in components. Add a token (plus its `@theme inline` entry) instead. Status tag colors are `--tag-<status>-{bg,fg,dot}`; errors use `danger*`.
- Use shadcn-svelte primitives from `$lib/components/ui` before building anything custom.
- Compact and quiet: `text-sm`, `h-8` table headers, `p-1` cells, borderless inline-editable cells, hover `bg-accent/60`, focus `bg-accent`, small `rounded-sm` tags.
- Destructive/error styling uses `danger` tokens.

## Shared-component rule
Every UI pattern lives in exactly one component, so a change there applies everywhere.
1. Before changing a component, grep for all its usages (`rg "ComponentName" web/src`).
2. Make the change in the shared component or token, never at one call site. If a call site needs a variant, add a prop.
3. If a pattern appears a second time, extract it into `web/src/lib/components/` instead of copying markup (e.g. `ErrorBanner`, `StatusBadge`).
4. After changing, check every usage still looks right and its tests pass.
