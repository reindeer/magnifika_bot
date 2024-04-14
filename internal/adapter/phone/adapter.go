package phone

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tobi696/googlesheetsparser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"gitlab.com/gorib/pry"
)

type User struct {
	A string
}

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
	sheetNames    []string
}

func (a *adapter) Init(ctx context.Context) error {
	creds, err := a.registry.Get(ctx, "phone_validation.credentials")
	if err != nil {
		return err
	}

	config, err := google.ConfigFromJSON([]byte(creds), sheets.SpreadsheetsReadonlyScope)
	if err != nil {
		return err
	}

	var token *oauth2.Token
	t, err := a.registry.Get(ctx, "phone_validation.token")
	if err == nil {
		err = json.Unmarshal([]byte(t), &token)
		if err != nil {
			return fmt.Errorf("unable to parse phone validation token: %w", err)
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
		err = a.registry.Save(ctx, "phone_validation.token", string(tb))
		if err != nil {
			return err
		}
	}

	a.service, err = sheets.NewService(ctx, option.WithHTTPClient(config.Client(ctx, token)))
	if err != nil {
		return err
	}

	a.sheetId, err = a.registry.Get(ctx, "phone_validation.sheet")
	if err != nil {
		return err
	}

	value, err := a.registry.Get(ctx, "phone_validation.sheet_names")
	if err != nil {
		return err
	}
	var names []string
	err = json.Unmarshal([]byte(value), &names)
	if err != nil {
		return err
	}
	a.sheetNames = names

	return nil
}
func (a *adapter) ValidatePhone(ctx context.Context, phone string) ([]string, error) {
	phone = strings.TrimLeft(phone, "+")
	var gates []string
	for _, gate := range a.sheetNames {
		users, err := googlesheetsparser.ParseSheetIntoStructSlice[User](googlesheetsparser.Options{
			Service:       a.service,
			SpreadsheetID: a.sheetId,
			SheetName:     gate,
		}.Build())
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			if user.A == phone {
				gates = append(gates, gate)
				break
			}
		}
	}

	return gates, nil
}
