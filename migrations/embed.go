package migrations

import "embed"

// FS contains the goose SQL migrations. Keeping them embedded means the
// migrate binary is the only artifact a deploy needs.
//
//go:embed *.sql
var FS embed.FS
