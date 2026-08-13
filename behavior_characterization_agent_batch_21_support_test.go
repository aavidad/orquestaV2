// Estas utilidades validan la propuesta del lote 21 contra trazabilidad y roadmap vigentes, nunca contra el runtime legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const behaviorBatch21ExpectedEntryCount = 20

type behaviorBatch21Group struct {
	ref        string
	capability string
	entryRefs  []string
	anchors    []string
}

type behaviorBatch21TaskEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

type behaviorBatch21Roadmap struct {
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

func behaviorBatch21Groups() map[string]behaviorBatch21Group {
	return map[string]behaviorBatch21Group{
		"programacion_acotada_preserva_contratos_y_complejidad": {
			ref:        "BEHAVIOR-AGENT-BATCH-21-CONTRACT-PRESERVING-PRODUCTION",
			capability: "STG-11",
			entryRefs:  []string{"TASKENTRY-5276168c9504f2c4618ffe01", "TASKENTRY-853ae80a1fa71da66ffb7450", "TASKENTRY-a0f68428bb80ee475ba585b1", "TASKENTRY-c7f44cd86245531c1c70e9b2"},
			anchors:    []string{"write-sets disjuntos", "contrato sin cambios", "declarar verde por LOC", "no son evidencia independiente"},
		},
		"quality_gates_durables_bloquean_avance_hasta_replan": {
			ref:        "BEHAVIOR-AGENT-BATCH-21-BLOCKING-QUALITY-GATES",
			capability: "STG-13",
			entryRefs:  []string{"TASKENTRY-c34b9f0c533e9ca1f96b0f04", "TASKENTRY-f20a734d94b9435c006fcb7d", "TASKENTRY-94649340a748836b6024c963", "TASKENTRY-77455f25db7d53c8b53b06c2"},
			anchors:    []string{"gate durable", "avance pese a blocked", "cerrar desde QualityGateRecorded", "no son evidencia independiente"},
		},
		"snapshot_diff_write_set_y_revision_identifican_candidato": {
			ref:        "BEHAVIOR-AGENT-BATCH-21-EXACT-CANDIDATE-DIFF",
			capability: "EVD-05",
			entryRefs:  []string{"TASKENTRY-50b212b9e2a18f391743c217", "TASKENTRY-69a062718da02a0371fda552", "TASKENTRY-c2002aa55ebf92ee00e44ae6", "TASKENTRY-1da35c8f34248d173829d90c"},
			anchors:    []string{"ACK.files incompleto", "baseline inmutable", "binario distinto del árbol revisado", "no son evidencia independiente"},
		},
		"ficheros_seguros_por_descriptor_y_publicacion_atomica": {
			ref:        "BEHAVIOR-AGENT-BATCH-21-DESCRIPTOR-SAFE-FILES",
			capability: "EVD-11",
			entryRefs:  []string{"TASKENTRY-42d6cd39cf907d80a337f428", "TASKENTRY-8522a6f66706f30737d4226c", "TASKENTRY-85108efe64009d81269a9190", "TASKENTRY-1a78b15d342c5d965d658fcb"},
			anchors:    []string{"Nlink igual a uno", "inode anónimo", "os.Open tras validar string", "no son evidencia independiente"},
		},
		"diagnostico_publico_redactado_acotado_y_no_autoritativo": {
			ref:        "BEHAVIOR-AGENT-BATCH-21-REDACTED-DIAGNOSTIC-EXPORT",
			capability: "UI-14",
			entryRefs:  []string{"TASKENTRY-095d9dd2ba872b9a9872db5b", "TASKENTRY-f3815785c6ef64190d41d8f8", "TASKENTRY-740f3153a638d6d2d5867276", "TASKENTRY-a50176a55be00ad330009d5d"},
			anchors:    []string{"redaction_level", "truncado sin marcador", "export que dispara reparación", "no son evidencia independiente"},
		},
	}
}

func validateBehaviorBatch21(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if err := validateBehaviorBatch21Roadmap(roadmap); err != nil {
		return err
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch21Ledger(ledger)
	if err != nil {
		return err
	}
	groups := behaviorBatch21Groups()
	expected := make(map[string]bool, behaviorBatch21ExpectedEntryCount)
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
	if len(records) != 5 || len(expected) != behaviorBatch21ExpectedEntryCount {
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
			return fmt.Errorf("entrada reutilizada de lotes 01-18: %s", ref)
		}
	}
	seenCharacterizations := make(map[string]struct{})
	allEvidence := make([]behaviorBatchEvidence, 0, behaviorBatch21ExpectedEntryCount)
	for _, record := range records {
		group, ok := groups[record.Behavior]
		_, duplicateCurrent := seenCharacterizations[record.Ref]
		_, duplicatePrevious := previousCharacterizationRefs[record.Ref]
		if !ok || duplicateCurrent || duplicatePrevious || !behaviorBatch21HeaderValid(record, group) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != 4 || len(record.Evidence) != 4 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		text := behaviorBatch21RecordText(record)
		for _, anchor := range group.anchors {
			if !strings.Contains(text, anchor) {
				return fmt.Errorf("ancla literal ausente en %s: %q", record.Behavior, anchor)
			}
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			seen, expectedRef := expected[evidence.EntryRef]
			_, reused := previousEntryRefs[evidence.EntryRef]
			if record.EntryRefs[index] != group.entryRefs[index] || evidence.EntryRef != group.entryRefs[index] || !expectedRef || seen || reused || !exists || !behaviorBatch21EvidenceMatches(evidence, entry) {
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
	if len(groups) != 0 || !behaviorBatch21RangesDisjoint(allEvidence) {
		return fmt.Errorf("faltan grupos o existen rangos solapados")
	}
	return nil
}

func decodeBehaviorBatch21Ledger(raw []byte) (map[string]behaviorBatch21TaskEntry, error) {
	result := make(map[string]behaviorBatch21TaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatch21TaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func behaviorBatch21EvidenceMatches(evidence behaviorBatchEvidence, entry behaviorBatch21TaskEntry) bool {
	return entry.CapabilityDecision == "accept" && entry.SemanticReviewState == "reviewed" && behaviorBatchEvidenceMatches(entry.CapabilityID, evidence, entry.behaviorBatchTaskEntry)
}

func behaviorBatch21HeaderValid(record behaviorBatchRecord, group behaviorBatch21Group) bool {
	return record.SchemaVersion == 1 && record.Ref == group.ref && record.CapabilityID == group.capability && record.Authority == "proposal_fixture_not_canonical_ledger" && record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" && record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork && !record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatch21RecordText(record behaviorBatchRecord) string {
	values := []string{record.Problem, record.DecisionAuthority}
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten, record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects, record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart, record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	for _, group := range groups {
		values = append(values, group...)
	}
	return strings.Join(values, "\n")
}

func behaviorBatch21RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := 0; left < len(evidence); left++ {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef && evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func validateBehaviorBatch21Roadmap(raw []byte) error {
	var roadmap behaviorBatch21Roadmap
	if err := json.Unmarshal(raw, &roadmap); err != nil {
		return err
	}
	type expectation struct {
		title, kind, owner, status, acceptance string
		evidence                               []string
	}
	expected := map[string]expectation{
		"STG-11": {"Programación o producción de contenido", "stage_template", "codex_e2e", "accredited", "AC-V22-CODEX-E2E", []string{"acceptance/v22_codex_e2e_test.go", "acceptance/fixtures/v22_codex_e2e.json", "product/evidence/v22_codex_e2e.json"}},
		"STG-13": {"Pruebas, análisis estático y validadores de dominio", "stage_template", "independent_reviews", "accredited", "AC-V18-INDEPENDENT-REVIEWS", []string{"acceptance/v18_independent_reviews_test.go", "acceptance/fixtures/v18_independent_reviews.json", "product/evidence/v18_independent_reviews.json"}},
		"EVD-05": {"Snapshot/diff exacto revisado y promovido", "evidence", "test_attestor", "accredited", "AC-V17-TEST-ATTESTOR", []string{"acceptance/v17_test_attestor_test.go", "acceptance/fixtures/v17_test_attestor.json", "product/evidence/v17_test_attestor.json"}},
		"EVD-11": {"Seguridad de ficheros: traversal, symlink, hardlink, owner, modo y fsync", "evidence", "credentials", "accredited", "AC-V08-CREDENTIALS", []string{"acceptance/v08_credentials_test.go", "acceptance/fixtures/v08_credentials.json", "product/evidence/v08_credentials.json"}},
		"UI-14":  {"Export diagnóstico redactado", "interface", "web_admin", "declared", "AC-V24-WEB-ADMIN", nil},
	}
	for _, capability := range roadmap.Capabilities {
		want, ok := expected[capability.ID]
		if !ok {
			continue
		}
		if capability.Title != want.title || capability.Decision != "accept" || capability.Kind != want.kind || capability.OwnerContext != want.owner || capability.Status != want.status || len(capability.AcceptanceContracts) != 1 || capability.AcceptanceContracts[0] != want.acceptance || !equalBehaviorBatch21Strings(capability.EvidenceRefs, want.evidence) {
			return fmt.Errorf("estado roadmap %s inesperado", capability.ID)
		}
		delete(expected, capability.ID)
	}
	if len(expected) != 0 {
		return fmt.Errorf("capabilities ausentes del roadmap: %v", expected)
	}
	return nil
}

func equalBehaviorBatch21Strings(left, right []string) bool {
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
