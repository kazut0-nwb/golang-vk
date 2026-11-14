package migrations

import "embed"

// FS содержит SQL-миграции, встроенные в приложение.
//
//go:embed *.sql
var FS embed.FS
