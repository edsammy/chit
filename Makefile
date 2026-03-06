VERSION := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build client server seed bridge cross deploy run run-server run-client run-bridge clean

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

deploy:
	@echo "Building (services stay running)..."
	nice -n 19 go build $(LDFLAGS) -o bin/chit-server.new ./cmd/server/
	nice -n 19 go build $(LDFLAGS) -o bin/chit-bridge.new ./cmd/bridge/
	nice -n 19 go build $(LDFLAGS) -o bin/seed ./cmd/seed/
	mkdir -p dist
	nice -n 19 env GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/chit-darwin-arm64 ./cmd/client/
	nice -n 19 env GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/chit-darwin-amd64 ./cmd/client/
	nice -n 19 env GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/chit-linux-amd64 ./cmd/client/
	@echo "Swapping and restarting..."
	mv bin/chit-server.new bin/chit-server
	mv bin/chit-bridge.new bin/chit-bridge
	sudo systemctl restart chit-server chit-bridge
	@echo "Deploy complete"

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
