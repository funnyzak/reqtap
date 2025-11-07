package static

import "embed"

// Assets contains embedded static resources for the web console.
//
//go:embed index.html login.html css/* js/*
var Assets embed.FS
