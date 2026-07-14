package orquesta_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const v2CandidateReviewProvenancePath = "product/traceability/task_candidate_review_provenance.json"

const v2SupersededCandidateReviewProvenanceSHA256 = "sha256:89d19a0ce9c68d0d2394ebe731d121779262bab10faf73fa51b3e0b02faf8b3b"

type v2CandidateReviewPass struct {
	ScopeCode             string `json:"scope_code"`
	PassRef               string `json:"pass_ref"`
	ReviewerRef           string `json:"reviewer_ref"`
	ReviewerRole          string `json:"reviewer_role"`
	ReviewMethod          string `json:"review_method"`
	InputFixtureRef       string `json:"input_fixture_ref"`
	InputFixtureSHA256    string `json:"input_fixture_sha256"`
	DecisionFixtureRef    string `json:"decision_fixture_ref"`
	DecisionFixtureSHA256 string `json:"decision_fixture_sha256"`
	InputCount            int    `json:"input_count"`
	EntryCount            int    `json:"entry_count"`
	ExclusionCount        int    `json:"exclusion_count"`
	InputScopeSHA256      string `json:"input_scope_sha256"`
	FinalDecisionsSHA256  string `json:"final_decisions_sha256"`
}

type v2CandidateCounterreviewProvenance struct {
	ScopeCode        string `json:"scope_code"`
	CounterreviewRef string `json:"counterreview_ref"`
	ReviewerRef      string `json:"reviewer_ref"`
	ReviewerRole     string `json:"reviewer_role"`
	ReviewMethod     string `json:"review_method"`
	FixtureRef       string `json:"fixture_ref"`
	FixtureSHA256    string `json:"fixture_sha256"`
	RowCount         int    `json:"row_count"`
	WaveCounts       struct {
		BroadS2 int `json:"broad_s2"`
		BroadS3 int `json:"broad_s3"`
	} `json:"wave_counts"`
	TransitionCounts struct {
		EntryToExclude   int `json:"entry_to_exclude"`
		ExcludeToExclude int `json:"exclude_to_exclude"`
	} `json:"transition_counts"`
	ReasonChangedCount         int    `json:"reason_changed_count"`
	PolicyCodeCorrectionCount  int    `json:"policy_code_correction_count"`
	SupersedesProvenanceSHA256 string `json:"supersedes_provenance_sha256"`
}

type v2CandidateReviewProvenance struct {
	DocumentKind        string                               `json:"document_kind"`
	SchemaVersion       int                                  `json:"schema_version"`
	EntriesAuthority    string                               `json:"entries_authority"`
	ExclusionsAuthority string                               `json:"exclusions_authority"`
	CapabilityAuthority string                               `json:"capability_authority"`
	Passes              []v2CandidateReviewPass              `json:"passes"`
	Counterreviews      []v2CandidateCounterreviewProvenance `json:"counterreviews"`
	Totals              struct {
		PassCount            int    `json:"pass_count"`
		InputCount           int    `json:"input_count"`
		EntryCount           int    `json:"entry_count"`
		ExclusionCount       int    `json:"exclusion_count"`
		InputUniverseSHA256  string `json:"input_universe_sha256"`
		FinalDecisionsSHA256 string `json:"final_decisions_sha256"`
	} `json:"totals"`
}

