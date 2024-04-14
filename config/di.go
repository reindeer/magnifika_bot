package config

import (
	"io/fs"

	"github.com/reindeer/magnifika_bot/internal/adapter/application"
	"github.com/reindeer/magnifika_bot/internal/adapter/phone"
	"github.com/reindeer/magnifika_bot/internal/adapter/registry"
	"github.com/reindeer/magnifika_bot/internal/bot/cmd/configure"
	migrateCommand "github.com/reindeer/magnifika_bot/internal/bot/cmd/migrate"
	"github.com/reindeer/magnifika_bot/internal/bot/cmd/serve"
	"github.com/reindeer/magnifika_bot/internal/bot/management"
	"github.com/reindeer/magnifika_bot/internal/bot/repository"
	"github.com/reindeer/magnifika_bot/pkg/authenticator"

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

	di.Define(authenticator.New,
		di.Alias[phone.Authenticator](),
		di.Alias[application.Authenticator](),
	)

	di.Define(registry.NewAdapter,
		di.Alias[configure.Registry](),
		di.Alias[phone.Registry](),
		di.Alias[application.Registry](),
		di.Alias[management.Registry](),
	)

	di.Define(phone.NewAdapter,
		di.Alias[management.PhoneAdapter](),
	)

	di.Define(application.NewAdapter,
		di.Alias[management.ApplicationAdapter](),
	)

	di.Wire[management.CustomerRepository](repository.NewCustomerRepository)
	di.Wire[serve.BotManagement](management.NewBotManagement)
	di.Wire[app.BaseCommand](func() app.BaseCommand { return app.NewCommand("bot:serve", "Start the bot") }, di.For[serve.Command]())
	di.Wire[serve.Command](serve.New, di.Tag(app.CommandTag))

	di.Wire[app.BaseCommand](func() app.BaseCommand { return app.NewCommand("configure", "Configure bot parameters") }, di.For[configure.Command]())
	di.Wire[configure.Command](configure.New, di.Tag(app.CommandTag))

	di.Wire[fs.FS](migrateCommand.NewMigrations, di.For[migrate.MigrateCommand]())
	migrate.InitMigrate()
}
