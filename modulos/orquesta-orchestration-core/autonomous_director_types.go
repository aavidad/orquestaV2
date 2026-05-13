package orquestacionnucleoapp

import (
	"context"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

type AutonomousDirectorPolicyPortV0 interface {
	DecideAutonomousDirectorV0(
		ctx context.Context,
		input AutonomousDirectorDecisionInputV0,
	) (AutonomousDirectorDecisionV0, error)
}

type AutonomousDirectorDecisionInputV0 struct {
	Run      orquestacoreworkflow.OrchestrationRunV0
	Stats    DirectorRunStatsV0
	Limits   AutonomousDirectorLimitsV0
	Requests AutonomousDirectorRequestHintsV0
}

type AutonomousDirectorRequestHintsV0 struct {
	CorrelationID string
	EvidenceRefs  []string
}

type AutonomousDirectorLimitsV0 struct {
	MaxTeamSize          int
	MaxParallelAgents    int
	MaxBursts            int
	MaxStepsPerBurst     int
	MaxDispatchesPerWait int
	MaxCommandsPerCycle  int
	MaxOutboxPerCycle    int
}

type AutonomousDirectorDecisionV0 struct {
	TeamSize               int
	MaxParallelAgents      int
	MaxBursts              int
	MaxStepsPerBurst       int
	MaxDispatchesPerWait   int
	MaxCommandsPerCycle    int
	MaxOutboxPerCycle      int
	RecommendedCapacity    orquestacoreworkflow.OrchestrationCapacityRecommendationV0
	QualityRecommendations []AutonomousQualityRecommendationV0
	Summary                string
	EvidenceRefs           []string
}

const (
	AutonomousQualityActionReplanV0      = "replan"
	AutonomousQualityActionAskDirectorV0 = "ask_director"
	AutonomousQualityActionStopAgentV0   = "stop_agent"
)

type AutonomousQualityRecommendationV0 struct {
	SubjectRef        string   `json:"subject_ref"`
	SubjectKind       string   `json:"subject_kind"`
	SignalRef         string   `json:"signal_ref"`
	ReasonCode        string   `json:"reason_code"`
	RecommendedAction string   `json:"recommended_action"`
	Summary           string   `json:"summary"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type AutonomousDirectorLoopRequestV0 struct {
	Loop   ProgressiveLoopRequestV0
	Stats  DirectorRunStatsV0
	Policy AutonomousDirectorPolicyPortV0
	Limits AutonomousDirectorLimitsV0
}

type AutonomousDirectorLoopResultV0 struct {
	Decision AutonomousDirectorDecisionV0
	Loop     ProgressiveLoopResultV0
	Stats    AutonomousDirectorLoopStatsV0
}

type AutonomousBatchExecutorTunerV0 interface {
	WithAutonomousBatchConcurrencyV0(maxConcurrency int) OutboxDispatchBatchExecutorPortV0
}
