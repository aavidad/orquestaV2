package orquestaruntimecodexdelivery

import (
	"context"
	"fmt"
	"strings"
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

type codexProgressStateRecordV0 struct {
	Current           orquestaruntime.AgentProgressHeartbeatV0
	Signature         string
	ActionSignature   string
	TickCounter       int
	ProgressCounter   int
	RepeatedCount     int
	NoProgressCount   int
	ReportedSignature string
	ReportedStatus    orquestaruntime.AgentProgressStatusV0
	FirstObservedAt   time.Time
	LastActivityAt    time.Time
	LastAckAt         time.Time
	ObservedAt        time.Time
}

func (record codexProgressStateRecordV0) acceptsSampleV0(sample CodexProgressSampleV0) bool {
	if record.Signature == "" || record.Signature != sample.Signature {
		return true
	}
	if sample.MinUnchangedInterval <= 0 || record.ObservedAt.IsZero() {
		return true
	}
	return !sample.ObservedAt.Before(record.ObservedAt.Add(sample.MinUnchangedInterval))
}

func (record codexProgressStateRecordV0) nextHeartbeatV0(
	sample CodexProgressSampleV0,
) codexProgressStateRecordV0 {
	tick := record.TickCounter + 1
	progress := record.ProgressCounter
	noProgress := record.NoProgressCount
	repeated := record.RepeatedCount
	reported := record.ReportedSignature
	reportedStatus := record.ReportedStatus
	firstObservedAt := record.FirstObservedAt
	lastActivityAt := record.LastActivityAt
	if firstObservedAt.IsZero() {
		firstObservedAt = sample.ObservedAt
	}
	signatureChanged := record.Signature == "" || record.Signature != sample.Signature
	if signatureChanged {
		progress++
		noProgress = 0
		reported = ""
		reportedStatus = ""
		lastActivityAt = sample.ObservedAt
		if record.ActionSignature != "" &&
			sample.ActionSignature != "" &&
			record.ActionSignature == sample.ActionSignature {
			repeated++
		} else {
			repeated = 0
		}
	} else {
		noProgress++
	}
	current := orquestaruntime.AgentProgressHeartbeatV0{
		HeartbeatRef:        codexProgressHeartbeatRefV0(sample, tick),
		RunID:               sample.RunID,
		AgentRequestID:      sample.AgentRequestID,
		ProcessRef:          sample.ProcessRef,
		TickCounter:         tick,
		ProgressCounter:     progress,
		RepeatedActionCount: repeated,
		EvidenceRefs:        compactCodexDeliveryRefsV0(sample.EvidenceRefs),
	}
	return codexProgressStateRecordV0{
		Current:           current,
		Signature:         sample.Signature,
		ActionSignature:   sample.ActionSignature,
		TickCounter:       tick,
		ProgressCounter:   progress,
		RepeatedCount:     repeated,
		NoProgressCount:   noProgress,
		ReportedSignature: reported,
		ReportedStatus:    reportedStatus,
		FirstObservedAt:   firstObservedAt,
		LastActivityAt:    lastActivityAt,
		LastAckAt:         record.LastAckAt,
		ObservedAt:        sample.ObservedAt,
	}
}

func (record codexProgressStateRecordV0) observationStateV0(
	signature string,
	accepted bool,
) CodexProgressObservationStateV0 {
	return CodexProgressObservationStateV0{
		Current:           record.Current,
		Previous:          record.progressBaselineHeartbeatV0(),
		Signature:         signature,
		ReportedSignature: record.ReportedSignature,
		ReportedStatus:    record.ReportedStatus,
		FirstObservedAt:   record.FirstObservedAt,
		LastActivityAt:    record.LastActivityAt,
		LastAckAt:         record.LastAckAt,
		ObservedAt:        record.ObservedAt,
		SampleAccepted:    accepted,
	}
}

func (record codexProgressStateRecordV0) previousHeartbeatV0() *orquestaruntime.AgentProgressHeartbeatV0 {
	if record.Current.HeartbeatRef == "" {
		return nil
	}
	previous := record.Current
	return &previous
}

func (record codexProgressStateRecordV0) progressBaselineHeartbeatV0() *orquestaruntime.AgentProgressHeartbeatV0 {
	if record.Current.HeartbeatRef == "" || record.NoProgressCount <= 0 {
		return nil
	}
	previous := record.Current
	previous.TickCounter = record.Current.TickCounter - record.NoProgressCount
	return &previous
}

func normalizeCodexProgressSampleV0(sample CodexProgressSampleV0) CodexProgressSampleV0 {
	sample.RunID = strings.TrimSpace(sample.RunID)
	sample.AgentRequestID = strings.TrimSpace(sample.AgentRequestID)
	sample.ProcessRef = strings.TrimSpace(sample.ProcessRef)
	sample.Signature = strings.TrimSpace(sample.Signature)
	sample.ActionSignature = strings.TrimSpace(sample.ActionSignature)
	sample.EvidenceRefs = compactCodexDeliveryRefsV0(sample.EvidenceRefs)
	if sample.ObservedAt.IsZero() {
		sample.ObservedAt = time.Now().UTC()
	}
	if sample.MinUnchangedInterval < 0 {
		sample.MinUnchangedInterval = 0
	}
	return sample
}

func normalizeCodexProgressReportMarkV0(mark CodexProgressReportMarkV0) CodexProgressReportMarkV0 {
	mark.RunID = strings.TrimSpace(mark.RunID)
	mark.AgentRequestID = strings.TrimSpace(mark.AgentRequestID)
	mark.ProcessRef = strings.TrimSpace(mark.ProcessRef)
	mark.Signature = strings.TrimSpace(mark.Signature)
	return mark
}

func validateCodexProgressSampleV0(sample CodexProgressSampleV0) error {
	switch {
	case sample.RunID == "":
		return fmt.Errorf("codex_progress_sample: run_id requerido")
	case sample.AgentRequestID == "":
		return fmt.Errorf("codex_progress_sample: agent_request_id requerido")
	case sample.ProcessRef == "":
		return fmt.Errorf("codex_progress_sample: process_ref requerido")
	case sample.Signature == "":
		return fmt.Errorf("codex_progress_sample: signature requerida")
	default:
		return nil
	}
}

func validateCodexProgressReportMarkV0(mark CodexProgressReportMarkV0) error {
	if err := validateCodexProgressSampleV0(CodexProgressSampleV0{
		RunID:          mark.RunID,
		AgentRequestID: mark.AgentRequestID,
		ProcessRef:     mark.ProcessRef,
		Signature:      mark.Signature,
	}); err != nil {
		return err
	}
	switch mark.Status {
	case orquestaruntime.AgentProgressingV0,
		orquestaruntime.AgentStalledV0,
		orquestaruntime.AgentLoopDetectedV0,
		orquestaruntime.AgentStoppedV0:
		return nil
	default:
		return fmt.Errorf("codex_progress_report_mark: status requerido")
	}
}

func codexProgressSampleKeyV0(sample CodexProgressSampleV0) string {
	return sample.RunID + "\x00" + sample.AgentRequestID + "\x00" + sample.ProcessRef
}

func codexProgressReportMarkKeyV0(mark CodexProgressReportMarkV0) string {
	return mark.RunID + "\x00" + mark.AgentRequestID + "\x00" + mark.ProcessRef
}

func codexProgressHeartbeatRefV0(sample CodexProgressSampleV0, tick int) string {
	return fmt.Sprintf("heartbeat-ref-%s-%06d", sample.AgentRequestID, tick)
}
