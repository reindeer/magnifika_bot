package migrate

import (
	"embed"
)

//go:embed migrations/*.sql
var migrations embed.FS

func NewMigrations() embed.FS {
	return migrations
}
