package auth

import "context"

type Authorizer interface {
	Url() string
	Token(ctx context.Context, code string) (string, error)
}

func NewAuth(authorizer Authorizer) *auth {
	return &auth{authorizer: authorizer}
}

type auth struct {
	authorizer Authorizer
}

func (a *auth) Url() string {
	return a.authorizer.Url()
}

func (a *auth) Token(ctx context.Context, code string) (string, error) {
	return a.authorizer.Token(ctx, code)
}
