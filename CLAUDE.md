# JobSeek

Job application tracker. Go API (flat `package main`: `main.go`, `api.go`, `db.go`, pure-Go SQLite, no CGO) + Svelte 5 / TypeScript / Tailwind v4 / shadcn-svelte frontend in `web/` (Vite).

**End goal:** a single binary (frontend embedded in the Go server) released via GitHub Releases, so installing it gives the whole app. Not built yet. Don't break the path: frontend builds to `web/dist`, no CGO, frontend only uses relative `/api` URLs.

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
