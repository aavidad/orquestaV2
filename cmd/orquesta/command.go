package main

import (
	"io"

	"orquesta/internal/bootstrap"
)

const commandUsage = bootstrap.CommandUsage

func runCommand(arguments []string, stdout, stderr io.Writer) int {
	return bootstrap.RunCommand(arguments, stdout, stderr)
}
