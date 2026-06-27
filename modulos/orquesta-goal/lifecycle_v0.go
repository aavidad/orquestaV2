package orquestagoal

import (
	"context"
	"strings"
)

type GoalWorkLifecyclePortsV0 struct {
	Launcher         GoalWorkLauncherPortV0
	Observer         GoalWorkObservationPortV0
	ClosureValidator GoalWorkClosureValidatorPortV0
	StateStore       GoalWorkStateStorePortV0
}

type GoalWorkStartRequestV0 struct {
	RunRef       string         `json:"run_ref,omitempty"`
	Spec         GoalWorkSpecV0 `json:"spec"`
	EvidenceRefs []string       `json:"evidence_refs,omitempty"`
}

type GoalWorkStartResultV0 struct {
	State        GoalWorkStateV0     `json:"state"`
	Receipt      GoalLaunchReceiptV0 `json:"receipt"`
	EvidenceRefs []string            `json:"evidence_refs,omitempty"`
}

type GoalWorkObserveRequestV0 struct {
	RunRef string `json:"run_ref"`
}

type GoalWorkObserveResultV0 struct {
	State            GoalWorkStateV0         `json:"state"`
	Result           GoalWorkResultV0        `json:"result,omitempty"`
	Closure          GoalClosureValidationV0 `json:"closure,omitempty"`
	Terminal         bool                    `json:"terminal,omitempty"`
	ClosureEvaluated bool                    `json:"closure_evaluated,omitempty"`
	Accepted         bool                    `json:"accepted,omitempty"`
	NeedsRework      bool                    `json:"needs_rework,omitempty"`
	EvidenceRefs     []string                `json:"evidence_refs,omitempty"`
}

type GoalWorkStateFromLaunchRequestV0 struct {
	RunRef        string              `json:"run_ref,omitempty"`
	Spec          GoalWorkSpecV0      `json:"spec"`
	LaunchReceipt GoalLaunchReceiptV0 `json:"launch_receipt"`
	EvidenceRefs  []string            `json:"evidence_refs,omitempty"`
}

type GoalWorkLifecycleIssueErrorV0 struct {
	Field string `json:"field"`
}

func (err GoalWorkLifecycleIssueErrorV0) Error() string {
	if strings.TrimSpace(err.Field) == "" {
		return "goal_work_lifecycle_invalid"
	}
	return "goal_work_lifecycle_invalid: " + strings.TrimSpace(err.Field)
}

func StartGoalWorkV0(
	ctx context.Context,
	request GoalWorkStartRequestV0,
	ports GoalWorkLifecyclePortsV0,
) (GoalWorkStartResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ports.Launcher == nil {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_launcher"}
	}
	if ports.StateStore == nil {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_store"}
	}
	spec := goalWorkSpecWithRunRefV0(request.Spec, request.RunRef)
	runRef, err := goalLifecycleRunRefV0(request.RunRef, spec)
	if err != nil {
		return GoalWorkStartResultV0{}, err
	}
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return GoalWorkStartResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "goal_spec"}
	}
	receipt, err := ports.Launcher.LaunchGoalWorkV0(ctx, spec)
	if err != nil {
		return GoalWorkStartResultV0{}, err
	}
	state, err := NewGoalWorkStateFromLaunchV0(GoalWorkStateFromLaunchRequestV0{
		RunRef:        runRef,
		Spec:          spec,
		LaunchReceipt: receipt,
		EvidenceRefs:  request.EvidenceRefs,
	})
	if err != nil {
		return GoalWorkStartResultV0{}, err
	}
	if err := ports.StateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		return GoalWorkStartResultV0{}, err
	}
	return GoalWorkStartResultV0{
		State:        state,
		Receipt:      state.LaunchReceipt,
		EvidenceRefs: append([]string(nil), state.EvidenceRefs...),
	}, nil
}

