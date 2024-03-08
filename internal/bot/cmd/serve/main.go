package serve

import (
	"context"
	"os"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/pborman/getopt/v2"

	"gitlab.com/gorib/pry"
	"gitlab.com/gorib/waffle/app"
)

type Command interface {
	app.Command
}

type handler struct {
	models.BotCommand
	handler func(ctx context.Context, b *bot.Bot, update *models.Update)
}

type BotManagement interface {
	Setup(ctx context.Context, token string) error
}

func New(token string, base app.BaseCommand, management BotManagement) *command {
	return &command{
		BaseCommand: base,
		management:  management,
		token:       token,
	}
}

type command struct {
	app.BaseCommand
	management BotManagement
	logger     pry.Logger
	token      string
	cancel     func()
}

func (c *command) Run(ctx context.Context) error {
	err := c.Usage()
	if err != nil {
		return err
	}

	ctx, c.cancel = context.WithCancel(ctx)
	return c.management.Setup(ctx, c.token)
}

func (c *command) Usage() error {
	var (
		help  bool
		token string
	)
	getopt.FlagLong(&help, "help", 'h', "Display this help message")
	getopt.FlagLong(&token, "token", 't', "Token to access the Telegram API")
	getopt.Parse()
	if help {
		getopt.SetParameters("")
		_, _ = os.Stderr.WriteString(c.Description() + "\n")
		getopt.PrintUsage(os.Stderr)
		os.Exit(2)
	}
	if token != "" {
		c.token = token
	}
	return nil
}

func (c *command) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	return nil
}
