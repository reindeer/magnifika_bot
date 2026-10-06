package bot

import (
	"context"
	"errors"
	"slices"
	"testing"

	"gitlab.com/gorib/pry"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

type fakeCustomers map[int64]string

func (f fakeCustomers) Customer(_ context.Context, id int64) (*model.Customer, error) {
	phone, ok := f[id]
	if !ok {
		return nil, model.ErrUnknownCustomer
	}
	return &model.Customer{Id: id, Phone: phone}, nil
}

func (f fakeCustomers) SaveCustomer(_ context.Context, customer *model.Customer) error {
	f[customer.Id] = customer.Phone
	return nil
}

type fakePhones struct {
	gates map[string][]string
	err   error
}

func (f fakePhones) Gates(_ context.Context, phone string) ([]string, error) {
	return f.gates[phone], f.err
}

type application struct {
	phone, plate string
	gates        []string
}

type fakeApplications struct {
	sent []application
}

func (f *fakeApplications) Apply(_ context.Context, phone, plate string, gates []string) error {
	f.sent = append(f.sent, application{phone: phone, plate: plate, gates: gates})
	return nil
}

var errSheets = errors.New("sheets are down")

func newTestBot(t *testing.T, customers Customers, phones Phones, applications Applications, messenger Messenger) *bot {
	t.Helper()
	b, err := NewBot("+71", "+72", "+73", customers, phones, applications, messenger, pry.Noop())
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestEmptyContactPhoneIsRefused(t *testing.T) {
	_, err := NewBot("+71", "", "+73", fakeCustomers{}, fakePhones{}, &fakeApplications{}, &fakeMessenger{}, pry.Noop())

	if err == nil {
		t.Fatal("a bot without the dispatcher phone was built")
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name   string
		phones fakePhones
		want   error
		stored bool
	}{
		{name: "registered phone", phones: fakePhones{gates: map[string][]string{"+70000000000": {"north"}}}, stored: true},
		{name: "unknown phone", phones: fakePhones{}, want: model.ErrUnknownPhone},
		{name: "sheets failure", phones: fakePhones{err: errSheets}, want: errSheets},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customers := fakeCustomers{}
			b := newTestBot(t, customers, tt.phones, &fakeApplications{}, &fakeMessenger{})

			err := b.register(t.Context(), 1, "+70000000000")

			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if _, stored := customers[1]; stored != tt.stored {
				t.Errorf("stored = %v, want %v", stored, tt.stored)
			}
		})
	}
}

func TestApply(t *testing.T) {
	tests := []struct {
		name      string
		customers fakeCustomers
		phones    fakePhones
		want      error
	}{
		{
			name:      "known customer",
			customers: fakeCustomers{1: "+70000000000"},
			phones:    fakePhones{gates: map[string][]string{"+70000000000": {"north", "south"}}},
		},
		{name: "unknown customer", customers: fakeCustomers{}, want: model.ErrUnknownCustomer},
		{name: "phone dropped from the list", customers: fakeCustomers{1: "+70000000000"}, phones: fakePhones{}, want: model.ErrPhoneChanged},
		{name: "sheets failure", customers: fakeCustomers{1: "+70000000000"}, phones: fakePhones{err: errSheets}, want: errSheets},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applications := &fakeApplications{}
			b := newTestBot(t, tt.customers, tt.phones, applications, &fakeMessenger{})

			err := b.apply(t.Context(), 1, "а000аа78")

			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if tt.want != nil {
				if len(applications.sent) != 0 {
					t.Errorf("sent %v after a failure", applications.sent)
				}
				return
			}
			want := []application{{phone: "+70000000000", plate: "а000аа78", gates: []string{"north", "south"}}}
			if !slices.EqualFunc(applications.sent, want, func(a, b application) bool {
				return a.phone == b.phone && a.plate == b.plate && slices.Equal(a.gates, b.gates)
			}) {
				t.Errorf("sent %v, want %v", applications.sent, want)
			}
		})
	}
}
