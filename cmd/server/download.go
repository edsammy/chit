package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
)

var version = "dev"

var validPlatforms = map[string]bool{
	"darwin-arm64": true,
	"darwin-amd64": true,
	"linux-amd64":  true,
}

func registerDownload(se *core.ServeEvent) {
	se.Router.GET("/api/version", func(e *core.RequestEvent) error {
		return e.JSON(200, map[string]string{"version": version})
	})

	se.Router.GET("/download/{platform}", func(e *core.RequestEvent) error {
		platform := e.Request.PathValue("platform")
		if !validPlatforms[platform] {
			return e.JSON(404, map[string]string{"error": "unsupported platform: " + platform})
		}

		path := filepath.Join("dist", "chit-"+platform)
		if _, err := os.Stat(path); err != nil {
			return e.JSON(404, map[string]string{"error": "binary not available for " + platform})
		}

		e.Response.Header().Set("Content-Disposition", `attachment; filename="chit"`)
		http.ServeFile(e.Response, e.Request, path)
		return nil
	})

	se.Router.GET("/install.sh", func(e *core.RequestEvent) error {
		scheme := e.Request.Header.Get("X-Forwarded-Proto")
		if scheme == "" {
			scheme = "http"
		}
		base := scheme + "://" + e.Request.Host

		script := fmt.Sprintf(`#!/bin/sh
set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

PLATFORM="${OS}-${ARCH}"
URL="%s/download/${PLATFORM}"
INSTALL_DIR="$HOME/.local/bin"

mkdir -p "$INSTALL_DIR"

echo "Downloading chit (${PLATFORM})..."
curl -fSL -o "$INSTALL_DIR/chit" "$URL"
chmod +x "$INSTALL_DIR/chit"

echo "Installed chit to $INSTALL_DIR/chit"

# Check if it's in PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) echo "Add to your PATH: export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac
`, base)

		e.Response.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(e.Response, script)
		return nil
	})
}
