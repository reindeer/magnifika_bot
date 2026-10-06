package google

import (
	"context"
	"encoding/json"

	"golang.org/x/oauth2"
)

func NewAuthorization(consent Consent) *authorization {
	return &authorization{consent: consent}
}

type authorization struct {
	consent Consent
}

func (a *authorization) Url() string {
	return a.consent.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func (a *authorization) Token(ctx context.Context, code string) (string, error) {
	token, err := a.consent.Exchange(ctx, code)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
