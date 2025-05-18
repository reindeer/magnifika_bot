package customer

import (
	"context"

	"gitlab.com/gorib/criteria"
	repository "gitlab.com/gorib/storage"
	"gitlab.com/gorib/storage/sql"
)

func NewAdapter(connection sql.Db) *adapter {
	return &adapter{
		repo: sql.NewRepository[storedCustomer](connection),
	}
}

type adapter struct {
	repo repository.SqlRepository[storedCustomer]
}

func (r *adapter) PhoneForCustomer(ctx context.Context, id int64) (string, error) {
	customer, err := r.repo.Get(ctx, sql.NewBuilder("customers").Where(criteria.And("customer_id", "eq", id)).Sort("customer_id"))
	if err != nil {
		return "", err
	}
	return customer.Phone, nil
}

func (r *adapter) SaveCustomer(ctx context.Context, id int64, phone string) error {
	fields, values := newStoredCustomer(id, phone).Inserts()
	_, err := sql.Exec(ctx, sql.NewBuilder("customers").Insert(fields...).Values(values...).Conflict(fields[0], fields[1]))
	if err != nil {
		return err
	}
	return nil
}
