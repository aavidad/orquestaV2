package orquesta_test

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

type traceHistoricalBugCapabilityReason struct {
	CapabilityID string `json:"capability_id"`
	Reason       string `json:"reason"`
}

type traceHistoricalBugIDReviewBinding struct {
	SchemaVersion     int                                  `json:"schema_version"`
	BugID             string                               `json:"bug_id"`
	OccurrenceRefs    []string                             `json:"occurrence_refs"`
	OccurrencesSHA256 string                               `json:"occurrences_sha256"`
	CapabilityReasons []traceHistoricalBugCapabilityReason `json:"capability_reasons"`
}

type traceHistoricalBugRowEnrichment struct {
	SchemaVersion        int    `json:"schema_version"`
	EntryRef             string `json:"entry_ref"`
	SourceRef            string `json:"source_ref"`
	SourceLine           int    `json:"source_line"`
	SourceSHA256         string `json:"source_sha256"`
	SemanticFieldsSHA256 string `json:"semantic_fields_sha256"`
	Basis                string `json:"basis"`
}

func TestTraceabilityRebuildHistoricalBugReviewBindings(t *testing.T) {
	var policy traceHistoricalBugExtractionPolicy
	traceDecodeStrict(t, "product/traceability/historical_bug_extraction_policy.json", &policy)
	reviews := traceReadHistoricalBugIDReviews(t, "product/traceability/historical_bug_id_reviews.jsonl")
	bindings := traceReadHistoricalBugIDReviewBindings(t, policy.NarrativeReviewBindingsAuthority)
	occurrences := traceReadHistoricalBugOccurrences(t, "product/traceability/historical_bug_occurrences.jsonl")
	ids := traceReadHistoricalBugIDs(t, "product/traceability/historical_bug_ids.jsonl")

	occurrencesByID := make(map[string][]traceHistoricalBugOccurrence)
	for _, occurrence := range occurrences {
		occurrencesByID[occurrence.BugID] = append(occurrencesByID[occurrence.BugID], occurrence)
	}
	idsByID := make(map[string]traceHistoricalBugID, len(ids))
	for _, entry := range ids {
		idsByID[entry.BugID] = entry
	}

	if len(bindings) != len(reviews) {
		t.Fatalf("historical bug review bindings=%d, want one for each of %d reviews", len(bindings), len(reviews))
	}
	if len(bindings) != policy.Baseline.NarrativeReviewBindingCount ||
		traceFileSHA256(t, policy.NarrativeReviewBindingsAuthority) != policy.Baseline.NarrativeReviewBindingsSHA256 {
		t.Fatalf("historical bug review binding baseline drift: count=%d digest=%s", len(bindings), traceFileSHA256(t, policy.NarrativeReviewBindingsAuthority))
	}
	for index, review := range reviews {
		binding := bindings[index]
		if binding.SchemaVersion != 1 || binding.BugID != review.BugID {
			t.Fatalf("historical bug review binding order/identity drift at %d: %#v, want %q", index, binding, review.BugID)
		}
		bugOccurrences := occurrencesByID[review.BugID]
		wantRefs := make([]string, 0, len(bugOccurrences))
		seenRefs := make(map[string]struct{}, len(bugOccurrences))
		for _, occurrence := range bugOccurrences {
			if occurrence.CoverageKind != "narrative_only" {
				t.Fatalf("narrative review %q binds row-covered occurrence %q", review.BugID, occurrence.OccurrenceRef)
			}
			if _, duplicate := seenRefs[occurrence.OccurrenceRef]; duplicate {
				t.Fatalf("narrative review %q repeats occurrence %q", review.BugID, occurrence.OccurrenceRef)
			}
			seenRefs[occurrence.OccurrenceRef] = struct{}{}
			wantRefs = append(wantRefs, occurrence.OccurrenceRef)
		}
		wantDigest := traceStringsDigest(wantRefs)
		if len(wantRefs) == 0 || !reflect.DeepEqual(binding.OccurrenceRefs, wantRefs) ||
			binding.OccurrencesSHA256 != wantDigest {
			t.Fatalf("historical bug review %q occurrence binding drift: refs=%v digest=%q, want refs=%v digest=%q",
				review.BugID, binding.OccurrenceRefs, binding.OccurrencesSHA256, wantRefs, wantDigest)
		}
		idEntry, exists := idsByID[review.BugID]
		if !exists || idEntry.CoverageKind != "narrative_only" ||
			idEntry.OccurrenceCount != len(wantRefs) || idEntry.OccurrencesSHA256 != wantDigest {
			t.Fatalf("historical bug review %q does not bind the generated ID occurrence summary", review.BugID)
		}
		if len(binding.CapabilityReasons) != len(review.CapabilityIDs) {
			t.Fatalf("historical bug review %q reasons=%d, want %d", review.BugID, len(binding.CapabilityReasons), len(review.CapabilityIDs))
		}
		seenReasons := make(map[string]struct{}, len(binding.CapabilityReasons))
		for reasonIndex, capabilityReason := range binding.CapabilityReasons {
			if capabilityReason.CapabilityID != review.CapabilityIDs[reasonIndex] {
				t.Fatalf("historical bug review %q capability reason order/union drift at %d: %q, want %q",
					review.BugID, reasonIndex, capabilityReason.CapabilityID, review.CapabilityIDs[reasonIndex])
			}
			reason := strings.TrimSpace(capabilityReason.Reason)
			if len(reason) < 24 {
				t.Fatalf("historical bug review %q capability %q lacks an explicit reason", review.BugID, capabilityReason.CapabilityID)
			}
			if _, duplicate := seenReasons[reason]; duplicate {
				t.Fatalf("historical bug review %q repeats one reason for different capabilities", review.BugID)
			}
			seenReasons[reason] = struct{}{}
		}
	}

	traceRequireHistoricalBugCapabilities(t, reviews, "BUG-ORQ-20260710-208O", []string{"AGT-03", "GOV-21", "TLS-01"})
	traceRequireHistoricalBugCapabilities(t, reviews, "BUG-ORQ-20260711-208AF", []string{"EVD-01", "GOV-06", "ORC-13"})
	traceRequireHistoricalBugCapabilities(t, reviews, "BUG-ORQ-20260711-269", []string{"GOV-06", "ORC-17", "ORC-23"})
}

