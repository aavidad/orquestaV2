package bootstrap

import (
	"os"
	"testing"

	"orquesta/internal/adapters/agent/codex"
)

func TestMain(main *testing.M) {
	if codex.IsLocalSupervisorInvocation(os.Args[1:]) {
		os.Exit(codex.RunLocalSupervisor())
	}
	os.Exit(main.Run())
}
