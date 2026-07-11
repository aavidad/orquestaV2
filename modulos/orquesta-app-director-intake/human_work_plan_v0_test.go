package orquestaappdirectorintake

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildHumanDirectorReviewablePlanV0ExecuteNowConRefsOpacas(t *testing.T) {
	result := BuildHumanDirectorReviewablePlanV0(validHumanDirectorWorkIntakeRequestForTestV0())
	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if result.Plan.Status != HumanDirectorPlanStatusReadyForReviewV0 || !result.Plan.ReviewRequired {
		t.Fatalf("plan no revisable: %+v", result.Plan)
	}
	if result.Plan.WorktreeRef != "worktree-ref-orquesta-main-20260523" ||
		result.Plan.BranchRef != "branch-ref-trabajo-plataforma-agentes" {
		t.Fatalf("refs opacas perdidas: %+v", result.Plan)
	}
	assertHumanDirectorContextRefV0(t, result.Plan.ContextRefs, "worktree_ref:worktree-ref-orquesta-main-20260523")
	assertHumanDirectorContextRefV0(t, result.Plan.ContextRefs, "branch_ref:branch-ref-trabajo-plataforma-agentes")
	if len(result.Plan.Steps) != 1 || result.Plan.Steps[0].Action != HumanDirectorPlanActionExecuteNowV0 {
		t.Fatalf("steps=%+v", result.Plan.Steps)
	}
	if !result.Plan.Steps[0].SafeRepairAllowed {
		t.Fatalf("safe repair no preservado: %+v", result.Plan.Steps[0])
	}
}

func TestBuildHumanDirectorReviewablePlanV0PreservaAcceptanceChecks(t *testing.T) {
	request := validHumanDirectorWorkIntakeRequestForTestV0()
	request.Request.AcceptanceChecks = []HumanDirectorAcceptanceCheckV0{{
		CriterionRef: " criterion-ref-human-001 ",
		Description:  " comprobacion humana explicita ",
		Command:      " go test -count=1 ./modulos/orquesta-app-director-intake ",
	}}

	result := BuildHumanDirectorReviewablePlanV0(request)
	if !result.Accepted || len(result.Plan.Steps) != 1 {
		t.Fatalf("result=%+v", result)
	}
	got := result.Plan.Steps[0].AcceptanceChecks
	if len(got) != 1 || got[0] != (HumanDirectorAcceptanceCheckV0{
		CriterionRef: "criterion-ref-human-001",
		Description:  "comprobacion humana explicita",
		Command:      "go test -count=1 ./modulos/orquesta-app-director-intake",
	}) {
		t.Fatalf("acceptance_checks=%+v", got)
	}
	request.Request.AcceptanceChecks[0].Command = "mutado"
	if result.Plan.Steps[0].AcceptanceChecks[0].Command == "mutado" {
		t.Fatalf("acceptance_checks comparte memoria con request")
	}
}

func TestBuildHumanDirectorReviewablePlanV0RechazaAcceptanceChecksInvalidos(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		checks []HumanDirectorAcceptanceCheckV0
		code   string
	}{
		{
			name:   "criterion_ref ausente",
			checks: []HumanDirectorAcceptanceCheckV0{{Command: "go test ./..."}},
			code:   "acceptance_check_criterion_ref_required",
		},
		{
			name:   "command ausente",
			checks: []HumanDirectorAcceptanceCheckV0{{CriterionRef: "criterion-ref-001"}},
			code:   "acceptance_check_command_required",
		},
		{
			name: "criterion_ref duplicado",
			checks: []HumanDirectorAcceptanceCheckV0{
				{CriterionRef: "criterion-ref-001", Command: "go test ./one"},
				{CriterionRef: " criterion-ref-001 ", Command: "go test ./two"},
			},
			code: "acceptance_check_criterion_ref_duplicate",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := validHumanDirectorWorkIntakeRequestForTestV0()
			request.Request.AcceptanceChecks = testCase.checks

			result := BuildHumanDirectorReviewablePlanV0(request)
			if result.Accepted || result.Plan.Status != HumanDirectorPlanStatusInvalidV0 {
				t.Fatalf("result=%+v", result)
			}
			if !humanDirectorPlanIssuesContainCodeV0(result.Issues, testCase.code) {
				t.Fatalf("issues=%+v want=%s", result.Issues, testCase.code)
			}
		})
	}
}

func TestHumanDirectorWorkRequestV0LegacySinAcceptanceChecksSerializaIgual(t *testing.T) {
	payload, err := json.Marshal(HumanDirectorWorkRequestV0{Objective: "objetivo"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(payload), "acceptance_checks") {
		t.Fatalf("payload legacy incluye acceptance_checks: %s", payload)
	}
}

func TestBuildHumanDirectorReviewablePlanV0EstudiaAntesSiWriteSetAmplio(t *testing.T) {
	request := validHumanDirectorWorkIntakeRequestForTestV0()
	request.Hints.WriteSet = nil

	result := BuildHumanDirectorReviewablePlanV0(request)
	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionStudyBeforeV0)
}

