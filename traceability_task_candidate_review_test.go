package orquesta_test

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type traceCandidateReviewInput struct {
	CandidateRef          string   `json:"candidate_ref"`
	SourceRef             string   `json:"source_ref"`
	SourceSHA256          string   `json:"source_sha256"`
	SourceDecision        string   `json:"source_decision"`
	SourceCapabilityHints []string `json:"source_capability_hints"`
	Strength              string   `json:"strength"`
	DetectionKind         string   `json:"detection_kind"`
	SourceEntryID         string   `json:"source_entry_id,omitempty"`
	SourceLine            int      `json:"source_line"`
	MarkerSHA256          string   `json:"marker_sha256"`
	SubjectFirstLine      int      `json:"subject_first_line"`
	SubjectLastLine       int      `json:"subject_last_line"`
	SubjectSHA256         string   `json:"subject_sha256"`
	SubjectExcerpt        string   `json:"subject_excerpt"`
}

type traceCandidateReviewDecision struct {
	CandidateRef        string `json:"candidate_ref"`
	SourceRef           string `json:"source_ref"`
	SourceLine          int    `json:"source_line"`
	Decision            string `json:"decision"`
	CapabilityID        string `json:"capability_id,omitempty"`
	SemanticReason      string `json:"semantic_reason"`
	ExclusionReasonCode string `json:"exclusion_reason_code,omitempty"`
}

type traceOriginalCandidateCounterreview struct {
	SchemaVersion        int    `json:"schema_version"`
	CandidateRef         string `json:"candidate_ref"`
	SourceRef            string `json:"source_ref"`
	SourceSHA256         string `json:"source_sha256"`
	SourceLine           int    `json:"source_line"`
	DetectionKind        string `json:"detection_kind"`
	OriginalPassSuffix   string `json:"original_pass_suffix"`
	OriginalDecision     string `json:"original_decision"`
	OriginalCapabilityID string `json:"original_capability_id"`
	FinalDecision        string `json:"final_decision"`
	ExclusionReasonCode  string `json:"exclusion_reason_code"`
	SemanticReason       string `json:"semantic_reason"`
	PrimaryReviewRef     string `json:"primary_review_ref"`
	CounterReviewRef     string `json:"counterreview_ref"`
}

const traceBroadS2S3CounterreviewPath = "product/traceability/fixtures/task_candidate_counterreview_broad_s2_s3.jsonl"

type traceCandidateCounterreview struct {
	Wave              string `json:"wave"`
	CandidateRef      string `json:"candidate_ref"`
	SourceRef         string `json:"source_ref"`
	SourceLine        int    `json:"source_line"`
	OldDecision       string `json:"old_decision"`
	OldAssignment     string `json:"old_assignment"`
	NewDecision       string `json:"new_decision"`
	NewAssignment     string `json:"new_assignment"`
	ReasonChanged     bool   `json:"reason_changed"`
	NewSemanticReason string `json:"new_semantic_reason"`
}

