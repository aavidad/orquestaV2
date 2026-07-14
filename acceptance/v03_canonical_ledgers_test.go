package acceptance_test

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
)

const v03FixturePath = "acceptance/fixtures/v03_canonical_ledgers.json"

var v03RequiredCandidateSubjects = []string{
	"acceptance/evidence_support_test.go",
	"acceptance/fixtures/v03_canonical_ledgers.json",
	"acceptance/v03_canonical_ledgers_test.go",
	"product/traceability/README.md",
	"product/traceability/disposition_reasons.json",
	"product/traceability/fixtures/task_candidate_adversarial.md",
	"product/traceability/fixtures/task_candidate_counterreview_broad_s2_s3.jsonl",
	"product/traceability/fixtures/task_candidate_counterreview_original_outside_blocks.jsonl",
	"product/traceability/historical_bug_extraction_policy.json",
	"product/traceability/historical_bug_id_review_bindings.jsonl",
	"product/traceability/historical_bug_id_reviews.jsonl",
	"product/traceability/historical_bug_ids.jsonl",
	"product/traceability/historical_bug_occurrences.jsonl",
	"product/traceability/historical_bug_row_enrichments.jsonl",
	"product/traceability/historical_bug_rows.jsonl",
	"product/traceability/legacy_go.json",
	"product/traceability/markdown_source_roles.jsonl",
	"product/traceability/pending_sources.json",
	"product/traceability/reviewed_task_blocks.jsonl",
	"product/traceability/schema.json",
	"product/traceability/source_dispositions.jsonl",
	"product/traceability/task_candidate_exclusions.jsonl",
	"product/traceability/task_candidate_review_provenance.json",
	"product/traceability/task_entries.jsonl",
	"product/traceability/task_extraction_policy.json",
	"product/traceability/task_semantic_review_provenance.json",
	"product_roadmap_test.go",
	"scripts/check_rebuild_write_set.sh",
	"traceability_historical_bug_ids_test.go",
	"traceability_historical_bug_semantics_test.go",
	"traceability_historical_bug_support_test.go",
	"traceability_legacy_census_test.go",
	"traceability_rebuild_bugs_test.go",
	"traceability_reviewed_task_blocks_test.go",
	"traceability_schema_test.go",
	"traceability_source_dispositions_test.go",
	"traceability_source_roles_test.go",
	"traceability_task_candidate_review_provenance_test.go",
	"traceability_task_candidate_review_test.go",
	"traceability_task_review_provenance_test.go",
	"traceability_tasks_v2_test.go",
	"traceability_test_helpers_test.go",
	"traceability_write_set_guard_test.go",
}

var v03RequiredLedgerPaths = []string{
	"product/traceability/disposition_reasons.json",
	"product/traceability/historical_bug_extraction_policy.json",
	"product/traceability/historical_bug_id_review_bindings.jsonl",
	"product/traceability/historical_bug_id_reviews.jsonl",
	"product/traceability/historical_bug_ids.jsonl",
	"product/traceability/historical_bug_occurrences.jsonl",
	"product/traceability/historical_bug_row_enrichments.jsonl",
	"product/traceability/historical_bug_rows.jsonl",
	"product/traceability/legacy_go.json",
	"product/traceability/markdown_source_roles.jsonl",
	"product/traceability/pending_sources.json",
	"product/traceability/reviewed_task_blocks.jsonl",
	"product/traceability/schema.json",
	"product/traceability/source_dispositions.jsonl",
	"product/traceability/task_candidate_exclusions.jsonl",
	"product/traceability/task_candidate_review_provenance.json",
	"product/traceability/task_entries.jsonl",
	"product/traceability/task_extraction_policy.json",
	"product/traceability/task_semantic_review_provenance.json",
}

