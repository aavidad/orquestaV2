package orquestaappplanner

import orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"

const AppMicrotaskPlanSchemaVersionV0 = "orquesta_app_microtask_plan.v0"

const (
	AppPlanScaleStandardV0 = "standard"
	AppPlanScaleLargeV0    = "large"
)

type AppPlanRequestV0 struct {
	RunRef      string `json:"run_ref"`
	AppRef      string `json:"app_ref"`
	AppName     string `json:"app_name"`
	AppKind     string `json:"app_kind,omitempty"`
	Scale       string `json:"scale,omitempty"`
	OccurredAt  string `json:"occurred_at,omitempty"`
	RequestedBy string `json:"requested_by,omitempty"`
	API         bool   `json:"api"`
	Web         bool   `json:"web"`
	Locale      string `json:"locale,omitempty"`
}

type AppMicrotaskPlanV0 struct {
	SchemaVersion string          `json:"schema_version"`
	RunRef        string          `json:"run_ref"`
	AppRef        string          `json:"app_ref"`
	Units         []AppWorkUnitV0 `json:"units"`
	EvidenceRefs  []string        `json:"evidence_refs,omitempty"`
}

type AppWorkUnitV0 struct {
	TaskRef             string                                                     `json:"task_ref"`
	ClaimRef            string                                                     `json:"claim_ref"`
	AgentRequestID      string                                                     `json:"agent_request_id"`
	DeliveryRef         string                                                     `json:"delivery_ref"`
	PhaseID             orquestacoreworkflow.OrchestrationPhaseIDV0                `json:"phase_id"`
	WorkProfileKind     orquestacoreworkflow.WorkProfileKindV0                     `json:"work_profile_kind,omitempty"`
	Role                string                                                     `json:"role"`
	Capacity            orquestacoreworkflow.OrchestrationCapacityRecommendationV0 `json:"capacity"`
	Title               string                                                     `json:"title"`
	Summary             string                                                     `json:"summary"`
	WriteSet            []string                                                   `json:"write_set"`
	AcceptanceCriteria  []string                                                   `json:"acceptance_criteria"`
	RequiredTests       []string                                                   `json:"required_tests,omitempty"`
	DependsOnDeliveries []string                                                   `json:"depends_on_deliveries,omitempty"`
	EvidenceRefs        []string                                                   `json:"evidence_refs,omitempty"`
}
