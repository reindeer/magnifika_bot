package serve

import (
	"context"
	"os"

	"github.com/pborman/getopt/v2"

	"gitlab.com/gorib/waffle/app"
)

type Command interface {
	app.Command
}

type BotManagement interface {
	Setup(ctx context.Context) error
}

func New(base app.BaseCommand, management BotManagement) *command {
	return &command{
		BaseCommand: base,
		management:  management,
	}
}

type command struct {
	app.BaseCommand
	management BotManagement
	cancel     func()
}

func (c *command) Run(ctx context.Context) error {
	err := c.Usage()
	if err != nil {
		return err
	}

	ctx, c.cancel = context.WithCancel(ctx)
	return c.management.Setup(ctx)
}

func (c *command) Usage() error {
	var (
		help bool
	)
	getopt.FlagLong(&help, "help", 'h', "Display this help message")
	getopt.Parse()
	if help {
		getopt.SetParameters("")
		_, _ = os.Stderr.WriteString(c.Description() + "\n")
		getopt.PrintUsage(os.Stderr)
		os.Exit(2)
	}
	return nil
}

func (c *command) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	return nil
}