func TestBuildHumanDirectorReviewablePlanV0NoRechazaTodoPorSolape(t *testing.T) {
	request := validHumanDirectorWorkIntakeRequestForTestV0()
	request.Hints.WriteSet = []string{
		"docs/decisiones.md",
		"modulos/orquesta-app-director-intake",
		"modulos/orquesta-app-director-intake/docs",
	}

	result := BuildHumanDirectorReviewablePlanV0(request)
	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionStudyBeforeV0)
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionExecuteNowV0)
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionPostponeOverlapV0)
	if result.Plan.Steps[len(result.Plan.Steps)-1].WriteSet[0] != "modulos/orquesta-app-director-intake" {
		t.Fatalf("postpone no conserva alcance solapado: %+v", result.Plan.Steps)
	}
}

func TestBuildHumanDirectorReviewablePlanV0PideRevisionSinObjetivo(t *testing.T) {
	request := validHumanDirectorWorkIntakeRequestForTestV0()
	request.Request.Objective = " "

	result := BuildHumanDirectorReviewablePlanV0(request)
	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionRequestReviewV0)
}

func TestBuildHumanDirectorReviewablePlanV0NoInvalidaRefsOperativasPendientes(t *testing.T) {
	request := validHumanDirectorWorkIntakeRequestForTestV0()
	request.WorktreeRef = " "
	request.BranchRef = " "

	result := BuildHumanDirectorReviewablePlanV0(request)
	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionRequestReviewV0)
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionStudyBeforeV0)
	for _, ref := range result.Plan.ContextRefs {
		if ref == "worktree_ref:" || ref == "branch_ref:" {
			t.Fatalf("ref operativa vacia filtrada incorrectamente: %v", result.Plan.ContextRefs)
		}
	}
}

func TestBuildHumanDirectorReviewablePlanV0NormalizaAliasesYGlobsSeguros(t *testing.T) {
	request := validHumanDirectorWorkIntakeRequestForTestV0()
	request.Hints.Areas = []string{"web_application", "api_rest"}
	request.Hints.WriteSet = []string{"modulos/orquesta-app-director-intake/**"}

	result := BuildHumanDirectorReviewablePlanV0(request)
	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if result.Plan.Steps[0].WriteSet[0] != "modulos/orquesta-app-director-intake" {
		t.Fatalf("glob no normalizado: %+v", result.Plan.Steps[0].WriteSet)
	}
	if result.Plan.ContextRefs[0] == "" {
		t.Fatalf("context_refs vacias")
	}
}

func TestBuildHumanDirectorReviewablePlanV0PideRevisionPorWriteSetInseguro(t *testing.T) {
	request := validHumanDirectorWorkIntakeRequestForTestV0()
	request.Hints.WriteSet = []string{"/tmp/fuera", "modulos/orquesta-app-director-intake"}

	result := BuildHumanDirectorReviewablePlanV0(request)
	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionRequestReviewV0)
	assertHumanDirectorPlanHasActionV0(t, result.Plan, HumanDirectorPlanActionExecuteNowV0)
}

func validHumanDirectorWorkIntakeRequestForTestV0() HumanDirectorWorkIntakeRequestV0 {
	return HumanDirectorWorkIntakeRequestV0{
		SchemaVersion: HumanDirectorWorkIntakeSchemaVersionV0,
		RequestRef:    "request-ref-human-work-001",
		ProjectRef:    "project-ref-orquesta",
		WorktreeRef:   "worktree-ref-orquesta-main-20260523",
		BranchRef:     "branch-ref-trabajo-plataforma-agentes",
		OccurredAt:    "2026-05-23T12:00:00Z",
		Request: HumanDirectorWorkRequestV0{
			Title:              "Entrada humana a plan del director",
			Objective:          "Crear contrato neutral de intake humano revisable por director.",
			AcceptanceCriteria: []string{"salida distingue acciones operativas"},
			RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-app-director-intake"},
			CompactRules:       []string{"comunicacion compacta"},
		},
		Rules: []string{"no preparar trabajo de codigo antes del plan revisable"},
		Limits: HumanDirectorWorkLimitsV0{
			MaxAgents:          6,
			MaxDepth:           2,
			MaxFanout:          4,
			MaxSteps:           6,
			MaxWriteSetEntries: 5,
		},
		Hints: HumanDirectorWorkHintsV0{
			Areas:           []string{"web_application"},
			WriteSet:        []string{"modulos/orquesta-app-director-intake"},
			OpaqueRefs:      []string{"task-ref-intake-human-work-001"},
			CanRepairSafely: []string{"normalizar aliases seguros"},
		},
		ContextRefs: []string{"source_task_ref:task-ref-intake-human-work-001"},
	}
}

func assertHumanDirectorPlanHasActionV0(
	t *testing.T,
	plan HumanDirectorReviewablePlanV0,
	action string,
) {
	t.Helper()
	for _, step := range plan.Steps {
		if step.Action == action {
			return
		}
	}
	t.Fatalf("accion %s no encontrada en %+v", action, plan.Steps)
}

func assertHumanDirectorContextRefV0(t *testing.T, refs []string, expected string) {
	t.Helper()
	for _, ref := range refs {
		if ref == expected {
			return
		}
	}
	t.Fatalf("context_ref %s no encontrada en %v", expected, refs)
}

func humanDirectorPlanIssuesContainCodeV0(issues []HumanDirectorPlanIssueV0, expected string) bool {
	for _, issue := range issues {
		if issue.Code == expected {
			return true
		}
	}
	return false
}
