package main

import (
	"os"
)

func main() {
	os.Exit(runMain(os.Args[1:], os.Stdout, os.Stderr))
}
