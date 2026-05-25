package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPProjectRoadmapDescriptorV0Compacto(t *testing.T) {
	descriptor := MCPProjectRoadmapDescriptorV0()

	if descriptor.Name != MCPProjectRoadmapResourceNameV0 ||
		descriptor.Version != MCPProjectRoadmapResourceVersionV0 ||
		descriptor.URI != MCPProjectRoadmapResourceURIV0 {
		t.Fatalf("descriptor identidad: %+v", descriptor)
	}
	if descriptor.ContentType != MCPProjectRoadmapContentTypeV0 {
		t.Fatalf("content type inesperado: %+v", descriptor)
	}
	if descriptor.SummaryKey == "" || strings.Contains(descriptor.SummaryKey, " ") {
		t.Fatalf("summary_key debe ser clave compacta: %+v", descriptor)
	}

	payload, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatalf("marshal descriptor: %v", err)
	}
	if len(payload) > 320 {
		t.Fatalf("descriptor demasiado largo: %d bytes", len(payload))
	}
}

func TestNewMCPProjectRoadmapResourceV0CompactoYUtilParaNucleo(t *testing.T) {
	resource := NewMCPProjectRoadmapResourceV0()
	if resource.URI != MCPProjectRoadmapResourceURIV0 ||
		resource.Version != "v0" ||
		resource.Scope != "orquesta-nucleo-reutilizable" {
		t.Fatalf("resource identidad: %+v", resource)
	}
	if resource.Freshness.Status != MCPResourceFreshnessStatusV0 ||
		!containsProjectRoadmapTestStringV0(resource.Freshness.BacklogRefs, "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md#T25-mcp-roadmap-backlog-state-sync") {
		t.Fatalf("freshness incompleta: %+v", resource.Freshness)
	}
	if len(resource.Roadmap) != 6 {
		t.Fatalf("roadmap=%d", len(resource.Roadmap))
	}
	if len(resource.Decisions) != 5 {
		t.Fatalf("decisiones=%d", len(resource.Decisions))
	}

	byID := map[string]MCPProjectRoadmapItemV0{}
	for _, item := range resource.Roadmap {
		byID[item.ID] = item
		if item.Area == "" || item.Status == "" || item.Owner == "" || item.Focus == "" {
			t.Fatalf("roadmap incompleto: %+v", item)
		}
		if item.SummaryKey == "" || item.ProgressKey == "" ||
			strings.Contains(item.SummaryKey, " ") || strings.Contains(item.ProgressKey, " ") {
			t.Fatalf("keys compactas invalidas: %+v", item)
		}
		if len(item.Contracts) == 0 || len(item.Guardrails) == 0 || len(item.CanonicalRefs) == 0 {
			t.Fatalf("roadmap sin contratos/guardrails/refs: %+v", item)
		}
		if strings.Contains(item.Status, "pendiente_") || strings.Contains(item.ProgressKey, "pendiente_") {
			t.Fatalf("roadmap no debe duplicar pendiente_* sin fuente viva: %+v", item)
		}
		if len(item.BacklogRefs) == 0 || len(item.Verification) == 0 {
			t.Fatalf("roadmap sin freshness por item: %+v", item)
		}
	}

	if byID["CORE-ROADMAP-001"].Status != "historico_compatibilidad_appspec_v0" {
		t.Fatalf("registro proyecto: %+v", byID["CORE-ROADMAP-001"])
	}
	if byID["CORE-ROADMAP-002"].Status != "vigente_workflow_task_store" ||
		!containsProjectRoadmapTestStringV0(byID["CORE-ROADMAP-002"].Contracts, "FunctionContract v0") {
		t.Fatalf("microtareas/function contract: %+v", byID["CORE-ROADMAP-002"])
	}
	if !containsProjectRoadmapTestStringV0(byID["CORE-ROADMAP-004"].Contracts, "RuntimeLaunchRequest v0") {
		t.Fatalf("runtime/capacidad: %+v", byID["CORE-ROADMAP-004"])
	}
	if byID["CORE-ROADMAP-005"].Status != "abierto_opes_derivados_cierre" ||
		!containsProjectRoadmapTestStringV0(byID["CORE-ROADMAP-005"].BacklogRefs, "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md#T18-opes-operational-closure-source") {
		t.Fatalf("domain work consumidores: %+v", byID["CORE-ROADMAP-005"])
	}
	if byID["CORE-ROADMAP-006"].Status != "vigente_dry_run_por_puerto" ||
		!containsProjectRoadmapTestStringV0(byID["CORE-ROADMAP-006"].Contracts, "DeploymentPlan v0") ||
		!containsProjectRoadmapTestStringV0(byID["CORE-ROADMAP-006"].BacklogRefs, mcpBacklogT74RefV0) {
		t.Fatalf("deployment plan: %+v", byID["CORE-ROADMAP-006"])
	}

	decisionsByID := map[string]MCPProjectDecisionCompactV0{}
	for _, decision := range resource.Decisions {
		decisionsByID[decision.ID] = decision
		if decision.Decision == "" || decision.DecisionKey == "" || decision.Motivo == "" || decision.MotivoKey == "" {
			t.Fatalf("decision incompleta: %+v", decision)
		}
		if strings.Contains(decision.DecisionKey, " ") || strings.Contains(decision.MotivoKey, " ") {
			t.Fatalf("decision keys no compactas: %+v", decision)
		}
		if len(decision.AppliesTo) == 0 || len(decision.Guardrails) == 0 {
			t.Fatalf("decision sin alcance/guardrails: %+v", decision)
		}
	}

	if !strings.Contains(decisionsByID["CORE-DEC-002"].Decision, "no persiste") {
		t.Fatalf("decision registro borrador: %+v", decisionsByID["CORE-DEC-002"])
	}
	if !containsProjectRoadmapTestStringV0(decisionsByID["CORE-DEC-005"].AppliesTo, "orquesta.project.roadmap.v0") {
		t.Fatalf("decision proyecciones compactas: %+v", decisionsByID["CORE-DEC-005"])
	}

	payload, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("marshal resource: %v", err)
	}
	text := string(payload)
	for _, forbidden := range []string{
		"request_minima_valida",
		"fixtures/",
		"POST /api",
		"sink_receipt_id",
		"completion",
		"oauth",
		"dsn",
		"/home/",
	} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
			t.Fatalf("resource contiene detalle no compacto %q: %s", forbidden, text)
		}
	}
	if len(text) > 12000 {
		t.Fatalf("resource demasiado largo: %d bytes", len(text))
	}
}

