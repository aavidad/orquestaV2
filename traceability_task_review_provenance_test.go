package orquesta_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const v2SemanticReviewBugRef = "product/traceability/rebuild_bugs.jsonl#BUG-REBUILD-20260714-008"

type v2TaskReviewProvenance struct {
	DocumentKind        string                     `json:"document_kind"`
	SchemaVersion       int                        `json:"schema_version"`
	EntriesAuthority    string                     `json:"entries_authority"`
	CapabilityAuthority string                     `json:"capability_authority"`
	BugRef              string                     `json:"bug_ref"`
	Passes              []v2TaskReviewPass         `json:"passes"`
	Invalidations       []v2TaskReviewInvalidation `json:"invalidations"`
	Totals              struct {
		ValidPassCount        int    `json:"valid_pass_count"`
		EntryCount            int    `json:"entry_count"`
		ConfirmedCount        int    `json:"confirmed_count"`
		CorrectedCount        int    `json:"corrected_count"`
		InvalidatedPassCount  int    `json:"invalidated_pass_count"`
		InvalidatedEntryCount int    `json:"invalidated_entry_count"`
		InputUniverseSHA256   string `json:"input_universe_sha256"`
		FinalDecisionsSHA256  string `json:"final_decisions_sha256"`
	} `json:"totals"`
}

type v2TaskReviewPass struct {
	ScopeCode            string `json:"scope_code"`
	PassRef              string `json:"pass_ref"`
	ReviewerRef          string `json:"reviewer_ref"`
	ReviewerRole         string `json:"reviewer_role"`
	ReviewMethod         string `json:"review_method"`
	EntryCount           int    `json:"entry_count"`
	InputScopeSHA256     string `json:"input_scope_sha256"`
	ConfirmedCount       int    `json:"confirmed_count"`
	CorrectedCount       int    `json:"corrected_count"`
	FinalDecisionsSHA256 string `json:"final_decisions_sha256"`
}

type v2TaskReviewInvalidation struct {
	InvalidationRef          string `json:"invalidation_ref"`
	ScopeCode                string `json:"scope_code"`
	AffectedEntryCount       int    `json:"affected_entry_count"`
	AffectedInputScopeSHA256 string `json:"affected_input_scope_sha256"`
	ReasonCode               string `json:"reason_code"`
	DetectorTestRef          string `json:"detector_test_ref"`
	SupersededByPassRef      string `json:"superseded_by_pass_ref"`
}

type v2TaskReviewExpected struct {
	Entries   int
	Confirmed int
	Corrected int
}

