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
		!spec.ReworkPolicy.PreferNewGoal {
		t.Fatalf("spec inesperado: %+v", spec)
	}
	if !reflect.DeepEqual(goalSpecPathsForTestV0(spec.WriteSet), request.WriteSet) {
		t.Fatalf("write_set=%v want=%v", goalSpecPathsForTestV0(spec.WriteSet), request.WriteSet)
	}
	if len(spec.RequiredTests) != 1 || spec.RequiredTests[0].Command != request.RequiredTests[0] {
		t.Fatalf("required_tests=%+v", spec.RequiredTests)
	}
	if !hasGoalContextRefForAutoprogrammingTestV0(spec.ContextRefs, "worktree", request.WorktreeRef) ||
		!hasGoalContextRefForAutoprogrammingTestV0(spec.ContextRefs, "workflow_task_context", "source_task_ref:task-ref-goal-ready-001") ||
		len(spec.RuleRefs) == 0 {
		t.Fatalf("context/rules incompletos: context=%+v rules=%+v", spec.ContextRefs, spec.RuleRefs)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0GeneraGoalSpecsPorGrupo(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = []AutoprogrammingTaskGroupCandidateV0{
			{
				TaskRef:     "task-ref-goal-api-001",
				Area:        "api",
				ContextRefs: []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
			},
			{
				TaskRef:     "task-ref-goal-web-001",
				Area:        "web",
				ContextRefs: []string{"goal_migration:goal-first", "goal_capability:starter", "goal_capability:observer", "goal_capability:closure-validator"},
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
