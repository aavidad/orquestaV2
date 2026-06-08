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

type ResidentDirectorPortV0 interface {
	RunResidentDirectorV0(
		context.Context,
		ResidentDirectorCommandV0,
	) (ResidentDirectorResultV0, error)
}

type ResidentDirectorCommandV0 struct {
	MaxActions    int
	OccurredAt    string
	CorrelationID string
	EvidenceRefs  []string
}

type ResidentDirectorResultV0 struct {
	Status          string
	RunRef          string
	ExecutedActions int
	EvidenceRefs    []string
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

type IdleSelfImprovementBlockerPortV0 interface {
	IdleSelfImprovementBlockersV0(
		context.Context,
		IdleSelfImprovementBlockerRequestV0,
	) (IdleSelfImprovementBlockerResultV0, error)
}

type IdleSelfImprovementRequestFilterPortV0 interface {
	FilterIdleSelfImprovementRequestsV0(
		context.Context,
		IdleSelfImprovementRequestFilterRequestV0,
	) (IdleSelfImprovementRequestFilterResultV0, error)
}

type IdleSelfImprovementBlockerRequestV0 struct {
	KnownRunRefs     []string
	KnownRequestRefs []string
}

type IdleSelfImprovementBlockerResultV0 struct {
	Blocked        bool
	Reason         string
	RunRefs        []string
	EvidenceRefs   []string
	Message        string
	RecoveryAction string
	NextActions    []string
}

type IdleSelfImprovementPlanRequestV0 struct {
	BaseRequest           IdleSelfImprovementRequestV0
	MaxRequests           int
	Trigger               string
	QueueSize             int
	FreeCapacity          int
	Skips                 int
	KnownRunRefs          []string
	KnownRequestRefs      []string
	RetryableRunRefs      []string
	RetryableRequestRefs  []string
	RetryableEvidenceRefs []string
}

type IdleSelfImprovementPlanResultV0 struct {
	Requests     []IdleSelfImprovementRequestV0
	EvidenceRefs []string
	Collisions   []BacklogScanCollisionV0
	Message      string
}

type IdleSelfImprovementRequestFilterRequestV0 struct {
	BaseRequest  IdleSelfImprovementRequestV0
	Requests     []IdleSelfImprovementRequestV0
	Trigger      string
	EvidenceRefs []string
	Message      string
}

type IdleSelfImprovementRequestFilterResultV0 struct {
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
	BacklogScanRef     string
	BacklogScanEpoch   string
	BacklogScanDocs    []BacklogScanDocumentV0
	ReservationRefs    []string
	OccurredAt         string
	PriorityScore      int
}

type BacklogScanDocumentV0 struct {
	Path            string                    `json:"path"`
	StartLine       int                       `json:"start_line,omitempty"`
	SHA256          string                    `json:"sha256"`
	Missing         bool                      `json:"missing,omitempty"`
	SectionRef      string                    `json:"section_ref,omitempty"`
	ScanEntries     []BacklogScanEntryV0      `json:"scan_entries,omitempty"`
	ScanEntryDigest string                    `json:"scan_entry_digest,omitempty"`
	ScanEntryCount  int                       `json:"scan_entry_count,omitempty"`
	ScanEntryIssues []BacklogScanEntryIssueV0 `json:"scan_entry_issues,omitempty"`
}

type BacklogScanEntryV0 struct {
	Ref       string `json:"ref"`
	Date      string `json:"date,omitempty"`
	Ordinal   int    `json:"ordinal,omitempty"`
	Line      int    `json:"line,omitempty"`
	BlockHash string `json:"block_hash,omitempty"`
}

type BacklogScanEntryIssueV0 struct {
	Code         string `json:"code"`
	ScanEntryRef string `json:"scan_entry_ref,omitempty"`
	Line         int    `json:"line,omitempty"`
	Message      string `json:"message,omitempty"`
}

type BacklogScanCollisionV0 struct {
	Code         string   `json:"code"`
	RequestRef   string   `json:"request_ref,omitempty"`
	SectionRef   string   `json:"section_ref,omitempty"`
	TaskID       string   `json:"task_id,omitempty"`
	InstanceRefs []string `json:"instance_refs,omitempty"`
	Message      string   `json:"message,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
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

type SelfWatchdogObservationPortV0 interface {
	ObserveSelfWatchdogV0(
		context.Context,
		SelfWatchdogObservationRequestV0,
	) (SelfWatchdogObservationV0, error)
}

type SelfWatchdogObservationRequestV0 struct {
	State      StateV0
	Config     SelfWatchdogConfigV0
	ObservedAt time.Time
}

type ClockPortV0 interface {
	Now() time.Time
}

type SystemClockV0 struct{}

func (SystemClockV0) Now() time.Time {
	return time.Now().UTC()
}
