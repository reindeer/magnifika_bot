package google

import (
	"context"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

type Credentials interface {
	TokenSource(ctx context.Context, token *oauth2.Token) oauth2.TokenSource
}

type Consent interface {
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
}

func NewConfig(credentials string) (*oauth2.Config, error) {
	return google.ConfigFromJSON([]byte(credentials), sheets.SpreadsheetsScope)
}