func TestTraceabilityRebuildTaskSemanticReviewProvenance(t *testing.T) {
	var provenance v2TaskReviewProvenance
	v2DecodeStrict(t, v2TaskProvenancePath, &provenance)
	if provenance.DocumentKind != "task_semantic_review_provenance" || provenance.SchemaVersion != 1 ||
		provenance.EntriesAuthority != v2TaskEntriesPath+"#basis_code=independent_semantic_review" ||
		provenance.CapabilityAuthority != "product/roadmap.json#capability_entries" ||
		provenance.BugRef != v2SemanticReviewBugRef {
		t.Fatalf("invalid task semantic review provenance header: %#v", provenance)
	}

	expected := map[string]v2TaskReviewExpected{
		"original_corrected_all":     {Entries: 82, Confirmed: 7, Corrected: 75},
		"original_confirmed_shard_1": {Entries: 189, Confirmed: 153, Corrected: 36},
		"original_confirmed_shard_2": {Entries: 164, Confirmed: 151, Corrected: 13},
		"original_confirmed_shard_3": {Entries: 261, Confirmed: 236, Corrected: 25},
	}
	entries := v2ReadJSONL[v2TaskEntry](t, v2TaskEntriesPath)
	byPass := make(map[string][]v2TaskEntry)
	var independent []v2TaskEntry
	for _, entry := range entries {
		if entry.BasisCode == "independent_semantic_review" {
			independent = append(independent, entry)
			byPass[entry.SemanticReviewPassRef] = append(byPass[entry.SemanticReviewPassRef], entry)
		} else if entry.SemanticReviewPassRef != "" && entry.BasisCode != "independent_candidate_semantic_review" {
			t.Fatalf("non-independent task entry %s carries pass %s", entry.EntryRef, entry.SemanticReviewPassRef)
		}
	}
	if len(independent) != 696 {
		t.Fatalf("independent provenance universe=%d, want 696", len(independent))
	}

	passByScope := make(map[string]v2TaskReviewPass, len(provenance.Passes))
	seenEntries := make(map[string]string, len(independent))
	expectedOrder := []string{"original_corrected_all", "original_confirmed_shard_1", "original_confirmed_shard_2", "original_confirmed_shard_3"}
	for passIndex, pass := range provenance.Passes {
		want, known := expected[pass.ScopeCode]
		if !known || passIndex >= len(expectedOrder) || pass.ScopeCode != expectedOrder[passIndex] {
			t.Fatalf("unknown or unordered review pass scope %q at index %d", pass.ScopeCode, passIndex)
		}
		if _, duplicate := passByScope[pass.ScopeCode]; duplicate {
			t.Fatalf("duplicate review pass scope %q", pass.ScopeCode)
		}
		if !strings.HasPrefix(pass.ReviewerRef, "TRACE-REVIEWER-") || len(pass.ReviewerRef) != len("TRACE-REVIEWER-")+24 ||
			pass.ReviewerRole != "independent_semantic_counterreviewer" ||
			pass.ReviewMethod != "source_block_and_roadmap_read_per_entry" {
			t.Fatalf("review pass %q lacks opaque reviewer/method provenance: %#v", pass.ScopeCode, pass)
		}
		rows := append([]v2TaskEntry(nil), byPass[pass.PassRef]...)
		sort.Slice(rows, func(i, j int) bool { return rows[i].EntryRef < rows[j].EntryRef })
		for _, entry := range rows {
			if prior, duplicate := seenEntries[entry.EntryRef]; duplicate {
				t.Fatalf("entry %s appears in review passes %s and %s", entry.EntryRef, prior, pass.ScopeCode)
			}
			seenEntries[entry.EntryRef] = pass.ScopeCode
		}
		inputDigest := v2TaskReviewInputDigest(t, rows)
		passSubject := []any{
			"task_semantic_review_pass_v1", pass.ScopeCode, pass.ReviewerRef, pass.ReviewerRole,
			pass.ReviewMethod, len(rows), inputDigest,
		}
		wantPassRef := "TASKREVIEWPASS-" + v2TaskReviewJSONDigest(t, passSubject)[7:31]
		confirmed, corrected := v2TaskReviewVerdictCounts(rows)
		if pass.PassRef != wantPassRef || pass.EntryCount != want.Entries || len(rows) != want.Entries ||
			pass.InputScopeSHA256 != inputDigest || pass.ConfirmedCount != want.Confirmed || confirmed != want.Confirmed ||
			pass.CorrectedCount != want.Corrected || corrected != want.Corrected ||
			pass.FinalDecisionsSHA256 != v2TaskReviewFinalDigest(t, rows) {
			t.Fatalf("review pass %q drift: got=%#v rows=%d confirmed=%d corrected=%d", pass.ScopeCode, pass, len(rows), confirmed, corrected)
		}
		passByScope[pass.ScopeCode] = pass
		delete(byPass, pass.PassRef)
	}
	if len(passByScope) != len(expected) || len(byPass) != 0 || len(seenEntries) != len(independent) {
		t.Fatalf("review pass partition incomplete: passes=%d unknown_passes=%v entries=%d", len(passByScope), sortedTaskReviewKeys(byPass), len(seenEntries))
	}

	sort.Slice(independent, func(i, j int) bool { return independent[i].EntryRef < independent[j].EntryRef })
	confirmed, corrected := v2TaskReviewVerdictCounts(independent)
	if provenance.Totals.ValidPassCount != 4 || provenance.Totals.EntryCount != 696 ||
		provenance.Totals.ConfirmedCount != 547 || confirmed != 547 ||
		provenance.Totals.CorrectedCount != 149 || corrected != 149 ||
		provenance.Totals.InvalidatedPassCount != 2 || provenance.Totals.InvalidatedEntryCount != 450 ||
		provenance.Totals.InputUniverseSHA256 != v2TaskReviewInputDigest(t, independent) ||
		provenance.Totals.FinalDecisionsSHA256 != v2TaskReviewFinalDigest(t, independent) {
		t.Fatalf("semantic review provenance totals drift: %#v confirmed=%d corrected=%d", provenance.Totals, confirmed, corrected)
	}

	wantInvalidationScopes := []string{"original_confirmed_shard_1", "original_confirmed_shard_3"}
	gotInvalidationScopes := make([]string, 0, len(provenance.Invalidations))
	for _, invalidation := range provenance.Invalidations {
		pass, known := passByScope[invalidation.ScopeCode]
		if !known {
			t.Fatalf("invalidation points to unknown scope %q", invalidation.ScopeCode)
		}
		const reason = "generated_keep_current_fallback"
		const detector = "traceability_tasks_v2_test.go#TestTraceabilityRebuildIndependentSemanticReviewsHaveNoGeneratedFallbacks"
		subject := []any{
			"task_semantic_review_invalidation_v1", v2SemanticReviewBugRef, invalidation.ScopeCode,
			pass.EntryCount, pass.InputScopeSHA256, reason, pass.PassRef,
		}
		wantRef := "TASKREVIEWINVALID-" + v2TaskReviewJSONDigest(t, subject)[7:31]
		if invalidation.InvalidationRef != wantRef || invalidation.AffectedEntryCount != pass.EntryCount ||
			invalidation.AffectedInputScopeSHA256 != pass.InputScopeSHA256 || invalidation.ReasonCode != reason ||
			invalidation.DetectorTestRef != detector || invalidation.SupersededByPassRef != pass.PassRef {
			t.Fatalf("invalid semantic review invalidation for %s: %#v", invalidation.ScopeCode, invalidation)
		}
		gotInvalidationScopes = append(gotInvalidationScopes, invalidation.ScopeCode)
	}
	sort.Strings(gotInvalidationScopes)
	if !reflect.DeepEqual(gotInvalidationScopes, wantInvalidationScopes) {
		t.Fatalf("invalidated scopes=%v, want %v", gotInvalidationScopes, wantInvalidationScopes)
	}
	traceRequireRebuildBug(t, "BUG-REBUILD-20260714-008", "closed")
}

