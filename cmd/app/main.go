package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/reindeer/magnifika_bot/config/diogen"

	"gitlab.com/gorib/env"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd, ok := diogen.Get(os.Args[1])
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Unknown command %q.\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	failed, err := cmd.Run(ctx, os.Args[2:])
	stop()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, describe(err))
	}
	if failed || err != nil {
		os.Exit(1)
	}
}

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage: %s <command> [args]\n\nAvailable commands:\n", os.Args[0])
	for name, description := range diogen.Commands() {
		_, _ = fmt.Fprintf(os.Stderr, "  %-14s %s\n", name, description)
	}
}

func describe(err error) string {
	if errors.Is(err, env.ErrNoEnvFound) || errors.Is(err, env.ErrWrongFormat) {
		message, _, _ := strings.Cut(err.Error(), "\n")
		return message
	}
	return err.Error()
}
