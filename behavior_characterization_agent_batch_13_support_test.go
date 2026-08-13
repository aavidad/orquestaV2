// Estas utilidades validan la propuesta del lote 13 contra ledger, roadmap y lotes previos; nunca ejecutan legacy.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch13ExpectedEntryCount = 20

type behaviorBatch13Group struct {
	ref       string
	entryRefs []string
	anchors   []string
}

type behaviorBatch13TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	Disposition         string `json:"disposition"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch13Roadmap struct {
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

func behaviorBatch13Groups() map[string]behaviorBatch13Group {
	return map[string]behaviorBatch13Group{
		"entrada_tick_derivada_de_snapshot_y_candidatos": {
			ref: "BEHAVIOR-AGENT-BATCH-13-TICK-INPUT",
			entryRefs: []string{
				"TASKENTRY-09eb087ac1ca2b7a70615d5b", "TASKENTRY-ca774f90a1f6ff69d3d014f9",
				"TASKENTRY-1c0aef6bac49889d8b41af01", "TASKENTRY-010284d1de4ada4b70efa7f6",
			},
			anchors: []string{"misma revisión", "vista descartable", "no son evidencia independiente", "implementación V2 acreditada"},
		},
		"candidatos_validados_desde_plan_sin_inventar_refs": {
			ref: "BEHAVIOR-AGENT-BATCH-13-CANDIDATE-INTEGRITY",
			entryRefs: []string{
				"TASKENTRY-9ef18b1e45c1a33395d247f3", "TASKENTRY-7da42ad4c935cf4955cec306",
				"TASKENTRY-f3f59b83f0394d43bf922cda", "TASKENTRY-c6f7e66c710a20f02d6c42bb",
			},
			anchors: []string{"sin crear sus refs", "no son evidencia independiente", "DCA-004", "WorkItem V2"},
		},
		"ciclo_acotado_por_puertos_y_parada_en_outbox": {
			ref: "BEHAVIOR-AGENT-BATCH-13-BOUNDED-CYCLE",
			entryRefs: []string{
				"TASKENTRY-afce079de17d99656122348c", "TASKENTRY-5db40a3b74c8f47ad7653c9f",
				"TASKENTRY-2ed684473b81371b8db41384", "TASKENTRY-a702b73ef0836eddf49198d5",
			},
			anchors: []string{"un único paso", "parada explícita", "no son evidencia independiente", "único scheduler y escritor"},
		},
		"cierre_causal_exige_entrega_revision_y_evidencia": {
			ref: "BEHAVIOR-AGENT-BATCH-13-CAUSAL-CLOSURE",
			entryRefs: []string{
				"TASKENTRY-a64aad1db3d7cf073fcf0bef", "TASKENTRY-7f82e03d0073f96b7f335cec",
				"TASKENTRY-0ccb32ed0f2e7caebe432faa", "TASKENTRY-83772cc6e356674e0650438a",
			},
			anchors: []string{"único writer", "no decide si el cierre está permitido", "no son evidencia independiente", "V2 ya acredita ORC-01"},
		},
		"contrato_publicado_precede_microtarea_y_avance": {
			ref: "BEHAVIOR-AGENT-BATCH-13-DURABLE-CONTRACT-GATES",
			entryRefs: []string{
				"TASKENTRY-36eedb4bb2e7b37b82a8e689", "TASKENTRY-bf00f0aa1d5a605fed7cb272",
				"TASKENTRY-1cdf9f40c51f52a0dde19458", "TASKENTRY-5f21be3e2e3a62e2ff3103d9",
			},
			anchors: []string{"DB forense", "no son evidencia independiente", "PhaseInstance inmutable", "OutputContract"},
		},
	}
}

func validateBehaviorBatch13(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch13Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch13Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch13Groups()
	expected := make(map[string]bool, behaviorBatch13ExpectedEntryCount)
	for _, group := range groups {
		if len(group.entryRefs) != 4 || len(group.anchors) != 4 {
			return fmt.Errorf("grupo esperado incompleto: %s", group.ref)
		}
		for _, ref := range group.entryRefs {
			if _, duplicate := expected[ref]; duplicate {
				return fmt.Errorf("entrada esperada duplicada: %s", ref)
			}
			expected[ref] = false
		}
	}
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch13ExpectedEntryCount {
		return fmt.Errorf("tamaño de lote inválido: conductas=%d entradas=%d", len(records), len(expected))
	}
	previousEntries := make(map[string]struct{})
	previousRefs := make(map[string]struct{})
	for _, raw := range previous {
		prior, decodeErr := decodeBehaviorBatchRecords(raw)
		if decodeErr != nil {
			return decodeErr
		}
		for _, record := range prior {
			previousRefs[record.Ref] = struct{}{}
			for _, ref := range record.EntryRefs {
				previousEntries[ref] = struct{}{}
			}
		}
	}
	seenRefs := make(map[string]struct{}, len(records))
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch13ExpectedEntryCount)
	for _, record := range records {
		group, known := groups[record.Behavior]
		_, duplicate := seenRefs[record.Ref]
		_, previous := previousRefs[record.Ref]
		if !known || duplicate || previous || !behaviorBatch13HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch13RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			_, reused := previousEntries[evidence.EntryRef]
			seen, wanted := expected[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] ||
				!wanted || seen || reused || !exists || !behaviorBatch13EvidenceMatches(evidence, entry) {
				return fmt.Errorf("procedencia incoherente: %s", evidence.EntryRef)
			}
			expected[evidence.EntryRef] = true
			allEvidence = append(allEvidence, evidence)
		}
		seenRefs[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	for ref, seen := range expected {
		if !seen {
			return fmt.Errorf("falta %s", ref)
		}
	}
	if len(groups) != 0 || !behaviorBatch13RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch13Ledger(raw []byte) (map[string]behaviorBatch13TaskEntry, error) {
	result := make(map[string]behaviorBatch13TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch13TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch13EvidenceMatches(evidence behaviorBatchEvidence, entry behaviorBatch13TaskEntry) bool {
	return entry.CapabilityID == "ORC-01" && entry.CapabilityDecision == "accept" &&
		entry.Disposition == "accepted_pending_reimplementation" && entry.SemanticReviewState == "reviewed" &&
		behaviorBatchEvidenceMatches("ORC-01", evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch13HeaderValid(record behaviorBatchRecord, group behaviorBatch13Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == "ORC-01" &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch13RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten,
		record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects,
		record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart,
		record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch13RangesDisjoint(evidence []behaviorBatchEvidence) bool {
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

func validateBehaviorBatch13Roadmap(raw []byte) error {
	var roadmap behaviorBatch13Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	for _, capability := range roadmap.Capabilities {
		if capability.ID != "ORC-01" {
			continue
		}
		if capability.Title != "DAG de trabajo con dependencias y criterios de cierre" ||
			capability.Decision != "accept" || capability.Kind != "orchestration" ||
			capability.OwnerContext != "goal_dag_phases" || capability.Status != "accredited" ||
			len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != "AC-V05-GOAL-DAG-PHASES" ||
			len(capability.EvidenceRefs) != 3 {
			return fmt.Errorf("estado roadmap ORC-01 inesperado")
		}
		return nil
	}
	return fmt.Errorf("ORC-01 ausente del roadmap")
}
