//go:build !dev

package frontend

import (
	"embed"
	"io/fs"
)

//go:embed dist
var distFiles embed.FS

// DistFS is the embedded frontend asset filesystem.
var DistFS fs.FS = distFiles
