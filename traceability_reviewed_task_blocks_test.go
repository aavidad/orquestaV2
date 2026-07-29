package orquesta_test

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	reviewedTaskBlocksPath                        = "product/traceability/reviewed_task_blocks.jsonl"
	reviewedTaskBlockSourceCounterreviewPath      = "product/traceability/fixtures/markdown_source_counterreview.jsonl"
	reviewedTaskBlockBroadSourceReviewS0Path      = "product/traceability/fixtures/markdown_broad_source_review_s0.jsonl"
	reviewedTaskBlockBroadSourceReviewS1Path      = "product/traceability/fixtures/markdown_broad_source_review_s1.jsonl"
	reviewedTaskBlockSupplementReviewPath         = "product/traceability/fixtures/markdown_task_block_supplement_review.jsonl"
	reviewedTaskBlockSourceCounterreviewRefPrefix = "v03_source_counterreview"
	reviewedTaskBlockBroadSourceReviewS0RefPrefix = "v03_broad_source_review_s0"
	reviewedTaskBlockBroadSourceReviewS1RefPrefix = "v03_broad_source_review_s1"
	reviewedTaskBlockSupplementReviewRefPrefix    = "v03_task_block_supplement_review"
)

type reviewedTaskBlockSource struct {
	sha256 string
	lines  []string
}

type reviewedTaskBlockReviewRange struct {
	FirstLine int    `json:"first_line"`
	LastLine  int    `json:"last_line"`
	Summary   string `json:"summary"`
}

type reviewedTaskBlockPrimaryReview struct {
	SourceRef        string                         `json:"source_ref"`
	Classification   string                         `json:"classification"`
	Reason           string                         `json:"reason"`
	CapabilityIDs    []string                       `json:"capability_ids"`
	ActionableBlocks []reviewedTaskBlockReviewRange `json:"actionable_blocks"`
	DelegatedRefs    []string                       `json:"delegated_refs"`
}

type reviewedTaskBlockCounterreview struct {
	ReviewKind                string                          `json:"review_kind"`
	SourceRef                 string                          `json:"source_ref"`
	Verdict                   string                          `json:"verdict"`
	CorrectedClassification   *string                         `json:"corrected_classification,omitempty"`
	CorrectedCapabilityIDs    *[]string                       `json:"corrected_capability_ids,omitempty"`
	CorrectedActionableBlocks *[]reviewedTaskBlockReviewRange `json:"corrected_actionable_blocks,omitempty"`
	Reason                    string                          `json:"reason"`
}

type reviewedTaskBlockReviewKey struct {
	SourceRef string
	FirstLine int
	LastLine  int
}

type reviewedTaskBlockReviewExpectation struct {
	Summary         string
	CapabilityHints []string
	ReviewRefs      []string
}

