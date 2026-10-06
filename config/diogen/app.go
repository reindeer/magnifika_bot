package diogen

import (
	"context"
	"fmt"
	"iter"
	"runtime/debug"
	"slices"

	"gitlab.com/gorib/pry"

	"github.com/reindeer/magnifika_bot/config/app"
	"github.com/reindeer/magnifika_bot/config/diogen/login"
	"github.com/reindeer/magnifika_bot/config/diogen/migrate"
	"github.com/reindeer/magnifika_bot/config/diogen/serve"
)

func Get(name string) (command, bool) {
	i := slices.IndexFunc(commands, func(c command) bool { return c.name == name })
	if i < 0 {
		return command{}, false
	}
	return commands[i], true
}

type application interface {
	AppApp() *app.App
	Shutdown() func() error
}

func load[Application application](appInit func() (Application, error)) func() (application, error) {
	return func() (application, error) {
		return appInit()
	}
}

type command struct {
	name        string
	description string
	load        func() (application, error)
}

// failed: the command failed, and that is in the log already.
// err: the container did not build, or did not close after the logger it would go to was closed.
func (c command) Run(ctx context.Context, args []string) (failed bool, err error) {
	container, err := c.load()
	if err != nil {
		return false, err
	}
	defer func() { err = container.Shutdown()() }()
	a := container.AppApp()
	// Logged before the close above shuts the logger's sinks.
	defer func() {
		if p := recover(); p != nil {
			a.Logger.Error(fmt.Errorf("panic: %v\n\n%s", p, debug.Stack()), pry.Ctx(ctx))
			failed = true
		}
	}()
	if runErr := a.Command.Run(ctx, args); runErr != nil {
		a.Logger.Error(runErr, pry.Ctx(ctx))
		return true, nil
	}
	return false, nil
}

var commands = []command{
	{name: "bot:serve", description: "Start the bot", load: load(serve.InitApp)},
	{name: "google:login", description: "Create google token to access to spreadsheets", load: load(login.InitApp)},
	{name: "migrate", description: "Migrate the database: up (default), down, status", load: load(migrate.InitApp)},
}

// name → description, in table order.
func Commands() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for _, c := range commands {
			if !yield(c.name, c.description) {
				return
			}
		}
	}
}
