// Este soporte valida procedencia, denominadores, roadmap y enlaces del lote 31.
package orquesta_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type behaviorBatch31Want struct {
	capability, ref string
	count           int
}

var behaviorBatch31Wants = map[string]behaviorBatch31Want{
	"promocion_dev_staging_prod_con_cierre_y_solapes_verificados":   {"OPS-22", "BEHAVIOR-AGENT-BATCH-31-CAUSAL-PROMOTION", 1},
	"targets_local_contenedor_y_systemd_por_adaptadores_explicitos": {"OPS-23", "BEHAVIOR-AGENT-BATCH-31-LOCAL-CONTAINER-SYSTEMD", 2},
	"desktop_y_mobile_store_solo_por_target_solicitado":             {"OPS-25", "BEHAVIOR-AGENT-BATCH-31-DESKTOP-MOBILE", 1},
	"getters_schema_docs_y_ui_derivados_del_mismo_registry":         {"OPS-27", "BEHAVIOR-AGENT-BATCH-31-GENERATED-CONFIG-SURFACES", 1},
	"config_doctor_clasifica_reuse_replace_new_y_retira_aliases":    {"OPS-29", "BEHAVIOR-AGENT-BATCH-31-CONFIG-DOCTOR", 1},
	"entorno_de_hijo_por_allowlist_exacta_y_receipt_redactado":      {"OPS-30", "BEHAVIOR-AGENT-BATCH-31-CHILD-ENV-ALLOWLIST", 2},
	"padre_espera_solo_hijos_contractuales_y_bloquea_fallos":        {"ORC-05", "BEHAVIOR-AGENT-BATCH-31-CHILDREN-ACK", 2},
	"routing_por_capacidad_coste_latencia_y_esfuerzo":               {"ORC-07", "BEHAVIOR-AGENT-BATCH-31-CAPABILITY-ROUTING", 2},
	"mensajeria_dirigida_y_handoff_antes_de_escalar":                {"ORC-15", "BEHAVIOR-AGENT-BATCH-31-AGENT-HANDOFF", 2},
	"checkpoint_como_evidencia_sin_time_travel_de_estado":           {"ORC-18", "BEHAVIOR-AGENT-BATCH-31-REJECTED-TIME-TRAVEL", 2},
}

type behaviorBatch31LedgerEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

