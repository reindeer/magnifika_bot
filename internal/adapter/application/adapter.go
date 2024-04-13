package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"gitlab.com/gorib/pry"
)

type Registry interface {
	Get(ctx context.Context, code string) (string, error)
	Save(ctx context.Context, code, value string) error
}

type Authenticator interface {
	GetToken(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error)
}

func NewAdapter(registry Registry, authenticator Authenticator, logger pry.Logger) (*adapter, error) {
	return &adapter{
		logger:        logger,
		registry:      registry,
		authenticator: authenticator,
	}, nil
}

type adapter struct {
	logger        pry.Logger
	registry      Registry
	authenticator Authenticator
	service       *sheets.Service
	sheetId       string
}

func (a *adapter) Init(ctx context.Context) error {
	creds, err := a.registry.Get(ctx, "application.credentials")
	if err != nil {
		return err
	}

	config, err := google.ConfigFromJSON([]byte(creds), sheets.SpreadsheetsScope)
	if err != nil {
		return err
	}

	var token *oauth2.Token
	t, err := a.registry.Get(ctx, "application.token")
	if err == nil {
		err = json.Unmarshal([]byte(t), &token)
		if err != nil {
			return fmt.Errorf("unable to parse application token: %w", err)
		}
	} else {
		token, err = a.authenticator.GetToken(ctx, config)
		if err != nil {
			return err
		}
		tb, err := json.Marshal(token)
		if err != nil {
			return err
		}
		err = a.registry.Save(ctx, "application.token", string(tb))
		if err != nil {
			return err
		}
	}

	a.service, err = sheets.NewService(ctx, option.WithHTTPClient(config.Client(ctx, token)))
	if err != nil {
		return err
	}

	a.sheetId, err = a.registry.Get(ctx, "application.sheet")
	return err
}

func (a *adapter) Application(ctx context.Context, phone, plate string, gates []string) error {
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{
			{phone, plate, strings.Join(gates, "\n")},
		},
	}

	writeRange := fmt.Sprintf("%s!D:D", time.Now().Format("02.01"))
	_, err := a.service.Spreadsheets.Values.Append(a.sheetId, writeRange, valueRange).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	return err
}
