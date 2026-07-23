package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"orquesta/internal/bootstrap"
	"orquesta/internal/i18n"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	catalog, err := i18n.LoadBundled()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "i18n_catalog_unavailable")
		return 1
	}
	if len(arguments) == 1 && arguments[0] == "version" {
		_, _ = fmt.Fprintln(stdout, version)
		return 0
	}
	if len(arguments) > 0 && arguments[0] == "command" {
		return runCommand(arguments[1:], stdout, stderr)
	}
	if len(arguments) == 0 || arguments[0] != "serve" {
		_, _ = fmt.Fprintln(stderr, catalog.Text(i18n.DefaultLocale, "error.invalid_request"))
		_, _ = fmt.Fprintln(stderr, commandUsage)
		_, _ = fmt.Fprintln(stderr, "orquesta serve [--config path] | orquesta version")
		return 2
	}
	flags := flag.NewFlagSet("orquesta serve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "")
	if err := flags.Parse(arguments[1:]); err != nil || flags.NArg() != 0 {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(stdout, "orquesta serve [--config path]")
			return 0
		}
		_, _ = fmt.Fprintln(stderr, catalog.Text(i18n.DefaultLocale, "error.invalid_request"))
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err = bootstrap.Run(ctx, bootstrap.Options{
		ConfigPath: *configPath,
		Version:    version,
		ReportError: func(err error) {
			slog.Error("worker_error", "code", err.Error())
		},
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s code=%s\n", catalog.Text(i18n.DefaultLocale, "error.internal"), err.Error())
		return 1
	}
	return 0
}
