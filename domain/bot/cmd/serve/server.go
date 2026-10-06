package serve

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"gitlab.com/gorib/pry"

	"github.com/reindeer/magnifika_bot/domain/bot/model"
)

type Updates interface {
	GetMe(ctx context.Context) (*models.User, error)
	RegisterHandlerMatchFunc(matchFunc bot.MatchFunc, f bot.HandlerFunc, m ...bot.Middleware) string
	Start(ctx context.Context)
}

type Bot interface {
	Handle(ctx context.Context, message model.Message) error
}

func NewServer(updates Updates, service Bot, logger pry.Logger) *Server {
	return &Server{updates: updates, service: service, logger: logger}
}

type Server struct {
	updates Updates
	service Bot
	logger  pry.Logger
}

func (s *Server) Run(ctx context.Context, _ []string) error {
	if _, err := s.updates.GetMe(ctx); err != nil {
		return err
	}
	s.updates.RegisterHandlerMatchFunc(func(*models.Update) bool { return true }, s.handle)
	s.updates.Start(ctx)
	return nil
}

func (s *Server) handle(ctx context.Context, _ *bot.Bot, update *models.Update) {
	// Nothing up the stack recovers: a panic on one message would end the bot.
	defer func() {
		if p := recover(); p != nil {
			s.logger.Error(fmt.Errorf("panic: %v\n\n%s", p, debug.Stack()), pry.Ctx(ctx), pry.Field("message", update))
		}
	}()
	if update.Message == nil || update.Message.From == nil {
		return
	}
	message := model.Message{
		ChatId:     update.Message.Chat.ID,
		CustomerId: update.Message.From.ID,
		Text:       update.Message.Text,
	}
	if update.Message.Contact != nil {
		message.SharedPhone = update.Message.Contact.PhoneNumber
	}
	if err := s.service.Handle(ctx, message); err != nil {
		s.logger.Error(err, pry.Ctx(ctx), pry.Field("message", update))
	}
}
