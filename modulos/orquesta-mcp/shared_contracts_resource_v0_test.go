package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPSharedContractsDescriptorV0CatalogaResources(t *testing.T) {
	descriptor := MCPSharedContractsDescriptorV0()
	if descriptor.Name != MCPSharedContractsResourceNameV0 ||
		descriptor.Version != MCPSharedContractsResourceVersionV0 ||
		descriptor.URI != MCPSharedContractsResourceURIV0 {
		t.Fatalf("descriptor compartido inesperado: %+v", descriptor)
	}

	descriptors := MCPSharedContractResourceDescriptorsV0()
	if len(descriptors) != 10 {
		t.Fatalf("descriptores=%d, quiero catalogo + 9 contratos", len(descriptors))
	}

	seen := map[string]bool{}
	for _, item := range descriptors {
		if item.ContentType != MCPSharedContractsContentTypeV0 {
			t.Fatalf("content type inesperado en %+v", item)
		}
		if item.SummaryKey == "" || strings.Contains(item.SummaryKey, " ") {
			t.Fatalf("summary_key debe ser clave i18n compacta: %+v", item)
		}
		if seen[item.URI] {
			t.Fatalf("uri duplicada: %s", item.URI)
		}
		seen[item.URI] = true
	}

	for _, uri := range []string{
		MCPSharedContractsResourceURIV0,
		MCPNuevaAppResourceURIV0,
		"orquesta://contracts/runtime-launch-request/v0",
		"orquesta://contracts/orquesta-event/v0",
		"orquesta://contracts/operational-status-query/v0",
		"orquesta://contracts/workspace-timeline-query/v0",
		"orquesta://contracts/governance-catalog/v0",
	} {
		if !seen[uri] {
			t.Fatalf("falta descriptor uri=%s en %+v", uri, descriptors)
		}
	}
}

func TestNewMCPSharedContractsResourceV0CompactoYSinDumps(t *testing.T) {
	resource := NewMCPSharedContractsResourceV0()
	if resource.URI != MCPSharedContractsResourceURIV0 || resource.Version != "v0" {
		t.Fatalf("resource identidad: %+v", resource)
	}
	if resource.Freshness.Status != MCPResourceFreshnessStatusV0 ||
		!containsSharedContractsTestStringV0(resource.Freshness.BacklogRefs, "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md#T25-mcp-roadmap-backlog-state-sync") {
		t.Fatalf("freshness incompleta: %+v", resource.Freshness)
	}
	if len(resource.Contracts) != 9 {
		t.Fatalf("contracts=%d", len(resource.Contracts))
	}

	byContract := map[string]MCPSharedContractCompactV0{}
	for _, contract := range resource.Contracts {
		byContract[contract.Contract] = contract
		if contract.ResourceURI == "" || contract.Owner == "" || contract.Input == "" || contract.Output == "" {
			t.Fatalf("contrato incompleto: %+v", contract)
		}
		if contract.SummaryKey == "" || contract.ProgressKey == "" {
			t.Fatalf("contrato sin keys compactas: %+v", contract)
		}
		if strings.Contains(contract.SummaryKey, " ") || strings.Contains(contract.ProgressKey, " ") {
			t.Fatalf("keys deben ser message keys, no texto visible: %+v", contract)
		}
		if strings.Contains(contract.ProgressKey, "pendiente_") {
			t.Fatalf("contrato no debe duplicar pendiente_* hardcodeado: %+v", contract)
		}
		if len(contract.BacklogRefs) == 0 || len(contract.Verification) == 0 {
			t.Fatalf("contrato sin freshness por item: %+v", contract)
		}
	}

	if byContract["SolicitarNuevaApp v0"].Owner != "orquesta-factory" {
		t.Fatalf("solicitar_nueva_app: %+v", byContract["SolicitarNuevaApp v0"])
	}
	if byContract["SolicitarNuevaApp v0"].ResourceURI != MCPNuevaAppResourceURIV0 ||
		byContract["SolicitarNuevaApp v0"].Input != "AppSpecRequestV0" ||
		byContract["SolicitarNuevaApp v0"].Output != "AppSpecV0 + BacklogInicialPropuestoV0" ||
		!containsSharedContractsTestStringV0(byContract["SolicitarNuevaApp v0"].PublicErrors, "idioma_invalido") ||
		!containsSharedContractsTestStringV0(byContract["SolicitarNuevaApp v0"].Guardrails, "validacion_delegada_en_factory") {
		t.Fatalf("solicitar_nueva_app snapshot: %+v", byContract["SolicitarNuevaApp v0"])
	}
	if byContract["RuntimeLaunchRequest v0"].MCPRole != "contract_reference" {
		t.Fatalf("runtime role: %+v", byContract["RuntimeLaunchRequest v0"])
	}
	if byContract["GovernanceCatalog v0"].MCPRole != "compact_read_resource" {
		t.Fatalf("governance role: %+v", byContract["GovernanceCatalog v0"])
	}
	if byContract["GovernanceCatalog v0"].Output != "{schema_version, request_id, correlation_id, catalog_version, freshness, source_refs, effective, counters, inactive_summary, output_budget}" ||
		!containsSharedContractsTestStringV0(byContract["GovernanceCatalog v0"].PublicErrors, "governance_catalog_invalid_request") ||
		!containsSharedContractsTestStringV0(byContract["GovernanceCatalog v0"].Guardrails, "solo_effective_y_contadores") ||
		!containsSharedContractsTestStringV0(byContract["GovernanceCatalog v0"].Guardrails, "output_budget_obligatorio") {
		t.Fatalf("governance shape publico: %+v", byContract["GovernanceCatalog v0"])
	}
	if byContract["OperationalStatusQuery v0"].Owner != "orquesta-observability" ||
		byContract["OperationalStatusQuery v0"].MCPRole != "compact_read_resource" {
		t.Fatalf("operational status: %+v", byContract["OperationalStatusQuery v0"])
	}
	if byContract["WorkspaceTimelineQuery v0"].Owner != "orquesta-observability" ||
		!containsSharedContractsTestStringV0(byContract["WorkspaceTimelineQuery v0"].Guardrails, "source_ausente_not_available") {
		t.Fatalf("workspace timeline: %+v", byContract["WorkspaceTimelineQuery v0"])
	}
	if byContract["DeploymentPlan v0"].Owner != "orquesta-deploy" ||
		!containsSharedContractsTestStringV0(byContract["DeploymentPlan v0"].BacklogRefs, mcpBacklogT74RefV0) ||
		!containsSharedContractsTestStringV0(byContract["DeploymentPlan v0"].Verification, "go test -count=1 ./modulos/orquesta-deploy ./modulos/orquesta-factory ./modulos/orquesta-app-planner ./modulos/orquesta-mcp") {
		t.Fatalf("deployment plan: %+v", byContract["DeploymentPlan v0"])
	}
	if byContract["GenerarI18nDocsIniciales v0"].Owner != "orquesta-i18n-docs" ||
		!containsSharedContractsTestStringV0(byContract["GenerarI18nDocsIniciales v0"].BacklogRefs, mcpBacklogT75RefV0) ||
		!containsSharedContractsTestStringV0(byContract["GenerarI18nDocsIniciales v0"].Guardrails, "owner_activo_i18n_docs") ||
		!containsSharedContractsTestStringV0(byContract["GenerarI18nDocsIniciales v0"].Guardrails, "fallback_locale_de_owner_activo") ||
		!containsSharedContractsTestStringV0(byContract["GenerarI18nDocsIniciales v0"].Guardrails, "required_keys_hash_de_owner_activo") ||
		!containsSharedContractsTestStringV0(byContract["GenerarI18nDocsIniciales v0"].Verification, "go test -count=1 ./modulos/orquesta-i18n-docs ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp") {
		t.Fatalf("i18n docs owner: %+v", byContract["GenerarI18nDocsIniciales v0"])
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
		"tabla",
	} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
			t.Fatalf("resource contiene detalle no compacto %q: %s", forbidden, text)
		}
	}
	if len(text) > 12000 {
		t.Fatalf("resource demasiado largo: %d bytes", len(text))
	}
}

