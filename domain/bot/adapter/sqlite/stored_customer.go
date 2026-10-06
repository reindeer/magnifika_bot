//go:generate go tool store $GOFILE
package sqlite

import (
	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

type storedCustomer struct {
	_ *model.Customer `store:"sort:customer_id"`

	Id    int64  `db:"customer_id"`
	Phone string `db:"phone"`
}
