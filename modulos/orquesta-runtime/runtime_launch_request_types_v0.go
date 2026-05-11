package orquestaruntime

import (
	"fmt"

	orquestacontext "orquesta/modulos/orquesta-context"
)

const (
	RuntimeLaunchRequestSchemaVersionV0 = "runtime_launch_request.v0"
	RuntimeLaunchModeNewSessionV0       = "new_session"
	RuntimeLaunchSourceModuleCoreV0     = "orquesta-core"

	FunctionContractSourceV0      = "FunctionContractV0"
	FunctionContractVersionV0     = "v0"
	FunctionContractStateActiveV0 = "activa"

	CapacityDecisionVersionV0 = "v0"

	DeliveryMailboxProtocolV0 = "orquesta_mailbox_ref.v0"

	SafetyPolicyReferencesOnlyV0 = "references_only"
	SafetyPolicyOpaqueRefsOnlyV0 = "opaque_refs_only"
	SafetyPolicyWriteSetClosedV0 = "closed"
)

type RuntimeLaunchErrorCodeV0 string

const (
	RuntimeLaunchRequestInvalidaV0 RuntimeLaunchErrorCodeV0 = "runtime_launch_request_invalida"
	FunctionContractRequeridoV0    RuntimeLaunchErrorCodeV0 = "function_contract_requerido"
	FunctionContractNoActivaV0     RuntimeLaunchErrorCodeV0 = "function_contract_no_activa"
	WriteSetRequeridoV0            RuntimeLaunchErrorCodeV0 = "write_set_requerido"
	WriteSetInvalidoV0             RuntimeLaunchErrorCodeV0 = "write_set_invalido"
	CapacityDecisionRequeridaV0    RuntimeLaunchErrorCodeV0 = "capacity_decision_requerida"
	CapacityDecisionNoSoportadaV0  RuntimeLaunchErrorCodeV0 = "capacity_decision_no_soportada"
	PoolRefRequeridoV0             RuntimeLaunchErrorCodeV0 = "pool_ref_requerido"
	ModelRefRequeridoV0            RuntimeLaunchErrorCodeV0 = "model_ref_requerido"
	HomeRefRequeridoV0             RuntimeLaunchErrorCodeV0 = "home_ref_requerido"
	CredentialRefRequeridoV0       RuntimeLaunchErrorCodeV0 = "credential_ref_requerido"
	ReferenciaNoOpacaV0            RuntimeLaunchErrorCodeV0 = "referencia_no_opaca"
	SecretoDetectadoV0             RuntimeLaunchErrorCodeV0 = "secreto_detectado"
	RutaHomeRealDetectadaV0        RuntimeLaunchErrorCodeV0 = "ruta_home_real_detectada"
	MailboxRefRequeridoV0          RuntimeLaunchErrorCodeV0 = "mailbox_ref_requerido"
	AckRefRequeridoV0              RuntimeLaunchErrorCodeV0 = "ack_ref_requerido"
	ReadinessRefRequeridoV0        RuntimeLaunchErrorCodeV0 = "readiness_ref_requerido"
	ContextBundleRequeridoV0       RuntimeLaunchErrorCodeV0 = "context_bundle_requerido"
	ContextBundleInvalidoV0        RuntimeLaunchErrorCodeV0 = "context_bundle_invalido"
	IdempotencyKeyRequeridaV0      RuntimeLaunchErrorCodeV0 = "idempotency_key_requerida"
	RuntimeNoDisponibleV0          RuntimeLaunchErrorCodeV0 = "runtime_no_disponible"
)

