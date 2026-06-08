package orquestacoreworkflow

import (
	"encoding/json"
	"errors"
	"reflect"
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

func TestWorkflowTaskV0AcceptsNeutralLineageMetadata(t *testing.T) {
	task := validWorkflowTaskV0()
	task.ParentTaskRef = " task-parent-001 "
	task.CohortRef = " cohort-wave-001 "
	task.WaveRef = " wave-001 "
	task.DelegationDepth = 2
	task.MaxDelegationDepth = 4
	task.MaxChildAgents = 3
	task.MaxSubagentsPerAgent = 3
	task.MaxRecursiveAgents = 12
	task.ChildTaskRefs = []string{" task-child-001 ", "task-child-002"}

	got, err := NewWorkflowTaskV0(task)
	if err != nil {
		t.Fatalf("NewWorkflowTaskV0 lineage metadata: %v", err)
	}
	if got.ParentTaskRef != "task-parent-001" ||
		got.CohortRef != "cohort-wave-001" ||
		got.WaveRef != "wave-001" ||
		got.DelegationDepth != 2 ||
		got.MaxDelegationDepth != 4 ||
		got.MaxChildAgents != 3 ||
		got.MaxSubagentsPerAgent != 3 ||
		got.MaxRecursiveAgents != 12 ||
		!reflect.DeepEqual(got.ChildTaskRefs, []string{"task-child-001", "task-child-002"}) {
		t.Fatalf("lineage metadata not normalized: %+v", got)
	}
}

func TestWorkflowTaskV0AcceptsContextRefs(t *testing.T) {
	task := validWorkflowTaskV0()
	task.ContextRefs = []string{
		" context-ref-scope-001 ",
		"context-ref-policy-001",
		"context-ref-scope-001",
	}

	got, err := NewWorkflowTaskV0(task)
	if err != nil {
		t.Fatalf("NewWorkflowTaskV0 context refs: %v", err)
	}
	want := []string{"context-ref-scope-001", "context-ref-policy-001"}
	if !reflect.DeepEqual(got.ContextRefs, want) {
		t.Fatalf("context_refs=%+v want %+v", got.ContextRefs, want)
	}
}

func TestWorkflowTaskV0AcceptsSkillRefs(t *testing.T) {
	task := validWorkflowTaskV0()
	task.SkillRefs = []string{
		" skill-ref-orquesta-programacion-autonoma-v0 ",
		"skill-ref-orquesta-programacion-revision-v0",
		"skill-ref-orquesta-programacion-autonoma-v0",
	}

	got, err := NewWorkflowTaskV0(task)
	if err != nil {
		t.Fatalf("NewWorkflowTaskV0 skill refs: %v", err)
	}
	want := []string{
		"skill-ref-orquesta-programacion-autonoma-v0",
		"skill-ref-orquesta-programacion-revision-v0",
	}
	if !reflect.DeepEqual(got.SkillRefs, want) {
		t.Fatalf("skill_refs=%+v want %+v", got.SkillRefs, want)
	}
}

func TestValidateWorkflowTaskV0RejectsInvalidContextRefs(t *testing.T) {
	for name, mutate := range map[string]func(*WorkflowTaskV0){
		"with_space": func(task *WorkflowTaskV0) {
			task.ContextRefs = []string{"context ref invalid"}
		},
		"with_path_separator": func(task *WorkflowTaskV0) {
			task.ContextRefs = []string{"external/context-ref-001"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			task := validWorkflowTaskV0()
			mutate(&task)

			err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
			assertWorkflowTaskErrorV0(t, err, ErrWorkflowTaskInvalidaV0, "context_refs")
		})
	}
}

func TestValidateWorkflowTaskV0RejectsInvalidSkillRefs(t *testing.T) {
	for name, mutate := range map[string]func(*WorkflowTaskV0){
		"with_space": func(task *WorkflowTaskV0) {
			task.SkillRefs = []string{"skill ref invalid"}
		},
		"with_path_separator": func(task *WorkflowTaskV0) {
			task.SkillRefs = []string{"skills/orquesta-programacion-autonoma"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			task := validWorkflowTaskV0()
			mutate(&task)

			err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
			assertWorkflowTaskErrorV0(t, err, ErrWorkflowTaskInvalidaV0, "skill_refs")
		})
	}
}

func TestValidateWorkflowTaskV0RejectsInvalidLineageMetadata(t *testing.T) {
	cases := map[string]struct {
		mutate func(*WorkflowTaskV0)
		field  string
	}{
		"self_parent": {
			mutate: func(task *WorkflowTaskV0) { task.ParentTaskRef = task.TaskID },
			field:  "parent_task_ref",
		},
		"parent_ref_not_compact": {
			mutate: func(task *WorkflowTaskV0) { task.ParentTaskRef = "task parent" },
			field:  "parent_task_ref",
		},
		"negative_depth": {
			mutate: func(task *WorkflowTaskV0) { task.DelegationDepth = -1 },
			field:  "delegation_depth",
		},
		"too_many_child_agents": {
			mutate: func(task *WorkflowTaskV0) { task.MaxChildAgents = maxWorkflowTaskCollectionV0 + 1 },
			field:  "max_child_agents",
		},
		"too_many_delegation_depth_limit": {
			mutate: func(task *WorkflowTaskV0) { task.MaxDelegationDepth = maxWorkflowTaskCollectionV0 + 1 },
			field:  "max_delegation_depth",
		},
		"too_many_subagents_per_agent": {
			mutate: func(task *WorkflowTaskV0) { task.MaxSubagentsPerAgent = maxWorkflowTaskCollectionV0 + 1 },
			field:  "max_subagents_per_agent",
		},
		"too_many_recursive_agents": {
			mutate: func(task *WorkflowTaskV0) { task.MaxRecursiveAgents = maxWorkflowTaskRecursiveAgentsV0 + 1 },
			field:  "max_recursive_agents",
		},
		"self_child": {
			mutate: func(task *WorkflowTaskV0) { task.ChildTaskRefs = []string{task.TaskID} },
			field:  "child_task_refs",
		},
		"duplicate_child": {
			mutate: func(task *WorkflowTaskV0) { task.ChildTaskRefs = []string{"task-child-001", "task-child-001"} },
			field:  "child_task_refs",
		},
		"child_ref_not_compact": {
			mutate: func(task *WorkflowTaskV0) { task.ChildTaskRefs = []string{"task child"} },
			field:  "child_task_refs",
		},
		"child_matches_parent": {
			mutate: func(task *WorkflowTaskV0) {
				task.ParentTaskRef = "task-parent-001"
				task.ChildTaskRefs = []string{"task-parent-001"}
			},
			field: "child_task_refs",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			task := validWorkflowTaskV0()
			tc.mutate(&task)

			err := ValidateWorkflowTaskV0(NormalizeWorkflowTaskV0(task))
			assertWorkflowTaskErrorV0(t, err, ErrWorkflowTaskInvalidaV0, tc.field)
		})
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
		"internal/storage/schema.go/",
		"modulos/orquesta-core-workflow/**",
		"modulos/orquesta-runtime-codex/README.md",
		"docs/runbooks/*",
	}

	got, err := NewWorkflowTaskV0(task)
	if err != nil {
		t.Fatalf("NewWorkflowTaskV0 with repo write_set paths: %v", err)
	}
	want := []string{
		"server/api.go",
		"internal/storage/schema.go",
		"modulos/orquesta-core-workflow",
		"modulos/orquesta-runtime-codex/README.md",
		"docs/runbooks",
	}
	if !reflect.DeepEqual(got.WriteSet, want) {
		t.Fatalf("write_set=%+v want %+v", got.WriteSet, want)
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
		"windows_path":     {path: `modulos\orquesta-core-workflow\file.go`, code: ErrWorkflowTaskInvalidaV0},
		"api_key":          {path: "config/api_key=valor.env", code: ErrDetalleProhibidoV0},
		"access_token":     {path: "runtime/access_token=valor.txt", code: ErrDetalleProhibidoV0},
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
		"secret":        func(task *WorkflowTaskV0) { task.TaskID = "task-secret=valor" },
		"access_token":  func(task *WorkflowTaskV0) { task.ContextRefs = []string{"access_token=valor"} },
		"client_secret": func(task *WorkflowTaskV0) { task.FunctionContractRefs[0].ContractRef = "oauth-client_secret=valor" },
		"password":      func(task *WorkflowTaskV0) { task.AcceptanceCriteria = []string{"sin password=valor"} },
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

func TestValidateWorkflowTaskV0AllowsOpaqueOperationalDetails(t *testing.T) {
	task := validWorkflowTaskV0()
	task.Summary = "coordinar runtime provider db sql oauth docker tmux y token budget por refs opacas."
	task.AcceptanceCriteria = append(task.AcceptanceCriteria, "normalizar adapter refs sin cortar por palabras operativas.")
	task.ContextRefs = []string{"runtime-ref-001", "provider-ref-001", "token-budget-ref-001"}
	task.FunctionContractRefs[0].ContractRef = "adapter:runtime:v0"

	if _, err := NewWorkflowTaskV0(task); err != nil {
		t.Fatalf("NewWorkflowTaskV0 opaque operational details: %v", err)
	}
}

func TestWorkflowTaskV0SerializationHasNoSecrets(t *testing.T) {
	task := mustWorkflowTaskV0(t, validWorkflowTaskV0())

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}
	serialized := strings.ToLower(string(data))
	for _, forbidden := range []string{
		"secret", "password", "credential", "api_key", "access_token",
		"refresh_token", "client_secret",
	} {
		if containsForbiddenFragmentV0(serialized, forbidden) {
			t.Fatalf("serialized task contains forbidden detail %q: %s", forbidden, serialized)
		}
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
