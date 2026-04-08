//go:build dev

package frontend

import (
	"io/fs"
	"testing/fstest"
)

// DistFS is a stub filesystem used in local development.
// In dev mode the frontend is served by the Vite dev server (make dev-frontend),
// so the Go server only needs enough content to start without fatal errors.
var DistFS fs.FS = fstest.MapFS{
	"dist/index.html":  {Data: []byte(`<!DOCTYPE html><html><head><title>Dev Mode</title></head><body><p>Backend running. Start the frontend with: make dev-frontend</p></body></html>`)},
	"dist/assets":      {Mode: fs.ModeDir},
	"dist/favicon.svg": {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
	"dist/vite.svg":    {Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
}
