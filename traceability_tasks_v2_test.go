package orquesta_test

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	v2TaskEntriesPath    = "product/traceability/task_entries.jsonl"
	v2TaskExclusionsPath = "product/traceability/task_candidate_exclusions.jsonl"
	v2TaskSourcesPath    = "product/traceability/source_dispositions.jsonl"
	v2TaskPolicyPath     = "product/traceability/task_extraction_policy.json"
	v2TaskProvenancePath = "product/traceability/task_semantic_review_provenance.json"
	v2GroupBSource       = "docs/historico/2026-06/TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md"
	v2RoadmapSource      = "docs/historico/2026-06/TAREA_OPES_ORQUESTA_ROADMAP_MEJORAS_2026-06-27.md"
)

type v2TaskPolicy struct {
	DocumentKind                       string   `json:"document_kind"`
	SchemaVersion                      int      `json:"schema_version"`
	SourcesAuthority                   string   `json:"sources_authority"`
	EntriesAuthority                   string   `json:"entries_authority"`
	ExclusionsAuthority                string   `json:"exclusions_authority"`
	SemanticReviewProvenanceAuthority  string   `json:"semantic_review_provenance_authority"`
	CandidateReviewProvenanceAuthority string   `json:"candidate_review_provenance_authority"`
	ReviewedBlocksAuthority            string   `json:"reviewed_blocks_authority"`
	CapabilityAuthority                string   `json:"capability_authority"`
	Fixture                            string   `json:"fixture"`
	Precedence                         []string `json:"precedence"`
	LineForms                          struct {
		Checkbox string `json:"checkbox"`
		IDField  string `json:"id_field"`
		Heading  string `json:"heading"`
		List     string `json:"list"`
		Table    string `json:"table"`
	} `json:"line_forms"`
	IDGrammar struct {
		StrictPatterns                        []string `json:"strict_patterns"`
		StructuredIDRequiresDigitUnlessPrefix string   `json:"structured_id_requires_digit_unless_prefix"`
		BroadPatterns                         []string `json:"broad_patterns"`
		MarkupPrefixesRemovedBeforeFirstToken []string `json:"markup_prefixes_removed_before_first_token"`
		ShortTokensWithoutDigitAreNotIDs      bool     `json:"short_tokens_without_digit_are_not_ids"`
	} `json:"id_grammar"`
	CandidateRule struct {
		Strict                                 []string `json:"strict"`
		Broad                                  []string `json:"broad"`
		GenericH2IsCandidate                   bool     `json:"generic_h2_is_candidate"`
		EveryBroadCandidateRequiresEntryOrExcl bool     `json:"every_broad_candidate_requires_entry_or_exclusion"`
		StrictCandidatesAreEntries             bool     `json:"strict_candidates_are_entries"`
	} `json:"candidate_rule"`
	SubjectRule struct {
		Heading              string `json:"heading"`
		FencedIDField        string `json:"fenced_id_field"`
		UnfencedIDField      string `json:"unfenced_id_field"`
		ListOrCheckbox       string `json:"list_or_checkbox"`
		Table                string `json:"table"`
		ReviewedBlock        string `json:"reviewed_block"`
		LineRangeHashSubject string `json:"line_range_hash_subject"`
	} `json:"subject_rule"`
	HashRule struct {
		Algorithm                     string `json:"algorithm"`
		MarkerSubject                 string `json:"marker_subject"`
		CandidateRefSubject           string `json:"candidate_ref_subject"`
		EntryRefSubject               string `json:"entry_ref_subject"`
		ExclusionRefSubject           string `json:"exclusion_ref_subject"`
		SemanticReviewRefSubject      string `json:"semantic_review_ref_subject"`
		SemanticReviewsDigestSubject  string `json:"semantic_reviews_digest_subject"`
		CandidateReviewRefSubject     string `json:"candidate_review_ref_subject"`
		CandidateReviewPassRefSubject string `json:"candidate_review_pass_ref_subject"`
		ReviewedBlockRefSubject       string `json:"reviewed_block_ref_subject"`
		LedgerDigestSubject           string `json:"ledger_digest_subject"`
	} `json:"hash_rule"`
	SourceHistoryValues      []string `json:"source_history_values"`
	CapabilityDecisionValues []string `json:"capability_decision_values"`
	DispositionRule          struct {
		CurrentAccept    string `json:"current_accept"`
		HistoricalAccept string `json:"historical_accept"`
		Reject           string `json:"reject"`
		Conditional      string `json:"conditional"`
	} `json:"disposition_rule"`
	BasisCodes           []string `json:"basis_codes"`
	ExclusionReasonCodes []string `json:"exclusion_reason_codes"`
	ClosurePolicy        string   `json:"closure_policy"`
	Baseline             struct {
		SourceCount                      int    `json:"source_count"`
		SourceWithoutCandidateCount      int    `json:"source_without_candidate_count"`
		SourceWithoutEntryCount          int    `json:"source_without_entry_count"`
		StrictCandidateCount             int    `json:"strict_candidate_count"`
		BroadCandidateCount              int    `json:"broad_candidate_count"`
		IncludedBroadCandidateCount      int    `json:"included_broad_candidate_count"`
		ExcludedBroadCandidateCount      int    `json:"excluded_broad_candidate_count"`
		EntryCount                       int    `json:"entry_count"`
		IndependentReviewUniverseCount   int    `json:"independent_review_universe_count"`
		IntegratedIndependentReviewCount int    `json:"integrated_independent_review_count"`
		ConfirmedCount                   int    `json:"confirmed_count"`
		CorrectedCount                   int    `json:"corrected_count"`
		SemanticReviewsSHA256            string `json:"semantic_reviews_sha256"`
		SemanticReviewProvenanceSHA256   string `json:"semantic_review_provenance_sha256"`
		PendingSemanticReviewCount       int    `json:"pending_semantic_review_count"`
		EntriesSHA256                    string `json:"entries_sha256"`
		ExclusionsSHA256                 string `json:"exclusions_sha256"`
		CandidateReviewPassCount         int    `json:"candidate_review_pass_count"`
		CandidateReviewInputCount        int    `json:"candidate_review_input_count"`
		CandidateReviewEntryCount        int    `json:"candidate_review_entry_count"`
		CandidateReviewExclusionCount    int    `json:"candidate_review_exclusion_count"`
		CandidateReviewProvenanceSHA256  string `json:"candidate_review_provenance_sha256"`
		ReviewedBlockCount               int    `json:"reviewed_block_count"`
		ReviewedBlockEntryCount          int    `json:"reviewed_block_entry_count"`
		ReviewedBlocksSHA256             string `json:"reviewed_blocks_sha256"`
	} `json:"baseline"`
}

