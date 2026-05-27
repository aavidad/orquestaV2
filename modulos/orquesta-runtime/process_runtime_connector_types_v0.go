package orquestaruntime

import (
	"fmt"
)

const ProcessRuntimeConnectorVersionV0 = "process_runtime_connector.v0"

type ProcessRuntimeStatusV0 string

const (
	ProcessRuntimeRunningV0  ProcessRuntimeStatusV0 = "running"
	ProcessRuntimeStoppingV0 ProcessRuntimeStatusV0 = "stopping"
	ProcessRuntimeStoppedV0  ProcessRuntimeStatusV0 = "stopped"
)

type ProcessRuntimeErrorCodeV0 string

const (
	ProcessRuntimeConfigInvalidaV0 ProcessRuntimeErrorCodeV0 = "process_runtime_config_invalida"
	ProcessRuntimeShellProhibidaV0 ProcessRuntimeErrorCodeV0 = "process_runtime_shell_prohibida"
	ProcessRuntimeEnvProhibidoV0   ProcessRuntimeErrorCodeV0 = "process_runtime_env_prohibido"
	ProcessRuntimeRefInvalidaV0    ProcessRuntimeErrorCodeV0 = "process_runtime_ref_invalida"
	ProcessRuntimeNoEncontradoV0   ProcessRuntimeErrorCodeV0 = "process_runtime_no_encontrado"
	ProcessRuntimeLaunchFallidoV0  ProcessRuntimeErrorCodeV0 = "process_runtime_launch_fallido"
	ProcessRuntimeStopFallidoV0    ProcessRuntimeErrorCodeV0 = "process_runtime_stop_fallido"
	ProcessRuntimeKillFallidoV0    ProcessRuntimeErrorCodeV0 = "process_runtime_kill_fallido"
	ProcessRuntimeContextDoneV0    ProcessRuntimeErrorCodeV0 = "process_runtime_context_done"
)

type ProcessRuntimeStopReasonCodeV0 string

const (
	ProcessRuntimeStopCooperativeSignalSentV0 ProcessRuntimeStopReasonCodeV0 = "cooperative_signal_sent"
	ProcessRuntimeStopSignalNotSupportedV0    ProcessRuntimeStopReasonCodeV0 = "signal_not_supported"
	ProcessRuntimeStopAlreadyStoppedV0        ProcessRuntimeStopReasonCodeV0 = "process_already_stopped"
	ProcessRuntimeStopGraceTimeoutV0          ProcessRuntimeStopReasonCodeV0 = "grace_timeout"
	ProcessRuntimeStopKillFailedV0            ProcessRuntimeStopReasonCodeV0 = "kill_failed"
)

type ProcessRuntimeLaunchRequestV0 struct {
	CommandPath   string                         `json:"-"`
	Args          []string                       `json:"-"`
	Env           []string                       `json:"-"`
	WorkingDir    string                         `json:"-"`
	LaunchReceipt *ProcessRuntimeLaunchReceiptV0 `json:"-"`
}

type ProcessRuntimeSnapshotV0 struct {
	SchemaVersion     string                         `json:"schema_version"`
	ProcessRef        string                         `json:"process_ref"`
	SessionRef        string                         `json:"session_ref,omitempty"`
	LaunchRef         string                         `json:"launch_ref,omitempty"`
	PID               int                            `json:"-"`
	StopRef           string                         `json:"stop_ref,omitempty"`
	Status            ProcessRuntimeStatusV0         `json:"status"`
	LaunchReceipt     *ProcessRuntimeLaunchReceiptV0 `json:"launch_receipt,omitempty"`
	LaunchReceiptRefs []string                       `json:"launch_receipt_refs,omitempty"`
	StopReasonCode    ProcessRuntimeStopReasonCodeV0 `json:"stop_reason_code,omitempty"`
	StopGraceDeadline string                         `json:"stop_grace_deadline,omitempty"`
}

type ProcessRuntimeErrorV0 struct {
	Code       ProcessRuntimeErrorCodeV0 `json:"code"`
	MessageKey string                    `json:"message_key"`
	Field      string                    `json:"field,omitempty"`
	Retryable  bool                      `json:"retryable"`
	Evidence   []string                  `json:"evidence,omitempty"`
}

func (e ProcessRuntimeErrorV0) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Field)
}

func ValidateProcessRuntimeLaunchRequestV0(req ProcessRuntimeLaunchRequestV0) error {
	return validateProcessRuntimeLaunchRequestV0(req)
}

func validateProcessRuntimeRefV0(processRef string) error {
	if processRef == "" || !opaqueRefPatternV0.MatchString(processRef) {
		return processRuntimeErrorV0(ProcessRuntimeRefInvalidaV0, "process_ref")
	}
	return nil
}

func processRuntimeErrorV0(code ProcessRuntimeErrorCodeV0, field string) ProcessRuntimeErrorV0 {
	return ProcessRuntimeErrorV0{
		Code:       code,
		MessageKey: "orquesta.runtime.process." + string(code),
		Field:      field,
		Retryable:  code == ProcessRuntimeLaunchFallidoV0 || code == ProcessRuntimeStopFallidoV0 || code == ProcessRuntimeKillFallidoV0,
	}
}
