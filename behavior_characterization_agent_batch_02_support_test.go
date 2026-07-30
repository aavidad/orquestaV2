// Estas utilidades validan la propuesta del lote 02 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import "fmt"

const behaviorBatch02ExpectedEntryCount = 20

type behaviorBatch02Group struct {
	capability string
	ref        string
	entryRefs  []string
}

func behaviorBatch02Groups() map[string]behaviorBatch02Group {
	return map[string]behaviorBatch02Group{
		"lanzamiento_neutral_con_contexto_resuelto": {
			capability: "AGT-01",
			ref:        "BEHAVIOR-AGENT-BATCH-02-NEUTRAL-LAUNCH",
			entryRefs: []string{
				"TASKENTRY-d1e54d14ccd52a1241c38235", "TASKENTRY-b775431b02ac42e6e6abe7ae",
				"TASKENTRY-b4a8648fae48a6c8bdd8f28c", "TASKENTRY-2278966876db001fec5515d0",
			},
		},
		"ciclo_goal_codex_lanzar_y_observar": {
			capability: "AGT-03",
			ref:        "BEHAVIOR-AGENT-BATCH-02-CODEX-GOAL",
			entryRefs: []string{
				"TASKENTRY-5185fbbdacdab248290eb08e", "TASKENTRY-2af116b44b208bdc73691748",
				"TASKENTRY-3fe356bf2d6c6a5bdd5baab7", "TASKENTRY-3fc32614b0ab3dd24e1a3e70",
			},
		},
		"supervision_y_recuperacion_codex": {
			capability: "AGT-03",
			ref:        "BEHAVIOR-AGENT-BATCH-02-CODEX-OBSERVATION",
			entryRefs: []string{
				"TASKENTRY-e70d8cacb9a74e0e83cfc5b7", "TASKENTRY-1dcd75ee0123362e5f7ad0ad",
				"TASKENTRY-9723454927b483dcc318bbd1", "TASKENTRY-46563efd9a3b573845351c6c",
			},
		},
		"parada_exacta_solicitada_y_confirmada": {
			capability: "ORC-16",
			ref:        "BEHAVIOR-AGENT-BATCH-02-EXACT-STOP",
			entryRefs: []string{
				"TASKENTRY-360b3fa23e9e1baae25cf807", "TASKENTRY-03c5c84d3410e22d813c9dc5",
				"TASKENTRY-6b861e0755c08b3994fbd20a", "TASKENTRY-8b259501ef01080ba30e4d43",
			},
		},
		"aislamiento_concurrente_y_recuperacion_de_parada": {
			capability: "ORC-16",
			ref:        "BEHAVIOR-AGENT-BATCH-02-CONCURRENT-STOP",
			entryRefs: []string{
				"TASKENTRY-1c94fc2afd72f537564317ca", "TASKENTRY-bc6d50166277d2cbf6b53340",
				"TASKENTRY-441f7629294dd33e46c7fb7d", "TASKENTRY-eb7b345234e1acbbeb21b0e1",
			},
		},
	}
}

func validateBehaviorBatch02(fixture, ledger, previous []byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatchLedger(ledger)
	if err != nil {
		return err
	}
	previousRecords, err := decodeBehaviorBatchRecords(previous)
	if err != nil {
		return err
	}
	groups := behaviorBatch02Groups()
	expected := make(map[string]bool, behaviorBatch02ExpectedEntryCount)
	for _, group := range groups {
		for _, ref := range group.entryRefs {
			if _, duplicate := expected[ref]; duplicate {
				return fmt.Errorf("entrada esperada duplicada: %s", ref)
			}
			expected[ref] = false
		}
	}
	if len(records) != len(groups) || len(expected) != behaviorBatch02ExpectedEntryCount {
		return fmt.Errorf("tamaño de lote inválido: conductas=%d entradas=%d",
			len(records), len(expected))
	}
	previousRefs := make(map[string]struct{})
	for _, record := range previousRecords {
		for _, ref := range record.EntryRefs {
			previousRefs[ref] = struct{}{}
		}
	}
	for ref := range expected {
		if _, reused := previousRefs[ref]; reused {
			return fmt.Errorf("entrada reutilizada del lote 01: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateRef := seenCharacterizations[record.Ref]
		if !ok || duplicateRef || !behaviorBatch02HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != len(group.entryRefs) || len(record.Evidence) != len(group.entryRefs) ||
			!behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			_, reused := previousRefs[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] ||
				evidence.EntryRef != group.entryRefs[index] || !expectedRef || seen || reused ||
				!exists || !behaviorBatchEvidenceMatches(record.CapabilityID, evidence, entry) {
				return fmt.Errorf("procedencia incoherente: %s", evidence.EntryRef)
			}
			expected[evidence.EntryRef] = true
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
	return nil
}

func behaviorBatch02HeaderValid(record behaviorBatchRecord, group behaviorBatch02Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref &&
		record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}
