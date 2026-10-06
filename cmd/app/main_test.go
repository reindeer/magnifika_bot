package main

import (
	dbSql "database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/gorib/env"

	"github.com/reindeer/magnifika_bot/config/diogen"
	"github.com/reindeer/magnifika_bot/config/diogen/login"
	"github.com/reindeer/magnifika_bot/config/diogen/migrate"
	"github.com/reindeer/magnifika_bot/config/diogen/serve"
)

const credentials = `{"installed":{"client_id":"id","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["urn:ietf:wg:oauth:2.0:oob"]}}`

func serveEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DB_PATH", t.TempDir())
	t.Setenv("TELEGRAM_TOKEN", "token")
	t.Setenv("PHONE_GUARD", "1")
	t.Setenv("PHONE_DISPATCHER", "2")
	t.Setenv("PHONE_EMERGENCY", "3")
	t.Setenv("GOOGLE_CREDENTIALS", credentials)
	t.Setenv("GOOGLE_TOKEN", `{"access_token":"access","token_type":"Bearer"}`)
	t.Setenv("GOOGLE_APPLICATION_SHEET", "applications")
	t.Setenv("GOOGLE_VALIDATION_SHEET", "validation")
	t.Setenv("GATES", "north, south")
}

func TestServeContainer(t *testing.T) {
	serveEnv(t)

	container, err := serve.InitApp()

	if err != nil {
		t.Fatal(err)
	}
	if err := container.Shutdown()(); err != nil {
		t.Error(err)
	}
}

func TestServeClosesDatabase(t *testing.T) {
	serveEnv(t)
	container, err := serve.InitApp()
	if err != nil {
		t.Fatal(err)
	}
	handle, err := container.SqlDb().DbSql()
	if err != nil {
		t.Fatal(err)
	}

	if err := container.Shutdown()(); err != nil {
		t.Fatal(err)
	}

	if err := handle.PingContext(t.Context()); err == nil {
		t.Error("database is still open after shutdown")
	}
}

func TestMigrateClosesDatabase(t *testing.T) {
	t.Setenv("DB_PATH", t.TempDir())
	container, err := migrate.InitApp()
	if err != nil {
		t.Fatal(err)
	}
	handle, err := container.SqlDb().DbSql()
	if err != nil {
		t.Fatal(err)
	}

	if err := container.Shutdown()(); err != nil {
		t.Fatal(err)
	}

	if err := handle.PingContext(t.Context()); err == nil {
		t.Error("database is still open after shutdown")
	}
}

func TestServeNeedsTelegramToken(t *testing.T) {
	serveEnv(t)
	unsetenv(t, "TELEGRAM_TOKEN")

	_, err := serve.InitApp()

	if !errors.Is(err, env.ErrNoEnvFound) {
		t.Fatalf("got %v, want %v", err, env.ErrNoEnvFound)
	}
	if got := describe(err); strings.Contains(got, "\n") || !strings.Contains(got, "TELEGRAM_TOKEN") {
		t.Errorf("described as %q", got)
	}
}

func TestLoginNeedsOnlyCredentials(t *testing.T) {
	t.Setenv("GOOGLE_CREDENTIALS", credentials)
	for _, name := range []string{"TELEGRAM_TOKEN", "GOOGLE_TOKEN", "GATES"} {
		unsetenv(t, name)
	}

	container, err := login.InitApp()

	if err != nil {
		t.Fatal(err)
	}
	if err := container.Shutdown()(); err != nil {
		t.Error(err)
	}
}

func TestMigrateNeedsNoSecrets(t *testing.T) {
	t.Setenv("DB_PATH", t.TempDir())
	for _, name := range []string{"TELEGRAM_TOKEN", "GOOGLE_CREDENTIALS"} {
		unsetenv(t, name)
	}

	container, err := migrate.InitApp()

	if err != nil {
		t.Fatal(err)
	}
	if err := container.Shutdown()(); err != nil {
		t.Error(err)
	}
}

func TestMigrateContinuesProductionHistory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DB_PATH", dir)
	db, err := dbSql.Open("sqlite", filepath.Join(dir, "bot.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`create table migrations (id integer primary key autoincrement, version_id integer not null, is_applied integer not null, tstamp timestamp default (datetime('now')))`,
		`insert into migrations (version_id, is_applied) values (0, 1), (20240308201106, 1), (20240413204123, 1)`,
		`create table customers (customer_id int not null unique, phone varchar(255) not null)`,
		`create table registry (code varchar(50) not null unique, value text null)`,
		`insert into customers values (1, '+70000000000')`,
	} {
		if _, err := db.ExecContext(t.Context(), statement); err != nil {
			t.Fatal(err)
		}
	}

	if failed, err := runCommand(t, "migrate", "up"); failed || err != nil {
		t.Fatalf("migrate failed: %v, err: %v", failed, err)
	}

	var registries, customers int
	if err := db.QueryRowContext(t.Context(), `select count(*) from sqlite_master where name = 'registry'`).Scan(&registries); err != nil {
		t.Fatal(err)
	}
	if registries != 0 {
		t.Error("registry table survived the migration")
	}
	if err := db.QueryRowContext(t.Context(), `select count(*) from customers`).Scan(&customers); err != nil {
		t.Fatal(err)
	}
	if customers != 1 {
		t.Errorf("customers = %d, want the one stored before", customers)
	}
}

func runCommand(t *testing.T, name string, args ...string) (failed bool, err error) {
	t.Helper()
	cmd, ok := diogen.Get(name)
	if !ok {
		t.Fatalf("no command %q", name)
	}
	return cmd.Run(t.Context(), args)
}

func unsetenv(t *testing.T, name string) {
	t.Helper()
	t.Setenv(name, "")
	if err := os.Unsetenv(name); err != nil {
		t.Fatal(err)
	}
}
