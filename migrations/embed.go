package migrations

import "embed"

// Files contains immutable, ordered database migrations shipped in the binary.
//
//go:embed *.sql
var Files embed.FS
