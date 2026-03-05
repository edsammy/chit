VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build client server seed bridge run run-server run-client run-bridge clean

build: client server seed bridge

client:
	go build $(LDFLAGS) -o chit ./cmd/client/

server:
	go build $(LDFLAGS) -o chit-server ./cmd/server/

seed:
	go build $(LDFLAGS) -o seed ./cmd/seed/

bridge:
	go build $(LDFLAGS) -o chit-bridge ./cmd/bridge/

run: run-server

run-server:
	./chit-server serve --http 0.0.0.0:8090

run-client:
	./chit

run-bridge:
	./chit-bridge

seed-defaults:
	./seed defaults

clean:
	rm -f chit chit-server seed chit-bridge
