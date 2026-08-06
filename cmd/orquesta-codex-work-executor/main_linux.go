//go:build linux

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	executor "orquesta/internal/adapters/executor/codexwork"
	protocol "orquesta/internal/agentprotocol/codexwork"
)

const codeArgumentsInvalid = "codexwork_executor.arguments_invalid"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, arguments []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(arguments) != 0 {
		writeCode(stderr, codeArgumentsInvalid)
		return 2
	}
	if err := executor.NewSealed().Run(ctx, stdin, stdout); err != nil {
		writeCode(stderr, safeCode(err))
		return 1
	}
	return 0
}

func safeCode(err error) string {
	if code := protocol.ErrorCode(err); code != "" {
		return string(code)
	}
	if code := executor.ErrorCode(err); code != "" {
		return string(code)
	}
	return string(executor.CodeCommandIO)
}

func writeCode(output io.Writer, code string) {
	_, _ = fmt.Fprintln(output, code)
}