func TestTraceabilityRebuildReviewedTaskBlocks(t *testing.T) {
	blocks := v2ReadJSONL[v2ReviewedTaskBlock](t, reviewedTaskBlocksPath)
	legacySources := traceLoadLegacySourceSnapshot(t, ".")
	if len(blocks) != 170 {
		t.Fatalf("reviewed task blocks=%d, want 170", len(blocks))
	}

	entries := v2ReadJSONL[v2TaskEntry](t, v2TaskEntriesPath)
	exclusions := v2ReadJSONL[v2TaskExclusion](t, v2TaskExclusionsPath)
	entriesByCandidate := make(map[string][]v2TaskEntry, len(entries))
	exclusionsByCandidate := make(map[string]int, len(exclusions))
	for _, entry := range entries {
		entriesByCandidate[entry.CandidateRef] = append(entriesByCandidate[entry.CandidateRef], entry)
	}
	for _, exclusion := range exclusions {
		exclusionsByCandidate[exclusion.CandidateRef]++
	}

	sources := make(map[string]reviewedTaskBlockSource)
	blockRefs := make(map[string]struct{}, len(blocks))
	reviewedBlockMembership := make(map[string]int)
	for index, block := range blocks {
		if block.SchemaVersion != 1 {
			t.Fatalf("reviewed task block %s schema_version=%d, want 1", block.BlockRef, block.SchemaVersion)
		}
		if index > 0 && !reviewedTaskBlockStrictlyBefore(blocks[index-1], block) {
			t.Fatalf(
				"reviewed task blocks not strictly ordered at %s:%d-%d after %s:%d-%d",
				block.SourceRef,
				block.FirstLine,
				block.LastLine,
				blocks[index-1].SourceRef,
				blocks[index-1].FirstLine,
				blocks[index-1].LastLine,
			)
		}
		if _, duplicate := blockRefs[block.BlockRef]; duplicate {
			t.Fatalf("duplicate reviewed task block ref %s", block.BlockRef)
		}
		blockRefs[block.BlockRef] = struct{}{}
		if strings.TrimSpace(block.Summary) == "" {
			t.Fatalf("reviewed task block %s has empty summary", block.BlockRef)
		}

		source := reviewedTaskBlockLoadSource(t, sources, block.SourceRef, legacySources)
		if block.SourceSHA256 != source.sha256 {
			t.Fatalf(
				"reviewed task block %s source hash=%s, want %s",
				block.BlockRef,
				block.SourceSHA256,
				source.sha256,
			)
		}
		if block.FirstLine < 1 || block.LastLine < block.FirstLine || block.LastLine > len(source.lines) {
			t.Fatalf(
				"reviewed task block %s invalid range %d-%d for %s with %d lines",
				block.BlockRef,
				block.FirstLine,
				block.LastLine,
				block.SourceRef,
				len(source.lines),
			)
		}
		blockText := strings.Join(source.lines[block.FirstLine-1:block.LastLine], "\n")
		wantBlockSHA := v2SHA([]byte(blockText))
		if block.BlockSHA256 != wantBlockSHA {
			t.Fatalf(
				"reviewed task block %s block hash=%s, want %s",
				block.BlockRef,
				block.BlockSHA256,
				wantBlockSHA,
			)
		}
		wantBlockRef := reviewedTaskBlockRef(block.SourceRef, block.FirstLine, block.LastLine, wantBlockSHA)
		if block.BlockRef != wantBlockRef {
			t.Fatalf("reviewed task block ref=%s, want %s", block.BlockRef, wantBlockRef)
		}

		reviewedTaskBlockRequireSortedValues(t, block.BlockRef, "capability_hints", block.CapabilityHints)
		reviewedTaskBlockRequireSortedValues(t, block.BlockRef, "review_refs", block.ReviewRefs)
		reviewedTaskBlockRequireSortedValues(t, block.BlockRef, "candidate_refs", block.CandidateRefs)

		resolved := make([]v2TaskEntry, 0, len(block.CandidateRefs))
		for _, candidateRef := range block.CandidateRefs {
			if count := exclusionsByCandidate[candidateRef]; count != 0 {
				t.Fatalf(
					"reviewed task block %s candidate %s has %d TaskExclusion records",
					block.BlockRef,
					candidateRef,
					count,
				)
			}
			candidateEntries := entriesByCandidate[candidateRef]
			if len(candidateEntries) != 1 {
				t.Fatalf(
					"reviewed task block %s candidate %s resolves to %d TaskEntry records, want 1",
					block.BlockRef,
					candidateRef,
					len(candidateEntries),
				)
			}
			resolved = append(resolved, candidateEntries[0])
		}

		switch block.CoverageKind {
		case "scanner_entries":
			reviewedTaskBlockValidateScannerEntries(t, block, resolved)
		case "reviewed_block_entry":
			reviewedTaskBlockValidateSyntheticEntry(t, block, resolved)
			reviewedBlockMembership[resolved[0].CandidateRef]++
		default:
			t.Fatalf("reviewed task block %s coverage_kind=%q", block.BlockRef, block.CoverageKind)
		}
	}

	for _, entry := range entries {
		if entry.DetectionKind != "reviewed_block" {
			continue
		}
		if count := reviewedBlockMembership[entry.CandidateRef]; count != 1 {
			t.Fatalf(
				"reviewed_block TaskEntry %s belongs to %d reviewed task blocks, want 1",
				entry.EntryRef,
				count,
			)
		}
	}
}

