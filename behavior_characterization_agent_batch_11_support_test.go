// Estas utilidades validan la propuesta del lote 11 contra trazabilidad y roadmap vigentes, nunca contra el runtime legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch11ExpectedEntryCount = 20

type behaviorBatch11Group struct {
	ref       string
	entryRefs []string
	anchors   []string
}

type behaviorBatch11TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch11Roadmap struct {
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

func behaviorBatch11Groups() map[string]behaviorBatch11Group {
	return map[string]behaviorBatch11Group{
		"presupuesto_ejecutable_y_ratchet_descendente": {
			ref: "BEHAVIOR-AGENT-BATCH-11-EXECUTABLE-BUDGET",
			entryRefs: []string{
				"TASKENTRY-76c5cb177737ae2b21014d61", "TASKENTRY-099a59fcac3213831a95cd5f",
				"TASKENTRY-a3b152e2bf01baba65036219", "TASKENTRY-af440cf93453bedb9ee10242",
			},
			anchors: []string{"snapshot o worktree real", "ratchet descendente", "TestResidualGoFileBudgetT90V0", "preflight APP-16"},
		},
		"fachadas_de_composicion_con_owners_acotados": {
			ref: "BEHAVIOR-AGENT-BATCH-11-THIN-COMPOSITION",
			entryRefs: []string{
				"TASKENTRY-5f27732a30e5eef74a0c075c", "TASKENTRY-925fb66a8123af8e1c71b10b",
				"TASKENTRY-a1bdab1ef71d2d54f09188cc", "TASKENTRY-54ad122bfdb71e5fc7dade25",
			},
			anchors: []string{"owners pequeños por responsabilidad", "puertos neutrales", "ningún lifecycle paralelo", "no son evidencia independiente"},
		},
		"adaptadores_acotados_y_contratos_neutrales": {
			ref: "BEHAVIOR-AGENT-BATCH-11-THIN-ADAPTERS",
			entryRefs: []string{
				"TASKENTRY-8be1b770ffc11de97f1101f7", "TASKENTRY-616fe4b180d52e24152683c2",
				"TASKENTRY-84e900823f072ac44c551803", "TASKENTRY-a653769735ba2792d27b845c",
			},
			anchors: []string{"memory, file y SQL", "shell ni HOME implícitos", "configuración global", "no son evidencia independiente"},
		},
		"loops_hubs_y_superficies_como_fachadas_finas": {
			ref: "BEHAVIOR-AGENT-BATCH-11-THIN-FACADES",
			entryRefs: []string{
				"TASKENTRY-19ce16e255527c36967dc72d", "TASKENTRY-09a41cb853eac4402f7e7db6",
				"TASKENTRY-521cc7c8c29bb0b8ed47b718", "TASKENTRY-d04536b45e1090c7732c85e2",
			},
			anchors: []string{"otro scheduler", "fachada de 74 líneas", "volvió a concentrar numerosos helpers", "no son evidencia independiente"},
		},
		"shards_y_providers_con_presupuesto_local": {
			ref: "BEHAVIOR-AGENT-BATCH-11-BOUNDED-PROVIDERS",
			entryRefs: []string{
				"TASKENTRY-76d440b6353430391672824e", "TASKENTRY-93761a23a94519c908b654bf",
				"TASKENTRY-6824843ff1a41d8ee9fdda07", "TASKENTRY-a7c0e0a0345a9c9160253af9",
			},
			anchors: []string{"shards append-only", "provider de replan dividido", "ninguna autoridad derivada de texto", "no son evidencia independiente"},
		},
	}
}

func validateBehaviorBatch11(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch11Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch11Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch11Groups()
	expected := make(map[string]bool, behaviorBatch11ExpectedEntryCount)
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
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch11ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de los lotes 01-10: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch11ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch11HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch11RecordText(record)
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
				!expectedRef || seen || reused || !exists || !behaviorBatch11EvidenceMatches(evidence, entry) {
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
	if len(groups) != 0 || !behaviorBatch11RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch11Ledger(raw []byte) (map[string]behaviorBatch11TaskEntry, error) {
	result := make(map[string]behaviorBatch11TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch11TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch11EvidenceMatches(evidence behaviorBatchEvidence, entry behaviorBatch11TaskEntry) bool {
	return entry.CapabilityID == "APP-16" && entry.CapabilityDecision == "accept" &&
		entry.SemanticReviewState == "reviewed" &&
		behaviorBatchEvidenceMatches("APP-16", evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch11HeaderValid(record behaviorBatchRecord, group behaviorBatch11Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == "APP-16" &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch11RecordText(record behaviorBatchRecord) string {
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

func behaviorBatch11RangesDisjoint(evidence []behaviorBatchEvidence) bool {
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

func validateBehaviorBatch11Roadmap(raw []byte) error {
	var roadmap behaviorBatch11Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	for _, capability := range roadmap.Capabilities {
		if capability.ID != "APP-16" {
			continue
		}
		if capability.Title != "Presupuesto de tamaño y complejidad por aplicación" ||
			capability.Decision != "accept" || capability.Kind != "generated_app_profile" ||
			capability.OwnerContext != "generated_apps" || capability.Status != "declared" ||
			len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != "AC-V33-GENERATED-APPS" ||
			len(capability.EvidenceRefs) != 0 {
			return fmt.Errorf("estado roadmap APP-16 inesperado")
		}
		return nil
	}
	return fmt.Errorf("APP-16 ausente del roadmap")
}
