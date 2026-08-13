// Estas utilidades validan el lote 24 contra ledger y roadmap; el legado nunca es autoridad runtime.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch24ExpectedEntryCount = 20

type behaviorBatch24TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch24Roadmap struct {
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

func behaviorBatch24Groups() map[string]struct {
	ref, capability, anchor string
	entries                 []string
} {
	return map[string]struct {
		ref, capability, anchor string
		entries                 []string
	}{
		"app_generada_nace_con_i18n_y_fallback":                 {"BEHAVIOR-AGENT-BATCH-24-I18N-BY-DEFAULT", "APP-04", "español por defecto y fallback", []string{"TASKENTRY-66f3e416a87aa14c774d917b", "TASKENTRY-9e2d38f9b9d24e106580ce6a"}},
		"cambio_app_existente_caso_uso_unico_adaptadores_finos": {"BEHAVIOR-AGENT-BATCH-24-EXISTING-APP-CHANGE", "EXT-02", "adaptadores finos", []string{"TASKENTRY-68869e4d5fceca38e5ff7a19", "TASKENTRY-2d96d044904c9a09652aff68"}},
		"config_mutable_con_cas_escritura_atomica_y_receipt":    {"BEHAVIOR-AGENT-BATCH-24-GOVERNED-CONFIG-MUTATION", "OPS-07", "pending_restart informa sin fingir activación inmediata", []string{"TASKENTRY-e36f10f090138f48e9c7eecf", "TASKENTRY-0dc9e213e614f38a64b65d59"}},
		"subagentes_recursivos_con_linaje_y_limites":            {"BEHAVIOR-AGENT-BATCH-24-RECURSIVE-SUBAGENTS", "ORC-04", "ningún subagente mantiene Goal", []string{"TASKENTRY-0652bc4ceb67521b3e4577b2", "TASKENTRY-5f90305a5c57fd55c6823c02"}},
		"provider_modelo_score_confianza_y_muestras":            {"BEHAVIOR-AGENT-BATCH-24-PROVIDER-SCORE-CONFIDENCE", "ORC-26", "score separado de confianza", []string{"TASKENTRY-1fffa6e9c3cab50f18bdb14e", "TASKENTRY-a6b56d584283dec2601b8ef9"}},
		"wizard_pregunta_solo_por_huecos_y_contradicciones":     {"BEHAVIOR-AGENT-BATCH-24-WIZARD-GAP-QUESTIONS", "WIZ-03", "pregunta con causa", []string{"TASKENTRY-82e9b07c581d356a7fc2dc7a", "TASKENTRY-719f23be3827603697f1d8b4"}},
		"plan_inicial_revisable_antes_de_ejecutar":              {"BEHAVIOR-AGENT-BATCH-24-REVIEWABLE-INITIAL-PLAN", "WIZ-24", "proponer, confirmar y ejecutar", []string{"TASKENTRY-ccdf2c12da401982cdca592c", "TASKENTRY-45c7ea5a6446f69599cc4f69"}},
		"mutacion_selectiva_demuestra_invariante_critico":       {"BEHAVIOR-AGENT-BATCH-24-SELECTIVE-MUTATION", "EVD-10", "rojo obligatorio", []string{"TASKENTRY-3133e19d869d9f435fe4d7a6", "TASKENTRY-414ca6fd665f1a2b7985ab15"}},
		"ejecucion_agente_reemplazable_sin_cambiar_trabajo":     {"BEHAVIOR-AGENT-BATCH-24-REPLACEABLE-ATTEMPTS", "GOV-05", "WorkItem estable", []string{"TASKENTRY-89bf91bbd1a4dda7eef034b3", "TASKENTRY-da3a4a419c35d562c55ed000"}},
		"opes_temporal_por_defecto_produccion_scope_confirmado": {"BEHAVIOR-AGENT-BATCH-24-SCOPED-OPES-INSTANCE", "OPE-19", "temporal por defecto", []string{"TASKENTRY-7abae2d9e11658a612b8d7d9", "TASKENTRY-5b36da5418c85273edac47a6"}},
	}
}

func validateBehaviorBatch24(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch24Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch24Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch24Groups()
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
	evidence := make([]behaviorBatchEvidence, 0, behaviorBatch24ExpectedEntryCount)
	if len(records) != 10 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicate := seen[record.Ref]
		if !ok || duplicate || !behaviorBatch24HeaderValid(record, group.ref, group.capability) || len(record.EntryRefs) != 2 || len(record.Evidence) != 2 || !behaviorBatchFieldsPresent(record) || !strings.Contains(behaviorBatch24Text(record), group.anchor) {
			return fmt.Errorf("conducta inválida: %s", record.Ref)
		}
		for index, item := range record.Evidence {
			entry, exists := entries[item.EntryRef]
			_, reused := used[item.EntryRef]
			if item.EntryRef != group.entries[index] || record.EntryRefs[index] != item.EntryRef || reused || !exists || !behaviorBatch24EvidenceMatches(item, entry) {
				return fmt.Errorf("procedencia inválida: %s", item.EntryRef)
			}
			used[item.EntryRef] = struct{}{}
			evidence = append(evidence, item)
		}
		seen[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(evidence) != behaviorBatch24ExpectedEntryCount || !behaviorBatch24RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch24Ledger(raw []byte) (map[string]behaviorBatch24TaskEntry, error) {
	result := map[string]behaviorBatch24TaskEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch24TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch24HeaderValid(record behaviorBatchRecord, ref, capability string) bool {
	return record.SchemaVersion == 1 && record.Ref == ref && record.CapabilityID == capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch24EvidenceMatches(e behaviorBatchEvidence, entry behaviorBatch24TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, e, entry.behaviorBatchTaskEntry)
}

func behaviorBatch24Text(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch24RangesDisjoint(items []behaviorBatchEvidence) bool {
	for left := range items {
		for right := left + 1; right < len(items); right++ {
			if items[left].SourceRef == items[right].SourceRef && items[left].FirstLine <= items[right].LastLine && items[right].FirstLine <= items[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch24Roadmap(raw []byte) error {
	var roadmap behaviorBatch24Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	expected := map[string]struct{ status, owner, acceptance string }{
		"APP-04": {"declared", "generated_apps", "AC-V33-GENERATED-APPS"}, "EXT-02": {"declared", "domain_plugins", "AC-V28-DOMAIN-PLUGINS"}, "OPS-07": {"declared", "web_admin", "AC-V24-WEB-ADMIN"}, "ORC-04": {"accredited", "mailbox", "AC-V13-MAILBOX"}, "ORC-26": {"declared", "provider_adapters", "AC-V25-PROVIDER-ADAPTERS"}, "WIZ-03": {"declared", "wizard", "AC-V23-WIZARD"}, "WIZ-24": {"declared", "wizard", "AC-V23-WIZARD"}, "EVD-10": {"declared", "cutover_accreditation", "AC-V34-CUTOVER-ACCREDITATION"}, "GOV-05": {"accredited", "atomic_state_outbox", "AC-V06-ATOMIC-STATE-OUTBOX"}, "OPE-19": {"declared", "opes", "AC-V30-OPES"},
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
