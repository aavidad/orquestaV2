package orquestarunqueue

import (
	"context"
	orquestacoreconcurrency "orquesta/modulos/orquesta-core-concurrency"
	"time"
)

const (
	RunQueueSchemaVersionV0     = "run_queue.v0"
	WorksetClaimSchemaVersionV0 = orquestacoreconcurrency.WorksetClaimSchemaVersionV0

	RunStatusReadyV0     = "ready"
	RunStatusRunningV0   = "running"
	RunStatusPausedV0    = "paused"
	RunStatusDeliveredV0 = "delivered"
	RunStatusCanceledV0  = "canceled"
	RunStatusStoppedV0   = "stopped"
	RunStatusClosedV0    = "closed"
)

const (
	RunQueueFairnessGroupPausedV0  = "fairness_group_paused"
	RunQueueFairnessGroupBoostedV0 = "fairness_group_boosted"
	RunQueueFairnessGroupMissingV0 = "fairness_group_missing"
)

type RunQueueReaderPortV0 interface {
	ListRunSchedulingCandidatesV0(context.Context, RunQueueReadRequestV0) ([]RunSchedulingCandidateV0, error)
}

type RunQueuePriorityWriterPortV0 interface {
	SetRunPriorityV0(context.Context, RunQueuePriorityCommandV0) (RunSchedulingCandidateV0, error)
}

type RunQueuePortV0 interface {
	RunQueueReaderPortV0
	RunQueuePriorityWriterPortV0
}

type WorksetClaimV0 = orquestacoreconcurrency.WorksetClaimV0
type ScopeRefV0 = orquestacoreconcurrency.ScopeRefV0

type RunQueueReadRequestV0 struct {
	QueueRef             string   `json:"queue_ref,omitempty"`
	AppRefs              []string `json:"app_refs,omitempty"`
	Limit                int      `json:"limit,omitempty"`
	IncludeNonExecutable bool     `json:"include_non_executable,omitempty"`
}

type RunQueuePriorityCommandV0 struct {
	RunRef           string                                   `json:"run_ref"`
	QueueRef         string                                   `json:"queue_ref,omitempty"`
	AppRef           string                                   `json:"app_ref,omitempty"`
	Status           string                                   `json:"status,omitempty"`
	PriorityScore    int                                      `json:"priority_score"`
	UpdatedAt        time.Time                                `json:"updated_at,omitempty"`
	FairnessGroupRef string                                   `json:"fairness_group_ref,omitempty"`
	AttemptGroup     RunQueueAttemptGroupV0                   `json:"attempt_group,omitempty"`
	ParentRunRef     string                                   `json:"parent_run_ref,omitempty"`
	SupersedesRunRef string                                   `json:"supersedes_run_ref,omitempty"`
	RescueReason     string                                   `json:"rescue_reason,omitempty"`
	RequestedBy      string                                   `json:"requested_by,omitempty"`
	Reason           string                                   `json:"reason,omitempty"`
	IdempotencyKey   string                                   `json:"idempotency_key,omitempty"`
	EvidenceRefs     []string                                 `json:"evidence_refs,omitempty"`
	WorksetClaims    []orquestacoreconcurrency.WorksetClaimV0 `json:"workset_claims,omitempty"`
}

type RunSchedulingCandidateV0 struct {
	RunRef           string                                   `json:"run_ref"`
	AppRef           string                                   `json:"app_ref"`
	Status           string                                   `json:"status"`
	PriorityScore    int                                      `json:"priority_score"`
	UpdatedAt        time.Time                                `json:"updated_at"`
	FairnessGroupRef string                                   `json:"fairness_group_ref,omitempty"`
	AttemptGroup     RunQueueAttemptGroupV0                   `json:"attempt_group,omitempty"`
	ParentRunRef     string                                   `json:"parent_run_ref,omitempty"`
	SupersedesRunRef string                                   `json:"supersedes_run_ref,omitempty"`
	RescueReason     string                                   `json:"rescue_reason,omitempty"`
	EvidenceRefs     []string                                 `json:"evidence_refs,omitempty"`
	WorksetClaims    []orquestacoreconcurrency.WorksetClaimV0 `json:"workset_claims,omitempty"`
}