type RuntimeLaunchRequestV0 struct {
	SchemaVersion    string                           `json:"schema_version"`
	RequestID        string                           `json:"request_id"`
	CorrelationID    string                           `json:"correlation_id"`
	IdempotencyKey   string                           `json:"idempotency_key"`
	RequestedAt      string                           `json:"requested_at"`
	Source           *RuntimeLaunchSourceV0           `json:"source"`
	Locale           string                           `json:"locale"`
	LaunchMode       string                           `json:"launch_mode"`
	Task             *RuntimeLaunchTaskV0             `json:"task"`
	FunctionContract *RuntimeFunctionContractV0       `json:"function_contract"`
	CapacityDecision *RuntimeCapacityDecisionV0       `json:"capacity_decision"`
	RuntimeBinding   *RuntimeBindingV0                `json:"runtime_binding"`
	EvidenceRefs     *RuntimeEvidenceRefsV0           `json:"evidence_refs"`
	ContextBundle    *orquestacontext.ContextBundleV0 `json:"context_bundle"`
	Delivery         *RuntimeDeliveryV0               `json:"delivery"`
	Safety           *RuntimeSafetyV0                 `json:"safety"`
}

type RuntimeLaunchSourceV0 struct {
	Module     string `json:"module"`
	AdapterRef string `json:"adapter_ref"`
}

type RuntimeLaunchTaskV0 struct {
	TaskRef    string `json:"task_ref"`
	ProjectRef string `json:"project_ref,omitempty"`
	PhaseRef   string `json:"phase_ref,omitempty"`
	Priority   string `json:"priority"`
}

type RuntimeFunctionContractV0 struct {
	SourceContract    string   `json:"source_contract"`
	ContractRef       string   `json:"contract_ref"`
	ContractVersion   string   `json:"contract_version"`
	State             string   `json:"state"`
	Titulo            string   `json:"titulo"`
	Objetivo          string   `json:"objetivo"`
	ArchivoObjetivo   string   `json:"archivo_objetivo"`
	SimboloObjetivo   string   `json:"simbolo_objetivo"`
	WriteSet          []string `json:"write_set"`
	TestsObligatorios []string `json:"tests_obligatorios"`
	CriterioCierre    []string `json:"criterio_cierre"`
}

type RuntimeCapacityDecisionV0 struct {
	DecisionRef     string `json:"decision_ref"`
	ContractVersion string `json:"contract_version"`
	NivelCapacidad  string `json:"nivel_capacidad"`
	ReasoningEffort string `json:"reasoning_effort"`
	PoolRef         string `json:"pool_ref"`
	ModelRef        string `json:"model_ref"`
	QuotaRef        string `json:"quota_ref"`
}

type RuntimeBindingV0 struct {
	LogicalAgentRef string `json:"logical_agent_ref"`
	RuntimeKind     string `json:"runtime_kind"`
	ConnectorRef    string `json:"connector_ref"`
	ProviderRef     string `json:"provider_ref"`
	ModelRef        string `json:"model_ref"`
	HomeRef         string `json:"home_ref"`
	CredentialKind  string `json:"credential_kind"`
	CredentialRef   string `json:"credential_ref"`
}

type RuntimeEvidenceRefsV0 struct {
	MailboxRef    string `json:"mailbox_ref"`
	AckRef        string `json:"ack_ref"`
	ReadinessRef  string `json:"readiness_ref"`
	CheckpointRef string `json:"checkpoint_ref,omitempty"`
}

type RuntimeDeliveryV0 struct {
	MailboxProtocol         string `json:"mailbox_protocol"`
	AckRequired             bool   `json:"ack_required"`
	ReadinessTimeoutSeconds int    `json:"readiness_timeout_seconds"`
	MaxStartupSeconds       int    `json:"max_startup_seconds"`
}

type RuntimeSafetyV0 struct {
	SecretsPolicy   string `json:"secrets_policy"`
	HomePathsPolicy string `json:"home_paths_policy"`
	ProviderPolicy  string `json:"provider_policy"`
	WriteSetPolicy  string `json:"write_set_policy"`
}

type RuntimeLaunchErrorV0 struct {
	Code          RuntimeLaunchErrorCodeV0 `json:"code"`
	MessageKey    string                   `json:"message_key"`
	Field         string                   `json:"field,omitempty"`
	Retryable     bool                     `json:"retryable"`
	CorrelationID string                   `json:"correlation_id,omitempty"`
	Evidence      []string                 `json:"evidence,omitempty"`
}

func (e RuntimeLaunchErrorV0) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Field)
}