func TestTraceabilityRebuildReviewedTaskBlocksApplySourceCounterreview(t *testing.T) {
	counterreviews := v2ReadJSONL[reviewedTaskBlockCounterreview](t, reviewedTaskBlockSourceCounterreviewPath)
	if len(counterreviews) != 28 {
		t.Fatalf("source counterreview rows=%d, want 28", len(counterreviews))
	}

	sourceReviews := reviewedTaskBlockPrimaryReviewsBySource(t, reviewedTaskBlockBroadSourceReviewS1Path, 20)
	sdkReviews := reviewedTaskBlockPrimaryReviewsBySource(t, reviewedTaskBlockBroadSourceReviewS0Path, 21)
	supplementReviews := reviewedTaskBlockPrimaryReviewsBySource(t, reviewedTaskBlockSupplementReviewPath, 7)
	counterreviewRef := reviewedTaskBlockFixtureRef(
		t,
		reviewedTaskBlockSourceCounterreviewRefPrefix,
		reviewedTaskBlockSourceCounterreviewPath,
	)
	sourceReviewRef := reviewedTaskBlockFixtureRef(
		t,
		reviewedTaskBlockBroadSourceReviewS1RefPrefix,
		reviewedTaskBlockBroadSourceReviewS1Path,
	)
	sdkReviewRef := reviewedTaskBlockFixtureRef(
		t,
		reviewedTaskBlockBroadSourceReviewS0RefPrefix,
		reviewedTaskBlockBroadSourceReviewS0Path,
	)
	supplementReviewRef := reviewedTaskBlockFixtureRef(
		t,
		reviewedTaskBlockSupplementReviewRefPrefix,
		reviewedTaskBlockSupplementReviewPath,
	)

	expected := make(map[reviewedTaskBlockReviewKey]reviewedTaskBlockReviewExpectation)
	primaryRefBySource := make(map[string]string, len(counterreviews))
	seenCounterreviewSources := make(map[string]struct{}, len(counterreviews))
	usedSourceReviews := make(map[string]struct{}, len(sourceReviews))
	usedSupplementReviews := make(map[string]struct{}, len(supplementReviews))
	kindCounts := make(map[string]int)

	for rowIndex, counterreview := range counterreviews {
		if counterreview.SourceRef == "" || strings.TrimSpace(counterreview.SourceRef) != counterreview.SourceRef {
			t.Fatalf("source counterreview row %d has invalid source_ref %q", rowIndex+1, counterreview.SourceRef)
		}
		if strings.TrimSpace(counterreview.Reason) == "" {
			t.Fatalf("source counterreview %s has empty adjudication reason", counterreview.SourceRef)
		}
		if _, duplicate := seenCounterreviewSources[counterreview.SourceRef]; duplicate {
			t.Fatalf("duplicate source counterreview for %s", counterreview.SourceRef)
		}
		seenCounterreviewSources[counterreview.SourceRef] = struct{}{}
		kindCounts[counterreview.ReviewKind]++

		var primary reviewedTaskBlockPrimaryReview
		var primaryRef string
		var found bool
		switch counterreview.ReviewKind {
		case "source_review":
			primary, found = sourceReviews[counterreview.SourceRef]
			primaryRef = sourceReviewRef
			usedSourceReviews[counterreview.SourceRef] = struct{}{}
		case "sdk_conflict":
			primary, found = sdkReviews[counterreview.SourceRef]
			primaryRef = sdkReviewRef
		case "block_supplement":
			primary, found = supplementReviews[counterreview.SourceRef]
			primaryRef = supplementReviewRef
			usedSupplementReviews[counterreview.SourceRef] = struct{}{}
		default:
			t.Fatalf("source counterreview %s review_kind=%q", counterreview.SourceRef, counterreview.ReviewKind)
		}
		if !found {
			t.Fatalf("source counterreview %s has no primary %s row", counterreview.SourceRef, counterreview.ReviewKind)
		}
		primaryRefBySource[counterreview.SourceRef] = primaryRef

		classification := primary.Classification
		capabilityHints := primary.CapabilityIDs
		actionableBlocks := primary.ActionableBlocks
		switch counterreview.Verdict {
		case "confirm":
			if counterreview.CorrectedClassification != nil ||
				counterreview.CorrectedCapabilityIDs != nil ||
				counterreview.CorrectedActionableBlocks != nil {
				t.Fatalf("confirmed source counterreview %s carries corrected_* fields", counterreview.SourceRef)
			}
		case "correct":
			if counterreview.CorrectedClassification == nil ||
				counterreview.CorrectedCapabilityIDs == nil ||
				counterreview.CorrectedActionableBlocks == nil {
				t.Fatalf("corrected source counterreview %s lacks complete corrected_* adjudication", counterreview.SourceRef)
			}
			classification = *counterreview.CorrectedClassification
			capabilityHints = *counterreview.CorrectedCapabilityIDs
			actionableBlocks = *counterreview.CorrectedActionableBlocks
		default:
			t.Fatalf("source counterreview %s verdict=%q", counterreview.SourceRef, counterreview.Verdict)
		}

		if classification == "" || strings.TrimSpace(classification) != classification {
			t.Fatalf("source counterreview %s resolves to invalid classification %q", counterreview.SourceRef, classification)
		}
		capabilityHints = reviewedTaskBlockSortedUniqueReviewValues(
			t,
			counterreview.SourceRef,
			"capability ids",
			capabilityHints,
		)
		if classification != "task_source" && classification != "manual_mixed" {
			actionableBlocks = nil
		}
		if len(actionableBlocks) != 0 && len(capabilityHints) == 0 {
			t.Fatalf("source counterreview %s resolves actionable blocks without capability ids", counterreview.SourceRef)
		}

		reviewRefs := []string{counterreviewRef, primaryRef}
		slices.Sort(reviewRefs)
		for _, actionableBlock := range actionableBlocks {
			if actionableBlock.FirstLine < 1 || actionableBlock.LastLine < actionableBlock.FirstLine {
				t.Fatalf(
					"source counterreview %s has invalid actionable range %d-%d",
					counterreview.SourceRef,
					actionableBlock.FirstLine,
					actionableBlock.LastLine,
				)
			}
			if strings.TrimSpace(actionableBlock.Summary) == "" {
				t.Fatalf(
					"source counterreview %s range %d-%d has empty summary",
					counterreview.SourceRef,
					actionableBlock.FirstLine,
					actionableBlock.LastLine,
				)
			}
			key := reviewedTaskBlockReviewKey{
				SourceRef: counterreview.SourceRef,
				FirstLine: actionableBlock.FirstLine,
				LastLine:  actionableBlock.LastLine,
			}
			if _, duplicate := expected[key]; duplicate {
				t.Fatalf(
					"duplicate adjudicated actionable block %s:%d-%d",
					key.SourceRef,
					key.FirstLine,
					key.LastLine,
				)
			}
			expected[key] = reviewedTaskBlockReviewExpectation{
				Summary:         actionableBlock.Summary,
				CapabilityHints: capabilityHints,
				ReviewRefs:      reviewRefs,
			}
		}
	}

	if kindCounts["source_review"] != 20 || kindCounts["sdk_conflict"] != 1 || kindCounts["block_supplement"] != 7 {
		t.Fatalf("source counterreview kind counts=%v, want source_review=20 sdk_conflict=1 block_supplement=7", kindCounts)
	}
	if len(usedSourceReviews) != len(sourceReviews) {
		t.Fatalf("source counterreview consumes %d/%d broad source s1 rows", len(usedSourceReviews), len(sourceReviews))
	}
	if len(usedSupplementReviews) != len(supplementReviews) {
		t.Fatalf("source counterreview consumes %d/%d supplement rows", len(usedSupplementReviews), len(supplementReviews))
	}
	if len(expected) != 29 {
		t.Fatalf("source counterreview resolves to %d reviewed task blocks, want 29", len(expected))
	}

	seenExpected := make(map[reviewedTaskBlockReviewKey]int, len(expected))
	counterreviewedBlockCount := 0
	for _, block := range v2ReadJSONL[v2ReviewedTaskBlock](t, reviewedTaskBlocksPath) {
		primaryRef, inCounterreviewScope := primaryRefBySource[block.SourceRef]
		hasCounterreview := slices.Contains(block.ReviewRefs, counterreviewRef)
		hasPrimaryReview := inCounterreviewScope && slices.Contains(block.ReviewRefs, primaryRef)
		if !hasCounterreview {
			if hasPrimaryReview {
				t.Fatalf(
					"stale primary-only reviewed task block survives counterreview: %s:%d-%d",
					block.SourceRef,
					block.FirstLine,
					block.LastLine,
				)
			}
			continue
		}
		counterreviewedBlockCount++
		if !inCounterreviewScope {
			t.Fatalf(
				"reviewed task block outside 28 adjudicated sources cites counterreview: %s:%d-%d",
				block.SourceRef,
				block.FirstLine,
				block.LastLine,
			)
		}
		key := reviewedTaskBlockReviewKey{
			SourceRef: block.SourceRef,
			FirstLine: block.FirstLine,
			LastLine:  block.LastLine,
		}
		want, exists := expected[key]
		if !exists {
			t.Fatalf(
				"counterreview leaves extra reviewed task block %s:%d-%d",
				block.SourceRef,
				block.FirstLine,
				block.LastLine,
			)
		}
		if block.Summary != want.Summary {
			t.Fatalf(
				"counterreviewed block %s:%d-%d summary=%q, want %q",
				block.SourceRef,
				block.FirstLine,
				block.LastLine,
				block.Summary,
				want.Summary,
			)
		}
		if !slices.Equal(block.CapabilityHints, want.CapabilityHints) {
			t.Fatalf(
				"counterreviewed block %s:%d-%d capability_hints=%v, want %v",
				block.SourceRef,
				block.FirstLine,
				block.LastLine,
				block.CapabilityHints,
				want.CapabilityHints,
			)
		}
		if !slices.Equal(block.ReviewRefs, want.ReviewRefs) {
			t.Fatalf(
				"counterreviewed block %s:%d-%d review_refs=%v, want exact lineage %v",
				block.SourceRef,
				block.FirstLine,
				block.LastLine,
				block.ReviewRefs,
				want.ReviewRefs,
			)
		}
		seenExpected[key]++
	}

	if counterreviewedBlockCount != len(expected) {
		t.Fatalf("counterreview-tagged reviewed task blocks=%d, want %d", counterreviewedBlockCount, len(expected))
	}
	for key := range expected {
		if seenExpected[key] != 1 {
			t.Fatalf(
				"adjudicated reviewed task block %s:%d-%d materialized %d times, want 1",
				key.SourceRef,
				key.FirstLine,
				key.LastLine,
				seenExpected[key],
			)
		}
	}
}

