.PHONY: dev test build

# Run the Go API on :8080 and the Vite dev server (proxies /api to it).
dev:
	go run . & bun run --cwd web dev; kill %1

test:
	bun run test

build:
	bun run build