func TestTraceabilityRebuildCandidateDecisionsRespectReviewedBlocksAndPolicy(t *testing.T) {
	inputs, err := filepath.Glob("product/traceability/fixtures/task_candidate_input_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(inputs)
	if len(inputs) < 10 {
		t.Fatalf("candidate review input passes=%d, want at least 10", len(inputs))
	}

	var policy v2TaskPolicy
	v2DecodeStrict(t, v2TaskPolicyPath, &policy)
	allowedExclusions := make(map[string]struct{}, len(policy.ExclusionReasonCodes))
	for _, code := range policy.ExclusionReasonCodes {
		allowedExclusions[code] = struct{}{}
	}
	capabilities := v2ReadCapabilities(t)
	seenCandidates := make(map[string]string)
	totalInputs, totalEntries, totalExclusions := 0, 0, 0
	for _, inputPath := range inputs {
		suffix := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(inputPath), "task_candidate_input_"), ".jsonl")
		decisionPath := filepath.Join(filepath.Dir(inputPath), "task_candidate_review_"+suffix+".jsonl")
		if _, err := filepath.Glob(decisionPath); err != nil {
			t.Fatal(err)
		}
		if !traceFileExists(decisionPath) {
			t.Errorf("candidate input pass %q has no durable decision fixture %q", suffix, decisionPath)
			continue
		}
		candidates := v2ReadJSONL[traceCandidateReviewInput](t, inputPath)
		decisions := v2ReadJSONL[traceCandidateReviewDecision](t, decisionPath)
		if len(candidates) == 0 || len(decisions) != len(candidates) {
			t.Errorf("candidate pass %q coverage=%d/%d", suffix, len(decisions), len(candidates))
			continue
		}
		for index, candidate := range candidates {
			if prior, duplicate := seenCandidates[candidate.CandidateRef]; duplicate {
				t.Errorf("candidate %s appears in passes %s and %s", candidate.CandidateRef, prior, suffix)
			}
			seenCandidates[candidate.CandidateRef] = suffix
			decision := decisions[index]
			if decision.CandidateRef != candidate.CandidateRef || decision.SourceRef != candidate.SourceRef || decision.SourceLine != candidate.SourceLine {
				t.Errorf("candidate pass %q row %d decision identity drift: candidate=%#v decision=%#v", suffix, index, candidate, decision)
				continue
			}
			if len(strings.TrimSpace(decision.SemanticReason)) < 24 || v2GeneratedSemanticFallbackRE.MatchString(decision.SemanticReason) {
				t.Errorf("candidate %s has non-specific semantic reason %q", candidate.CandidateRef, decision.SemanticReason)
			}
			switch decision.Decision {
			case "entry":
				totalEntries++
				if decision.ExclusionReasonCode != "" {
					t.Errorf("entry candidate %s carries exclusion code %q", candidate.CandidateRef, decision.ExclusionReasonCode)
				}
				if _, known := capabilities[decision.CapabilityID]; !known {
					t.Errorf("entry candidate %s references unknown capability %q", candidate.CandidateRef, decision.CapabilityID)
				}
			case "exclude":
				totalExclusions++
				if candidate.Strength != "broad" || decision.CapabilityID != "" {
					t.Errorf("candidate %s cannot be excluded: strength=%q capability=%q", candidate.CandidateRef, candidate.Strength, decision.CapabilityID)
				}
				if _, allowed := allowedExclusions[decision.ExclusionReasonCode]; !allowed {
					t.Errorf("candidate %s uses undeclared exclusion reason %q", candidate.CandidateRef, decision.ExclusionReasonCode)
				}
			default:
				t.Errorf("candidate %s has unknown decision %q", candidate.CandidateRef, decision.Decision)
			}
		}
		totalInputs += len(candidates)
	}
	if totalInputs == 0 || totalEntries+totalExclusions != totalInputs {
		t.Fatalf("candidate review partition inputs=%d entries=%d exclusions=%d", totalInputs, totalEntries, totalExclusions)
	}
	t.Logf("candidate review fixtures: passes=%d inputs=%d entries=%d exclusions=%d", len(inputs), totalInputs, totalEntries, totalExclusions)
}