func ObserveGoalWorkV0(
	ctx context.Context,
	request GoalWorkObserveRequestV0,
	ports GoalWorkLifecyclePortsV0,
) (GoalWorkObserveResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runRef := strings.TrimSpace(request.RunRef)
	if runRef == "" {
		return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "run_ref"}
	}
	if ports.StateStore == nil {
		return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_state_store"}
	}
	if ports.Observer == nil {
		return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_observer"}
	}
	state, err := ports.StateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	state, err = NewGoalWorkStateV0(state)
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	result, err := ports.Observer.ObserveGoalWorkV0(ctx, GoalObservationRequestFromStateV0(state))
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	result = NormalizeGoalWorkResultV0(result)
	if issues := ValidateGoalWorkResultV0(result); len(issues) > 0 {
		return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "goal_result"}
	}
	state.LastResult = &result
	state.Status = result.Status
	state.EvidenceRefs = compactGoalStringsV0(append(state.EvidenceRefs, result.EvidenceRefs...))

	terminal := GoalWorkResultTerminalV0(result.Status)
	closure := GoalClosureValidationV0{}
	if terminal {
		if ports.ClosureValidator == nil {
			return GoalWorkObserveResultV0{}, GoalWorkLifecycleIssueErrorV0{Field: "ports.goal_closure_validator"}
		}
		closure, err = ports.ClosureValidator.ValidateGoalWorkClosureV0(ctx, state.Spec, result)
		if err != nil {
			return GoalWorkObserveResultV0{}, err
		}
		state.LastClosure = &closure
		state.EvidenceRefs = compactGoalStringsV0(append(state.EvidenceRefs, closure.EvidenceRefs...))
	}
	state, err = NewGoalWorkStateV0(state)
	if err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	if err := ports.StateStore.SaveGoalWorkStateV0(ctx, state); err != nil {
		return GoalWorkObserveResultV0{}, err
	}
	return GoalWorkObserveResultV0{
		State:            state,
		Result:           result,
		Closure:          closure,
		Terminal:         terminal,
		ClosureEvaluated: terminal,
		Accepted:         closure.Accepted,
		NeedsRework:      closure.NeedsRework,
		EvidenceRefs:     append([]string(nil), state.EvidenceRefs...),
	}, nil
}

func NewGoalWorkStateFromLaunchV0(
	request GoalWorkStateFromLaunchRequestV0,
) (GoalWorkStateV0, error) {
	spec := goalWorkSpecWithRunRefV0(request.Spec, request.RunRef)
	runRef, err := goalLifecycleRunRefV0(request.RunRef, spec)
	if err != nil {
		return GoalWorkStateV0{}, err
	}
	if issues := ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		return GoalWorkStateV0{}, GoalWorkLifecycleIssueErrorV0{Field: "goal_spec"}
	}
	receipt := NormalizeGoalLaunchReceiptV0(request.LaunchReceipt)
	if receipt.Status == "" {
		receipt.Status = GoalStatusRunningV0
	}
	if receipt.GoalRef == "" {
		receipt.GoalRef = spec.GoalRef
	}
	if receipt.ExternalGoalRef == "" {
		receipt.ExternalGoalRef = spec.GoalRef
	}
	if issues := ValidateGoalLaunchReceiptV0(receipt); len(issues) > 0 {
		return GoalWorkStateV0{}, GoalWorkLifecycleIssueErrorV0{Field: "launch_receipt"}
	}
	state := GoalWorkStateV0{
		SchemaVersion:   GoalWorkStateSchemaV0,
		RunRef:          runRef,
		GoalRef:         receipt.GoalRef,
		ExternalGoalRef: receipt.ExternalGoalRef,
		Status:          goalWorkStateStatusFromLaunchReceiptV0(receipt),
		Spec:            spec,
		LaunchReceipt:   receipt,
		EvidenceRefs: compactGoalStringsV0(append(
			append(append([]string(nil), request.EvidenceRefs...), spec.EvidenceRefs...),
			receipt.EvidenceRefs...,
		)),
	}
	return NewGoalWorkStateV0(state)
}

