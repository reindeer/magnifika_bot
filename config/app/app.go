package app

import (
	"context"

	"gitlab.com/gorib/pry"
)

type Command interface {
	Run(ctx context.Context, args []string) error
}

// The root of a command's container: the logger is built with the command, read or not.
type App struct {
	Command Command
	Logger  pry.Logger
}

func New(command Command, logger pry.Logger) *App {
	return &App{Command: command, Logger: logger}
}