func reviewedTaskBlockPrimaryReviewsBySource(
	t *testing.T,
	path string,
	wantCount int,
) map[string]reviewedTaskBlockPrimaryReview {
	t.Helper()
	rows := v2ReadJSONL[reviewedTaskBlockPrimaryReview](t, path)
	if len(rows) != wantCount {
		t.Fatalf("primary review fixture %s rows=%d, want %d", path, len(rows), wantCount)
	}
	bySource := make(map[string]reviewedTaskBlockPrimaryReview, len(rows))
	for rowIndex, row := range rows {
		if row.SourceRef == "" || strings.TrimSpace(row.SourceRef) != row.SourceRef {
			t.Fatalf("primary review fixture %s row %d has invalid source_ref %q", path, rowIndex+1, row.SourceRef)
		}
		if row.Classification == "" || strings.TrimSpace(row.Classification) != row.Classification {
			t.Fatalf("primary review fixture %s source %s has invalid classification %q", path, row.SourceRef, row.Classification)
		}
		if strings.TrimSpace(row.Reason) == "" {
			t.Fatalf("primary review fixture %s source %s has empty reason", path, row.SourceRef)
		}
		if _, duplicate := bySource[row.SourceRef]; duplicate {
			t.Fatalf("primary review fixture %s duplicates source %s", path, row.SourceRef)
		}
		bySource[row.SourceRef] = row
	}
	return bySource
}

