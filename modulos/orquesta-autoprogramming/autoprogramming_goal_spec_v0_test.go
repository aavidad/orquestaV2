package orquestaautoprogramming

import (
	"reflect"
	"testing"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestBuildAutoprogrammingProgrammableWorkV0GeneraGoalSpecsCuandoGoalListo(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:            "task-ref-goal-ready-001",
			Area:               "Autoprogramming",
			Title:              "Migrar autoprogramacion a Goal",
			Objective:          "Compilar contrato GoalWorkSpecV0 sin arrancar runtime.",
			ContextRefs:        []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
			AcceptanceCriteria: []string{"GoalWorkSpecV0 valida y conserva write-set/pruebas."},
			SkillRefs:          []string{"skill-ref-autoprogramming"},
		}}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming/autoprogramming_goal_spec_v0.go",
			"modulos/orquesta-autoprogramming/autoprogramming_goal_spec_v0_test.go",
		}
		request.RequiredTests = []string{"go test -count=1 ./modulos/orquesta-autoprogramming"}
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if result.Work.GoalMigration.Status != AutoprogrammingGoalMigrationGoalReadyV0 ||
		len(result.Work.GoalSpecs) != 1 {
		t.Fatalf("goal_migration/specs inesperados: %+v specs=%d", result.Work.GoalMigration, len(result.Work.GoalSpecs))
	}
	if !reflect.DeepEqual(result.Work.BatchRequiredTests, request.RequiredTests) {
		t.Fatalf("batch_required_tests=%v want=%v", result.Work.BatchRequiredTests, request.RequiredTests)
	}
	if len(result.Work.Tasks) != 0 || len(result.Work.Profiles) != 0 {
		t.Fatalf("goal-ready no debe publicar superficie WorkflowTask legacy: tasks=%d profiles=%d", len(result.Work.Tasks), len(result.Work.Profiles))
	}
	spec := result.Work.GoalSpecs[0]
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		t.Fatalf("goal spec invalido: %+v spec=%+v", issues, spec)
	}
	if spec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		spec.WorkKind != AutoprogrammingGoalWorkKindV0 ||
		spec.RequestRef != request.RequestRef ||
		spec.ProjectRef != request.ProjectRef ||
		spec.ClosurePolicy.RequireRequiredTests != true ||
		!spec.ClosurePolicy.RequireIndependentRequiredTestAttestation ||
		spec.ImplementerAgentRef != "" || spec.ImplementerCredentialRef != "" ||
		spec.ClosurePolicy.RequiredAttestorTrustPolicyRef != "" ||
		spec.WriteSetSHA256 != orquestagoal.GoalWriteSetSHA256V0(spec.WriteSet) ||
		!spec.ReworkPolicy.PreferNewGoal {
		t.Fatalf("spec inesperado: %+v", spec)
	}
	if !reflect.DeepEqual(goalSpecPathsForTestV0(spec.WriteSet), request.WriteSet) {
		t.Fatalf("write_set=%v want=%v", goalSpecPathsForTestV0(spec.WriteSet), request.WriteSet)
	}
	if len(spec.RequiredTests) != 1 || spec.RequiredTests[0].Command != request.RequiredTests[0] ||
		spec.RequiredTests[0].CommandRef == "" || spec.RequiredTests[0].CommandSHA256 == "" ||
		spec.RequiredTests[0].DefinitionSHA256 == "" {
		t.Fatalf("required_tests=%+v", spec.RequiredTests)
	}
	if len(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs) != 0 {
		t.Fatalf("legacy closure criteria refs=%v", spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs)
	}
	if !hasGoalContextRefForAutoprogrammingTestV0(spec.ContextRefs, "worktree", request.WorktreeRef) ||
		!hasGoalContextRefForAutoprogrammingTestV0(spec.ContextRefs, "workflow_task_context", "source_task_ref:task-ref-goal-ready-001") ||
		len(spec.RuleRefs) == 0 {
		t.Fatalf("context/rules incompletos: context=%+v rules=%+v", spec.ContextRefs, spec.RuleRefs)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0SeparaTestsBatchDeGoalsMultiples(t *testing.T) {
	const batchTest = "go test -count=1 ./..."
	const taskTestA = "go test -count=1 ./modulos/orquesta-autoprogramming -run TestGoalA"
	const taskTestB = "go test -count=1 ./modulos/orquesta-autoprogramming -run TestGoalB"
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.RequiredTests = []string{batchTest}
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{TaskRef: "task-ref-goal-batch-a", Area: "area-a", RequiredTests: []string{taskTestA}, ContextRefs: []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"}},
			{TaskRef: "task-ref-goal-batch-b", Area: "area-b", RequiredTests: []string{taskTestB}, ContextRefs: []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"}},
		}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/area-a.go", "modulos/orquesta-autoprogramming/area-b.go"}
	}))

	if !result.Accepted || !reflect.DeepEqual(result.Work.BatchRequiredTests, []string{batchTest}) || len(result.Work.GoalSpecs) != 2 {
		t.Fatalf("result=%+v", result)
	}
	if !hasGoalRequiredTestCommandForAutoprogrammingTestV0(result.Work.GoalSpecs[0].RequiredTests, taskTestA) ||
		hasGoalRequiredTestCommandForAutoprogrammingTestV0(result.Work.GoalSpecs[0].RequiredTests, batchTest) ||
		!hasGoalRequiredTestCommandForAutoprogrammingTestV0(result.Work.GoalSpecs[1].RequiredTests, taskTestB) ||
		hasGoalRequiredTestCommandForAutoprogrammingTestV0(result.Work.GoalSpecs[1].RequiredTests, batchTest) {
		t.Fatalf("goal required_tests=%+v / %+v", result.Work.GoalSpecs[0].RequiredTests, result.Work.GoalSpecs[1].RequiredTests)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0ProyectaSoloAutorizacionesDeSusTasksV0(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{
				TaskRef: "task-ref-destructive-a", Area: "area-a", WriteSet: []string{"modulos/orquesta-autoprogramming/area-a.go", "modulos/orquesta-autoprogramming/area-a-renamed.go"}, RequiredTests: []string{"go test ./area-a"},
				ContextRefs:               []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
				DestructiveAuthorizations: []orquestagoal.GoalDestructiveChangeAuthorizationV0{{Kind: orquestagoal.GoalDestructiveChangeAuthorizationRenameV0, PreviousPath: "modulos/orquesta-autoprogramming/area-a.go", CurrentPath: "modulos/orquesta-autoprogramming/area-a-renamed.go"}},
			},
			{
				TaskRef: "task-ref-destructive-b", Area: "area-b", WriteSet: []string{"modulos/orquesta-autoprogramming/area-b.go"}, RequiredTests: []string{"go test ./area-b"},
				ContextRefs:               []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
				DestructiveAuthorizations: []orquestagoal.GoalDestructiveChangeAuthorizationV0{{Kind: orquestagoal.GoalDestructiveChangeAuthorizationRemoveV0, Path: "modulos/orquesta-autoprogramming/area-b.go"}},
			},
		}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/area-a.go", "modulos/orquesta-autoprogramming/area-a-renamed.go", "modulos/orquesta-autoprogramming/area-b.go"}
	}))

	if !result.Accepted || len(result.Work.GoalSpecs) != 2 {
		t.Fatalf("result=%+v", result)
	}
	first, second := result.Work.GoalSpecs[0].DestructiveAuthorizations, result.Work.GoalSpecs[1].DestructiveAuthorizations
	if len(first) != 1 || first[0].Kind != orquestagoal.GoalDestructiveChangeAuthorizationRenameV0 ||
		len(second) != 1 || second[0].Kind != orquestagoal.GoalDestructiveChangeAuthorizationRemoveV0 {
		t.Fatalf("authorizations=%+v / %+v", first, second)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0RechazaAutorizacionFueraDelWriteSetAntesDeLaunchV0(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks[0].ContextRefs = []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"}
		request.Tasks[0].DestructiveAuthorizations = []orquestagoal.GoalDestructiveChangeAuthorizationV0{{
			Kind: orquestagoal.GoalDestructiveChangeAuthorizationRemoveV0, Path: "modulos/orquesta-goal/forbidden.go",
		}}
	}))
	if result.Accepted || len(result.Issues) == 0 || result.Issues[0].Code != autoprogrammingGoalSpecIssueV0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0ConvierteBaselineDeTaskEnContextoGoalTipadoV0(t *testing.T) {
	const baselineRef = "worktree-baseline-ref-goal-context-001"
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-goal-baseline-context-001",
			Area:    "autoprogramming",
			ContextRefs: []string{
				"goal_migration:goal-first",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
				autoprogrammingWorktreeBaselinePrefixV0 + baselineRef,
			},
		}}
	}))
	if !result.Accepted || len(result.Work.GoalSpecs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	refs := result.Work.GoalSpecs[0].ContextRefs
	if !hasGoalContextRefForAutoprogrammingTestV0(refs, "worktree_baseline", baselineRef) {
		t.Fatalf("context_refs=%+v", refs)
	}
	if hasGoalContextRefForAutoprogrammingTestV0(refs, "workflow_task_context", autoprogrammingWorktreeBaselinePrefixV0+baselineRef) {
		t.Fatalf("baseline conservado como contexto generico: %+v", refs)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0GoalSpecAddsAcceptanceCheckTestAndPolicy(t *testing.T) {
	check := AutoprogrammingAcceptanceCheckV0{
		CriterionRef: "criterion-ref-bug-208ag-001",
		Description:  "El transporte tipado conserva el criterio verificable.",
		Command:      "go test -count=1 ./modulos/orquesta-autoprogramming -run TestBUG208AG",
	}
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-goal-acceptance-check-001", Area: "autoprogramming",
			ContextRefs:      []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
			AcceptanceChecks: []AutoprogrammingAcceptanceCheckV0{check},
		}}
	}))
	if !result.Accepted || len(result.Work.GoalSpecs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	spec := result.Work.GoalSpecs[0]
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) != 0 {
		t.Fatalf("goal spec invalido: %+v", issues)
	}
	acceptanceTest := goalRequiredTestForCommandForAutoprogrammingTestV0(spec.RequiredTests, check.Command)
	if acceptanceTest == nil || !reflect.DeepEqual(acceptanceTest.AcceptanceCriteria, []string{check.Description}) ||
		!reflect.DeepEqual(acceptanceTest.AcceptanceCriteriaRefs, []string{check.CriterionRef}) ||
		!reflect.DeepEqual(spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs, []string{check.CriterionRef}) {
		t.Fatalf("acceptance check no transportado: test=%+v policy=%+v", acceptanceTest, spec.ClosurePolicy)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0GoalSpecMergesAcceptanceCheckWithExistingCommand(t *testing.T) {
	command := "go test -count=1 ./modulos/orquesta-autoprogramming"
	check := AutoprogrammingAcceptanceCheckV0{CriterionRef: "criterion-ref-bug-208ag-merge", Description: "El comando existente queda enriquecido.", Command: command}
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.RequiredTests = []string{command}
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-goal-acceptance-merge-001", Area: "autoprogramming",
			ContextRefs:      []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
			AcceptanceChecks: []AutoprogrammingAcceptanceCheckV0{check},
		}}
	}))
	if !result.Accepted || len(result.Work.GoalSpecs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	spec := result.Work.GoalSpecs[0]
	if len(spec.RequiredTests) != 1 || spec.RequiredTests[0].DefinitionSHA256 != orquestagoal.FreezeGoalRequiredTestV0(spec.RequiredTests[0]).DefinitionSHA256 ||
		!reflect.DeepEqual(spec.RequiredTests[0].AcceptanceCriteriaRefs, []string{check.CriterionRef}) {
		t.Fatalf("required_tests=%+v", spec.RequiredTests)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0GoalReadyDejaBindingConfiableALaComposicion(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-goal-attestation-missing-001", Area: "autoprogramming",
			ContextRefs: []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
		}}
	})
	result := BuildAutoprogrammingProgrammableWorkV0(request)
	if !result.Accepted || len(result.Work.GoalSpecs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	if issues := orquestagoal.ValidateGoalRequiredTestAttestationBindingV0(result.Work.GoalSpecs[0]); len(issues) == 0 {
		t.Fatal("unbound spec must require trusted composition binding before launch")
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0GeneraGoalSpecsPorGrupo(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{
				TaskRef:       "task-ref-goal-api-001",
				Area:          "api",
				RequiredTests: []string{"go test -count=1 ./modulos/orquesta-autoprogramming -run TestGoalAPI"},
				ContextRefs:   []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
			},
			{
				TaskRef:       "task-ref-goal-web-001",
				Area:          "web",
				RequiredTests: []string{"go test -count=1 ./modulos/orquesta-autoprogramming -run TestGoalWeb"},
				ContextRefs:   []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
			},
		}
		request.WriteSet = []string{
			"modulos/orquesta-autoprogramming/api/goal.go",
			"modulos/orquesta-autoprogramming/web/goal.go",
		}
		request.MaxTaskRefs = 2
		request.MaxAreas = 2
		request.MaxWriteSetEntries = 2
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted ||
		result.Work.GoalMigration.Status != AutoprogrammingGoalMigrationGoalReadyV0 ||
		len(result.Work.GoalSpecs) != 2 {
		t.Fatalf("result=%+v", result)
	}
	for _, spec := range result.Work.GoalSpecs {
		if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
			t.Fatalf("goal spec invalido: %+v spec=%+v", issues, spec)
		}
		if spec.RunRef != "" {
			t.Fatalf("run_ref debe rellenarlo la composicion: %+v", spec)
		}
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0GoalSpecInyectaGuardStartupLockDesdeIncidencia(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef: "task-ref-goal-startup-lock-001",
			Area:    "Runtime Codex",
			ContextRefs: []string{
				"goal_migration:goal-first",
				"goal_capability:starter",
				"goal_capability:observer",
				"goal_capability:closure-validator",
				autoprogrammingStartupLockRemoteIncidentRefV0,
			},
		}}
		request.WriteSet = []string{
			"modulos/orquesta-runtime-codex/codex_wrapper_v0.go",
			"modulos/orquesta-runtime-codex/codex_wrapper_v0_test.go",
		}
		request.MaxWriteSetEntries = 2
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted ||
		result.Work.GoalMigration.Status != AutoprogrammingGoalMigrationGoalReadyV0 ||
		len(result.Work.GoalSpecs) != 1 {
		t.Fatalf("result=%+v", result)
	}
	spec := result.Work.GoalSpecs[0]
	if issues := orquestagoal.ValidateGoalWorkSpecV0(spec); len(issues) > 0 {
		t.Fatalf("goal spec invalido: %+v spec=%+v", issues, spec)
	}
	if !hasGoalRequiredTestCommandForAutoprogrammingTestV0(spec.RequiredTests, autoprogrammingStartupLockRegressionTestV0) {
		t.Fatalf("required_tests=%+v", spec.RequiredTests)
	}
	if !hasGoalRuleRefForAutoprogrammingTestV0(spec.RuleRefs, autoprogrammingStartupLockContractRefV0) {
		t.Fatalf("rule_refs=%+v", spec.RuleRefs)
	}
	for _, want := range []string{
		"ORQUESTA_CODEX_STARTUP_LOCK_SECONDS",
		"ORQUESTA_CODEX_STARTUP_LOCK_TIMEOUT_SECONDS",
		"ORQUESTA_CODEX_STARTUP_LOCK_STALE_SECONDS",
	} {
		if !stringsSliceContainsSubstringForAutoprogrammingTestV0(spec.AcceptanceCriteria, want) {
			t.Fatalf("criteria no contiene %q: %v", want, spec.AcceptanceCriteria)
		}
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0NoGeneraGoalSpecsSinCapacidades(t *testing.T) {
	result := BuildAutoprogrammingProgrammableWorkV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{{
			TaskRef:     "task-ref-goal-blocked-001",
			Area:        "Autoprogramming",
			ContextRefs: []string{"goal_migration:goal-first", "goal_capability:starter"},
		}}
		request.WriteSet = []string{"modulos/orquesta-autoprogramming/autoprogramming_goal_migration_v0.go"}
	}))

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if result.Work.GoalMigration.Status != AutoprogrammingGoalMigrationBlockedByGoalCapabilityV0 {
		t.Fatalf("goal_migration=%+v", result.Work.GoalMigration)
	}
	if len(result.Work.GoalSpecs) != 0 {
		t.Fatalf("goal_specs=%+v", result.Work.GoalSpecs)
	}
}

func goalSpecPathsForTestV0(values []orquestagoal.GoalWriteScopeV0) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Path)
	}
	return out
}

func hasGoalContextRefForAutoprogrammingTestV0(values []orquestagoal.GoalContextRefV0, kind, ref string) bool {
	for _, value := range values {
		if value.Kind == kind && value.Ref == ref {
			return true
		}
	}
	return false
}

func hasGoalRequiredTestCommandForAutoprogrammingTestV0(
	values []orquestagoal.GoalRequiredTestV0,
	command string,
) bool {
	for _, value := range values {
		if value.Command == command {
			return true
		}
	}
	return false
}

func goalRequiredTestForCommandForAutoprogrammingTestV0(values []orquestagoal.GoalRequiredTestV0, command string) *orquestagoal.GoalRequiredTestV0 {
	for index := range values {
		if values[index].Command == command {
			return &values[index]
		}
	}
	return nil
}

func hasGoalRuleRefForAutoprogrammingTestV0(
	values []orquestagoal.GoalRuleRefV0,
	ref string,
) bool {
	for _, value := range values {
		if value.Ref == ref {
			return true
		}
	}
	return false
}
