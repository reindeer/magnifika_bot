package config

import (
	"io/fs"

	_ "github.com/mattn/go-sqlite3"
	migrateCommand "github.com/reindeer/magnifika_bot/internal/bot/cmd/migrate"
	"github.com/reindeer/magnifika_bot/internal/bot/cmd/serve"
	"github.com/reindeer/magnifika_bot/internal/bot/management"
	"github.com/reindeer/magnifika_bot/internal/bot/repository"

	"gitlab.com/gorib/di"
	"gitlab.com/gorib/env"
	"gitlab.com/gorib/pry"
	"gitlab.com/gorib/pry/channels"
	"gitlab.com/gorib/storage/sql"
	"gitlab.com/gorib/waffle/app"
	"gitlab.com/gorib/waffle/tools/migrate"
)

func InitDi() {
	di.Wire[pry.Logger](func() pry.Logger {
		sentry, err := channels.Sentry(env.Value("SENTRY_DSN", ""), env.Value("ENVIRONMENT", "nocontour_noenv"))
		if err != nil {
			panic(err)
		}
		logger, err := pry.New(
			env.Value("LOGLEVEL", "info"),
			pry.ToChannels(sentry),
			pry.WithCaller(),
		)
		if err != nil {
			panic(err)
		}
		return logger
	})

	di.Wire[sql.Db](sql.NewDb, di.Defaults(map[int]any{
		0: "sqlite3",
		1: "bot.sqlite",
	}))
	di.Wire[management.OperatingManagement](management.NewTst)
	di.Wire[management.CustomerRepository](repository.NewCustomerRepository)
	di.Wire[serve.BotManagement](management.NewBotManagement, di.Defaults(map[int]any{
		0: map[string]string{
			management.EmergenceShortcut:  env.NeedValue[string]("EMERGENCY_CONTACT"),
			management.DispatcherShortcut: env.NeedValue[string]("DISPATCHER_CONTACT"),
			management.GuardShortcut:      env.NeedValue[string]("GUARD_CONTACT"),
		},
	}))
	di.Wire[app.BaseCommand](func() app.BaseCommand { return app.NewCommand("bot:serve", "Start the bot") }, di.For[serve.Command]())
	di.Wire[serve.Command](serve.New, di.Tag(app.CommandTag), di.Defaults(map[int]any{
		0: env.Value("BOT_TOKEN", ""),
	}))

	di.Wire[fs.FS](migrateCommand.NewMigrations, di.For[migrate.MigrateCommand]())
	migrate.InitMigrate()
}
