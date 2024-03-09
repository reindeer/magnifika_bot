package repository

import (
	"context"

	"github.com/reindeer/magnifika_bot/internal/bot/repository/builder"

	repository "gitlab.com/gorib/storage"
	"gitlab.com/gorib/storage/sql"
)

func NewCustomerRepository(connection sql.Db) *customerRepository {
	return &customerRepository{
		Repository: sql.NewRepository(connection),
	}
}

type customerRepository struct {
	repository.Repository
}

func (r *customerRepository) PhoneForCustomer(ctx context.Context, id int64) (string, error) {
	ctx, commit, rollback := r.Begin(ctx)
	defer rollback()
	customer, err := sql.Get(ctx, builder.NewSqlite(`select * from customers where customer_id=?`, &[]any{id}), new(struct {
		CustomerId int64  `db:"customer_id"`
		Phone      string `db:"phone"`
	}))
	if err != nil {
		return "", err
	}
	commit()
	return customer.Phone, nil
}

func (r *customerRepository) SaveCustomer(ctx context.Context, id int64, phone string) error {
	ctx, commit, rollback := r.Begin(ctx)
	defer rollback()
	_, err := sql.Exec(ctx, builder.NewSqlite(`insert into customers (customer_id, phone) values (?, ?) on conflict (customer_id) do update set phone=?`, &[]any{id, phone, phone}))
	if err != nil {
		return err
	}
	commit()
	return nil
}
