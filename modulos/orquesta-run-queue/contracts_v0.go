package orquestarunqueue

import (
	"context"
	"time"
)

const (
	RunQueueSchemaVersionV0 = "run_queue.v0"

	RunStatusPausedV0    = "paused"
	RunStatusDeliveredV0 = "delivered"
	RunStatusCanceledV0  = "canceled"
	RunStatusStoppedV0   = "stopped"
	RunStatusClosedV0    = "closed"
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

type RunQueueReadRequestV0 struct {
	QueueRef string   `json:"queue_ref,omitempty"`
	AppRefs  []string `json:"app_refs,omitempty"`
	Limit    int      `json:"limit,omitempty"`
}

type RunQueuePriorityCommandV0 struct {
	RunRef         string    `json:"run_ref"`
	QueueRef       string    `json:"queue_ref,omitempty"`
	AppRef         string    `json:"app_ref,omitempty"`
	Status         string    `json:"status,omitempty"`
	PriorityScore  int       `json:"priority_score"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	RequestedBy    string    `json:"requested_by,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string  `json:"evidence_refs,omitempty"`
}

type RunSchedulingCandidateV0 struct {
	RunRef           string    `json:"run_ref"`
	AppRef           string    `json:"app_ref"`
	Status           string    `json:"status"`
	PriorityScore    int       `json:"priority_score"`
	UpdatedAt        time.Time `json:"updated_at"`
	FairnessGroupRef string    `json:"fairness_group_ref,omitempty"`
	EvidenceRefs     []string  `json:"evidence_refs,omitempty"`
}

type RankedRunCandidateV0 struct {
	RunSchedulingCandidateV0
	Rank       int `json:"rank"`
	AgingBoost int `json:"aging_boost"`
}

type RunQueueRankingPolicyV0 struct {
	Now                  time.Time `json:"now"`
	AgingAfterSeconds    int       `json:"aging_after_seconds"`
	AgingStepSeconds     int       `json:"aging_step_seconds"`
	AgingBoostPerStep    int       `json:"aging_boost_per_step"`
	MaxAgingBoost        int       `json:"max_aging_boost"`
	MissingUpdatedAtLast bool      `json:"missing_updated_at_last"`
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