type RunQueueAttemptGroupV0 struct {
	GroupRef     string   `json:"group_ref,omitempty"`
	ConsumerRef  string   `json:"consumer_ref,omitempty"`
	ObjectiveRef string   `json:"objective_ref,omitempty"`
	WorkItemRef  string   `json:"work_item_ref,omitempty"`
	WriteSetRefs []string `json:"write_set_refs,omitempty"`
}

type RunQueueAttemptProjectionV0 struct {
	GroupRef         string                 `json:"group_ref"`
	AttemptGroup     RunQueueAttemptGroupV0 `json:"attempt_group,omitempty"`
	OriginalRunRef   string                 `json:"original_run_ref,omitempty"`
	RunRefs          []string               `json:"run_refs,omitempty"`
	RescueRunRefs    []string               `json:"rescue_run_refs,omitempty"`
	ActiveAttemptRef string                 `json:"active_attempt_ref,omitempty"`
	ParentRunRef     string                 `json:"parent_run_ref,omitempty"`
	SupersedesRunRef string                 `json:"supersedes_run_ref,omitempty"`
	RescueReason     string                 `json:"rescue_reason,omitempty"`
	Status           string                 `json:"status,omitempty"`
	StatusCounts     map[string]int         `json:"status_counts,omitempty"`
	EvidenceRefs     []string               `json:"evidence_refs,omitempty"`
}

type RankedRunCandidateV0 struct {
	RunSchedulingCandidateV0
	Rank                int      `json:"rank"`
	AgingBoost          int      `json:"aging_boost"`
	FairnessBoost       int      `json:"fairness_boost,omitempty"`
	FairnessPaused      bool     `json:"fairness_paused,omitempty"`
	FairnessReasonCodes []string `json:"fairness_reason_codes,omitempty"`
}

type RunQueueWorksetBlockV0 struct {
	RunRef          string   `json:"run_ref"`
	BlockedByRunRef string   `json:"blocked_by_run_ref,omitempty"`
	ConflictRefs    []string `json:"conflict_refs,omitempty"`
	Reason          string   `json:"reason"`
}

type RunQueueWorksetEvaluationV0 struct {
	AllowedRunRefs []string                 `json:"allowed_run_refs,omitempty"`
	Blocks         []RunQueueWorksetBlockV0 `json:"blocks,omitempty"`
	EvidenceRefs   []string                 `json:"evidence_refs,omitempty"`
}

type RunQueueWorksetPolicyV0 struct {
	RequireClaims bool `json:"require_claims,omitempty"`
}

type RunQueueRankingPolicyV0 struct {
	Now                         time.Time            `json:"now"`
	AgingAfterSeconds           int                  `json:"aging_after_seconds"`
	AgingStepSeconds            int                  `json:"aging_step_seconds"`
	AgingBoostPerStep           int                  `json:"aging_boost_per_step"`
	MaxAgingBoost               int                  `json:"max_aging_boost"`
	MissingUpdatedAtLast        bool                 `json:"missing_updated_at_last"`
	FairnessWindowSeconds       int                  `json:"fairness_window_seconds,omitempty"`
	MaxRunsPerFairnessGroup     int                  `json:"max_runs_per_fairness_group,omitempty"`
	FairnessBoostAfterSeconds   int                  `json:"fairness_boost_after_seconds,omitempty"`
	FairnessBoostPerWindow      int                  `json:"fairness_boost_per_window,omitempty"`
	MissingFairnessGroupPolicy  string               `json:"missing_fairness_group_policy,omitempty"`
	DefaultFairnessGroupRef     string               `json:"default_fairness_group_ref,omitempty"`
	FairnessGroupLastSelectedAt map[string]time.Time `json:"fairness_group_last_selected_at,omitempty"`
	FairnessGroupRunCounts      map[string]int       `json:"fairness_group_run_counts,omitempty"`
	RequireWorksetClaims        bool                 `json:"require_workset_claims,omitempty"`
}

func DefaultRunQueueRankingPolicyV0(now time.Time) RunQueueRankingPolicyV0 {
	return RunQueueRankingPolicyV0{
		Now:                  now,
		AgingAfterSeconds:    1800,
		AgingStepSeconds:     1800,
		AgingBoostPerStep:    1,
		MaxAgingBoost:        3,
		MissingUpdatedAtLast: true,
	}
}
