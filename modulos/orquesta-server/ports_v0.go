package orquestaserver

import (
	"context"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

type SupervisorPortV0 interface {
	RunGlobalSupervisorV0(
		context.Context,
		orquestarunsupervisor.RunSupervisorCommandV0,
	) (orquestarunsupervisor.RunSupervisorResultV0, error)
}

type IdleSelfImprovementPortV0 interface {
	PrepareIdleSelfImprovementV0(
		context.Context,
		IdleSelfImprovementRequestV0,
	) (IdleSelfImprovementResultV0, error)
}

type IdleSelfImprovementPlannerPortV0 interface {
	PlanIdleSelfImprovementV0(
		context.Context,
		IdleSelfImprovementPlanRequestV0,
	) (IdleSelfImprovementPlanResultV0, error)
}

type IdleSelfImprovementPlanRequestV0 struct {
	BaseRequest      IdleSelfImprovementRequestV0
	MaxRequests      int
	Trigger          string
	QueueSize        int
	FreeCapacity     int
	KnownRunRefs     []string
	KnownRequestRefs []string
}

type IdleSelfImprovementPlanResultV0 struct {
	Requests     []IdleSelfImprovementRequestV0
	EvidenceRefs []string
	Message      string
}

type IdleSelfImprovementRequestV0 struct {
	RequestRef         string
	CorrelationID      string
	ProjectRef         string
	WorktreeRef        string
	BranchRef          string
	RequestedBy        string
	Source             string
	FailureKind        string
	FailureSummary     string
	SuggestedArea      string
	WriteSet           []string
	RequiredTests      []string
	AcceptanceCriteria []string
	CompactRules       []string
	ContextRefs        []string
	EvidenceRefs       []string
	OccurredAt         string
	PriorityScore      int
}

type IdleSelfImprovementResultV0 struct {
	Accepted     bool
	RunRef       string
	RequestRef   string
	Status       string
	Message      string
	NextActions  []string
	EvidenceRefs []string
}

type StateStorePortV0 interface {
	SaveServerStateV0(context.Context, StateV0) error
	LoadServerStateV0(context.Context) (StateV0, error)
}

type AuditSinkPortV0 interface {
	AppendAuditEventV0(context.Context, AuditEventV0) error
}

type StartupCheckPortV0 interface {
	PrepareStartupV0(context.Context, StartupCheckCommandV0) (StartupCheckResultV0, error)
}

type ClockPortV0 interface {
	Now() time.Time
}

type SystemClockV0 struct{}

func (SystemClockV0) Now() time.Time {
	return time.Now().UTC()
}
