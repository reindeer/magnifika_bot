package customer

func newStoredCustomer(customerId int64, phone string) *storedCustomer {
	return &storedCustomer{
		CustomerId: customerId,
		Phone:      phone,
	}
}

type storedCustomer struct {
	CustomerId int64  `db:"customer_id"`
	Phone      string `db:"phone"`
}

func (m *storedCustomer) Unwrap() (int64, string) {
	return m.CustomerId, m.Phone
}

func (m *storedCustomer) Inserts() ([]string, []any) {
	return []string{
			"customer_id",
			"phone",
		}, []any{
			m.CustomerId,
			m.Phone,
		}
}