func NormalizeGoalLaunchReceiptV0(receipt GoalLaunchReceiptV0) GoalLaunchReceiptV0 {
	receipt.SchemaVersion = strings.TrimSpace(receipt.SchemaVersion)
	if receipt.SchemaVersion == "" {
		receipt.SchemaVersion = GoalWorkLaunchReceiptSchemaV0
	}
	receipt.Status = strings.TrimSpace(receipt.Status)
	receipt.GoalRef = strings.TrimSpace(receipt.GoalRef)
	receipt.ExternalGoalRef = strings.TrimSpace(receipt.ExternalGoalRef)
	for i := range receipt.EvidenceRefs {
		receipt.EvidenceRefs[i] = strings.TrimSpace(receipt.EvidenceRefs[i])
	}
	for i := range receipt.Issues {
		receipt.Issues[i].Code = strings.TrimSpace(receipt.Issues[i].Code)
		receipt.Issues[i].Field = strings.TrimSpace(receipt.Issues[i].Field)
	}
	return receipt
}

func ValidateGoalLaunchReceiptV0(receipt GoalLaunchReceiptV0) []GoalWorkIssueV0 {
	receipt = NormalizeGoalLaunchReceiptV0(receipt)
	var issues []GoalWorkIssueV0
	if receipt.Status != "" && !validGoalLaunchReceiptStatusV0(receipt.Status) {
		issues = append(issues, GoalWorkIssueV0{Code: ErrGoalStatusInvalidV0, Field: "status"})
	}
	validateGoalRefsV0(&issues, "goal_ref", receipt.GoalRef)
	validateGoalRefsV0(&issues, "external_goal_ref", receipt.ExternalGoalRef)
	for _, evidenceRef := range receipt.EvidenceRefs {
		validateRequiredGoalRefV0(&issues, "evidence_refs", evidenceRef)
	}
	return issues
}

func GoalObservationRequestFromStateV0(state GoalWorkStateV0) GoalObservationRequestV0 {
	return NormalizeGoalObservationRequestV0(GoalObservationRequestV0{
		GoalRef:         state.GoalRef,
		ExternalGoalRef: state.ExternalGoalRef,
	})
}

func GoalWorkResultTerminalV0(status string) bool {
	switch strings.TrimSpace(status) {
	case GoalStatusCompleteV0, GoalStatusBlockedV0, GoalStatusInvalidV0:
		return true
	default:
		return false
	}
}

func goalWorkSpecWithRunRefV0(spec GoalWorkSpecV0, runRef string) GoalWorkSpecV0 {
	spec = NormalizeGoalWorkSpecV0(spec)
	runRef = strings.TrimSpace(runRef)
	if spec.RunRef == "" {
		spec.RunRef = runRef
	}
	return NormalizeGoalWorkSpecV0(spec)
}

func goalLifecycleRunRefV0(requestRunRef string, spec GoalWorkSpecV0) (string, error) {
	requestRunRef = strings.TrimSpace(requestRunRef)
	specRunRef := strings.TrimSpace(spec.RunRef)
	if requestRunRef != "" && specRunRef != "" && requestRunRef != specRunRef {
		return "", GoalWorkLifecycleIssueErrorV0{Field: "run_ref"}
	}
	runRef := firstGoalLifecycleValueV0(requestRunRef, specRunRef)
	if runRef == "" {
		return "", GoalWorkLifecycleIssueErrorV0{Field: "run_ref"}
	}
	return runRef, nil
}

func goalWorkStateStatusFromLaunchReceiptV0(receipt GoalLaunchReceiptV0) string {
	switch strings.TrimSpace(receipt.Status) {
	case GoalStatusCompleteV0, GoalStatusBlockedV0, GoalStatusInvalidV0:
		return strings.TrimSpace(receipt.Status)
	default:
		return GoalStatusRunningV0
	}
}

func validGoalLaunchReceiptStatusV0(status string) bool {
	switch strings.TrimSpace(status) {
	case GoalStatusAcceptedV0,
		GoalStatusRunningV0,
		GoalStatusCompleteV0,
		GoalStatusBlockedV0,
		GoalStatusInvalidV0:
		return true
	default:
		return false
	}
}

func firstGoalLifecycleValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactGoalStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}
