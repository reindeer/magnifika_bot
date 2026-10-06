package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"

	"gitlab.com/gorib/pry"
)

//go:embed migrations/*.sql
var embedded embed.FS

func Migrations() (fs.FS, error) {
	return fs.Sub(embedded, "migrations")
}

func NewMigrator(provider *goose.Provider, logger pry.Logger) *Migrator {
	return &Migrator{provider: provider, logger: logger}
}

type Migrator struct {
	provider *goose.Provider
	logger   pry.Logger
}

func (m *Migrator) Run(ctx context.Context, args []string) error {
	command := "up"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "up":
		results, err := m.provider.Up(ctx)
		for _, result := range results {
			m.logger.Info(result, pry.Ctx(ctx))
		}
		return err
	case "down":
		result, err := m.provider.Down(ctx)
		if result != nil {
			m.logger.Info(result, pry.Ctx(ctx))
		}
		return err
	case "status":
		statuses, err := m.provider.Status(ctx)
		for _, status := range statuses {
			fields := []pry.Option{pry.Ctx(ctx), pry.Field("state", status.State)}
			if status.State == goose.StateApplied {
				fields = append(fields, pry.Field("applied", status.AppliedAt))
			}
			m.logger.Info(status.Source.Path, fields...)
		}
		return err
	default:
		return fmt.Errorf("unknown migrate command %q, expected up, down or status", command)
	}
}
