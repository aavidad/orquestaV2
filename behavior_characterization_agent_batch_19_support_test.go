// Estas utilidades validan el lote 19 contra trazabilidad y roadmap vigentes, nunca contra el runtime legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch19ExpectedEntryCount = 20

type behaviorBatch19Group struct {
	ref, capability string
	entryRefs       []string
	anchors         []string
}

type behaviorBatch19TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch19Roadmap struct {
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

type behaviorBatch19RoadmapWant struct {
	title, kind, owner, acceptance, status string
	evidence                               []string
}

func behaviorBatch19Groups() map[string]behaviorBatch19Group {
	return map[string]behaviorBatch19Group{
		"telemetria_estructurada_compacta_y_sin_autoridad": {
			ref: "BEHAVIOR-AGENT-BATCH-19-STRUCTURED-TELEMETRY", capability: "OPS-19",
			entryRefs: []string{"TASKENTRY-c4a3e317aa9e91719acc1565", "TASKENTRY-a439baaf076e482e622eeab3", "TASKENTRY-1d524896bd4c8cba5e8b4059", "TASKENTRY-ccf6716aed2868287e9b922b"},
			anchors:   []string{"OrquestaEventV0", "payload compacto", "read-only", "no son evidencia independiente"},
		},
		"paralelismo_derivado_de_claims_disjuntos": {
			ref: "BEHAVIOR-AGENT-BATCH-19-DISJOINT-WORKSETS", capability: "ORC-02",
			entryRefs: []string{"TASKENTRY-70c9f54147714b67d2860b51", "TASKENTRY-795111af1b2251c70a2ad667", "TASKENTRY-3a6a742465e0cb716b931502", "TASKENTRY-a23ee2b56eb913de89ddf6af"},
			anchors:   []string{"WorksetClaimV0", "DetectWorksetConflictsV0", "EvaluateParallelGroupsV0", "no son evidencia independiente"},
		},
		"appspec_validada_versionada_antes_del_goal": {
			ref: "BEHAVIOR-AGENT-BATCH-19-IMMUTABLE-APPSPEC", capability: "WIZ-07",
			entryRefs: []string{"TASKENTRY-0f37a5489a2fe0386a993bc6", "TASKENTRY-791d5b018b32cade2b988887", "TASKENTRY-f11464630f4783f7e9035210", "TASKENTRY-0c7e9eea92faf845b6114679"},
			anchors:   []string{"AppSpecRequestV0", "versionable", "SolicitarNuevaAppV0", "no son evidencia independiente"},
		},
		"consejo_secuencial_con_quorum_veto_y_evidencia": {
			ref: "BEHAVIOR-AGENT-BATCH-19-EVIDENCED-COUNCIL", capability: "GOV-13",
			entryRefs: []string{"TASKENTRY-262cd78580af396e9752269c", "TASKENTRY-26c1d4a19ff7f474888ab476", "TASKENTRY-e58c86da14792f23290eceab", "TASKENTRY-4aae53dcb414f8d6cff1f9a6"},
			anchors:   []string{"propuesta, crítica y voto", "quorum", "veto de seguridad", "no son evidencia independiente"},
		},
		"reentrada_causal_sin_repetir_terminales": {
			ref: "BEHAVIOR-AGENT-BATCH-19-RESTART-RECONCILIATION", capability: "ORC-17",
			entryRefs: []string{"TASKENTRY-949621dc3161e71771116823", "TASKENTRY-99441851fd09789df82b7b7e", "TASKENTRY-1968aa4644e91db676dc447c", "TASKENTRY-64fe9644dad398aca720c686"},
			anchors:   []string{"foto reentrable", "CapacityDecided", "ProcessRegistry", "no son evidencia independiente"},
		},
	}
}

func validateBehaviorBatch19(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch19Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch19Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch19Groups()
	expected := make(map[string]bool, behaviorBatch19ExpectedEntryCount)
	for _, group := range groups {
		if len(group.entryRefs) != 4 || len(group.anchors) != 4 {
			return fmt.Errorf("grupo incompleto: %s", group.ref)
		}
		for _, ref := range group.entryRefs {
			if _, duplicate := expected[ref]; duplicate {
				return fmt.Errorf("entrada esperada duplicada: %s", ref)
			}
			expected[ref] = false
		}
	}
	if len(records) != 5 || len(expected) != behaviorBatch19ExpectedEntryCount {
		return fmt.Errorf("tamaño de lote inválido: conductas=%d entradas=%d", len(records), len(expected))
	}
	previousEntries := make(map[string]struct{})
	previousBehaviors := make(map[string]struct{})
	for _, raw := range previous {
		prior, decodeErr := decodeBehaviorBatchRecords(raw)
		if decodeErr != nil {
			return decodeErr
		}
		for _, record := range prior {
			previousBehaviors[record.Ref] = struct{}{}
			for _, ref := range record.EntryRefs {
				previousEntries[ref] = struct{}{}
			}
		}
	}
	for ref := range expected {
		if _, reused := previousEntries[ref]; reused {
			return fmt.Errorf("entrada reutilizada de lotes anteriores: %s", ref)
		}
	}
	seenBehaviors := make(map[string]struct{})
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch19ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicate := seenBehaviors[record.Ref]
		_, previousDuplicate := previousBehaviors[record.Ref]
		if !ok || duplicate || previousDuplicate || !behaviorBatch19HeaderValid(record, group) || len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro inválido: %s", record.Ref)
		}
		text := behaviorBatch19RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			seen, wanted := expected[evidence.EntryRef]
			_, reused := previousEntries[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] || !wanted || seen || reused || !exists || !behaviorBatch19EvidenceMatches(group.capability, evidence, entry) {
				return fmt.Errorf("procedencia incoherente: %s", evidence.EntryRef)
			}
			expected[evidence.EntryRef] = true
			allEvidence = append(allEvidence, evidence)
		}
		seenBehaviors[record.Ref] = struct{}{}
		delete(groups, record.Behavior)
	}
	for ref, seen := range expected {
		if !seen {
			return fmt.Errorf("falta %s", ref)
		}
	}
	if len(groups) != 0 || !behaviorBatch19RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch19Ledger(raw []byte) (map[string]behaviorBatch19TaskEntry, error) {
	result := make(map[string]behaviorBatch19TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch19TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch19EvidenceMatches(capability string, evidence behaviorBatchEvidence, entry behaviorBatch19TaskEntry) bool {
	return entry.CapabilityID == capability && entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(capability, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch19HeaderValid(record behaviorBatchRecord, group behaviorBatch19Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch19RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	for _, group := range [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts} {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch19RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := range evidence {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef && evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch19Roadmap(raw []byte) error {
	var roadmap behaviorBatch19Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	wants := map[string]behaviorBatch19RoadmapWant{
		"OPS-19": {"Métricas, logs estructurados y trazas", "operations", "operations_telemetry", "AC-V32-OPERATIONS-TELEMETRY", "declared", nil},
		"ORC-02": {"Paralelización por write-sets disjuntos", "orchestration", "goal_dag_phases", "AC-V05-GOAL-DAG-PHASES", "accredited", []string{"acceptance/v05_goal_dag_phases_test.go", "acceptance/fixtures/v05_goal_dag_phases.json", "product/evidence/v05_goal_dag_phases.json"}},
		"WIZ-07": {"`AppSpec` tipada, completa, versionada e inmutable al arrancar Goal", "intake", "wizard", "AC-V23-WIZARD", "declared", nil},
		"GOV-13": {"Propuesta, crítica, voto, veto de seguridad y decisión con receipts", "governance", "council", "AC-V19-COUNCIL", "accredited", []string{"acceptance/v19_council_test.go", "acceptance/fixtures/v19_council.json", "product/evidence/v19_council.json"}},
		"ORC-17": {"Recuperación tras restart sin reejecutar terminales", "orchestration", "atomic_state_outbox", "AC-V06-ATOMIC-STATE-OUTBOX", "accredited", []string{"acceptance/v06_atomic_state_outbox_test.go", "acceptance/fixtures/v06_atomic_state_outbox.json", "product/evidence/v06_atomic_state_outbox.json"}},
	}
	for _, capability := range roadmap.Capabilities {
		want, exists := wants[capability.ID]
		if !exists {
			continue
		}
		if capability.Title != want.title || capability.Decision != "accept" || capability.Kind != want.kind || capability.OwnerContext != want.owner || capability.Status != want.status || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != want.acceptance || !behaviorBatch19StringsEqual(capability.EvidenceRefs, want.evidence) {
			return fmt.Errorf("estado roadmap %s inesperado", capability.ID)
		}
		delete(wants, capability.ID)
	}
	if len(wants) != 0 {
		return fmt.Errorf("capacidades ausentes: %v", wants)
	}
	return nil
}

func behaviorBatch19StringsEqual(left, right []string) bool {
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
