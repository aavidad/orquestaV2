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
	"strings"
	"syscall"

	"orquesta/internal/bootstrap"
	"orquesta/internal/i18n"
)

var version = "dev"

// runRuntime is the single production composition boundary. Keeping the
// bootstrap call explicit also lets the command contract prove that a rejected
// microVM configuration is returned to the operator without a second runtime
// attempt or an isolation fallback.
var runRuntime = bootstrap.Run

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if exitCode, handled := bootstrap.DispatchPrivateInvocation(arguments); handled {
		return exitCode
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
	if len(arguments) > 1 && arguments[0] == "credentials" && arguments[1] == "provision-codex-microvm" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return bootstrap.RunProvisionCodexMicroVMCredentials(
			ctx, arguments[2:], catalog, stdout, stderr,
		)
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
	err = runRuntime(ctx, bootstrap.Options{
		ConfigPath:  *configPath,
		Version:     version,
		ReportError: reportWorkerError,
	})
	if err != nil {
		code, messageKey := serveErrorPresentation(err)
		text, textErr := catalog.Text(i18n.DefaultLocale, messageKey)
		if textErr != nil {
			_, _ = fmt.Fprintln(stderr, "code=i18n_catalog_unavailable")
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "%s code=%s\n", text, code)
		return 1
	}
	return 0
}

var serveErrorMessageKeys = map[string]string{"bootstrap.runtime_isolation_not_composed": "error.bootstrap.runtime_isolation_not_composed"}

func serveErrorPresentation(err error) (string, string) {
	if err != nil {
		if key, ok := serveErrorMessageKeys[err.Error()]; ok {
			return err.Error(), key
		}
	}
	return "internal", "error.internal"
}

type workerCauseCodeError interface {
	error
	CauseCode() string
}

func reportWorkerError(err error) {
	if err == nil {
		return
	}
	slog.LogAttrs(context.Background(), slog.LevelError, "worker_error", workerErrorLogAttrs(err)...)
}

func workerErrorLogAttrs(err error) []slog.Attr {
	attributes := []slog.Attr{slog.String("code", err.Error())}
	var cause workerCauseCodeError
	if errors.As(err, &cause) {
		code := cause.CauseCode()
		if validWorkerCauseCode(code) {
			attributes = append(attributes, slog.String("cause_code", code))
		}
	}
	return attributes
}

func validWorkerCauseCode(code string) bool {
	if len(code) == 0 || len(code) > 160 || !strings.HasPrefix(code, "test_attestor.") {
		return false
	}
	for _, character := range code {
		if character >= 'a' && character <= 'z' ||
			character >= '0' && character <= '9' ||
			character == '.' || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func writeCatalogText(writer io.Writer, catalog *i18n.Catalog, locale, key string) bool {
	text, err := catalog.Text(locale, key)
	if err != nil {
		return false
	}
	_, _ = fmt.Fprintln(writer, text)
	return true
}
