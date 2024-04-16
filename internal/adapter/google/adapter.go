package google

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Tobi696/googlesheetsparser"
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

func NewAdapter(registry Registry, logger pry.Logger) (*adapter, error) {
	return &adapter{
		logger:   logger,
		registry: registry,
	}, nil
}

type User struct {
	A string `sheets:"А"`
}

type adapter struct {
	logger               pry.Logger
	registry             Registry
	service              *sheets.Service
	applicationSheetId   string
	validationSheetId    string
	validationSheetNames []string
}

func (a *adapter) Init(ctx context.Context) error {
	creds, err := a.registry.Get(ctx, "google.credentials")
	if err != nil {
		return err
	}

	config, err := google.ConfigFromJSON([]byte(creds), sheets.SpreadsheetsScope)
	if err != nil {
		return err
	}

	var token *oauth2.Token
	t, err := a.registry.Get(ctx, "google.token")
	if err == nil {
		err = json.Unmarshal([]byte(t), &token)
		if err != nil {
			return fmt.Errorf("unable to parse application token: %w", err)
		}
	} else {
		authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

		var authCode string
		if _, err := fmt.Scan(&authCode); err != nil {
			return fmt.Errorf("unable to read authorization code: %w", err)
		}

		token, err = config.Exchange(ctx, authCode)
		if err != nil {
			return err
		}
		tb, err := json.Marshal(token)
		if err != nil {
			return err
		}
		err = a.registry.Save(ctx, "google.token", string(tb))
		if err != nil {
			return err
		}
	}

	a.service, err = sheets.NewService(ctx, option.WithHTTPClient(config.Client(ctx, token)))
	if err != nil {
		return err
	}

	a.applicationSheetId, err = a.registry.Get(ctx, "application.sheet")
	if err != nil {
		return err
	}

	a.validationSheetId, err = a.registry.Get(ctx, "phone_validation.sheet")
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
	a.validationSheetNames = names
	return nil
}

func (a *adapter) ValidatePhone(ctx context.Context, phone string) ([]string, error) {
	phone = strings.TrimLeft(phone, "+")
	var gates []string
	for _, gate := range a.validationSheetNames {
		users, err := googlesheetsparser.ParseSheetIntoStructSlice[User](googlesheetsparser.Options{
			Service:       a.service,
			SpreadsheetID: a.validationSheetId,
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

func (a *adapter) Application(ctx context.Context, phone, plate string, gates []string) error {
	date := time.Now().Format("02.01")

	writeRange := fmt.Sprintf("%s!A:E", date)
	values := &sheets.ValueRange{
		Values: [][]any{
			{strings.Join(gates, "\n"), "", "", plate, phone},
		},
	}

	err := a.send(ctx, writeRange, values)
	if err != nil && strings.Contains(err.Error(), "googleapi: Error 400: Unable to parse range:") {
		if err = a.createSheet(ctx, date); err == nil {
			err = a.send(ctx, writeRange, values)
		}
	}
	return err
}

func (a *adapter) createSheet(ctx context.Context, name string) error {
	req := sheets.Request{
		AddSheet: &sheets.AddSheetRequest{
			Properties: &sheets.SheetProperties{
				Title: name,
			},
		},
	}

	rbb := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{&req},
	}

	_, err := a.service.Spreadsheets.BatchUpdate(a.applicationSheetId, rbb).Context(ctx).Do()
	return err
}

func (a *adapter) send(ctx context.Context, range_ string, values *sheets.ValueRange) error {
	_, err := a.service.Spreadsheets.Values.Append(a.applicationSheetId, range_, values).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	return err
}
