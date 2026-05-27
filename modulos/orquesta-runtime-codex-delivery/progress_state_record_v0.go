package orquestaruntimecodexdelivery

import (
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

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
