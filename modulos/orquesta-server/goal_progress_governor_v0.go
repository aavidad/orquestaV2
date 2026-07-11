package orquestaserver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	idleSelfImprovementGoalHighConsumptionNoProgressReasonV0 = "goal_high_consumption_without_progress"
	idleSelfImprovementGoalReviewReplanRecommendedActionV0   = "review_replan_goal_first"
	idleSelfImprovementGoalReviewReplanRefV0                 = "rework-plan-ref-review-replan-goal-first"
)

type idleSelfImprovementGoalProgressUpdateV0 struct {
	UsefulProgressAt           time.Time
	UsefulProgressSignature    string
	ObservedConsumption        int64
	InvalidCheckpointSignature string
	InvalidCheckpointRepeats   int
	ClearInvalidCheckpoint     bool
}

type idleSelfImprovementGoalProgressEvaluationV0 struct {
	ProgressObserved           bool
	ProgressSignature          string
	LastProgressAt             time.Time
	ObservedConsumption        int64
	ConsumptionGrowing         bool
	WindowExceeded             bool
	InvalidCheckpointSignature string
	InvalidCheckpointRepeats   int
	InvalidCheckpointRepeated  bool
	ShouldBlock                bool
	EvidenceRefs               []string
}

func (runtime *RuntimeV0) reconcileIdleSelfImprovementGoalProgressV0(
	ctx context.Context,
	result orquestagoal.GoalWorkObserveActiveResultV0,
	now time.Time,
) orquestagoal.GoalWorkObserveActiveResultV0 {
	if runtime == nil || runtime.tracker == nil || runtime.goalStateStore == nil || len(result.Observations) == 0 {
		return result
	}
	state := runtime.tracker.SnapshotV0()
	index, observed, ok := idleSelfImprovementGoalObservationIndexV0(state, result)
	if !ok || observed.Status != orquestagoal.GoalStatusRunningV0 {
		return result
	}
	evaluation := idleSelfImprovementGoalProgressEvaluationFromObservationV0(
		runtime.config,
		state,
		result.Observations[index],
		observed,
		now,
	)
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkIdleSelfImprovementGoalProgressV0(
			idleSelfImprovementGoalProgressUpdateFromEvaluationV0(evaluation),
			now,
		),
		"idle_self_improvement_goal_progress",
	)
	if !evaluation.ShouldBlock {
		return result
	}
	blocked := runtime.blockIdleSelfImprovementGoalWithoutProgressV0(
		ctx,
		result.Observations[index],
		observed,
		evaluation,
	)
	result.Observations[index].Result = blocked
	result.Observations[index].State.Status = blocked.Status
	result.Observations[index].State.LastResult = &blocked
	result.Observations[index].State.EvidenceRefs = compactConfigStringsV0(
		append(result.Observations[index].State.EvidenceRefs, blocked.EvidenceRefs...),
	)
	result.Observations[index].Terminal = true
	result.Observations[index].Accepted = false
	result.Observations[index].NeedsRework = true
	result.Observations[index].EvidenceRefs = compactConfigStringsV0(
		append(result.Observations[index].EvidenceRefs, blocked.EvidenceRefs...),
	)
	result.EvidenceRefs = compactConfigStringsV0(append(result.EvidenceRefs, blocked.EvidenceRefs...))
	result.Issues = append(result.Issues, orquestagoal.GoalWorkObserveActiveIssueV0{
		RunRef:  strings.TrimSpace(result.Observations[index].State.RunRef),
		GoalRef: strings.TrimSpace(blocked.GoalRef),
		Code:    idleSelfImprovementGoalHighConsumptionNoProgressReasonV0,
		Field:   "goal_progress",
		Message: idleSelfImprovementGoalReviewReplanRecommendedActionV0,
	})
	return result
}