type v2TaskSource struct {
	SchemaVersion      int      `json:"schema_version"`
	Kind               string   `json:"kind"`
	AdditionalRoles    []string `json:"additional_roles,omitempty"`
	SourceRef          string   `json:"source_ref"`
	SourceSHA256       string   `json:"source_sha256"`
	Decision           string   `json:"decision"`
	CapabilityIDs      []string `json:"capability_ids"`
	ReasonCode         string   `json:"reason_code"`
	EvidenceState      string   `json:"evidence_state"`
	LessonID           string   `json:"lesson_id,omitempty"`
	InvariantTestRef   string   `json:"invariant_test_ref,omitempty"`
	LessonState        string   `json:"lesson_state,omitempty"`
	ScopeRef           string   `json:"scope_ref,omitempty"`
	TrustRef           string   `json:"trust_ref,omitempty"`
	ActivationTestRef  string   `json:"activation_test_ref,omitempty"`
	ReviewBasis        string   `json:"review_basis,omitempty"`
	ReviewNote         string   `json:"review_note,omitempty"`
	TransformationNote string   `json:"transformation_note,omitempty"`
	DelegatedLedgerRef string   `json:"delegated_ledger_ref,omitempty"`
}

type v2TaskEntry struct {
	SchemaVersion            int      `json:"schema_version"`
	EntryRef                 string   `json:"entry_ref"`
	CandidateRef             string   `json:"candidate_ref"`
	SourceRef                string   `json:"source_ref"`
	SourceSHA256             string   `json:"source_sha256"`
	SourceHistory            string   `json:"source_history"`
	SourceLine               int      `json:"source_line"`
	DetectionKind            string   `json:"detection_kind"`
	SourceEntryID            string   `json:"source_entry_id,omitempty"`
	MarkerSHA256             string   `json:"marker_sha256"`
	SubjectFirstLine         int      `json:"subject_first_line"`
	SubjectLastLine          int      `json:"subject_last_line"`
	SubjectSHA256            string   `json:"subject_sha256"`
	OriginalState            string   `json:"original_state"`
	CapabilityID             string   `json:"capability_id"`
	CapabilityDecision       string   `json:"capability_decision"`
	Disposition              string   `json:"disposition"`
	BasisCode                string   `json:"basis_code"`
	SemanticReviewState      string   `json:"semantic_review_state"`
	SemanticReason           string   `json:"semantic_reason"`
	SemanticReviewPassRef    string   `json:"semantic_review_pass_ref,omitempty"`
	SemanticReviewRef        string   `json:"semantic_review_ref,omitempty"`
	SemanticReviewVerdict    string   `json:"semantic_review_verdict,omitempty"`
	PreviousCapabilityID     string   `json:"previous_capability_id,omitempty"`
	ReplacementCapabilityIDs []string `json:"replacement_capability_ids,omitempty"`
	DecisionRef              string   `json:"decision_ref"`
	ClosureEvidence          string   `json:"closure_evidence"`
}

type v2TaskExclusion struct {
	SchemaVersion    int    `json:"schema_version"`
	ExclusionRef     string `json:"exclusion_ref"`
	CandidateRef     string `json:"candidate_ref"`
	SourceRef        string `json:"source_ref"`
	SourceSHA256     string `json:"source_sha256"`
	SourceLine       int    `json:"source_line"`
	DetectionKind    string `json:"detection_kind"`
	SourceEntryID    string `json:"source_entry_id,omitempty"`
	MarkerSHA256     string `json:"marker_sha256"`
	SubjectFirstLine int    `json:"subject_first_line"`
	SubjectLastLine  int    `json:"subject_last_line"`
	SubjectSHA256    string `json:"subject_sha256"`
	ReasonCode       string `json:"reason_code"`
	ReviewState      string `json:"review_state"`
	ReviewPassRef    string `json:"review_pass_ref,omitempty"`
	ReviewRef        string `json:"review_ref,omitempty"`
	SemanticReason   string `json:"semantic_reason,omitempty"`
}

type v2ReviewedTaskBlock struct {
	SchemaVersion   int      `json:"schema_version"`
	BlockRef        string   `json:"block_ref"`
	SourceRef       string   `json:"source_ref"`
	SourceSHA256    string   `json:"source_sha256"`
	FirstLine       int      `json:"first_line"`
	LastLine        int      `json:"last_line"`
	BlockSHA256     string   `json:"block_sha256"`
	Summary         string   `json:"summary"`
	CapabilityHints []string `json:"capability_hints"`
	ReviewRefs      []string `json:"review_refs"`
	CoverageKind    string   `json:"coverage_kind"`
	CandidateRefs   []string `json:"candidate_refs"`
}

type v2TaskCandidate struct {
	SourceRef        string
	SourceSHA256     string
	Strength         string
	Kind             string
	ID               string
	Line             int
	MarkerSHA256     string
	SubjectFirstLine int
	SubjectLastLine  int
	SubjectSHA256    string
}

type v2Capability struct {
	Decision string
	Title    string
}

func TestTraceabilityRebuildTaskEntries(t *testing.T) {
	traceTestCanonicalTaskEntriesV2(t)
}

func TestTraceabilityRebuildIndependentSemanticReviewsHaveNoGeneratedFallbacks(t *testing.T) {
	entries := v2ReadJSONL[v2TaskEntry](t, v2TaskEntriesPath)
	invalid := make([]string, 0)
	independent := 0
	for _, entry := range entries {
		if entry.BasisCode != "independent_semantic_review" {
			continue
		}
		independent++
		if v2GeneratedSemanticFallbackRE.MatchString(entry.SemanticReason) {
			invalid = append(invalid, entry.EntryRef)
			continue
		}
		if _, generated := v2GeneratedSemanticCannedReasons[entry.SemanticReason]; generated {
			invalid = append(invalid, entry.EntryRef)
		}
	}
	if independent != 696 {
		t.Fatalf("independent semantic review universe=%d, want 696", independent)
	}
	if len(invalid) != 0 {
		limit := min(len(invalid), 12)
		t.Fatalf("%d independent semantic receipts still use a generated confirmation fallback; sample=%v", len(invalid), invalid[:limit])
	}
}

