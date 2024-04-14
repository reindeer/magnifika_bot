package registry

import (
	"context"
	sqlDb "database/sql"
	"errors"
	"fmt"

	"gitlab.com/gorib/criteria"
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

	rec, err := sql.Get[record](ctx, sql.NewBuilder("registry").Where(criteria.And("code", "eq", code)).Sort("code"))
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

	_, err := sql.Exec(ctx, sql.NewBuilder("registry").Insert("code", "value").Values([][]any{{code, value}}).Conflict("code", "value"))
	return err
}

func (a *adapter) List(ctx context.Context) (map[string]string, error) {
	ctx, commit, rollback := a.Begin(ctx)
	defer rollback()
	defer commit()

	rows, err := sql.Select[[]*record](ctx, sql.NewBuilder("registry").Sort("code"))
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Code] = row.Value
	}
	return values, nil
}
