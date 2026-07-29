package orquesta_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type traceHistoricalBugExtractionPolicy struct {
	DocumentKind                     string `json:"document_kind"`
	SchemaVersion                    int    `json:"schema_version"`
	SourcesAuthority                 string `json:"sources_authority"`
	RichRowsAuthority                string `json:"rich_rows_authority"`
	OccurrencesAuthority             string `json:"occurrences_authority"`
	NarrativeReviewsAuthority        string `json:"narrative_reviews_authority"`
	NarrativeReviewBindingsAuthority string `json:"narrative_review_bindings_authority"`
	RichRowEnrichmentsAuthority      string `json:"rich_row_enrichments_authority"`
	IDsAuthority                     string `json:"ids_authority"`
	DetectionRule                    struct {
		BasePattern           string   `json:"base_pattern"`
		SlashComponentPattern string   `json:"slash_component_pattern"`
		SlashExpansion        string   `json:"slash_expansion"`
		LetterSuffixExpansion string   `json:"letter_suffix_expansion"`
		LetterRangeExpansion  string   `json:"letter_range_expansion"`
		RowCoverage           string   `json:"row_coverage"`
		RichRowSourceRefs     []string `json:"rich_row_source_refs"`
	} `json:"detection_rule"`
	HashRule struct {
		Algorithm            string `json:"algorithm"`
		TextSubject          string `json:"text_subject"`
		OccurrenceRefSubject string `json:"occurrence_ref_subject"`
		EntryRefSubject      string `json:"entry_ref_subject"`
	} `json:"hash_rule"`
	CapabilityAuthority string `json:"capability_authority"`
	ClosurePolicy       string `json:"closure_policy"`
	Baseline            struct {
		SourceCount                   int    `json:"source_count"`
		SourcesWithExplicitID         int    `json:"sources_with_explicit_id"`
		SourcesWithoutExplicitID      int    `json:"sources_without_explicit_id"`
		SourcesWithoutIDLessonsSHA256 string `json:"sources_without_id_lessons_sha256"`
		RichRowCount                  int    `json:"rich_row_count"`
		OccurrenceCount               int    `json:"occurrence_count"`
		IDCount                       int    `json:"id_count"`
		RowCoveredIDCount             int    `json:"row_covered_id_count"`
		NarrativeOnlyIDCount          int    `json:"narrative_only_id_count"`
		NarrativeReviewBindingCount   int    `json:"narrative_review_binding_count"`
		RichRowEnrichmentCount        int    `json:"rich_row_enrichment_count"`
		RowsSHA256                    string `json:"rows_sha256"`
		OccurrencesSHA256             string `json:"occurrences_sha256"`
		NarrativeReviewBindingsSHA256 string `json:"narrative_review_bindings_sha256"`
		RichRowEnrichmentsSHA256      string `json:"rich_row_enrichments_sha256"`
		IDsSHA256                     string `json:"ids_sha256"`
	} `json:"baseline"`
}

type traceHistoricalBugRow struct {
	SchemaVersion           int      `json:"schema_version"`
	EntryRef                string   `json:"entry_ref"`
	SourceRef               string   `json:"source_ref"`
	SourceSection           string   `json:"source_section"`
	SourceLine              int      `json:"source_line"`
	DetectionKind           string   `json:"detection_kind"`
	TextSHA256              string   `json:"text_sha256"`
	SourceBugID             string   `json:"source_bug_id"`
	ReportedIDs             []string `json:"reported_ids"`
	DuplicateIDOccurrence   int      `json:"duplicate_id_occurrence"`
	DuplicateIDCount        int      `json:"duplicate_id_count"`
	OriginalState           string   `json:"original_state"`
	Area                    string   `json:"area"`
	Symptom                 string   `json:"symptom"`
	ArchitecturalHypothesis string   `json:"architectural_hypothesis"`
	Action                  string   `json:"action"`
	CapabilityIDs           []string `json:"capability_ids"`
	Disposition             string   `json:"disposition"`
	LessonTestRef           string   `json:"lesson_test_ref"`
	CitedExistingTestRefs   []string `json:"cited_existing_test_refs"`
	ClosureEvidence         string   `json:"closure_evidence"`
	ReviewNote              string   `json:"review_note"`
}