func traceTestCanonicalTaskEntriesV2(t *testing.T) {
	t.Helper()
	var policy v2TaskPolicy
	v2DecodeStrict(t, v2TaskPolicyPath, &policy)
	v2ValidateTaskPolicy(t, policy)

	sources := v2ReadJSONL[v2TaskSource](t, v2TaskSourcesPath)
	legacySources := traceLoadLegacySourceSnapshot(t, ".")
	entries := v2ReadJSONL[v2TaskEntry](t, v2TaskEntriesPath)
	exclusions := v2ReadJSONL[v2TaskExclusion](t, v2TaskExclusionsPath)
	capabilities := v2ReadCapabilities(t)

	taskSources := make(map[string]v2TaskSource)
	var orderedTaskSources []v2TaskSource
	var runtimeModelsSkill v2TaskSource
	for _, source := range sources {
		if source.SourceRef == "skills/orquesta-runtime-modelos/SKILL.md" {
			runtimeModelsSkill = source
		}
		if source.Kind != "task" {
			continue
		}
		if _, duplicate := taskSources[source.SourceRef]; duplicate {
			t.Fatalf("duplicate task source %q", source.SourceRef)
		}
		taskSources[source.SourceRef] = source
		orderedTaskSources = append(orderedTaskSources, source)
	}
	if len(taskSources) != policy.Baseline.SourceCount {
		t.Fatalf("task sources=%d, want %d", len(taskSources), policy.Baseline.SourceCount)
	}
	sort.Slice(orderedTaskSources, func(i, j int) bool { return orderedTaskSources[i].SourceRef < orderedTaskSources[j].SourceRef })

	candidates := make(map[string]v2TaskCandidate)
	detectedBySource := make(map[string]int)
	strictCount, broadCount := 0, 0
	for _, source := range orderedTaskSources {
		content := traceReadGitIndexOverlayFile(t, source.SourceRef, legacySources)
		if got := v2SHA(content); got != source.SourceSHA256 {
			t.Fatalf("task source hash drift %s: %s != %s", source.SourceRef, got, source.SourceSHA256)
		}
		for _, candidate := range v2ScanTaskCandidates(source, content) {
			ref := v2TaskCandidateRef(candidate)
			if _, duplicate := candidates[ref]; duplicate {
				t.Fatalf("duplicate task candidate %s", ref)
			}
			candidates[ref] = candidate
			detectedBySource[source.SourceRef]++
			if candidate.Strength == "strict" {
				strictCount++
			} else {
				broadCount++
			}
		}
	}
	reviewedBlockCandidates := v2ReviewedBlockCandidates(t)
	for _, candidate := range reviewedBlockCandidates {
		ref := v2TaskCandidateRef(candidate)
		if _, duplicate := candidates[ref]; duplicate {
			t.Fatalf("reviewed block candidate collides with scanner candidate %s", ref)
		}
		candidates[ref] = candidate
		detectedBySource[candidate.SourceRef]++
		strictCount++
	}

	entriesByCandidate := make(map[string]v2TaskEntry, len(entries))
	entryCapabilitiesBySource := make(map[string]map[string]struct{})
	pendingBySource := make(map[string]int)
	pendingCount, independentCount := 0, 0
	verdictCounts := make(map[string]int)
	previousEntryKey := ""
	for _, entry := range entries {
		key := fmt.Sprintf("%s\x00%09d\x00%s", entry.SourceRef, entry.SourceLine, entry.DetectionKind)
		if key <= previousEntryKey {
			t.Fatalf("task entries not strictly ordered at %q after %q", key, previousEntryKey)
		}
		previousEntryKey = key
		candidate, exists := candidates[entry.CandidateRef]
		if !exists {
			t.Fatalf("task entry %s has no independently detected candidate %s", entry.EntryRef, entry.CandidateRef)
		}
		if _, duplicate := entriesByCandidate[entry.CandidateRef]; duplicate {
			t.Fatalf("candidate %s has duplicate TaskEntry", entry.CandidateRef)
		}
		entriesByCandidate[entry.CandidateRef] = entry
		v2ValidateTaskEntryIdentity(t, entry, candidate)
		capability, known := capabilities[entry.CapabilityID]
		if !known || entry.CapabilityDecision != capability.Decision {
			t.Fatalf("task entry %s capability decision=%q for %q, roadmap=%#v", entry.EntryRef, entry.CapabilityDecision, entry.CapabilityID, capability)
		}
		wantHistory := "current"
		if taskSources[entry.SourceRef].Decision == "historical" {
			wantHistory = "historical"
		}
		if entry.SourceHistory != wantHistory || entry.Disposition != v2TaskDisposition(wantHistory, capability.Decision) ||
			entry.DecisionRef != "product/roadmap.json#capability_entries/"+entry.CapabilityID || entry.ClosureEvidence != "not_verified" {
			t.Fatalf("task entry %s history/decision/disposition/closure drift: %#v", entry.EntryRef, entry)
		}
		if len(strings.TrimSpace(entry.SemanticReason)) < 10 {
			t.Fatalf("task entry %s lacks semantic reason", entry.EntryRef)
		}
		switch entry.BasisCode {
		case "legacy_heading_objective_review":
			if entry.SemanticReviewState != "pending" || entry.SemanticReason != "pending_manual_semantic_review" || entry.SemanticReviewPassRef != "" || entry.SemanticReviewRef != "" || entry.SemanticReviewVerdict != "" || entry.PreviousCapabilityID != "" {
				t.Fatalf("pending task entry %s has false semantic receipt: %#v", entry.EntryRef, entry)
			}
			pendingCount++
			pendingBySource[entry.SourceRef]++
		case "independent_semantic_review":
			v2ValidateIndependentTaskReview(t, entry)
			independentCount++
			verdictCounts[entry.SemanticReviewVerdict]++
		case "independent_candidate_semantic_review":
			v2ValidateAssignedCandidateReview(t, entry)
		case "explicit_task_semantic_review":
			if !strings.Contains(entry.SemanticReason, capability.Title) || strings.Contains(entry.SemanticReason, " maps to ") {
				t.Fatalf("explicit task reason %s is tautological or omits capability title: %q", entry.EntryRef, entry.SemanticReason)
			}
			v2RequireNoIntegratedReview(t, entry)
		case "task_id_reference_review":
			if entry.SourceEntryID == "" || !strings.Contains(entry.SemanticReason, "canonical definition ID "+entry.SourceEntryID) || !strings.Contains(entry.SemanticReason, capability.Title) {
				t.Fatalf("task reference %s does not cite canonical ID/title: %q", entry.EntryRef, entry.SemanticReason)
			}
			v2RequireNoIntegratedReview(t, entry)
		default:
			if entry.SemanticReviewState != "reviewed" {
				t.Fatalf("task entry %s basis=%q is not reviewed", entry.EntryRef, entry.BasisCode)
			}
			v2RequireNoIntegratedReview(t, entry)
		}
		if entryCapabilitiesBySource[entry.SourceRef] == nil {
			entryCapabilitiesBySource[entry.SourceRef] = make(map[string]struct{})
		}
		entryCapabilitiesBySource[entry.SourceRef][entry.CapabilityID] = struct{}{}
	}

	exclusionsByCandidate := make(map[string]v2TaskExclusion, len(exclusions))
	previousExclusionKey := ""
	for _, exclusion := range exclusions {
		key := fmt.Sprintf("%s\x00%09d\x00%s", exclusion.SourceRef, exclusion.SourceLine, exclusion.DetectionKind)
		if key <= previousExclusionKey {
			t.Fatalf("task exclusions not strictly ordered at %q after %q", key, previousExclusionKey)
		}
		previousExclusionKey = key
		candidate, exists := candidates[exclusion.CandidateRef]
		if !exists || candidate.Strength != "broad" {
			t.Fatalf("task exclusion %s has no broad candidate", exclusion.ExclusionRef)
		}
		if _, duplicate := exclusionsByCandidate[exclusion.CandidateRef]; duplicate {
			t.Fatalf("candidate %s has duplicate exclusion", exclusion.CandidateRef)
		}
		exclusionsByCandidate[exclusion.CandidateRef] = exclusion
		v2ValidateTaskExclusionIdentity(t, exclusion, candidate)
		v2ValidateCandidateExclusionReview(t, exclusion)
	}

	includedBroad := 0
	for candidateRef, candidate := range candidates {
		_, hasEntry := entriesByCandidate[candidateRef]
		_, hasExclusion := exclusionsByCandidate[candidateRef]
		if candidate.Strength == "strict" {
			if !hasEntry || hasExclusion {
				t.Fatalf("strict candidate %s must have exactly one TaskEntry", candidateRef)
			}
			continue
		}
		if hasEntry == hasExclusion {
			t.Fatalf("broad candidate %s must have exactly one entry/exclusion", candidateRef)
		}
		if hasEntry {
			includedBroad++
		}
	}
	if strictCount != policy.Baseline.StrictCandidateCount || broadCount != policy.Baseline.BroadCandidateCount ||
		includedBroad != policy.Baseline.IncludedBroadCandidateCount || len(exclusions) != policy.Baseline.ExcludedBroadCandidateCount || len(entries) != policy.Baseline.EntryCount {
		t.Fatalf("task baseline drift: strict=%d broad=%d included=%d excluded=%d entries=%d", strictCount, broadCount, includedBroad, len(exclusions), len(entries))
	}
	zeroDetected, zeroEntries := 0, 0
	for _, source := range orderedTaskSources {
		if detectedBySource[source.SourceRef] == 0 {
			zeroDetected++
		}
		if len(entryCapabilitiesBySource[source.SourceRef]) == 0 {
			zeroEntries++
		}
		v2ValidateTaskSourceDisposition(t, source, detectedBySource[source.SourceRef], entryCapabilitiesBySource[source.SourceRef], pendingBySource[source.SourceRef], exclusions)
	}
	if zeroDetected != policy.Baseline.SourceWithoutCandidateCount || zeroEntries != policy.Baseline.SourceWithoutEntryCount {
		t.Fatalf("task source zero baseline drift: no candidates=%d no entries=%d", zeroDetected, zeroEntries)
	}
	if pendingCount+independentCount != policy.Baseline.IndependentReviewUniverseCount || pendingCount != policy.Baseline.PendingSemanticReviewCount || independentCount != policy.Baseline.IntegratedIndependentReviewCount ||
		verdictCounts["confirmed"] != policy.Baseline.ConfirmedCount || verdictCounts["corrected"] != policy.Baseline.CorrectedCount ||
		v2TaskSemanticReviewsDigest(t, entries) != policy.Baseline.SemanticReviewsSHA256 {
		t.Fatalf("task independent review coverage: pending=%d integrated=%d universe=%d policy=%+v", pendingCount, independentCount, pendingCount+independentCount, policy.Baseline)
	}
	v2ValidateRejectedRoadmapTask(t, entries)
	v2ValidateSameSourceTaskReferences(t, entries)
	v2ValidateRuntimeModelsSkill(t, runtimeModelsSkill)
	v2ValidateTaskFixture(t, policy.Fixture)
	v2RequireDigest(t, v2TaskEntriesPath, policy.Baseline.EntriesSHA256)
	v2RequireDigest(t, v2TaskExclusionsPath, policy.Baseline.ExclusionsSHA256)
	v2RequireDigest(t, v2TaskProvenancePath, policy.Baseline.SemanticReviewProvenanceSHA256)
	v2RequireDigest(t, "product/traceability/task_candidate_review_provenance.json", policy.Baseline.CandidateReviewProvenanceSHA256)
	v2RequireDigest(t, "product/traceability/reviewed_task_blocks.jsonl", policy.Baseline.ReviewedBlocksSHA256)
	if len(reviewedBlockCandidates) != policy.Baseline.ReviewedBlockEntryCount ||
		len(v2ReadJSONL[v2ReviewedTaskBlock](t, "product/traceability/reviewed_task_blocks.jsonl")) != policy.Baseline.ReviewedBlockCount {
		t.Fatalf("reviewed block baseline drift: entries=%d blocks=%d", len(reviewedBlockCandidates), len(v2ReadJSONL[v2ReviewedTaskBlock](t, "product/traceability/reviewed_task_blocks.jsonl")))
	}
	for _, obsolete := range []string{
		"product/traceability/task_entry_semantic_reviews.jsonl",
		"product/traceability/task_source_capability_map.jsonl",
	} {
		if _, err := os.Stat(obsolete); !os.IsNotExist(err) {
			t.Fatalf("obsolete parallel task authority still exists: %s", obsolete)
		}
	}
	if pendingCount != 0 {
		t.Fatalf("task semantic review incomplete: pending=%d/%d; integrate three independently reviewed shards before V03 accreditation", pendingCount, policy.Baseline.IndependentReviewUniverseCount)
	}
	t.Logf("task_entries_v2: sources=%d strict=%d broad=%d entries=%d exclusions=%d independent_reviews=%d pending=0 closure_inferred=0", len(taskSources), strictCount, broadCount, len(entries), len(exclusions), independentCount)
}

