package config

import (
	"io/fs"
	"path/filepath"

	"github.com/reindeer/magnifika_bot/cmd/app/tools/googlelogin"
	migrateCommand "github.com/reindeer/magnifika_bot/cmd/app/tools/migrate"
	"github.com/reindeer/magnifika_bot/cmd/app/tools/serve"
	"github.com/reindeer/magnifika_bot/internal/adapter/customer"
	"github.com/reindeer/magnifika_bot/internal/adapter/google"
	"github.com/reindeer/magnifika_bot/internal/domain/bot"

	"gitlab.com/gorib/di"
	"gitlab.com/gorib/env"
	"gitlab.com/gorib/pry"
	"gitlab.com/gorib/pry/channels"
	"gitlab.com/gorib/storage/sql"
	"gitlab.com/gorib/waffle/app"
	"gitlab.com/gorib/waffle/tools/migrate"
)

func InitDi() {
	di.MustWire[pry.Logger](func() (pry.Logger, error) {
		environment := env.Value("ENVIRONMENT", "nocontour_noenv")
		sentry, err := channels.Sentry("error", channels.WithSentryConnection(env.Value("SENTRY_DSN", ""), environment))
		if err != nil {
			return nil, err
		}
		return pry.New(env.Value("LOGLEVEL", "info"), pry.ToChannels(sentry))
	})

	di.MustWire[sql.Db](sql.NewDb, di.Defaults(map[int]any{
		0: "sqlite3",
		1: filepath.Join(env.Value("DB_PATH", "."), "bot.sqlite"),
	}))

	di.MustDefine(customer.NewAdapter,
		di.Alias[bot.CustomerRepository](),
	)

	di.MustDefine(google.NewAdapter,
		di.Defaults(map[int]any{
			0: env.Value("GOOGLE_CREDENTIALS", ""),
			1: env.Value("GOOGLE_TOKEN", ""),
			2: env.Value("GOOGLE_APPLICATION_SHEET", ""),
			3: env.Value("GOOGLE_VALIDATION_SHEET", ""),
			4: env.Array("GATES", []string{}),
		}),
		di.Alias[bot.ApplicationService](),
		di.Alias[bot.PhoneRepository](),
		di.Alias[googlelogin.GoogleService](),
	)

	di.MustWire[serve.BotManagement](bot.NewBotManagement, di.Defaults(map[int]any{
		0: map[string]string{
			bot.GuardShortcut:      env.Value("PHONE_GUARD", ""),
			bot.DispatcherShortcut: env.Value("PHONE_DISPATCHER", ""),
			bot.EmergencyShortcut:  env.Value("PHONE_EMERGENCY", ""),
		},
		1: env.Value("TELEGRAM_TOKEN", ""),
	}))
	di.MustWire[app.BaseCommand](app.NewCommand, di.For[serve.Command](), di.Defaults(map[int]any{
		0: "bot:serve",
		1: "Start the bot",
	}))
	di.MustWire[serve.Command](serve.New, di.Tag(app.CommandTag))

	di.MustWire[app.BaseCommand](app.NewCommand, di.For[googlelogin.Command](), di.Defaults(map[int]any{
		0: "google:login",
		1: "Create google token to access to spreadsheets",
	}))
	di.MustWire[googlelogin.Command](googlelogin.New, di.Tag(app.CommandTag))

	di.MustWire[fs.FS](migrateCommand.NewMigrations, di.For[migrate.MigrateCommand]())
	migrate.InitMigrate()
}
