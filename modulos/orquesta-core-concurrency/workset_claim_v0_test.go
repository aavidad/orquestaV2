package orquestacoreconcurrency

import (
	"reflect"
	"testing"
)

func TestNormalizeWorksetClaimV0DeduplicaYCompactaScopesDeterminista(t *testing.T) {
	claim := WorksetClaimV0{
		ClaimRef: " claim:2 ",
		RunRef:   " run:1 ",
		TaskRef:  " task:1 ",
		ReadSet: []ScopeRefV0{
			{Ref: " modulos/orquesta-core-concurrency/a.go "},
			{Ref: "modulos/orquesta-core-concurrency"},
			{Ref: "modulos/orquesta-core-concurrency/a.go"},
			{Ref: "./docs/tareas.md"},
			{Ref: "docs/./tareas.md"},
		},
		WriteSet: []ScopeRefV0{
			{Ref: "docs/pruebas.md"},
			{Ref: "./docs/pruebas.md"},
			{Ref: "modulos/orquesta-core-concurrency/workset_claim_v0.go"},
			{Ref: "modulos/orquesta-core-concurrency"},
		},
		DependsOn:    []string{" claim:1 ", "claim:1", "claim:0"},
		EvidenceRefs: []string{" docs/pruebas.md ", "docs/pruebas.md"},
	}

	normalized, issues := NormalizeWorksetClaimV0(claim)
	if len(issues) > 0 {
		t.Fatalf("claim valido rechazado: %+v", issues)
	}

	requireScopeRefsV0(t, normalized.ReadSet, []string{
		"docs/tareas.md",
		"modulos/orquesta-core-concurrency",
	})
	requireScopeRefsV0(t, normalized.WriteSet, []string{
		"docs/pruebas.md",
		"modulos/orquesta-core-concurrency",
	})
	if normalized.SchemaVersion != WorksetClaimSchemaVersionV0 {
		t.Fatalf("schema=%q, want %q", normalized.SchemaVersion, WorksetClaimSchemaVersionV0)
	}
	if normalized.ClaimRef != "claim:2" || normalized.RunRef != "run:1" || normalized.TaskRef != "task:1" {
		t.Fatalf("refs no normalizadas: %+v", normalized)
	}
	if !reflect.DeepEqual(normalized.DependsOn, []string{"claim:0", "claim:1"}) {
		t.Fatalf("depends_on=%v", normalized.DependsOn)
	}
	if !reflect.DeepEqual(normalized.EvidenceRefs, []string{"docs/pruebas.md"}) {
		t.Fatalf("evidence_refs=%v", normalized.EvidenceRefs)
	}
}

func TestScopeRefsOverlapV0RespetaPrefijosDeSegmento(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  bool
	}{
		{name: "padre contiene hijo", left: "modulos/x", right: "modulos/x/a.go", want: true},
		{name: "mismo scope", left: "modulos/x/a.go", right: "modulos/x/a.go", want: true},
		{name: "hermanos no solapan", left: "modulos/x/a.go", right: "modulos/x/b.go", want: false},
		{name: "prefijo textual no basta", left: "modulos/x", right: "modulos/x-extra/a.go", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ScopeRefsOverlapV0(ScopeRefV0{Ref: test.left}, ScopeRefV0{Ref: test.right})
			if got != test.want {
				t.Fatalf("overlap=%v, want %v", got, test.want)
			}
		})
	}
}

func TestNormalizeWorksetClaimV0RechazaScopesInvalidos(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		code WorksetClaimIssueCodeV0
	}{
		{name: "path absoluto unix", ref: "/home/alberto/proyecto/a.go", code: ErrScopeRefAbsolutoV0},
		{name: "path absoluto windows", ref: "C:\\Users\\alberto\\proyecto\\a.go", code: ErrScopeRefAbsolutoV0},
		{name: "traversal", ref: "modulos/orquesta-core-concurrency/../orquesta-core/a.go", code: ErrScopeRefTraversalV0},
		{name: "home tilde", ref: "~/Trabajo/orquesta/a.go", code: ErrScopeRefHomeV0},
		{name: "home variable", ref: "$HOME/Trabajo/orquesta/a.go", code: ErrScopeRefHomeV0},
		{name: "scope vacio", ref: "   ", code: ErrScopeRefVacioV0},
		{name: "db reservado", ref: "db/tables/work_items", code: ErrScopeRefReservadoV0},
		{name: "provider reservado", ref: "provider/openai", code: ErrScopeRefReservadoV0},
		{name: "model reservado", ref: "model/gpt", code: ErrScopeRefReservadoV0},
		{name: "modelo reservado", ref: "modelo/gpt", code: ErrScopeRefReservadoV0},
		{name: "runtime reservado", ref: "runtime/process", code: ErrScopeRefReservadoV0},
		{name: "oauth reservado", ref: "oauth/tokens", code: ErrScopeRefReservadoV0},
		{name: "secreto reservado", ref: "config/secret", code: ErrScopeRefReservadoV0},
		{name: "secretos reservado", ref: "config/secretos", code: ErrScopeRefReservadoV0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			claim := baseValidWorksetClaimV0()
			claim.WriteSet = []ScopeRefV0{{Ref: test.ref}}
			_, issues := NormalizeWorksetClaimV0(claim)
			requireIssueCodeV0(t, issues, test.code)
		})
	}
}

func TestNormalizeWorksetClaimV0AceptaRutasRealesConRuntimeEnNombre(t *testing.T) {
	tests := []string{
		"modulos/orquesta-runtime-codex/README.md",
		"modulos/orquesta-runtime-codex-delivery/source_v0.go",
	}

	for _, ref := range tests {
		t.Run(ref, func(t *testing.T) {
			claim := baseValidWorksetClaimV0()
			claim.WriteSet = []ScopeRefV0{{Ref: ref}}

			normalized, issues := NormalizeWorksetClaimV0(claim)
			if len(issues) > 0 {
				t.Fatalf("ruta real rechazada: %+v", issues)
			}
			requireScopeRefsV0(t, normalized.WriteSet, []string{ref})
		})
	}
}

func TestValidateWorksetClaimV0RechazaWriteSetVacio(t *testing.T) {
	claim := baseValidWorksetClaimV0()
	claim.WriteSet = nil

	issues := ValidateWorksetClaimV0(claim)
	requireIssueCodeV0(t, issues, ErrWorksetClaimWriteSetVacioV0)
}

func baseValidWorksetClaimV0() WorksetClaimV0 {
	return WorksetClaimV0{
		SchemaVersion: WorksetClaimSchemaVersionV0,
		ClaimRef:      "claim:1",
		RunRef:        "run:1",
		TaskRef:       "task:1",
		WriteSet:      []ScopeRefV0{{Ref: "modulos/orquesta-core-concurrency/workset_claim_v0.go"}},
	}
}

func requireScopeRefsV0(t *testing.T, refs []ScopeRefV0, want []string) {
	t.Helper()
	got := make([]string, 0, len(refs))
	for _, ref := range refs {
		got = append(got, ref.Ref)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("refs=%v, want %v", got, want)
	}
}

func requireIssueCodeV0(t *testing.T, issues []WorksetClaimIssueV0, code WorksetClaimIssueCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issues=%+v no contienen %q", issues, code)
}
