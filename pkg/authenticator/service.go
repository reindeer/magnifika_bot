package authenticator

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

func New() *client {
	return &client{}
}

type client struct {
}

func (c *client) GetToken(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	return config.Exchange(ctx, authCode)
}
