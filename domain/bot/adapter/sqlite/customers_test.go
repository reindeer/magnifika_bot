package sqlite

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"

	"gitlab.com/gorib/sql"

	"github.com/reindeer/magnifika_bot/domain/bot/cmd/migrate"
	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

func newTestCustomers(t *testing.T) *customers {
	t.Helper()
	db, err := sql.NewDb("sqlite", filepath.Join(t.TempDir(), "bot.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	handle, err := db.DbSql()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = handle.Close() })
	migrations, err := migrate.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, handle, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(t.Context()); err != nil {
		t.Fatal(err)
	}
	c, err := NewCustomers(db)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestUnknownCustomer(t *testing.T) {
	c := newTestCustomers(t)

	_, err := c.Customer(t.Context(), 1)

	if !errors.Is(err, model.ErrUnknownCustomer) {
		t.Fatalf("got %v, want %v", err, model.ErrUnknownCustomer)
	}
}

func TestSaveCustomerReplacesPhone(t *testing.T) {
	c := newTestCustomers(t)

	for _, phone := range []string{"+70000000000", "+71111111111"} {
		if err := c.SaveCustomer(t.Context(), &model.Customer{Id: 1, Phone: phone}); err != nil {
			t.Fatal(err)
		}
	}
	customer, err := c.Customer(t.Context(), 1)

	if err != nil {
		t.Fatal(err)
	}
	if customer.Phone != "+71111111111" {
		t.Errorf("phone = %q, want the one saved last", customer.Phone)
	}
}