func TestTraceabilityRebuildBroadS2S3CounterreviewBindsExactFinalDecisions(t *testing.T) {
	rows := v2ReadJSONL[traceCandidateCounterreview](t, traceBroadS2S3CounterreviewPath)
	if len(rows) != 55 {
		t.Fatalf("broad s2/s3 counterreview rows=%d, want 55", len(rows))
	}

	var policy v2TaskPolicy
	v2DecodeStrict(t, v2TaskPolicyPath, &policy)
	allowedExclusions := make(map[string]struct{}, len(policy.ExclusionReasonCodes))
	for _, code := range policy.ExclusionReasonCodes {
		allowedExclusions[code] = struct{}{}
	}
	capabilities := v2ReadCapabilities(t)

	type candidateFinalDecision struct {
		wave      string
		candidate traceCandidateReviewInput
		decision  traceCandidateReviewDecision
	}
	finalByCandidate := make(map[string]candidateFinalDecision)
	for _, wave := range []string{"broad_s2", "broad_s3"} {
		inputPath := filepath.Join("product/traceability/fixtures", "task_candidate_input_"+wave+".jsonl")
		decisionPath := filepath.Join("product/traceability/fixtures", "task_candidate_review_"+wave+".jsonl")
		candidates := v2ReadJSONL[traceCandidateReviewInput](t, inputPath)
		decisions := v2ReadJSONL[traceCandidateReviewDecision](t, decisionPath)
		if len(candidates) == 0 || len(decisions) != len(candidates) {
			t.Fatalf("counterreview wave %s final coverage=%d/%d", wave, len(decisions), len(candidates))
		}
		for index, candidate := range candidates {
			decision := decisions[index]
			if decision.CandidateRef != candidate.CandidateRef || decision.SourceRef != candidate.SourceRef || decision.SourceLine != candidate.SourceLine {
				t.Fatalf("counterreview wave %s row %d input/final identity drift", wave, index)
			}
			if _, duplicate := finalByCandidate[candidate.CandidateRef]; duplicate {
				t.Fatalf("counterreview final universe duplicates candidate %s", candidate.CandidateRef)
			}
			finalByCandidate[candidate.CandidateRef] = candidateFinalDecision{wave: wave, candidate: candidate, decision: decision}
		}
	}

	seen := make(map[string]struct{}, len(rows))
	waveCounts := make(map[string]int)
	transitionCounts := make(map[string]int)
	reasonChangedCount := 0
	invalidOldPolicyCodeCount := 0
	for index, row := range rows {
		if _, duplicate := seen[row.CandidateRef]; duplicate {
			t.Fatalf("counterreview row %d duplicates candidate %s", index, row.CandidateRef)
		}
		seen[row.CandidateRef] = struct{}{}
		final, found := finalByCandidate[row.CandidateRef]
		if !found {
			t.Fatalf("counterreview row %d references unknown candidate %s", index, row.CandidateRef)
		}
		if row.Wave != final.wave || row.SourceRef != final.candidate.SourceRef || row.SourceLine != final.candidate.SourceLine ||
			final.candidate.Strength != "broad" {
			t.Fatalf("counterreview row %d does not bind exact input: row=%#v input=%#v wave=%s", index, row, final.candidate, final.wave)
		}
		if row.NewDecision != final.decision.Decision || row.NewSemanticReason != final.decision.SemanticReason ||
			row.NewSemanticReason != strings.TrimSpace(row.NewSemanticReason) || len(row.NewSemanticReason) < 24 ||
			v2GeneratedSemanticFallbackRE.MatchString(row.NewSemanticReason) {
			t.Fatalf("counterreview row %d does not bind exact final decision/reason for %s", index, row.CandidateRef)
		}

		switch row.OldDecision {
		case "entry":
			if _, known := capabilities[row.OldAssignment]; !known {
				t.Fatalf("counterreview row %d old entry has unknown capability %q", index, row.OldAssignment)
			}
		case "exclude":
			if _, allowed := allowedExclusions[row.OldAssignment]; !allowed {
				if row.OldAssignment != "task_extraction_policy" {
					t.Fatalf("counterreview row %d old exclusion has unclassified policy code %q", index, row.OldAssignment)
				}
				invalidOldPolicyCodeCount++
			}
		default:
			t.Fatalf("counterreview row %d has invalid old decision %q", index, row.OldDecision)
		}

		switch row.NewDecision {
		case "entry":
			if row.NewAssignment != final.decision.CapabilityID || final.decision.ExclusionReasonCode != "" {
				t.Fatalf("counterreview row %d final entry assignment drift for %s", index, row.CandidateRef)
			}
			if _, known := capabilities[row.NewAssignment]; !known {
				t.Fatalf("counterreview row %d final entry has unknown capability %q", index, row.NewAssignment)
			}
		case "exclude":
			if row.NewAssignment != final.decision.ExclusionReasonCode || final.decision.CapabilityID != "" {
				t.Fatalf("counterreview row %d final exclusion assignment drift for %s", index, row.CandidateRef)
			}
			if _, allowed := allowedExclusions[row.NewAssignment]; !allowed {
				t.Fatalf("counterreview row %d final exclusion uses undeclared policy code %q", index, row.NewAssignment)
			}
		default:
			t.Fatalf("counterreview row %d has invalid new decision %q", index, row.NewDecision)
		}

		waveCounts[row.Wave]++
		transitionCounts[row.OldDecision+"->"+row.NewDecision]++
		if row.ReasonChanged {
			reasonChangedCount++
		}
	}
	if len(seen) != len(rows) || waveCounts["broad_s2"] != 46 || waveCounts["broad_s3"] != 9 || len(waveCounts) != 2 {
		t.Fatalf("counterreview wave partition=%v unique=%d, want broad_s2=46 broad_s3=9 unique=55", waveCounts, len(seen))
	}
	if transitionCounts["entry->exclude"] != 36 || transitionCounts["exclude->exclude"] != 19 || len(transitionCounts) != 2 {
		t.Fatalf("counterreview transitions=%v, want entry->exclude=36 exclude->exclude=19", transitionCounts)
	}
	if reasonChangedCount != 45 || invalidOldPolicyCodeCount != 10 {
		t.Fatalf("counterreview reason_changed=%d invalid_old_policy_codes=%d, want 45/10", reasonChangedCount, invalidOldPolicyCodeCount)
	}
}

