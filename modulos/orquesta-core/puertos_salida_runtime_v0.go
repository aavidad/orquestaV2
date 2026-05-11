package orquestacore

import "context"

const RuntimeLaunchRequestContractVersionV0 = "runtime_launch_request.v0"

type RuntimeLauncherPortV0 interface {
	LanzarRuntimeV0(context.Context, RuntimeLaunchRequestCoreV0) (RuntimeLaunchAcceptedCoreV0, error)
}

type RuntimeLaunchRequestCoreV0 struct {
	SchemaVersion    string                     `json:"schema_version"`
	RequestID        string                     `json:"request_id"`
	CorrelationID    string                     `json:"correlation_id"`
	IdempotencyKey   string                     `json:"idempotency_key"`
	FunctionContract FunctionContractV0         `json:"function_contract"`
	CapacityDecision CapacityDecisionEvidenceV0 `json:"capacity_decision"`
	Binding          RuntimeBindingRefsV0       `json:"binding"`
	Safety           RuntimeSafetyPolicyV0      `json:"safety"`
}

type RuntimeLaunchAcceptedCoreV0 struct {
	LaunchID         string   `json:"launch_id"`
	CorrelationID    string   `json:"correlation_id"`
	RuntimeHandleRef string   `json:"runtime_handle_ref"`
	MailboxRef       string   `json:"mailbox_ref"`
	AckRef           string   `json:"ack_ref"`
	ReadinessRef     string   `json:"readiness_ref"`
	Warnings         []string `json:"warnings"`
}

type RuntimeLaunchErrorV0 struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	Field         string   `json:"field,omitempty"`
	Retryable     bool     `json:"retryable"`
	Evidence      []string `json:"evidence,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
}

func (err RuntimeLaunchErrorV0) Error() string {
	return err.Code
}

type CapacityDecisionEvidenceV0 struct {
	DecisionID      string `json:"decision_id"`
	NivelCapacidad  string `json:"nivel_capacidad"`
	ReasoningEffort string `json:"reasoning_effort"`
	PoolRef         string `json:"pool_ref"`
	ModelRef        string `json:"model_ref"`
	HomeRef         string `json:"home_ref"`
	CredentialRef   string `json:"credential_ref"`
	QuotaRef        string `json:"quota_ref,omitempty"`
}

type RuntimeBindingRefsV0 struct {
	AgentRef     string `json:"agent_ref,omitempty"`
	RuntimeRef   string `json:"runtime_ref"`
	ConnectorRef string `json:"connector_ref"`
	MailboxRef   string `json:"mailbox_ref"`
	AckRef       string `json:"ack_ref"`
	ReadinessRef string `json:"readiness_ref"`
}

type RuntimeSafetyPolicyV0 struct {
	WriteSetClosed  bool     `json:"write_set_closed"`
	NoSecrets       bool     `json:"no_secrets"`
	NoTranscripts   bool     `json:"no_transcripts"`
	AllowedWriteSet []string `json:"allowed_write_set"`
}
