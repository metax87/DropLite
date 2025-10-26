package migrations

import "embed"

// UpFiles embeds all upward migrations for use by the migrator.
//
//go:embed *.up.sql
var UpFiles embed.FS
