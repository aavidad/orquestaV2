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

	"orquesta/internal/adapters/agent/codex"
	"orquesta/internal/bootstrap"
	"orquesta/internal/i18n"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if codex.IsLocalSupervisorInvocation(arguments) {
		return codex.RunLocalSupervisor()
	}
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
		return runCommand(arguments[1:], catalog, stdout, stderr)
	}
	if len(arguments) == 0 || arguments[0] != "serve" {
		if !writeCatalogText(stderr, catalog, i18n.DefaultLocale, "error.invalid_request") ||
			!writeCatalogText(stderr, catalog, i18n.DefaultLocale, "cli.root.usage") {
			_, _ = fmt.Fprintln(stderr, "code=i18n_catalog_unavailable")
			return 1
		}
		return 2
	}
	flags := flag.NewFlagSet("orquesta serve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "")
	if err := flags.Parse(arguments[1:]); err != nil || flags.NArg() != 0 {
		if errors.Is(err, flag.ErrHelp) {
			if writeCatalogText(stdout, catalog, i18n.DefaultLocale, "cli.serve.usage") {
				return 0
			}
			_, _ = fmt.Fprintln(stderr, "code=i18n_catalog_unavailable")
			return 1
		}
		if !writeCatalogText(stderr, catalog, i18n.DefaultLocale, "error.invalid_request") {
			_, _ = fmt.Fprintln(stderr, "code=i18n_catalog_unavailable")
			return 1
		}
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
		text, textErr := catalog.Text(i18n.DefaultLocale, "error.internal")
		if textErr != nil {
			_, _ = fmt.Fprintln(stderr, "code=i18n_catalog_unavailable")
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "%s code=internal\n", text)
		return 1
	}
	return 0
}

func writeCatalogText(writer io.Writer, catalog *i18n.Catalog, locale, key string) bool {
	text, err := catalog.Text(locale, key)
	if err != nil {
		return false
	}
	_, _ = fmt.Fprintln(writer, text)
	return true
}
