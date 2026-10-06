package di

import (
	dbSql "database/sql"
	"io"
	"path/filepath"

	"github.com/go-telegram/bot"
	"github.com/pressly/goose/v3"
	"github.com/reindeer/magnifika_bot/domain/bot/cmd/migrate"
	"golang.org/x/oauth2"

	"gitlab.com/gorib/di"
	"gitlab.com/gorib/env"
	"gitlab.com/gorib/pry"
	"gitlab.com/gorib/pry/channels"
	pryzerolog "gitlab.com/gorib/pry/zerolog"
	"gitlab.com/gorib/sql"

	"github.com/reindeer/magnifika_bot/config/app"
	"github.com/reindeer/magnifika_bot/domain/bot/adapter/google"
	"github.com/reindeer/magnifika_bot/domain/bot/adapter/sqlite"
	"github.com/reindeer/magnifika_bot/domain/bot/adapter/telegram"
	"github.com/reindeer/magnifika_bot/domain/bot/cmd/login"
	"github.com/reindeer/magnifika_bot/domain/bot/cmd/serve"
	authService "github.com/reindeer/magnifika_bot/domain/bot/service/auth"
	botService "github.com/reindeer/magnifika_bot/domain/bot/service/bot"
)

func Wiring() {
	di.DefaultCloser[io.Closer]()

	di.Wire[pry.Logger](func() (pry.Logger, error) {
		level, err := pry.ParseLevel(env.Value("LOGLEVEL", "info"))
		if err != nil {
			return nil, err
		}
		sentry, err := channels.Sentry(env.Value("SENTRY_LEVEL", "error"), channels.WithSentryConnection(
			env.Value("SENTRY_DSN", ""),
			env.Value("ENVIRONMENT", "noenv"),
			channels.WithSentryService(env.Value("SERVICE", "magnifika_bot")),
		))
		if err != nil {
			return nil, err
		}
		return pry.New(level, pryzerolog.New(pryzerolog.Plain()), pry.ToSinks(sentry)), nil
	})

	di.Wire[sql.Db](sql.NewDb, di.Args(map[string]any{
		"driver": "sqlite",
		"dsn":    filepath.Join(env.Value("DB_PATH", "."), "bot.sqlite"),
	}))

	di.Wire[botService.Customers](sqlite.NewCustomers)

	di.Define(google.NewConfig,
		di.Args(map[string]any{"credentials": env.NeedValue[string]("GOOGLE_CREDENTIALS")}),
		di.Alias[google.Credentials](),
		di.Alias[google.Consent](),
	)
	di.Define(google.NewSheets,
		di.Args(map[string]any{
			"token":              env.NeedJson[*oauth2.Token]("GOOGLE_TOKEN"),
			"applicationSheetId": env.NeedValue[string]("GOOGLE_APPLICATION_SHEET"),
			"validationSheetId":  env.NeedValue[string]("GOOGLE_VALIDATION_SHEET"),
			"gates":              env.NeedArray[string]("GATES"),
		}),
		di.Alias[botService.Phones](),
		di.Alias[botService.Applications](),
	)
	di.Wire[authService.Authorizer](google.NewAuthorization)

	di.Define(bot.New,
		di.Args(map[string]any{
			"token":   env.NeedValue[string]("TELEGRAM_TOKEN"),
			"options": []bot.Option{bot.WithSkipGetMe(), bot.WithNotAsyncHandlers()},
		}),
		di.Alias[telegram.Sender](),
		di.Alias[serve.Updates](),
		// Its Close is the Bot API method that logs the bot out of the server, not a resource release.
		di.Close[di.NoCloser](),
	)
	di.Wire[botService.Messenger](telegram.NewMessenger)

	di.Wire[serve.Bot](botService.NewBot, di.Args(map[string]any{
		"guardPhone":      env.NeedValue[string]("PHONE_GUARD"),
		"dispatcherPhone": env.NeedValue[string]("PHONE_DISPATCHER"),
		"emergencyPhone":  env.NeedValue[string]("PHONE_EMERGENCY"),
	}))
	di.Wire[login.Auth](authService.NewAuth)

	di.Define(func(db sql.Db) (*dbSql.DB, error) { return db.DbSql() }, di.Close[di.NoCloser]())
	di.Define(migrate.Migrations)
	di.Define(goose.NewProvider, di.Args(map[string]any{
		"dialect": goose.DialectSQLite3,
		"opts":    []goose.ProviderOption{goose.WithTableName("migrations")},
	}))

	di.Define(app.New)
	di.Entry[*app.App]()
}
