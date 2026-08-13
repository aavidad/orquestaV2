// Estas utilidades validan el lote 23 contra ledger y roadmap; nunca ejecutan legacy.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
)

type behaviorBatch23Entry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	Disposition         string `json:"disposition"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch23Group struct {
	ref, capability string
	entries         []string
}

func behaviorBatch23Groups() map[string]behaviorBatch23Group {
	return map[string]behaviorBatch23Group{
		"actualizacion_y_rollback_externos_sin_guardian_go":             {"BEHAVIOR-AGENT-BATCH-23-EXTERNAL-UPDATE-ROLLBACK", "OPS-21", []string{"TASKENTRY-06062bd630685f0c80167a6f", "TASKENTRY-bc6e92c3f425da6b78c89f4f"}},
		"cola_global_con_prioridad_fairness_y_aging":                    {"BEHAVIOR-AGENT-BATCH-23-FAIR-GLOBAL-QUEUE", "ORC-11", []string{"TASKENTRY-0c05c408eb8009a387f8bf4e", "TASKENTRY-90dd36c636c10a02b7bf1402"}},
		"e2e_por_api_y_mcp_publicos_con_receipt":                        {"BEHAVIOR-AGENT-BATCH-23-PUBLIC-SURFACE-E2E", "EVD-09", []string{"TASKENTRY-e410686f9460cfb978f101f6", "TASKENTRY-5d58bd69789279e3c7bbed3e"}},
		"agente_logico_separado_de_binding_y_proceso":                   {"BEHAVIOR-AGENT-BATCH-23-LOGICAL-AGENT-BINDING", "GOV-21", []string{"TASKENTRY-c26ac11a19561e7012883c8b", "TASKENTRY-e9290283257865cfad28114d"}},
		"wizard_web_conversacional_sobre_contrato_publico":              {"BEHAVIOR-AGENT-BATCH-23-CONVERSATIONAL-WEB-WIZARD", "UI-05", []string{"TASKENTRY-0d138f5642336b5b74b5360b", "TASKENTRY-e426e22eaf284ba025541ce0"}},
		"i18n_con_owner_unico_fallback_es_y_codigos_estables":           {"BEHAVIOR-AGENT-BATCH-23-I18N-SINGLE-OWNER", "UI-18", []string{"TASKENTRY-09cd4b824d4d3e09a1c3354e", "TASKENTRY-65c35aeb7bc43bd4c9face6e"}},
		"revision_independiente_con_resultado_durable_antes_de_aceptar": {"BEHAVIOR-AGENT-BATCH-23-INDEPENDENT-REVIEW-GATE", "STG-14", []string{"TASKENTRY-305e82add1c00b8c9c995f04", "TASKENTRY-44d6a8087db1b438986baa48"}},
		"rotacion_opt_in_con_handoff_durable_y_epoch":                   {"BEHAVIOR-AGENT-BATCH-23-DURABLE-SESSION-HANDOFF", "ORC-22", []string{"TASKENTRY-cf7d3998cac86b7c33caa874", "TASKENTRY-7cb55b463a307112cbd8a687"}},
		"abstracciones_solo_con_requisito_y_consumidor_real":            {"BEHAVIOR-AGENT-BATCH-23-CONSUMER-BACKED-ABSTRACTIONS", "APP-15", []string{"TASKENTRY-5f2a2b52a99d395e7171b49e", "TASKENTRY-fde3fa6f37825d6b1f0d312b"}},
		"trabajo_de_dominio_neutral_por_refs_y_puerto":                  {"BEHAVIOR-AGENT-BATCH-23-NEUTRAL-DOMAIN-WORK", "EXT-01", []string{"TASKENTRY-647e4f0c85598ed40a0a6d58", "TASKENTRY-b25b1f9334106f29c944d2cf"}},
	}
}

func validateBehaviorBatch23(fixture, ledger, roadmap []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch23Ledger(ledger)
	if err != nil {
		return err
	}
	if err := validateBehaviorBatch23Roadmap(roadmap); err != nil {
		return err
	}
	groups := behaviorBatch23Groups()
	priorEntry, priorRef := map[string]bool{}, map[string]bool{}
	for _, raw := range previous {
		prior, e := decodeBehaviorBatchRecords(raw)
		if e != nil {
			return e
		}
		for _, record := range prior {
			priorRef[record.Ref] = true
			for _, ref := range record.EntryRefs {
				priorEntry[ref] = true
			}
		}
	}
	seenEntry, seenRef := map[string]bool{}, map[string]bool{}
	var evidence []behaviorBatchEvidence
	if len(records) != 10 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		if !ok || seenRef[record.Ref] || priorRef[record.Ref] || !behaviorBatchHeaderValid(record, group.capability, group.ref) || !behaviorBatchFieldsPresent(record) || len(record.EntryRefs) != 2 || len(record.Evidence) != 2 {
			return fmt.Errorf("registro inválido: %s", record.Ref)
		}
		for i, ev := range record.Evidence {
			entry, exists := entries[ev.EntryRef]
			if !exists || ev.EntryRef != group.entries[i] || record.EntryRefs[i] != group.entries[i] || seenEntry[ev.EntryRef] || priorEntry[ev.EntryRef] || entry.CapabilityDecision != "accept" || entry.SemanticReviewState != "reviewed" || (entry.Disposition != "accepted_pending_reimplementation" && entry.Disposition != "historical_superseded_by_capability") || !behaviorBatchEvidenceMatches(group.capability, ev, entry.behaviorBatchTaskEntry) {
				return fmt.Errorf("procedencia inválida: %s", ev.EntryRef)
			}
			seenEntry[ev.EntryRef] = true
			evidence = append(evidence, ev)
		}
		seenRef[record.Ref] = true
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(seenEntry) != 20 || !behaviorBatch23RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch23Ledger(raw []byte) (map[string]behaviorBatch23Entry, error) {
	result := map[string]behaviorBatch23Entry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch23Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}
func behaviorBatch23RangesDisjoint(v []behaviorBatchEvidence) bool {
	for i := range v {
		for j := i + 1; j < len(v); j++ {
			if v[i].SourceRef == v[j].SourceRef && v[i].FirstLine <= v[j].LastLine && v[j].FirstLine <= v[i].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch23Roadmap(raw []byte) error {
	type cap struct {
		ID        string   `json:"id"`
		Title     string   `json:"title"`
		Decision  string   `json:"decision"`
		Kind      string   `json:"kind"`
		Owner     string   `json:"owner_context"`
		Contracts []string `json:"acceptance_contracts"`
		Status    string   `json:"status"`
		Evidence  []string `json:"evidence_refs"`
	}
	var root struct {
		Capabilities []cap `json:"capability_entries"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	want := map[string]cap{
		"OPS-21": {Title: "Update/rollback por subcomando `orquesta` o servicio OS externo; sin Guardian Go propio", Kind: "operations", Owner: "operations_telemetry", Contracts: []string{"AC-V32-OPERATIONS-TELEMETRY"}, Status: "declared"},
		"ORC-11": {Title: "Cuotas/fairness entre proyectos y Goals", Kind: "orchestration", Owner: "budgets_effects", Contracts: []string{"AC-V15-BUDGETS-EFFECTS"}, Status: "accredited", Evidence: make([]string, 3)},
		"EVD-09": {Title: "E2E por la misma API/MCP pública del operador", Kind: "evidence", Owner: "codex_e2e", Contracts: []string{"AC-V22-CODEX-E2E"}, Status: "accredited", Evidence: make([]string, 3)},
		"GOV-21": {Title: "Agente lógico separado de provider, cuenta, credencial, proceso y sesión", Kind: "governance", Owner: "authority_rules", Contracts: []string{"AC-V02-AUTHORITY-RULES"}, Status: "accredited", Evidence: make([]string, 3)},
		"UI-05":  {Title: "Wizard web", Kind: "interface", Owner: "web_admin", Contracts: []string{"AC-V24-WEB-ADMIN"}, Status: "declared"},
		"UI-18":  {Title: "Todo texto humano de Orquesta —web, Wizard, CLI, notificaciones, errores y documentación pública— usa i18n; español es default/fallback y los códigos máquina son estables", Kind: "interface", Owner: "i18n", Contracts: []string{"AC-V21-I18N"}, Status: "accredited", Evidence: make([]string, 3)},
		"STG-14": {Title: "Revisión independiente y crítica adversarial", Kind: "stage_template", Owner: "independent_reviews", Contracts: []string{"AC-V18-INDEPENDENT-REVIEWS"}, Status: "accredited", Evidence: make([]string, 3)},
		"ORC-22": {Title: "Sesiones reanudables y handoff de contexto", Kind: "orchestration", Owner: "context_rag_evals", Contracts: []string{"AC-V27-CONTEXT-RAG-EVALS"}, Status: "declared"},
		"APP-15": {Title: "Sin capas, interfaces o servicios sin consumidor real", Kind: "generated_app_profile", Owner: "generated_apps", Contracts: []string{"AC-V33-GENERATED-APPS"}, Status: "declared"},
		"EXT-01": {Title: "Contrato genérico `DomainPlugin`/DomainWork", Kind: "extension", Owner: "domain_plugins", Contracts: []string{"AC-V28-DOMAIN-PLUGINS"}, Status: "declared"},
	}
	for _, got := range root.Capabilities {
		expected, ok := want[got.ID]
		if !ok {
			continue
		}
		if got.Title != expected.Title || got.Decision != "accept" || got.Kind != expected.Kind || got.Owner != expected.Owner || got.Status != expected.Status || len(got.Contracts) != 1 || got.Contracts[0] != expected.Contracts[0] || len(got.Evidence) != len(expected.Evidence) {
			return fmt.Errorf("roadmap drift: %s", got.ID)
		}
		delete(want, got.ID)
	}
	if len(want) != 0 {
		return fmt.Errorf("capabilities ausentes: %v", want)
	}
	return nil
}
