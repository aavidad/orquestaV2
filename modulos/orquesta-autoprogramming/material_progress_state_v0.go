package orquestaautoprogramming

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

const MaterialProgressStateSchemaVersionV0 = "material_progress_state.v0"

type MaterialProgressStateV0 struct {
	SchemaVersion            string                       `json:"schema_version"`
	StoreVersion             uint64                       `json:"store_version"`
	RunRef                   string                       `json:"run_ref"`
	GoalRef                  string                       `json:"goal_ref"`
	Policy                   MaterialProgressPolicyV0     `json:"policy"`
	Segment                  MaterialProgressSegmentV0    `json:"segment"`
	LastCheckpoint           MaterialProgressCheckpointV0 `json:"last_checkpoint"`
	LastDecision             MaterialProgressDecisionV0   `json:"last_decision"`
	BaselineRef              string                       `json:"baseline_ref"`
	WriteSetSHA256           string                       `json:"write_set_sha256"`
	ContextRevisionRef       string                       `json:"context_revision_ref"`
	ObservedAt               time.Time                    `json:"observed_at"`
	LastActionIdempotencyKey string                       `json:"last_action_idempotency_key"`
	LastCheckpointRef        string                       `json:"last_checkpoint_ref"`
	EvidenceRefs             []string                     `json:"evidence_refs,omitempty"`
}

type MaterialProgressStateValidationResultV0 struct {
	Accepted bool                      `json:"accepted"`
	State    MaterialProgressStateV0   `json:"state"`
	Issues   []MaterialProgressIssueV0 `json:"issues,omitempty"`
}

type MaterialProgressStateReaderPortV0 interface {
	LoadMaterialProgressStateV0(context.Context, string, string) (MaterialProgressStateV0, error)
}

type MaterialProgressStateStorePortV0 interface {
	MaterialProgressStateReaderPortV0
	CompareAndSwapMaterialProgressStateV0(context.Context, uint64, MaterialProgressStateV0) (MaterialProgressStateV0, error)
}

type MaterialProgressStateCASConflictErrorV0 struct {
	RunRef          string
	GoalRef         string
	ExpectedVersion uint64
	ActualVersion   uint64
}

func (err MaterialProgressStateCASConflictErrorV0) Error() string {
	return "material_progress_state_cas_conflict"
}

func NormalizeMaterialProgressStateV0(state MaterialProgressStateV0) MaterialProgressStateV0 {
	state.SchemaVersion = strings.TrimSpace(state.SchemaVersion)
	state.RunRef = materialProgressStateNormalizeRefV0(state.RunRef)
	state.GoalRef = materialProgressStateNormalizeRefV0(state.GoalRef)
	state.BaselineRef = materialProgressStateNormalizeRefV0(state.BaselineRef)
	state.ContextRevisionRef = materialProgressStateNormalizeRefV0(state.ContextRevisionRef)
	state.LastActionIdempotencyKey = strings.TrimSpace(state.LastActionIdempotencyKey)
	state.LastCheckpointRef = materialProgressStateNormalizeRefV0(state.LastCheckpointRef)
	state.WriteSetSHA256 = strings.ToLower(strings.TrimSpace(state.WriteSetSHA256))
	state.Segment.ContextRevisionRef = materialProgressStateNormalizeRefV0(state.Segment.ContextRevisionRef)
	state.Segment.EvidenceRefs = materialProgressStateRefsV0(state.Segment.EvidenceRefs)
	state.LastCheckpoint.ContextRevisionRef = materialProgressStateNormalizeRefV0(state.LastCheckpoint.ContextRevisionRef)
	state.LastCheckpoint.EvidenceRefs = materialProgressStateRefsV0(state.LastCheckpoint.EvidenceRefs)
	state.LastDecision.Segment.ContextRevisionRef = materialProgressStateNormalizeRefV0(state.LastDecision.Segment.ContextRevisionRef)
	state.LastDecision.Segment.EvidenceRefs = materialProgressStateRefsV0(state.LastDecision.Segment.EvidenceRefs)
	state.EvidenceRefs = materialProgressStateRefsV0(append(
		append(append([]string{}, state.EvidenceRefs...), state.Segment.EvidenceRefs...),
		state.LastCheckpoint.EvidenceRefs...,
	))
	if !state.ObservedAt.IsZero() {
		state.ObservedAt = state.ObservedAt.UTC()
	}
	return state
}

func ValidateMaterialProgressStateV0(state MaterialProgressStateV0) MaterialProgressStateValidationResultV0 {
	state = NormalizeMaterialProgressStateV0(state)
	issues := materialProgressStateRequiredIssuesV0(state)
	progress := ValidateMaterialProgressV0(MaterialProgressInputV0{
		Policy:     state.Policy,
		Segment:    state.Segment,
		Checkpoint: state.LastCheckpoint,
	})
	issues = append(issues, progress.Issues...)
	issues = append(issues, materialProgressStateDecisionIssuesV0(state, progress)...)
	return MaterialProgressStateValidationResultV0{
		Accepted: len(issues) == 0,
		State:    state,
		Issues:   issues,
	}
}