type traceHistoricalBugOccurrence struct {
	SchemaVersion  int    `json:"schema_version"`
	OccurrenceRef  string `json:"occurrence_ref"`
	BugID          string `json:"bug_id"`
	RawBugRef      string `json:"raw_bug_ref"`
	ComponentIndex int    `json:"component_index"`
	SourceRef      string `json:"source_ref"`
	SourceLine     int    `json:"source_line"`
	SourceColumn   int    `json:"source_column"`
	TextSHA256     string `json:"text_sha256"`
	CoverageKind   string `json:"coverage_kind"`
}

type traceHistoricalBugID struct {
	SchemaVersion         int      `json:"schema_version"`
	BugID                 string   `json:"bug_id"`
	CoverageKind          string   `json:"coverage_kind"`
	CapabilityIDs         []string `json:"capability_ids"`
	VerifiedCapabilityIDs []string `json:"verified_capability_ids,omitempty"`
	RebuildEvidenceRefs   []string `json:"rebuild_evidence_refs,omitempty"`
	Disposition           string   `json:"disposition"`
	LessonTestRef         string   `json:"lesson_test_ref"`
	LessonState           string   `json:"lesson_state"`
	OccurrenceCount       int      `json:"occurrence_count"`
	OccurrencesSHA256     string   `json:"occurrences_sha256"`
	ClosureEvidence       string   `json:"closure_evidence"`
	ReviewNote            string   `json:"review_note"`
}

type traceHistoricalBugIDReview struct {
	SchemaVersion int      `json:"schema_version"`
	BugID         string   `json:"bug_id"`
	CapabilityIDs []string `json:"capability_ids"`
	ReviewNote    string   `json:"review_note"`
}

