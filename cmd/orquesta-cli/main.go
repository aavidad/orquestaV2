package main

import (
	"context"
	"os"

	orquestacli "orquesta/modulos/orquesta-cli"
)

func main() {
	code := orquestacli.RunOrquestaCLIV0(context.Background(), os.Args[1:], orquestacli.OrquestaCLIRunnerV0{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	os.Exit(code)
}