func TestTraceabilityRebuildOriginalBroadCorrectionsAreDurableAndCounterreviewed(t *testing.T) {
	const correctionPath = "product/traceability/fixtures/task_candidate_counterreview_original_outside_blocks.jsonl"
	const primaryReview = "v03_original_outside_blocks_review@sha256:8295e367a1bc48c2be77d02dad547e26c814f8e790b1d27bbd0ba5c2a3753082"
	const integralReview = "v03_original_integral_counterreview@sha256:465f878f16eeeb145af10181159f47a4d237a448c451eb728e229e3603341f7f"
	const restReview = "v03_original_rest_counterreview@sha256:aba2502e33192a4cdad2ee131d264f57fdf3d128c0950c1326f025de4196417a"
	rows := v2ReadJSONL[traceOriginalCandidateCounterreview](t, correctionPath)
	if len(rows) != 177 {
		t.Fatalf("original broad corrections=%d, want 177", len(rows))
	}
	capabilities := v2ReadCapabilities(t)
	seen := make(map[string]struct{}, len(rows))
	previousKey := ""
	counterCounts := make(map[string]int)
	for _, row := range rows {
		key := row.OriginalPassSuffix + "\x00" + row.SourceRef + "\x00" + fmt.Sprintf("%09d", row.SourceLine) + "\x00" + row.CandidateRef
		if key <= previousKey {
			t.Fatalf("original broad corrections not ordered at %q after %q", key, previousKey)
		}
		previousKey = key
		if _, duplicate := seen[row.CandidateRef]; duplicate {
			t.Fatalf("duplicate original broad correction %s", row.CandidateRef)
		}
		seen[row.CandidateRef] = struct{}{}
		if row.SchemaVersion != 1 || row.OriginalDecision != "entry" || row.FinalDecision != "exclude" || row.PrimaryReviewRef != primaryReview ||
			len(strings.TrimSpace(row.SemanticReason)) < 24 || row.ExclusionReasonCode == "" {
			t.Fatalf("invalid original broad correction: %#v", row)
		}
		if _, known := capabilities[row.OriginalCapabilityID]; !known {
			t.Fatalf("original broad correction %s has unknown prior capability %q", row.CandidateRef, row.OriginalCapabilityID)
		}
		wantCounter := restReview
		if row.SourceRef == "docs/informe_auditoria_integral_orquesta_2026-07-13.md" {
			wantCounter = integralReview
		}
		if row.CounterReviewRef != wantCounter {
			t.Fatalf("original broad correction %s counterreview=%q, want %q", row.CandidateRef, row.CounterReviewRef, wantCounter)
		}
		counterCounts[row.CounterReviewRef]++
		inputPath := filepath.Join("product/traceability/fixtures", "task_candidate_input_"+row.OriginalPassSuffix+".jsonl")
		decisionPath := filepath.Join("product/traceability/fixtures", "task_candidate_review_"+row.OriginalPassSuffix+".jsonl")
		candidate, foundCandidate := traceFindCandidateReviewInput(t, inputPath, row.CandidateRef)
		decision, foundDecision := traceFindCandidateReviewDecision(t, decisionPath, row.CandidateRef)
		if !foundCandidate || !foundDecision || candidate.Strength != "broad" || candidate.SourceRef != row.SourceRef || candidate.SourceSHA256 != row.SourceSHA256 ||
			candidate.SourceLine != row.SourceLine || candidate.DetectionKind != row.DetectionKind || decision.Decision != "exclude" || decision.CapabilityID != "" ||
			decision.ExclusionReasonCode != row.ExclusionReasonCode || decision.SemanticReason != row.SemanticReason {
			t.Fatalf("original broad correction %s does not bind exact input/final decision", row.CandidateRef)
		}
	}
	if counterCounts[integralReview] != 75 || counterCounts[restReview] != 102 {
		t.Fatalf("original broad counterreview partition=%v, want 75+102", counterCounts)
	}
}

func traceFindCandidateReviewInput(t *testing.T, path, candidateRef string) (traceCandidateReviewInput, bool) {
	t.Helper()
	for _, candidate := range v2ReadJSONL[traceCandidateReviewInput](t, path) {
		if candidate.CandidateRef == candidateRef {
			return candidate, true
		}
	}
	return traceCandidateReviewInput{}, false
}

func traceFindCandidateReviewDecision(t *testing.T, path, candidateRef string) (traceCandidateReviewDecision, bool) {
	t.Helper()
	for _, decision := range v2ReadJSONL[traceCandidateReviewDecision](t, path) {
		if decision.CandidateRef == candidateRef {
			return decision, true
		}
	}
	return traceCandidateReviewDecision{}, false
}

func traceFileExists(path string) bool {
	matches, err := filepath.Glob(path)
	return err == nil && len(matches) == 1 && matches[0] == path
}
