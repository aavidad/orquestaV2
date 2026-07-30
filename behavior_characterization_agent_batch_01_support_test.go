// Estas utilidades validan el fixture contra la trazabilidad congelada, nunca contra el árbol legado.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type behaviorBatchEvidence struct {
	EntryRef   string `json:"task_entry_ref"`
	SourceRef  string `json:"source_ref"`
	SourceSHA  string `json:"source_sha256"`
	FirstLine  int    `json:"subject_first_line"`
	LastLine   int    `json:"subject_last_line"`
	SubjectSHA string `json:"subject_sha256"`
}

type behaviorBatchAccess struct {
	Permissions []string `json:"permissions"`
	Secrets     []string `json:"secrets"`
	Effects     []string `json:"effects"`
}

type behaviorBatchRecovery struct {
	Failure     []string `json:"failure"`
	Retry       []string `json:"retry"`
	Concurrency []string `json:"concurrency"`
	Restart     []string `json:"restart"`
}

type behaviorBatchRecord struct {
	SchemaVersion       int                     `json:"schema_version"`
	Ref                 string                  `json:"characterization_ref"`
	Behavior            string                  `json:"behavior_key"`
	CapabilityID        string                  `json:"capability_id"`
	Authority           string                  `json:"authority"`
	EntryRefs           []string                `json:"task_entry_refs"`
	Problem             string                  `json:"problem"`
	Users               []string                `json:"users"`
	Inputs              []string                `json:"inputs"`
	Outputs             []string                `json:"outputs"`
	StateRead           []string                `json:"state_read"`
	StateWritten        []string                `json:"state_written"`
	DecisionAuthority   string                  `json:"decision_authority"`
	Permissions         behaviorBatchAccess     `json:"permissions_secrets_effects"`
	Recovery            behaviorBatchRecovery   `json:"failure_retry_concurrency_restart"`
	Worked              []string                `json:"worked"`
	Failed              []string                `json:"did_not_work"`
	Preserve            []string                `json:"preserve"`
	Avoid               []string                `json:"avoid"`
	Evidence            []behaviorBatchEvidence `json:"evidence"`
	Uncertainties       []string                `json:"uncertainties"`
	Attempts            []string                `json:"attempts"`
	ReviewState         string                  `json:"review_state"`
	Disposition         string                  `json:"disposition"`
	CanonicalChange     bool                    `json:"canonical_state_change"`
	CreatesWork         bool                    `json:"creates_work_item"`
	ClosesCapability    bool                    `json:"closes_capability"`
	ClaimsAccreditation bool                    `json:"claims_accreditation"`
}

type behaviorBatchTaskEntry struct {
	EntryRef        string `json:"entry_ref"`
	CapabilityID    string `json:"capability_id"`
	SourceRef       string `json:"source_ref"`
	SourceSHA       string `json:"source_sha256"`
	FirstLine       int    `json:"subject_first_line"`
	LastLine        int    `json:"subject_last_line"`
	SubjectSHA      string `json:"subject_sha256"`
	ClosureEvidence string `json:"closure_evidence"`
}