func TestMCPProjectRoadmapLookupV0NormalizaEntradas(t *testing.T) {
	itemCases := map[string]string{
		" RuntimeLaunchRequest v0 ": "CORE-ROADMAP-004",
		"runtime_launch_request":    "CORE-ROADMAP-004",
		"domain_work_consumidores":  "CORE-ROADMAP-005",
		"deployment_plan":           "CORE-ROADMAP-006",
		"DeploymentPlan v0":         "CORE-ROADMAP-006",
		"CORE_ROADMAP_002":          "CORE-ROADMAP-002",
	}

	for input, wantID := range itemCases {
		got, ok := MCPProjectRoadmapItemByKeyV0(input)
		if !ok {
			t.Fatalf("no encontro roadmap para %q", input)
		}
		if got.ID != wantID {
			t.Fatalf("roadmap para %q: got=%s want=%s", input, got.ID, wantID)
		}
	}

	decisionCases := map[string]string{
		"CORE_DEC_002": "CORE-DEC-002",
		"mcp.project.decisions.core.proyecciones_compactas.v0":      "CORE-DEC-005",
		"orquesta.project.roadmap.v0":                               "CORE-DEC-005",
		"mcp.project.decisions.core.function_contract_write_set.v0": "CORE-DEC-001",
	}

	for input, wantID := range decisionCases {
		got, ok := MCPProjectDecisionByKeyV0(input)
		if !ok {
			t.Fatalf("no encontro decision para %q", input)
		}
		if got.ID != wantID {
			t.Fatalf("decision para %q: got=%s want=%s", input, got.ID, wantID)
		}
	}

	if _, ok := MCPProjectRoadmapItemByKeyV0("roadmap-inexistente"); ok {
		t.Fatalf("roadmap inexistente no debe encontrarse")
	}
	if _, ok := MCPProjectDecisionByKeyV0("decision-inexistente"); ok {
		t.Fatalf("decision inexistente no debe encontrarse")
	}
}