func v2ValidateTaskPolicy(t *testing.T, policy v2TaskPolicy) {
	t.Helper()
	if policy.DocumentKind != "task_entry_extraction_policy" || policy.SchemaVersion != 2 ||
		policy.SourcesAuthority != v2TaskSourcesPath+"#kind=task" || policy.EntriesAuthority != v2TaskEntriesPath ||
		policy.ExclusionsAuthority != v2TaskExclusionsPath || policy.CapabilityAuthority != "product/roadmap.json#capability_entries" ||
		policy.SemanticReviewProvenanceAuthority != v2TaskProvenancePath ||
		policy.CandidateReviewProvenanceAuthority != "product/traceability/task_candidate_review_provenance.json" ||
		policy.ReviewedBlocksAuthority != "product/traceability/reviewed_task_blocks.jsonl" ||
		policy.HashRule.Algorithm != "sha256" || policy.HashRule.LedgerDigestSubject != "exact_jsonl_file_bytes_including_terminal_lf" ||
		policy.HashRule.SemanticReviewRefSubject != "canonical_json_array_[task_semantic_review_v3,entry_ref,source_ref,source_line,old_capability_id,verdict,capability_id,basis_code,semantic_review_pass_ref,reason]" ||
		policy.HashRule.SemanticReviewsDigestSubject != "entries_sorted_by_entry_ref_each_line_json_array_[entry_ref,previous_capability_id,capability_id,verdict,semantic_review_pass_ref,semantic_review_ref,semantic_reason]_including_terminal_lf" ||
		policy.HashRule.CandidateReviewRefSubject != "canonical_json_array_entry_or_exclusion_review_v1_with_pass_ref_and_semantic_reason" ||
		policy.HashRule.CandidateReviewPassRefSubject != "canonical_json_array_task_candidate_review_pass_v1_with_fixture_hashes_and_decision_digests" ||
		policy.HashRule.ReviewedBlockRefSubject != "source_ref_nul_first_line_nul_last_line_nul_block_sha256" ||
		policy.SubjectRule.ReviewedBlock != "exact_reviewed_actionable_line_range" ||
		policy.CandidateRule.GenericH2IsCandidate || !policy.CandidateRule.EveryBroadCandidateRequiresEntryOrExcl || !policy.CandidateRule.StrictCandidatesAreEntries ||
		policy.ClosurePolicy != "legacy_state_is_characterization_only_and_never_rebuild_closure" {
		t.Fatalf("invalid task extraction policy V2: %#v", policy)
	}
	if !reflect.DeepEqual(policy.SourceHistoryValues, []string{"current", "historical"}) ||
		!reflect.DeepEqual(policy.CapabilityDecisionValues, []string{"accept", "reject", "conditional"}) ||
		!v2Contains(policy.BasisCodes, "independent_semantic_review") ||
		!v2Contains(policy.BasisCodes, "independent_candidate_semantic_review") ||
		!v2Contains(policy.CandidateRule.Strict, "reviewed_block") || len(policy.ExclusionReasonCodes) != 6 {
		t.Fatalf("task policy vocabulary drift: history=%v decisions=%v basis=%v exclusions=%v", policy.SourceHistoryValues, policy.CapabilityDecisionValues, policy.BasisCodes, policy.ExclusionReasonCodes)
	}
	if policy.Baseline.CandidateReviewPassCount != 11 ||
		policy.Baseline.CandidateReviewInputCount != policy.Baseline.CandidateReviewEntryCount+policy.Baseline.CandidateReviewExclusionCount ||
		policy.Baseline.ReviewedBlockCount != 170 || policy.Baseline.ReviewedBlockEntryCount != 68 {
		t.Fatalf("task policy candidate/block baseline drift: %#v", policy.Baseline)
	}
}

