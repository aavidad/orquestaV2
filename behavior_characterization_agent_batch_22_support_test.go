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

var behaviorBatch22Wants = map[string]struct{ capability, ref string }{
	"auditoria_durable_redactada_y_fallo_visible":           {"GOV-17", "BEHAVIOR-AGENT-BATCH-22-DURABLE-AUDIT"},
	"credential_store_privado_con_owner_scope_y_revocacion": {"OPS-03", "BEHAVIOR-AGENT-BATCH-22-PRIVATE-CREDENTIAL-STORE"},
	"artefactos_causales_por_ref_evidencia_y_ledger":        {"EVD-01", "BEHAVIOR-AGENT-BATCH-22-CAUSAL-ARTIFACTS"},
	"receipt_de_efecto_validado_distinto_de_ack":            {"EVD-03", "BEHAVIOR-AGENT-BATCH-22-EFFECT-RECEIPT"},
	"worktree_aislado_snapshot_y_write_set_verificado":      {"EXT-10", "BEHAVIOR-AGENT-BATCH-22-ISOLATED-WORKTREE"},
	"nucleo_generico_por_puertos_sin_adaptadores_concretos": {"GOV-01", "BEHAVIOR-AGENT-BATCH-22-GENERIC-CORE"},
	"efecto_autorizado_por_permiso_presupuesto_y_riesgo":    {"GOV-15", "BEHAVIOR-AGENT-BATCH-22-EFFECT-GOVERNANCE"},
	"mcp_tools_resources_y_prompts_con_contratos_distintos": {"TLS-02", "BEHAVIOR-AGENT-BATCH-22-MCP-KINDS"},
	"salud_readiness_y_config_efectiva_redactadas":          {"UI-10", "BEHAVIOR-AGENT-BATCH-22-REDACTED-STATUS"},
	"deploy_dry_run_por_puerto_sin_target_implicito":        {"EXT-14", "BEHAVIOR-AGENT-BATCH-22-DEPLOY-PORT"},
}

type behaviorBatch22LedgerEntry struct {
	behaviorBatchTaskEntry
	CapabilityDecision  string `json:"capability_decision"`
	SemanticReviewState string `json:"semantic_review_state"`
}

func validateBehaviorBatch22(fixture, ledger, roadmap []byte, previous [][]byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	entries, err := decodeBehaviorBatch22Ledger(ledger)
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
	seenCaps, seenRefs := map[string]struct{}{}, map[string]struct{}{}
	for _, record := range records {
		want, ok := behaviorBatch22Wants[record.Behavior]
		_, duplicate := seenRefs[record.Ref]
		if !ok || duplicate || record.Ref != want.ref || record.CapabilityID != want.capability ||
			record.Authority != "proposal_fixture_not_canonical_ledger" ||
			record.ReviewState != "bootstrap_first_review_pending_independent_counterreview" ||
			record.Disposition != "not_evaluated" || record.CanonicalChange || record.CreatesWork || record.ClosesCapability || record.ClaimsAccreditation ||
			len(record.EntryRefs) != 2 || len(record.Evidence) != 2 || !behaviorBatchFieldsPresent(record) {
			return fmt.Errorf("registro inválido: %s", record.Ref)
		}
		for index, evidence := range record.Evidence {
			entry, exists := entries[evidence.EntryRef]
			_, reused := used[evidence.EntryRef]
			if !exists || reused || record.EntryRefs[index] != evidence.EntryRef || entry.CapabilityDecision != "accept" ||
				entry.SemanticReviewState != "reviewed" || !behaviorBatchEvidenceMatches(record.CapabilityID, evidence, entry.behaviorBatchTaskEntry) {
				return fmt.Errorf("procedencia inválida: %s", evidence.EntryRef)
			}
			if _, repeated := used[evidence.EntryRef]; repeated {
				return fmt.Errorf("ref repetida: %s", evidence.EntryRef)
			}
			used[evidence.EntryRef] = struct{}{}
		}
		seenRefs[record.Ref], seenCaps[record.CapabilityID] = struct{}{}, struct{}{}
	}
	if len(seenCaps) != 10 || len(seenRefs) != 10 {
		return errors.New("capabilities o refs duplicadas")
	}
	return validateBehaviorBatch22Roadmap(roadmap)
}

func decodeBehaviorBatch22Ledger(raw []byte) (map[string]behaviorBatch22LedgerEntry, error) {
	result := map[string]behaviorBatch22LedgerEntry{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64<<10), 2<<20)
	for scanner.Scan() {
		var entry behaviorBatch22LedgerEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return nil, err
		}
		result[entry.EntryRef] = entry
	}
	return result, scanner.Err()
}

