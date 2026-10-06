package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

type limitedSender struct {
	contacts int
}

func (s *limitedSender) SendMessage(context.Context, *bot.SendMessageParams) (*models.Message, error) {
	return &models.Message{}, nil
}

func (s *limitedSender) SendContact(context.Context, *bot.SendContactParams) (*models.Message, error) {
	s.contacts++
	return nil, &bot.TooManyRequestsError{Message: "too many requests", RetryAfter: 60}
}

func TestContactsPauseAfterRateLimit(t *testing.T) {
	sender := &limitedSender{}
	m := NewMessenger(sender)
	contact := model.ContactCard{Name: "Охрана", Phone: "+71"}

	first := m.SendContact(t.Context(), 1, contact, nil)
	second := m.SendContact(t.Context(), 1, contact, nil)

	if !errors.Is(first, model.ErrRateLimited) || !errors.Is(second, model.ErrRateLimited) {
		t.Fatalf("got %v and %v, want %v", first, second, model.ErrRateLimited)
	}
	if sender.contacts != 1 {
		t.Errorf("asked Telegram %d times, want once until the pause is over", sender.contacts)
	}
}