func idleSelfImprovementGoalProgressEvaluationFromObservationV0(
	config ConfigV0,
	state StateV0,
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
	now time.Time,
) idleSelfImprovementGoalProgressEvaluationV0 {
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	progressSignature := idleSelfImprovementGoalUsefulProgressSignatureV0(state, result)
	previousSignature := strings.TrimSpace(state.IdleSelfImprovementGoalUsefulProgressSignature)
	progressObserved := progressSignature != "" && progressSignature != previousSignature
	lastProgressAt := parseServerTimeV0(state.IdleSelfImprovementGoalUsefulProgressAt)
	if lastProgressAt.IsZero() {
		lastProgressAt = parseServerTimeV0(state.IdleSelfImprovementCheck)
	}
	if lastProgressAt.IsZero() || progressObserved {
		lastProgressAt = now.UTC()
	}
	consumption := idleSelfImprovementGoalObservedConsumptionV0(observation, result)
	previousConsumption := state.IdleSelfImprovementGoalObservedConsumption
	consumptionGrowing := previousConsumption > 0 && consumption > previousConsumption
	invalidSignature := idleSelfImprovementGoalInvalidCheckpointSignatureV0(result)
	invalidRepeats := 0
	if invalidSignature != "" {
		if invalidSignature == strings.TrimSpace(state.IdleSelfImprovementGoalInvalidCheckpointSignature) {
			invalidRepeats = state.IdleSelfImprovementGoalInvalidCheckpointRepeats + 1
		} else {
			invalidRepeats = 1
		}
	}
	window := NormalizeSelfWatchdogConfigV0(config.SelfWatchdog).NoProgressFor
	windowExceeded := !progressObserved && !lastProgressAt.IsZero() && now.UTC().Sub(lastProgressAt) >= window
	evidenceRefs := []string{}
	if consumptionGrowing {
		evidenceRefs = append(evidenceRefs, "evidence-ref-goal-consumption-growing")
	}
	if windowExceeded {
		evidenceRefs = append(evidenceRefs, "evidence-ref-goal-last-useful-progress-window-exceeded")
	}
	invalidRepeated := invalidRepeats >= 2
	if invalidRepeated {
		evidenceRefs = append(evidenceRefs, "evidence-ref-goal-invalid-checkpoint-repeated")
	}
	shouldBlock := !progressObserved && windowExceeded && consumptionGrowing
	if shouldBlock {
		evidenceRefs = append(evidenceRefs,
			"evidence-ref-goal-high-consumption-without-progress",
			"evidence-ref-recommended-action-review-replan-goal-first",
		)
	}
	return idleSelfImprovementGoalProgressEvaluationV0{
		ProgressObserved:           progressObserved,
		ProgressSignature:          progressSignature,
		LastProgressAt:             lastProgressAt,
		ObservedConsumption:        consumption,
		ConsumptionGrowing:         consumptionGrowing,
		WindowExceeded:             windowExceeded,
		InvalidCheckpointSignature: invalidSignature,
		InvalidCheckpointRepeats:   invalidRepeats,
		InvalidCheckpointRepeated:  invalidRepeated,
		ShouldBlock:                shouldBlock,
		EvidenceRefs:               compactConfigStringsV0(evidenceRefs),
	}
}

func idleSelfImprovementGoalProgressUpdateFromEvaluationV0(
	evaluation idleSelfImprovementGoalProgressEvaluationV0,
) idleSelfImprovementGoalProgressUpdateV0 {
	update := idleSelfImprovementGoalProgressUpdateV0{
		ObservedConsumption:        evaluation.ObservedConsumption,
		InvalidCheckpointSignature: evaluation.InvalidCheckpointSignature,
		InvalidCheckpointRepeats:   evaluation.InvalidCheckpointRepeats,
	}
	if evaluation.ProgressObserved {
		update.UsefulProgressAt = evaluation.LastProgressAt
		update.UsefulProgressSignature = evaluation.ProgressSignature
		update.ClearInvalidCheckpoint = true
	}
	return update
}

