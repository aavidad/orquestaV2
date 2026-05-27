package orquestaoperatormcp

const (
	OperatorMCPSchemaVersionV0           = "operator_mcp.v0"
	OperatorMCPCapabilitiesURIV0         = "orquesta://operator/capabilities/v0"
	OperatorMCPStatusToolNameV0          = "orquesta.operator.status.query.v0"
	OperatorMCPBurstToolNameV0           = "orquesta.operator.supervised_burst.v0"
	OperatorMCPOutboxToolNameV0          = "orquesta.operator.outbox.pending.v0"
	OperatorMCPDirectedQueryToolV0       = "orquesta.operator.directed_query.v0"
	OperatorMCPMaxBurstStepsLimitV0      = 20
	OperatorMCPMaxOutboxLimitV0          = 50
	OperatorMCPMaxQuestionRunesV0        = 1000
	ErrOperatorMCPRequiredFieldV0        = i18nPublicOperatorRequiredFieldV0
	ErrOperatorMCPOpaqueRefV0            = i18nPublicOperatorOpaqueRefV0
	ErrOperatorMCPBudgetInvalidV0        = i18nPublicOperatorBudgetInvalidV0
	ErrOperatorMCPLimitInvalidV0         = i18nPublicOperatorLimitInvalidV0
	ErrOperatorMCPQuestionInvalidV0      = i18nPublicOperatorQuestionV0
	ErrOperatorMCPSectionInvalidV0       = i18nPublicOperatorSectionV0
	ErrOperatorMCPPortUnavailableV0      = i18nPublicOperatorPortV0
	ErrOperatorMCPPortErrorV0            = i18nPublicOperatorPortErrorV0
	ErrOperatorMCPConnectorUnavailableV0 = i18nPublicOperatorConnectorV0
	ErrOperatorMCPTimeoutV0              = i18nPublicOperatorTimeoutV0
	ErrOperatorMCPCancelledV0            = i18nPublicOperatorCancelledV0
)

type OperatorMCPIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type OperatorMCPToolDescriptorV0 struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	ConnectorRef      string   `json:"connector_ref"`
	InputShape        string   `json:"input_shape"`
	InputRefs         []string `json:"input_refs"`
	RequiredInputRefs []string `json:"required_input_refs,omitempty"`
	OutputShape       string   `json:"output_shape"`
	PublicErrors      []string `json:"public_errors,omitempty"`
}

type OperatorMCPCapabilitiesV0 struct {
	SchemaVersion string                        `json:"schema_version"`
	ResourceURI   string                        `json:"resource_uri"`
	Tools         []OperatorMCPToolDescriptorV0 `json:"tools"`
	OpaqueRefs    []string                      `json:"opaque_refs"`
	Guardrails    []string                      `json:"guardrails"`
}

type OperatorStatusQueryV0 struct {
	RequestRef         string   `json:"request_ref"`
	SubjectRef         string   `json:"subject_ref"`
	StatusConnectorRef string   `json:"status_connector_ref"`
	IncludeSections    []string `json:"include_sections,omitempty"`
}

type OperatorSupervisedBurstRequestV0 struct {
	RequestRef        string   `json:"request_ref"`
	RunRef            string   `json:"run_ref"`
	BurstConnectorRef string   `json:"burst_connector_ref"`
	SupervisionRef    string   `json:"supervision_ref"`
	MaxSteps          int      `json:"max_steps"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}

type OperatorPendingOutboxQueryV0 struct {
	RequestRef         string   `json:"request_ref"`
	SubjectRef         string   `json:"subject_ref"`
	OutboxConnectorRef string   `json:"outbox_connector_ref"`
	Limit              int      `json:"limit"`
	IncludeKinds       []string `json:"include_kinds,omitempty"`
}

type OperatorDirectedQueryV0 struct {
	QueryRef          string   `json:"query_ref"`
	TargetRef         string   `json:"target_ref"`
	QueryConnectorRef string   `json:"query_connector_ref"`
	Question          string   `json:"question"`
	EvidenceRefs      []string `json:"evidence_refs,omitempty"`
}