type v03Fixture struct {
	SchemaVersion           int                     `json:"schema_version"`
	ContractID              string                  `json:"contract_id"`
	Command                 string                  `json:"command"`
	ReceiptPath             string                  `json:"receipt_path"`
	CandidateSubjects       []string                `json:"candidate_subjects"`
	LedgerDigests           []v03LedgerDigest       `json:"ledger_digests"`
	ProhibitedLegacyLedgers []string                `json:"prohibited_legacy_ledgers"`
	SourceDispositions      v03SourceBaseline       `json:"source_dispositions"`
	Tasks                   v03TaskBaseline         `json:"tasks"`
	Bugs                    v03BugBaseline          `json:"bugs"`
	ClosureInferredCount    int                     `json:"closure_inferred_count"`
	RejectedTask            v03RejectedTask         `json:"rejected_task"`
	DelegatedGroupB         v03DelegatedGroupB      `json:"delegated_group_b"`
	TransformedSkill        v03TransformedSkill     `json:"transformed_skill"`
	CriticalBugMappings     []v03CriticalBugMapping `json:"critical_bug_mappings"`
}

type v03LedgerDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type v03SourceBaseline struct {
	Total int `json:"total"`
	Task  int `json:"task"`
	Bug   int `json:"bug"`
	Skill int `json:"skill"`
}

type v03TaskBaseline struct {
	SourceCount                    int `json:"source_count"`
	StrictCandidateCount           int `json:"strict_candidate_count"`
	BroadCandidateCount            int `json:"broad_candidate_count"`
	EntryCount                     int `json:"entry_count"`
	ExcludedBroadCandidateCount    int `json:"excluded_broad_candidate_count"`
	IndependentReviewUniverseCount int `json:"independent_review_universe_count"`
	ConfirmedCount                 int `json:"confirmed_count"`
	CorrectedCount                 int `json:"corrected_count"`
	PendingSemanticReviewCount     int `json:"pending_semantic_review_count"`
	CandidateReviewPassCount       int `json:"candidate_review_pass_count"`
	CandidateReviewInputCount      int `json:"candidate_review_input_count"`
	CandidateReviewEntryCount      int `json:"candidate_review_entry_count"`
	CandidateReviewExclusionCount  int `json:"candidate_review_exclusion_count"`
	ReviewedBlockCount             int `json:"reviewed_block_count"`
	ReviewedBlockEntryCount        int `json:"reviewed_block_entry_count"`
}

type v03BugBaseline struct {
	SourceCount          int `json:"source_count"`
	RichRowCount         int `json:"rich_row_count"`
	OccurrenceCount      int `json:"occurrence_count"`
	IDCount              int `json:"id_count"`
	RowCoveredIDCount    int `json:"row_covered_id_count"`
	NarrativeOnlyIDCount int `json:"narrative_only_id_count"`
}

type v03RejectedTask struct {
	SourceRef                string   `json:"source_ref"`
	SourceEntryID            string   `json:"source_entry_id"`
	CapabilityID             string   `json:"capability_id"`
	CapabilityDecision       string   `json:"capability_decision"`
	ReplacementCapabilityIDs []string `json:"replacement_capability_ids"`
}

type v03DelegatedGroupB struct {
	SourceRef              string `json:"source_ref"`
	ExcludedCandidateCount int    `json:"excluded_candidate_count"`
	ReasonCode             string `json:"reason_code"`
	DelegatedLedgerRef     string `json:"delegated_ledger_ref"`
}

type v03TransformedSkill struct {
	SourceRef          string   `json:"source_ref"`
	Decision           string   `json:"decision"`
	CapabilityIDs      []string `json:"capability_ids"`
	TransformationNote string   `json:"transformation_note"`
}

type v03CriticalBugMapping struct {
	BugID         string   `json:"bug_id"`
	CapabilityIDs []string `json:"capability_ids"`
}

type v03TaskPolicy struct {
	DocumentKind  string `json:"document_kind"`
	SchemaVersion int    `json:"schema_version"`
	Baseline      struct {
		SourceCount                      int `json:"source_count"`
		StrictCandidateCount             int `json:"strict_candidate_count"`
		BroadCandidateCount              int `json:"broad_candidate_count"`
		ExcludedBroadCandidateCount      int `json:"excluded_broad_candidate_count"`
		EntryCount                       int `json:"entry_count"`
		IndependentReviewUniverseCount   int `json:"independent_review_universe_count"`
		IntegratedIndependentReviewCount int `json:"integrated_independent_review_count"`
		ConfirmedCount                   int `json:"confirmed_count"`
		CorrectedCount                   int `json:"corrected_count"`
		PendingSemanticReviewCount       int `json:"pending_semantic_review_count"`
		CandidateReviewPassCount         int `json:"candidate_review_pass_count"`
		CandidateReviewInputCount        int `json:"candidate_review_input_count"`
		CandidateReviewEntryCount        int `json:"candidate_review_entry_count"`
		CandidateReviewExclusionCount    int `json:"candidate_review_exclusion_count"`
		ReviewedBlockCount               int `json:"reviewed_block_count"`
		ReviewedBlockEntryCount          int `json:"reviewed_block_entry_count"`
	} `json:"baseline"`
}

