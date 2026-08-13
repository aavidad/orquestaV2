// Estas utilidades validan la propuesta del lote 12 contra trazabilidad y roadmap vigentes, nunca contra el runtime legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch12ExpectedEntryCount = 20

type behaviorBatch12Group struct {
	ref       string
	entryRefs []string
	anchors   []string
}

type behaviorBatch12TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch12Roadmap struct {
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

func behaviorBatch12Groups() map[string]behaviorBatch12Group {
	return map[string]behaviorBatch12Group{
		"reserva_causal_antes_de_lanzar_o_fusionar": {
			ref: "BEHAVIOR-AGENT-BATCH-12-CLAIM-BEFORE-LAUNCH",
			entryRefs: []string{
				"TASKENTRY-ad9fc8e67198a91192637027", "TASKENTRY-fc0468f9c08cafe7cac31ce2",
				"TASKENTRY-3db9f8eae151580ccf3117be", "TASKENTRY-23735ef0b20089c4da56c110",
			},
			anchors: []string{"ranking como exclusión mutua", "backlogTaskIDAllocationV0", "API durable completa", "no son evidencia independiente"},
		},
		"refs_deterministas_sin_colision_silenciosa": {
			ref: "BEHAVIOR-AGENT-BATCH-12-COLLISION-SAFE-REFS",
			entryRefs: []string{
				"TASKENTRY-49360dba860862d34a80c823", "TASKENTRY-da8ecdc3590d6c4e64c2bf6e",
				"TASKENTRY-209ba2587771bf163540c73c", "TASKENTRY-a32d31d438d2fc7fd5449e96",
			},
			anchors: []string{"UnixNano como identidad", "deterministicRefDigestPrefixV0", "BuildDomainWorkJobIdentityV0", "no son evidencia independiente"},
		},
		"progreso_expiracion_y_guardian_con_lease_vivo": {
			ref: "BEHAVIOR-AGENT-BATCH-12-LEASE-EXPIRY",
			entryRefs: []string{
				"TASKENTRY-9568b0c76dbc7f3dc366e76f", "TASKENTRY-96c145a4caa67dc71bc53271",
				"TASKENTRY-6de1599b04b7b6f762583c30", "TASKENTRY-b98fb0ec43e4b1fc7d2347f9",
			},
			anchors: []string{"AgentProgressLeaseBridgeV0", "guardian_promotion_lease_lost", "no ejecuta stop ni replan automáticamente", "no son evidencia independiente"},
		},
		"replay_exacto_rechaza_ref_con_payload_distinto": {
			ref: "BEHAVIOR-AGENT-BATCH-12-PAYLOAD-AWARE-REPLAY",
			entryRefs: []string{
				"TASKENTRY-0eac1c8b550777a11ce595f8", "TASKENTRY-00928b970e18cc9397e839fe",
				"TASKENTRY-94d4b7d564d2de5b7a3513cf", "TASKENTRY-865b50cb1516d53d10887928",
			},
			anchors: []string{"fingerprint canónico", "snapshot, evento y outbox atómicos en V2", "continúa el inventario de identidad fuerte", "no son evidencia independiente"},
		},
		"identidad_orden_y_dedupe_documental": {
			ref: "BEHAVIOR-AGENT-BATCH-12-BACKLOG-IDENTITY",
			entryRefs: []string{
				"TASKENTRY-12bfce3096178ff0152fc252", "TASKENTRY-28bae1da40d5a6d9f036d452",
				"TASKENTRY-7c27b6fba336246697a348db", "TASKENTRY-3a0ba62596f55f1de4f899ea",
			},
			anchors: []string{"task instance refs", "backlog_proposal_fingerprint", "documentos no escriben lifecycle", "no son evidencia independiente"},
		},
	}
}

func validateBehaviorBatch12(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch12Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch12Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch12Groups()
	expected := make(map[string]bool, behaviorBatch12ExpectedEntryCount)
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
	if len(records) != 5 || len(groups) != 5 || len(expected) != behaviorBatch12ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de los lotes 01-11: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch12ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch12HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch12RecordText(record)
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
				!expectedRef || seen || reused || !exists || !behaviorBatch12EvidenceMatches(evidence, entry) {
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
	if len(groups) != 0 || !behaviorBatch12RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch12Ledger(raw []byte) (map[string]behaviorBatch12TaskEntry, error) {
	result := make(map[string]behaviorBatch12TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch12TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch12EvidenceMatches(evidence behaviorBatchEvidence, entry behaviorBatch12TaskEntry) bool {
	return entry.CapabilityID == "ORC-12" && entry.CapabilityDecision == "accept" &&
		entry.SemanticReviewState == "reviewed" &&
		behaviorBatchEvidenceMatches("ORC-12", evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch12HeaderValid(record behaviorBatchRecord, group behaviorBatch12Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == "ORC-12" &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch12RecordText(record behaviorBatchRecord) string {
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

func behaviorBatch12RangesDisjoint(evidence []behaviorBatchEvidence) bool {
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

func validateBehaviorBatch12Roadmap(raw []byte) error {
	var roadmap behaviorBatch12Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	for _, capability := range roadmap.Capabilities {
		if capability.ID != "ORC-12" {
			continue
		}
		if capability.Title != "Leases, CAS, idempotencia, retry y backoff" ||
			capability.Decision != "accept" || capability.Kind != "orchestration" ||
			capability.OwnerContext != "atomic_state_outbox" || capability.Status != "accredited" ||
			len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != "AC-V06-ATOMIC-STATE-OUTBOX" ||
			len(capability.EvidenceRefs) != 3 ||
			capability.EvidenceRefs[0] != "acceptance/v06_atomic_state_outbox_test.go" ||
			capability.EvidenceRefs[1] != "acceptance/fixtures/v06_atomic_state_outbox.json" ||
			capability.EvidenceRefs[2] != "product/evidence/v06_atomic_state_outbox.json" {
			return fmt.Errorf("estado roadmap ORC-12 inesperado")
		}
		return nil
	}
	return fmt.Errorf("ORC-12 ausente del roadmap")
}
