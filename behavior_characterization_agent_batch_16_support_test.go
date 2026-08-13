// Estas utilidades validan la propuesta del lote 16 contra trazabilidad y roadmap vigentes, nunca contra el runtime legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch16ExpectedEntryCount = 20

type behaviorBatch16Group struct {
	ref        string
	capability string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch16TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch16Roadmap struct {
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

func behaviorBatch16Groups() map[string]behaviorBatch16Group {
	return map[string]behaviorBatch16Group{
		"tests_requeridos_atestados_sobre_el_mismo_candidato": {
			ref:        "BEHAVIOR-AGENT-BATCH-16-INDEPENDENT-REQUIRED-TESTS",
			capability: "EVD-04",
			entryRefs:  []string{"TASKENTRY-5946b8840ca9db706e194260", "TASKENTRY-53ba1e8d10a571bfa15fbb3f", "TASKENTRY-cf5df82d16d2c3ff73a21cad", "TASKENTRY-5f6ee155410eb3261278c8b7"},
			anchors:    []string{"TestAttestor independiente", "misma identidad", "go test sin tests ejecutados", "no son evidencia independiente"},
		},
		"cli_fino_sobre_casos_de_uso_publicos": {
			ref:        "BEHAVIOR-AGENT-BATCH-16-THIN-CLI",
			capability: "UI-03",
			entryRefs:  []string{"TASKENTRY-6c6d63e0479de113aef58995", "TASKENTRY-f28148a3924d3dd0ed3df245", "TASKENTRY-3633bf735b684fc319203a36", "TASKENTRY-5ab7d02016bfac2d03320695"},
			anchors:    []string{"cliente fino", "fallback local silencioso", "CliOutputEnvelopeV0", "no son evidencia independiente"},
		},
		"goal_unica_autoridad_de_lifecycle": {
			ref:        "BEHAVIOR-AGENT-BATCH-16-SINGLE-GOAL-AUTHORITY",
			capability: "GOV-03",
			entryRefs:  []string{"TASKENTRY-2d650b3796af335ef92c56cd", "TASKENTRY-3ec1320414deb7b0c30adbaa", "TASKENTRY-7c367f598ee31c61adc824a9", "TASKENTRY-3653a3c6a661654076c2e1a4"},
			anchors:    []string{"cuatro fuentes de verdad", "Goal como agregado de mando", "dual write o fallback legacy", "no son evidencia independiente"},
		},
		"shutdown_cooperativo_en_dos_fases": {
			ref:        "BEHAVIOR-AGENT-BATCH-16-TWO-PHASE-SHUTDOWN",
			capability: "OPS-16",
			entryRefs:  []string{"TASKENTRY-4f4d91925082b559a57ca32c", "TASKENTRY-feeddf13dbd7b6c08115a34a", "TASKENTRY-e974c560346955677361146f", "TASKENTRY-ecd00951d789622b36d5a4f1"},
			anchors:    []string{"exit_pending explícito", "ready igual a muerto", "cero procesos propios", "no son evidencia independiente"},
		},
		"progreso_factual_sin_porcentaje_fabricado": {
			ref:        "BEHAVIOR-AGENT-BATCH-16-FACTUAL-PROGRESS",
			capability: "ORC-23",
			entryRefs:  []string{"TASKENTRY-4a07ad5c6115f81c7fb28d55", "TASKENTRY-040f5fdbab41a7acc47d6cf0", "TASKENTRY-e6a9aab0baff8f7c7ebcb640", "TASKENTRY-f2479a4ccc28b937812d52b9"},
			anchors:    []string{"1 % por proceso vivo", "entrega distinta de cierre", "read-model sin autoridad", "no son evidencia independiente"},
		},
	}
}

func validateBehaviorBatch16(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch16Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch16Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch16Groups()
	expected := make(map[string]bool, behaviorBatch16ExpectedEntryCount)
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
	if len(records) != 5 || len(expected) != behaviorBatch16ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de lotes 01-14: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{})
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch16ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch16HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch16RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla literal ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] || !expectedRef || seen || reused || !exists || !behaviorBatch16EvidenceMatches(evidence, entry) {
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
	if len(groups) != 0 || !behaviorBatch16RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch16Ledger(raw []byte) (map[string]behaviorBatch16TaskEntry, error) {
	result := make(map[string]behaviorBatch16TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch16TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch16EvidenceMatches(evidence behaviorBatchEvidence, entry behaviorBatch16TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch16HeaderValid(record behaviorBatchRecord, group behaviorBatch16Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch16RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch16RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := 0; left < len(evidence); left++ {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef && evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch16Roadmap(raw []byte) error {
	var roadmap behaviorBatch16Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	type expectation struct {
		title, kind, owner, status, acceptance string
		evidence                               []string
	}
	expected := map[string]expectation{
		"EVD-04": {"Tests requeridos ejecutados por atestador independiente", "evidence", "test_attestor", "accredited", "AC-V17-TEST-ATTESTOR", []string{"acceptance/v17_test_attestor_test.go", "acceptance/fixtures/v17_test_attestor.json", "product/evidence/v17_test_attestor.json"}},
		"UI-03":  {"CLI fino", "interface", "web_admin", "declared", "AC-V24-WEB-ADMIN", nil},
		"GOV-03": {"`Goal` como única autoridad de identidad, generación y lifecycle", "governance", "authority_rules", "accredited", "AC-V02-AUTHORITY-RULES", []string{"acceptance/v02_authority_rules_test.go", "acceptance/fixtures/v02_authority_rules.json", "product/evidence/v02_authority_rules.json"}},
		"OPS-16": {"Shutdown cooperativo y cero procesos propios residuales", "operations", "operations_telemetry", "declared", "AC-V32-OPERATIONS-TELEMETRY", nil},
		"ORC-23": {"Progreso honesto basado en hechos, no porcentajes inventados", "orchestration", "operations_telemetry", "declared", "AC-V32-OPERATIONS-TELEMETRY", nil},
	}
	for _, capability := range roadmap.Capabilities {
		want, ok := expected[capability.ID]
		if !ok {
			continue
		}
		if capability.Title != want.title || capability.Decision != "accept" || capability.Kind != want.kind || capability.OwnerContext != want.owner || capability.Status != want.status || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != want.acceptance || !equalBehaviorBatch16Strings(capability.EvidenceRefs, want.evidence) {
			return fmt.Errorf("estado roadmap %s inesperado", capability.ID)
		}
		delete(expected, capability.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("capabilities ausentes del roadmap: %v", expected)
	}
	return nil
}

func equalBehaviorBatch16Strings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
