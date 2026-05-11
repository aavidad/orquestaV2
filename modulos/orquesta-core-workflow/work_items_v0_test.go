package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestNewWorkflowTaskV0AcceptsValidCompactTask(t *testing.T) {
	task, err := NewWorkflowTaskV0(validWorkflowTaskV0())
	if err != nil {
		t.Fatalf("NewWorkflowTaskV0: %v", err)
	}

	if task.SchemaVersion != WorkflowTaskSchemaVersionV0 {
		t.Fatalf("schema_version=%q, want %q", task.SchemaVersion, WorkflowTaskSchemaVersionV0)
	}
	if task.PhaseID != OrchestrationPhaseProgramacionV0 {
		t.Fatalf("phase_id=%q, want %q", task.PhaseID, OrchestrationPhaseProgramacionV0)
	}
	if len(task.WriteSet) != 2 || task.WriteSet[0] != "work_items_v0.go" {
		t.Fatalf("write_set not normalized: %+v", task.WriteSet)
	}
	if len(task.FunctionContractRefs) != 1 || task.FunctionContractRefs[0].ContractRef == "" {
		t.Fatalf("function_contract_refs not preserved: %+v", task.FunctionContractRefs)
	}
	if len(task.RequiredTests) != 1 || task.RequiredTests[0] != "go test ./..." {
		t.Fatalf("required_tests not preserved: %+v", task.RequiredTests)
	}
}

func TestWorkflowTaskV0KeepsFunctionNameOnlyRefCompatible(t *testing.T) {
	task := validWorkflowTaskV0()
	task.FunctionContractRefs = []WorkflowFunctionContractRefV0{
		{FunctionName: "NewWorkflowTaskV0"},
	}

	got, err := NewWorkflowTaskV0(task)
	if err != nil {
		t.Fatalf("NewWorkflowTaskV0 function-name-only ref: %v", err)
	}
	if got.FunctionContractRefs[0].ContractRef != "" || got.FunctionContractRefs[0].FunctionName != "NewWorkflowTaskV0" {
		t.Fatalf("function_contract_refs=%+v", got.FunctionContractRefs)
	}
}

func TestValidateWorkflowTaskV0RejectsUnknownPhase(t *testing.T) {
	task := validWorkflowTaskV0()
	task.PhaseID = OrchestrationPhaseIDV0("despliegue")

	err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
	var publicErr OrchestrationValidationIssueV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected phase validation issue, got %T %v", err, err)
	}
	if publicErr.Code != OrchestrationFaseNoSoportadaV0 {
		t.Fatalf("code=%q, want %q", publicErr.Code, OrchestrationFaseNoSoportadaV0)
	}
}

func TestValidateWorkflowTaskV0RejectsEmptyWriteSet(t *testing.T) {
	task := validWorkflowTaskV0()
	task.WriteSet = nil

	err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
	assertWorkflowTaskErrorV0(t, err, ErrWorkflowTaskInvalidaV0, "write_set")
}

func TestValidateWorkflowTaskV0AcceptsWriteSetRepoPathsWithOperationalNames(t *testing.T) {
	task := validWorkflowTaskV0()
	task.WriteSet = []string{
		"server/api.go",
		"internal/storage/schema.go",
		"modulos/orquesta-runtime-codex/README.md",
	}

	if _, err := NewWorkflowTaskV0(task); err != nil {
		t.Fatalf("NewWorkflowTaskV0 with repo write_set paths: %v", err)
	}
}

func TestValidateWorkflowTaskV0RejectsUnsafeWriteSetPaths(t *testing.T) {
	cases := map[string]struct {
		path string
		code string
	}{
		"absolute":         {path: "/tmp/repo/file.go", code: ErrWorkflowTaskInvalidaV0},
		"traversal":        {path: "../internal/storage/schema.go", code: ErrWorkflowTaskInvalidaV0},
		"nested_traversal": {path: "modulos/../internal/storage/schema.go", code: ErrWorkflowTaskInvalidaV0},
		"url":              {path: "https://example.test/file.go", code: ErrWorkflowTaskInvalidaV0},
		"env_ref":          {path: "$HOME/config.go", code: ErrWorkflowTaskInvalidaV0},
		"tilde_ref":        {path: "~/repo/file.go", code: ErrWorkflowTaskInvalidaV0},
		"secret":           {path: "config/secret.env", code: ErrDetalleProhibidoV0},
		"token":            {path: "runtime/token.txt", code: ErrDetalleProhibidoV0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			task := validWorkflowTaskV0()
			task.WriteSet = []string{tc.path}

			err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
			assertWorkflowTaskErrorV0(t, err, tc.code, "write_set")
		})
	}
}

