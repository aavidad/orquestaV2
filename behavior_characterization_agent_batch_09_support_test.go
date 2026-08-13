// Estas utilidades validan la propuesta del lote 09 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch09ExpectedEntryCount = 20

type behaviorBatch09Group struct {
	capability string
	ref        string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch09TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

func behaviorBatch09Groups() map[string]behaviorBatch09Group {
	return map[string]behaviorBatch09Group{
		"microtareas_acotadas_por_contrato": {
			capability: "ORC-03",
			ref:        "BEHAVIOR-AGENT-BATCH-09-BOUNDED-MICROTASKS",
			entryRefs: []string{
				"TASKENTRY-81f803b80141658d0be397df", "TASKENTRY-4aa3b55b97a2dc97a96d1b06",
				"TASKENTRY-941b676ec06d8c61dd7e96eb", "TASKENTRY-d8002ecf51f23e9f721df779",
			},
			anchors: []string{
				"Emitir microtareas desde Orquesta sobre especificaciones de funcion",
				"dividir una solicitud Go API + web en microtareas pequenas.",
				"sin microfragmentar contenido",
				"hay que partirlo antes de programar.",
			},
		},
		"propuesta_y_decision_de_replan_causales": {
			capability: "ORC-03",
			ref:        "BEHAVIOR-AGENT-BATCH-09-CAUSAL-REPLAN",
			entryRefs: []string{
				"TASKENTRY-463520e080642be9396a930b", "TASKENTRY-f5a86d4c86f06f3012b423db",
				"TASKENTRY-80905ac4bbe58dfc22a3a3a3", "TASKENTRY-46a54f15e5f8eee201b2d0df",
			},
			anchors: []string{
				"DTO puro `ReplanProposalV0`",
				"convertir el cambio aceptado en replanificacion completa",
				"solo abre decision durable trazada a `failed_agents`",
				"Conflictos de write-set, replan y estado vivo post-ola.",
			},
		},
		"senales_de_review_y_fallo_no_mutan_por_si_solas": {
			capability: "ORC-03",
			ref:        "BEHAVIOR-AGENT-BATCH-09-REPLAN-SIGNALS",
			entryRefs: []string{
				"TASKENTRY-71eacc81fc85a18be82de462", "TASKENTRY-42bf6acd271bba5c51f08167",
				"TASKENTRY-f88402075b44628ffe4b6653", "TASKENTRY-0cf56c50ad0f0d826f3d83a9",
			},
			anchors: []string{
				"traducir `ReviewResultV0 changes_requested/rejected` a `ReplanProposalV0`",
				"No crea tareas, no pide capacidad, no relanza agentes",
				"sin inventar replan ni relanzar agentes",
				"Condición sospechosa de yield de progreso a replan",
			},
		},
		"split_materializado_desde_followups_explicitos": {
			capability: "ORC-03",
			ref:        "BEHAVIOR-AGENT-BATCH-09-EXPLICIT-SPLIT",
			entryRefs: []string{
				"TASKENTRY-a38d0b88586c627dcb8c9189", "TASKENTRY-0b3ba01670e5a1d6eacfc5d5",
				"TASKENTRY-625c88028eff95eeae097765", "TASKENTRY-ebaea274838f1874809d5ff0",
			},
			anchors: []string{
				"split_task",
				"`CreateMicrotask`",
				"ReplanFollowupCandidates explícitos",
				"followup explícito de `split_task`",
			},
		},
		"replan_acotado_sin_churn": {
			capability: "ORC-03",
			ref:        "BEHAVIOR-AGENT-BATCH-09-BOUNDED-REPLAN",
			entryRefs: []string{
				"TASKENTRY-83419b8ef61c43097b441bec", "TASKENTRY-5b11e363ae4ee8baf9a3b5ba",
				"TASKENTRY-4e4c85dc33b7316497862704", "TASKENTRY-e127aa4068df82061ddb7c0d",
			},
			anchors: []string{
				"Cablear el guard de límite de replan en el scheduler",
				"Guardas o tests focales de plan, alias, fanout, presupuesto y repair policy.",
				"`DomainWorkJobRequestV0` deduplicados",
				"No bloquear replan causal con progress candidates ya parados.",
			},
		},
	}
}

func validateBehaviorBatch09(fixture, ledger []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch09Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch09Groups()
	expected := make(map[string]bool, behaviorBatch09ExpectedEntryCount)
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
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch09ExpectedEntryCount {
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
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch09ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch09HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch09RecordText(record)
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
				!expectedRef || seen || reused || !exists || !behaviorBatch09EvidenceMatches(record.CapabilityID, evidence, entry) {
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
	if !behaviorBatch09RangesDisjoint(allEvidence) {
		return fmt.Errorf("el lote contiene rangos de procedencia solapados")
	}
	return nil
}

func decodeBehaviorBatch09Ledger(raw []byte) (map[string]behaviorBatch09TaskEntry, error) {
	result := make(map[string]behaviorBatch09TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch09TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch09EvidenceMatches(capability string, evidence behaviorBatchEvidence, entry behaviorBatch09TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" &&
		behaviorBatchEvidenceMatches(capability, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch09HeaderValid(record behaviorBatchRecord, group behaviorBatch09Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch09RecordText(record behaviorBatchRecord) string {
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

func behaviorBatch09RangesDisjoint(evidence []behaviorBatchEvidence) bool {
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
