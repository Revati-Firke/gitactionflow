package migrations

import "embed"

// FS holds versioned SQL migration files.
//
//go:embed *.sql
var FS embed.FS
