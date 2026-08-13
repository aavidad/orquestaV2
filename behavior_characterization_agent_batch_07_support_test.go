// Estas utilidades validan la propuesta del lote 07 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch07ExpectedEntryCount = 20

type behaviorBatch07Group struct {
	capability string
	ref        string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch07TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

func behaviorBatch07Groups() map[string]behaviorBatch07Group {
	return map[string]behaviorBatch07Group{
		"mutacion_y_outbox_coordinadas": {
			capability: "ORC-13",
			ref:        "BEHAVIOR-AGENT-BATCH-07-ATOMIC-OUTBOX",
			entryRefs: []string{
				"TASKENTRY-c514901a23a3da2d638c2ace", "TASKENTRY-52e05128d73e427d49f1c5cb",
				"TASKENTRY-b6edf657a807b33118c7d75a", "TASKENTRY-20d117c877ff0721be1e78b0",
			},
			anchors: []string{
				"Rails neutrales y outbox/replay sin producto.",
				"Fix pequeno en tasks/outbox/plan-state con test causal.",
				"integracion durable de `AskDirector`",
				"outbox-recorder. Estado: cerrado en `orquesta-director-cycle`.",
			},
		},
		"outbox_pendiente_bloquea_avance": {
			capability: "ORC-13",
			ref:        "BEHAVIOR-AGENT-BATCH-07-PENDING-BARRIER",
			entryRefs: []string{
				"TASKENTRY-818cb169791bbb8d8c9a9ec1", "TASKENTRY-a0dfed4c8d114aa2fac85baf",
				"TASKENTRY-85e02fc636ed8c121573fc31", "TASKENTRY-cec27e3ab8ef1b4f23430cda",
			},
			anchors: []string{
				"sin payloads grandes ni outbox pendiente",
				"bloqueo del siguiente tick",
				"pruebas de outbox nueva y outbox pendiente previa",
				"lista outbox pendiente por puerto",
			},
		},
		"recuperacion_outbox_sin_duplicar_efecto": {
			capability: "ORC-13",
			ref:        "BEHAVIOR-AGENT-BATCH-07-RECOVERY",
			entryRefs: []string{
				"TASKENTRY-bf42b3acd1b1f5117849fd9d", "TASKENTRY-eff89e95c1a06a75c8e3c36d",
				"TASKENTRY-2d90fce333bb7965fa16e533", "TASKENTRY-3787bd0469953effbea54d3d",
			},
			anchors: []string{
				"Reemitir outbox pendiente si se persistio el evento",
				"una ref ya reflejada no pueda reconstruir outbox con otro comando o payload",
				"Recuperar outbox `StopAgent` de leases ya expirados.",
				"Reemitir comandos idempotentes para recuperar outbox perdida.",
			},
		},
		"claim_lease_ack_correlacionados": {
			capability: "ORC-13",
			ref:        "BEHAVIOR-AGENT-BATCH-07-ACK",
			entryRefs: []string{
				"TASKENTRY-5e9c82f4eb607980d7ea0dac", "TASKENTRY-9efc46d03f95066fe25456db",
				"TASKENTRY-ac6c772fa0631485b5bed4cf", "TASKENTRY-eb8b750de8fb8253235e5e1b",
			},
			anchors: []string{
				"claim idempotente por message_id/claim_ref",
				"exclusion de message_id ya reclamado",
				"ACK missing queda pendiente",
				"InMemoryOutboxLedgerV0 entre core-workflow y dispatcher runtime fake antes del dispatch.",
			},
		},
		"puerto_ledger_y_adaptadores_separados": {
			capability: "ORC-13",
			ref:        "BEHAVIOR-AGENT-BATCH-07-PORTS",
			entryRefs: []string{
				"TASKENTRY-099a3c7a8a525f2cddf9a0d3", "TASKENTRY-23113c0112bb306e14f45471",
				"TASKENTRY-787597375a2eeb3e9ec3f3df", "TASKENTRY-4192a15526c73d90b17e0302",
			},
			anchors: []string{
				"puerto minimo de ledger save/list",
				"caso de uso/puerto reutilizable del director",
				"ledger/outbox ACK en memoria para pruebas de contrato, no productivo",
				"ledger outbox file-based para recuperacion durable de dispatch/ACK",
			},
		},
	}
}

func validateBehaviorBatch07(fixture, ledger []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch07Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch07Groups()
	expected := make(map[string]bool, behaviorBatch07ExpectedEntryCount)
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
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch07ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de los lotes 01/02/03/04/05/06: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch07ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch07HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch07RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla literal ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] ||
				!expectedRef || seen || reused || !exists || !behaviorBatch07EvidenceMatches(record.CapabilityID, evidence, entry) {
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
	if !behaviorBatch07RangesDisjoint(allEvidence) {
		return fmt.Errorf("el lote contiene rangos de procedencia solapados")
	}
	return nil
}

func decodeBehaviorBatch07Ledger(raw []byte) (map[string]behaviorBatch07TaskEntry, error) {
	result := make(map[string]behaviorBatch07TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch07TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch07EvidenceMatches(capability string, evidence behaviorBatchEvidence, entry behaviorBatch07TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" &&
		behaviorBatchEvidenceMatches(capability, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch07HeaderValid(record behaviorBatchRecord, group behaviorBatch07Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch07RecordText(record behaviorBatchRecord) string {
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

func behaviorBatch07RangesDisjoint(evidence []behaviorBatchEvidence) bool {
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
