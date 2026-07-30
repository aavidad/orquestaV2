// Estas utilidades validan la propuesta del lote 04 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import "fmt"

const behaviorBatch04ExpectedEntryCount = 20

type behaviorBatch04Group struct {
	capability string
	ref        string
	entryRefs  []string
}

func behaviorBatch04Groups() map[string]behaviorBatch04Group {
	return map[string]behaviorBatch04Group{
		"protocolo_comun_de_dispatch": {
			capability: "AGT-01",
			ref:        "BEHAVIOR-AGENT-BATCH-04-COMMON-DISPATCH",
			entryRefs: []string{
				"TASKENTRY-a45cf00438c4e53899e03cc2", "TASKENTRY-8a7fca6246893b62e19bf5f4",
				"TASKENTRY-1009a6671c11cc3a7d330737", "TASKENTRY-0c822c9eda56463ecc352346",
			},
		},
		"peticion_neutral_de_runtime": {
			capability: "AGT-01",
			ref:        "BEHAVIOR-AGENT-BATCH-04-NEUTRAL-RUNTIME-REQUEST",
			entryRefs: []string{
				"TASKENTRY-cdb97bbde780b922b42f7511", "TASKENTRY-a07560757406b8ee9aed01ce",
				"TASKENTRY-048f69806a440f72936bb12a", "TASKENTRY-f0f546a9ef7b43d4a7a4a172",
			},
		},
		"solicitud_durable_a_launch_validado": {
			capability: "AGT-01",
			ref:        "BEHAVIOR-AGENT-BATCH-04-DURABLE-TO-VALIDATED-LAUNCH",
			entryRefs: []string{
				"TASKENTRY-8bf521c288cc4b6fb2549bc2", "TASKENTRY-14ea639b0e47a1ef643f2b69",
				"TASKENTRY-38f1cc26afb65fb16e196812", "TASKENTRY-1ea764410f9a5a17f4f9462d",
			},
		},
		"runtime_de_proceso_controlado_y_aislado": {
			capability: "AGT-01",
			ref:        "BEHAVIOR-AGENT-BATCH-04-CONTROLLED-PROCESS",
			entryRefs: []string{
				"TASKENTRY-26e1ead2098adb95820b2b77", "TASKENTRY-39b2ee319fe5b2895fb87f70",
				"TASKENTRY-35216e34707831db6a4a5806", "TASKENTRY-eb70f820708296a7a48acffc",
			},
		},
		"adaptador_externo_opt_in": {
			capability: "AGT-01",
			ref:        "BEHAVIOR-AGENT-BATCH-04-EXTERNAL-OPT-IN",
			entryRefs: []string{
				"TASKENTRY-0721cd7e43eed090a6dcf2cb", "TASKENTRY-da140253154051394392a5a3",
				"TASKENTRY-2015fd1ee26b31e2f6590703", "TASKENTRY-e1e1e62e4e8b78e8d24a4a6d",
			},
		},
	}
}

func validateBehaviorBatch04(fixture, ledger []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatchLedger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch04Groups()
	expected := make(map[string]bool, behaviorBatch04ExpectedEntryCount)
	for _, group := range groups {
		if len(group.entryRefs) != 4 {
			return fmt.Errorf("grupo esperado sin cuatro entradas: %s", group.ref)
		}
		for _, ref := range group.entryRefs {
			if _, duplicate := expected[ref]; duplicate {
				return fmt.Errorf("entrada esperada duplicada: %s", ref)
			}
			expected[ref] = false
		}
	}
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch04ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de los lotes 01/02/03: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch04HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 ||
			!behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
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

func behaviorBatch04HeaderValid(record behaviorBatchRecord, group behaviorBatch04Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref &&
		record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}