type v03BugPolicy struct {
	DocumentKind  string `json:"document_kind"`
	SchemaVersion int    `json:"schema_version"`
	Baseline      struct {
		SourceCount          int `json:"source_count"`
		RichRowCount         int `json:"rich_row_count"`
		OccurrenceCount      int `json:"occurrence_count"`
		IDCount              int `json:"id_count"`
		RowCoveredIDCount    int `json:"row_covered_id_count"`
		NarrativeOnlyIDCount int `json:"narrative_only_id_count"`
	} `json:"baseline"`
}

type v03SourceDisposition struct {
	Kind               string   `json:"kind"`
	AdditionalRoles    []string `json:"additional_roles"`
	SourceRef          string   `json:"source_ref"`
	Decision           string   `json:"decision"`
	CapabilityIDs      []string `json:"capability_ids"`
	ReasonCode         string   `json:"reason_code"`
	EvidenceState      string   `json:"evidence_state"`
	TransformationNote string   `json:"transformation_note"`
	DelegatedLedgerRef string   `json:"delegated_ledger_ref"`
}

type v03TaskEntry struct {
	SchemaVersion            int      `json:"schema_version"`
	SourceRef                string   `json:"source_ref"`
	SourceEntryID            string   `json:"source_entry_id"`
	CapabilityID             string   `json:"capability_id"`
	CapabilityDecision       string   `json:"capability_decision"`
	ReplacementCapabilityIDs []string `json:"replacement_capability_ids"`
	BasisCode                string   `json:"basis_code"`
	SemanticReviewVerdict    string   `json:"semantic_review_verdict"`
	ClosureEvidence          string   `json:"closure_evidence"`
}

type v03TaskExclusion struct {
	SchemaVersion int    `json:"schema_version"`
	SourceRef     string `json:"source_ref"`
	ReasonCode    string `json:"reason_code"`
	ReviewPassRef string `json:"review_pass_ref"`
}

type v03ReviewedTaskBlock struct {
	CoverageKind string `json:"coverage_kind"`
}

type v03HistoricalBugReview struct {
	BugID         string   `json:"bug_id"`
	CapabilityIDs []string `json:"capability_ids"`
}

type v03ClosureRecord struct {
	ClosureEvidence string `json:"closure_evidence"`
}

func TestAcceptanceV03CanonicalLedgers(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v03Fixture](t, filepath.Join(repositoryRoot, v03FixturePath))
	v03ValidateFixture(t, fixture)

	t.Run("static_ledgers_match_exact_digests", func(t *testing.T) {
		for _, ledger := range fixture.LedgerDigests {
			got := evidenceFileSHA256(t, filepath.Join(repositoryRoot, filepath.FromSlash(ledger.Path)))
			if got != ledger.SHA256 {
				t.Errorf("ledger %s digest=%s, want %s", ledger.Path, got, ledger.SHA256)
			}
		}
		for _, prohibited := range fixture.ProhibitedLegacyLedgers {
			_, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(prohibited)))
			if !os.IsNotExist(err) {
				t.Errorf("parallel V1 ledger %s must not exist: %v", prohibited, err)
			}
		}
	})

	t.Run("canonical_counts_and_semantic_receipts_match", func(t *testing.T) {
		v03AssertCanonicalCounts(t, repositoryRoot, fixture)
	})

	t.Run("operator_and_migration_decisions_are_preserved", func(t *testing.T) {
		v03AssertDecisions(t, repositoryRoot, fixture)
	})

	t.Run("historical_state_never_infers_rebuild_closure", func(t *testing.T) {
		v03AssertNoInferredClosure(t, repositoryRoot, fixture.ClosureInferredCount)
	})

}

