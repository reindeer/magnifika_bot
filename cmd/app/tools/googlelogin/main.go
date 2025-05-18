package googlelogin

import (
	"context"

	"gitlab.com/gorib/waffle/app"
)

type Command interface {
	app.Command
}

type GoogleService interface {
	Login(ctx context.Context) error
}

func New(base app.BaseCommand, service GoogleService) *command {
	return &command{
		BaseCommand: base,
		service:     service,
	}
}

type command struct {
	app.BaseCommand
	service GoogleService
	cancel  func()
}

func (c *command) Run(ctx context.Context) error {
	err := c.Usage()
	if err != nil {
		return err
	}

	ctx, c.cancel = context.WithCancel(ctx)
	return c.service.Login(ctx)
}

func (c *command) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	return nil
}
