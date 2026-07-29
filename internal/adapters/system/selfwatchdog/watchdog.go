// Package selfwatchdog contains the operational policy that correlates process
// telemetry with durable progress and typed live causes. It never inspects
// domain content and can only admit cooperative shutdown through a scoped
// port.
package selfwatchdog

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

const (
	checkpointSchemaVersion = 1
	evidenceCode            = "self_watchdog_cpu_without_progress"
	nextActionCode          = "inspect_self_watchdog_evidence_before_restart"
)

type ErrorCode string

const (
	ErrorInvalidComposition ErrorCode = "self_watchdog_invalid_composition"
	ErrorTelemetryInvalid   ErrorCode = "self_watchdog_telemetry_invalid"
	ErrorProgressInvalid    ErrorCode = "self_watchdog_progress_invalid"
	ErrorCheckpointInvalid  ErrorCode = "self_watchdog_checkpoint_invalid"
	ErrorReceiptInvalid     ErrorCode = "self_watchdog_receipt_invalid"
)

type Error struct {
	Code  ErrorCode
	Cause error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func HasErrorCode(err error, code ErrorCode) bool {
	var watchdogError *Error
	return errors.As(err, &watchdogError) && watchdogError.Code == code
}

// Identity scopes both durable state and shutdown. OwnerRef is stable across a
// service restart; InstanceRef identifies exactly one process composition.
type Identity struct {
	OwnerRef     string
	InstanceRef  string
	FencingToken uint64
}

// TelemetrySample is normalized by the telemetry adapter. CPUPercent is a
// process-composition percentage in [0,100], not a host-wide policy decision.
type TelemetrySample struct {
	ObservedAt  time.Time
	CPUPercent  float64
	MemoryBytes uint64
	Uptime      time.Duration
}

// ProgressSnapshot is structural operational data. The four counters are live
// causes and therefore suppress shutdown without inspecting names, logs,
// prompts, artifacts, or any other content.
type ProgressSnapshot struct {
	ObservedAt            time.Time
	LastDurableProgressAt time.Time
	DurableRevision       string
	ActiveGoals           uint64
	ActiveTests           uint64
	ActiveFirecracker     uint64
	ActiveQueries         uint64
	ShutdownInProgress    bool
}

func (snapshot ProgressSnapshot) hasLiveCause() bool {
	return snapshot.ActiveGoals > 0 ||
		snapshot.ActiveTests > 0 ||
		snapshot.ActiveFirecracker > 0 ||
		snapshot.ActiveQueries > 0 ||
		snapshot.ShutdownInProgress
}

type TelemetryPort interface {
	Sample(context.Context) (TelemetrySample, error)
}

type ProgressPort interface {
	ObserveProgress(context.Context) (ProgressSnapshot, error)
}

type Evidence struct {
	Code                  string
	NextActionCode        string
	OwnerRef              string
	InstanceRef           string
	FencingToken          uint64
	IdempotencyKey        string
	HighCPUStartedAt      time.Time
	ObservedAt            time.Time
	CPUPercent            float64
	CPUHighPercent        float64
	SustainedFor          time.Duration
	NoProgressFor         time.Duration
	LastDurableProgressAt time.Time
	DurableRevision       string
	MemoryBytes           uint64
	Uptime                time.Duration
}

type EvidenceReceipt struct {
	OwnerRef       string
	InstanceRef    string
	FencingToken   uint64
	EvidenceRef    string
	IdempotencyKey string
}

type EvidencePort interface {
	PublishEvidence(context.Context, Evidence) (EvidenceReceipt, error)
}

type ShutdownMode string

const ShutdownCooperative ShutdownMode = "cooperative"

// ShutdownAdmissionRequest has no PID or process-name selector. The shutdown
// adapter is bound to OwnerRef, InstanceRef, and FencingToken and may admit
// shutdown only for that exact composition.
type ShutdownAdmissionRequest struct {
	OwnerRef       string
	InstanceRef    string
	FencingToken   uint64
	Mode           ShutdownMode
	ReasonCode     string
	EvidenceRef    string
	IdempotencyKey string
	RequestedAt    time.Time
}

type ShutdownAdmissionReceipt struct {
	OwnerRef       string
	InstanceRef    string
	FencingToken   uint64
	Mode           ShutdownMode
	IdempotencyKey string
}

type ShutdownPort interface {
	// AdmitOwnShutdown durably admits an idempotent intent and returns without
	// calling or awaiting Runtime.Shutdown. Completion is owned by the
	// lifecycle composition, outside the watchdog tick. An error guarantees
	// that admission did not occur; an adapter with an unknown result must
	// reconcile its own idempotency key before returning.
	AdmitOwnShutdown(context.Context, ShutdownAdmissionRequest) (ShutdownAdmissionReceipt, error)
}

type IncidentCheckpoint struct {
	IdempotencyKey       string
	HighCPUStartedAt     time.Time
	TriggeredAt          time.Time
	Evidence             Evidence
	EvidenceRef          string
	EvidencePublished    bool
	ShutdownIntentStored bool
}

type Checkpoint struct {
	SchemaVersion int
	InstanceRef   string
	FencingToken  uint64
	HighCPUSince  time.Time
	Incident      IncidentCheckpoint
}

type CheckpointSnapshot struct {
	Found      bool
	Revision   uint64
	Checkpoint Checkpoint
}

type CheckpointWrite struct {
	OwnerRef         string
	ExpectedRevision uint64
	Checkpoint       Checkpoint
}

type CheckpointReceipt struct {
	OwnerRef         string
	InstanceRef      string
	FencingToken     uint64
	PreviousRevision uint64
	Revision         uint64
}

type CheckpointPort interface {
	LoadCheckpoint(context.Context, string) (CheckpointSnapshot, error)
	// StoreCheckpoint applies expected-revision CAS and preserves the highest
	// fencing token accepted for an owner.
	StoreCheckpoint(context.Context, CheckpointWrite) (CheckpointReceipt, error)
}

type Ports struct {
	Telemetry TelemetryPort
	Progress  ProgressPort
	Evidence  EvidencePort
	Shutdown  ShutdownPort
	State     CheckpointPort
}

// Policy is the adapter-local projection of canonical configuration. It has
// no defaults, aliases, environment access, or registry behavior.
type Policy struct {
	ObservationInterval time.Duration
	CPUHighPercent      float64
	SustainedFor        time.Duration
	NoProgressFor       time.Duration
}

type Decision string

const (
	DecisionBelowThreshold          Decision = "below_cpu_threshold"
	DecisionLiveCause               Decision = "live_operational_cause"
	DecisionRecentDurableProgress   Decision = "recent_durable_progress"
	DecisionSustainedWindowPending  Decision = "sustained_window_pending"
	DecisionShutdownAdmitted        Decision = "cooperative_shutdown_admitted"
	DecisionShutdownAlreadyAdmitted Decision = "cooperative_shutdown_already_admitted"
)

type Outcome struct {
	Decision         Decision
	IncidentKey      string
	EvidenceRef      string
	ShutdownAdmitted bool
}

// Watchdog serializes ticks so one process instance cannot race itself. Cross
// process coordination remains the responsibility of the checkpoint CAS port.
type Watchdog struct {
	policy   Policy
	identity Identity
	ports    Ports

	mu                  sync.Mutex
	loaded              bool
	revision            uint64
	checkpoint          Checkpoint
	admittedIncidentKey string
}

func New(policy Policy, identity Identity, ports Ports) (*Watchdog, error) {
	if identity.OwnerRef == "" || identity.InstanceRef == "" || identity.FencingToken == 0 ||
		ports.Telemetry == nil || ports.Progress == nil || ports.Evidence == nil ||
		ports.Shutdown == nil || ports.State == nil ||
		policy.ObservationInterval <= 0 ||
		policy.CPUHighPercent <= 0 || policy.CPUHighPercent > 100 ||
		policy.SustainedFor < policy.ObservationInterval ||
		policy.NoProgressFor < policy.ObservationInterval {
		return nil, compositionError(errors.New("self_watchdog.dependency_missing"))
	}
	return &Watchdog{policy: policy, identity: identity, ports: ports}, nil
}

// Tick observes one atomic policy point. Port failures are fail-safe: no
// evidence or shutdown is inferred from missing telemetry/progress.
func (watchdog *Watchdog) Tick(ctx context.Context) (Outcome, error) {
	if watchdog == nil || ctx == nil {
		return Outcome{}, compositionError(errors.New("self_watchdog.context_required"))
	}
	if err := ctx.Err(); err != nil {
		return Outcome{}, err
	}
	watchdog.mu.Lock()
	defer watchdog.mu.Unlock()

	if err := watchdog.load(ctx); err != nil {
		return Outcome{}, err
	}
	if watchdog.checkpoint.Incident.IdempotencyKey != "" {
		if watchdog.admittedIncidentKey == watchdog.checkpoint.Incident.IdempotencyKey {
			return Outcome{
				Decision:    DecisionShutdownAlreadyAdmitted,
				IncidentKey: watchdog.checkpoint.Incident.IdempotencyKey,
				EvidenceRef: watchdog.checkpoint.Incident.EvidenceRef,
			}, nil
		}
		if watchdog.checkpoint.Incident.ShutdownIntentStored {
			return watchdog.completePersistedIncident(ctx)
		}
		return watchdog.resumeIncident(ctx)
	}

	sample, err := watchdog.ports.Telemetry.Sample(ctx)
	if err != nil {
		return Outcome{}, err
	}
	if err := validateTelemetry(sample); err != nil {
		return Outcome{}, err
	}
	sample = normalizeTelemetry(sample)
	progress, err := watchdog.ports.Progress.ObserveProgress(ctx)
	if err != nil {
		return Outcome{}, err
	}
	if err := validateProgress(sample, progress, watchdog.policy.ObservationInterval); err != nil {
		return Outcome{}, err
	}
	progress = normalizeProgress(progress)

	if progress.hasLiveCause() {
		if err := watchdog.clearHighCPU(ctx); err != nil {
			return Outcome{}, err
		}
		return Outcome{Decision: DecisionLiveCause}, nil
	}
	if sample.CPUPercent < watchdog.policy.CPUHighPercent {
		if err := watchdog.clearHighCPU(ctx); err != nil {
			return Outcome{}, err
		}
		return Outcome{Decision: DecisionBelowThreshold}, nil
	}
	if watchdog.checkpoint.HighCPUSince.IsZero() {
		next := watchdog.checkpoint
		next.HighCPUSince = sample.ObservedAt
		if err := watchdog.persist(ctx, next); err != nil {
			return Outcome{}, err
		}
	}
	if sample.ObservedAt.Before(watchdog.checkpoint.HighCPUSince) {
		return Outcome{}, telemetryError(errors.New("self_watchdog.telemetry_time_regressed"))
	}
	quietFor := sample.Uptime
	if !progress.LastDurableProgressAt.IsZero() {
		quietFor = sample.ObservedAt.Sub(progress.LastDurableProgressAt)
	}
	if quietFor < watchdog.policy.NoProgressFor {
		return Outcome{Decision: DecisionRecentDurableProgress}, nil
	}
	if sample.ObservedAt.Sub(watchdog.checkpoint.HighCPUSince) < watchdog.policy.SustainedFor {
		return Outcome{Decision: DecisionSustainedWindowPending}, nil
	}

	incidentKey := buildIncidentKey(watchdog.identity, watchdog.checkpoint.HighCPUSince)
	next := watchdog.checkpoint
	next.Incident = IncidentCheckpoint{
		IdempotencyKey:   incidentKey,
		HighCPUStartedAt: watchdog.checkpoint.HighCPUSince,
		TriggeredAt:      sample.ObservedAt,
		Evidence: Evidence{
			Code: evidenceCode, NextActionCode: nextActionCode,
			OwnerRef: watchdog.identity.OwnerRef, InstanceRef: watchdog.identity.InstanceRef,
			FencingToken:     watchdog.identity.FencingToken,
			IdempotencyKey:   incidentKey,
			HighCPUStartedAt: watchdog.checkpoint.HighCPUSince, ObservedAt: sample.ObservedAt,
			CPUPercent: sample.CPUPercent, CPUHighPercent: watchdog.policy.CPUHighPercent,
			SustainedFor: watchdog.policy.SustainedFor, NoProgressFor: watchdog.policy.NoProgressFor,
			LastDurableProgressAt: progress.LastDurableProgressAt,
			DurableRevision:       progress.DurableRevision,
			MemoryBytes:           sample.MemoryBytes, Uptime: sample.Uptime,
		},
	}
	if err := watchdog.persist(ctx, next); err != nil {
		return Outcome{}, err
	}
	return watchdog.completePersistedIncident(ctx)
}

func (watchdog *Watchdog) load(ctx context.Context) error {
	if watchdog.loaded {
		return nil
	}
	snapshot, err := watchdog.ports.State.LoadCheckpoint(ctx, watchdog.identity.OwnerRef)
	if err != nil {
		return err
	}
	if snapshot.Found {
		if err := validateCheckpoint(snapshot); err != nil {
			return err
		}
		watchdog.checkpoint = snapshot.Checkpoint
		watchdog.revision = snapshot.Revision
	}
	watchdog.loaded = true
	if watchdog.checkpoint.InstanceRef == watchdog.identity.InstanceRef {
		if watchdog.checkpoint.FencingToken != watchdog.identity.FencingToken {
			return checkpointError(errors.New("self_watchdog.instance_fence_mismatch"))
		}
		return watchdog.validateCheckpointBinding()
	}
	if watchdog.checkpoint.InstanceRef != "" &&
		watchdog.identity.FencingToken <= watchdog.checkpoint.FencingToken {
		return checkpointError(errors.New("self_watchdog.instance_fence_stale"))
	}
	// CPU sustained by a previous process instance is not evidence against the
	// new, higher-fenced one. Reset it before observing the restarted
	// composition.
	return watchdog.persist(ctx, Checkpoint{
		SchemaVersion: checkpointSchemaVersion,
		InstanceRef:   watchdog.identity.InstanceRef,
		FencingToken:  watchdog.identity.FencingToken,
	})
}

func (watchdog *Watchdog) clearHighCPU(ctx context.Context) error {
	if watchdog.checkpoint.HighCPUSince.IsZero() {
		return nil
	}
	next := watchdog.checkpoint
	next.HighCPUSince = time.Time{}
	return watchdog.persist(ctx, next)
}

func (watchdog *Watchdog) resumeIncident(ctx context.Context) (Outcome, error) {
	// Re-observe only to veto a stale shutdown intent if work or progress has
	// appeared since a crash. If it remains valid, the persisted evidence is
	// replayed byte-for-byte by semantic value under the original key.
	sample, err := watchdog.ports.Telemetry.Sample(ctx)
	if err != nil {
		return Outcome{}, err
	}
	if err := validateTelemetry(sample); err != nil {
		return Outcome{}, err
	}
	sample = normalizeTelemetry(sample)
	progress, err := watchdog.ports.Progress.ObserveProgress(ctx)
	if err != nil {
		return Outcome{}, err
	}
	if err := validateProgress(sample, progress, watchdog.policy.ObservationInterval); err != nil {
		return Outcome{}, err
	}
	progress = normalizeProgress(progress)
	if sample.ObservedAt.Before(watchdog.checkpoint.Incident.TriggeredAt) {
		return Outcome{}, telemetryError(errors.New("self_watchdog.incident_time_regressed"))
	}
	if progress.hasLiveCause() {
		if err := watchdog.cancelPendingIncident(ctx); err != nil {
			return Outcome{}, err
		}
		return Outcome{Decision: DecisionLiveCause}, nil
	}
	if sample.CPUPercent < watchdog.policy.CPUHighPercent {
		if err := watchdog.cancelPendingIncident(ctx); err != nil {
			return Outcome{}, err
		}
		return Outcome{Decision: DecisionBelowThreshold}, nil
	}
	quietFor := sample.Uptime
	if !progress.LastDurableProgressAt.IsZero() {
		quietFor = sample.ObservedAt.Sub(progress.LastDurableProgressAt)
	}
	if quietFor < watchdog.policy.NoProgressFor {
		if err := watchdog.cancelPendingIncident(ctx); err != nil {
			return Outcome{}, err
		}
		return Outcome{Decision: DecisionRecentDurableProgress}, nil
	}
	return watchdog.completePersistedIncident(ctx)
}

func (watchdog *Watchdog) cancelPendingIncident(ctx context.Context) error {
	next := watchdog.checkpoint
	next.HighCPUSince = time.Time{}
	next.Incident = IncidentCheckpoint{}
	return watchdog.persist(ctx, next)
}

func (watchdog *Watchdog) completePersistedIncident(ctx context.Context) (Outcome, error) {
	incident := watchdog.checkpoint.Incident
	if !incident.EvidencePublished {
		receipt, err := watchdog.ports.Evidence.PublishEvidence(ctx, incident.Evidence)
		if err != nil {
			return Outcome{}, err
		}
		if receipt.OwnerRef != watchdog.identity.OwnerRef ||
			receipt.InstanceRef != watchdog.identity.InstanceRef ||
			receipt.FencingToken != watchdog.identity.FencingToken ||
			receipt.IdempotencyKey != incident.IdempotencyKey ||
			receipt.EvidenceRef == "" {
			return Outcome{}, receiptError(errors.New("self_watchdog.evidence_receipt_mismatch"))
		}
		next := watchdog.checkpoint
		next.Incident.EvidenceRef = receipt.EvidenceRef
		next.Incident.EvidencePublished = true
		if err := watchdog.persist(ctx, next); err != nil {
			return Outcome{}, err
		}
		incident = watchdog.checkpoint.Incident
	}
	if !incident.ShutdownIntentStored {
		next := watchdog.checkpoint
		next.Incident.ShutdownIntentStored = true
		if err := watchdog.persist(ctx, next); err != nil {
			return Outcome{}, err
		}
		incident = watchdog.checkpoint.Incident
	}
	receipt, err := watchdog.ports.Shutdown.AdmitOwnShutdown(ctx, ShutdownAdmissionRequest{
		OwnerRef: watchdog.identity.OwnerRef, InstanceRef: watchdog.identity.InstanceRef,
		FencingToken: watchdog.identity.FencingToken,
		Mode:         ShutdownCooperative, ReasonCode: evidenceCode,
		EvidenceRef: incident.EvidenceRef, IdempotencyKey: incident.IdempotencyKey,
		RequestedAt: incident.TriggeredAt,
	})
	if err != nil {
		return Outcome{}, err
	}
	if receipt.OwnerRef != watchdog.identity.OwnerRef ||
		receipt.InstanceRef != watchdog.identity.InstanceRef ||
		receipt.FencingToken != watchdog.identity.FencingToken ||
		receipt.Mode != ShutdownCooperative ||
		receipt.IdempotencyKey != incident.IdempotencyKey {
		return Outcome{}, receiptError(errors.New("self_watchdog.shutdown_receipt_mismatch"))
	}
	watchdog.admittedIncidentKey = incident.IdempotencyKey
	return Outcome{
		Decision:         DecisionShutdownAdmitted,
		IncidentKey:      incident.IdempotencyKey,
		EvidenceRef:      incident.EvidenceRef,
		ShutdownAdmitted: true,
	}, nil
}

func (watchdog *Watchdog) persist(ctx context.Context, checkpoint Checkpoint) error {
	checkpoint.SchemaVersion = checkpointSchemaVersion
	checkpoint.InstanceRef = watchdog.identity.InstanceRef
	checkpoint.FencingToken = watchdog.identity.FencingToken
	receipt, err := watchdog.ports.State.StoreCheckpoint(ctx, CheckpointWrite{
		OwnerRef:         watchdog.identity.OwnerRef,
		ExpectedRevision: watchdog.revision,
		Checkpoint:       checkpoint,
	})
	if err != nil {
		watchdog.loaded = false
		return err
	}
	if receipt.OwnerRef != watchdog.identity.OwnerRef ||
		receipt.InstanceRef != watchdog.identity.InstanceRef ||
		receipt.FencingToken != watchdog.identity.FencingToken ||
		receipt.PreviousRevision != watchdog.revision ||
		receipt.Revision <= receipt.PreviousRevision {
		watchdog.loaded = false
		return receiptError(errors.New("self_watchdog.checkpoint_receipt_mismatch"))
	}
	watchdog.checkpoint = checkpoint
	watchdog.revision = receipt.Revision
	return nil
}

func validateTelemetry(sample TelemetrySample) error {
	if sample.ObservedAt.IsZero() || sample.CPUPercent < 0 || sample.CPUPercent > 100 || sample.Uptime < 0 {
		return telemetryError(errors.New("self_watchdog.telemetry_sample_invalid"))
	}
	return nil
}

func validateProgress(sample TelemetrySample, progress ProgressSnapshot, maximumSkew time.Duration) error {
	if progress.ObservedAt.IsZero() ||
		progress.ObservedAt.Before(sample.ObservedAt.Add(-maximumSkew)) ||
		progress.ObservedAt.After(sample.ObservedAt.Add(maximumSkew)) ||
		progress.LastDurableProgressAt.After(progress.ObservedAt) ||
		(progress.LastDurableProgressAt.IsZero() != (progress.DurableRevision == "")) {
		return progressError(errors.New("self_watchdog.progress_snapshot_invalid"))
	}
	return nil
}

func validateCheckpoint(snapshot CheckpointSnapshot) error {
	checkpoint := snapshot.Checkpoint
	if snapshot.Revision == 0 ||
		checkpoint.SchemaVersion != checkpointSchemaVersion ||
		checkpoint.InstanceRef == "" ||
		checkpoint.FencingToken == 0 ||
		(checkpoint.Incident.IdempotencyKey == "" &&
			(checkpoint.Incident.EvidencePublished || checkpoint.Incident.ShutdownIntentStored ||
				checkpoint.Incident.EvidenceRef != "")) ||
		(checkpoint.Incident.EvidencePublished != (checkpoint.Incident.EvidenceRef != "")) ||
		(checkpoint.Incident.IdempotencyKey != "" &&
			(checkpoint.Incident.HighCPUStartedAt.IsZero() ||
				checkpoint.Incident.TriggeredAt.IsZero() ||
				checkpoint.Incident.Evidence.IdempotencyKey != checkpoint.Incident.IdempotencyKey ||
				checkpoint.Incident.Evidence.OwnerRef == "" ||
				checkpoint.Incident.Evidence.InstanceRef != checkpoint.InstanceRef)) ||
		(checkpoint.Incident.EvidencePublished && checkpoint.Incident.EvidenceRef == "") ||
		(checkpoint.Incident.ShutdownIntentStored && !checkpoint.Incident.EvidencePublished) {
		return checkpointError(errors.New("self_watchdog.checkpoint_snapshot_invalid"))
	}
	return nil
}

func (watchdog *Watchdog) validateCheckpointBinding() error {
	incident := watchdog.checkpoint.Incident
	if incident.IdempotencyKey == "" {
		return nil
	}
	evidence := incident.Evidence
	if evidence.Code != evidenceCode ||
		evidence.NextActionCode != nextActionCode ||
		evidence.OwnerRef != watchdog.identity.OwnerRef ||
		evidence.InstanceRef != watchdog.identity.InstanceRef ||
		evidence.FencingToken != watchdog.identity.FencingToken ||
		evidence.IdempotencyKey != incident.IdempotencyKey ||
		incident.IdempotencyKey != buildIncidentKey(watchdog.identity, incident.HighCPUStartedAt) ||
		!incident.HighCPUStartedAt.Equal(watchdog.checkpoint.HighCPUSince) ||
		!evidence.HighCPUStartedAt.Equal(incident.HighCPUStartedAt) ||
		!evidence.ObservedAt.Equal(incident.TriggeredAt) ||
		incident.TriggeredAt.Before(incident.HighCPUStartedAt.Add(watchdog.policy.SustainedFor)) ||
		evidence.CPUHighPercent != watchdog.policy.CPUHighPercent ||
		evidence.SustainedFor != watchdog.policy.SustainedFor ||
		evidence.NoProgressFor != watchdog.policy.NoProgressFor ||
		evidence.CPUPercent > 100 ||
		evidence.Uptime < 0 ||
		evidence.CPUPercent < evidence.CPUHighPercent ||
		(evidence.LastDurableProgressAt.IsZero() != (evidence.DurableRevision == "")) ||
		(!evidence.LastDurableProgressAt.IsZero() &&
			evidence.ObservedAt.Sub(evidence.LastDurableProgressAt) < evidence.NoProgressFor) ||
		(evidence.LastDurableProgressAt.IsZero() && evidence.Uptime < evidence.NoProgressFor) {
		return checkpointError(errors.New("self_watchdog.checkpoint_binding_invalid"))
	}
	return nil
}

func buildIncidentKey(identity Identity, highCPUStartedAt time.Time) string {
	digest := sha256.New()
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(identity.OwnerRef)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write([]byte(identity.OwnerRef))
	binary.BigEndian.PutUint64(length[:], uint64(len(identity.InstanceRef)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write([]byte(identity.InstanceRef))
	var encoded [16]byte
	binary.BigEndian.PutUint64(encoded[:8], identity.FencingToken)
	binary.BigEndian.PutUint64(encoded[8:], uint64(highCPUStartedAt.UnixNano()))
	_, _ = digest.Write(encoded[:])
	return "self-watchdog:sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func normalizeTelemetry(sample TelemetrySample) TelemetrySample {
	sample.ObservedAt = sample.ObservedAt.UTC()
	return sample
}

func normalizeProgress(progress ProgressSnapshot) ProgressSnapshot {
	progress.ObservedAt = progress.ObservedAt.UTC()
	if !progress.LastDurableProgressAt.IsZero() {
		progress.LastDurableProgressAt = progress.LastDurableProgressAt.UTC()
	}
	return progress
}

func compositionError(cause error) error {
	return &Error{Code: ErrorInvalidComposition, Cause: cause}
}

func telemetryError(cause error) error {
	return &Error{Code: ErrorTelemetryInvalid, Cause: cause}
}

func progressError(cause error) error {
	return &Error{Code: ErrorProgressInvalid, Cause: cause}
}

func checkpointError(cause error) error {
	return &Error{Code: ErrorCheckpointInvalid, Cause: cause}
}

func receiptError(cause error) error {
	return &Error{Code: ErrorReceiptInvalid, Cause: cause}
}
