package orquestaruncontrol

import "context"

const RunControlSchemaVersionV0 = "run_control.v0"

type RunControlStatusV0 string

const (
	RunControlStatusRunningV0         RunControlStatusV0 = "running"
	RunControlStatusPausedV0          RunControlStatusV0 = "paused"
	RunControlStatusStopRequestedV0   RunControlStatusV0 = "stop_requested"
	RunControlStatusStoppedV0         RunControlStatusV0 = "stopped"
	RunControlStatusCancelRequestedV0 RunControlStatusV0 = "cancel_requested"
	RunControlStatusCanceledV0        RunControlStatusV0 = "canceled"
)

type RunControlStateV0 struct {
	RunRef             string             `json:"run_ref"`
	Status             RunControlStatusV0 `json:"status"`
	CheckpointRecorded bool               `json:"checkpoint_recorded"`
	Forced             bool               `json:"forced,omitempty"`
	EvidenceRefs       []string           `json:"evidence_refs,omitempty"`
	Meta               RunControlMetaV0   `json:"meta,omitempty"`
}

type RunControlMetaV0 struct {
	RequestedBy    string `json:"requested_by,omitempty"`
	Reason         string `json:"reason,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type RunControlReadRequestV0 struct {
	RunRef string `json:"run_ref"`
}

type RunControlReaderPortV0 interface {
	ReadRunControlStateV0(context.Context, RunControlReadRequestV0) (RunControlStateV0, error)
}

type RunControlWriterPortV0 interface {
	PauseRunV0(context.Context, PauseRunCommandV0) (RunControlStateV0, error)
	ResumeRunV0(context.Context, ResumeRunCommandV0) (RunControlStateV0, error)
	StopRunV0(context.Context, StopRunCommandV0) (RunControlStateV0, error)
	CancelRunV0(context.Context, CancelRunCommandV0) (RunControlStateV0, error)
}

type RunControlPortV0 interface {
	RunControlReaderPortV0
	RunControlWriterPortV0
}