func TestAcceptanceV03CanonicalLedgersReceipt(t *testing.T) {
	repositoryRoot := evidenceRepositoryRoot(t)
	fixture := evidenceDecodeStrictJSON[v03Fixture](t, filepath.Join(repositoryRoot, v03FixturePath))
	v03ValidateFixture(t, fixture)
	evidenceAssertReceiptV2(t, repositoryRoot, evidenceReceiptV2Expectation{
		Contract: fixture.ContractID, ValidationCommand: fixture.Command,
		ExecutionArgv: []string{
			"go", "test", "-mod=vendor", "-count=1", ".", "./acceptance", "-run",
			"^(TestTraceabilityRebuild.*|TestProductRoadmap.*|TestAcceptanceV03CanonicalLedgers)$",
		},
		OutputPath: "product/evidence/v03_canonical_ledgers.output.txt", FixturePath: v03FixturePath,
		ReceiptPath: fixture.ReceiptPath, CandidateSubjects: fixture.CandidateSubjects,
		ExecutedNotBefore: "2026-07-14T00:00:00+02:00",
		ExpectedGitHead:   "a301a3bbacd80c1ea2d47422a2964339dcd70980",
	})
}

func v03ValidateFixture(t *testing.T, fixture v03Fixture) {
	t.Helper()
	if fixture.SchemaVersion != 1 || fixture.ContractID != "AC-V03-CANONICAL-LEDGERS" ||
		fixture.Command != "go test -mod=vendor -count=1 . ./acceptance -run '^(TestTraceabilityRebuild.*|TestProductRoadmap.*|TestAcceptanceV03CanonicalLedgers)$'" ||
		fixture.ReceiptPath != "product/evidence/v03_canonical_ledgers.json" {
		t.Fatalf("invalid V03 fixture identity: %+v", fixture)
	}
	if !sort.StringsAreSorted(fixture.CandidateSubjects) || !slices.Equal(fixture.CandidateSubjects, v03RequiredCandidateSubjects) {
		t.Fatal("V03 candidate subjects and ledger digests must be non-empty and ordered")
	}
	for _, subject := range fixture.CandidateSubjects {
		if subject == fixture.ReceiptPath || subject == "product/roadmap.json" || subject == "product/traceability/rebuild_bugs.jsonl" {
			t.Fatalf("mutable or self-referential V03 candidate subject %q", subject)
		}
	}
	previous := ""
	ledgerPaths := make([]string, 0, len(fixture.LedgerDigests))
	for _, ledger := range fixture.LedgerDigests {
		if ledger.Path <= previous || !strings.HasPrefix(ledger.SHA256, "sha256:") || len(ledger.SHA256) != len("sha256:")+64 {
			t.Fatalf("invalid or unordered ledger digest: %+v", ledger)
		}
		previous = ledger.Path
		ledgerPaths = append(ledgerPaths, ledger.Path)
	}
	if !slices.Equal(ledgerPaths, v03RequiredLedgerPaths) {
		t.Fatalf("V03 static ledger set=%v, want %v", ledgerPaths, v03RequiredLedgerPaths)
	}
	if fixture.Tasks.PendingSemanticReviewCount != 0 || fixture.ClosureInferredCount != 0 ||
		fixture.Tasks.ConfirmedCount+fixture.Tasks.CorrectedCount != fixture.Tasks.IndependentReviewUniverseCount ||
		fixture.Tasks.CandidateReviewInputCount != fixture.Tasks.CandidateReviewEntryCount+fixture.Tasks.CandidateReviewExclusionCount ||
		fixture.SourceDispositions.Task+fixture.SourceDispositions.Bug+fixture.SourceDispositions.Skill != fixture.SourceDispositions.Total {
		t.Fatalf("internally inconsistent V03 fixture baselines: %+v", fixture)
	}
	if len(fixture.ProhibitedLegacyLedgers) != 2 || len(fixture.CriticalBugMappings) != 3 {
		t.Fatalf("incomplete V03 negative/critical controls: %+v", fixture)
	}
}

