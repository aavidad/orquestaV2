// Estas utilidades validan la propuesta del lote 08 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch08ExpectedEntryCount = 20

type behaviorBatch08Group struct {
	capability string
	ref        string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch08TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

func behaviorBatch08Groups() map[string]behaviorBatch08Group {
	return map[string]behaviorBatch08Group{
		"materializacion_dag_y_dependencias_validas": {
			capability: "ORC-01",
			ref:        "BEHAVIOR-AGENT-BATCH-08-DAG-MATERIALIZATION",
			entryRefs: []string{
				"TASKENTRY-73299fa04483a25a8f27571e", "TASKENTRY-9878dcb491cef69d3c9504c4",
				"TASKENTRY-02eff140fed95884ea82992d", "TASKENTRY-ed48f08ebfd763173572f6a1",
			},
			anchors: []string{
				"materializar tareas durables del plan antes de programacion.",
				"missing, self-dependency, ciclos y dependencias no cerradas.",
				"Definir DTOs puros y validadores para work items/microtareas",
				"transicion durable `CreateMicrotask`",
			},
		},
		"frontera_lista_y_tick_determinista": {
			capability: "ORC-01",
			ref:        "BEHAVIOR-AGENT-BATCH-08-READY-FRONTIER",
			entryRefs: []string{
				"TASKENTRY-8ed4916eb6e3ebd8408b1551", "TASKENTRY-4f01587a9c8bed79999a08bc",
				"TASKENTRY-8152a702d69d58bc184f013b", "TASKENTRY-e788fbf8d4f44b7b052c2e82",
			},
			anchors: []string{
				"frontera viva del run",
				"BuildDirectorSchedulerTick v0",
				"tick puro minimo para programacion con capacidad, gate y agente.",
				"contrato `ExecuteDirectorCycleStepV0`",
			},
		},
		"cierre_causal_por_entrega_revision_y_evidencia": {
			capability: "ORC-01",
			ref:        "BEHAVIOR-AGENT-BATCH-08-CAUSAL-CLOSURE",
			entryRefs: []string{
				"TASKENTRY-d184bcae775ec14ff044df27", "TASKENTRY-2d0b19361de9a01b64d11402",
				"TASKENTRY-9fe474974a948d7b0523ce3d", "TASKENTRY-5f47cec4f514c5068c6a900c",
			},
			anchors: []string{
				"Autocerrar tareas con evidencia de entrega valida",
				"Delivery→review→closure causal",
				"`CloseTask` -> `TaskClosed`",
				"`CloseRun` -> `RunClosed`",
			},
		},
		"estado_durable_replay_y_bloqueos": {
			capability: "ORC-01",
			ref:        "BEHAVIOR-AGENT-BATCH-08-DURABLE-REPLAY",
			entryRefs: []string{
				"TASKENTRY-53194088e184ecadaa71b7b5", "TASKENTRY-f64ba2620ece63a6b48de7b6",
				"TASKENTRY-4f8548b1978e207cd88b933a", "TASKENTRY-a08aacc9ecc8ff31e412d9e4",
			},
			anchors: []string{
				"nucleo durable",
				"RunStarted`, `PhaseOpened` y `RunBlocked",
				"`ApplyEventV0`",
				"Resolver blockers durables exactos",
			},
		},
		"avance_residente_sin_cierre_por_quiescencia": {
			capability: "ORC-01",
			ref:        "BEHAVIOR-AGENT-BATCH-08-RESIDENT-PROGRESSION",
			entryRefs: []string{
				"TASKENTRY-bd5919d019b339d57bbf1a4a", "TASKENTRY-504853adeea53518229b2d09",
				"TASKENTRY-cffd1a92ae81f470f03e4872", "TASKENTRY-a33775ddf65490b9b4e4485a",
			},
			anchors: []string{
				"autonomía residente en autoprogramación",
				"frontera de tareas autoprogramming",
				"`ExecuteDirectorCycleStepsV0` usa `max_steps`",
				"El runner es un tick",
			},
		},
	}
}

func validateBehaviorBatch08(fixture, ledger []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch08Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch08Groups()
	expected := make(map[string]bool, behaviorBatch08ExpectedEntryCount)
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
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch08ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de los lotes 01/02/03/04/05/06/07: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch08ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch08HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch08RecordText(record)
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
				!expectedRef || seen || reused || !exists || !behaviorBatch08EvidenceMatches(record.CapabilityID, evidence, entry) {
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
	if !behaviorBatch08RangesDisjoint(allEvidence) {
		return fmt.Errorf("el lote contiene rangos de procedencia solapados")
	}
	return nil
}

func decodeBehaviorBatch08Ledger(raw []byte) (map[string]behaviorBatch08TaskEntry, error) {
	result := make(map[string]behaviorBatch08TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch08TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch08EvidenceMatches(capability string, evidence behaviorBatchEvidence, entry behaviorBatch08TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" &&
		behaviorBatchEvidenceMatches(capability, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch08HeaderValid(record behaviorBatchRecord, group behaviorBatch08Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch08RecordText(record behaviorBatchRecord) string {
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

func behaviorBatch08RangesDisjoint(evidence []behaviorBatchEvidence) bool {
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