func TestMCPProjectRoadmapMappersV0NormalizanListas(t *testing.T) {
	item := toMCPProjectRoadmapItemV0(mcpProjectRoadmapItemSourceV0{
		ID:          " CORE-ROADMAP-X ",
		Area:        " area ",
		Owner:       " owner ",
		Focus:       " focus ",
		SummaryKey:  " summary.key ",
		ProgressKey: " progress.key ",
		Contracts:   []string{" FunctionContract v0 ", "FunctionContract v0", ""},
		Dependencies: []string{
			" dep ",
			"dep",
		},
		Guardrails: []string{" guard ", "guard", ""},
		CanonicalRefs: []string{
			" ../CONTRATOS.md ",
			"../CONTRATOS.md",
		},
		BacklogRefs:  []string{" backlog-ref ", "backlog-ref"},
		Verification: []string{" go test ", "go test"},
	})

	if item.ID != "CORE-ROADMAP-X" || item.Status != "pendiente" || item.Area != "area" || item.Owner != "owner" {
		t.Fatalf("item trim/default: %+v", item)
	}
	if len(item.Contracts) != 1 || item.Contracts[0] != "FunctionContract v0" {
		t.Fatalf("contracts dedupe: %+v", item.Contracts)
	}
	if len(item.Dependencies) != 1 || item.Dependencies[0] != "dep" {
		t.Fatalf("dependencies dedupe: %+v", item.Dependencies)
	}
	if len(item.Guardrails) != 1 || item.Guardrails[0] != "guard" {
		t.Fatalf("guardrails dedupe: %+v", item.Guardrails)
	}
	if len(item.CanonicalRefs) != 1 || item.CanonicalRefs[0] != "../CONTRATOS.md" {
		t.Fatalf("canonical refs dedupe: %+v", item.CanonicalRefs)
	}
	if len(item.BacklogRefs) != 1 || item.BacklogRefs[0] != "backlog-ref" {
		t.Fatalf("backlog refs dedupe: %+v", item.BacklogRefs)
	}
	if len(item.Verification) != 1 || item.Verification[0] != "go test" {
		t.Fatalf("verification dedupe: %+v", item.Verification)
	}

	decision := toMCPProjectDecisionCompactV0(mcpProjectDecisionSourceV0{
		ID:          " CORE-DEC-X ",
		Decision:    " decision ",
		DecisionKey: " decision.key ",
		Motivo:      " motivo ",
		MotivoKey:   " motivo.key ",
		AppliesTo:   []string{" FunctionContract v0 ", "FunctionContract v0"},
		Guardrails:  []string{" guard ", "guard"},
		CanonicalRefs: []string{
			" ../CONTRATOS.md ",
			"../CONTRATOS.md",
		},
	})

	if decision.ID != "CORE-DEC-X" || decision.Status != "aceptada" ||
		decision.Decision != "decision" || decision.Motivo != "motivo" {
		t.Fatalf("decision trim/default: %+v", decision)
	}
	if len(decision.AppliesTo) != 1 || decision.AppliesTo[0] != "FunctionContract v0" {
		t.Fatalf("decision applies_to dedupe: %+v", decision.AppliesTo)
	}
	if len(decision.Guardrails) != 1 || decision.Guardrails[0] != "guard" {
		t.Fatalf("decision guardrails dedupe: %+v", decision.Guardrails)
	}
	if len(decision.CanonicalRefs) != 1 || decision.CanonicalRefs[0] != "../CONTRATOS.md" {
		t.Fatalf("decision canonical refs dedupe: %+v", decision.CanonicalRefs)
	}
}

func containsProjectRoadmapTestStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
