package orquestaappcodexstack

import "testing"

func TestCodexStackWorktreeIgnorePrefixesV0IncluyeRuntimeLocalV0(t *testing.T) {
	prefixes := codexStackWorktreeIgnorePrefixesV0()
	if !codexStackStringInSetForTestV0(prefixes, ".orquesta-local-runtime") {
		t.Fatalf("prefixes=%+v", prefixes)
	}
	if !codexStackStringInSetForTestV0(prefixes, ".orquesta-server") {
		t.Fatalf("prefixes=%+v", prefixes)
	}
	if !codexStackStringInSetForTestV0(prefixes, ".orquesta-purged") {
		t.Fatalf("prefixes=%+v", prefixes)
	}
	if !codexStackStringInSetForTestV0(prefixes, "bin") ||
		!codexStackStringInSetForTestV0(prefixes, "orquesta_state") ||
		!codexStackStringInSetForTestV0(prefixes, "runtime_orquesta") {
		t.Fatalf("artefactos locales OPES no excluidos: prefixes=%+v", prefixes)
	}
	if !codexStackStringInSetForTestV0(prefixes, ".orquesta-guardian") ||
		!codexStackStringInSetForTestV0(prefixes, ".orquesta-logs") {
		t.Fatalf("prefixes=%+v", prefixes)
	}
}
