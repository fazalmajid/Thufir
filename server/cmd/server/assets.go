package main

import "embed"

// staticFiles holds the pre-built frontend assets.
// The `static/` directory is populated by the Dockerfile (or manually for
// local testing) before `go build` runs — it is never committed to source
// control except for the placeholder .gitkeep.
//
//go:embed all:static
var staticFiles embed.FS