func MaterialProgressActionIdempotencyKeyV0(
	runRef string,
	goalRef string,
	checkpointRef string,
	action MaterialProgressActionV0,
) string {
	runRef = materialProgressStateNormalizeRefV0(runRef)
	goalRef = materialProgressStateNormalizeRefV0(goalRef)
	checkpointRef = materialProgressStateNormalizeRefV0(checkpointRef)
	if !materialProgressStateRefValidV0(runRef) || !materialProgressStateRefValidV0(goalRef) ||
		!materialProgressStateRefValidV0(checkpointRef) || !materialProgressStateActionValidV0(action) {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		"material-progress-action.v0", runRef, goalRef, checkpointRef, string(action),
	}, "\n")))
	return "material-progress-action-v0:" + hex.EncodeToString(sum[:])
}

func MaterialProgressCheckpointRefV0(
	runRef string,
	goalRef string,
	baselineRef string,
	writeSetSHA256 string,
	checkpoint MaterialProgressCheckpointV0,
) string {
	runRef = materialProgressStateNormalizeRefV0(runRef)
	goalRef = materialProgressStateNormalizeRefV0(goalRef)
	baselineRef = materialProgressStateNormalizeRefV0(baselineRef)
	writeSetSHA256 = strings.ToLower(strings.TrimSpace(writeSetSHA256))
	checkpoint.ContextRevisionRef = materialProgressStateNormalizeRefV0(checkpoint.ContextRevisionRef)
	checkpoint.EvidenceRefs = materialProgressStateRefsV0(checkpoint.EvidenceRefs)
	if !materialProgressStateRefValidV0(runRef) || !materialProgressStateRefValidV0(goalRef) ||
		!materialProgressStateRefValidV0(baselineRef) || !materialProgressStateSHA256ValidV0(writeSetSHA256) ||
		!materialProgressStateRefValidV0(checkpoint.ContextRevisionRef) ||
		!materialProgressClassValidV0(checkpoint.MaterialClass) {
		return ""
	}
	for _, ref := range checkpoint.EvidenceRefs {
		if !materialProgressStateRefValidV0(ref) {
			return ""
		}
	}
	evidenceRefs := append([]string(nil), checkpoint.EvidenceRefs...)
	sort.Strings(evidenceRefs)
	sum := sha256.Sum256([]byte(strings.Join([]string{
		"material-progress-checkpoint.v0",
		runRef,
		goalRef,
		baselineRef,
		writeSetSHA256,
		strconv.FormatInt(checkpoint.Sequence, 10),
		strconv.FormatInt(checkpoint.TokensAccumulated, 10),
		checkpoint.ContextRevisionRef,
		string(checkpoint.MaterialClass),
		strings.Join(evidenceRefs, "\x00"),
	}, "\n")))
	return "material-progress-checkpoint-v0:" + hex.EncodeToString(sum[:])
}

func materialProgressStateRequiredIssuesV0(state MaterialProgressStateV0) []MaterialProgressIssueV0 {
	var issues []MaterialProgressIssueV0
	if state.SchemaVersion != MaterialProgressStateSchemaVersionV0 {
		issues = append(issues, materialProgressIssueV0("state_schema_version_invalid", "schema_version", "schema_version de estado requerida"))
	}
	if state.StoreVersion == 0 {
		issues = append(issues, materialProgressIssueV0("state_store_version_invalid", "store_version", "store_version positivo requerido"))
	}
	for _, ref := range []struct{ field, value string }{
		{"run_ref", state.RunRef}, {"goal_ref", state.GoalRef}, {"baseline_ref", state.BaselineRef},
		{"context_revision_ref", state.ContextRevisionRef}, {"last_checkpoint_ref", state.LastCheckpointRef},
	} {
		if !materialProgressStateRefValidV0(ref.value) {
			issues = append(issues, materialProgressIssueV0("state_"+ref.field+"_invalid", ref.field, "ref segura requerida"))
		}
	}
	for _, ref := range state.EvidenceRefs {
		if !materialProgressStateRefValidV0(ref) {
			issues = append(issues, materialProgressIssueV0("state_evidence_ref_invalid", "evidence_refs", "evidence ref compacta requerida"))
		}
	}
	if !materialProgressStateSHA256ValidV0(state.WriteSetSHA256) {
		issues = append(issues, materialProgressIssueV0("state_write_set_sha256_invalid", "write_set_sha256", "sha256 requerida"))
	}
	if state.ObservedAt.IsZero() {
		issues = append(issues, materialProgressIssueV0("state_observed_at_missing", "observed_at", "observacion requerida"))
	}
	if state.ContextRevisionRef != state.Segment.ContextRevisionRef ||
		state.ContextRevisionRef != state.LastCheckpoint.ContextRevisionRef {
		issues = append(issues, materialProgressIssueV0("state_context_revision_ref_mismatch", "context_revision_ref", "contexto del estado, tramo y checkpoint debe coincidir"))
	}
	if state.Policy.MaxReplans < 0 || state.Segment.ReplansUsed > state.Policy.MaxReplans {
		issues = append(issues, materialProgressIssueV0("state_replan_budget_invalid", "segment.replans_used", "presupuesto de replan invalido"))
	}
	wantCheckpointRef := MaterialProgressCheckpointRefV0(
		state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, state.LastCheckpoint,
	)
	if wantCheckpointRef == "" || state.LastCheckpointRef != wantCheckpointRef {
		issues = append(issues, materialProgressIssueV0("state_checkpoint_ref_invalid", "last_checkpoint_ref", "checkpoint ref determinista requerida"))
	}
	return issues
}

