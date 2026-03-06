VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

VPS := chit@chit.nance.app
VPS_DIR := /opt/chit

.PHONY: build client server seed bridge cross cross-all deploy deploy-remote run run-server run-client run-bridge clean

build: client server seed bridge

client:
	go build $(LDFLAGS) -o bin/chit ./cmd/client/

server:
	go build $(LDFLAGS) -o bin/chit-server ./cmd/server/

seed:
	go build $(LDFLAGS) -o bin/seed ./cmd/seed/

bridge:
	go build $(LDFLAGS) -o bin/chit-bridge ./cmd/bridge/

cross:
	mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/chit-darwin-arm64 ./cmd/client/
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/chit-darwin-amd64 ./cmd/client/
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/chit-linux-amd64 ./cmd/client/

cross-all: cross
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/chit-server ./cmd/server/
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/chit-bridge ./cmd/bridge/
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/seed ./cmd/seed/

deploy: build cross
	sudo systemctl restart chit-server chit-bridge

deploy-remote: cross-all
	ssh $(VPS) "sudo systemctl stop chit-server chit-bridge"
	scp dist/chit-server dist/chit-bridge dist/seed $(VPS):$(VPS_DIR)/bin/
	scp dist/chit-darwin-* dist/chit-linux-* $(VPS):$(VPS_DIR)/dist/
	ssh $(VPS) "sudo systemctl start chit-server chit-bridge"

run: run-server

run-server:
	bin/chit-server serve --http 0.0.0.0:8090

run-client:
	bin/chit

run-bridge:
	@test -f .bridge.env || (echo "create .bridge.env first (see .bridge.env.example)" && exit 1)
	set -a && . ./.bridge.env && set +a && bin/chit-bridge

seed-defaults:
	bin/seed defaults

invite:
	bin/seed invite

clean:
	rm -rf bin/ dist/