func (runtime *RuntimeV0) blockIdleSelfImprovementGoalWithoutProgressV0(
	ctx context.Context,
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
	evaluation idleSelfImprovementGoalProgressEvaluationV0,
) orquestagoal.GoalWorkResultV0 {
	result = orquestagoal.NormalizeGoalWorkResultV0(result)
	if strings.TrimSpace(result.GoalRef) == "" {
		result.GoalRef = strings.TrimSpace(observation.State.GoalRef)
	}
	if strings.TrimSpace(result.ExternalGoalRef) == "" {
		result.ExternalGoalRef = strings.TrimSpace(observation.State.ExternalGoalRef)
	}
	evidenceRefs := compactConfigStringsV0(append([]string{
		"evidence-ref-goal-high-consumption-without-progress",
	}, evaluation.EvidenceRefs...))
	blocked := result
	blocked.Status = orquestagoal.GoalStatusBlockedV0
	blocked.Summary = idleSelfImprovementGoalHighConsumptionNoProgressReasonV0
	blocked.ReworkPlanRefs = compactConfigStringsV0(append(
		blocked.ReworkPlanRefs,
		idleSelfImprovementGoalReviewReplanRefV0,
	))
	blocked.EvidenceRefs = compactConfigStringsV0(append(blocked.EvidenceRefs, evidenceRefs...))
	blocked.Issues = append(blocked.Issues, orquestagoal.GoalWorkIssueV0{
		Code:   idleSelfImprovementGoalHighConsumptionNoProgressReasonV0,
		Field:  "goal_progress",
		Detail: "recommended_action=" + idleSelfImprovementGoalReviewReplanRecommendedActionV0,
	})
	stopEvidence, stopIssues := runtime.requestIdleSelfImprovementGoalCooperativeStopV0(ctx, observation, blocked)
	blocked.EvidenceRefs = compactConfigStringsV0(append(blocked.EvidenceRefs, stopEvidence...))
	blocked.Issues = append(blocked.Issues, stopIssues...)
	blocked = orquestagoal.NormalizeGoalWorkResultV0(blocked)
	runtime.persistBlockedIdleSelfImprovementGoalStateV0(ctx, observation.State, blocked)
	return blocked
}

func (runtime *RuntimeV0) requestIdleSelfImprovementGoalCooperativeStopV0(
	ctx context.Context,
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
) ([]string, []orquestagoal.GoalWorkIssueV0) {
	runRef := strings.TrimSpace(observation.State.RunRef)
	if runRef == "" {
		return []string{"evidence-ref-goal-cooperative-stop-run-ref-missing"}, []orquestagoal.GoalWorkIssueV0{{
			Code:  "goal_cooperative_stop_run_ref_missing",
			Field: "run_ref",
		}}
	}
	if runtime == nil || runtime.goalStopper == nil {
		return []string{"evidence-ref-goal-cooperative-stop-port-unavailable"}, []orquestagoal.GoalWorkIssueV0{{
			Code:  "goal_cooperative_stop_port_unavailable",
			Field: "goal_stopper",
		}}
	}
	stopResult, err := runtime.goalStopper.RequestGoalCooperativeStopV0(ctx, GoalCooperativeStopRequestV0{
		RunRef:                      runRef,
		GoalRef:                     strings.TrimSpace(result.GoalRef),
		ExternalGoalRef:             strings.TrimSpace(result.ExternalGoalRef),
		RequireConfirmedBackendStop: true,
		Reason:                      idleSelfImprovementGoalHighConsumptionNoProgressReasonV0,
		RecommendedAction:           idleSelfImprovementGoalReviewReplanRecommendedActionV0,
		RequestedBy:                 "orquesta-server-goal-progress-governor",
		IdempotencyKey:              "idem-goal-progress-stop-" + serverGoalProgressSafeRefPartV0(runRef),
		EvidenceRefs:                result.EvidenceRefs,
	})
	if err != nil {
		return []string{"evidence-ref-goal-cooperative-stop-request-failed"}, []orquestagoal.GoalWorkIssueV0{{
			Code:   "goal_cooperative_stop_request_failed",
			Field:  "goal_stopper",
			Detail: err.Error(),
		}}
	}
	refs := append([]string(nil), stopResult.EvidenceRefs...)
	if stopResult.Requested {
		refs = append(refs, "evidence-ref-goal-cooperative-stop-requested")
	}
	return compactConfigStringsV0(refs), nil
}

