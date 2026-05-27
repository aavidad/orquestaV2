package orquestaruntimecodexdelivery

import (
	"context"
	"sync"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type CodexProgressStateStorePortV0 interface {
	ObserveCodexProgressV0(
		context.Context,
		CodexProgressSampleV0,
	) (CodexProgressObservationStateV0, error)
	MarkCodexProgressReportedV0(context.Context, CodexProgressReportMarkV0) error
}

type CodexProgressSampleV0 struct {
	RunID                string
	AgentRequestID       string
	ProcessRef           string
	Signature            string
	ActionSignature      string
	EvidenceRefs         []string
	ObservedAt           time.Time
	MinUnchangedInterval time.Duration
}

type CodexProgressReportMarkV0 struct {
	RunID          string
	AgentRequestID string
	ProcessRef     string
	Signature      string
	Status         orquestaruntime.AgentProgressStatusV0
}

type CodexProgressObservationStateV0 struct {
	Current           orquestaruntime.AgentProgressHeartbeatV0
	Previous          *orquestaruntime.AgentProgressHeartbeatV0
	Signature         string
	ReportedSignature string
	ReportedStatus    orquestaruntime.AgentProgressStatusV0
	FirstObservedAt   time.Time
	LastActivityAt    time.Time
	LastAckAt         time.Time
	ObservedAt        time.Time
	SampleAccepted    bool
}

type InMemoryCodexProgressStateStoreV0 struct {
	mu      sync.Mutex
	records map[string]codexProgressStateRecordV0
}

func NewInMemoryCodexProgressStateStoreV0() *InMemoryCodexProgressStateStoreV0 {
	return &InMemoryCodexProgressStateStoreV0{
		records: map[string]codexProgressStateRecordV0{},
	}
}

func (store *InMemoryCodexProgressStateStoreV0) ObserveCodexProgressV0(
	ctx context.Context,
	sample CodexProgressSampleV0,
) (CodexProgressObservationStateV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return CodexProgressObservationStateV0{}, err
	}
	sample = normalizeCodexProgressSampleV0(sample)
	if err := validateCodexProgressSampleV0(sample); err != nil {
		return CodexProgressObservationStateV0{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureRecordsLockedV0()
	key := codexProgressSampleKeyV0(sample)
	record := store.records[key]
	if !record.acceptsSampleV0(sample) {
		return record.observationStateV0(sample.Signature, false), nil
	}
	next := record.nextHeartbeatV0(sample)
	store.records[key] = next
	return next.observationStateV0(sample.Signature, true), nil
}

func (store *InMemoryCodexProgressStateStoreV0) MarkCodexProgressReportedV0(
	ctx context.Context,
	mark CodexProgressReportMarkV0,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	mark = normalizeCodexProgressReportMarkV0(mark)
	if err := validateCodexProgressReportMarkV0(mark); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.ensureRecordsLockedV0()
	key := codexProgressReportMarkKeyV0(mark)
	record := store.records[key]
	record.ReportedSignature = mark.Signature
	record.ReportedStatus = mark.Status
	store.records[key] = record
	return nil
}

func (store *InMemoryCodexProgressStateStoreV0) ensureRecordsLockedV0() {
	if store.records == nil {
		store.records = map[string]codexProgressStateRecordV0{}
	}
}