func v2ValidateTaskEntryIdentity(t *testing.T, entry v2TaskEntry, candidate v2TaskCandidate) {
	t.Helper()
	wantEntryRef := "TASKENTRY-" + strings.TrimPrefix(entry.CandidateRef, "TASKCAND-")
	if entry.SchemaVersion != 2 || entry.EntryRef != wantEntryRef || entry.SourceRef != candidate.SourceRef ||
		entry.SourceSHA256 != candidate.SourceSHA256 || entry.SourceLine != candidate.Line || entry.DetectionKind != candidate.Kind ||
		entry.SourceEntryID != candidate.ID || entry.MarkerSHA256 != candidate.MarkerSHA256 ||
		entry.SubjectFirstLine != candidate.SubjectFirstLine || entry.SubjectLastLine != candidate.SubjectLastLine ||
		entry.SubjectSHA256 != candidate.SubjectSHA256 || entry.OriginalState == "" {
		t.Fatalf("task entry identity drift for %s: entry=%#v candidate=%#v", entry.EntryRef, entry, candidate)
	}
}

func v2ValidateTaskExclusionIdentity(t *testing.T, exclusion v2TaskExclusion, candidate v2TaskCandidate) {
	t.Helper()
	wantRef := "TASKEXCL-" + v2SHAHex([]byte(exclusion.CandidateRef + "\n" + exclusion.ReasonCode))[:24]
	if exclusion.SchemaVersion != 2 || exclusion.ExclusionRef != wantRef || exclusion.SourceRef != candidate.SourceRef ||
		exclusion.SourceSHA256 != candidate.SourceSHA256 || exclusion.SourceLine != candidate.Line || exclusion.DetectionKind != candidate.Kind ||
		exclusion.SourceEntryID != candidate.ID || exclusion.MarkerSHA256 != candidate.MarkerSHA256 ||
		exclusion.SubjectFirstLine != candidate.SubjectFirstLine || exclusion.SubjectLastLine != candidate.SubjectLastLine ||
		exclusion.SubjectSHA256 != candidate.SubjectSHA256 || exclusion.ReviewState != "excluded_after_manual_semantic_review" {
		t.Fatalf("task exclusion identity drift for %s: exclusion=%#v candidate=%#v", exclusion.ExclusionRef, exclusion, candidate)
	}
}

