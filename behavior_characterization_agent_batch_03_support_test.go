// Estas utilidades validan la propuesta del lote 03 contra trazabilidad congelada, nunca contra el legado.
package orquesta_test

import "fmt"

const behaviorBatch03ExpectedEntryCount = 20
const behaviorBatch03ExpectedEntriesPerGroup = 4

type behaviorBatch03Group struct {
	capability string
	ref        string
	entryRefs  []string
}

func behaviorBatch03Groups() map[string]behaviorBatch03Group {
	return map[string]behaviorBatch03Group{
		"catalogo_vivo_y_gestion_de_modelos": {
			capability: "AGT-02",
			ref:        "BEHAVIOR-AGENT-BATCH-03-LIVE-MODEL-CATALOG",
			entryRefs: []string{
				"TASKENTRY-0e134e32b7e58e4891044af3", "TASKENTRY-198a334113256d91946e9e38",
				"TASKENTRY-f8e0fbd36baf3b4959c0e2ce", "TASKENTRY-7aa4321139f572605c05ee5b",
			},
		},
		"hermes_externo_api_mcp": {
			capability: "AGT-04",
			ref:        "BEHAVIOR-AGENT-BATCH-03-HERMES-EXTERNAL",
			entryRefs: []string{
				"TASKENTRY-5343c610d0a48051639ce4ac", "TASKENTRY-9b9d5dadceb5c15e68afb3e7",
				"TASKENTRY-4740e233a8b5ffa9c62fe958", "TASKENTRY-f501a2617166e19b9d2ae193",
			},
		},
		"conector_ollama_y_gestion_de_modelos": {
			capability: "AGT-07",
			ref:        "BEHAVIOR-AGENT-BATCH-03-OLLAMA-CONNECTOR",
			entryRefs: []string{
				"TASKENTRY-55e30afd659186a835e9a134", "TASKENTRY-4863de8986320442b3bca03b",
				"TASKENTRY-e2142ba2fdf6bdc5229b9c7a", "TASKENTRY-a740c8d398b4bcc88678962a",
			},
		},
		"modelos_ollama_por_perfil_de_tarea": {
			capability: "AGT-07",
			ref:        "BEHAVIOR-AGENT-BATCH-03-OLLAMA-TASK-PROFILES",
			entryRefs: []string{
				"TASKENTRY-ec568f96f74b8e9bb5c28567", "TASKENTRY-9e8ce0a2ddf30895939064b8",
				"TASKENTRY-6e76350a86a2148be804e92c", "TASKENTRY-5b4501a7dd21eb3a3aa8a4d7",
			},
		},
		"revisiones_independientes_del_mismo_candidato": {
			capability: "EVD-06",
			ref:        "BEHAVIOR-AGENT-BATCH-03-INDEPENDENT-SAME-CANDIDATE-REVIEWS",
			entryRefs: []string{
				"TASKENTRY-94dc0810ec5db110265bc17c", "TASKENTRY-3eb5af9a4549eef26dc646b1",
				"TASKENTRY-3595dcccbf32e2875428f760", "TASKENTRY-0cc1335da2bdd2c1040c6344",
			},
		},
	}
}

func validateBehaviorBatch03(fixture, ledger []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatchLedger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch03Groups()
	expected := make(map[string]bool, behaviorBatch03ExpectedEntryCount)
	for _, group := range groups {
		if len(group.entryRefs) != behaviorBatch03ExpectedEntriesPerGroup {
			return fmt.Errorf("el grupo %s contiene %d referencias; se exigen %d",
				group.ref, len(group.entryRefs), behaviorBatch03ExpectedEntriesPerGroup)
		}
		for _, ref := range group.entryRefs {
			if _, duplicate := expected[ref]; duplicate {
				return fmt.Errorf("entrada esperada duplicada: %s", ref)
			}
			expected[ref] = false
		}
	}
	if len(records) != len(groups) || len(expected) != behaviorBatch03ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de los lotes 01/02: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch03HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != behaviorBatch03ExpectedEntriesPerGroup ||
			len(record.Evidence) != behaviorBatch03ExpectedEntriesPerGroup ||
			len(record.EntryRefs) != len(group.entryRefs) ||
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

func behaviorBatch03HeaderValid(record behaviorBatchRecord, group behaviorBatch03Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref &&
		record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}
