# Thufir build system
#
# Targets:
#   make          — full production build (frontend + embed + Go binary)
#   make dev      — build frontend once, then run Go server
#   make clean    — remove build artefacts
#   make run      — full build then execute the binary

BINARY    := thufir
EMBED_DIR := server/cmd/server/static
SERVER    := server

.PHONY: all build frontend server clean run dev

all: build

## Full production build.
build: frontend server

## Install JS deps (skipped when node_modules is up to date).
node_modules: package.json package-lock.json
	npm install
	@touch node_modules

## Build the frontend with Vite and copy assets into the Go embed path.
frontend: node_modules
	npm run build
	cp -r dist/. $(EMBED_DIR)/

## Compile the Go binary (embed path must be populated by the `frontend` target).
server:
	cd $(SERVER) && CGO_ENABLED=0 go build -ldflags="-s -w" -o ../$(BINARY) ./cmd/server

## Remove all build outputs.
clean:
	rm -rf dist $(BINARY)
	find $(EMBED_DIR) -mindepth 1 \
		! -name '.gitignore' ! -name 'placeholder.txt' \
		-delete 2>/dev/null || true

## Full build then run.
run: build
	./$(BINARY)

## Development: build frontend once, then run Go server (serves embedded assets).
## Re-run `make dev` after frontend changes to rebuild and restart.
dev: frontend
	cd $(SERVER) && go run ./cmd/server
