package serve

import (
	"context"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"

	"gitlab.com/gorib/pry"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

type panickingBot struct{}

func (panickingBot) Handle(context.Context, model.Message) error { panic("sideways") }

type recordingLogger struct {
	pry.Logger
	errors []error
}

func (l *recordingLogger) Error(message any, _ ...pry.Option) {
	if err, ok := message.(error); ok {
		l.errors = append(l.errors, err)
	}
}

func TestAPanickingMessageIsLoggedAndTheBotGoesOn(t *testing.T) {
	logger := &recordingLogger{Logger: pry.Noop()}
	s := NewServer(nil, panickingBot{}, logger)
	update := &models.Update{Message: &models.Message{From: &models.User{ID: 1}, Chat: models.Chat{ID: 1}}}

	s.handle(t.Context(), nil, update)

	if len(logger.errors) != 1 || !strings.HasPrefix(logger.errors[0].Error(), "panic: sideways") {
		t.Errorf("logged %v, want the panic", logger.errors)
	}
}
