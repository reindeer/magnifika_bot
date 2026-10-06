package bot

import (
	"context"
	"fmt"

	"gitlab.com/gorib/pry"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

type Customers interface {
	Customer(ctx context.Context, id int64) (*model.Customer, error)
	SaveCustomer(ctx context.Context, customer *model.Customer) error
}

type Phones interface {
	Gates(ctx context.Context, phone string) ([]string, error)
}

type Applications interface {
	Apply(ctx context.Context, phone, plate string, gates []string) error
}

type Messenger interface {
	SendText(ctx context.Context, chatId int64, text string, keyboard []string) error
	SendContact(ctx context.Context, chatId int64, contact model.ContactCard, keyboard []string) error
}

func NewBot(
	guardPhone, dispatcherPhone, emergencyPhone string,
	customers Customers,
	phones Phones,
	applications Applications,
	messenger Messenger,
	logger pry.Logger,
) (*bot, error) {
	contacts := map[string]string{
		GuardShortcut:      guardPhone,
		DispatcherShortcut: dispatcherPhone,
		EmergencyShortcut:  emergencyPhone,
	}
	for shortcut, phone := range contacts {
		if phone == "" {
			return nil, fmt.Errorf("phone for %q is empty", shortcut)
		}
	}
	return &bot{
		contacts:     contacts,
		customers:    customers,
		phones:       phones,
		applications: applications,
		messenger:    messenger,
		logger:       logger,
	}, nil
}

type bot struct {
	contacts     map[string]string
	customers    Customers
	phones       Phones
	applications Applications
	messenger    Messenger
	logger       pry.Logger
}

func (b *bot) phone(ctx context.Context, customerId int64) (string, error) {
	customer, err := b.customers.Customer(ctx, customerId)
	if err != nil {
		return "", err
	}
	return customer.Phone, nil
}

func (b *bot) register(ctx context.Context, customerId int64, phone string) error {
	gates, err := b.phones.Gates(ctx, phone)
	if err != nil {
		return err
	}
	if len(gates) == 0 {
		return model.ErrUnknownPhone
	}
	return b.customers.SaveCustomer(ctx, &model.Customer{Id: customerId, Phone: phone})
}

func (b *bot) apply(ctx context.Context, customerId int64, plate string) error {
	phone, gates, err := b.applicant(ctx, customerId)
	if err != nil {
		return err
	}
	return b.applications.Apply(ctx, phone, plate, gates)
}

func (b *bot) applicant(ctx context.Context, customerId int64) (string, []string, error) {
	customer, err := b.customers.Customer(ctx, customerId)
	if err != nil {
		return "", nil, err
	}
	gates, err := b.phones.Gates(ctx, customer.Phone)
	if err != nil {
		return "", nil, err
	}
	if len(gates) == 0 {
		return "", nil, model.ErrPhoneChanged
	}
	return customer.Phone, gates, nil
}