func TestTraceabilityRebuildTaskCandidateReviewProvenance(t *testing.T) {
	var provenance v2CandidateReviewProvenance
	v2DecodeStrict(t, v2CandidateReviewProvenancePath, &provenance)
	var policy v2TaskPolicy
	v2DecodeStrict(t, v2TaskPolicyPath, &policy)
	if provenance.DocumentKind != "task_candidate_review_provenance" || provenance.SchemaVersion != 1 ||
		provenance.EntriesAuthority != v2TaskEntriesPath+"#basis_code=independent_candidate_semantic_review" ||
		provenance.ExclusionsAuthority != v2TaskExclusionsPath+"#review_pass_ref" ||
		provenance.CapabilityAuthority != "product/roadmap.json#capability_entries" {
		t.Fatalf("invalid candidate review provenance header: %#v", provenance)
	}

	inputPaths, err := filepath.Glob("product/traceability/fixtures/task_candidate_input_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(inputPaths)
	if len(inputPaths) != 11 || len(provenance.Passes) != len(inputPaths) {
		t.Fatalf("candidate review passes=%d inputs=%d, want 11/11", len(provenance.Passes), len(inputPaths))
	}

	entries := v2ReadJSONL[v2TaskEntry](t, v2TaskEntriesPath)
	exclusions := v2ReadJSONL[v2TaskExclusion](t, v2TaskExclusionsPath)
	entriesByPass := make(map[string][]v2TaskEntry)
	exclusionsByPass := make(map[string][]v2TaskExclusion)
	for _, entry := range entries {
		if entry.BasisCode == "independent_candidate_semantic_review" {
			entriesByPass[entry.SemanticReviewPassRef] = append(entriesByPass[entry.SemanticReviewPassRef], entry)
		}
	}
	for _, exclusion := range exclusions {
		if exclusion.ReviewPassRef != "" {
			exclusionsByPass[exclusion.ReviewPassRef] = append(exclusionsByPass[exclusion.ReviewPassRef], exclusion)
		}
	}

	seen := make(map[string]string)
	allCandidates := make([]traceCandidateReviewInput, 0)
	allDecisions := make([]traceCandidateReviewDecision, 0)
	totalEntries, totalExclusions := 0, 0
	for index, inputPath := range inputPaths {
		suffix := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(inputPath), "task_candidate_input_"), ".jsonl")
		decisionPath := filepath.Join(filepath.Dir(inputPath), "task_candidate_review_"+suffix+".jsonl")
		pass := provenance.Passes[index]
		if pass.ScopeCode != suffix || pass.InputFixtureRef != inputPath || pass.DecisionFixtureRef != decisionPath {
			t.Fatalf("candidate pass %d path/scope drift: %#v", index, pass)
		}
		candidates := v2ReadJSONL[traceCandidateReviewInput](t, inputPath)
		decisions := v2ReadJSONL[traceCandidateReviewDecision](t, decisionPath)
		if len(candidates) == 0 || len(decisions) != len(candidates) {
			t.Fatalf("candidate pass %s coverage=%d/%d", suffix, len(decisions), len(candidates))
		}
		for rowIndex, candidate := range candidates {
			if prior, duplicate := seen[candidate.CandidateRef]; duplicate {
				t.Fatalf("candidate %s appears in passes %s and %s", candidate.CandidateRef, prior, suffix)
			}
			seen[candidate.CandidateRef] = suffix
			if decisions[rowIndex].CandidateRef != candidate.CandidateRef {
				t.Fatalf("candidate pass %s row %d order drift", suffix, rowIndex)
			}
		}
		inputSHA := v2CandidateReviewFileSHA(t, inputPath)
		decisionSHA := v2CandidateReviewFileSHA(t, decisionPath)
		inputDigest := v2CandidateReviewInputDigest(t, candidates)
		decisionDigest := v2CandidateReviewDecisionDigest(t, decisions)
		role := "independent_candidate_semantic_reviewer"
		if suffix == "followup" {
			role = "operator_integrator_semantic_reviewer"
		}
		reviewerRef := "TRACE-REVIEWER-" + v2SHAHex([]byte("task-candidate-reviewer-v1\n" + suffix + "\n" + decisionSHA))[:24]
		method := "source_block_and_roadmap_read_per_candidate"
		passSubject := []any{"task_candidate_review_pass_v1", suffix, reviewerRef, role, method, len(candidates), inputDigest, decisionDigest, inputPath, inputSHA, decisionPath, decisionSHA}
		wantPassRef := "TASKREVIEWPASS-" + v2CandidateReviewJSONSHA(t, passSubject)[7:31]
		entryCount, exclusionCount := v2CandidateDecisionCounts(decisions)
		if pass.PassRef != wantPassRef || pass.ReviewerRef != reviewerRef || pass.ReviewerRole != role || pass.ReviewMethod != method ||
			pass.InputFixtureSHA256 != inputSHA || pass.DecisionFixtureSHA256 != decisionSHA || pass.InputCount != len(candidates) ||
			pass.EntryCount != entryCount || pass.ExclusionCount != exclusionCount || pass.InputScopeSHA256 != inputDigest ||
			pass.FinalDecisionsSHA256 != decisionDigest {
			t.Fatalf("candidate review pass %s drift: %#v", suffix, pass)
		}
		v2ValidateCandidatePassReceipts(t, pass, candidates, decisions, entriesByPass[pass.PassRef], exclusionsByPass[pass.PassRef])
		delete(entriesByPass, pass.PassRef)
		delete(exclusionsByPass, pass.PassRef)
		allCandidates = append(allCandidates, candidates...)
		allDecisions = append(allDecisions, decisions...)
		totalEntries += entryCount
		totalExclusions += exclusionCount
	}
	if len(entriesByPass) != 0 || len(exclusionsByPass) != 0 {
		t.Fatalf("candidate receipts reference unknown passes: entries=%v exclusions=%v", sortedTaskReviewKeys(entriesByPass), sortedTaskReviewKeys(exclusionsByPass))
	}
	if provenance.Totals.PassCount != len(inputPaths) || provenance.Totals.InputCount != len(allCandidates) ||
		provenance.Totals.EntryCount != totalEntries || provenance.Totals.ExclusionCount != totalExclusions ||
		provenance.Totals.InputUniverseSHA256 != v2CandidateReviewInputDigest(t, allCandidates) ||
		provenance.Totals.FinalDecisionsSHA256 != v2CandidateReviewDecisionDigest(t, allDecisions) {
		t.Fatalf("candidate review totals drift: %#v", provenance.Totals)
	}
	if provenance.Totals.InputCount != provenance.Totals.EntryCount+provenance.Totals.ExclusionCount {
		t.Fatalf("candidate review is not an exact partition: %#v", provenance.Totals)
	}
	if provenance.Totals.PassCount != policy.Baseline.CandidateReviewPassCount ||
		provenance.Totals.InputCount != policy.Baseline.CandidateReviewInputCount ||
		provenance.Totals.EntryCount != policy.Baseline.CandidateReviewEntryCount ||
		provenance.Totals.ExclusionCount != policy.Baseline.CandidateReviewExclusionCount {
		t.Fatalf("candidate review policy baseline drift: provenance=%#v policy=%#v", provenance.Totals, policy.Baseline)
	}
	v2ValidateBroadS2S3CounterreviewProvenance(t, provenance.Counterreviews, policy)
}

