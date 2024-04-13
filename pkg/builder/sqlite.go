package builder

import (
	"gitlab.com/gorib/storage/sql"
)

func NewSqlite(query string, params *[]any) *sqlite {
	return &sqlite{
		Builder: sql.NewBuilder("-"),
		query:   query,
		params:  params,
	}
}

type sqlite struct {
	sql.Builder
	query  string
	params *[]any
}

func (m *sqlite) Build() (string, *[]any, error) {
	return m.query, m.params, nil
}

func (m *sqlite) Returning(fields ...string) sql.Builder {
	return m
}
