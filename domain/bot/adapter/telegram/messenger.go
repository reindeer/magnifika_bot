package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

type Sender interface {
	SendMessage(ctx context.Context, params *bot.SendMessageParams) (*models.Message, error)
	SendContact(ctx context.Context, params *bot.SendContactParams) (*models.Message, error)
}

func NewMessenger(sender Sender) *messenger {
	return &messenger{sender: sender}
}

type messenger struct {
	sender Sender

	mu                    sync.Mutex
	contactsBlockedBefore time.Time
}

func (m *messenger) SendText(ctx context.Context, chatId int64, text string, keyboard []string) error {
	_, err := m.sender.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        text,
		ReplyMarkup: markup(keyboard),
	})
	return err
}

func (m *messenger) SendContact(ctx context.Context, chatId int64, contact model.ContactCard, keyboard []string) error {
	if m.blocked() {
		return model.ErrRateLimited
	}
	_, err := m.sender.SendContact(ctx, &bot.SendContactParams{
		ChatID:      chatId,
		PhoneNumber: contact.Phone,
		FirstName:   contact.Name,
		ReplyMarkup: markup(keyboard),
	})
	if tooMany, ok := errors.AsType[*bot.TooManyRequestsError](err); ok {
		m.block(time.Duration(tooMany.RetryAfter) * time.Second)
		return fmt.Errorf("%w: %w", model.ErrRateLimited, err)
	}
	return err
}

func (m *messenger) blocked() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.contactsBlockedBefore.After(time.Now())
}

func (m *messenger) block(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contactsBlockedBefore = time.Now().Add(duration)
}

func markup(keyboard []string) *models.ReplyKeyboardMarkup {
	rows := make([][]models.KeyboardButton, 0, len(keyboard))
	for _, button := range keyboard {
		rows = append(rows, []models.KeyboardButton{{Text: button}})
	}
	return &models.ReplyKeyboardMarkup{Keyboard: rows, IsPersistent: true, ResizeKeyboard: true}
}
