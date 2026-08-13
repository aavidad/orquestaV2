// Este soporte valida procedencia, no solape, roadmap y enlaces AST exactos del lote 26.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

type behaviorBatch26Want struct {
	capability, ref string
	count           int
}

var behaviorBatch26Wants = map[string]behaviorBatch26Want{
	"integracion_serializada_tras_entrega_validada":      {"STG-16", "BEHAVIOR-AGENT-BATCH-26-MERGE-GATE", 2},
	"autor_primaria_y_adversarial_sobre_mismo_candidato": {"GOV-12", "BEHAVIOR-AGENT-BATCH-26-INDEPENDENT-REVIEWS", 2},
	"watchdog_por_cpu_heartbeat_y_progreso_material":     {"OPS-18", "BEHAVIOR-AGENT-BATCH-26-IDLE-WATCHDOG", 2},
	"pipeline_secuencial_con_segunda_opinion_opcional":   {"AGT-10", "BEHAVIOR-AGENT-BATCH-26-SEQUENTIAL-MODELS", 2},
	"plugin_externo_opt_in_por_mcp_o_http":               {"EXT-15", "BEHAVIOR-AGENT-BATCH-26-EXTERNAL-PLUGIN", 1},
	"launch_receipt_con_identidad_no_inferida":           {"EVD-02", "BEHAVIOR-AGENT-BATCH-26-LAUNCH-IDENTITY", 1},
	"stream_sse_reconectable_con_fallback_controlado":    {"UI-06", "BEHAVIOR-AGENT-BATCH-26-EVENT-STREAM", 1},
}

type behaviorBatch26LedgerEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

func validateBehaviorBatch26(fixture, ledger, roadmap []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch26Ledger(ledger)
	if err != nil {
		return err
	}
	if len(records) != 7 {
		return fmt.Errorf("conductas=%d", len(records))
	}
	used := map[string]struct{}{}
	for _, raw := range previous {
		prior, decodeErr := decodeBehaviorBatchRecords(raw)
		if decodeErr != nil {
			return decodeErr
		}
		for _, record := range prior {
			for _, ref := range record.EntryRefs {
				used[ref] = struct{}{}
			}
		}
	}
	seenRefs, seenCaps, refs := map[string]struct{}{}, map[string]struct{}{}, 0
	for _, record := range records {
		want, exists := behaviorBatch26Wants[record.Behavior]
		_, duplicate := seenRefs[record.Ref]
		if !exists || duplicate || record.Ref != want.ref || record.CapabilityID != want.capability || record.Authority != "proposal_fixture_not_canonical_ledger" || record.ReviewState != "bootstrap_first_review_pending_independent_counterreview" || record.Disposition != "not_evaluated" || record.CanonicalChange || record.CreatesWork || record.ClosesCapability || record.ClaimsAccreditation || len(record.EntryRefs) != want.count || len(record.Evidence) != want.count || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro inválido: %s", record.Ref)
		}
		for index, evidence := range record.Evidence {
			entry, ok := entries[evidence.EntryRef]
			_, reused := used[evidence.EntryRef]
			if !ok || reused || record.EntryRefs[index] != evidence.EntryRef || entry.CapabilityDecision != "accept" || entry.SemanticReviewState != "reviewed" || !behaviorBatchEvidenceMatches(record.CapabilityID, evidence, entry.behaviorBatchTaskEntry) {
				return fmt.Errorf("procedencia inválida: %s", evidence.EntryRef)
			}
			used[evidence.EntryRef], refs = struct{}{}, refs+1
		}
		seenRefs[record.Ref], seenCaps[record.CapabilityID] = struct{}{}, struct{}{}
	}
	if len(seenRefs) != 7 || len(seenCaps) != 7 || refs != 11 {
		return fmt.Errorf("refs=%d conducts=%d caps=%d", refs, len(seenRefs), len(seenCaps))
	}
	return validateBehaviorBatch26Roadmap(roadmap)
}

