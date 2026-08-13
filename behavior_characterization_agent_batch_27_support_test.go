// Estas utilidades validan el lote 27 contra ledger y roadmap; el legado nunca es autoridad runtime.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch27ExpectedEntryCount = 14

type behaviorBatch27TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch27Roadmap struct {
	Capabilities []struct {
		ID                  string   `json:"id"`
		Title               string   `json:"title"`
		Decision            string   `json:"decision"`
		Kind                string   `json:"kind"`
		OwnerContext        string   `json:"owner_context"`
		Status              string   `json:"status"`
		AcceptanceContracts []string `json:"acceptance_contracts"`
		EvidenceRefs        []string `json:"evidence_refs"`
	} `json:"capability_entries"`
}

func behaviorBatch27Groups() map[string]struct {
	ref, capability, anchor string
	entries                 []string
} {
	return map[string]struct {
		ref, capability, anchor string
		entries                 []string
	}{
		"claude_adapter_mismo_protocolo_resultado_durable":     {"BEHAVIOR-AGENT-BATCH-27-CLAUDE-COMMON-PROTOCOL", "AGT-05", "lifecycle Claude", []string{"TASKENTRY-3781daf1295fb9c441d55ad4"}},
		"gemini_api_separada_de_cli_y_ruteada_por_rol":         {"BEHAVIOR-AGENT-BATCH-27-GEMINI-API-ADAPTER", "AGT-06", "API y CLI separados", []string{"TASKENTRY-5d220213c0b9edf713512e8a"}},
		"seleccion_manual_modelo_provider_esfuerzo_por_rol":    {"BEHAVIOR-AGENT-BATCH-27-MANUAL-MODEL-BY-ROLE", "AGT-11", "modelos realmente disponibles", []string{"TASKENTRY-ffca0ca921ce8f152b4828fa"}},
		"app_monolito_modular_servicios_solo_con_causa":        {"BEHAVIOR-AGENT-BATCH-27-MODULAR-MONOLITH-DEFAULT", "APP-01", "monolito modular por defecto", []string{"TASKENTRY-5d555bcf07b68e3423cd69fc"}},
		"dominio_generado_sin_infraestructura_concreta":        {"BEHAVIOR-AGENT-BATCH-27-PURE-GENERATED-DOMAIN", "APP-03", "persistencia sin policy", []string{"TASKENTRY-7bb2113423b3381bb4899bd2", "TASKENTRY-b84ccce4fec097e3bca8b279"}},
		"claves_i18n_estructuradas_locales_bcp47_y_paridad":    {"BEHAVIOR-AGENT-BATCH-27-STRUCTURED-I18N-KEYS", "APP-05", "locale BCP-47", []string{"TASKENTRY-1bbd12fc2e89875dd3160e76", "TASKENTRY-e50c4ab133430149c2343387"}},
		"ui_generada_accesible_responsive_y_completa":          {"BEHAVIOR-AGENT-BATCH-27-ACCESSIBLE-RESPONSIVE-UI", "APP-07", "foco visible", []string{"TASKENTRY-3826a25db311005fa209ad3b"}},
		"errores_tipados_logs_metricas_diagnostico_redactado":  {"BEHAVIOR-AGENT-BATCH-27-TYPED-PUBLIC-DIAGNOSTICS", "APP-08", "err.Error público", []string{"TASKENTRY-b2f8ee4bfb01cea34a01f9a6"}},
		"auth_por_principal_claims_y_riesgo_real":              {"BEHAVIOR-AGENT-BATCH-27-RISK-BASED-AUTH", "APP-09", "substring de caller", []string{"TASKENTRY-7c1340d53e54b2f0627e7a91", "TASKENTRY-f321b7c7e590d216fd2934dc"}},
		"tests_generados_semanticos_y_proporcionales_por_capa": {"BEHAVIOR-AGENT-BATCH-27-PROPORTIONAL-TEST-PYRAMID", "APP-11", "assert semántico", []string{"TASKENTRY-48b62e2c7285d845cc0cabe4", "TASKENTRY-73ae36019e35340c6daaea77"}},
	}
}

