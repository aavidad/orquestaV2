package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"

	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/config"
	"orquesta/internal/i18n"
)

const stateMigrationUsageKey = "cli.state.migrate.usage"

var stateMigrationMessageKeys = map[string]string{
	"cli.state_migration_arguments_invalid":           "error.cli.state_migration_arguments_invalid",
	"cli.state_migration_canceled":                    "error.cli.state_migration_canceled",
	"cli.state_migration_committed_postcheck_pending": "error.cli.state_migration_committed_postcheck_pending",
	"cli.state_migration_configuration_invalid":       "error.cli.state_migration_configuration_invalid",
	"cli.state_migration_failed":                      "error.cli.state_migration_failed",
}

type stateMigrationDependencies struct {
	loadSnapshot func(context.Context, string) (config.Snapshot, error)
	migrate      func(context.Context, sqlite.StateMigrationRequest) (sqlite.StateMigrationReceipt, error)
}

func productionStateMigrationDependencies() stateMigrationDependencies {
	return stateMigrationDependencies{
		loadSnapshot: loadStateMigrationConfigSnapshot,
		migrate:      sqlite.MigrateStateOnly,
	}
}

// RunStateMigration composes only canonical configuration and the bounded
// SQLite maintenance adapter. It deliberately has no runtime dependency.
func RunStateMigration(
	ctx context.Context,
	arguments []string,
	catalog *i18n.Catalog,
	stdout io.Writer,
	stderr io.Writer,
) int {
	return runStateMigration(
		ctx, arguments, catalog, stdout, stderr, productionStateMigrationDependencies(),
	)
}

func runStateMigration(
	ctx context.Context,
	arguments []string,
	catalog *i18n.Catalog,
	stdout io.Writer,
	stderr io.Writer,
	dependencies stateMigrationDependencies,
) int {
	flags := flag.NewFlagSet("orquesta state migrate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "")
	mode := flags.String("mode", "", "")
	expectFrom := flags.Int("expect-from", 0, "")
	expectTo := flags.Int("expect-to", 0, "")
	parseErr := flags.Parse(arguments)
	locale := i18n.DefaultLocale

	if catalog == nil {
		_, _ = io.WriteString(stderr, "code=i18n_catalog_unavailable\n")
		return 1
	}
	if errors.Is(parseErr, flag.ErrHelp) {
		if usage, err := catalog.Text(locale, stateMigrationUsageKey); err == nil {
			_, _ = io.WriteString(stdout, usage+"\n")
			return 0
		}
		_, _ = io.WriteString(stderr, "code=i18n_catalog_unavailable\n")
		return 1
	}
	if parseErr != nil || flags.NArg() != 0 ||
		!validStateMigrationConfigPath(*configPath) ||
		*mode != string(sqlite.StateMigrationModeNoRuntime) ||
		*expectFrom != 39 || *expectTo != 41 {
		writeStateMigrationDiagnostic(
			stderr, catalog, locale,
			"cli.state_migration_arguments_invalid",
			stateMigrationMessageKeys["cli.state_migration_arguments_invalid"],
			"",
		)
		return 2
	}
	if ctx == nil || dependencies.loadSnapshot == nil || dependencies.migrate == nil {
		writeStateMigrationDiagnostic(
			stderr, catalog, locale,
			"cli.state_migration_configuration_invalid",
			stateMigrationMessageKeys["cli.state_migration_configuration_invalid"],
			"",
		)
		return 2
	}
	if err := ctx.Err(); err != nil {
		writeStateMigrationDiagnostic(
			stderr, catalog, locale,
			"cli.state_migration_canceled",
			stateMigrationMessageKeys["cli.state_migration_canceled"],
			"",
		)
		return 1
	}

	snapshot, err := dependencies.loadSnapshot(ctx, *configPath)
	if err != nil || !filepath.IsAbs(snapshot.StateSQLitePath()) {
		writeStateMigrationDiagnostic(
			stderr, catalog, locale,
			"cli.state_migration_configuration_invalid",
			stateMigrationMessageKeys["cli.state_migration_configuration_invalid"],
			"",
		)
		return 2
	}
	locale = snapshot.APILocale()
	if _, err := catalog.Resolve(locale); err != nil {
		writeStateMigrationDiagnostic(
			stderr, catalog, i18n.DefaultLocale,
			"cli.state_migration_configuration_invalid",
			stateMigrationMessageKeys["cli.state_migration_configuration_invalid"],
			"",
		)
		return 2
	}

	receipt, err := dependencies.migrate(ctx, sqlite.StateMigrationRequest{
		Options: sqlite.Options{
			Path: snapshot.StateSQLitePath(), BusyTimeout: snapshot.StateSQLiteBusyTimeout(),
			MaxOpenConnections: int(snapshot.StateSQLiteMaxOpenConnections()),
		},
		Mode: sqlite.StateMigrationMode(*mode), ExpectFrom: *expectFrom, ExpectTo: *expectTo,
	})
	if err != nil {
		var committed *sqlite.CommittedStateMigrationError
		if errors.As(err, &committed) {
			if writeStateMigrationReceipt(stdout, committed.Receipt) != nil {
				writeStateMigrationDiagnostic(
					stderr, catalog, locale,
					"cli.output_invalid", "error.cli.output_invalid", "",
				)
				return 1
			}
			writeStateMigrationDiagnostic(
				stderr, catalog, locale,
				"cli.state_migration_committed_postcheck_pending",
				stateMigrationMessageKeys["cli.state_migration_committed_postcheck_pending"],
				stateMigrationCauseCode(err),
			)
			return 1
		}
		code := "cli.state_migration_failed"
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			code = "cli.state_migration_canceled"
		}
		writeStateMigrationDiagnostic(
			stderr, catalog, locale, code, stateMigrationMessageKeys[code], stateMigrationCauseCode(err),
		)
		return 1
	}
	if err := writeStateMigrationReceipt(stdout, receipt); err != nil {
		writeStateMigrationDiagnostic(
			stderr, catalog, locale,
			"cli.output_invalid", "error.cli.output_invalid", "",
		)
		return 1
	}
	return 0
}

func writeStateMigrationReceipt(writer io.Writer, receipt sqlite.StateMigrationReceipt) error {
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	_, err = writer.Write(encoded)
	return err
}

func validStateMigrationConfigPath(path string) bool {
	if path == "" || strings.TrimSpace(path) != path || !filepath.IsAbs(path) {
		return false
	}
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink == 0 && info.Mode().IsRegular()
}

func writeStateMigrationDiagnostic(
	writer io.Writer,
	catalog *i18n.Catalog,
	locale string,
	code string,
	key string,
	causeCode string,
) {
	text, err := catalog.Text(locale, key)
	if err != nil {
		_, _ = io.WriteString(writer, "code=i18n_catalog_unavailable\n")
		return
	}
	if causeCode != "" {
		_, _ = io.WriteString(writer, text+" code="+code+" cause_code="+causeCode+"\n")
		return
	}
	_, _ = io.WriteString(writer, text+" code="+code+"\n")
}

func stateMigrationCauseCode(err error) string {
	for current := err; current != nil; current = errors.Unwrap(current) {
		candidate := current.Error()
		if !strings.HasPrefix(candidate, "sqlite.") || len(candidate) > 160 {
			continue
		}
		valid := true
		for _, character := range candidate {
			if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' ||
				character == '.' || character == '_' || character == '-' {
				continue
			}
			valid = false
			break
		}
		if valid {
			return candidate
		}
	}
	return ""
}
