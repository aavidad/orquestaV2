// Estas utilidades validan la propuesta del lote 15 contra trazabilidad y roadmap vigentes, nunca contra el runtime legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch15ExpectedEntryCount = 20

type behaviorBatch15Group struct {
	ref        string
	capability string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch15TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch15Roadmap struct {
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

type behaviorBatch15RoadmapWant struct {
	title, decision, kind, owner, acceptance, status string
	evidence                                         []string
}

func behaviorBatch15Groups() map[string]behaviorBatch15Group {
	return map[string]behaviorBatch15Group{
		"estado_operativo_honesto_y_accionable": {
			ref: "BEHAVIOR-AGENT-BATCH-15-HONEST-STATUS", capability: "UI-13",
			entryRefs: []string{"TASKENTRY-d280135cfa433911099f03d5", "TASKENTRY-19c1ff5d97bd834a3d50ea4b", "TASKENTRY-e232f1b5d28f496da65c6fa8", "TASKENTRY-d59f4d2190d0e1c69e3ed2b1"},
			anchors:   []string{"WillFinishAlone", "running_stale", "reconcile_pending", "no son evidencia independiente"},
		},
		"saneamiento_egress_con_evidencia_sin_secreto": {
			ref: "BEHAVIOR-AGENT-BATCH-15-EGRESS-REDACTION", capability: "EVD-12",
			entryRefs: []string{"TASKENTRY-09e9011bb19c3d83c5b2f0e0", "TASKENTRY-f85c60383e31835c0f9d9c34", "TASKENTRY-2fac2e2fa5264b00f54b49d4", "TASKENTRY-7b3e0a5bc6c3ef87ae147992"},
			anchors:   []string{"ContextSanitizerPortV0", "antes del egress", "evidencia pública", "no son evidencia independiente"},
		},
		"presupuesto_atribuible_medible_y_degradacion": {
			ref: "BEHAVIOR-AGENT-BATCH-15-ATTRIBUTED-BUDGET", capability: "ORC-09",
			entryRefs: []string{"TASKENTRY-82ddfb89d007e2f753c6e7ad", "TASKENTRY-d96f39ced43992e6d528842a", "TASKENTRY-7e683be1a697e7befa547bac", "TASKENTRY-88da2fec93458637cafc7dbf"},
			anchors:   []string{"usage/quota", "aplazar o degradar", "por agente y Goal", "no son evidencia independiente"},
		},
		"heuristicas_advisory_con_bloqueo_solo_material": {
			ref: "BEHAVIOR-AGENT-BATCH-15-ADVISORY-RAILS", capability: "ORC-24",
			entryRefs: []string{"TASKENTRY-08a0faf38db3a44c1dfdbcd5", "TASKENTRY-cf9d955b2b3f8b94b65a1bb1", "TASKENTRY-19281a8313957d72bb6e3f5f", "TASKENTRY-8519f92b0dc02d750501aa34"},
			anchors:   []string{"soft", "hard", "recoverable", "no son evidencia independiente"},
		},
		"api_http_tipado_fino_limitado_y_unico": {
			ref: "BEHAVIOR-AGENT-BATCH-15-TYPED-HTTP", capability: "UI-01",
			entryRefs: []string{"TASKENTRY-8cf12135f27608b6ddb39007", "TASKENTRY-ba54d22fc36f43595f816471", "TASKENTRY-318985a50027196d90b171c4", "TASKENTRY-6f19ddad70d85512aa5a48ee"},
			anchors:   []string{"único escritor", "JSON acotado", "cliente fino", "no son evidencia independiente"},
		},
	}
}

func validateBehaviorBatch15(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch15Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch15Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch15Groups()
	expected := make(map[string]bool, behaviorBatch15ExpectedEntryCount)
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
	if len(records) != 5 || len(expected) != behaviorBatch15ExpectedEntryCount {
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
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch15ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch15HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch15RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla literal ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] || !expectedRef || seen || reused || !exists || !behaviorBatch15EvidenceMatches(group.capability, evidence, entry) {
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
	if len(groups) != 0 || !behaviorBatch15RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch15Ledger(raw []byte) (map[string]behaviorBatch15TaskEntry, error) {
	result := make(map[string]behaviorBatch15TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch15TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch15EvidenceMatches(capability string, evidence behaviorBatchEvidence, entry behaviorBatch15TaskEntry) bool {
	return entry.CapabilityID == capability && entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(capability, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch15HeaderValid(record behaviorBatchRecord, group behaviorBatch15Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch15RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch15RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := 0; left < len(evidence); left++ {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef && evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch15Roadmap(raw []byte) error {
	var roadmap behaviorBatch15Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	wants := map[string]behaviorBatch15RoadmapWant{
		"UI-13":  {"Estado honesto: idle, vacío, bloqueado y roto no son equivalentes", "accept", "interface", "web_admin", "AC-V24-WEB-ADMIN", "declared", nil},
		"EVD-12": {"Redacción y leak scan de secretos en prompts, resultados, logs, artefactos, receipts y effective config", "accept", "evidence", "credentials", "AC-V08-CREDENTIALS", "accredited", []string{"acceptance/v08_credentials_test.go", "acceptance/fixtures/v08_credentials.json", "product/evidence/v08_credentials.json"}},
		"ORC-09": {"Presupuestos globales y por Goal: tokens, dinero, tiempo, procesos y disco", "accept", "orchestration", "budgets_effects", "AC-V15-BUDGETS-EFFECTS", "accredited", []string{"acceptance/v15_budgets_effects_test.go", "acceptance/fixtures/v15_budgets_effects.json", "product/evidence/v15_budgets_effects.json"}},
		"ORC-24": {"Heurísticas de contenido solo advisory; Director decide", "accept", "orchestration", "director_lease", "AC-V12-DIRECTOR-LEASE", "accredited", []string{"acceptance/v12_director_lease_test.go", "acceptance/fixtures/v12_director_lease.json", "product/evidence/v12_director_lease.json"}},
		"UI-01":  {"API HTTP tipada", "accept", "interface", "web_admin", "AC-V24-WEB-ADMIN", "declared", nil},
	}
	for _, capability := range roadmap.Capabilities {
		want, exists := wants[capability.ID]
		if !exists {
			continue
		}
		if capability.Title != want.title || capability.Decision != want.decision || capability.Kind != want.kind || capability.OwnerContext != want.owner || capability.Status != want.status || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != want.acceptance || !equalBehaviorBatch15Strings(capability.EvidenceRefs, want.evidence) {
			return fmt.Errorf("estado roadmap %s inesperado", capability.ID)
		}
		delete(wants, capability.ID)
	}
	if len(wants) != 0 {
		return fmt.Errorf("capacidades ausentes del roadmap: %v", wants)
	}
	return nil
}

func equalBehaviorBatch15Strings(left, right []string) bool {
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