func validateBehaviorBatch31(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if bytes.Contains(fixture, []byte("canonical_source")) {
		return errors.New("fixture publica cuerpo legacy")
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch31Ledger(ledger)
	if err != nil {
		return err
	}
	if len(records) != 10 {
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
	seenRefs, seenCaps, refs, exhausted, collapsed := map[string]struct{}{}, map[string]struct{}{}, 0, 0, 0
	allEvidence := make([]behaviorBatchEvidence, 0, 16)
	for _, record := range records {
		want, exists := behaviorBatch31Wants[record.Behavior]
		_, duplicate := seenRefs[record.Ref]
		if !exists || duplicate || record.Ref != want.ref || record.CapabilityID != want.capability || record.Authority != "proposal_fixture_not_canonical_ledger" || record.ReviewState != "bootstrap_first_review_pending_independent_counterreview" || record.Disposition != "not_evaluated" || record.CanonicalChange || record.CreatesWork || record.ClosesCapability || record.ClaimsAccreditation || len(record.EntryRefs) != want.count || len(record.Evidence) != want.count || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro inválido: %s", record.Ref)
		}
		text := strings.Join(record.Uncertainties, "\n")
		if strings.Contains(text, "denominator_exhausted") {
			exhausted++
		}
		if strings.Contains(text, "source_overlap_collapsed") {
			collapsed++
		}
		for index, evidence := range record.Evidence {
			entry, ok := entries[evidence.EntryRef]
			_, reused := used[evidence.EntryRef]
			wantDecision := "accept"
			if record.CapabilityID == "ORC-18" {
				wantDecision = "reject"
			}
			if !ok || reused || record.EntryRefs[index] != evidence.EntryRef || entry.CapabilityDecision != wantDecision || entry.SemanticReviewState != "reviewed" || !behaviorBatchEvidenceMatches(record.CapabilityID, evidence, entry.behaviorBatchTaskEntry) {
				return fmt.Errorf("procedencia inválida: %s", evidence.EntryRef)
			}
			used[evidence.EntryRef], refs = struct{}{}, refs+1
			allEvidence = append(allEvidence, evidence)
		}
		seenRefs[record.Ref], seenCaps[record.CapabilityID] = struct{}{}, struct{}{}
	}
	if len(seenRefs) != 10 || len(seenCaps) != 10 || refs != 16 || exhausted != 3 || collapsed != 1 || !behaviorBatch31RangesDisjoint(allEvidence) {
		return fmt.Errorf("refs=%d caps=%d exhausted=%d collapsed=%d", refs, len(seenCaps), exhausted, collapsed)
	}
	return validateBehaviorBatch31Roadmap(roadmap)
}

func behaviorBatch31RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := range evidence {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef && evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func decodeBehaviorBatch31Ledger(raw []byte) (map[string]behaviorBatch31LedgerEntry, error) {
	result := map[string]behaviorBatch31LedgerEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		var entry behaviorBatch31LedgerEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func validateBehaviorBatch31Roadmap(raw []byte) error {
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
		status, acceptance string
		evidence           int
	}{"OPS-22": {"declared", "AC-V29-DEPLOY-NOTIFICATIONS", 0}, "OPS-23": {"declared", "AC-V29-DEPLOY-NOTIFICATIONS", 0}, "OPS-25": {"declared", "AC-V29-DEPLOY-NOTIFICATIONS", 0}, "OPS-27": {"accredited", "AC-V07-CONFIG", 3}, "OPS-29": {"accredited", "AC-V07-CONFIG", 3}, "OPS-30": {"accredited", "AC-V07-CONFIG", 3}, "ORC-05": {"accredited", "AC-V13-MAILBOX", 3}, "ORC-07": {"declared", "AC-V25-PROVIDER-ADAPTERS", 0}, "ORC-15": {"declared", "AC-V27-CONTEXT-RAG-EVALS", 0}, "ORC-18": {"declared", "AC-V05-GOAL-DAG-PHASES", 0}}
	for _, entry := range root.Entries {
		want, exists := wants[entry.ID]
		if !exists {
			continue
		}
		wantDecision := "accept"
		if entry.ID == "ORC-18" {
			wantDecision = "reject"
		}
		if entry.Decision != wantDecision || entry.Status != want.status || len(entry.Acceptance) != 1 || entry.Acceptance[0] != want.acceptance || len(entry.Evidence) != want.evidence {
			return fmt.Errorf("roadmap %s inesperado", entry.ID)
		}
		delete(wants, entry.ID)
	}
	if len(wants) != 0 {
		return fmt.Errorf("roadmap incompleto: %v", wants)
	}
	return nil
}

func validateBehaviorBatch31Assessments(raw, fixture, snapshot []byte) error {
	if bytes.Contains(raw, []byte("canonical_source")) {
		return errors.New("assessment publica cuerpo legacy")
	}
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
	byOccurrence := map[string]map[string]any{}
	for _, line := range bytes.Split(bytes.TrimSuffix(snapshot, []byte("\n")), []byte("\n")) {
		var item map[string]any
		if err := json.Unmarshal(line, &item); err != nil {
			return err
		}
		if ref, ok := item["occurrence_ref"].(string); ok {
			byOccurrence[ref] = item
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
		ref := row["characterization_ref"].(string)
		if !strings.HasPrefix(ref, "BEHAVIOR-AGENT-BATCH-31-") {
			continue
		}
		mapping := row["function_mapping"].(map[string]any)
		if _, ok := sources[ref]; !ok || row["review_state"] != "independent_bootstrap_counterreview_completed_advisory" || row["authority"] != "advisory_not_capability_state" || mapping["snapshot_census_sha256"] != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" || len(strings.TrimSpace(row["green_rationale"].(string))) < 30 || len(strings.TrimSpace(row["agent_recommendation"].(string))) < 30 || len(strings.TrimSpace(mapping["rationale"].(string))) < 30 {
			return fmt.Errorf("assessment inválido: %s", ref)
		}
		if !containsBehaviorBatch31Value([]string{"documented_green_unverified", "partial_or_mixed", "failed_or_negative", "unknown"}, row["historical_green_state"].(string)) || !containsBehaviorBatch31Value([]string{"v2_existing", "contract", "idea", "negative_lesson"}, row["reuse_kind"].(string)) || !containsBehaviorBatch31Value([]string{"exact_or_primary", "partial_overlap", "not_implemented"}, row["exact_v2_fit"].(string)) {
			return fmt.Errorf("taxonomía inválida: %s", ref)
		}
		links := mapping["links"].([]any)
		switch mapping["state"] {
		case "linked_exact":
			if len(links) < 1 || len(links) > 3 {
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
			exact, exists := byOccurrence[occurrence]
			if !exists || link["review_state"] != "bootstrap_symbol_mapping_reviewed_advisory" || !containsBehaviorBatch31Value([]string{"primary_implementation", "supporting_mechanism", "negative_example"}, link["relation"].(string)) || len(strings.TrimSpace(link["semantic_rationale"].(string))) < 30 {
				return fmt.Errorf("occurrence inválida: %s", occurrence)
			}
			for _, key := range []string{"declaration_ref", "variant_ref", "source_root", "source_path", "git_blob_oid", "blob_digest", "package_path", "package_name", "symbol_kind", "name", "receiver", "signature", "source_sha256", "ast_sha256", "body_sha256", "start_line", "end_line"} {
				if fmt.Sprint(link[key]) != fmt.Sprint(exact[key]) {
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
	if len(seen) != 10 || linked != 10 || noDirect != 3 {
		return fmt.Errorf("assessments=%d links=%d no_direct=%d", len(seen), linked, noDirect)
	}
	return nil
}

func containsBehaviorBatch31Value(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func behaviorBatch31Previous(t *testing.T) [][]byte {
	t.Helper()
	paths, err := filepath.Glob("product/traceability/fixtures/behavior_characterization_agent_batch_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	result := make([][]byte, 0, len(paths))
	for _, path := range paths {
		if strings.HasSuffix(path, "batch_31.jsonl") {
			continue
		}
		result = append(result, behaviorBatch31Read(t, path))
	}
	return result
}
func behaviorBatch31Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func behaviorBatch31Path(env, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(env)); value != "" {
		return value
	}
	return fallback
}