func decodeBehaviorBatch26Ledger(raw []byte) (map[string]behaviorBatch26LedgerEntry, error) {
	result := map[string]behaviorBatch26LedgerEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		var entry behaviorBatch26LedgerEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func validateBehaviorBatch26Roadmap(raw []byte) error {
	var root struct {
		Entries []struct {
			ID, Decision, Status string
			Acceptance           []string `json:"acceptance_contracts"`
			Evidence             []string `json:"evidence_refs"`
		} `json:"capability_entries"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	wants := map[string]struct {
		decision, status, acceptance string
		evidence                     int
	}{
		"STG-16": {"accept", "accredited", "AC-V18-INDEPENDENT-REVIEWS", 3}, "GOV-12": {"accept", "accredited", "AC-V18-INDEPENDENT-REVIEWS", 3}, "OPS-18": {"accept", "declared", "AC-V32-OPERATIONS-TELEMETRY", 0}, "AGT-10": {"accept", "declared", "AC-V25-PROVIDER-ADAPTERS", 0}, "EXT-15": {"accept", "declared", "AC-V28-DOMAIN-PLUGINS", 0}, "EVD-02": {"accept", "accredited", "AC-V06-ATOMIC-STATE-OUTBOX", 3}, "UI-06": {"accept", "declared", "AC-V24-WEB-ADMIN", 0},
	}
	for _, entry := range root.Entries {
		want, exists := wants[entry.ID]
		if !exists {
			continue
		}
		if entry.Decision != want.decision || entry.Status != want.status || len(entry.Acceptance) != 1 || entry.Acceptance[0] != want.acceptance || len(entry.Evidence) != want.evidence {
			return fmt.Errorf("roadmap %s inesperado", entry.ID)
		}
		delete(wants, entry.ID)
	}
	if len(wants) != 0 {
		return fmt.Errorf("roadmap incompleto: %v", wants)
	}
	return nil
}

func validateBehaviorBatch26Assessments(raw, fixture, snapshot []byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	sources := map[string]map[string]struct{}{}
	for _, record := range records {
		sources[record.Ref] = map[string]struct{}{}
		for _, evidence := range record.Evidence {
			sources[record.Ref][evidence.SourceRef] = struct{}{}
		}
	}
	snapshotByOccurrence := map[string]map[string]any{}
	for _, line := range bytes.Split(bytes.TrimSuffix(snapshot, []byte("\n")), []byte("\n")) {
		var item map[string]any
		if err := json.Unmarshal(line, &item); err != nil {
			return err
		}
		if ref, ok := item["occurrence_ref"].(string); ok {
			snapshotByOccurrence[ref] = item
		}
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' || bytes.Contains(raw, []byte("\r")) {
		return errors.New("assessment JSONL inválido")
	}
	seen, linked, noDirect := map[string]struct{}{}, 0, 0
	for lineNo, line := range bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n")) {
		var row map[string]any
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.UseNumber()
		if err := decoder.Decode(&row); err != nil {
			return err
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return errors.New("valor posterior")
		}
		var compact bytes.Buffer
		if err := json.Compact(&compact, line); err != nil || !bytes.Equal(compact.Bytes(), line) {
			return fmt.Errorf("assessment %d no canónico", lineNo+1)
		}
		ref, _ := row["characterization_ref"].(string)
		if !strings.HasPrefix(ref, "BEHAVIOR-AGENT-BATCH-26-") {
			continue
		}
		mapping, _ := row["function_mapping"].(map[string]any)
		if _, ok := sources[ref]; !ok || row["review_state"] != "independent_bootstrap_counterreview_completed_advisory" || row["authority"] != "advisory_not_capability_state" || mapping["snapshot_census_sha256"] != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" {
			return fmt.Errorf("assessment inválido: %s", ref)
		}
		links, _ := mapping["links"].([]any)
		state, _ := mapping["state"].(string)
		switch state {
		case "linked_exact":
			if len(links) < 1 || len(links) > 2 {
				return fmt.Errorf("links inválidos: %s", ref)
			}
			linked += len(links)
		case "reviewed_no_direct_function":
			if len(links) != 0 {
				return fmt.Errorf("no_direct con links: %s", ref)
			}
			noDirect++
		default:
			return fmt.Errorf("mapping state inválido: %s", ref)
		}
		for _, value := range links {
			link := value.(map[string]any)
			occurrence := link["occurrence_ref"].(string)
			exact, exists := snapshotByOccurrence[occurrence]
			if !exists || link["review_state"] != "bootstrap_symbol_mapping_reviewed_advisory" {
				return fmt.Errorf("occurrence inválida: %s", occurrence)
			}
			for _, key := range []string{"declaration_ref", "variant_ref", "source_path", "git_blob_oid", "signature", "ast_sha256", "body_sha256"} {
				if link[key] != exact[key] {
					return fmt.Errorf("drift %s en %s", key, occurrence)
				}
			}
			for _, evidence := range link["evidence_refs"].([]any) {
				if _, ok := sources[ref][evidence.(string)]; !ok {
					return fmt.Errorf("evidencia ajena: %s", evidence)
				}
			}
		}
		if _, duplicate := seen[ref]; duplicate {
			return fmt.Errorf("assessment duplicado: %s", ref)
		}
		seen[ref] = struct{}{}
	}
	if len(seen) != 7 || linked != 5 || noDirect != 2 {
		return fmt.Errorf("assessments=%d links=%d no_direct=%d", len(seen), linked, noDirect)
	}
	return nil
}

func behaviorBatch26Previous(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 25)
	for index := 1; index <= 25; index++ {
		result = append(result, behaviorBatch26Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", index)))
	}
	return result
}
func behaviorBatch26Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func behaviorBatch26Path(env, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(env)); value != "" {
		return value
	}
	return fallback
}