func reviewedTaskBlockFixtureRef(t *testing.T, prefix, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read review fixture %s: %v", path, err)
	}
	return prefix + "@" + v2SHA(content)
}

func reviewedTaskBlockSortedUniqueReviewValues(
	t *testing.T,
	sourceRef string,
	field string,
	values []string,
) []string {
	t.Helper()
	result := slices.Clone(values)
	slices.Sort(result)
	for index, value := range result {
		if value == "" || strings.TrimSpace(value) != value {
			t.Fatalf("source review %s %s[%d]=%q is empty or untrimmed", sourceRef, field, index, value)
		}
		if index > 0 && result[index-1] == value {
			t.Fatalf("source review %s has duplicate %s %q", sourceRef, field, value)
		}
	}
	return result
}

func reviewedTaskBlockLoadSource(
	t *testing.T,
	cache map[string]reviewedTaskBlockSource,
	sourceRef string,
	legacySources traceGitIndexSnapshot,
) reviewedTaskBlockSource {
	t.Helper()
	if cached, exists := cache[sourceRef]; exists {
		return cached
	}
	clean := filepath.ToSlash(filepath.Clean(sourceRef))
	if sourceRef == "" || filepath.IsAbs(sourceRef) || clean != sourceRef || strings.HasPrefix(clean, "../") {
		t.Fatalf("reviewed task block has non-canonical source_ref %q", sourceRef)
	}
	content := traceReadGitIndexOverlayFile(t, sourceRef, legacySources)
	source := reviewedTaskBlockSource{
		sha256: v2SHA(content),
		lines:  v2SplitLines(content),
	}
	cache[sourceRef] = source
	return source
}

func reviewedTaskBlockStrictlyBefore(left, right v2ReviewedTaskBlock) bool {
	if left.SourceRef != right.SourceRef {
		return left.SourceRef < right.SourceRef
	}
	if left.FirstLine != right.FirstLine {
		return left.FirstLine < right.FirstLine
	}
	return left.LastLine < right.LastLine
}

