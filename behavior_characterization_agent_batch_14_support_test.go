// Estas utilidades validan la propuesta del lote 14 contra trazabilidad y roadmap vigentes, nunca contra el runtime legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch14ExpectedEntryCount = 20

type behaviorBatch14Group struct {
	ref       string
	entryRefs []string
	anchors   []string
}

type behaviorBatch14TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch14Roadmap struct {
	Capabilities []struct {
		ID                  string   `json:"id"`
		Title               string   `json:"title"`
		Decision            string   `json:"decision"`
		Kind                string   `json:"kind"`
		OwnerContext        string   `json:"owner_context"`
		AcceptanceContracts []string `json:"acceptance_contracts"`
		Status              string   `json:"status"`
		EvidenceRefs        []string `json:"evidence_refs"`
	} `json:"capability_entries"`
}

func behaviorBatch14Groups() map[string]behaviorBatch14Group {
	return map[string]behaviorBatch14Group{
		"self_change_como_dominio_y_comando_ordinario": {
			ref:       "BEHAVIOR-AGENT-BATCH-14-ORDINARY-SELF-CHANGE",
			entryRefs: []string{"TASKENTRY-0b97e8cf27e312bbe5a462ff", "TASKENTRY-0ddd77fb314d8fed982dd268", "TASKENTRY-661c9e15fbab17dd2edda457", "TASKENTRY-ecc4060e3129ab1670f4b403"},
			anchors:   []string{"AutoprogrammingRequestSourceV0", "GoalWorkSpecV0 neutral", "decisiones normales", "no son evidencia independiente"},
		},
		"scanner_idle_deduplicado_y_acotado": {
			ref:       "BEHAVIOR-AGENT-BATCH-14-BOUNDED-IDLE-SCANNER",
			entryRefs: []string{"TASKENTRY-2683c8af5ca553461556da59", "TASKENTRY-313f7cc57b5cfd0f5cb0de00", "TASKENTRY-8185ad6e72dc87f9ebc1aaad", "TASKENTRY-f9abc76b23055c94c397a86a"},
			anchors:   []string{"scanner documental acotado", "presupuesto", "deduplica", "no son evidencia independiente"},
		},
		"automejora_opt_in_sujeta_a_gates_y_receipts": {
			ref:       "BEHAVIOR-AGENT-BATCH-14-GATED-SELF-CHANGE",
			entryRefs: []string{"TASKENTRY-300cec9a63a525d033b7d35f", "TASKENTRY-4c1ca64a559b9d89ba9a07aa", "TASKENTRY-560300703f8aab32be923d60", "TASKENTRY-98c9bbcca10d0c191dbec0bf"},
			anchors:   []string{"opt-in explícito", "gates", "receipt", "no son evidencia independiente"},
		},
		"hallazgos_reproducibles_a_tareas_acotadas": {
			ref:       "BEHAVIOR-AGENT-BATCH-14-SCOPED-FINDINGS",
			entryRefs: []string{"TASKENTRY-0115f8259d3abdbb4278f3fc", "TASKENTRY-0a2659bd2b9386c53e16d0aa", "TASKENTRY-4a6eee5175c94962ef6e8d0e", "TASKENTRY-c8a314e9c7739e36ddb09e4c"},
			anchors:   []string{"hallazgo reproducible", "write-set", "verificación manual", "no son evidencia independiente"},
		},
		"backlog_canonico_con_scope_epoch_y_overlap": {
			ref:       "BEHAVIOR-AGENT-BATCH-14-CANONICAL-BACKLOG",
			entryRefs: []string{"TASKENTRY-05692102fbfa60a631fc83fd", "TASKENTRY-a46383ebcdc0618d8da27905", "TASKENTRY-f9ddb94046e72f1fb348b13d", "TASKENTRY-fe1f57338473b1342db4aa1a"},
			anchors:   []string{"preflight canónico", "alias", "epoch", "no son evidencia independiente"},
		},
	}
}

func validateBehaviorBatch14(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch14Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch14Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch14Groups()
	expected := make(map[string]bool, behaviorBatch14ExpectedEntryCount)
	for _, group := range groups {
		if len(group.entryRefs) != 4 || len(group.anchors) != 4 {
			return fmt.Errorf("grupo esperado sin cuatro entradas y anclas: %s", group.ref)
		}
		for _, ref := range group.entryRefs {
			if _, duplicate := expected[ref]; duplicate {
				return fmt.Errorf("entrada esperada duplicada: %s", ref)
			}
			expected[ref] = false
		}
	}
	if len(records) != 5 || len(expected) != behaviorBatch14ExpectedEntryCount {
		return fmt.Errorf("tamaño de lote inválido: conductas=%d entradas=%d", len(records), len(expected))
	}
	previousEntryRefs := make(map[string]struct{})
	previousCharacterizationRefs := make(map[string]struct{})
	for _, raw := range previous {
		previousRecords, decodeErr := decodeBehaviorBatchRecords(raw)
		if decodeErr != nil {
			return decodeErr
		}
		for _, record := range previousRecords {
			previousCharacterizationRefs[record.Ref] = struct{}{}
			for _, ref := range record.EntryRefs {
				previousEntryRefs[ref] = struct{}{}
			}
		}
	}
	for ref := range expected {
		if _, reused := previousEntryRefs[ref]; reused {
			return fmt.Errorf("entrada reutilizada de lotes 01-11: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{})
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch14ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch14HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch14RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla literal ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] || !expectedRef || seen || reused || !exists || !behaviorBatch14EvidenceMatches(evidence, entry) {
				return fmt.Errorf("procedencia incoherente: %s", evidence.EntryRef)
			}
			expected[evidence.EntryRef] = true
			allEvidence = append(allEvidence, evidence)
		}
		seenCharacterizations[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	for ref, seen := range expected {
		if !seen {
			return fmt.Errorf("falta %s", ref)
		}
	}
	if len(groups) != 0 || !behaviorBatch14RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch14Ledger(raw []byte) (map[string]behaviorBatch14TaskEntry, error) {
	result := make(map[string]behaviorBatch14TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch14TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch14EvidenceMatches(evidence behaviorBatchEvidence, entry behaviorBatch14TaskEntry) bool {
	return entry.CapabilityID == "GOV-18" && entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches("GOV-18", evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch14HeaderValid(record behaviorBatchRecord, group behaviorBatch14Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == "GOV-18" && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch14RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch14RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := 0; left < len(evidence); left++ {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef && evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch14Roadmap(raw []byte) error {
	var roadmap behaviorBatch14Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	for _, capability := range roadmap.Capabilities {
		if capability.ID != "GOV-18" {
			continue
		}
		if capability.Title != "Autoprogramación tratada como dominio `self_change`, sin privilegios ocultos" || capability.Decision != "accept" || capability.Kind != "governance" || capability.OwnerContext != "generated_apps" || capability.Status != "declared" || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != "AC-V33-GENERATED-APPS" || len(capability.EvidenceRefs) != 0 {
			return fmt.Errorf("estado roadmap GOV-18 inesperado")
		}
		return nil
	}
	return fmt.Errorf("GOV-18 ausente del roadmap")
}
