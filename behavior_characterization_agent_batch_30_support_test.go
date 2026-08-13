// Este soporte valida procedencia, denominadores y enlaces exactos del lote 30.
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

type behaviorBatch30Want struct {
	capability, ref string
	count           int
}

var behaviorBatch30Wants = map[string]behaviorBatch30Want{
	"inventario_de_artefactos_aceptados_antes_de_crear_trabajo":        {"OPE-01", "BEHAVIOR-AGENT-BATCH-30-REUSE-INVENTORY", 1},
	"visual_como_insumo_reutilizable_hasta_tener_refs_materializables": {"OPE-05", "BEHAVIOR-AGENT-BATCH-30-VISUAL-REUSE", 1},
	"revision_legal_pedagogica_factual_y_editorial_por_receipts":       {"OPE-08", "BEHAVIOR-AGENT-BATCH-30-MULTIDOMAIN-QA", 1},
	"ensamblado_de_tema_validado_por_refs_opacas":                      {"OPE-11", "BEHAVIOR-AGENT-BATCH-30-TOPIC-ASSEMBLY", 2},
	"paquete_final_promovido_solo_con_evidencia_editorial":             {"OPE-17", "BEHAVIOR-AGENT-BATCH-30-GOVERNED-PUBLICATION", 1},
	"configuracion_editable_no_sensible_con_credential_ref":            {"OPS-02", "BEHAVIOR-AGENT-BATCH-30-EDITABLE-CONFIG", 2},
	"defaults_y_aliases_resueltos_solo_en_registry":                    {"OPS-06", "BEHAVIOR-AGENT-BATCH-30-REGISTRY-ONLY", 2},
	"credencial_resuelta_al_lanzar_sin_copiarla_al_goal":               {"OPS-08", "BEHAVIOR-AGENT-BATCH-30-CREDENTIAL-REFERENCE", 2},
	"un_solo_motor_de_estado_activo_por_despliegue":                    {"OPS-12", "BEHAVIOR-AGENT-BATCH-30-SINGLE-STATE-ENGINE", 2},
	"retencion_y_purga_con_scope_manifest_y_bloqueos":                  {"OPS-17", "BEHAVIOR-AGENT-BATCH-30-RETENTION", 2},
}

type behaviorBatch30LedgerEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

func validateBehaviorBatch30(fixture, ledger, roadmap []byte, previous [][]byte) error {
	if bytes.Contains(fixture, []byte("canonical_source")) {
		return errors.New("fixture publica cuerpo legacy")
	}
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch30Ledger(ledger)
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
	seenRefs, seenCaps, refs, exhausted := map[string]struct{}{}, map[string]struct{}{}, 0, 0
	allEvidence := make([]behaviorBatchEvidence, 0, 16)
	for _, record := range records {
		want, exists := behaviorBatch30Wants[record.Behavior]
		_, duplicate := seenRefs[record.Ref]
		if !exists || duplicate || record.Ref != want.ref || record.CapabilityID != want.capability || record.Authority != "proposal_fixture_not_canonical_ledger" || record.ReviewState != "bootstrap_first_review_pending_independent_counterreview" || record.Disposition != "not_evaluated" || record.CanonicalChange || record.CreatesWork || record.ClosesCapability || record.ClaimsAccreditation || len(record.EntryRefs) != want.count || len(record.Evidence) != want.count || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro inválido: %s", record.Ref)
		}
		text := strings.Join(record.Uncertainties, "\n")
		if want.count == 1 {
			if !strings.Contains(text, "denominator_exhausted") {
				return fmt.Errorf("falta denominator_exhausted: %s", record.Ref)
			}
			exhausted++
		}
		for index, evidence := range record.Evidence {
			entry, ok := entries[evidence.EntryRef]
			_, reused := used[evidence.EntryRef]
			if !ok || reused || record.EntryRefs[index] != evidence.EntryRef || entry.CapabilityDecision != "accept" || entry.SemanticReviewState != "reviewed" || !behaviorBatchEvidenceMatches(record.CapabilityID, evidence, entry.behaviorBatchTaskEntry) {
				return fmt.Errorf("procedencia inválida: %s", evidence.EntryRef)
			}
			used[evidence.EntryRef], refs = struct{}{}, refs+1
			allEvidence = append(allEvidence, evidence)
		}
		seenRefs[record.Ref], seenCaps[record.CapabilityID] = struct{}{}, struct{}{}
	}
	if len(seenRefs) != 10 || len(seenCaps) != 10 || refs != 16 || exhausted != 4 || !behaviorBatch30RangesDisjoint(allEvidence) {
		return fmt.Errorf("refs=%d caps=%d exhausted=%d", refs, len(seenCaps), exhausted)
	}
	return validateBehaviorBatch30Roadmap(roadmap)
}

