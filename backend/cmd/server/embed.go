package main

import "embed"

// frontendFS holds the built Svelte SPA. During the Docker build the compiled
// assets are copied into ./frontend_dist before `go build`. For backend-only
// development the directory contains only a placeholder and the server falls
// back to a "frontend not built" response.
//
//go:embed all:frontend_dist
var frontendFS embed.FS
