package stopreason

import "strings"

const SchemaVersionV0 = "orquesta_supervisor_stop_reason_projection.v0"

const (
	SourceRunSupervisorV0             = "run_supervisor"
	SourceDirectorSupervisorV0        = "director_supervisor"
	SourceDirectorSupervisedBurstV0   = "director_supervised_burst"
	CategoryContinueV0                = "continue"
	CategoryIdleV0                    = "idle"
	CategoryWaitOutboxV0              = "wait_outbox"
	CategoryWaitExternalV0            = "wait_external"
	CategoryNeedsDirectorV0           = "needs_director"
	CategoryBlockedV0                 = "blocked"
	CategoryQuiescentV0               = "quiescent"
	CategoryBudgetV0                  = "budget"
	CategoryErrorV0                   = "error"
	CategoryContextV0                 = "context"
	CategoryConfigV0                  = "config"
	PublicReasonContinueV0            = "continue_budget_available"
	PublicReasonIdleNoExecutionV0     = "idle_no_execution"
	PublicReasonWaitOutboxV0          = "wait_outbox"
	PublicReasonWaitExternalV0        = "wait_external"
	PublicReasonNeedsDirectorV0       = "needs_director"
	PublicReasonBlockedV0             = "blocked"
	PublicReasonQuiescentV0           = "quiescent"
	PublicReasonBudgetMaxTicksV0      = "budget_max_ticks"
	PublicReasonBudgetMaxExecutionsV0 = "budget_max_executions"
	PublicReasonBudgetMaxStepsV0      = "budget_max_steps"
	PublicReasonErrorTickV0           = "error_tick"
	PublicReasonErrorStepV0           = "error_step"
	PublicReasonContextDoneV0         = "context_done"
	PublicReasonConfigMissingTickerV0 = "config_missing_ticker"
	PublicReasonUnknownV0             = "unknown"
)

type ProjectionInputV0 struct {
	Source            string
	StopReason        string
	Executions        int
	Skips             int
	Ticks             int
	Steps             int
	PendingOutboxRefs []string
	WaitingReasons    []string
	BlockedRefs       []string
	ErrorRefs         []string
	EvidenceRefs      []string
}

type ProjectionV0 struct {
	SchemaVersion     string   `json:"schema_version"`
	Source            string   `json:"source"`
	StopReason        string   `json:"stop_reason"`
	Category          string   `json:"category"`
	PublicReason      string   `json:"public_reason"`
	Executions        int      `json:"executions,omitempty"`
	Skips             int      `json:"skips,omitempty"`
	Ticks             int      `json:"ticks,omitempty"`
	Steps             int      `json:"steps,omitempty"`
	PendingOutboxRefs []string `json:"pending_outbox_refs,omitempty"`
	WaitingReasons    []string `json:"waiting_reasons,omitempty"`
	BlockedRefs       []string `json:"blocked_refs,omitempty"`
	ErrorRefs         []string `json:"error_refs,omitempty"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

func ProjectV0(input ProjectionInputV0) ProjectionV0 {
	source := strings.TrimSpace(input.Source)
	stop := strings.TrimSpace(input.StopReason)
	category, public := classifyV0(stop)
	return ProjectionV0{
		SchemaVersion:     SchemaVersionV0,
		Source:            source,
		StopReason:        stop,
		Category:          category,
		PublicReason:      public,
		Executions:        input.Executions,
		Skips:             input.Skips,
		Ticks:             input.Ticks,
		Steps:             input.Steps,
		PendingOutboxRefs: compactStringsV0(input.PendingOutboxRefs),
		WaitingReasons:    compactStringsV0(input.WaitingReasons),
		BlockedRefs:       compactStringsV0(input.BlockedRefs),
		ErrorRefs:         compactStringsV0(input.ErrorRefs),
		EvidenceRefs:      compactStringsV0(input.EvidenceRefs),
	}
}

func classifyV0(stop string) (string, string) {
	switch stop {
	case "continue":
		return CategoryContinueV0, PublicReasonContinueV0
	case "no_execution":
		return CategoryIdleV0, PublicReasonIdleNoExecutionV0
	case "wait_outbox":
		return CategoryWaitOutboxV0, PublicReasonWaitOutboxV0
	case "wait_external", "candidate_pending":
		return CategoryWaitExternalV0, PublicReasonWaitExternalV0
	case "needs_director":
		return CategoryNeedsDirectorV0, PublicReasonNeedsDirectorV0
	case "blocked":
		return CategoryBlockedV0, PublicReasonBlockedV0
	case "stop_quiescent":
		return CategoryQuiescentV0, PublicReasonQuiescentV0
	case "max_ticks":
		return CategoryBudgetV0, PublicReasonBudgetMaxTicksV0
	case "max_executions":
		return CategoryBudgetV0, PublicReasonBudgetMaxExecutionsV0
	case "stop_max_steps":
		return CategoryBudgetV0, PublicReasonBudgetMaxStepsV0
	case "tick_error":
		return CategoryErrorV0, PublicReasonErrorTickV0
	case "stop_error":
		return CategoryErrorV0, PublicReasonErrorStepV0
	case "context_done":
		return CategoryContextV0, PublicReasonContextDoneV0
	case "ticker_required":
		return CategoryConfigV0, PublicReasonConfigMissingTickerV0
	default:
		return CategoryBlockedV0, PublicReasonUnknownV0
	}
}

func compactStringsV0(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