func (runtime *RuntimeV0) persistBlockedIdleSelfImprovementGoalStateV0(
	ctx context.Context,
	observedState orquestagoal.GoalWorkStateV0,
	blocked orquestagoal.GoalWorkResultV0,
) {
	if runtime == nil || runtime.goalStateStore == nil {
		return
	}
	runRef := strings.TrimSpace(observedState.RunRef)
	if runRef == "" {
		return
	}
	state := observedState
	if loaded, err := runtime.goalStateStore.LoadGoalWorkStateV0(ctx, runRef); err == nil {
		state = loaded
	}
	state.Status = orquestagoal.GoalStatusBlockedV0
	state.LastResult = &blocked
	state.LastClosure = &orquestagoal.GoalClosureValidationV0{
		Status:       orquestagoal.GoalStatusBlockedV0,
		NeedsRework:  true,
		EvidenceRefs: append([]string(nil), blocked.EvidenceRefs...),
		Issues: []orquestagoal.GoalWorkIssueV0{{
			Code:  idleSelfImprovementGoalHighConsumptionNoProgressReasonV0,
			Field: "goal_progress",
		}},
	}
	state.EvidenceRefs = compactConfigStringsV0(append(state.EvidenceRefs, blocked.EvidenceRefs...))
	_ = runtime.goalStateStore.SaveGoalWorkStateV0(ctx, state)
}

func (tracker *StatusTrackerV0) MarkIdleSelfImprovementGoalProgressV0(
	update idleSelfImprovementGoalProgressUpdateV0,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		if !update.UsefulProgressAt.IsZero() {
			state.IdleSelfImprovementGoalUsefulProgressAt = formatTimeV0(update.UsefulProgressAt)
		}
		if strings.TrimSpace(update.UsefulProgressSignature) != "" {
			state.IdleSelfImprovementGoalUsefulProgressSignature = strings.TrimSpace(update.UsefulProgressSignature)
		}
		if update.ObservedConsumption > 0 {
			state.IdleSelfImprovementGoalObservedConsumption = update.ObservedConsumption
		}
		if update.ClearInvalidCheckpoint {
			state.IdleSelfImprovementGoalInvalidCheckpointSignature = ""
			state.IdleSelfImprovementGoalInvalidCheckpointRepeats = 0
			return
		}
		if strings.TrimSpace(update.InvalidCheckpointSignature) != "" {
			state.IdleSelfImprovementGoalInvalidCheckpointSignature = strings.TrimSpace(update.InvalidCheckpointSignature)
			state.IdleSelfImprovementGoalInvalidCheckpointRepeats = update.InvalidCheckpointRepeats
		}
	})
}

func idleSelfImprovementGoalObservationIndexV0(
	state StateV0,
	result orquestagoal.GoalWorkObserveActiveResultV0,
) (int, orquestagoal.GoalWorkResultV0, bool) {
	expected := idleSelfImprovementGoalExpectedRefsV0(state)
	if len(expected) == 0 {
		return -1, orquestagoal.GoalWorkResultV0{}, false
	}
	for index, observation := range result.Observations {
		observed := orquestagoal.NormalizeGoalWorkResultV0(observation.Result)
		for _, ref := range goalObserverObservationRefsV0(observation, observed) {
			if _, ok := expected[ref]; ok {
				if strings.TrimSpace(observed.GoalRef) == "" {
					observed.GoalRef = strings.TrimSpace(observation.State.GoalRef)
				}
				if strings.TrimSpace(observed.ExternalGoalRef) == "" {
					observed.ExternalGoalRef = strings.TrimSpace(observation.State.ExternalGoalRef)
				}
				return index, observed, true
			}
		}
	}
	return -1, orquestagoal.GoalWorkResultV0{}, false
}

