// Estas utilidades validan la propuesta del lote 10 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch10ExpectedEntryCount = 16

type behaviorBatch10Group struct {
	capability string
	ref        string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch10TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

func behaviorBatch10Groups() map[string]behaviorBatch10Group {
	return map[string]behaviorBatch10Group{
		"retry_saliente_con_presupuesto_y_deadline": {
			capability: "ORC-12",
			ref:        "BEHAVIOR-AGENT-BATCH-10-OUTBOUND-RETRY",
			entryRefs: []string{
				"TASKENTRY-01f2042a0abbe1756bb04e99", "TASKENTRY-4e7c75c9dae6ac5578fa4cf1",
				"TASKENTRY-82e7e3b85106264bbc33979b", "TASKENTRY-dd29f719efe049393d453b94",
			},
			anchors: []string{
				"RetryPolicyV0 opt-in", "non_idempotent_mutation_retry_blocked",
				"si/cuándo reintentar y cómo publicar backoff/rate",
				"cabeceras salientes de conectores de dominio",
			},
		},
		"reintento_publico_segun_clase_de_operacion": {
			capability: "ORC-12",
			ref:        "BEHAVIOR-AGENT-BATCH-10-PUBLIC-RETRY-SAFETY",
			entryRefs: []string{
				"TASKENTRY-69948a28abdc2ed175f8ae96", "TASKENTRY-6339a8863067d3d9bf66796f",
				"TASKENTRY-a22429c23b7c62952d4abe30", "TASKENTRY-523d64b37d1a5be19e202280",
			},
			anchors: []string{
				"read_only", "mutation_non_idempotent", "clasificación HTTP de rutas mutables",
				"solo se gobierna identidad pública de request/correlation/idempotency",
			},
		},
		"reconciliacion_antes_de_repetir_efecto_externo": {
			capability: "ORC-12",
			ref:        "BEHAVIOR-AGENT-BATCH-10-AMBIGUOUS-RECOVERY",
			entryRefs: []string{
				"TASKENTRY-c91e0018b7bb638739a71107", "TASKENTRY-ef49ea69d5fa65bc1dbeb86f",
				"TASKENTRY-347cde5ea9f134816bddfd63", "TASKENTRY-26b7bf47b568e6a7193e1255",
			},
			anchors: []string{
				"recovery_required", "submitted con receipt_ref", "submitted_after_timeout",
				"OutboxDeliveryLeaseV0",
			},
		},
		"presupuesto_y_reentrada_de_reparaciones": {
			capability: "ORC-12",
			ref:        "BEHAVIOR-AGENT-BATCH-10-REPAIR-BUDGET",
			entryRefs: []string{
				"TASKENTRY-80e1bc27c2df869edc6b7907", "TASKENTRY-ab6edd670db25f351bcc596f",
				"TASKENTRY-2bd81dc14dca3dcd168d292c", "TASKENTRY-4a5268304dd44b6058e65224",
			},
			anchors: []string{
				"failure_packet_hash", "guardian_repair_attempt_budget_exhausted",
				"limita el número y reentrada de intentos de repair", "72 temas y 143 registros",
			},
		},
	}
}

func validateBehaviorBatch10(fixture, ledger []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch10Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch10Groups()
	expected := make(map[string]bool, behaviorBatch10ExpectedEntryCount)
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
	if len(records) != 4 || len(groups) != 4 || len(expected) != behaviorBatch10ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de los lotes 01-07: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch10ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch10HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch10RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla literal ausente en %s: %q", record.Behavior, anchor)
			}
		}
		if !strings.Contains(text, "no son evidencia independiente") {
			return fmt.Errorf("fuentes solapadas sin calificar en %s", record.Behavior)
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] ||
				!expectedRef || seen || reused || !exists || !behaviorBatch10EvidenceMatches(record.CapabilityID, evidence, entry) {
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
	if len(groups) != 0 {
		return fmt.Errorf("faltan grupos: %v", groups)
	}
	if !behaviorBatch10RangesDisjoint(allEvidence) {
		return fmt.Errorf("el lote contiene rangos de procedencia solapados")
	}
	return nil
}

func decodeBehaviorBatch10Ledger(raw []byte) (map[string]behaviorBatch10TaskEntry, error) {
	result := make(map[string]behaviorBatch10TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch10TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch10EvidenceMatches(capability string, evidence behaviorBatchEvidence, entry behaviorBatch10TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" &&
		behaviorBatchEvidenceMatches(capability, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch10HeaderValid(record behaviorBatchRecord, group behaviorBatch10Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch10RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{
		record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten,
		record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects,
		record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart,
		record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts,
	}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch10RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := 0; left < len(evidence); left++ {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef &&
				evidence[left].FirstLine <= evidence[right].LastLine &&
				evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}
