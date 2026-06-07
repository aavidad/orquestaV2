package orquestacontext

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildContextBundleV0ProgramacionPequenoPorRefs(t *testing.T) {
	bundle := BuildContextBundleV0(validContextBundleRequestV0())
	if !bundle.Valid() {
		t.Fatalf("bundle invalido: %+v", bundle.Issues)
	}
	requireContextEntryV0(t, bundle, ContextLayerCommonRulesV0, "orquesta-common-rules:v0")
	requireContextEntryV0(t, bundle, ContextLayerModuleContextV0, "modulos/orquesta-runtime/AGENTS.md")
	requireContextEntryV0(t, bundle, ContextLayerModuleContextV0, "modulos/orquesta-runtime/docs/contratos.md")
	requireContextEntryV0(t, bundle, ContextLayerPhaseContextV0, "modulos/orquesta-runtime/docs/pruebas.md")
	requireContextEntryV0(t, bundle, ContextLayerTaskContextV0, "process_runtime_connector_v0.go")
	requireContextEntryV0(t, bundle, ContextLayerContractContextV0, "ProcessRuntimeConnectorV0")
	if len(bundle.Entries) > 14 {
		t.Fatalf("contexto demasiado grande: entries=%d", len(bundle.Entries))
	}
	assertContextBundleJSONSafeV0(t, bundle)
}

func TestBuildContextBundleV0BrainstormingIncluyeDecisiones(t *testing.T) {
	request := validContextBundleRequestV0()
	request.TargetModule = "orquesta-director"
	request.Phase = "brainstorming_arquitectura"
	request.TaskKind = "arquitectura"
	request.CapacityLevel = "xhigh"
	request.WriteSet = []string{"docs/decisiones.md"}
	request.ContractRefs = []string{"ContextBundleV0"}

	bundle := BuildContextBundleV0(request)
	if !bundle.Valid() {
		t.Fatalf("bundle invalido: %+v", bundle.Issues)
	}
	requireContextEntryV0(t, bundle, ContextLayerPhaseContextV0, "modulos/orquesta-director/docs/decisiones.md")
	if containsContextSourceV0(bundle, "modulos/orquesta-director/docs/pruebas.md") {
		t.Fatalf("brainstorming no debe cargar pruebas por defecto: %+v", bundle.Entries)
	}
}

func TestBuildContextBundleV0CrossModuleQuedaComoContratoYPreguntaDirector(t *testing.T) {
	request := validContextBundleRequestV0()
	request.CrossModuleRefs = []string{"orquesta-capacity:CapacityDecisionV0"}

	bundle := BuildContextBundleV0(request)
	if !bundle.Valid() {
		t.Fatalf("bundle invalido: %+v", bundle.Issues)
	}
	requireContextEntryV0(t, bundle, ContextLayerContractContextV0, "orquesta-capacity:CapacityDecisionV0")
	if !strings.Contains(bundle.Summary.DirectorQuestionPolicy, "CONSULTA_AL_DIRECTOR") {
		t.Fatalf("politica de consulta ausente: %+v", bundle.Summary)
	}
}

func TestBuildContextBundleV0RailsOfflineNoBloqueaDetalleOperativo(t *testing.T) {
	request := validContextBundleRequestV0()
	request.ReadSet = []string{"/home/user/.codex/auth.json"}

	bundle := BuildContextBundleV0(request)
	if !bundle.Valid() {
		t.Fatalf("rails offline no debe bloquear bundle: %+v", bundle.Issues)
	}
}

func TestBuildContextBundleV0RechazaExcesoEntradas(t *testing.T) {
	request := validContextBundleRequestV0()
	request.MaxEntries = 8

	bundle := BuildContextBundleV0(request)
	requireContextIssueV0(t, bundle.Issues, ErrContextBundleTamanoInvalidoV0)
}

func validContextBundleRequestV0() ContextBundleRequestV0 {
	return ContextBundleRequestV0{
		SchemaVersion: ContextBundleRequestSchemaVersionV0,
		BundleRef:     "context-bundle-runtime-001",
		WorkOrderRef:  "work-order-runtime-001",
		TargetModule:  "orquesta-runtime",
		Phase:         "programacion",
		TaskKind:      "microtarea_codigo",
		Objective:     "Validar el conector de proceso real con parada selectiva.",
		CapacityLevel: "high",
		ReadSet:       []string{"process_runtime_connector_types_v0.go"},
		WriteSet:      []string{"process_runtime_connector_v0.go", "process_runtime_connector_v0_test.go"},
		ContractRefs:  []string{"ProcessRuntimeConnectorV0"},
		EvidenceRefs:  []string{"evidence-runtime-race-ok"},
	}
}

func requireContextEntryV0(t *testing.T, bundle ContextBundleV0, layer string, source string) {
	t.Helper()
	for _, entry := range bundle.Entries {
		if entry.Layer == layer && entry.SourceRef == source {
			return
		}
	}
	t.Fatalf("entry layer=%q source=%q no encontrada en %+v", layer, source, bundle.Entries)
}

func containsContextSourceV0(bundle ContextBundleV0, source string) bool {
	for _, entry := range bundle.Entries {
		if entry.SourceRef == source {
			return true
		}
	}
	return false
}

func requireContextIssueV0(t *testing.T, issues []ContextBundleIssueV0, code ContextBundleIssueCodeV0) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q no encontrada en %+v", code, issues)
}

func assertContextBundleJSONSafeV0(t *testing.T, bundle ContextBundleV0) {
	t.Helper()
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	lower := strings.ToLower(string(data))
	for _, forbidden := range []string{"oauth", "token", "secret", "transcript", "provider", "sqlite", "postgres", "mysql", "/home/"} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("bundle contiene detalle prohibido %q: %s", forbidden, string(data))
		}
	}
}