func behaviorBatch30RangesDisjoint(evidence []behaviorBatchEvidence) bool {
	for left := range evidence {
		for right := left + 1; right < len(evidence); right++ {
			if evidence[left].SourceRef == evidence[right].SourceRef && evidence[left].FirstLine <= evidence[right].LastLine && evidence[right].FirstLine <= evidence[left].LastLine {
				return false
			}
		}
	}
	return true
}

func decodeBehaviorBatch30Ledger(raw []byte) (map[string]behaviorBatch30LedgerEntry, error) {
	result := map[string]behaviorBatch30LedgerEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		var entry behaviorBatch30LedgerEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func validateBehaviorBatch30Roadmap(raw []byte) error {
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
	}{"OPE-01": {"declared", "AC-V30-OPES", 0}, "OPE-05": {"declared", "AC-V30-OPES", 0}, "OPE-08": {"declared", "AC-V30-OPES", 0}, "OPE-11": {"declared", "AC-V30-OPES", 0}, "OPE-17": {"declared", "AC-V30-OPES", 0}, "OPS-02": {"accredited", "AC-V07-CONFIG", 3}, "OPS-06": {"accredited", "AC-V07-CONFIG", 3}, "OPS-08": {"accredited", "AC-V08-CREDENTIALS", 3}, "OPS-12": {"accredited", "AC-V06-ATOMIC-STATE-OUTBOX", 3}, "OPS-17": {"declared", "AC-V32-OPERATIONS-TELEMETRY", 0}}
	for _, entry := range root.Entries {
		want, exists := wants[entry.ID]
		if !exists {
			continue
		}
		if entry.Decision != "accept" || entry.Status != want.status || len(entry.Acceptance) != 1 || entry.Acceptance[0] != want.acceptance || len(entry.Evidence) != want.evidence {
			return fmt.Errorf("roadmap %s inesperado", entry.ID)
		}
		delete(wants, entry.ID)
	}
	if len(wants) != 0 {
		return fmt.Errorf("roadmap incompleto: %v", wants)
	}
	return nil
}

func validateBehaviorBatch30Assessments(raw, fixture, snapshot []byte) error {
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
		if !strings.HasPrefix(ref, "BEHAVIOR-AGENT-BATCH-30-") {
			continue
		}
		mapping := row["function_mapping"].(map[string]any)
		if _, ok := sources[ref]; !ok || row["review_state"] != "independent_bootstrap_counterreview_completed_advisory" || row["authority"] != "advisory_not_capability_state" || mapping["snapshot_census_sha256"] != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" {
			return fmt.Errorf("assessment inválido: %s", ref)
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
	if len(seen) != 10 || linked != 5 || noDirect != 5 {
		return fmt.Errorf("assessments=%d links=%d no_direct=%d", len(seen), linked, noDirect)
	}
	return nil
}

func behaviorBatch30Previous(t *testing.T) [][]byte {
	t.Helper()
	paths, err := filepath.Glob("product/traceability/fixtures/behavior_characterization_agent_batch_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	result := make([][]byte, 0, len(paths))
	for _, path := range paths {
		if strings.HasSuffix(path, "batch_30.jsonl") {
			continue
		}
		result = append(result, behaviorBatch30Read(t, path))
	}
	return result
}
func behaviorBatch30Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func behaviorBatch30Path(env, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(env)); value != "" {
		return value
	}
	return fallback
}
