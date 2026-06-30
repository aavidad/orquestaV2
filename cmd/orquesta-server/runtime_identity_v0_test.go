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