func v2ValidateIndependentTaskReview(t *testing.T, entry v2TaskEntry) {
	t.Helper()
	if entry.SemanticReviewState != "reviewed" || len(strings.TrimSpace(entry.SemanticReason)) < 24 ||
		entry.SemanticReviewPassRef == "" || entry.SemanticReviewRef == "" ||
		(entry.SemanticReviewVerdict != "confirmed" && entry.SemanticReviewVerdict != "corrected") || entry.PreviousCapabilityID == "" {
		t.Fatalf("task entry %s has incomplete independent review receipt: %#v", entry.EntryRef, entry)
	}
	if (entry.SemanticReviewVerdict == "confirmed") != (entry.PreviousCapabilityID == entry.CapabilityID) {
		t.Fatalf("task entry %s verdict=%s old=%s new=%s is inconsistent", entry.EntryRef, entry.SemanticReviewVerdict, entry.PreviousCapabilityID, entry.CapabilityID)
	}
	subject, err := json.Marshal([]any{
		"task_semantic_review_v3", entry.EntryRef, entry.SourceRef, entry.SourceLine, entry.PreviousCapabilityID,
		entry.SemanticReviewVerdict, entry.CapabilityID, entry.BasisCode, entry.SemanticReviewPassRef, strings.TrimSpace(entry.SemanticReason),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRef := "TASKREVIEW-" + v2SHAHex(subject)[:24]
	if entry.SemanticReviewRef != wantRef {
		t.Fatalf("task entry %s semantic review ref=%s, want %s", entry.EntryRef, entry.SemanticReviewRef, wantRef)
	}
}

func v2ValidateAssignedCandidateReview(t *testing.T, entry v2TaskEntry) {
	t.Helper()
	if entry.SemanticReviewState != "reviewed" || len(strings.TrimSpace(entry.SemanticReason)) < 24 ||
		entry.SemanticReviewPassRef == "" || entry.SemanticReviewRef == "" ||
		entry.SemanticReviewVerdict != "assigned" || entry.PreviousCapabilityID != "" {
		t.Fatalf("task entry %s has incomplete candidate review receipt: %#v", entry.EntryRef, entry)
	}
	subject, err := json.Marshal([]any{
		"task_candidate_semantic_review_v1", entry.EntryRef, entry.SourceRef, entry.SourceLine,
		"assigned", entry.CapabilityID, entry.BasisCode, entry.SemanticReviewPassRef, strings.TrimSpace(entry.SemanticReason),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRef := "TASKREVIEW-" + v2SHAHex(subject)[:24]
	if entry.SemanticReviewRef != wantRef {
		t.Fatalf("task entry %s candidate review ref=%s, want %s", entry.EntryRef, entry.SemanticReviewRef, wantRef)
	}
}

func v2ValidateCandidateExclusionReview(t *testing.T, exclusion v2TaskExclusion) {
	t.Helper()
	if exclusion.ReviewPassRef == "" && exclusion.ReviewRef == "" && exclusion.SemanticReason == "" {
		return
	}
	if exclusion.ReviewPassRef == "" || exclusion.ReviewRef == "" || len(strings.TrimSpace(exclusion.SemanticReason)) < 24 {
		t.Fatalf("task exclusion %s has partial candidate review receipt: %#v", exclusion.ExclusionRef, exclusion)
	}
	subject, err := json.Marshal([]any{
		"task_candidate_exclusion_review_v1", exclusion.ExclusionRef, exclusion.CandidateRef,
		exclusion.SourceRef, exclusion.SourceLine, exclusion.ReasonCode, exclusion.ReviewPassRef,
		strings.TrimSpace(exclusion.SemanticReason),
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRef := "TASKREVIEW-" + v2SHAHex(subject)[:24]
	if exclusion.ReviewRef != wantRef {
		t.Fatalf("task exclusion %s candidate review ref=%s, want %s", exclusion.ExclusionRef, exclusion.ReviewRef, wantRef)
	}
}

func v2RequireNoIntegratedReview(t *testing.T, entry v2TaskEntry) {
	t.Helper()
	if entry.SemanticReviewState != "reviewed" || entry.SemanticReviewPassRef != "" || entry.SemanticReviewRef != "" || entry.SemanticReviewVerdict != "" || entry.PreviousCapabilityID != "" {
		t.Fatalf("task entry %s basis=%s has invalid integrated review fields", entry.EntryRef, entry.BasisCode)
	}
}

func v2ValidateTaskSourceDisposition(t *testing.T, source v2TaskSource, detectedCount int, entryCapabilities map[string]struct{}, pendingCount int, exclusions []v2TaskExclusion) {
	t.Helper()
	if source.SchemaVersion != 1 || source.SourceRef == "" || len(source.CapabilityIDs) == 0 || !sort.StringsAreSorted(source.CapabilityIDs) {
		t.Fatalf("invalid task source disposition: %#v", source)
	}
	if detectedCount == 0 {
		if len(entryCapabilities) != 0 || source.ReviewBasis != "source_whole_document_review" ||
			!strings.Contains(source.ReviewNote, source.SourceRef) || !strings.Contains(source.EvidenceState, "source_level_semantic_disposition_complete") {
			t.Fatalf("candidate-free task source lacks exact whole-document disposition: %#v", source)
		}
		return
	}
	if len(entryCapabilities) == 0 {
		if source.SourceRef != v2GroupBSource || detectedCount != 37 || source.Decision != "historical" ||
			source.ReasonCode != "historical_task_candidates_delegated_to_bug_ledger" ||
			source.EvidenceState != "exact_narrative_incident_dispositions_characterization_pending" ||
			source.ReviewBasis != "all_task_candidates_explicitly_excluded" || len(source.CapabilityIDs) <= 1 ||
			source.DelegatedLedgerRef != "product/traceability/historical_bug_occurrences.jsonl" {
			t.Fatalf("Group B incident source delegation is incomplete: %#v", source)
		}
		covered := 0
		for _, exclusion := range exclusions {
			if exclusion.SourceRef != source.SourceRef {
				continue
			}
			exactNarrativeDisposition := exclusion.ReasonCode == "bug_or_incident_owned_by_bug_ledger" &&
				exclusion.ReviewState == "excluded_after_manual_semantic_review" && exclusion.SourceLine == exclusion.SubjectFirstLine &&
				exclusion.SubjectLastLine >= exclusion.SourceLine && exclusion.SubjectSHA256 != ""
			if !exactNarrativeDisposition {
				t.Fatalf("Group B incident heading %s:%d lacks exact narrative disposition", exclusion.SourceRef, exclusion.SourceLine)
			}
			covered++
		}
		if covered != 37 {
			t.Fatalf("Group B incident narrative coverage=%d/37", covered)
		}
		if _, err := os.Stat(source.DelegatedLedgerRef); err != nil {
			t.Fatalf("Group B delegated bug ledger unavailable: %v", err)
		}
		return
	}
	wantCapabilities := make([]string, 0, len(entryCapabilities))
	for capability := range entryCapabilities {
		wantCapabilities = append(wantCapabilities, capability)
	}
	sort.Strings(wantCapabilities)
	if !reflect.DeepEqual(source.CapabilityIDs, wantCapabilities) || source.ReviewBasis != "" || source.ReviewNote != "" || source.DelegatedLedgerRef != "" {
		t.Fatalf("task source %s capability union/review fields drift: got=%v want=%v row=%#v", source.SourceRef, source.CapabilityIDs, wantCapabilities, source)
	}
	if pendingCount > 0 {
		if source.EvidenceState != "semantic_mapping_review_pending" || !strings.HasSuffix(source.ReasonCode, "task_source") {
			t.Fatalf("task source %s hides %d pending semantic mappings: %#v", source.SourceRef, pendingCount, source)
		}
	} else if source.EvidenceState != "semantic_disposition_complete_characterization_pending" || !strings.Contains(source.ReasonCode, "entries_semantically_disposed") {
		t.Fatalf("task source %s completed semantic disposition state drift: %#v", source.SourceRef, source)
	}
}

func v2ValidateRejectedRoadmapTask(t *testing.T, entries []v2TaskEntry) {
	t.Helper()
	found := 0
	for _, entry := range entries {
		if entry.SourceRef != v2RoadmapSource || entry.SourceLine != 80 {
			continue
		}
		found++
		if entry.SourceEntryID != "T-6" || entry.CapabilityID != "ORC-18" || entry.CapabilityDecision != "reject" ||
			entry.Disposition != "rejected_by_operator" || entry.BasisCode != "operator_roadmap_decision" ||
			!reflect.DeepEqual(entry.ReplacementCapabilityIDs, []string{"EVD-15", "OPS-14"}) {
			t.Fatalf("operator-rejected T-6 disposition drift: %#v", entry)
		}
	}
	if found != 1 {
		t.Fatalf("operator-rejected T-6 exact entry count=%d, want 1", found)
	}
}

func v2ValidateSameSourceTaskReferences(t *testing.T, entries []v2TaskEntry) {
	t.Helper()
	definitions := make(map[string]v2TaskEntry)
	var references []v2TaskEntry
	for _, entry := range entries {
		if entry.SourceRef != "modulos/orquesta-cli/docs/tareas.md" || (entry.SourceEntryID != "CLI-004" && entry.SourceEntryID != "CLI-006") {
			continue
		}
		if entry.DetectionKind == "id_field" || entry.DetectionKind == "heading_id" {
			definitions[entry.SourceEntryID] = entry
		}
		if entry.DetectionKind == "list_id" {
			references = append(references, entry)
		}
	}
	if len(definitions) != 2 || len(references) != 2 {
		t.Fatalf("CLI same-source reference fixture drift: definitions=%v references=%v", definitions, references)
	}
	for _, reference := range references {
		definition := definitions[reference.SourceEntryID]
		if reference.CapabilityID != definition.CapabilityID || reference.BasisCode != "task_id_reference_review" {
			t.Fatalf("CLI reference %s resolved outside same-source canonical definition: ref=%#v definition=%#v", reference.SourceEntryID, reference, definition)
		}
	}
}

func v2ValidateRuntimeModelsSkill(t *testing.T, source v2TaskSource) {
	t.Helper()
	want := []string{"AGT-09", "OPS-01", "OPS-05", "OPS-06", "OPS-08", "TLS-06", "TLS-09"}
	if source.Kind != "skill" || source.Decision != "conditional" || !reflect.DeepEqual(source.CapabilityIDs, want) ||
		source.EvidenceState != "pending_scope_trust_and_activation_test" ||
		source.TransformationNote != "No reutilizar literals ORQUESTA_OLLAMA_*; toda clave entra primero en registro tipado y el adaptador consume accesores/credential_ref." {
		t.Fatalf("runtime-modelos skill disposition/transformation drift: %#v", source)
	}
}

func v2ValidateTaskFixture(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := v2TaskSource{SourceRef: path, SourceSHA256: v2SHA(content)}
	candidates := v2ScanTaskCandidates(source, content)
	strengths := map[string]int{}
	byLine := map[int]v2TaskCandidate{}
	for _, candidate := range candidates {
		strengths[candidate.Strength]++
		byLine[candidate.Line] = candidate
	}
	if strengths["strict"] != 8 || strengths["broad"] != 2 || len(candidates) != 10 || byLine[3].Kind != "" || byLine[25].Kind != "" || byLine[17].Kind != "" {
		t.Fatalf("adversarial task fixture grammar drift: strengths=%v lines=%v", strengths, byLine)
	}
	if byLine[8].Kind != "id_field" || byLine[8].SubjectFirstLine != 7 || byLine[8].SubjectLastLine != 11 ||
		byLine[27].Kind != "heading_id" || byLine[29].Kind != "list_id" || byLine[27].ID != byLine[29].ID {
		t.Fatalf("adversarial task fixture subject/same-source identity drift: %#v", byLine)
	}
}

var (
	v2CheckboxRE                     = regexp.MustCompile(`^[ \t]*[-*+][ \t]+\[[ xX]\][ \t]+(.+)$`)
	v2IDFieldRE                      = regexp.MustCompile("^[ \\t]*(?:\\*\\*|__|`)?ID(?:\\*\\*|__|`)?[ \\t]*:[ \\t]*(.+)$")
	v2HeadingRE                      = regexp.MustCompile(`^[ \t]{0,3}(#{2,6})[ \t]+(.+)$`)
	v2ListRE                         = regexp.MustCompile(`^([ \t]*)(?:[-*+]|[0-9]+[.)])[ \t]+(.+)$`)
	v2NumericHeading                 = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*[.):]?[ \t]+`)
	v2IssueIDRE                      = regexp.MustCompile(`^#[0-9]+$`)
	v2ShortTaskIDRE                  = regexp.MustCompile(`^(?:[A-Z]{1,5}[0-9]+|[TWO]-[0-9]+[A-Za-z]?)$`)
	v2StructuredIDRE                 = regexp.MustCompile(`^[A-Z][A-Z0-9]*(?:-[A-Z0-9]+)+$`)
	v2CompositeIDRE                  = regexp.MustCompile(`^(?:T[0-9]+)(?:/T?[0-9]+)+$`)
	v2TableRuleRE                    = regexp.MustCompile(`^:?-{3,}:?$`)
	v2GeneratedSemanticFallbackRE    = regexp.MustCompile(`^.+ trata «.*»; el alcance corresponde a (?:GOV|WIZ|STG|ORC|EVD|UI|AGT|EXT|OPE|OPS|TLS|CTX|APP)-[0-9]{2}\.$`)
	v2GeneratedSemanticCannedReasons = map[string]struct{}{
		"APG-006 compara workers frescos y sesión continua con handoff compacto en fronteras semánticas; corresponde a compactación durable CTX-06.": {},
		"CORE-005 evidencia conserva la fuente forense en cuarentena como backup no operativo; corresponde a backup/restauración EVD-15.":            {},
		"T12 ejecuta un smoke OPES temporal goal-first hasta completed_syllabus_package; corresponde a smokes reales OPS-20.":                        {},
		"OPMCP-012 conserva como residual único el smoke API/MCP contra Hermes temporal; corresponde a smokes reales OPS-20.":                        {},
		"T12 acredita el smoke OPES temporal goal-first hasta completed_syllabus_package; corresponde a smokes reales OPS-20.":                       {},
	}
)

func v2ScanTaskCandidates(source v2TaskSource, content []byte) []v2TaskCandidate {
	lines := v2SplitLines(content)
	result := make([]v2TaskCandidate, 0)
	for index, line := range lines {
		strength, kind, id := v2ClassifyTaskLine(line)
		if strength == "" {
			continue
		}
		first, last := v2TaskSubjectRange(lines, index, kind)
		result = append(result, v2TaskCandidate{
			SourceRef: source.SourceRef, SourceSHA256: source.SourceSHA256, Strength: strength, Kind: kind,
			ID: id, Line: index + 1, MarkerSHA256: v2SHA([]byte(line)), SubjectFirstLine: first,
			SubjectLastLine: last, SubjectSHA256: v2SHA([]byte(strings.Join(lines[first-1:last], "\n"))),
		})
	}
	return result
}

func v2ClassifyTaskLine(line string) (string, string, string) {
	if v2CheckboxRE.MatchString(line) {
		return "strict", "checkbox", ""
	}
	if groups := v2IDFieldRE.FindStringSubmatch(line); groups != nil {
		return "strict", "id_field", v2CleanTaskToken(groups[1])
	}
	if groups := v2HeadingRE.FindStringSubmatch(line); groups != nil {
		body := strings.TrimSpace(groups[2])
		if id := v2LeadingTaskToken(body); v2StrongTaskID(id) {
			return "strict", "heading_id", id
		}
		if v2NumericHeading.MatchString(v2StripTaskMarkup(body)) {
			return "broad", "numbered_heading", v2CleanTaskToken(body)
		}
		return "", "", ""
	}
	if groups := v2ListRE.FindStringSubmatch(line); groups != nil {
		body := strings.TrimSpace(groups[2])
		id := v2LeadingTaskToken(body)
		if v2StrongTaskID(id) {
			return "strict", "list_id", id
		}
		if strings.HasPrefix(body, "**") || strings.HasPrefix(body, "__") || v2BroadTaskID(id) {
			return "broad", "list_candidate", id
		}
		return "", "", ""
	}
	if strings.HasPrefix(strings.TrimSpace(line), "|") {
		for _, rawCell := range strings.Split(strings.TrimSpace(line), "|") {
			cell := strings.TrimSpace(rawCell)
			if cell == "" || v2TableRuleRE.MatchString(cell) {
				continue
			}
			id := v2LeadingTaskToken(cell)
			if v2StrongTaskID(id) {
				return "strict", "table_id", id
			}
			if strings.Contains(cell, "[ ]") || strings.Contains(cell, "[x]") || strings.Contains(cell, "[X]") {
				return "broad", "table_checkbox_candidate", id
			}
			break
		}
	}
	return "", "", ""
}

func v2TaskSubjectRange(lines []string, index int, kind string) (int, int) {
	start, end := index, index
	if kind == "heading_id" || kind == "numbered_heading" {
		level := len(v2HeadingRE.FindStringSubmatch(lines[index])[1])
		for end+1 < len(lines) {
			match := v2HeadingRE.FindStringSubmatch(lines[end+1])
			if match != nil && len(match[1]) <= level {
				break
			}
			end++
		}
		return start + 1, end + 1
	}
	if kind == "id_field" {
		fenceStart := -1
		for lineIndex := index - 1; lineIndex >= 0; lineIndex-- {
			trimmed := strings.TrimSpace(lines[lineIndex])
			if strings.HasPrefix(trimmed, "```") {
				fenceStart = lineIndex
				break
			}
			if v2HeadingRE.MatchString(lines[lineIndex]) || v2IDFieldRE.MatchString(lines[lineIndex]) {
				break
			}
		}
		if fenceStart >= 0 {
			for lineIndex := index + 1; lineIndex < len(lines); lineIndex++ {
				if strings.HasPrefix(strings.TrimSpace(lines[lineIndex]), "```") {
					return fenceStart + 1, lineIndex + 1
				}
			}
		}
		for end+1 < len(lines) && !v2HeadingRE.MatchString(lines[end+1]) && !v2IDFieldRE.MatchString(lines[end+1]) {
			end++
		}
		return start + 1, end + 1
	}
	if kind == "table_id" || kind == "table_checkbox_candidate" {
		return start + 1, end + 1
	}
	indent := v2LeadingWhitespace(lines[index])
	for end+1 < len(lines) {
		next := lines[end+1]
		if v2HeadingRE.MatchString(next) {
			break
		}
		if match := v2ListRE.FindStringSubmatch(next); match != nil && len(match[1]) <= indent {
			break
		}
		end++
	}
	return start + 1, end + 1
}

func v2LeadingTaskToken(value string) string {
	value = v2StripTaskMarkup(value)
	end := len(value)
	for index, char := range value {
		if strings.ContainsRune(" \t:)(,;*`", char) {
			end = index
			break
		}
	}
	return v2CleanTaskToken(value[:end])
}

func v2StripTaskMarkup(value string) string {
	value = strings.TrimSpace(value)
	for {
		before := value
		for _, prefix := range []string{"**", "__", "`", "*", "_"} {
			value = strings.TrimPrefix(value, prefix)
		}
		value = strings.TrimSpace(value)
		if value == before {
			return value
		}
	}
}

func v2CleanTaskToken(value string) string {
	return strings.Trim(strings.TrimSpace(value), "`*_[](){}:;,.—–-")
}

func v2StrongTaskID(id string) bool {
	if v2IssueIDRE.MatchString(id) || v2ShortTaskIDRE.MatchString(id) {
		return true
	}
	if !v2StructuredIDRE.MatchString(id) {
		return false
	}
	for _, char := range id {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return strings.HasPrefix(id, "REQ-")
}

func v2BroadTaskID(id string) bool {
	return v2StrongTaskID(id) || v2StructuredIDRE.MatchString(id) || v2CompositeIDRE.MatchString(id)
}

func v2LeadingWhitespace(value string) int {
	count := 0
	for _, char := range value {
		if char != ' ' && char != '\t' {
			break
		}
		count++
	}
	return count
}

func v2TaskCandidateRef(candidate v2TaskCandidate) string {
	subject := strings.Join([]string{candidate.SourceRef, strconv.Itoa(candidate.Line), candidate.Kind, candidate.MarkerSHA256}, "\n")
	return "TASKCAND-" + v2SHAHex([]byte(subject))[:24]
}

func v2ReviewedBlockCandidates(t *testing.T) []v2TaskCandidate {
	t.Helper()
	blocks := v2ReadJSONL[v2ReviewedTaskBlock](t, "product/traceability/reviewed_task_blocks.jsonl")
	result := make([]v2TaskCandidate, 0)
	for _, block := range blocks {
		if block.CoverageKind != "reviewed_block_entry" {
			continue
		}
		if len(block.CandidateRefs) != 1 {
			t.Fatalf("reviewed task block %s candidate refs=%d, want 1", block.BlockRef, len(block.CandidateRefs))
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
		if want := v2TaskCandidateRef(candidate); block.CandidateRefs[0] != want {
			t.Fatalf("reviewed task block %s candidate ref=%s, want %s", block.BlockRef, block.CandidateRefs[0], want)
		}
		result = append(result, candidate)
	}
	return result
}

func v2TaskDisposition(history, decision string) string {
	switch decision {
	case "reject":
		return "rejected_by_operator"
	case "conditional":
		return "conditional_deferred"
	case "accept":
		if history == "historical" {
			return "historical_superseded_by_capability"
		}
		return "accepted_pending_reimplementation"
	default:
		return "invalid"
	}
}

func v2ReadCapabilities(t *testing.T) map[string]v2Capability {
	t.Helper()
	content, err := os.ReadFile("product/roadmap.json")
	if err != nil {
		t.Fatal(err)
	}
	var roadmap struct {
		CapabilityEntries []struct {
			ID       string `json:"id"`
			Decision string `json:"decision"`
			Title    string `json:"title"`
		} `json:"capability_entries"`
	}
	if err := json.Unmarshal(content, &roadmap); err != nil {
		t.Fatal(err)
	}
	result := make(map[string]v2Capability, len(roadmap.CapabilityEntries))
	for _, capability := range roadmap.CapabilityEntries {
		if _, duplicate := result[capability.ID]; duplicate {
			t.Fatalf("duplicate roadmap capability %q", capability.ID)
		}
		result[capability.ID] = v2Capability{Decision: capability.Decision, Title: capability.Title}
	}
	return result
}

func v2ReadJSONL[T any](t *testing.T, path string) []T {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var result []T
	line := 0
	for scanner.Scan() {
		line++
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		var value T
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("decode %s:%d: %v", path, line, err)
		}
		if decoder.More() {
			t.Fatalf("multiple JSON values at %s:%d", path, line)
		}
		result = append(result, value)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func v2DecodeStrict(t *testing.T, path string, value any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func v2RequireDigest(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := v2SHA(content); got != want {
		t.Fatalf("%s digest=%s, want %s", path, got, want)
	}
}

func v2TaskSemanticReviewsDigest(t *testing.T, entries []v2TaskEntry) string {
	t.Helper()
	reviews := make([]v2TaskEntry, 0, 696)
	for _, entry := range entries {
		if entry.BasisCode == "independent_semantic_review" {
			reviews = append(reviews, entry)
		}
	}
	sort.Slice(reviews, func(i, j int) bool { return reviews[i].EntryRef < reviews[j].EntryRef })
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetEscapeHTML(false)
	for _, entry := range reviews {
		row := []string{
			entry.EntryRef,
			entry.PreviousCapabilityID,
			entry.CapabilityID,
			entry.SemanticReviewVerdict,
			entry.SemanticReviewPassRef,
			entry.SemanticReviewRef,
			entry.SemanticReason,
		}
		if err := encoder.Encode(row); err != nil {
			t.Fatal(err)
		}
	}
	return v2SHA(content.Bytes())
}

func v2SplitLines(content []byte) []string {
	content = bytes.TrimSuffix(content, []byte("\n"))
	if len(content) == 0 {
		return nil
	}
	parts := bytes.Split(content, []byte("\n"))
	result := make([]string, len(parts))
	for index := range parts {
		result[index] = strings.TrimSuffix(string(parts[index]), "\r")
	}
	return result
}

func v2SHA(content []byte) string {
	return "sha256:" + v2SHAHex(content)
}

func v2SHAHex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func v2Contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
