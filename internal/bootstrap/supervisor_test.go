package bootstrap

import (
	"os"
	"testing"
)

func TestMain(main *testing.M) {
	if exitCode, handled := DispatchPrivateInvocation(os.Args[1:]); handled {
		os.Exit(exitCode)
	}
	os.Exit(main.Run())
}

func TestPrivateInvocationMatcherIsExact(t *testing.T) {
	if !isPrivateInvocation([]string{"__orquesta_internal_codex_supervisor_v1"}) ||
		isPrivateInvocation(nil) ||
		isPrivateInvocation([]string{"__orquesta_internal_codex_supervisor_v1", "extra"}) {
		t.Fatal("private supervisor dispatch matcher is not exact")
	}
}