func idleSelfImprovementGoalExpectedRefsV0(state StateV0) map[string]struct{} {
	expected := map[string]struct{}{}
	if message := state.IdleSelfImprovementOperationalMessage; message != nil {
		for _, ref := range compactConfigStringsV0(message.GoalRefs) {
			expected[ref] = struct{}{}
		}
	}
	if state.IdleSelfImprovementGoalSpec != nil {
		spec := orquestagoal.NormalizeGoalWorkSpecV0(*state.IdleSelfImprovementGoalSpec)
		if strings.TrimSpace(spec.GoalRef) != "" {
			expected[strings.TrimSpace(spec.GoalRef)] = struct{}{}
		}
	}
	if state.IdleSelfImprovementGoalReceipt != nil {
		receipt := copyGoalLaunchReceiptForServerStateV0(*state.IdleSelfImprovementGoalReceipt)
		for _, ref := range []string{receipt.GoalRef, receipt.ExternalGoalRef} {
			ref = strings.TrimSpace(ref)
			if ref != "" {
				expected[ref] = struct{}{}
			}
		}
	}
	return expected
}

func idleSelfImprovementGoalUsefulProgressSignatureV0(
	state StateV0,
	result orquestagoal.GoalWorkResultV0,
) string {
	writeSet := idleSelfImprovementGoalWriteSetV0(state)
	markers := []string{}
	for _, artifactPath := range result.ArtifactPaths {
		if clean, ok := serverGoalProgressPathUnderWriteSetV0(artifactPath, writeSet); ok {
			markers = append(markers, "artifact_path:"+clean)
		}
	}
	for _, artifact := range result.MaterializedArtifacts {
		clean, underWriteSet := serverGoalProgressPathUnderWriteSetV0(
			firstNonEmptyConfigStringV0(artifact.Path, artifact.Scope),
			writeSet,
		)
		if underWriteSet && !serverGoalProgressInvalidCheckpointArtifactV0(artifact) {
			markers = append(markers, "artifact:"+clean+"|"+strings.TrimSpace(artifact.ArtifactRef)+"|"+strings.TrimSpace(artifact.Status))
		}
		if serverGoalProgressValidCheckpointArtifactV0(artifact) {
			markers = append(markers, "checkpoint_valid:"+strings.TrimSpace(artifact.ArtifactRef)+"|"+clean)
		}
	}
	for _, ref := range append(result.EvidenceRefs, result.Checklist.CompletedRefs...) {
		ref = strings.TrimSpace(ref)
		if serverGoalProgressBackendPhaseRefV0(ref) {
			markers = append(markers, "backend_phase:"+ref)
		}
	}
	return serverGoalProgressSignatureV0(markers)
}

func idleSelfImprovementGoalInvalidCheckpointSignatureV0(
	result orquestagoal.GoalWorkResultV0,
) string {
	markers := []string{}
	for _, artifact := range result.MaterializedArtifacts {
		if serverGoalProgressInvalidCheckpointArtifactV0(artifact) {
			markers = append(markers, strings.Join(compactConfigStringsV0([]string{
				artifact.ArtifactRef,
				artifact.Path,
				artifact.ArtifactType,
				artifact.Status,
			}), "|"))
		}
	}
	return serverGoalProgressSignatureV0(markers)
}

func idleSelfImprovementGoalWriteSetV0(state StateV0) []string {
	scopes := []string{}
	if state.IdleSelfImprovementGoalSpec != nil {
		spec := orquestagoal.NormalizeGoalWorkSpecV0(*state.IdleSelfImprovementGoalSpec)
		for _, scope := range spec.WriteSet {
			scopes = append(scopes, scope.Path)
		}
	}
	return compactConfigStringsV0(scopes)
}