func TestTraceabilityRebuildHistoricalBugIDs(t *testing.T) {
	var policy traceHistoricalBugExtractionPolicy
	traceDecodeStrict(t, "product/traceability/historical_bug_extraction_policy.json", &policy)
	if policy.DocumentKind != "historical_bug_id_extraction_policy" || policy.SchemaVersion != 1 ||
		policy.SourcesAuthority != "product/traceability/source_dispositions.jsonl" ||
		policy.RichRowsAuthority != "product/traceability/historical_bug_rows.jsonl" ||
		policy.OccurrencesAuthority != "product/traceability/historical_bug_occurrences.jsonl" ||
		policy.NarrativeReviewsAuthority != "product/traceability/historical_bug_id_reviews.jsonl" ||
		policy.NarrativeReviewBindingsAuthority != "product/traceability/historical_bug_id_review_bindings.jsonl" ||
		policy.RichRowEnrichmentsAuthority != "product/traceability/historical_bug_row_enrichments.jsonl" ||
		policy.IDsAuthority != "product/traceability/historical_bug_ids.jsonl" ||
		policy.CapabilityAuthority != "product/roadmap.json#capability_entries" ||
		policy.ClosurePolicy != "historical_status_and_cited_tests_never_imply_rebuild_closure" ||
		policy.DetectionRule.BasePattern != `\bBUG-ORQ(?:-[A-Z0-9]+)+` ||
		policy.DetectionRule.SlashComponentPattern != `^/[A-Z0-9]+` ||
		policy.HashRule.Algorithm != "sha256" ||
		policy.HashRule.TextSubject != "exact_source_line_bytes_without_line_ending" ||
		policy.HashRule.EntryRefSubject != "source_ref_newline_source_line_newline_text_sha256" ||
		policy.HashRule.OccurrenceRefSubject != "bug_id_newline_raw_bug_ref_newline_component_index_newline_source_ref_newline_source_line_newline_source_column_newline_text_sha256" ||
		policy.DetectionRule.SlashExpansion != "reuse_everything_before_the_last_hyphen" ||
		policy.DetectionRule.LetterSuffixExpansion != "reuse_leading_digits_from_the_first_suffix" ||
		policy.DetectionRule.LetterRangeExpansion != "expand_compact_same_numeric_prefix_inclusive_ascii_range" ||
		policy.DetectionRule.RowCoverage != "only_the_expanded_first_cell_id_of_a_reviewed_rich_inventory_row" ||
		!reflect.DeepEqual(policy.DetectionRule.RichRowSourceRefs, []string{
			"docs/incidencias/incidencia_orquesta_invariantes_causales_nucleo_2026-07-11.md",
			"docs/inventario_bugs_orquesta_2026-06-30.md",
		}) {
		t.Fatalf("invalid historical bug extraction policy: %#v", policy)
	}
	basePattern, err := regexp.Compile(policy.DetectionRule.BasePattern)
	if err != nil {
		t.Fatalf("compile historical bug base pattern: %v", err)
	}
	slashPattern, err := regexp.Compile(policy.DetectionRule.SlashComponentPattern)
	if err != nil {
		t.Fatalf("compile historical bug slash pattern: %v", err)
	}
	legacySources := traceLoadLegacySourceSnapshot(t, ".")

	accepted := traceAcceptedCapabilities(t)
	sourceRows := traceReadDispositionJSONL(t, policy.SourcesAuthority)
	bugSources := make(map[string]traceSourceDisposition)
	for _, source := range sourceRows {
		if traceSourceHasRole(source, "bug") {
			bugSources[source.SourceRef] = source
		}
	}
	if len(bugSources) != policy.Baseline.SourceCount {
		t.Fatalf("historical bug source count=%d, want %d", len(bugSources), policy.Baseline.SourceCount)
	}

	rows := traceReadHistoricalBugRows(t, policy.RichRowsAuthority)
	rowsByLine := traceValidateHistoricalBugRows(t, rows, bugSources, accepted, policy.DetectionRule.RichRowSourceRefs, legacySources)
	rowJSON := traceMarshalJSONLines(t, rows)
	if len(rows) != policy.Baseline.RichRowCount || traceStringsDigest(rowJSON) != policy.Baseline.RowsSHA256 {
		t.Fatalf("historical bug rich-row baseline drift: rows=%d digest=%s, want rows=%d digest=%s",
			len(rows), traceStringsDigest(rowJSON), policy.Baseline.RichRowCount, policy.Baseline.RowsSHA256)
	}

	extractedOccurrences := traceExtractHistoricalBugOccurrences(t, bugSources, rowsByLine, basePattern, slashPattern, legacySources)
	sourcesWithID := make(map[string]struct{})
	for _, occurrence := range extractedOccurrences {
		sourcesWithID[occurrence.SourceRef] = struct{}{}
	}
	sourceOnlyLessonLines := make([]string, 0, len(bugSources)-len(sourcesWithID))
	for sourceRef, source := range bugSources {
		if _, hasID := sourcesWithID[sourceRef]; hasID {
			continue
		}
		if source.LessonID == "" || source.InvariantTestRef == "" || source.LessonState != "pending_invariant_test" {
			t.Fatalf("bug source without explicit BUG-ORQ ID lacks source-level lesson: %#v", source)
		}
		sourceOnlyLessonLines = append(sourceOnlyLessonLines, strings.Join([]string{
			source.SourceRef,
			source.SourceSHA256,
			source.LessonID,
			source.InvariantTestRef,
			source.LessonState,
		}, "|"))
	}
	sort.Strings(sourceOnlyLessonLines)
	if len(sourcesWithID) != policy.Baseline.SourcesWithExplicitID ||
		len(sourceOnlyLessonLines) != policy.Baseline.SourcesWithoutExplicitID ||
		traceStringsDigest(sourceOnlyLessonLines) != policy.Baseline.SourcesWithoutIDLessonsSHA256 {
		t.Fatalf("historical bug source-ID boundary drift: with_id=%d without_id=%d source_lessons_digest=%s",
			len(sourcesWithID), len(sourceOnlyLessonLines), traceStringsDigest(sourceOnlyLessonLines))
	}
	if os.Getenv("ORQUESTA_TRACEABILITY_EMIT_HISTORICAL_BUG_OCCURRENCES") == "1" {
		encoder := json.NewEncoder(os.Stdout)
		for _, occurrence := range extractedOccurrences {
			if err := encoder.Encode(occurrence); err != nil {
				t.Fatal(err)
			}
		}
	}
	wantOccurrences := traceReadHistoricalBugOccurrences(t, policy.OccurrencesAuthority)
	if !reflect.DeepEqual(extractedOccurrences, wantOccurrences) {
		t.Fatalf("historical bug occurrence ledger drift: extracted=%d ledger=%d", len(extractedOccurrences), len(wantOccurrences))
	}
	occurrenceJSON := traceMarshalJSONLines(t, extractedOccurrences)
	if len(extractedOccurrences) != policy.Baseline.OccurrenceCount ||
		traceStringsDigest(occurrenceJSON) != policy.Baseline.OccurrencesSHA256 {
		t.Fatalf("historical bug occurrence baseline drift: occurrences=%d digest=%s, want occurrences=%d digest=%s",
			len(extractedOccurrences), traceStringsDigest(occurrenceJSON),
			policy.Baseline.OccurrenceCount, policy.Baseline.OccurrencesSHA256)
	}

	occurrencesByID := make(map[string][]traceHistoricalBugOccurrence)
	for _, occurrence := range extractedOccurrences {
		occurrencesByID[occurrence.BugID] = append(occurrencesByID[occurrence.BugID], occurrence)
	}
	rowCapabilities := traceHistoricalBugRowCapabilities(rows)
	rowLessonRefs := traceHistoricalBugRowLessonRefs(rows)
	reviews := traceReadHistoricalBugIDReviews(t, policy.NarrativeReviewsAuthority)
	reviewsByID := make(map[string]traceHistoricalBugIDReview, len(reviews))
	previousReviewID := ""
	for _, review := range reviews {
		if review.SchemaVersion != 1 || review.BugID <= previousReviewID ||
			len(review.CapabilityIDs) == 0 || !sort.StringsAreSorted(review.CapabilityIDs) ||
			strings.TrimSpace(review.ReviewNote) == "" {
			t.Fatalf("invalid or unordered narrative bug review: %#v", review)
		}
		previousReviewID = review.BugID
		occurrences, exists := occurrencesByID[review.BugID]
		if !exists {
			t.Fatalf("narrative bug review %q has no occurrence", review.BugID)
		}
		for _, occurrence := range occurrences {
			if occurrence.CoverageKind == "row_covered" {
				t.Fatalf("row-covered bug %q must derive from rich rows, not narrative review", review.BugID)
			}
		}
		for index, capabilityID := range review.CapabilityIDs {
			if index > 0 && capabilityID == review.CapabilityIDs[index-1] {
				t.Fatalf("narrative bug review %q repeats capability %q", review.BugID, capabilityID)
			}
			if _, exists := accepted[capabilityID]; !exists {
				t.Fatalf("narrative bug review %q maps non-accepted capability %q", review.BugID, capabilityID)
			}
		}
		if _, duplicate := reviewsByID[review.BugID]; duplicate {
			t.Fatalf("duplicate narrative bug review %q", review.BugID)
		}
		reviewsByID[review.BugID] = review
	}
	for bugID, occurrences := range occurrencesByID {
		rowCovered := false
		for _, occurrence := range occurrences {
			rowCovered = rowCovered || occurrence.CoverageKind == "row_covered"
		}
		_, reviewed := reviewsByID[bugID]
		if rowCovered == reviewed {
			if rowCovered {
				t.Fatalf("row-covered bug %q unexpectedly has narrative review", bugID)
			}
			t.Fatalf("narrative-only bug %q lacks exact capability review", bugID)
		}
	}
	bindingsByID := make(map[string]traceHistoricalBugIDReviewBinding)
	for _, binding := range traceReadHistoricalBugIDReviewBindings(t, policy.NarrativeReviewBindingsAuthority) {
		bindingsByID[binding.BugID] = binding
	}
	generatedIDs := traceBuildHistoricalBugIDs(occurrencesByID, rowCapabilities, rowLessonRefs, reviewsByID, bindingsByID)
	if os.Getenv("ORQUESTA_TRACEABILITY_EMIT_HISTORICAL_BUG_IDS") == "1" {
		encoder := json.NewEncoder(os.Stdout)
		for _, entry := range generatedIDs {
			if err := encoder.Encode(entry); err != nil {
				t.Fatal(err)
			}
		}
	}
	ids := traceReadHistoricalBugIDs(t, policy.IDsAuthority)
	if !reflect.DeepEqual(generatedIDs, ids) {
		t.Fatalf("historical bug ID ledger drift: generated=%d ledger=%d", len(generatedIDs), len(ids))
	}
	previousID := ""
	coverageCounts := map[string]int{}
	for _, entry := range ids {
		if entry.SchemaVersion != 1 || entry.BugID <= previousID || strings.TrimSpace(entry.ReviewNote) == "" ||
			entry.Disposition != "historical_lesson_pending" ||
			len(entry.CapabilityIDs) == 0 || !sort.StringsAreSorted(entry.CapabilityIDs) {
			t.Fatalf("invalid or unordered historical bug ID: %#v", entry)
		}
		traceValidateHistoricalBugRebuildEvidence(t, entry)
		previousID = entry.BugID
		for index, capabilityID := range entry.CapabilityIDs {
			if index > 0 && capabilityID == entry.CapabilityIDs[index-1] {
				t.Fatalf("historical bug %q repeats capability %q", entry.BugID, capabilityID)
			}
			if _, exists := accepted[capabilityID]; !exists {
				t.Fatalf("historical bug %q maps non-accepted capability %q", entry.BugID, capabilityID)
			}
		}
		occurrences, exists := occurrencesByID[entry.BugID]
		if !exists {
			t.Fatalf("historical bug ID %q has no exact occurrence", entry.BugID)
		}
		refs := make([]string, 0, len(occurrences))
		rowCovered := false
		for _, occurrence := range occurrences {
			refs = append(refs, occurrence.OccurrenceRef)
			rowCovered = rowCovered || occurrence.CoverageKind == "row_covered"
		}
		wantCoverage := "narrative_only"
		if rowCovered {
			wantCoverage = "row_covered"
		}
		if entry.CoverageKind != wantCoverage || entry.OccurrenceCount != len(occurrences) ||
			entry.OccurrencesSHA256 != traceStringsDigest(refs) {
			t.Fatalf("historical bug %q occurrence summary drift: %#v", entry.BugID, entry)
		}
		if entry.CoverageKind == "row_covered" {
			if !reflect.DeepEqual(entry.CapabilityIDs, rowCapabilities[entry.BugID]) {
				t.Fatalf("row-covered bug %q capabilities=%v, want exact rich-row union=%v",
					entry.BugID, entry.CapabilityIDs, rowCapabilities[entry.BugID])
			}
			if !traceSortedContains(rowLessonRefs[entry.BugID], entry.LessonTestRef) {
				t.Fatalf("row-covered bug %q lesson %q does not come from its rich rows %v",
					entry.BugID, entry.LessonTestRef, rowLessonRefs[entry.BugID])
			}
		} else {
			wantLessonRef := "planned:historical-bug-lessons/" + traceSHA256Hex(entry.BugID)[:20]
			if entry.LessonTestRef != wantLessonRef {
				t.Fatalf("narrative-only bug %q lesson=%q, want %q", entry.BugID, entry.LessonTestRef, wantLessonRef)
			}
		}
		wantLessonState := "candidate_not_accredited"
		if strings.HasPrefix(entry.LessonTestRef, "planned:") {
			wantLessonState = "pending_invariant_test"
		}
		if entry.LessonState != wantLessonState {
			t.Fatalf("historical bug %q lesson state=%q, want %q", entry.BugID, entry.LessonState, wantLessonState)
		}
		coverageCounts[entry.CoverageKind]++
		delete(occurrencesByID, entry.BugID)
	}
	if len(occurrencesByID) != 0 {
		missing := make([]string, 0, len(occurrencesByID))
		for bugID := range occurrencesByID {
			missing = append(missing, bugID)
		}
		sort.Strings(missing)
		t.Fatalf("historical bug IDs without disposition=%d: %v", len(missing), missing)
	}
	idJSON := traceMarshalJSONLines(t, ids)
	if len(ids) != policy.Baseline.IDCount ||
		coverageCounts["row_covered"] != policy.Baseline.RowCoveredIDCount ||
		coverageCounts["narrative_only"] != policy.Baseline.NarrativeOnlyIDCount ||
		traceStringsDigest(idJSON) != policy.Baseline.IDsSHA256 {
		t.Fatalf("historical bug ID baseline drift: ids=%d row_covered=%d narrative_only=%d digest=%s",
			len(ids), coverageCounts["row_covered"], coverageCounts["narrative_only"], traceStringsDigest(idJSON))
	}
	t.Logf("historical_bugs: sources=%d rich_rows=%d occurrences=%d ids=%d row_covered=%d narrative_only=%d closure_inferred=0",
		len(bugSources), len(rows), len(extractedOccurrences), len(ids),
		coverageCounts["row_covered"], coverageCounts["narrative_only"])
}

