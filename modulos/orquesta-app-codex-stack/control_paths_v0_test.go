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
	if !reviewReworkSkipProjectDirV0(".orquesta-local-runtime-20260525") {
		t.Fatalf("runtime local no excluido de targets review/rework")
	}
	if !reviewReworkSkipProjectDirV0(".orquesta-local-state-serial-20260528") ||
		!reviewReworkSkipProjectDirV0(".orquesta-purged-20260528T001627Z") {
		t.Fatalf("artefactos locales no excluidos")
	}
	if !reviewReworkSkipProjectDirV0(".orquesta-guardian") ||
		!reviewReworkSkipProjectDirV0(".orquesta-logs") {
		t.Fatalf("artefactos guardian/logs no excluidos")
	}
	if !reviewReworkSkipProjectDirV0(".orquesta-smoke-work") {
		t.Fatalf("smoke work local no excluido de targets review/rework")
	}
}