func TestValidateWorkflowTaskV0RejectsEmptyAcceptanceCriterion(t *testing.T) {
	task := validWorkflowTaskV0()
	task.AcceptanceCriteria = []string{"compila", " "}

	err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
	assertWorkflowTaskErrorV0(t, err, ErrWorkflowTaskInvalidaV0, "acceptance_criteria")
}

func TestValidateWorkflowTaskV0RejectsEmptyRequiredTest(t *testing.T) {
	task := validWorkflowTaskV0()
	task.RequiredTests = []string{"go test ./...", " "}

	err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
	assertWorkflowTaskErrorV0(t, err, ErrWorkflowTaskInvalidaV0, "required_tests")
}

func TestValidateWorkflowTaskV0RejectsForbiddenDetails(t *testing.T) {
	cases := map[string]func(*WorkflowTaskV0){
		"db":        func(task *WorkflowTaskV0) { task.Summary = "depende de db real" },
		"runtime":   func(task *WorkflowTaskV0) { task.Summary = "depende del runtime real" },
		"proveedor": func(task *WorkflowTaskV0) { task.Title = "configurar proveedor remoto" },
		"HOME":      func(task *WorkflowTaskV0) { task.AcceptanceCriteria = []string{"no leer $HOME"} },
		"oauth":     func(task *WorkflowTaskV0) { task.FunctionContractRefs[0].ContractRef = "oauth:client" },
		"docker":    func(task *WorkflowTaskV0) { task.Summary = "usar Docker local" },
		"tmux":      func(task *WorkflowTaskV0) { task.Summary = "sesion tmux" },
		"secret":    func(task *WorkflowTaskV0) { task.TaskID = "task-secret" },
		"token":     func(task *WorkflowTaskV0) { task.RunID = "run-token" },
		"password":  func(task *WorkflowTaskV0) { task.AcceptanceCriteria = []string{"sin password"} },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			task := validWorkflowTaskV0()
			mutate(&task)

			err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
			assertWorkflowTaskErrorCodeV0(t, err, ErrDetalleProhibidoV0)
		})
	}
}

func TestWorkflowTaskV0SerializationHasNoAdaptersOrSecrets(t *testing.T) {
	task := mustWorkflowTaskV0(t, validWorkflowTaskV0())

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	serialized := strings.ToLower(string(data))
	for _, forbidden := range []string{
		"db", "sql", "runtime", "provider", "proveedor", "home",
		"oauth", "docker", "tmux", "secret", "token", "password",
	} {
		if containsForbiddenFragmentV0(serialized, forbidden) {
			t.Fatalf("serialized task contains forbidden detail %q: %s", forbidden, serialized)
		}
	}
	if strings.Contains(serialized, "adapter") || strings.Contains(serialized, "adaptador") {
		t.Fatalf("serialized task contains adapter detail: %s", serialized)
	}
}

func validWorkflowTaskV0() WorkflowTaskV0 {
	return WorkflowTaskV0{
		TaskID:  "task-work-item-001",
		RunID:   "run-work-item-001",
		PhaseID: OrchestrationPhaseProgramacionV0,
		Title:   "Definir DTO de work item",
		Summary: "Contrato compacto para una unidad pequena de trabajo",
		WriteSet: []string{
			"work_items_v0.go",
			"work_items_v0_test.go",
		},
		AcceptanceCriteria: []string{
			"constructor publico normaliza campos",
			"validador rechaza refs vacias y detalles prohibidos",
		},
		RequiredTests: []string{"go test ./..."},
		FunctionContractRefs: []WorkflowFunctionContractRefV0{
			{ContractRef: "contract:workflow-task:v0", FunctionName: "NewWorkflowTaskV0"},
		},
	}
}

func mustWorkflowTaskV0(t *testing.T, task WorkflowTaskV0) WorkflowTaskV0 {
	t.Helper()
	normalized, err := NewWorkflowTaskV0(task)
	if err != nil {
		t.Fatalf("NewWorkflowTaskV0: %v", err)
	}
	return normalized
}

func assertWorkflowTaskErrorV0(t *testing.T, err error, code string, field string) {
	t.Helper()
	var publicErr WorkflowTaskErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected WorkflowTaskErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code || publicErr.Field != field {
		t.Fatalf("error=%+v, want code=%s field=%s", publicErr, code, field)
	}
}

func assertWorkflowTaskErrorCodeV0(t *testing.T, err error, code string) {
	t.Helper()
	var publicErr WorkflowTaskErrorV0
	if !errors.As(err, &publicErr) {
		t.Fatalf("expected WorkflowTaskErrorV0, got %T %v", err, err)
	}
	if publicErr.Code != code {
		t.Fatalf("code=%q, want %q", publicErr.Code, code)
	}
}
