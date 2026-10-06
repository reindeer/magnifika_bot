package sqlite

import (
	"context"
	"errors"

	"gitlab.com/gorib/criteria"
	"gitlab.com/gorib/sql"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

const customersTable = "customers"

func NewCustomers(db sql.Db) (*customers, error) {
	repo, err := sql.NewRepository[storedCustomer](db)
	if err != nil {
		return nil, err
	}
	return &customers{repo: repo}, nil
}

type customers struct {
	repo sql.Repository[storedCustomer]
}

func (c *customers) Customer(ctx context.Context, id int64) (*model.Customer, error) {
	stored, err := c.repo.Get(ctx, sql.Select().From(sql.Table(customersTable)).Where(criteria.And("customer_id", "eq", id)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrUnknownCustomer
	}
	if err != nil {
		return nil, err
	}
	return stored.Unwrap(), nil
}

func (c *customers) SaveCustomer(ctx context.Context, customer *model.Customer) error {
	fields, values := newStoredCustomer(customer).Inserts()
	_, err := c.repo.Get(ctx, sql.Insert(fields...).Into(sql.Table(customersTable)).Values(values...).Conflict("customer_id", "phone"))
	return err
}