func validateBehaviorBatch22Roadmap(raw []byte) error {
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
	}{
		"GOV-17": {"accredited", "AC-V20-COMMAND-REGISTRY", 3}, "OPS-03": {"accredited", "AC-V08-CREDENTIALS", 3},
		"EVD-01": {"accredited", "AC-V17-TEST-ATTESTOR", 3}, "EVD-03": {"accredited", "AC-V15-BUDGETS-EFFECTS", 3},
		"EXT-10": {"accredited", "AC-V16-WORKSPACE-GIT", 3}, "GOV-01": {"declared", "AC-V33-GENERATED-APPS", 0},
		"GOV-15": {"accredited", "AC-V15-BUDGETS-EFFECTS", 3}, "TLS-02": {"declared", "AC-V26-TOOLS-SKILLS-SDK", 0},
		"UI-10": {"declared", "AC-V24-WEB-ADMIN", 0}, "EXT-14": {"declared", "AC-V29-DEPLOY-NOTIFICATIONS", 0},
	}
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

func validateBehaviorBatch22Assessments(raw, fixture, snapshot []byte) error {
	records, err := decodeBehaviorBatchRecords(fixture)
	if err != nil {
		return err
	}
	sources := map[string]map[string]struct{}{}
	for _, record := range records {
		sources[record.Ref] = map[string]struct{}{}
		for _, ev := range record.Evidence {
			sources[record.Ref][ev.SourceRef] = struct{}{}
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
	seen, linked := map[string]struct{}{}, 0
	for lineNo, line := range bytes.Split(bytes.TrimSuffix(raw, []byte("\n")), []byte("\n")) {
		var row map[string]any
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.UseNumber()
		if err := decoder.Decode(&row); err != nil {
			return fmt.Errorf("assessment %d: %w", lineNo+1, err)
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			return errors.New("valor posterior")
		}
		var compact bytes.Buffer
		if err := json.Compact(&compact, line); err != nil || !bytes.Equal(compact.Bytes(), line) {
			return fmt.Errorf("assessment %d no canónico", lineNo+1)
		}
		ref, _ := row["characterization_ref"].(string)
		if !strings.HasPrefix(ref, "BEHAVIOR-AGENT-BATCH-22-") {
			continue
		}
		mapping, _ := row["function_mapping"].(map[string]any)
		if _, ok := sources[ref]; !ok || row["review_state"] != "independent_bootstrap_counterreview_completed_advisory" || row["authority"] != "advisory_not_capability_state" || mapping["snapshot_census_sha256"] != "sha256:5e8f30dc3c6640c57f595e3757cc6e07ba4be6eb36a6bcdd094152a2967fddfc" {
			return fmt.Errorf("assessment inválido: %s", ref)
		}
		links, _ := mapping["links"].([]any)
		if ref == "BEHAVIOR-AGENT-BATCH-22-PRIVATE-CREDENTIAL-STORE" {
			if mapping["state"] != "reviewed_no_direct_function" || len(links) != 0 {
				return errors.New("OPS-03 debe quedar sin función directa")
			}
		} else {
			if mapping["state"] != "linked_exact" || len(links) < 1 || len(links) > 2 {
				return fmt.Errorf("links inválidos: %s", ref)
			}
			linked += len(links)
			for _, value := range links {
				link := value.(map[string]any)
				occurrence, _ := link["occurrence_ref"].(string)
				exact, exists := snapshotByOccurrence[occurrence]
				if !exists || link["review_state"] != "bootstrap_symbol_mapping_reviewed_advisory" {
					return fmt.Errorf("occurrence no exacta: %s", occurrence)
				}
				for _, key := range []string{"declaration_ref", "variant_ref", "source_path", "git_blob_oid", "signature", "ast_sha256", "body_sha256"} {
					if link[key] != exact[key] {
						return fmt.Errorf("drift %s en %s", key, occurrence)
					}
				}
				for _, ev := range link["evidence_refs"].([]any) {
					if _, ok := sources[ref][ev.(string)]; !ok {
						return fmt.Errorf("evidencia ajena: %s", ev)
					}
				}
			}
		}
		if _, duplicate := seen[ref]; duplicate {
			return fmt.Errorf("assessment duplicado: %s", ref)
		}
		seen[ref] = struct{}{}
	}
	if len(seen) != 10 || linked != 9 {
		return fmt.Errorf("assessments=%d links=%d", len(seen), linked)
	}
	return nil
}

func behaviorBatch22Previous(t *testing.T) [][]byte {
	t.Helper()
	result := make([][]byte, 0, 21)
	for index := 1; index <= 21; index++ {
		result = append(result, behaviorBatch22Read(t, fmt.Sprintf("product/traceability/fixtures/behavior_characterization_agent_batch_%02d.jsonl", index)))
	}
	return result
}

func behaviorBatch22Read(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func behaviorBatch22Path(env, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(env)); value != "" {
		return value
	}
	return fallback
}
