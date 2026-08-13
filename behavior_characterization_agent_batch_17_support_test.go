// Estas utilidades validan la propuesta del lote 17 contra ledger, roadmap y lotes previos; nunca ejecutan legacy.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch17ExpectedEntryCount = 20

type behaviorBatch17Group struct {
	ref        string
	capability string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch17TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	Disposition         string `json:"disposition"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch17Roadmap struct {
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

type behaviorBatch17RoadmapExpected struct {
	title, kind, owner, contract, status string
	evidenceCount                        int
}

func behaviorBatch17Groups() map[string]behaviorBatch17Group {
	return map[string]behaviorBatch17Group{
		"servidor_mcp_publico_real_fino_y_opt_in": {
			ref: "BEHAVIOR-AGENT-BATCH-17-REAL-MCP-SERVER", capability: "UI-02",
			entryRefs: []string{"TASKENTRY-e0f2506885afda6217d4334c", "TASKENTRY-fa05924d15cdea75cc51c6e4", "TASKENTRY-9935aa7ada1f101a8dcd0da8", "TASKENTRY-45ed0ac143de990691f63600"},
			anchors:   []string{"adaptadores finos", "ninguna mutación actual", "command registry", "no son evidencia independiente"},
		},
		"mailbox_causal_con_claim_lease_delivery_y_ack": {
			ref: "BEHAVIOR-AGENT-BATCH-17-CAUSAL-MAILBOX", capability: "ORC-14",
			entryRefs: []string{"TASKENTRY-9095e497bff94371737e05ae", "TASKENTRY-d6133be5e9a5a97f9a2551b9", "TASKENTRY-0cf3ca2df1b3ffc4f329aa2a", "TASKENTRY-3dec7fb43871e8c4f0688520"},
			anchors:   []string{"claim con lease y fencing", "ACK del destinatario", "artefacto sin ACK", "no son evidencia independiente"},
		},
		"capability_spec_unica_para_schema_dispatch_permisos_coste_y_receipts": {
			ref: "BEHAVIOR-AGENT-BATCH-17-CANONICAL-TOOL-SPEC", capability: "TLS-01",
			entryRefs: []string{"TASKENTRY-180a99122b94d561c3542410", "TASKENTRY-35ad10bbcf296b17b172c08f", "TASKENTRY-1780362793c7fccb2fbe1321", "TASKENTRY-6cd1eb96942e633998f35bc2"},
			anchors:   []string{"CapabilitySpec única", "acción aceptada por llamada directa", "permisos, coste y receipts", "no son evidencia independiente"},
		},
		"context_bundle_minimo_resuelto_antes_de_lanzar_workitem": {
			ref: "BEHAVIOR-AGENT-BATCH-17-MINIMAL-CONTEXT-BUNDLE", capability: "CTX-01",
			entryRefs: []string{"TASKENTRY-e4c7b7b2e0e3bff234ea8fa6", "TASKENTRY-4fce619e471f637bc4ef270d", "TASKENTRY-43179e7e47d1586407e0ead0", "TASKENTRY-c2d30080b337cf0c3489fb2e"},
			anchors:   []string{"bundle mínimo por WorkItem", "APP-PLAN-003 en curso", "materialización detrás de puerto", "no son evidencia independiente"},
		},
		"smoke_real_opt_in_aislado_con_ack_receipts_y_cleanup": {
			ref: "BEHAVIOR-AGENT-BATCH-17-ISOLATED-REAL-SMOKE", capability: "OPS-20",
			entryRefs: []string{"TASKENTRY-89fe46a66b0d0f00e6da152a", "TASKENTRY-ec53e647b932c7ad455b68b4", "TASKENTRY-94278c5cc5d5ce0b9a609d85", "TASKENTRY-4c6304c3c2cd83a72f9faae4"},
			anchors:   []string{"cero procesos vivos", "Hermes real pendiente", "harness aislado", "no son evidencia independiente"},
		},
	}
}

func behaviorBatch17RoadmapExpectedValues() map[string]behaviorBatch17RoadmapExpected {
	return map[string]behaviorBatch17RoadmapExpected{
		"UI-02":  {"Servidor MCP real", "interface", "command_registry", "AC-V20-COMMAND-REGISTRY", "accredited", 3},
		"ORC-14": {"Mailbox real: admitido, reclamado, entregado y ACK del destinatario", "orchestration", "mailbox", "AC-V13-MAILBOX", "accredited", 3},
		"TLS-01": {"Registro único de tools con schema, versión, permisos, coste y receipts", "tooling", "tools_skills_sdk", "AC-V26-TOOLS-SKILLS-SDK", "declared", 0},
		"CTX-01": {"Context bundle mínimo por WorkItem", "context", "context_rag_evals", "AC-V27-CONTEXT-RAG-EVALS", "declared", 0},
		"OPS-20": {"Smokes reales y nightly", "operations", "operations_telemetry", "AC-V32-OPERATIONS-TELEMETRY", "declared", 0},
	}
}

func validateBehaviorBatch17(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch17Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch17Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch17Groups()
	expected := make(map[string]bool, behaviorBatch17ExpectedEntryCount)
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
	if len(records) != 5 || len(expected) != behaviorBatch17ExpectedEntryCount {
		return fmt.Errorf("tamaño inválido: conductas=%d entradas=%d", len(records), len(expected))
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
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch17ExpectedEntryCount)
	for _, record := range records {
		group, known := groups[record.Behavior]
		_, duplicate := seenRefs[record.Ref]
		_, usedBefore := previousRefs[record.Ref]
		if !known || duplicate || usedBefore || !behaviorBatch17HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch17RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			seen, wanted := expected[evidence.EntryRef]
			_, reused := previousEntries[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] ||
				!wanted || seen || reused || !exists || !behaviorBatch17EvidenceMatches(group.capability, evidence, entry) {
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
	if len(groups) != 0 || !behaviorBatch17RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch17Ledger(raw []byte) (map[string]behaviorBatch17TaskEntry, error) {
	result := make(map[string]behaviorBatch17TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch17TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch17EvidenceMatches(capability string, evidence behaviorBatchEvidence, entry behaviorBatch17TaskEntry) bool {
	dispositionOK := entry.Disposition == "accepted_pending_reimplementation" || entry.Disposition == "historical_superseded_by_capability"
	return entry.CapabilityID == capability && entry.CapabilityDecision == "accept" && dispositionOK &&
		entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(capability, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch17HeaderValid(record behaviorBatchRecord, group behaviorBatch17Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability &&
		record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch17RecordText(record behaviorBatchRecord) string {
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

func behaviorBatch17RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := 0; left < len(evidence); left++ {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef &&
				evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch17Roadmap(raw []byte) error {
	var roadmap behaviorBatch17Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	expected := behaviorBatch17RoadmapExpectedValues()
	for _, capability := range roadmap.Capabilities {
		want, exists := expected[capability.ID]
		if !exists {
			continue
		}
		if capability.Title != want.title || capability.Decision != "accept" || capability.Kind != want.kind ||
			capability.OwnerContext != want.owner || capability.Status != want.status || len(capability.AcceptanceContracts) != 1 ||
			capability.AcceptanceContracts[0] != want.contract || len(capability.EvidenceRefs) != want.evidenceCount {
			return fmt.Errorf("estado roadmap inesperado: %s", capability.ID)
		}
		delete(expected, capability.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("capabilities ausentes del roadmap: %v", expected)
	}
	return nil
}