func v2ValidateBroadS2S3CounterreviewProvenance(t *testing.T, counterreviews []v2CandidateCounterreviewProvenance, policy v2TaskPolicy) {
	t.Helper()
	if len(counterreviews) != 1 {
		t.Fatalf("candidate counterreview provenance rows=%d, want 1", len(counterreviews))
	}
	counterreview := counterreviews[0]
	rows := v2ReadJSONL[traceCandidateCounterreview](t, traceBroadS2S3CounterreviewPath)
	fixtureSHA := v2CandidateReviewFileSHA(t, traceBroadS2S3CounterreviewPath)
	if len(rows) != 55 {
		t.Fatalf("candidate counterreview fixture rows=%d, want 55", len(rows))
	}
	waveCounts := make(map[string]int)
	transitionCounts := make(map[string]int)
	reasonChangedCount := 0
	policyCodeCorrectionCount := 0
	allowedExclusions := make(map[string]struct{}, len(policy.ExclusionReasonCodes))
	for _, code := range policy.ExclusionReasonCodes {
		allowedExclusions[code] = struct{}{}
	}
	for _, row := range rows {
		waveCounts[row.Wave]++
		transitionCounts[row.OldDecision+"->"+row.NewDecision]++
		if row.ReasonChanged {
			reasonChangedCount++
		}
		if row.OldDecision == "exclude" {
			if _, allowed := allowedExclusions[row.OldAssignment]; !allowed {
				policyCodeCorrectionCount++
			}
		}
	}
	role := "independent_candidate_counterreviewer"
	method := "source_block_policy_and_final_decision_read_per_candidate"
	reviewerRef := "TRACE-REVIEWER-" + v2SHAHex([]byte("task-candidate-counterreviewer-v1\nbroad_s2_s3\n" + fixtureSHA))[:24]
	counterreviewSubject := []any{
		"task_candidate_counterreview_v1", "broad_s2_s3", reviewerRef, role, method,
		traceBroadS2S3CounterreviewPath, fixtureSHA, len(rows), waveCounts["broad_s2"], waveCounts["broad_s3"],
		transitionCounts["entry->exclude"], transitionCounts["exclude->exclude"], reasonChangedCount, policyCodeCorrectionCount,
	}
	wantCounterreviewRef := "TASKCOUNTERREVIEW-" + v2CandidateReviewJSONSHA(t, counterreviewSubject)[7:31]
	if counterreview.ScopeCode != "broad_s2_s3" || counterreview.CounterreviewRef != wantCounterreviewRef ||
		counterreview.ReviewerRef != reviewerRef || counterreview.ReviewerRole != role || counterreview.ReviewMethod != method ||
		counterreview.FixtureRef != traceBroadS2S3CounterreviewPath || counterreview.FixtureSHA256 != fixtureSHA ||
		counterreview.RowCount != len(rows) || counterreview.WaveCounts.BroadS2 != 46 || counterreview.WaveCounts.BroadS3 != 9 ||
		counterreview.TransitionCounts.EntryToExclude != 36 || counterreview.TransitionCounts.ExcludeToExclude != 19 ||
		counterreview.ReasonChangedCount != 45 || counterreview.PolicyCodeCorrectionCount != 10 ||
		counterreview.SupersedesProvenanceSHA256 != v2SupersededCandidateReviewProvenanceSHA256 {
		t.Fatalf("candidate counterreview provenance drift: %#v", counterreview)
	}
	if counterreview.WaveCounts.BroadS2 != waveCounts["broad_s2"] || counterreview.WaveCounts.BroadS3 != waveCounts["broad_s3"] || len(waveCounts) != 2 ||
		counterreview.TransitionCounts.EntryToExclude != transitionCounts["entry->exclude"] ||
		counterreview.TransitionCounts.ExcludeToExclude != transitionCounts["exclude->exclude"] || len(transitionCounts) != 2 ||
		counterreview.ReasonChangedCount != reasonChangedCount || counterreview.PolicyCodeCorrectionCount != policyCodeCorrectionCount {
		t.Fatalf("candidate counterreview provenance does not describe fixture exactly: provenance=%#v waves=%v transitions=%v", counterreview, waveCounts, transitionCounts)
	}
	currentProvenanceSHA := v2CandidateReviewFileSHA(t, v2CandidateReviewProvenancePath)
	if currentProvenanceSHA != policy.Baseline.CandidateReviewProvenanceSHA256 {
		t.Fatalf("candidate counterreview provenance is not the policy baseline: current=%s baseline=%s",
			currentProvenanceSHA, policy.Baseline.CandidateReviewProvenanceSHA256)
	}
}

