VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build client server seed bridge run run-server run-client run-bridge clean

build: client server seed bridge

client:
	go build $(LDFLAGS) -o bin/chit ./cmd/client/

server:
	go build $(LDFLAGS) -o bin/chit-server ./cmd/server/

seed:
	go build $(LDFLAGS) -o bin/seed ./cmd/seed/

bridge:
	go build $(LDFLAGS) -o bin/chit-bridge ./cmd/bridge/

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
	rm -rf bin/