func v03AssertCanonicalCounts(t *testing.T, repositoryRoot string, fixture v03Fixture) {
	t.Helper()
	var taskPolicy v03TaskPolicy
	v03DecodeJSON(t, filepath.Join(repositoryRoot, "product/traceability/task_extraction_policy.json"), &taskPolicy)
	if taskPolicy.DocumentKind != "task_entry_extraction_policy" || taskPolicy.SchemaVersion != 2 {
		t.Fatalf("task policy is not the V2 authority: %+v", taskPolicy)
	}
	gotTask := v03TaskBaseline{
		SourceCount:                    taskPolicy.Baseline.SourceCount,
		StrictCandidateCount:           taskPolicy.Baseline.StrictCandidateCount,
		BroadCandidateCount:            taskPolicy.Baseline.BroadCandidateCount,
		EntryCount:                     taskPolicy.Baseline.EntryCount,
		ExcludedBroadCandidateCount:    taskPolicy.Baseline.ExcludedBroadCandidateCount,
		IndependentReviewUniverseCount: taskPolicy.Baseline.IndependentReviewUniverseCount,
		ConfirmedCount:                 taskPolicy.Baseline.ConfirmedCount,
		CorrectedCount:                 taskPolicy.Baseline.CorrectedCount,
		PendingSemanticReviewCount:     taskPolicy.Baseline.PendingSemanticReviewCount,
		CandidateReviewPassCount:       taskPolicy.Baseline.CandidateReviewPassCount,
		CandidateReviewInputCount:      taskPolicy.Baseline.CandidateReviewInputCount,
		CandidateReviewEntryCount:      taskPolicy.Baseline.CandidateReviewEntryCount,
		CandidateReviewExclusionCount:  taskPolicy.Baseline.CandidateReviewExclusionCount,
		ReviewedBlockCount:             taskPolicy.Baseline.ReviewedBlockCount,
		ReviewedBlockEntryCount:        taskPolicy.Baseline.ReviewedBlockEntryCount,
	}
	if !reflect.DeepEqual(gotTask, fixture.Tasks) ||
		taskPolicy.Baseline.IntegratedIndependentReviewCount != fixture.Tasks.IndependentReviewUniverseCount {
		t.Fatalf("task policy baseline=%+v integrated=%d, want %+v", gotTask, taskPolicy.Baseline.IntegratedIndependentReviewCount, fixture.Tasks)
	}

	entries := v03ReadJSONL[v03TaskEntry](t, filepath.Join(repositoryRoot, "product/traceability/task_entries.jsonl"))
	if len(entries) != fixture.Tasks.EntryCount {
		t.Fatalf("task entries=%d, want %d", len(entries), fixture.Tasks.EntryCount)
	}
	verdicts := map[string]int{"confirmed": 0, "corrected": 0}
	independent := 0
	candidateReviewedEntries := 0
	for _, entry := range entries {
		if entry.SchemaVersion != 2 {
			t.Fatalf("task entry uses non-V2 schema: %+v", entry)
		}
		if entry.BasisCode == "independent_semantic_review" {
			independent++
			if _, exists := verdicts[entry.SemanticReviewVerdict]; !exists {
				t.Fatalf("independent task entry lacks a closed verdict: %+v", entry)
			}
			verdicts[entry.SemanticReviewVerdict]++
		} else if entry.BasisCode == "independent_candidate_semantic_review" {
			candidateReviewedEntries++
		}
	}
	if independent != fixture.Tasks.IndependentReviewUniverseCount || verdicts["confirmed"] != fixture.Tasks.ConfirmedCount || verdicts["corrected"] != fixture.Tasks.CorrectedCount {
		t.Fatalf("integrated task semantic reviews=%d confirmed=%d corrected=%d, want %+v", independent, verdicts["confirmed"], verdicts["corrected"], fixture.Tasks)
	}
	if candidateReviewedEntries != fixture.Tasks.CandidateReviewEntryCount {
		t.Fatalf("integrated candidate-reviewed task entries=%d, want %d", candidateReviewedEntries, fixture.Tasks.CandidateReviewEntryCount)
	}
	exclusions := v03ReadJSONL[v03TaskExclusion](t, filepath.Join(repositoryRoot, "product/traceability/task_candidate_exclusions.jsonl"))
	candidateReviewedExclusions := 0
	for _, exclusion := range exclusions {
		if exclusion.ReviewPassRef != "" {
			candidateReviewedExclusions++
		}
	}
	if candidateReviewedExclusions != fixture.Tasks.CandidateReviewExclusionCount {
		t.Fatalf("integrated candidate-reviewed exclusions=%d, want %d", candidateReviewedExclusions, fixture.Tasks.CandidateReviewExclusionCount)
	}
	blocks := v03ReadJSONL[v03ReviewedTaskBlock](t, filepath.Join(repositoryRoot, "product/traceability/reviewed_task_blocks.jsonl"))
	reviewedBlockEntries := 0
	for _, block := range blocks {
		if block.CoverageKind == "reviewed_block_entry" {
			reviewedBlockEntries++
		}
	}
	if len(blocks) != fixture.Tasks.ReviewedBlockCount || reviewedBlockEntries != fixture.Tasks.ReviewedBlockEntryCount {
		t.Fatalf("reviewed task blocks=%d synthetic=%d, want %d/%d", len(blocks), reviewedBlockEntries, fixture.Tasks.ReviewedBlockCount, fixture.Tasks.ReviewedBlockEntryCount)
	}

	var bugPolicy v03BugPolicy
	v03DecodeJSON(t, filepath.Join(repositoryRoot, "product/traceability/historical_bug_extraction_policy.json"), &bugPolicy)
	if bugPolicy.DocumentKind != "historical_bug_id_extraction_policy" || bugPolicy.SchemaVersion != 1 {
		t.Fatalf("invalid historical bug policy: %+v", bugPolicy)
	}
	gotBug := v03BugBaseline{
		SourceCount:          bugPolicy.Baseline.SourceCount,
		RichRowCount:         bugPolicy.Baseline.RichRowCount,
		OccurrenceCount:      bugPolicy.Baseline.OccurrenceCount,
		IDCount:              bugPolicy.Baseline.IDCount,
		RowCoveredIDCount:    bugPolicy.Baseline.RowCoveredIDCount,
		NarrativeOnlyIDCount: bugPolicy.Baseline.NarrativeOnlyIDCount,
	}
	if !reflect.DeepEqual(gotBug, fixture.Bugs) {
		t.Fatalf("historical bug policy baseline=%+v, want %+v", gotBug, fixture.Bugs)
	}

	sources := v03ReadJSONL[v03SourceDisposition](t, filepath.Join(repositoryRoot, "product/traceability/source_dispositions.jsonl"))
	kindCounts := make(map[string]int)
	for _, source := range sources {
		kindCounts[source.Kind]++
	}
	gotSources := v03SourceBaseline{Total: len(sources), Task: kindCounts["task"], Bug: kindCounts["bug"], Skill: kindCounts["skill"]}
	if !reflect.DeepEqual(gotSources, fixture.SourceDispositions) {
		t.Fatalf("source dispositions=%+v, want %+v", gotSources, fixture.SourceDispositions)
	}
}

