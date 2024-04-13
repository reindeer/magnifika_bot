package registry

import (
	"context"
	sqlDb "database/sql"
	"errors"
	"fmt"

	"github.com/reindeer/magnifika_bot/pkg/builder"

	repository "gitlab.com/gorib/storage"
	"gitlab.com/gorib/storage/sql"
)

type record struct {
	Code  string `db:"code"`
	Value string `db:"value"`
}

func NewAdapter(connection sql.Db) *adapter {
	return &adapter{Repository: sql.NewRepository(connection)}
}

type adapter struct {
	repository.Repository
}

func (a *adapter) Get(ctx context.Context, code string) (string, error) {
	ctx, commit, rollback := a.Begin(ctx)
	defer rollback()
	defer commit()

	rec, err := sql.Get[record](ctx, builder.NewSqlite("select value from registry where code=?", &[]any{code}))
	if err != nil {
		if errors.Is(err, sqlDb.ErrNoRows) {
			return "", fmt.Errorf("no key found: %s", code)
		}
		return "", err
	}
	return rec.Value, nil
}

func (a *adapter) Save(ctx context.Context, code, value string) error {
	ctx, commit, rollback := a.Begin(ctx)
	defer rollback()
	defer commit()

	_, err := sql.Exec(ctx, builder.NewSqlite("insert into registry (code,value) values (?, ?) on conflict (code) do update set value=excluded.value", &[]any{code, value}))
	return err
}

func (a *adapter) List(ctx context.Context) (map[string]string, error) {
	ctx, commit, rollback := a.Begin(ctx)
	defer rollback()
	defer commit()

	rows, err := sql.Select[[]*record](ctx, builder.NewSqlite("select * from registry", &[]any{}))
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Code] = row.Value
	}
	return values, nil
}
