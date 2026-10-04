package webassets

import "embed"

// Embedded contains the server-side fallback page. A production frontend
// build can replace the contents of static/ as part of the release build
// without changing the HTTP server.
//
//go:embed static/*
var Embedded embed.FS
