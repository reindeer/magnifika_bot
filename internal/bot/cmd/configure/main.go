package configure

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/pborman/getopt/v2"
	"golang.org/x/exp/maps"

	"gitlab.com/gorib/waffle/app"
)

type Command interface {
	app.Command
}

type Registry interface {
	List(ctx context.Context) (map[string]string, error)
	Save(ctx context.Context, code, value string) error
}

func New(base app.BaseCommand, registry Registry) *command {
	return &command{
		BaseCommand: base,
		registry:    registry,
	}
}

type command struct {
	app.BaseCommand
	registry Registry
	code     string
}

func (c *command) Run(ctx context.Context) error {
	err := c.Usage()
	if err != nil {
		return err
	}

	in := bufio.NewReader(os.Stdin)
	value, err := in.ReadString('\n')
	value = strings.Trim(value, "\n")
	if err != nil {
		return fmt.Errorf("unable to read value: %w", err)
	}

	return c.registry.Save(ctx, c.code, value)
}

func (c *command) Usage() error {
	var (
		help bool
		code string
	)
	getopt.FlagLong(&help, "help", 'h', "Display this help message")
	getopt.FlagLong(&code, "code", 'c', "Code to be configured")

	getopt.Parse()
	if help || code == "" {
		getopt.SetParameters("code")
		_, _ = os.Stderr.WriteString(c.Description() + "\n")
		getopt.PrintUsage(os.Stderr)
		if code == "" {
			records, err := c.registry.List(context.Background())
			if err != nil {
				return err
			}
			codes := maps.Keys(records)
			slices.Sort(codes)
			_, _ = os.Stderr.WriteString("Current records:\n")
			for _, code := range codes {
				_, _ = os.Stderr.WriteString("\t" + code + "\n")
			}
		}
		os.Exit(2)
	}

	c.code = code
	return nil
}