func idleSelfImprovementGoalObservedConsumptionV0(
	observation orquestagoal.GoalWorkObserveResultV0,
	result orquestagoal.GoalWorkResultV0,
) int64 {
	usage := result.ContextBudget.ContextBudgetTotalBytes
	if usage <= 0 {
		usage = result.ContextBudget.StaticPromptBytes +
			result.ContextBudget.QueriedContextBytes +
			result.ContextBudget.MaterializedContextBytes +
			result.ContextBudget.DynamicContextBytes
	}
	refs := append([]string(nil), result.EvidenceRefs...)
	refs = append(refs, observation.EvidenceRefs...)
	usage = maxInt64V0(usage, serverGoalProgressTokensFromEvidenceRefsV0(refs))
	return usage
}

func serverGoalProgressTokensFromEvidenceRefsV0(refs []string) int64 {
	var usage int64
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		for _, prefix := range []string{
			"evidence-ref-codex-goal-cached-input-tokens-",
			"evidence-ref-codex-goal-input-tokens-",
			"evidence-ref-codex-goal-tokens-used-",
			"evidence-ref-goal-tokens-used-",
		} {
			if strings.HasPrefix(ref, prefix) {
				parsed, err := strconv.ParseInt(strings.TrimPrefix(ref, prefix), 10, 64)
				if err == nil && parsed > usage {
					usage = parsed
				}
			}
		}
	}
	return usage
}

func serverGoalProgressValidCheckpointArtifactV0(artifact orquestagoal.GoalMaterializedArtifactV0) bool {
	return serverGoalProgressCheckpointArtifactV0(artifact) &&
		strings.TrimSpace(artifact.Status) == "valid"
}

func serverGoalProgressInvalidCheckpointArtifactV0(artifact orquestagoal.GoalMaterializedArtifactV0) bool {
	return serverGoalProgressCheckpointArtifactV0(artifact) &&
		strings.TrimSpace(artifact.Status) == "invalid"
}

func serverGoalProgressCheckpointArtifactV0(artifact orquestagoal.GoalMaterializedArtifactV0) bool {
	combined := strings.ToLower(strings.Join([]string{
		artifact.ArtifactRef,
		artifact.Path,
		artifact.ArtifactType,
	}, "|"))
	return strings.Contains(combined, "checkpoint")
}

func serverGoalProgressBackendPhaseRefV0(ref string) bool {
	ref = strings.ToLower(strings.TrimSpace(ref))
	if ref == "" || strings.Contains(ref, "checkpoint") {
		return false
	}
	return strings.Contains(ref, "phase") || strings.Contains(ref, "fase")
}

func serverGoalProgressPathUnderWriteSetV0(candidate string, writeSet []string) (string, bool) {
	clean, ok := serverGoalProgressCleanRelPathV0(candidate)
	if !ok || len(writeSet) == 0 {
		return "", false
	}
	for _, scope := range writeSet {
		cleanScope, ok := serverGoalProgressCleanRelPathV0(scope)
		if !ok {
			continue
		}
		if clean == cleanScope || strings.HasPrefix(clean, cleanScope+"/") {
			return clean, true
		}
	}
	return "", false
}

func serverGoalProgressCleanRelPathV0(value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	if value == "" || strings.HasPrefix(value, "/") {
		return "", false
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", false
	}
	return clean, true
}

func serverGoalProgressSignatureV0(markers []string) string {
	markers = compactConfigStringsV0(markers)
	if len(markers) == 0 {
		return ""
	}
	sort.Strings(markers)
	sum := sha256.Sum256([]byte(strings.Join(markers, "\n")))
	return fmt.Sprintf("sha256:%x", sum[:])
}

func serverGoalProgressSafeRefPartV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte('-')
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}

func maxInt64V0(left int64, right int64) int64 {
	if right > left {
		return right
	}
	return left
}