func v2TaskReviewInputDigest(t *testing.T, entries []v2TaskEntry) string {
	t.Helper()
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetEscapeHTML(false)
	for _, entry := range entries {
		if err := encoder.Encode([]any{entry.EntryRef, entry.SourceRef, entry.SourceLine, entry.SubjectSHA256, entry.PreviousCapabilityID}); err != nil {
			t.Fatal(err)
		}
	}
	return v2SHA(content.Bytes())
}

func v2TaskReviewFinalDigest(t *testing.T, entries []v2TaskEntry) string {
	t.Helper()
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetEscapeHTML(false)
	for _, entry := range entries {
		if err := encoder.Encode([]any{
			entry.EntryRef, entry.PreviousCapabilityID, entry.CapabilityID, entry.SemanticReviewVerdict,
			entry.SemanticReviewPassRef, entry.SemanticReviewRef, entry.SemanticReason,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return v2SHA(content.Bytes())
}

func v2TaskReviewJSONDigest(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return v2SHA(data)
}

func v2TaskReviewVerdictCounts(entries []v2TaskEntry) (confirmed, corrected int) {
	for _, entry := range entries {
		switch entry.SemanticReviewVerdict {
		case "confirmed":
			confirmed++
		case "corrected":
			corrected++
		}
	}
	return confirmed, corrected
}

func sortedTaskReviewKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func traceRequireRebuildBug(t *testing.T, bugID, status string) {
	t.Helper()
	for _, bug := range traceReadRebuildBugs(t, "product/traceability/rebuild_bugs.jsonl") {
		if bug.BugID == bugID {
			if bug.Status != status {
				t.Fatalf("rebuild bug %s status=%s, want %s", bugID, bug.Status, status)
			}
			return
		}
	}
	t.Fatalf("rebuild bug %s is absent", bugID)
}
