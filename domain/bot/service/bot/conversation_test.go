package bot

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

type failingCustomers struct{ err error }

func (f failingCustomers) Customer(context.Context, int64) (*model.Customer, error) {
	return nil, f.err
}

func (f failingCustomers) SaveCustomer(context.Context, *model.Customer) error { return f.err }

type fakeMessenger struct {
	contactErr error
	sent       []string
}

func (f *fakeMessenger) SendText(_ context.Context, chatId int64, text string, keyboard []string) error {
	if !slices.Equal(keyboard, []string{ApplicationShortcut, EmergencyShortcut, DispatcherShortcut, GuardShortcut}) {
		return fmt.Errorf("keyboard %v", keyboard)
	}
	f.sent = append(f.sent, fmt.Sprintf("text to %d: %s", chatId, text))
	return nil
}

func (f *fakeMessenger) SendContact(_ context.Context, chatId int64, contact model.ContactCard, _ []string) error {
	if f.contactErr != nil {
		return f.contactErr
	}
	f.sent = append(f.sent, fmt.Sprintf("contact to %d: %s %s", chatId, contact.Name, contact.Phone))
	return nil
}

func TestHandle(t *testing.T) {
	known := fakeCustomers{1: "+70000000000"}
	listed := fakePhones{gates: map[string][]string{"+70000000000": {"north"}}}
	tests := []struct {
		name      string
		message   model.Message
		customers Customers
		phones    fakePhones
		messenger fakeMessenger
		want      string
	}{
		{name: "start of a stranger", message: model.Message{Text: StartShortcut}, customers: fakeCustomers{}, want: "text to 7: " + WelcomePhrase},
		{name: "start of a customer", message: model.Message{Text: StartShortcut}, customers: known, want: "text to 7: Привет, рад видеть тебя снова! Если у тебя новый телефон, пришли мне его. Сейчас у меня записан: +70000000000"},
		{name: "start failure", message: model.Message{Text: StartShortcut}, customers: failingCustomers{errSheets}, want: "text to 7: " + OopsPhrase},
		{name: "contact button", message: model.Message{Text: GuardShortcut}, customers: fakeCustomers{}, want: "contact to 7: Магнифика: Охрана +71"},
		{name: "contact while rate limited", message: model.Message{Text: GuardShortcut}, customers: fakeCustomers{}, messenger: fakeMessenger{contactErr: model.ErrRateLimited}, want: "text to 7: Магнифика: Охрана\n+71"},
		{name: "contact refused", message: model.Message{Text: GuardShortcut}, customers: fakeCustomers{}, messenger: fakeMessenger{contactErr: errors.New("bad request")}, want: "text to 7: Магнифика: Охрана\n+71"},
		{name: "typed phone", message: model.Message{Text: "+70000000000"}, customers: fakeCustomers{}, phones: listed, want: "text to 7: " + ReadyForApplicationPhrase},
		{name: "shared contact", message: model.Message{SharedPhone: "+70000000000"}, customers: fakeCustomers{}, phones: listed, want: "text to 7: " + ReadyForApplicationPhrase},
		{name: "phone unknown to the management", message: model.Message{Text: "+70000000000"}, customers: fakeCustomers{}, want: "text to 7: " + NotFoundPhrase},
		{name: "registration failure", message: model.Message{Text: "+70000000000"}, customers: fakeCustomers{}, phones: fakePhones{err: errSheets}, want: "text to 7: " + OopsPhrase},
		{name: "application button", message: model.Message{Text: ApplicationShortcut}, customers: known, phones: listed, want: "text to 7: " + WaitForPlatePhrase},
		{name: "application button failure", message: model.Message{Text: ApplicationShortcut}, customers: known, phones: fakePhones{err: errSheets}, want: "text to 7: " + OopsPhrase},
		{name: "plate", message: model.Message{Text: "а000аа78."}, customers: known, phones: listed, want: "text to 7: " + ApplicationSentPhrase},
		{name: "plate of a stranger", message: model.Message{Text: "а000аа78"}, customers: fakeCustomers{}, want: "text to 7: " + UnknownPersonPhrase},
		{name: "plate after the phone left the list", message: model.Message{Text: "а000аа78"}, customers: known, want: "text to 7: " + PhoneChangedPhrase},
		{name: "plate rejected", message: model.Message{Text: "а000аа78"}, customers: known, phones: fakePhones{err: errSheets}, want: "text to 7: " + ApplicationFailedPhrase},
		{name: "anything else", message: model.Message{Text: "hello"}, customers: fakeCustomers{}, want: "text to 7: " + UnknownPhrase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := newTestBot(t, tt.customers, tt.phones, &fakeApplications{}, &tt.messenger)
			tt.message.ChatId = 7
			tt.message.CustomerId = 1

			if err := b.Handle(t.Context(), tt.message); err != nil {
				t.Fatal(err)
			}

			if !slices.Equal(tt.messenger.sent, []string{tt.want}) {
				t.Errorf("sent %q, want %q", tt.messenger.sent, tt.want)
			}
		})
	}
}

func TestPlateIsTrimmed(t *testing.T) {
	applications := &fakeApplications{}
	b := newTestBot(t, fakeCustomers{1: "+70000000000"}, fakePhones{gates: map[string][]string{"+70000000000": {"north"}}}, applications, &fakeMessenger{})

	if err := b.Handle(t.Context(), model.Message{CustomerId: 1, Text: " а000аа78, "}); err != nil {
		t.Fatal(err)
	}

	if len(applications.sent) != 1 || applications.sent[0].plate != "а000аа78" {
		t.Errorf("sent %v", applications.sent)
	}
}
