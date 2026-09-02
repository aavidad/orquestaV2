package main

import (
	"io"

	"orquesta/internal/bootstrap"
)

func runGateA(arguments []string, stdout, stderr io.Writer) int {
	return bootstrap.RunCandidateGateA(arguments, stdout, stderr)
}
