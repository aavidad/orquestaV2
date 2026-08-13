// Estas utilidades validan el lote 36 contra ledger y roadmap; el legado nunca es autoridad runtime.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch36ExpectedEntryCount = 5

type behaviorBatch36TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch36Roadmap struct {
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

func behaviorBatch36Groups() map[string]struct {
	ref, capability, anchor string
	entries                 []string
} {
	return map[string]struct {
		ref, capability, anchor string
		entries                 []string
	}{
		"diagramas_derivados_base_textual_y_verificables":             {"BEHAVIOR-AGENT-BATCH-36-VERIFIABLE-DIAGRAMS", "WIZ-22", "imagen como autoridad", []string{"TASKENTRY-05f1a2ddfa0d59aa05aa0f16"}},
		"confirmacion_explicita_ligada_dossier_ref_e_invalidable":     {"BEHAVIOR-AGENT-BATCH-36-DOSSIER-CONFIRMATION", "WIZ-23", "booleano global", []string{"TASKENTRY-d3e702147903f16cde983fda", "TASKENTRY-dfe35f43993d879a19f1e2de"}},
		"packs_dominio_combinables_preguntas_integraciones_sin_motor": {"BEHAVIOR-AGENT-BATCH-36-COMBINABLE-DOMAIN-PACKS", "WIZ-25", "if por dominio en motor", []string{"TASKENTRY-a18be4dff536ec323acbea87", "TASKENTRY-5d58c430b5ba7372c3a42809"}},
	}
}

func validateBehaviorBatch36(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch36Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch36Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch36Groups()
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
	evidence := make([]behaviorBatchEvidence, 0, behaviorBatch36ExpectedEntryCount)
	if len(records) != 3 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicate := seen[record.Ref]
		if !ok || duplicate || !behaviorBatch36HeaderValid(record, group.ref, group.capability) || len(record.EntryRefs) != len(group.entries) || len(record.Evidence) != len(group.entries) || !behaviorBatchFieldsPresent(record) || !strings.Contains(behaviorBatch36Text(record), group.anchor) {
			return fmt.Errorf("conducta inválida: %s", record.Ref)
		}
		for index, item := range record.Evidence {
			entry, exists := entries[item.EntryRef]
			_, reused := used[item.EntryRef]
			if item.EntryRef != group.entries[index] || record.EntryRefs[index] != item.EntryRef || reused || !exists || !behaviorBatch36EvidenceMatches(item, entry) {
				return fmt.Errorf("procedencia inválida: %s", item.EntryRef)
			}
			used[item.EntryRef] = struct{}{}
			evidence = append(evidence, item)
		}
		seen[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	if len(groups) != 0 || len(evidence) != behaviorBatch36ExpectedEntryCount || !behaviorBatch36RangesDisjoint(evidence) {
		return fmt.Errorf("cobertura o rangos inválidos")
	}
	return nil
}

func decodeBehaviorBatch36Ledger(raw []byte) (map[string]behaviorBatch36TaskEntry, error) {
	result := map[string]behaviorBatch36TaskEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch36TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch36HeaderValid(record behaviorBatchRecord, ref, capability string) bool {
	return record.SchemaVersion == 1 && record.Ref == ref && record.CapabilityID == capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch36EvidenceMatches(e behaviorBatchEvidence, entry behaviorBatch36TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, e, entry.behaviorBatchTaskEntry)
}

func behaviorBatch36Text(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch36RangesDisjoint(items []behaviorBatchEvidence) bool {
	for left := range items {
		for right := left + 1; right < len(items); right++ {
			if items[left].SourceRef == items[right].SourceRef && items[left].FirstLine <= items[right].LastLine && items[right].FirstLine <= items[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch36Roadmap(raw []byte) error {
	var roadmap behaviorBatch36Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	expected := map[string]struct{ decision, status, owner, acceptance string }{
		"WIZ-22": {"accept", "declared", "wizard", "AC-V23-WIZARD"}, "WIZ-23": {"accept", "declared", "wizard", "AC-V23-WIZARD"}, "WIZ-25": {"accept", "declared", "wizard", "AC-V23-WIZARD"},
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