func validateBehaviorBatch(fixture, ledger []byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatchLedger(ledger)
	if err != nil {
		return err
	}
	expected := make(map[string]bool, len(behaviorBatchExpectedRefs))
	for _, ref := range behaviorBatchExpectedRefs {
		expected[ref] = false
	}
	wantGroups := map[string]struct {
		capability string
		count      int
		ref        string
	}{
		"disponibilidad_y_cuota_vivas":           {"AGT-12", 7, "BEHAVIOR-AGENT-BATCH-01-LIVE-QUOTA"},
		"pools_sesiones_y_reservas":              {"ORC-28", 5, "BEHAVIOR-AGENT-BATCH-01-POOLS"},
		"decision_durable_previa_al_lanzamiento": {"ORC-28", 6, "BEHAVIOR-AGENT-BATCH-01-CAPACITY-DECISION"},
		"relevo_preventivo_de_sesion":            {"ORC-29", 6, "BEHAVIOR-AGENT-BATCH-01-HANDOFF"},
	}
	seenCharacterizations := make(map[string]struct{}, len(records))
	if len(records) != len(wantGroups) {
		return fmt.Errorf("registros=%d", len(records))
	}
	for _, record := range records {
		want, ok := wantGroups[record.Behavior]
		_, duplicateRef := seenCharacterizations[record.Ref]
		if !ok || duplicateRef || !behaviorBatchHeaderValid(record, want.capability, want.ref) {
			return fmt.Errorf("cabecera o grupo inválido: %s", record.Ref)
		}
		if len(record.EntryRefs) != want.count || len(record.Evidence) != want.count ||
			!behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro incompleto: %s", record.Behavior)
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			if record.EntryRefs[index] != evidence.EntryRef || expected[evidence.EntryRef] ||
				!exists || !behaviorBatchEvidenceMatches(record.CapabilityID, evidence, entry) {
				return fmt.Errorf("procedencia incoherente: %s", evidence.EntryRef)
			}
			expected[evidence.EntryRef] = true
		}
		seenCharacterizations[record.Ref] = struct{}{}
		delete(wantGroups, record.Behavior)
	}
	for ref, seen := range expected {
		if !seen {
			return fmt.Errorf("falta %s", ref)
		}
	}
	if len(wantGroups) != 0 {
		return fmt.Errorf("faltan grupos: %v", wantGroups)
	}
	return nil
}

func behaviorBatchHeaderValid(record behaviorBatchRecord, capability, ref string) bool {
	return record.SchemaVersion == 1 && record.Ref == ref &&
		record.CapabilityID == capability && record.Authority == "proposal_fixture_not_canonical_ledger" &&
		record.ReviewState == "bootstrap_first_review_pending_independent_counterreview" &&
		record.Disposition == "not_evaluated" && !record.CanonicalChange && !record.CreatesWork &&
		!record.ClosesCapability && !record.ClaimsAccreditation
}

func behaviorBatchEvidenceMatches(capability string, evidence behaviorBatchEvidence, entry behaviorBatchTaskEntry) bool {
	return entry.CapabilityID == capability && entry.ClosureEvidence == "not_verified" &&
		entry.SourceRef == evidence.SourceRef && entry.SourceSHA == evidence.SourceSHA &&
		entry.FirstLine == evidence.FirstLine && entry.LastLine == evidence.LastLine &&
		entry.SubjectSHA == evidence.SubjectSHA
}

func behaviorBatchFieldsPresent(record behaviorBatchRecord) bool {
	groups := [][]string{record.Users, record.Inputs, record.Outputs, record.StateRead, record.StateWritten,
		record.Permissions.Permissions, record.Permissions.Secrets, record.Permissions.Effects,
		record.Recovery.Failure, record.Recovery.Retry, record.Recovery.Concurrency, record.Recovery.Restart,
		record.Worked, record.Failed, record.Preserve, record.Avoid, record.Uncertainties, record.Attempts}
	if strings.TrimSpace(record.Problem) == "" || strings.TrimSpace(record.DecisionAuthority) == "" {
		return false
	}
	for _, group := range groups {
		if !behaviorBatchStringGroupValid(group) {
			return false
		}
	}
	return true
}

func behaviorBatchStringGroupValid(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			return false
		}
		if _, duplicate := seen[normalized]; duplicate {
			return false
		}
		seen[normalized] = struct{}{}
	}
	return len(values) != 0
}

func decodeBehaviorBatchRecords(raw []byte) ([]behaviorBatchRecord, error) {
	if !bytes.HasSuffix(raw, []byte("\n")) {
		return nil, errors.New("el fixture debe terminar en LF")
	}
	var records []behaviorBatchRecord
	for _, line := range bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n")) {
		var record behaviorBatchRecord
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, err
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return nil, errors.New("JSON con valores posteriores")
		}
		canonical, err := json.Marshal(record)
		if err != nil || !bytes.Equal(canonical, line) {
			return nil, fmt.Errorf("JSON no canónico: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}

func decodeBehaviorBatchLedger(raw []byte) (map[string]behaviorBatchTaskEntry, error) {
	result := make(map[string]behaviorBatchTaskEntry)
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		var entry behaviorBatchTaskEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}
