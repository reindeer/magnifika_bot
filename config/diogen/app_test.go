package diogen

import (
	"context"
	"errors"
	"strings"
	"testing"

	"gitlab.com/gorib/pry"

	"github.com/reindeer/magnifika_bot/config/app"
)

type commandFunc func(context.Context, []string) error

func (f commandFunc) Run(ctx context.Context, args []string) error { return f(ctx, args) }

type recordingLogger struct {
	pry.Logger
	errors []error
}

func (l *recordingLogger) Error(message any, _ ...pry.Option) {
	if err, ok := message.(error); ok {
		l.errors = append(l.errors, err)
	}
}

type fakeContainer struct {
	app   *app.App
	close func() error
}

func (c fakeContainer) AppApp() *app.App       { return c.app }
func (c fakeContainer) Shutdown() func() error { return c.close }

func built(run commandFunc, logger pry.Logger, close func() error) command {
	container := fakeContainer{app: app.New(run, logger), close: close}
	return command{load: func() (application, error) { return container, nil }}
}

func TestABuildFailureIsReturnedAndNothingRuns(t *testing.T) {
	building := errors.New("no token")
	c := command{load: func() (application, error) { return nil, building }}

	failed, err := c.Run(t.Context(), nil)

	if failed || !errors.Is(err, building) {
		t.Fatalf("got failed %v, err %v; want not failed and %v", failed, err, building)
	}
}

func TestCommandFailureIsLoggedBeforeClosing(t *testing.T) {
	failure := errors.New("sideways")
	logger := &recordingLogger{Logger: pry.Noop()}
	var loggedBeforeClose bool
	c := built(func(context.Context, []string) error { return failure }, logger, func() error {
		loggedBeforeClose = len(logger.errors) == 1
		return nil
	})

	failed, err := c.Run(t.Context(), nil)

	if !failed || err != nil {
		t.Fatalf("got failed %v, err %v; want failed and no error", failed, err)
	}
	if !loggedBeforeClose || !errors.Is(logger.errors[0], failure) {
		t.Errorf("logged %v before close: %v", logger.errors, loggedBeforeClose)
	}
}

func TestCloseFailureIsReturnedBesideTheCommandFailure(t *testing.T) {
	closing := errors.New("cannot close")
	c := built(func(context.Context, []string) error { return errors.New("sideways") }, pry.Noop(), func() error { return closing })

	failed, err := c.Run(t.Context(), nil)

	if !failed || !errors.Is(err, closing) {
		t.Fatalf("got failed %v, err %v; want failed and %v", failed, err, closing)
	}
}

func TestAPanicIsAFailureLoggedBeforeClosing(t *testing.T) {
	logger := &recordingLogger{Logger: pry.Noop()}
	closing := errors.New("cannot close")
	var loggedBeforeClose bool
	c := built(func(context.Context, []string) error { panic("sideways") }, logger, func() error {
		loggedBeforeClose = len(logger.errors) == 1 && strings.HasPrefix(logger.errors[0].Error(), "panic: sideways")
		return closing
	})

	failed, err := c.Run(t.Context(), nil)

	if !failed || !errors.Is(err, closing) {
		t.Fatalf("got failed %v, err %v; want failed and %v", failed, err, closing)
	}
	if !loggedBeforeClose {
		t.Errorf("logged %v before close, want the panic", logger.errors)
	}
}