func validateBehaviorBatch27(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch27Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch27Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch27Groups()
	used := map[string]struct{}{}
	for _, raw := range previous {
		prior, decodeErr := decodeBehaviorBatchRecords(raw)
		if decodeErr != nil {
			return decodeErr
		}
		for _, record := range prior {
			for _, ref := range record.EntryRefs {
				used[ref] = struct{}{}
			}
		}
	}
	seen := map[string]struct{}{}
	evidence := make([]behaviorBatchEvidence, 0, behaviorBatch27ExpectedEntryCount)
	if len(records) != 10 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicate := seen[record.Ref]
		if !ok || duplicate || !behaviorBatch27HeaderValid(record, group.ref, group.capability) || len(record.EntryRefs) != len(group.entries) || len(record.Evidence) != len(group.entries) || !behaviorBatchFieldsPresent(record) || !strings.Contains(behaviorBatch27Text(record), group.anchor) {
			return fmt.Errorf("conducta inválida: %s", record.Ref)
		}
		for index, item := range record.Evidence {
			entry, exists := entries[item.EntryRef]
			_, reused := used[item.EntryRef]
			if item.EntryRef != group.entries[index] || record.EntryRefs[index] != item.EntryRef || reused || !exists || !behaviorBatch27EvidenceMatches(item, entry) {
				return fmt.Errorf("procedencia inválida: %s", item.EntryRef)
			}
			used[item.EntryRef] = struct{}{}
			evidence = append(evidence, item)
		}
		seen[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(evidence) != behaviorBatch27ExpectedEntryCount || !behaviorBatch27RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch27Ledger(raw []byte) (map[string]behaviorBatch27TaskEntry, error) {
	result := map[string]behaviorBatch27TaskEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch27TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch27HeaderValid(record behaviorBatchRecord, ref, capability string) bool {
	return record.SchemaVersion == 1 && record.Ref == ref && record.CapabilityID == capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch27EvidenceMatches(e behaviorBatchEvidence, entry behaviorBatch27TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, e, entry.behaviorBatchTaskEntry)
}

func behaviorBatch27Text(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch27RangesDisjoint(items []behaviorBatchEvidence) bool {
	for left := range items {
		for right := left + 1; right < len(items); right++ {
			if items[left].SourceRef == items[right].SourceRef && items[left].FirstLine <= items[right].LastLine && items[right].FirstLine <= items[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch27Roadmap(raw []byte) error {
	var roadmap behaviorBatch27Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	expected := map[string]struct{ status, owner, acceptance string }{
		"AGT-05": {"declared", "provider_adapters", "AC-V25-PROVIDER-ADAPTERS"}, "AGT-06": {"declared", "provider_adapters", "AC-V25-PROVIDER-ADAPTERS"}, "AGT-11": {"declared", "provider_adapters", "AC-V25-PROVIDER-ADAPTERS"}, "APP-01": {"declared", "generated_apps", "AC-V33-GENERATED-APPS"}, "APP-03": {"declared", "generated_apps", "AC-V33-GENERATED-APPS"}, "APP-05": {"declared", "generated_apps", "AC-V33-GENERATED-APPS"}, "APP-07": {"declared", "generated_apps", "AC-V33-GENERATED-APPS"}, "APP-08": {"declared", "generated_apps", "AC-V33-GENERATED-APPS"}, "APP-09": {"declared", "generated_apps", "AC-V33-GENERATED-APPS"}, "APP-11": {"declared", "generated_apps", "AC-V33-GENERATED-APPS"},
	}
	for _, capability := range roadmap.Capabilities {
		want, ok := expected[capability.ID]
		if !ok {
			continue
		}
		if capability.Decision != "accept" || capability.Status != want.status || capability.OwnerContext != want.owner || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != want.acceptance || (want.status == "accredited" && len(capability.EvidenceRefs) == 0) || (want.status == "declared" && len(capability.EvidenceRefs) != 0) {
			return fmt.Errorf("roadmap inesperado: %s", capability.ID)
		}
		delete(expected, capability.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("faltan capabilities: %v", expected)
	}
	return nil
}