func TestTraceabilityRebuildHistoricalBugEmitModesDoNotBypassValidation(t *testing.T) {
	const sourcePath = "traceability_historical_bug_ids_test.go"
	const emitEnvironmentPrefix = "ORQUESTA_TRACEABILITY_EMIT_HISTORICAL_BUG_"
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	files := token.NewFileSet()
	syntax, err := parser.ParseFile(files, sourcePath, content, parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	parsed := files.File(syntax.Pos())
	checked := 0
	ast.Inspect(syntax, func(node ast.Node) bool {
		statement, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		start := parsed.Offset(statement.Cond.Pos())
		end := parsed.Offset(statement.Cond.End())
		condition := string(content[start:end])
		if !strings.Contains(condition, emitEnvironmentPrefix) {
			return true
		}
		checked++
		ast.Inspect(statement.Body, func(bodyNode ast.Node) bool {
			if _, bypass := bodyNode.(*ast.ReturnStmt); bypass {
				t.Errorf("emit-only condition %q contains a return that bypasses canonical ledger validation", condition)
			}
			return true
		})
		return true
	})
	if checked != 2 {
		t.Fatalf("emit-only condition count=%d, want 2", checked)
	}
}

func TestTraceabilityRebuildHistoricalBugCapabilityAuthorityResolves(t *testing.T) {
	var policy traceHistoricalBugExtractionPolicy
	traceDecodeStrict(t, "product/traceability/historical_bug_extraction_policy.json", &policy)
	path, fragment, ok := strings.Cut(policy.CapabilityAuthority, "#")
	if !ok || path != "product/roadmap.json" || fragment != "capability_entries" {
		t.Fatalf("historical bug capability authority does not resolve to roadmap capabilities: %q", policy.CapabilityAuthority)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(content, &document); err != nil {
		t.Fatal(err)
	}
	var capabilities []json.RawMessage
	if raw, exists := document[fragment]; !exists {
		t.Fatalf("authority fragment %q is absent from %s", fragment, path)
	} else if err := json.Unmarshal(raw, &capabilities); err != nil || len(capabilities) == 0 {
		t.Fatalf("authority fragment %q is not a non-empty capability array: count=%d err=%v", fragment, len(capabilities), err)
	}
}

func TestTraceabilityRebuildHistoricalBugSourceCoverage(t *testing.T) {
	var policy traceHistoricalBugExtractionPolicy
	traceDecodeStrict(t, "product/traceability/historical_bug_extraction_policy.json", &policy)

	bugRoleSources := make(map[string]traceSourceDisposition)
	for _, source := range traceReadDispositionJSONL(t, policy.SourcesAuthority) {
		if traceSourceHasRole(source, "bug") {
			bugRoleSources[source.SourceRef] = source
		}
	}
	literalSources := make(map[string]struct{})
	for _, root := range []string{"docs", "skills"} {
		err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			slashPath := filepath.ToSlash(path)
			if info.IsDir() {
				if slashPath == "docs/reconstruccion" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".md" {
				return nil
			}
			inScope := strings.HasPrefix(slashPath, "docs/") ||
				(strings.HasPrefix(slashPath, "skills/") && filepath.Base(path) == "SKILL.md" && strings.Count(slashPath, "/") == 2)
			if !inScope {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(content), "BUG-ORQ") {
				literalSources[slashPath] = struct{}{}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	legacySources := traceLoadLegacySourceSnapshot(t, ".")
	for path, content := range legacySources.Contents {
		if strings.HasPrefix(path, "modulos/orquesta-") &&
			strings.Contains(path, "/docs/") &&
			strings.Contains(string(content), "BUG-ORQ") {
			literalSources[path] = struct{}{}
		}
	}

	occurrenceSources := make(map[string]struct{})
	for _, occurrence := range traceReadHistoricalBugOccurrences(t, policy.OccurrencesAuthority) {
		occurrenceSources[occurrence.SourceRef] = struct{}{}
	}
	missingRole := make([]string, 0)
	missingOccurrence := make([]string, 0)
	for sourceRef := range literalSources {
		if _, exists := bugRoleSources[sourceRef]; !exists {
			missingRole = append(missingRole, sourceRef)
		}
		if _, exists := occurrenceSources[sourceRef]; !exists {
			missingOccurrence = append(missingOccurrence, sourceRef)
		}
	}
	sort.Strings(missingRole)
	sort.Strings(missingOccurrence)
	if len(literalSources) != policy.Baseline.SourcesWithExplicitID || len(missingRole) != 0 || len(missingOccurrence) != 0 {
		t.Fatalf("historical BUG-ORQ filesystem coverage: literal_sources=%d want=%d missing_role=%v missing_occurrence=%v",
			len(literalSources), policy.Baseline.SourcesWithExplicitID, missingRole, missingOccurrence)
	}
	t.Logf("historical BUG-ORQ filesystem coverage: literal_sources=%d missing_role=0 missing_occurrence=0", len(literalSources))
}