func v2ValidateCandidatePassReceipts(t *testing.T, pass v2CandidateReviewPass, candidates []traceCandidateReviewInput, decisions []traceCandidateReviewDecision, entries []v2TaskEntry, exclusions []v2TaskExclusion) {
	t.Helper()
	entryByCandidate := make(map[string]v2TaskEntry, len(entries))
	exclusionByCandidate := make(map[string]v2TaskExclusion, len(exclusions))
	for _, entry := range entries {
		entryByCandidate[entry.CandidateRef] = entry
	}
	for _, exclusion := range exclusions {
		exclusionByCandidate[exclusion.CandidateRef] = exclusion
	}
	for index, candidate := range candidates {
		decision := decisions[index]
		switch decision.Decision {
		case "entry":
			entry, exists := entryByCandidate[candidate.CandidateRef]
			if !exists || entry.CapabilityID != decision.CapabilityID || entry.SemanticReason != strings.TrimSpace(decision.SemanticReason) {
				t.Fatalf("candidate pass %s entry receipt mismatch for %s", pass.ScopeCode, candidate.CandidateRef)
			}
			delete(entryByCandidate, candidate.CandidateRef)
		case "exclude":
			exclusion, exists := exclusionByCandidate[candidate.CandidateRef]
			if !exists || exclusion.ReasonCode != decision.ExclusionReasonCode || exclusion.SemanticReason != strings.TrimSpace(decision.SemanticReason) {
				t.Fatalf("candidate pass %s exclusion receipt mismatch for %s", pass.ScopeCode, candidate.CandidateRef)
			}
			delete(exclusionByCandidate, candidate.CandidateRef)
		default:
			t.Fatalf("candidate pass %s invalid decision %q", pass.ScopeCode, decision.Decision)
		}
	}
	if len(entryByCandidate) != 0 || len(exclusionByCandidate) != 0 {
		t.Fatalf("candidate pass %s receipt surplus entries=%d exclusions=%d", pass.ScopeCode, len(entryByCandidate), len(exclusionByCandidate))
	}
}

func v2CandidateReviewInputDigest(t *testing.T, values []traceCandidateReviewInput) string {
	t.Helper()
	copyValues := append([]traceCandidateReviewInput(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i].CandidateRef < copyValues[j].CandidateRef })
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetEscapeHTML(false)
	for _, item := range copyValues {
		if err := encoder.Encode([]any{item.CandidateRef, item.SourceRef, item.SourceLine, item.Strength, item.DetectionKind, item.SubjectSHA256}); err != nil {
			t.Fatal(err)
		}
	}
	return v2SHA(content.Bytes())
}

func v2CandidateReviewDecisionDigest(t *testing.T, values []traceCandidateReviewDecision) string {
	t.Helper()
	copyValues := append([]traceCandidateReviewDecision(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i].CandidateRef < copyValues[j].CandidateRef })
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetEscapeHTML(false)
	for _, item := range copyValues {
		if err := encoder.Encode([]any{item.CandidateRef, item.Decision, item.CapabilityID, item.ExclusionReasonCode, strings.TrimSpace(item.SemanticReason)}); err != nil {
			t.Fatal(err)
		}
	}
	return v2SHA(content.Bytes())
}

func v2CandidateReviewFileSHA(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return v2SHA(content)
}

func v2CandidateReviewJSONSHA(t *testing.T, value any) string {
	t.Helper()
	content, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return v2SHA(content)
}

func v2CandidateDecisionCounts(values []traceCandidateReviewDecision) (entries, exclusions int) {
	for _, value := range values {
		if value.Decision == "entry" {
			entries++
		} else if value.Decision == "exclude" {
			exclusions++
		}
	}
	return entries, exclusions
}
