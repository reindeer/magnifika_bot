package google

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func NewSheets(
	credentials Credentials,
	token *oauth2.Token,
	applicationSheetId, validationSheetId string,
	gates []string,
) (*spreadsheets, error) {
	if applicationSheetId == "" || validationSheetId == "" {
		return nil, errors.New("application and validation sheet ids are required")
	}
	for _, gate := range gates {
		if gate == "" {
			return nil, errors.New("gate names must not be empty")
		}
	}
	// Refreshes outlive the request that triggers them, so they get a context of their own with a deadline.
	refresh := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Timeout: 30 * time.Second})
	return &spreadsheets{
		tokens:             credentials.TokenSource(refresh, token),
		applicationSheetId: applicationSheetId,
		validationSheetId:  validationSheetId,
		gates:              gates,
	}, nil
}

type spreadsheets struct {
	tokens             oauth2.TokenSource
	applicationSheetId string
	validationSheetId  string
	gates              []string

	// Two first applications of a day would both create the day's sheet, and the second would fail.
	writes sync.Mutex
}

func (s *spreadsheets) Gates(ctx context.Context, phone string) ([]string, error) {
	service, err := s.service(ctx)
	if err != nil {
		return nil, err
	}

	phone = strings.TrimLeft(phone, "+")
	var gates []string
	for _, gate := range s.gates {
		values, err := service.Spreadsheets.Values.Get(s.validationSheetId, gate+"!A:A").Context(ctx).Do()
		if err != nil {
			return nil, err
		}
		for _, row := range values.Values {
			if len(row) > 0 && row[0] == phone {
				gates = append(gates, gate)
				break
			}
		}
	}
	return gates, nil
}

func (s *spreadsheets) Apply(ctx context.Context, phone, plate string, gates []string) error {
	service, err := s.service(ctx)
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

	s.writes.Lock()
	defer s.writes.Unlock()
	err = s.append(ctx, service, writeRange, values)
	if err != nil && strings.Contains(err.Error(), "googleapi: Error 400: Unable to parse range:") {
		if err = s.createSheet(ctx, service, date); err == nil {
			err = s.append(ctx, service, writeRange, values)
		}
	}
	return err
}

func (s *spreadsheets) createSheet(ctx context.Context, service *sheets.Service, name string) error {
	request := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{Title: name},
			},
		}},
	}
	_, err := service.Spreadsheets.BatchUpdate(s.applicationSheetId, request).Context(ctx).Do()
	return err
}

func (s *spreadsheets) append(ctx context.Context, service *sheets.Service, writeRange string, values *sheets.ValueRange) error {
	_, err := service.Spreadsheets.Values.Append(s.applicationSheetId, writeRange, values).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	return err
}

func (s *spreadsheets) service(ctx context.Context) (*sheets.Service, error) {
	return sheets.NewService(ctx, option.WithTokenSource(s.tokens))
}
