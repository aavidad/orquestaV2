// Validación estructural estricta del manifiesto: cada clave pública debe
// estar presente y ningún cero implícito puede fingir una decisión explícita.
package orquesta_test

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func legacyRootsReadAndCheckKeys(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(legacySourceRootsPath)
	if err != nil {
		t.Fatal(err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatal(err)
	}
	legacyRequireKeys(t, top, "document_kind", "schema_version", "assembled_at", "closed", "closure_reason", "source_universe_statement", "logical_path_policy", "vocabularies", "observation_method", "observation_batches", "unaccredited_historical_reports", "ignored_entries_advisory", "root_ids_sha256", "roots", "git_facts", "file_facts", "collections", "v1_v22_tags", "git_object_reconciliation", "unresolved_source_decision_root_ids", "pending_observation_batch_ids", "blocking_physical_census_root_ids")
	legacyRequireKeys(t, legacyRawObject(t, top["logical_path_policy"]), "root_refs", "local_mapping_tracked", "profile_ids_published", "rule")
	legacyRequireKeys(t, legacyRawObject(t, top["vocabularies"]), "root_classes", "scope_kinds", "natures", "source_decisions", "git_census_statuses", "physical_census_statuses", "git_object_coverages")
	method := legacyRawObject(t, top["observation_method"])
	legacyRequireKeys(t, method, "git_version", "commands", "semantics", "compared_ref_sets")
	legacyRequireKeys(t, legacyRawObject(t, method["semantics"]), "top_level_candidates", "dirty_entry_count", "ref_count", "commit_count", "worktree_membership", "legacy_principal_worktree_reconciliation", "vec_worktree_membership", "external_metadata", "script_digest", "reconciliation")
	legacyRequireArrayKeys(t, method["compared_ref_sets"], "left_root_id", "right_root_id", "namespace")
	for _, batch := range legacyRawArray(t, top["observation_batches"]) {
		legacyRequireKeys(t, batch, "id", "kind", "started_at", "finished_at", "time_precision", "status", "raw_census_sha256", "reason", "metrics", "worktree_counts")
		legacyRequireArrayKeys(t, batch["worktree_counts"], "common_git_dir_ref", "total_count", "includes_main", "additional_member_count", "dirty_total_count", "dirty_additional_member_count")
	}
	legacyRequireArrayKeys(t, top["unaccredited_historical_reports"], "id", "observation_batch_id", "subject_root_id", "reported_total_count", "includes_main", "reported_dirty_total_count", "evidence_status", "reason")
	legacyRequireKeys(t, legacyRawObject(t, top["ignored_entries_advisory"]), "observation_batch_id", "status", "affected_root_count", "ignored_entry_count", "reason")
	rootKeys := []string{"id", "logical_root", "relative_path", "exists_at_observation", "class", "scope_kind", "nature", "source_decision", "source_decision_reason", "observation_batch_id", "git_census_status", "git_census_status_reason", "physical_census_status", "physical_census_status_reason", "physical_census_required", "git_object_coverage", "evidence_refs"}
	for _, root := range legacyRawArray(t, top["roots"]) {
		legacyRequireKeys(t, root, rootKeys...)
		for _, key := range rootKeys {
			if string(root[key]) == "null" {
				t.Fatalf("la raíz %s omite mediante null el campo obligatorio %q", root["id"], key)
			}
		}
	}
	legacyRequireArrayKeys(t, top["git_facts"], "root_id", "common_git_dir_ref", "head_oid", "ref_count", "commit_count", "reachable_only_from_root", "dirty_entry_count", "prunable", "observation_batch_id")
	legacyRequireArrayKeys(t, top["file_facts"], "root_id", "sha256", "byte_size", "verification", "observation_batch_id")
	collections := legacyRawArray(t, top["collections"])
	for _, collection := range collections {
		legacyRequireKeys(t, collection, "root_id", "members_sha256", "members", "observation_batch_id")
		legacyRequireArrayKeys(t, collection["members"], "path_alias", "exists_at_observation", "nature", "head_oid", "common_git_dir_ref", "dirty_entry_count", "prunable", "sha256", "byte_size")
	}
	legacyRequireArrayKeys(t, top["v1_v22_tags"], "name", "tag_object_oid", "commit_oid", "observed_root_ids", "observation_batch_id")
	legacyRequireArrayKeys(t, top["git_object_reconciliation"], "root_id", "reachable_only_from_root", "contained_in_current_product", "evidence_status", "reason", "observation_batch_id")
	return raw
}

func legacyRawObject(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func legacyRawArray(t *testing.T, raw json.RawMessage) []map[string]json.RawMessage {
	t.Helper()
	var values []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		t.Fatal(err)
	}
	return values
}

func legacyRequireArrayKeys(t *testing.T, raw json.RawMessage, want ...string) {
	t.Helper()
	for _, value := range legacyRawArray(t, raw) {
		legacyRequireKeys(t, value, want...)
	}
}

func legacyRequireKeys(t *testing.T, raw map[string]json.RawMessage, want ...string) {
	t.Helper()
	got := make([]string, 0, len(raw))
	for key := range raw {
		got = append(got, key)
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("claves JSON inesperadas: obtenidas=%v esperadas=%v", got, want)
	}
}

func legacyRootsValidateBatches(t *testing.T, document legacyRootsDocument) map[string]legacyRootsBatch {
	t.Helper()
	if len(document.ObservationBatches) != 7 {
		t.Fatalf("lotes multitemporales incompletos: %d", len(document.ObservationBatches))
	}
	assembled, err := time.Parse(time.RFC3339, document.AssembledAt)
	if err != nil {
		t.Fatalf("instante de ensamblado inválido: %v", err)
	}
	batches := make(map[string]legacyRootsBatch, len(document.ObservationBatches))
	var pending []string
	counts := make(map[string]legacyRootsWorktrees)
	for _, batch := range document.ObservationBatches {
		if batch.ID == "" || batch.Kind == "" || batch.Status != "pendiente" || batch.RawCensusSHA256 != nil ||
			batch.TimePrecision == "" || strings.TrimSpace(batch.Reason) == "" {
			t.Fatalf("lote falsamente cerrado o incompleto: %#v", batch)
		}
		if _, duplicate := batches[batch.ID]; duplicate {
			t.Fatalf("lote duplicado: %q", batch.ID)
		}
		if batch.TimePrecision == "desconocida" {
			if batch.StartedAt != nil || batch.FinishedAt != nil {
				t.Fatalf("lote de tiempo desconocido con instante inventado: %q", batch.ID)
			}
		} else {
			if batch.StartedAt == nil || batch.FinishedAt == nil {
				t.Fatalf("lote fechado sin ventana completa: %q", batch.ID)
			}
			start, startErr := time.Parse(time.RFC3339, *batch.StartedAt)
			finish, finishErr := time.Parse(time.RFC3339, *batch.FinishedAt)
			if startErr != nil || finishErr != nil || finish.Before(start) || assembled.Before(finish) {
				t.Fatalf("ventana temporal inválida en %q", batch.ID)
			}
		}
		for name, value := range batch.Metrics {
			if name == "" || value < 0 {
				t.Fatalf("métrica inválida en %q", batch.ID)
			}
		}
		for _, count := range batch.WorktreeCounts {
			if count.CommonGitDirRef == "" || !count.IncludesMain || count.TotalCount != count.AdditionalMemberCount+1 {
				t.Fatalf("recuento sin semántica de raíz en %q: %#v", batch.ID, count)
			}
			if count.DirtyTotalCount != nil && (*count.DirtyTotalCount < 0 || *count.DirtyTotalCount > count.TotalCount) ||
				count.DirtyAdditionalMemberCount != nil && (*count.DirtyAdditionalMemberCount < 0 || *count.DirtyAdditionalMemberCount > count.AdditionalMemberCount) {
				t.Fatalf("suciedad incoherente en %q: %#v", batch.ID, count)
			}
			if _, duplicate := counts[count.CommonGitDirRef]; duplicate {
				t.Fatalf("recuento repetido para %q", count.CommonGitDirRef)
			}
			counts[count.CommonGitDirRef] = count
		}
		batches[batch.ID] = batch
		pending = append(pending, batch.ID)
	}
	legacyRequireSameStrings(t, pending, document.PendingObservationBatchIDs, "lotes pendientes")
	want := map[string][2]int{
		"legacy_principal":      {153, 152},
		"producto_actual":       {47, 46},
		"legacy_autonomia":      {5, 4},
		"legacy_copia_temprana": {14, 13},
		"consumidor_municipal":  {3, 2},
		"vec_principal":         {89, 88},
	}
	for ref, expected := range want {
		got, exists := counts[ref]
		if !exists || got.TotalCount != expected[0] || got.AdditionalMemberCount != expected[1] {
			t.Fatalf("recuento total/adicional incorrecto para %q: %#v", ref, got)
		}
	}
	vec := counts["vec_principal"]
	if vec.DirtyTotalCount == nil || *vec.DirtyTotalCount != 2 ||
		vec.DirtyAdditionalMemberCount == nil || *vec.DirtyAdditionalMemberCount != 1 {
		t.Fatalf("la observación viva VEC no conserva 89/2: %#v", vec)
	}
	if len(document.UnaccreditedHistoricalReports) != 1 {
		t.Fatal("falta el informe histórico adversarial VEC")
	}
	report := document.UnaccreditedHistoricalReports[0]
	if report.ID != "vec_88_3" || report.ReportedTotalCount != 88 || report.ReportedDirtyTotalCount != 3 ||
		!report.IncludesMain || report.EvidenceStatus != "no_acreditado" || batches[report.ObservationBatchID].ID == "" {
		t.Fatalf("informe VEC 88/3 acreditado o deformado: %#v", report)
	}
	ignored := document.IgnoredEntriesAdvisory
	if ignored.Status != "pendiente" || ignored.AffectedRootCount != 8 || ignored.IgnoredEntryCount != 18 ||
		batches[ignored.ObservationBatchID].ID == "" || strings.TrimSpace(ignored.Reason) == "" {
		t.Fatalf("negativo de ignorados incompleto: %#v", ignored)
	}
	return batches
}

func legacyRootsObservedCount(t *testing.T, document legacyRootsDocument, ref string) legacyRootsWorktrees {
	t.Helper()
	for _, batch := range document.ObservationBatches {
		for _, count := range batch.WorktreeCounts {
			if count.CommonGitDirRef == ref {
				return count
			}
		}
	}
	t.Fatalf("falta recuento para %q", ref)
	return legacyRootsWorktrees{}
}
