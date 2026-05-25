package orquestaappcodexstack

import "testing"

func TestCodexStackWorktreeIgnorePrefixesV0IncluyeRuntimeLocalV0(t *testing.T) {
	prefixes := codexStackWorktreeIgnorePrefixesV0()
	if !codexStackStringInSetForTestV0(prefixes, ".orquesta-local-runtime") {
		t.Fatalf("prefixes=%+v", prefixes)
	}
	if !reviewReworkSkipProjectDirV0(".orquesta-local-runtime-20260525") {
		t.Fatalf("runtime local no excluido de targets review/rework")
	}
}
