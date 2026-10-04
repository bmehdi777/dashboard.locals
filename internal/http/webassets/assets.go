package webassets

import (
	"embed"
	"io/fs"
)

// Embedded contains the optional production frontend and the server-side
// fallback page. The production build copies the frontend output into static/
// before compiling the binary; generated files are intentionally not tracked.
//
//go:embed fallback/* static/.keep static/*
var Embedded embed.FS

// DefaultFS prefers compiled frontend assets and falls back to the minimal
// server page when no frontend build has been embedded yet.
func DefaultFS() fs.FS {
	static, err := fs.Sub(Embedded, "static")
	if err == nil {
		if _, statErr := fs.Stat(static, "index.html"); statErr == nil {
			return static
		}
	}
	fallback, err := fs.Sub(Embedded, "fallback")
	if err != nil {
		return Embedded
	}
	return fallback
}