func v03AssertDecisions(t *testing.T, repositoryRoot string, fixture v03Fixture) {
	t.Helper()
	entries := v03ReadJSONL[v03TaskEntry](t, filepath.Join(repositoryRoot, "product/traceability/task_entries.jsonl"))
	var rejected []v03TaskEntry
	for _, entry := range entries {
		if entry.SourceRef == fixture.RejectedTask.SourceRef && entry.SourceEntryID == fixture.RejectedTask.SourceEntryID {
			rejected = append(rejected, entry)
		}
	}
	if len(rejected) != 1 || rejected[0].CapabilityID != fixture.RejectedTask.CapabilityID ||
		rejected[0].CapabilityDecision != fixture.RejectedTask.CapabilityDecision ||
		!slices.Equal(rejected[0].ReplacementCapabilityIDs, fixture.RejectedTask.ReplacementCapabilityIDs) {
		t.Fatalf("operator rejection T-6 drift: %+v, want %+v", rejected, fixture.RejectedTask)
	}

	sources := v03ReadJSONL[v03SourceDisposition](t, filepath.Join(repositoryRoot, "product/traceability/source_dispositions.jsonl"))
	byRef := make(map[string]v03SourceDisposition, len(sources))
	for _, source := range sources {
		byRef[source.SourceRef] = source
	}
	groupB, exists := byRef[fixture.DelegatedGroupB.SourceRef]
	if !exists || groupB.ReasonCode != "historical_task_candidates_delegated_to_bug_ledger" || groupB.DelegatedLedgerRef != fixture.DelegatedGroupB.DelegatedLedgerRef {
		t.Fatalf("Group B delegation drift: %+v", groupB)
	}
	exclusions := v03ReadJSONL[v03TaskExclusion](t, filepath.Join(repositoryRoot, "product/traceability/task_candidate_exclusions.jsonl"))
	delegatedCount := 0
	for _, exclusion := range exclusions {
		if exclusion.SourceRef == fixture.DelegatedGroupB.SourceRef {
			if exclusion.SchemaVersion != 2 || exclusion.ReasonCode != fixture.DelegatedGroupB.ReasonCode {
				t.Fatalf("Group B exclusion is not delegated to bugs: %+v", exclusion)
			}
			delegatedCount++
		}
	}
	if delegatedCount != fixture.DelegatedGroupB.ExcludedCandidateCount {
		t.Fatalf("Group B delegated candidates=%d, want %d", delegatedCount, fixture.DelegatedGroupB.ExcludedCandidateCount)
	}

	skill, exists := byRef[fixture.TransformedSkill.SourceRef]
	if !exists || skill.Decision != fixture.TransformedSkill.Decision ||
		!slices.Equal(skill.CapabilityIDs, fixture.TransformedSkill.CapabilityIDs) ||
		skill.TransformationNote != fixture.TransformedSkill.TransformationNote {
		t.Fatalf("runtime model skill transformation drift: %+v, want %+v", skill, fixture.TransformedSkill)
	}

	reviews := v03ReadJSONL[v03HistoricalBugReview](t, filepath.Join(repositoryRoot, "product/traceability/historical_bug_id_reviews.jsonl"))
	capabilitiesByBug := make(map[string][]string, len(reviews))
	for _, review := range reviews {
		capabilitiesByBug[review.BugID] = review.CapabilityIDs
	}
	for _, mapping := range fixture.CriticalBugMappings {
		if got := capabilitiesByBug[mapping.BugID]; !slices.Equal(got, mapping.CapabilityIDs) {
			t.Errorf("critical bug %s capabilities=%v, want %v", mapping.BugID, got, mapping.CapabilityIDs)
		}
	}
}

