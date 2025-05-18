package google

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

func NewAdapter(
	credentials, token string,
	applicationSheetId, validationSheetId string,
	validationSheetNames []string,
	logger pry.Logger,
) (*adapter, error) {
	var config *oauth2.Config
	if credentials != "" {
		c, err := google.ConfigFromJSON([]byte(credentials), sheets.SpreadsheetsScope)
		if err != nil {
			return nil, err
		}
		config = c
	}
	var t *oauth2.Token
	if token != "" {
		if err := json.Unmarshal([]byte(token), &t); err != nil {
			return nil, fmt.Errorf("unable to parse application token: %w", err)
		}
	}

	return &adapter{
		logger:               logger,
		token:                t,
		applicationSheetId:   applicationSheetId,
		validationSheetId:    validationSheetId,
		validationSheetNames: validationSheetNames,
		config:               config,
	}, nil
}

type User struct {
	A string `sheets:"А"`
}

type adapter struct {
	logger               pry.Logger
	config               *oauth2.Config
	token                *oauth2.Token
	applicationSheetId   string
	validationSheetId    string
	validationSheetNames []string
}

func (a *adapter) Login(ctx context.Context) error {
	if a.config == nil {
		return fmt.Errorf("no google credentials provided")
	}
	var token *oauth2.Token
	authURL := a.config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	scan := make(chan struct {
		Code  string
		Error error
	})
	go func() {
		var code string
		_, err := fmt.Scan(&code)
		scan <- struct {
			Code  string
			Error error
		}{Code: code, Error: err}
	}()

	var authCode string
	select {
	case <-ctx.Done():
		return nil
	case s := <-scan:
		if s.Error != nil {
			return fmt.Errorf("unable to read authorization code: %w", s.Error)
		}
		authCode = s.Code
	}

	token, err := a.config.Exchange(ctx, authCode)
	if err != nil {
		return err
	}
	tb, err := json.Marshal(token)
	if err != nil {
		return err
	}

	fmt.Printf("\n\nAdd to your env file:\nGOOGLE_TOKEN=%s\n", tb)

	return nil
}

func (a *adapter) ValidatePhone(ctx context.Context, phone string) ([]string, error) {
	if a.validationSheetId == "" {
		return nil, fmt.Errorf("no phone validation sheet id")
	}
	service, err := a.service(ctx)
	if err != nil {
		return nil, err
	}

	phone = strings.TrimLeft(phone, "+")
	var gates []string
	for _, gate := range a.validationSheetNames {
		values, err := service.Spreadsheets.Values.Get(a.validationSheetId, gate+"!A:A").Do()
		if err != nil {
			return nil, err
		}
		for _, row := range values.Values {
			if len(row) == 0 {
				continue
			}
			if row[0] == phone {
				gates = append(gates, gate)
				break
			}
		}
	}

	return gates, nil
}

func (a *adapter) Application(ctx context.Context, phone, plate string, gates []string) error {
	if a.applicationSheetId == "" {
		return fmt.Errorf("no application sheet id")
	}
	service, err := a.service(ctx)
	if err != nil {
		return err
	}

	date := time.Now().Format("02.01")

	writeRange := date + "!A:E"
	values := &sheets.ValueRange{
		Values: [][]any{
			{strings.Join(gates, "\n"), "", "", plate, phone},
		},
	}

	err = a.send(ctx, service, writeRange, values)
	if err != nil && strings.Contains(err.Error(), "googleapi: Error 400: Unable to parse range:") {
		if err = a.createSheet(ctx, service, date); err == nil {
			err = a.send(ctx, service, writeRange, values)
		}
	}
	return err
}

func (a *adapter) createSheet(ctx context.Context, service *sheets.Service, name string) error {
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

	_, err := service.Spreadsheets.BatchUpdate(a.applicationSheetId, rbb).Context(ctx).Do()
	return err
}

func (a *adapter) send(ctx context.Context, service *sheets.Service, range_ string, values *sheets.ValueRange) error {
	_, err := service.Spreadsheets.Values.Append(a.applicationSheetId, range_, values).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	return err
}

func (a *adapter) service(ctx context.Context) (*sheets.Service, error) {
	if a.config == nil {
		return nil, fmt.Errorf("no google credentials provided")
	}
	if a.token == nil {
		return nil, fmt.Errorf("empty token, please run google:login command")
	}
	return sheets.NewService(ctx, option.WithHTTPClient(a.config.Client(ctx, a.token)))
}
