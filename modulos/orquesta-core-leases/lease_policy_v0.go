package orquestacoreleases

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

type AgentLeaseTimeoutActionV0 string

const (
	AgentLeaseTimeoutRetryV0       AgentLeaseTimeoutActionV0 = "retry"
	AgentLeaseTimeoutStopAgentV0   AgentLeaseTimeoutActionV0 = "stop_agent"
	AgentLeaseTimeoutAskDirectorV0 AgentLeaseTimeoutActionV0 = "ask_director"
	AgentLeaseTimeoutReplanTaskV0  AgentLeaseTimeoutActionV0 = "replan_task"
	AgentLeaseTimeoutAlertOnlyV0   AgentLeaseTimeoutActionV0 = "alert_only"
)

type AgentHeartbeatStatusV0 string

const (
	AgentHeartbeatAliveV0       AgentHeartbeatStatusV0 = "alive"
	AgentHeartbeatProgressingV0 AgentHeartbeatStatusV0 = "progressing"
	AgentHeartbeatStalledV0     AgentHeartbeatStatusV0 = "stalled"
	AgentHeartbeatStoppedV0     AgentHeartbeatStatusV0 = "stopped"
	AgentHeartbeatFailedV0      AgentHeartbeatStatusV0 = "failed"
)

type AgentLeaseIssueCodeV0 string

const (
	ErrAgentLeaseJSONInvalidoV0       AgentLeaseIssueCodeV0 = "agent_lease_json_invalido"
	ErrAgentLeaseInvalidoV0           AgentLeaseIssueCodeV0 = "agent_lease_invalido"
	ErrAgentLeaseReferenciaNoOpacaV0  AgentLeaseIssueCodeV0 = "referencia_no_opaca"
	ErrAgentLeaseDetalleProhibidoV0   AgentLeaseIssueCodeV0 = "detalle_prohibido"
	ErrAgentLeaseTiempoInvalidoV0     AgentLeaseIssueCodeV0 = "tiempo_invalido"
	ErrAgentLeaseAccionInvalidaV0     AgentLeaseIssueCodeV0 = "timeout_action_invalida"
	ErrAgentHeartbeatStatusInvalidoV0 AgentLeaseIssueCodeV0 = "heartbeat_status_invalido"
	ErrAgentHeartbeatObservedAtV0     AgentLeaseIssueCodeV0 = "heartbeat_observed_at_invalido"
	ErrAgentLeaseObservedAtV0         AgentLeaseIssueCodeV0 = "lease_observed_at_invalido"
	ErrAgentLeaseDecisionInvalidaV0   AgentLeaseIssueCodeV0 = "assessment_decision_invalida"
)

type AgentLeasePolicyV0 struct {
	LeasePolicyRef          string                    `json:"lease_policy_ref"`
	LaunchTimeoutSeconds    int                       `json:"launch_timeout_seconds"`
	HeartbeatTimeoutSeconds int                       `json:"heartbeat_timeout_seconds"`
	TotalTimeoutSeconds     int                       `json:"total_timeout_seconds"`
	TimeoutAction           AgentLeaseTimeoutActionV0 `json:"timeout_action"`
	MaxRetries              int                       `json:"max_retries"`
	EvidenceRefs            []string                  `json:"evidence_refs,omitempty"`
}

type AgentHeartbeatReportV0 struct {
	HeartbeatRef      string                 `json:"heartbeat_ref"`
	RunRef            string                 `json:"run_ref"`
	AgentRequestID    string                 `json:"agent_request_id"`
	LeaseRef          string                 `json:"lease_ref"`
	ObservedAt        string                 `json:"observed_at"`
	Status            AgentHeartbeatStatusV0 `json:"status"`
	ProgressReportRef string                 `json:"progress_report_ref,omitempty"`
	EvidenceRefs      []string               `json:"evidence_refs,omitempty"`
}

type AgentLeaseIssueV0 struct {
	Code  AgentLeaseIssueCodeV0 `json:"code"`
	Field string                `json:"field,omitempty"`
}

type AgentLeaseValidationErrorV0 struct {
	Issues []AgentLeaseIssueV0 `json:"issues"`
}

func (err AgentLeaseValidationErrorV0) Error() string {
	if len(err.Issues) == 0 {
		return string(ErrAgentLeaseInvalidoV0)
	}
	return string(err.Issues[0].Code)
}

func DecodeAgentLeasePolicyV0(data []byte) (AgentLeasePolicyV0, error) {
	if issues := detectForbiddenAgentLeaseJSONKeysV0(data); len(issues) > 0 {
		return AgentLeasePolicyV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	var policy AgentLeasePolicyV0
	if err := decodeStrictAgentLeaseJSONV0(data, &policy); err != nil {
		return AgentLeasePolicyV0{}, err
	}
	if issues := ValidateAgentLeasePolicyV0(policy); len(issues) > 0 {
		return AgentLeasePolicyV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	return policy, nil
}

func DecodeAgentHeartbeatReportV0(data []byte) (AgentHeartbeatReportV0, error) {
	if issues := detectForbiddenAgentLeaseJSONKeysV0(data); len(issues) > 0 {
		return AgentHeartbeatReportV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	var report AgentHeartbeatReportV0
	if err := decodeStrictAgentLeaseJSONV0(data, &report); err != nil {
		return AgentHeartbeatReportV0{}, err
	}
	if issues := ValidateAgentHeartbeatReportV0(report); len(issues) > 0 {
		return AgentHeartbeatReportV0{}, AgentLeaseValidationErrorV0{Issues: issues}
	}
	return report, nil
}

func decodeStrictAgentLeaseJSONV0(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return AgentLeaseValidationErrorV0{
			Issues: []AgentLeaseIssueV0{{Code: ErrAgentLeaseJSONInvalidoV0}},
		}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return AgentLeaseValidationErrorV0{
			Issues: []AgentLeaseIssueV0{{Code: ErrAgentLeaseJSONInvalidoV0}},
		}
	}
	return nil
}

func (policy AgentLeasePolicyV0) Validate() []AgentLeaseIssueV0 {
	return ValidateAgentLeasePolicyV0(policy)
}

func (policy AgentLeasePolicyV0) Valid() bool {
	return len(ValidateAgentLeasePolicyV0(policy)) == 0
}

func (report AgentHeartbeatReportV0) Validate() []AgentLeaseIssueV0 {
	return ValidateAgentHeartbeatReportV0(report)
}

func (report AgentHeartbeatReportV0) Valid() bool {
	return len(ValidateAgentHeartbeatReportV0(report)) == 0
}