func v03AssertNoInferredClosure(t *testing.T, repositoryRoot string, want int) {
	t.Helper()
	inferred := 0
	for _, relative := range []string{
		"product/traceability/task_entries.jsonl",
		"product/traceability/historical_bug_rows.jsonl",
		"product/traceability/historical_bug_ids.jsonl",
	} {
		for _, record := range v03ReadJSONL[v03ClosureRecord](t, filepath.Join(repositoryRoot, relative)) {
			if record.ClosureEvidence != "not_verified" {
				inferred++
			}
		}
	}
	if inferred != want {
		t.Fatalf("historical records with inferred rebuild closure=%d, want %d", inferred, want)
	}
}

func v03DecodeJSON(t *testing.T, filename string, target any) {
	t.Helper()
	handle, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	decoder := json.NewDecoder(handle)
	if err := decoder.Decode(target); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("trailing JSON in %s: %v", filename, err)
	}
}

func v03ReadJSONL[T any](t *testing.T, filename string) []T {
	t.Helper()
	handle, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	var rows []T
	scanner := bufio.NewScanner(handle)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		content := scanner.Bytes()
		if len(strings.TrimSpace(string(content))) == 0 {
			t.Fatalf("blank JSONL row in %s:%d", filename, line)
		}
		var row T
		if err := json.Unmarshal(content, &row); err != nil {
			t.Fatalf("decode %s:%d: %v", filename, line, err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}
