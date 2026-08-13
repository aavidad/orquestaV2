// Estas utilidades validan el lote 34 contra ledger y roadmap; el legado nunca es autoridad runtime.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch34ExpectedEntryCount = 13

type behaviorBatch34TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch34Roadmap struct {
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

func behaviorBatch34Groups() map[string]struct {
	ref, capability, anchor string
	entries                 []string
} {
	return map[string]struct {
		ref, capability, anchor string
		entries                 []string
	}{
		"consola_rescate_explicita_sin_autoridad_paralela":        {"BEHAVIOR-AGENT-BATCH-34-RESCUE-CONSOLE", "UI-17", "--local por defecto", []string{"TASKENTRY-0bbf07b210eb89b317757e80", "TASKENTRY-2d7ce562b889e42320ad6442"}},
		"wizard_distingue_nueva_app_de_cambio_existente":          {"BEHAVIOR-AGENT-BATCH-34-NEW-OR-EXISTING-APP", "WIZ-01", "validator como efecto", []string{"TASKENTRY-b178c26008354b5f2ba85464", "TASKENTRY-40dc634370ba07f9abe14af4"}},
		"wizard_rico_tematico_acotado_seis_rondas":                {"BEHAVIOR-AGENT-BATCH-34-RICH-BOUNDED-WIZARD", "WIZ-02", "seis rondas por defecto", []string{"TASKENTRY-a70e5be004c3860862e43766", "TASKENTRY-03595f5bfc55f6c09c2d3a19"}},
		"preview_plan_presupuesto_riesgos_efectos_sin_ejecutar":   {"BEHAVIOR-AGENT-BATCH-34-PLAN-RISK-EFFECT-PREVIEW", "WIZ-08", "preview que muta", []string{"TASKENTRY-a2a0ecd0f0c2bc85576b6afe", "TASKENTRY-2ddda661a9a47e986e9389db"}},
		"plantillas_app_y_perfiles_calidad_defaults_revisables":   {"BEHAVIOR-AGENT-BATCH-34-APP-TEMPLATES-QUALITY", "WIZ-11", "default explicado", []string{"TASKENTRY-ca90b61e7c3008feaff219b1"}},
		"a2ui_declarativa_renderiza_sin_decidir_dominio":          {"BEHAVIOR-AGENT-BATCH-34-DECLARATIVE-A2UI", "WIZ-12", "A2UI que escribe estado", []string{"TASKENTRY-7efc78c0a89a8be0ad241ab7"}},
		"chat_formulario_vistas_mismo_dossier_con_cas":            {"BEHAVIOR-AGENT-BATCH-34-CHAT-FORM-SHARED-STATE", "WIZ-15", "last write wins", []string{"TASKENTRY-a85784e6b2abb87255cb7a78", "TASKENTRY-428d8884316118bb633f243b"}},
		"slot_filling_texto_libre_con_procedencia_y_confirmacion": {"BEHAVIOR-AGENT-BATCH-34-SLOT-FILLING", "WIZ-16", "procedencia de span", []string{"TASKENTRY-b6571e33775d1fe2620d6fa3"}},
	}
}

func validateBehaviorBatch34(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch34Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch34Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch34Groups()
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
	evidence := make([]behaviorBatchEvidence, 0, behaviorBatch34ExpectedEntryCount)
	if len(records) != 8 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicate := seen[record.Ref]
		if !ok || duplicate || !behaviorBatch34HeaderValid(record, group.ref, group.capability) || len(record.EntryRefs) != len(group.entries) || len(record.Evidence) != len(group.entries) || !behaviorBatchFieldsPresent(record) || !strings.Contains(behaviorBatch34Text(record), group.anchor) {
			return fmt.Errorf("conducta inválida: %s", record.Ref)
		}
		for index, item := range record.Evidence {
			entry, exists := entries[item.EntryRef]
			_, reused := used[item.EntryRef]
			if item.EntryRef != group.entries[index] || record.EntryRefs[index] != item.EntryRef || reused || !exists || !behaviorBatch34EvidenceMatches(item, entry) {
				return fmt.Errorf("procedencia inválida: %s", item.EntryRef)
			}
			used[item.EntryRef] = struct{}{}
			evidence = append(evidence, item)
		}
		seen[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(evidence) != behaviorBatch34ExpectedEntryCount || !behaviorBatch34RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch34Ledger(raw []byte) (map[string]behaviorBatch34TaskEntry, error) {
	result := map[string]behaviorBatch34TaskEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch34TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch34HeaderValid(record behaviorBatchRecord, ref, capability string) bool {
	return record.SchemaVersion == 1 && record.Ref == ref && record.CapabilityID == capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch34EvidenceMatches(e behaviorBatchEvidence, entry behaviorBatch34TaskEntry) bool {
	decisionValid := entry.CapabilityDecision == "accept" || (entry.CapabilityID == "WIZ-12" && entry.CapabilityDecision == "reject")
	return decisionValid && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, e, entry.behaviorBatchTaskEntry)
}

func behaviorBatch34Text(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch34RangesDisjoint(items []behaviorBatchEvidence) bool {
	for left := range items {
		for right := left + 1; right < len(items); right++ {
			if items[left].SourceRef == items[right].SourceRef && items[left].FirstLine <= items[right].LastLine && items[right].FirstLine <= items[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch34Roadmap(raw []byte) error {
	var roadmap behaviorBatch34Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	expected := map[string]struct{ decision, status, owner, acceptance string }{
		"UI-17": {"accept", "declared", "web_admin", "AC-V24-WEB-ADMIN"}, "WIZ-01": {"accept", "declared", "wizard", "AC-V23-WIZARD"}, "WIZ-02": {"accept", "declared", "wizard", "AC-V23-WIZARD"}, "WIZ-08": {"accept", "declared", "wizard", "AC-V23-WIZARD"}, "WIZ-11": {"accept", "declared", "wizard", "AC-V23-WIZARD"}, "WIZ-12": {"reject", "declared", "wizard", "AC-V23-WIZARD"}, "WIZ-15": {"accept", "declared", "wizard", "AC-V23-WIZARD"}, "WIZ-16": {"accept", "declared", "wizard", "AC-V23-WIZARD"},
	}
	for _, capability := range roadmap.Capabilities {
		want, ok := expected[capability.ID]
		if !ok {
			continue
		}
		if capability.Decision != want.decision || capability.Status != want.status || capability.OwnerContext != want.owner || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != want.acceptance || len(capability.EvidenceRefs) != 0 {
			return fmt.Errorf("roadmap inesperado: %s", capability.ID)
		}
		delete(expected, capability.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("faltan capabilities: %v", expected)
	}
	return nil
}
