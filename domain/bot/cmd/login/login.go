package login

import (
	"context"
	"fmt"
	"io"

	"gitlab.com/gorib/pry"
)

type Auth interface {
	Url() string
	Token(ctx context.Context, code string) (string, error)
}

func NewLogin(auth Auth, in io.Reader, logger pry.Logger) *Login {
	return &Login{auth: auth, in: in, logger: logger}
}

type Login struct {
	auth   Auth
	in     io.Reader
	logger pry.Logger
}

func (l *Login) Run(ctx context.Context, _ []string) error {
	l.logger.Info("Go to the link in your browser then type the authorization code", pry.Field("url", l.auth.Url()))

	type scanned struct {
		code string
		err  error
	}
	scan := make(chan scanned, 1)
	go func() {
		var code string
		_, err := fmt.Fscan(l.in, &code)
		scan <- scanned{code: code, err: err}
	}()

	var code string
	select {
	case <-ctx.Done():
		return nil
	case s := <-scan:
		if s.err != nil {
			return fmt.Errorf("unable to read authorization code: %w", s.err)
		}
		code = s.code
	}

	token, err := l.auth.Token(ctx, code)
	if err != nil {
		return err
	}
	l.logger.Info("Add the token to your env file", pry.Field("GOOGLE_TOKEN", token))
	return nil
}