func materialProgressStateDecisionIssuesV0(
	state MaterialProgressStateV0,
	progress MaterialProgressValidationResultV0,
) []MaterialProgressIssueV0 {
	decision := state.LastDecision
	var issues []MaterialProgressIssueV0
	if !decision.Accepted || !materialProgressStateActionValidV0(decision.Action) || len(decision.Issues) != 0 {
		issues = append(issues, materialProgressIssueV0("state_last_decision_invalid", "last_decision", "decision aceptada y tipada requerida"))
	}
	if !materialProgressStateSegmentsEqualV0(decision.Segment, state.Segment) {
		issues = append(issues, materialProgressIssueV0("state_last_decision_segment_mismatch", "last_decision.segment", "decision debe conservar tramo persistido"))
	}
	if progress.Accepted {
		expected := DecideMaterialProgressV0(progress.Input)
		if decision.Action != expected.Action || decision.TokensWithoutMaterial != expected.TokensWithoutMaterial {
			issues = append(issues, materialProgressIssueV0("state_last_decision_checkpoint_mismatch", "last_decision", "decision no coincide con checkpoint"))
		}
		if decision.MaterialProgressed {
			if !materialProgressClassRenewsV0(state.LastCheckpoint.MaterialClass) ||
				state.Segment.StartSequence != state.LastCheckpoint.Sequence ||
				state.Segment.StartTokensAccumulated != state.LastCheckpoint.TokensAccumulated ||
				!materialProgressStateContainsAllRefsV0(state.Segment.EvidenceRefs, state.LastCheckpoint.EvidenceRefs) {
				issues = append(issues, materialProgressIssueV0("state_material_renewal_invalid", "last_decision", "renovacion debe quedar anclada al checkpoint material"))
			}
		} else if expected.MaterialProgressed {
			issues = append(issues, materialProgressIssueV0("state_material_progress_mismatch", "last_decision.material_progressed", "progreso material no puede omitirse"))
		}
	}
	wantKey := MaterialProgressActionIdempotencyKeyV0(state.RunRef, state.GoalRef, state.LastCheckpointRef, decision.Action)
	if wantKey == "" || state.LastActionIdempotencyKey != wantKey {
		issues = append(issues, materialProgressIssueV0("state_action_idempotency_key_invalid", "last_action_idempotency_key", "clave idempotente determinista requerida"))
	}
	return issues
}

func materialProgressStateNormalizeRefV0(value string) string {
	return strings.TrimSpace(value)
}

func materialProgressStateRefsV0(values []string) []string {
	return compactStringsV0(values)
}

func materialProgressStateRefValidV0(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 600 && !strings.ContainsAny(value, " /\\\t\r\n\x00")
}

func materialProgressStateSHA256ValidV0(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func materialProgressStateActionValidV0(action MaterialProgressActionV0) bool {
	switch action {
	case MaterialProgressActionContinueV0, MaterialProgressActionWarningV0,
		MaterialProgressActionReplanRequiredV0, MaterialProgressActionHardStopRequiredV0:
		return true
	default:
		return false
	}
}

func materialProgressStateSegmentsEqualV0(left, right MaterialProgressSegmentV0) bool {
	if left.StartSequence != right.StartSequence ||
		left.StartTokensAccumulated != right.StartTokensAccumulated ||
		left.ReplansUsed != right.ReplansUsed ||
		left.ContextRevisionRef != right.ContextRevisionRef ||
		len(left.EvidenceRefs) != len(right.EvidenceRefs) {
		return false
	}
	for index := range left.EvidenceRefs {
		if left.EvidenceRefs[index] != right.EvidenceRefs[index] {
			return false
		}
	}
	return true
}

func materialProgressStateContainsAllRefsV0(values, wants []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, want := range wants {
		if _, ok := seen[want]; !ok {
			return false
		}
	}
	return true
}