func TestTraceabilityRebuildHistoricalBugRowProvenance(t *testing.T) {
	var policy traceHistoricalBugExtractionPolicy
	traceDecodeStrict(t, "product/traceability/historical_bug_extraction_policy.json", &policy)
	rows := traceReadHistoricalBugRows(t, "product/traceability/historical_bug_rows.jsonl")
	enrichments := traceReadHistoricalBugRowEnrichments(t, policy.RichRowEnrichmentsAuthority)
	sourceRows := traceReadDispositionJSONL(t, "product/traceability/source_dispositions.jsonl")
	sourceSHAByRef := make(map[string]string)
	for _, source := range sourceRows {
		if traceSourceHasRole(source, "bug") {
			sourceSHAByRef[source.SourceRef] = source.SourceSHA256
		}
	}

	enrichmentsByKey := make(map[string]traceHistoricalBugRowEnrichment, len(enrichments))
	previousKey := ""
	for _, enrichment := range enrichments {
		key := traceHistoricalBugRowKey(enrichment.SourceRef, enrichment.SourceLine)
		if enrichment.SchemaVersion != 1 || key <= previousKey || enrichment.SourceLine < 1 ||
			strings.TrimSpace(enrichment.Basis) == "" {
			t.Fatalf("invalid or unordered historical bug row enrichment: %#v", enrichment)
		}
		previousKey = key
		if _, duplicate := enrichmentsByKey[key]; duplicate {
			t.Fatalf("duplicate historical bug row enrichment %s:%d", enrichment.SourceRef, enrichment.SourceLine)
		}
		enrichmentsByKey[key] = enrichment
	}

	for _, row := range rows {
		lines := traceReadSourceLines(t, row.SourceRef)
		line := lines[row.SourceLine-1]
		key := traceHistoricalBugRowKey(row.SourceRef, row.SourceLine)
		if row.SourceRef == "docs/inventario_bugs_orquesta_2026-06-30.md" {
			traceRequireHistoricalBugExactInventoryCells(t, row, traceMarkdownCellsOutsideCode(line))
			if _, exists := enrichmentsByKey[key]; exists {
				t.Fatalf("exact rich historical bug row %s:%d must not use manual enrichment", row.SourceRef, row.SourceLine)
			}
			continue
		}

		enrichment, exists := enrichmentsByKey[key]
		if !exists {
			t.Fatalf("manual rich historical bug row %s:%d lacks provenance", row.SourceRef, row.SourceLine)
		}
		wantSemanticSHA := traceHistoricalBugSemanticFieldsDigest(row)
		if enrichment.EntryRef != row.EntryRef || enrichment.SourceSHA256 != sourceSHAByRef[row.SourceRef] ||
			enrichment.SemanticFieldsSHA256 != wantSemanticSHA {
			t.Fatalf("manual rich historical bug row %s:%d provenance drift: %#v, semantic=%q",
				row.SourceRef, row.SourceLine, enrichment, wantSemanticSHA)
		}
		delete(enrichmentsByKey, key)
	}
	if len(enrichmentsByKey) != 0 {
		t.Fatalf("historical bug row enrichments without a manual row: %v", enrichmentsByKey)
	}
	if len(enrichments) != policy.Baseline.RichRowEnrichmentCount ||
		traceFileSHA256(t, policy.RichRowEnrichmentsAuthority) != policy.Baseline.RichRowEnrichmentsSHA256 {
		t.Fatalf("historical bug row enrichment baseline drift: count=%d digest=%s", len(enrichments), traceFileSHA256(t, policy.RichRowEnrichmentsAuthority))
	}
}

