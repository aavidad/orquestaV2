package main

import (
	"strings"
	"testing"
)

func TestServerRuntimeIdentityFromExecutableV0CalculaSHAyBuildRefV0(t *testing.T) {
	identity := serverRuntimeIdentityFromExecutableV0()

	if identity.SchemaVersion == "" ||
		identity.BinaryPathRef == "" ||
		identity.BinaryName == "" ||
		identity.BinarySHA256 == "" ||
		len(identity.BinarySHA256) != 64 ||
		identity.BuildRef == "" {
		t.Fatalf("identity insuficiente=%+v", identity)
	}
	if strings.Contains(identity.BuildRef, "/") ||
		strings.Contains(identity.BuildRef, "\\") {
		t.Fatalf("build_ref filtra path: %q", identity.BuildRef)
	}
}

func TestServerRuntimeBuildInfoCommitV0UsaCommitExplicitoDelBuildAisladoV0(t *testing.T) {
	previous := serverRuntimeBuildCommitOverrideV0
	t.Cleanup(func() { serverRuntimeBuildCommitOverrideV0 = previous })
	serverRuntimeBuildCommitOverrideV0 = strings.Repeat("a", 40)

	commit, modified := serverRuntimeBuildInfoCommitV0()
	if commit != strings.Repeat("a", 40) || modified {
		t.Fatalf("commit=%q modified=%v", commit, modified)
	}

	serverRuntimeBuildCommitOverrideV0 = "not-a-git-sha"
	if validServerRuntimeBuildCommitOverrideV0(serverRuntimeBuildCommitOverrideV0) {
		t.Fatal("override invalido aceptado")
	}
}
