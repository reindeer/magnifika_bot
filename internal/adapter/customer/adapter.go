package customer

import (
	"context"

	"gitlab.com/gorib/criteria"
	repository "gitlab.com/gorib/storage"
	"gitlab.com/gorib/storage/sql"
)

func NewAdapter(connection sql.Db) *adapter {
	return &adapter{
		Repository: sql.NewRepository(connection),
	}
}

type adapter struct {
	repository.Repository
}

func (r *adapter) PhoneForCustomer(ctx context.Context, id int64) (string, error) {
	ctx, commit, rollback := r.Begin(ctx)
	defer rollback()
	customer, err := sql.Get(ctx, sql.NewBuilder("customers").Where(criteria.And("customer_id", "eq", id)).Sort("customer_id"), new(struct {
		CustomerId int64  `db:"customer_id"`
		Phone      string `db:"phone"`
	}))
	if err != nil {
		return "", err
	}
	commit()
	return customer.Phone, nil
}

func (r *adapter) SaveCustomer(ctx context.Context, id int64, phone string) error {
	ctx, commit, rollback := r.Begin(ctx)
	defer rollback()
	_, err := sql.Exec(ctx, sql.NewBuilder("customers").Insert("customer_id", "phone").Values([]any{id, phone}).Conflict("customer_id", "phone"))
	if err != nil {
		return err
	}
	commit()
	return nil
}