func reviewedTaskBlockRef(sourceRef string, firstLine, lastLine int, blockSHA string) string {
	subject := strings.Join(
		[]string{sourceRef, strconv.Itoa(firstLine), strconv.Itoa(lastLine), blockSHA},
		"\x00",
	)
	return "TASKBLOCK-" + v2SHAHex([]byte(subject))[:24]
}

func reviewedTaskBlockRequireSortedValues(
	t *testing.T,
	blockRef string,
	field string,
	values []string,
) {
	t.Helper()
	if len(values) == 0 {
		t.Fatalf("reviewed task block %s has empty %s", blockRef, field)
	}
	for index, value := range values {
		if value == "" || strings.TrimSpace(value) != value {
			t.Fatalf("reviewed task block %s %s[%d]=%q is empty or untrimmed", blockRef, field, index, value)
		}
		if index > 0 && values[index-1] >= value {
			t.Fatalf(
				"reviewed task block %s %s not strictly sorted at %q after %q",
				blockRef,
				field,
				value,
				values[index-1],
			)
		}
	}
}

func reviewedTaskBlockValidateScannerEntries(
	t *testing.T,
	block v2ReviewedTaskBlock,
	entries []v2TaskEntry,
) {
	t.Helper()
	for _, entry := range entries {
		if entry.DetectionKind == "reviewed_block" {
			t.Fatalf(
				"scanner_entries block %s references synthetic TaskEntry %s",
				block.BlockRef,
				entry.EntryRef,
			)
		}
		if entry.SourceRef != block.SourceRef || entry.SourceSHA256 != block.SourceSHA256 {
			t.Fatalf(
				"scanner_entries block %s TaskEntry %s source identity differs",
				block.BlockRef,
				entry.EntryRef,
			)
		}
		lineOverlaps := entry.SourceLine >= block.FirstLine && entry.SourceLine <= block.LastLine
		subjectOverlaps := entry.SubjectFirstLine <= block.LastLine &&
			entry.SubjectLastLine >= block.FirstLine
		if !lineOverlaps && !subjectOverlaps {
			t.Fatalf(
				"scanner_entries block %s TaskEntry %s line/subject %d/%d-%d does not overlap %d-%d",
				block.BlockRef,
				entry.EntryRef,
				entry.SourceLine,
				entry.SubjectFirstLine,
				entry.SubjectLastLine,
				block.FirstLine,
				block.LastLine,
			)
		}
	}
}

func reviewedTaskBlockValidateSyntheticEntry(
	t *testing.T,
	block v2ReviewedTaskBlock,
	entries []v2TaskEntry,
) {
	t.Helper()
	if len(entries) != 1 {
		t.Fatalf(
			"reviewed_block_entry %s resolves to %d TaskEntry records, want 1",
			block.BlockRef,
			len(entries),
		)
	}
	entry := entries[0]
	if entry.DetectionKind != "reviewed_block" {
		t.Fatalf(
			"reviewed_block_entry %s TaskEntry %s detection_kind=%q",
			block.BlockRef,
			entry.EntryRef,
			entry.DetectionKind,
		)
	}
	if entry.SourceRef != block.SourceRef ||
		entry.SourceSHA256 != block.SourceSHA256 ||
		entry.SourceLine != block.FirstLine ||
		entry.MarkerSHA256 != block.BlockSHA256 ||
		entry.SubjectFirstLine != block.FirstLine ||
		entry.SubjectLastLine != block.LastLine ||
		entry.SubjectSHA256 != block.BlockSHA256 {
		t.Fatalf(
			"reviewed_block_entry %s TaskEntry %s identity does not equal source/range/block hash",
			block.BlockRef,
			entry.EntryRef,
		)
	}
	candidate := v2TaskCandidate{
		SourceRef:        block.SourceRef,
		SourceSHA256:     block.SourceSHA256,
		Strength:         "strict",
		Kind:             "reviewed_block",
		Line:             block.FirstLine,
		MarkerSHA256:     block.BlockSHA256,
		SubjectFirstLine: block.FirstLine,
		SubjectLastLine:  block.LastLine,
		SubjectSHA256:    block.BlockSHA256,
	}
	if want := v2TaskCandidateRef(candidate); entry.CandidateRef != want {
		t.Fatalf(
			"reviewed_block_entry %s candidate_ref=%s, want %s",
			block.BlockRef,
			entry.CandidateRef,
			want,
		)
	}
}