func TestMCPSharedContractResourceByNameV0NormalizaEntradas(t *testing.T) {
	cases := map[string]string{
		" RuntimeLaunchRequest v0 ":                  "orquesta://contracts/runtime-launch-request/v0",
		"runtime_launch_request":                     "orquesta://contracts/runtime-launch-request/v0",
		"orquesta://contracts/governance-catalog/v0": "orquesta://contracts/governance-catalog/v0",
		"operational_status_query":                   "orquesta://contracts/operational-status-query/v0",
		"solicitar-nueva-app":                        MCPNuevaAppResourceURIV0,
		"orquesta://contracts/solicitar-nueva-app/v0": MCPNuevaAppResourceURIV0,
	}

	for input, wantURI := range cases {
		got, ok := MCPSharedContractResourceByNameV0(input)
		if !ok {
			t.Fatalf("no encontro contrato para %q", input)
		}
		if got.ResourceURI != wantURI {
			t.Fatalf("uri para %q: got=%s want=%s", input, got.ResourceURI, wantURI)
		}
	}

	if _, ok := MCPSharedContractResourceByNameV0("contrato-inexistente"); ok {
		t.Fatalf("contrato inexistente no debe encontrarse")
	}
}

func TestToMCPSharedContractCompactV0NormalizaListas(t *testing.T) {
	got := toMCPSharedContractCompactV0(mcpSharedContractSourceV0{
		Name:    "  DemoContract ",
		Slug:    " Demo-Contract ",
		Version: "v0",
		Owner:   " owner ",
		CanonicalRefs: []string{
			" docs/contratos.md ",
			"docs/contratos.md",
			"",
		},
		PublicErrors: []string{" error_uno ", "error_uno", "error_dos"},
		Guardrails:   []string{" guard ", "guard", ""},
		Input:        " input ",
		Output:       " output ",
		BacklogRefs:  []string{" backlog-ref ", "backlog-ref"},
		Verification: []string{" go test ", "go test"},
	})

	if got.Contract != "DemoContract v0" || got.ResourceURI != "orquesta://contracts/demo-contract/v0" {
		t.Fatalf("identidad normalizada: %+v", got)
	}
	if got.Owner != "owner" || got.Input != "input" || got.Output != "output" {
		t.Fatalf("campos trim: %+v", got)
	}
	if len(got.CanonicalRefs) != 1 || got.CanonicalRefs[0] != "docs/contratos.md" {
		t.Fatalf("canonical refs: %+v", got.CanonicalRefs)
	}
	if len(got.PublicErrors) != 2 || got.PublicErrors[0] != "error_uno" || got.PublicErrors[1] != "error_dos" {
		t.Fatalf("errores publicos: %+v", got.PublicErrors)
	}
	if len(got.Guardrails) != 1 || got.Guardrails[0] != "guard" {
		t.Fatalf("guardrails: %+v", got.Guardrails)
	}
	if len(got.BacklogRefs) != 1 || got.BacklogRefs[0] != "backlog-ref" {
		t.Fatalf("backlog refs: %+v", got.BacklogRefs)
	}
	if len(got.Verification) != 1 || got.Verification[0] != "go test" {
		t.Fatalf("verification: %+v", got.Verification)
	}
}

func containsSharedContractsTestStringV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