func TestTraceabilityRebuildHistoricalBugLetterSuffixExpansion(t *testing.T) {
	got := traceExpandHistoricalBugRef("BUG-ORQ-20260710-208/A/AF")
	want := []string{
		"BUG-ORQ-20260710-208",
		"BUG-ORQ-20260710-208A",
		"BUG-ORQ-20260710-208AF",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("historical bug letter suffix expansion=%v, want %v", got, want)
	}
}

func TestTraceabilityRebuildHistoricalBugLetterRangeExpansion(t *testing.T) {
	got := traceExpandHistoricalBugRef("BUG-ORQ-20260710-208A-D")
	want := []string{
		"BUG-ORQ-20260710-208A",
		"BUG-ORQ-20260710-208B",
		"BUG-ORQ-20260710-208C",
		"BUG-ORQ-20260710-208D",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("historical bug letter range expansion=%v, want %v", got, want)
	}
}

func traceReadHistoricalBugIDReviewBindings(t *testing.T, path string) []traceHistoricalBugIDReviewBinding {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []traceHistoricalBugIDReviewBinding
	traceScanStrictJSONL(t, file, path, func(decoder *json.Decoder, _ int) {
		var entry traceHistoricalBugIDReviewBinding
		if err := decoder.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	})
	return entries
}

func traceReadHistoricalBugRowEnrichments(t *testing.T, path string) []traceHistoricalBugRowEnrichment {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var entries []traceHistoricalBugRowEnrichment
	traceScanStrictJSONL(t, file, path, func(decoder *json.Decoder, _ int) {
		var entry traceHistoricalBugRowEnrichment
		if err := decoder.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	})
	return entries
}

func traceRequireHistoricalBugCapabilities(t *testing.T, reviews []traceHistoricalBugIDReview, bugID string, want []string) {
	t.Helper()
	index := sort.Search(len(reviews), func(index int) bool { return reviews[index].BugID >= bugID })
	if index == len(reviews) || reviews[index].BugID != bugID {
		t.Fatalf("historical bug review %q not found", bugID)
	}
	if !reflect.DeepEqual(reviews[index].CapabilityIDs, want) {
		t.Fatalf("historical bug %q capabilities=%v, want %v", bugID, reviews[index].CapabilityIDs, want)
	}
}

func traceRequireHistoricalBugExactInventoryCells(t *testing.T, row traceHistoricalBugRow, cells []string) {
	t.Helper()
	var want []string
	switch len(cells) {
	case 7:
		want = []string{cells[1], cells[2], cells[3], cells[4], cells[6]}
	case 6:
		want = []string{cells[2], cells[1], cells[3], "not_declared_in_row", cells[5]}
	default:
		t.Fatalf("rich historical bug row %s:%d has unsupported Markdown cells=%d", row.SourceRef, row.SourceLine, len(cells))
	}
	got := []string{row.OriginalState, row.Area, row.Symptom, row.ArchitecturalHypothesis, row.Action}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rich historical bug row %s:%d semantic fields are not exact source cells: got=%v want=%v",
			row.SourceRef, row.SourceLine, got, want)
	}
}

func traceMarkdownCellsOutsideCode(line string) []string {
	parts := make([]string, 0, 8)
	var current strings.Builder
	inCode := false
	for _, char := range line {
		switch char {
		case '`':
			inCode = !inCode
			current.WriteRune(char)
		case '|':
			if inCode {
				current.WriteRune(char)
			} else {
				parts = append(parts, strings.TrimSpace(current.String()))
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}
	parts = append(parts, strings.TrimSpace(current.String()))
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func traceHistoricalBugSemanticFieldsDigest(row traceHistoricalBugRow) string {
	return traceStringsDigest([]string{
		row.EntryRef,
		row.SourceRef,
		strconv.Itoa(row.SourceLine),
		row.TextSHA256,
		row.OriginalState,
		row.Area,
		row.Symptom,
		row.ArchitecturalHypothesis,
		row.Action,
	})
}
