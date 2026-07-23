package main

import (
	"io"

	"orquesta/internal/bootstrap"
	"orquesta/internal/i18n"
)

func runCommand(arguments []string, catalog *i18n.Catalog, stdout, stderr io.Writer) int {
	return bootstrap.RunCommand(arguments, catalog, stdout, stderr)
}
