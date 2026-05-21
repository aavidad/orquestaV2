package orquestacoreworkflow

const (
	WorkflowTaskSchemaVersionV0 = "workflow_task.v0"

	ErrWorkflowTaskInvalidaV0        = "workflow_task_invalida"
	ErrWorkflowTaskPayloadInvalidoV0 = "payload_invalido"
)

const (
	maxWorkflowTaskPayloadBytesV0 = 4096
	maxWorkflowTaskStringV0       = 600
	maxWorkflowTaskCollectionV0   = 40
)

var forbiddenWorkflowTaskFragmentsV0 = []string{
	"db",
	"database",
	"sql",
	"dsn",
	"runtime",
	"provider",
	"proveedor",
	"home",
	"oauth",
	"docker",
	"tmux",
	"secret",
	"secreto",
	"token",
	"password",
	"credential",
	"credencial",
	"api_key",
}

type WorkflowTaskV0 struct {
	SchemaVersion        string                          `json:"schema_version"`
	TaskID               string                          `json:"task_id"`
	RunID                string                          `json:"run_id"`
	PhaseID              OrchestrationPhaseIDV0          `json:"phase_id"`
	WorkProfileKind      WorkProfileKindV0               `json:"work_profile_kind,omitempty"`
	Title                string                          `json:"title"`
	Summary              string                          `json:"summary,omitempty"`
	WriteSet             []string                        `json:"write_set"`
	AcceptanceCriteria   []string                        `json:"acceptance_criteria"`
	RequiredTests        []string                        `json:"required_tests,omitempty"`
	DependsOn            []string                        `json:"depends_on,omitempty"`
	ParentTaskRef        string                          `json:"parent_task_ref,omitempty"`
	CohortRef            string                          `json:"cohort_ref,omitempty"`
	WaveRef              string                          `json:"wave_ref,omitempty"`
	DelegationDepth      int                             `json:"delegation_depth,omitempty"`
	MaxChildAgents       int                             `json:"max_child_agents,omitempty"`
	ChildTaskRefs        []string                        `json:"child_task_refs,omitempty"`
	FunctionContractRefs []WorkflowFunctionContractRefV0 `json:"function_contract_refs,omitempty"`
}

type WorkflowFunctionContractRefV0 struct {
	ContractRef  string `json:"contract_ref,omitempty"`
	FunctionName string `json:"function_name,omitempty"`
}

type WorkflowTaskErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

func (err WorkflowTaskErrorV0) Error() string {
	return err.Code
}

func NewWorkflowTaskV0(task WorkflowTaskV0) (WorkflowTaskV0, error) {
	normalized := NormalizeWorkflowTaskV0(task)
	if err := ValidateWorkflowTaskV0(normalized); err != nil {
		return WorkflowTaskV0{}, err
	}
	return normalized, nil
}
