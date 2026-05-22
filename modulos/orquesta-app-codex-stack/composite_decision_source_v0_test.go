package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
)

func TestCompositeDirectorDecisionSourceV0AceptaPlanGoConBootstrap(t *testing.T) {
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{
				Decisions: codexStackDirectorDecisionsForTestV0(
					"run-ref-stack-policy-001",
					"brainstorm-ref-stack-policy-001",
				),
			},
		},
	}

	decisions, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("ListDirectorAgentDecisionsV0: %v", err)
	}
	if len(decisions) == 0 {
		t.Fatalf("decisions vacias")
	}
}

func TestCompositeDirectorDecisionSourceV0RechazaVoteRefIncoherente(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-vote-ref-001",
		"brainstorm-ref-stack-policy-vote-ref-001",
	)
	for i := range decisions {
		if decisions[i].AcceptDecision != nil {
			decisions[i].AcceptDecision.VoteRef = "vote-ref-no-publicado-001"
		}
	}
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err == nil || !strings.Contains(err.Error(), "accept_decision.vote_ref") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompositeDirectorDecisionSourceV0RechazaPlanGoSinGoMod(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-002",
		"brainstorm-ref-stack-policy-002",
	)
	decisions = codexStackMutateFirstMicrotaskForPolicyTestV0(
		decisions,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.WriteSet = []string{"cmd/server", "internal/agenda"}
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err == nil || !strings.Contains(err.Error(), "go.mod") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompositeDirectorDecisionSourceV0RechazaPlanGoSinCmd(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-003",
		"brainstorm-ref-stack-policy-003",
	)
	decisions = codexStackMutateFirstMicrotaskForPolicyTestV0(
		decisions,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.WriteSet = []string{"go.mod", "internal/agenda"}
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err == nil || !strings.Contains(err.Error(), "cmd/server") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompositeDirectorDecisionSourceV0NoPermiteAnswerQuestionTaparPlanInvalido(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-004",
		"brainstorm-ref-stack-policy-004",
	)
	decisions = codexStackMutateFirstMicrotaskForPolicyTestV0(
		decisions,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.WriteSet = []string{"internal/agenda"}
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	decisions = append([]orquestadirectoragent.DirectorAgentDecisionV0{
		codexStackAnswerQuestionDecisionForPolicyTestV0("run-ref-stack-policy-004"),
	}, decisions...)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err == nil || !strings.Contains(err.Error(), "go.mod") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompositeDirectorDecisionSourceV0CompletaDependsOnBootstrap(t *testing.T) {
	decisions := codexStackBatchDirectorDecisionsForTestV0(
		"run-ref-stack-policy-dep-001",
		"brainstorm-ref-stack-policy-dep-001",
	)
	decisions = codexStackMutateTaskForPolicyTestV0(
		decisions,
		"task-ref-stack-agenda-domain",
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.DependsOn = nil
		},
	)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	task, ok := codexStackMicrotaskForPolicyTestV0(got, "task-ref-stack-agenda-domain")
	if !ok || !codexStackStringInSetForTestV0(task.DependsOn, "task-ref-stack-agenda-bootstrap") {
		t.Fatalf("depends_on no completado: %+v", task)
	}
}

func TestCompositeDirectorDecisionSourceV0NormalizaOpenPhaseConPhaseDestino(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-open-phase-001",
		"brainstorm-ref-stack-policy-open-phase-001",
	)
	for i := range decisions {
		if decisions[i].OpenPhase != nil {
			decisions[i].PhaseID = decisions[i].OpenPhase.PhaseID
		}
	}
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{
			Run: orquestacoreworkflow.OrchestrationRunV0{
				CurrentPhase: orquestacoreworkflow.OrchestrationPhaseBrainstormingArquitecturaV0,
			},
		},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	assertCodexStackOpenPhaseForPolicyTestV0(
		t,
		got,
		"director-decision-stack-open-plan-001",
		orquestacoreworkflow.OrchestrationPhaseVotacionYDecisionV0,
	)
	assertCodexStackOpenPhaseForPolicyTestV0(
		t,
		got,
		"director-decision-stack-open-program-001",
		orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0,
	)
}

func TestCompositeDirectorDecisionSourceV0NormalizaPublishContractDecisionRefAlAcceptPrevio(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-contract-ref-001",
		"brainstorm-ref-stack-policy-contract-ref-001",
	)
	for i := range decisions {
		if decisions[i].PublishContract != nil {
			decisions[i].PublishContract.DecisionRef = decisions[i].DecisionRef
		}
	}
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	contract, ok := codexStackPublishContractForPolicyTestV0(got, "director-decision-stack-contract-001")
	if !ok || contract.DecisionRef != "decision-ref-stack-001" {
		t.Fatalf("publish_function_contract.decision_ref no normalizado: %+v", contract)
	}
}

func TestCompositeDirectorDecisionSourceV0NoInventaPublishContractSinAcceptPrevio(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-contract-no-accept-001",
		"brainstorm-ref-stack-policy-contract-no-accept-001",
	)
	filtered := make([]orquestadirectoragent.DirectorAgentDecisionV0, 0, len(decisions))
	for _, decision := range decisions {
		if decision.AcceptDecision != nil {
			continue
		}
		if decision.PublishContract != nil {
			decision.PublishContract.DecisionRef = decision.DecisionRef
		}
		filtered = append(filtered, decision)
	}
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: filtered},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err == nil || !strings.Contains(err.Error(), "publish_function_contract.decision_ref") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompositeDirectorDecisionSourceV0CompletaReadmeParaAppGoCompleta(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-readme-001",
		"brainstorm-ref-stack-policy-readme-001",
	)
	decisions = codexStackMutateFirstMicrotaskForPolicyTestV0(
		decisions,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.WriteSet = []string{"go.mod", "cmd/server", "internal", "web", "docs"}
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	task, ok := codexStackMicrotaskForPolicyTestV0(got, "task-ref-stack-agenda-001")
	if !ok || !codexStackStringInSetForTestV0(task.WriteSet, "README.md") {
		t.Fatalf("README.md no completado en write_set: %+v", task)
	}
}

func TestCompositeDirectorDecisionSourceV0TraduceWriteSetRaizDeAppGoCompleta(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-root-001",
		"brainstorm-ref-stack-policy-root-001",
	)
	decisions = codexStackMutateFirstMicrotaskForPolicyTestV0(
		decisions,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.Summary = "Construir modulo Go autonomo con HTTP, web y pruebas."
			task.WriteSet = []string{"."}
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	task, ok := codexStackMicrotaskForPolicyTestV0(got, "task-ref-stack-agenda-001")
	if !ok {
		t.Fatalf("task no encontrada")
	}
	for _, want := range []string{"go.mod", "cmd/server", "internal", "web", "README.md"} {
		if !codexStackStringInSetForTestV0(task.WriteSet, want) {
			t.Fatalf("write_set no contiene %s: %+v", want, task.WriteSet)
		}
	}
	if codexStackStringInSetForTestV0(task.WriteSet, ".") {
		t.Fatalf("write_set raiz no traducido: %+v", task.WriteSet)
	}
}

func TestCompositeDirectorDecisionSourceV0CompletaWebSiLaAppGoLoPide(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-web-001",
		"brainstorm-ref-stack-policy-web-001",
	)
	decisions = codexStackMutateFirstMicrotaskForPolicyTestV0(
		decisions,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.Summary = "Construir modulo Go autonomo con API y web."
			task.WriteSet = []string{"go.mod", "cmd/server/**", "internal/**", "README.md", "docs/**"}
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	task, ok := codexStackMicrotaskForPolicyTestV0(got, "task-ref-stack-agenda-001")
	if !ok || !codexStackStringInSetForTestV0(task.WriteSet, "web") {
		t.Fatalf("web no completado en write_set: %+v", task)
	}
}

func TestCompositeDirectorDecisionSourceV0NoDuplicaWebSiWebAdminInternoLoCubre(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-webadmin-001",
		"brainstorm-ref-stack-policy-webadmin-001",
	)
	decisions = codexStackMutateFirstMicrotaskForPolicyTestV0(
		decisions,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.Summary = "Construir modulo Go autonomo con API y web."
			task.WriteSet = []string{
				"go.mod",
				"cmd/server/main.go",
				"internal/domain/**",
				"internal/httpapi/**",
				"internal/webadmin/**",
				"README.md",
				"docs/**",
			}
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	task, ok := codexStackMicrotaskForPolicyTestV0(got, "task-ref-stack-agenda-001")
	if !ok {
		t.Fatalf("task no encontrada")
	}
	if codexStackStringInSetForTestV0(task.WriteSet, "web") {
		t.Fatalf("web duplicado pese a internal/webadmin: %+v", task.WriteSet)
	}
}

func TestCompositeDirectorDecisionSourceV0NoRompeWriteSetMaximoPorCompletarWeb(t *testing.T) {
	decisions := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-web-max-001",
		"brainstorm-ref-stack-policy-web-max-001",
	)
	decisions = codexStackMutateFirstMicrotaskForPolicyTestV0(
		decisions,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.Summary = "Construir modulo Go autonomo con API y web."
			task.WriteSet = []string{
				"go.mod",
				"cmd/server/main.go",
				"internal/domain/**",
				"internal/app/**",
				"internal/ports/**",
				"internal/http/**",
				"internal/memory/**",
				"internal/i18n/**",
				"README.md",
				"docs/**",
			}
			for len(task.WriteSet) < compositeDirectorAgentTextListMaxV0 {
				task.WriteSet = append(task.WriteSet, fmt.Sprintf("docs/extra-%02d.md", len(task.WriteSet)))
			}
			task.AcceptanceCriteria = append(task.AcceptanceCriteria,
				"Web de administracion servida por el proceso y conectada a la API.",
			)
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: decisions},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	task, ok := codexStackMicrotaskForPolicyTestV0(got, "task-ref-stack-agenda-001")
	if !ok {
		t.Fatalf("task no encontrada")
	}
	if len(task.WriteSet) != compositeDirectorAgentTextListMaxV0 ||
		codexStackStringInSetForTestV0(task.WriteSet, "web") {
		t.Fatalf("write_set no debe ampliarse hasta invalidarse: %+v", task.WriteSet)
	}
}

func TestCompositeDirectorDecisionSourceV0NoDuplicaReadmeSiOtraTareaLoCubre(t *testing.T) {
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{
				Decisions: codexStackBatchDirectorDecisionsForTestV0(
					"run-ref-stack-policy-readme-002",
					"brainstorm-ref-stack-policy-readme-002",
				),
			},
		},
	}

	got, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err != nil {
		t.Fatalf("err=%v", err)
	}
	task, ok := codexStackMicrotaskForPolicyTestV0(got, "task-ref-stack-agenda-bootstrap")
	if !ok {
		t.Fatalf("bootstrap no encontrado")
	}
	if codexStackStringInSetForTestV0(task.WriteSet, "README.md") {
		t.Fatalf("README.md duplicado en bootstrap: %+v", task.WriteSet)
	}
}

func TestCompositeDirectorDecisionSourceV0NoPermiteQueCambioTapePlanInicialInvalido(t *testing.T) {
	initial := codexStackDirectorDecisionsForTestV0(
		"run-ref-stack-policy-005",
		"brainstorm-ref-stack-policy-005",
	)
	initial = codexStackMutateFirstMicrotaskForPolicyTestV0(
		initial,
		func(task *orquestadirectoragent.DirectorAgentMicrotaskV0) {
			task.WriteSet = []string{"internal/agenda"}
			task.RequiredTests = []string{"go test ./..."}
		},
	)
	change := []orquestadirectoragent.DirectorAgentDecisionV0{
		codexStackAnswerQuestionDecisionForPolicyTestV0("run-ref-stack-policy-005"),
	}
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Decisions: initial},
			codexStackStaticDecisionSourceForTestV0{Decisions: change},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err == nil || !strings.Contains(err.Error(), "go.mod") {
		t.Fatalf("err=%v", err)
	}
}

func TestCompositeDirectorDecisionSourceV0PropagaErrorDeFuente(t *testing.T) {
	source := compositeDirectorDecisionSourceV0{
		Sources: []orquestadirectoragentworkflow.DirectorAgentDecisionSourcePortV0{
			codexStackStaticDecisionSourceForTestV0{Err: fmt.Errorf("fuente rota")},
		},
	}

	_, err := source.ListDirectorAgentDecisionsV0(
		context.Background(),
		orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0{},
	)

	if err == nil || !strings.Contains(err.Error(), "fuente rota") {
		t.Fatalf("err=%v", err)
	}
}

func assertCodexStackOpenPhaseForPolicyTestV0(
	t *testing.T,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
	decisionRef string,
	wantPhase orquestacoreworkflow.OrchestrationPhaseIDV0,
) {
	t.Helper()
	for _, decision := range decisions {
		if decision.DecisionRef != decisionRef {
			continue
		}
		if decision.OpenPhase == nil {
			t.Fatalf("%s no es open_phase: %+v", decisionRef, decision)
		}
		if decision.PhaseID != string(wantPhase) {
			t.Fatalf("%s phase_id=%q want %q", decisionRef, decision.PhaseID, wantPhase)
		}
		return
	}
	t.Fatalf("decision %s no encontrada", decisionRef)
}

func codexStackPublishContractForPolicyTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
	decisionRef string,
) (orquestadirectoragent.DirectorAgentPublishContractCommandV0, bool) {
	for _, decision := range decisions {
		if decision.DecisionRef != decisionRef || decision.PublishContract == nil {
			continue
		}
		return *decision.PublishContract, true
	}
	return orquestadirectoragent.DirectorAgentPublishContractCommandV0{}, false
}

func codexStackAnswerQuestionDecisionForPolicyTestV0(
	runRef string,
) orquestadirectoragent.DirectorAgentDecisionV0 {
	return orquestadirectoragent.DirectorAgentDecisionV0{
		SchemaVersion: orquestadirectoragent.DirectorAgentDecisionSchemaVersionV0,
		DecisionRef:   "director-decision-answer-policy-001",
		RunID:         runRef,
		PhaseID:       "programacion",
		CommandType:   orquestadirectoragent.DirectorAgentCommandAnswerQuestionV0,
		CommandRef:    "command-ref-answer-policy-001",
		Summary:       "Responder cambio aceptado.",
		EvidenceRefs:  []string{"evidence-ref-answer-policy-001"},
		AnswerQuestion: &orquestadirectoragent.DirectorAgentAnswerQuestionCommandV0{
			AnswerID:     "answer-ref-policy-001",
			QuestionID:   "question-ref-policy-001",
			Decision:     orquestadirectoragent.DirectorAgentAnswerReplanV0,
			Summary:      "Aceptar cambio como replanificacion.",
			EvidenceRefs: []string{"evidence-ref-answer-policy-001"},
		},
	}
}

type codexStackStaticDecisionSourceForTestV0 struct {
	Decisions []orquestadirectoragent.DirectorAgentDecisionV0
	Err       error
}

func (source codexStackStaticDecisionSourceForTestV0) ListDirectorAgentDecisionsV0(
	ctx context.Context,
	request orquestadirectoragentworkflow.DirectorAgentDecisionSourceRequestV0,
) ([]orquestadirectoragent.DirectorAgentDecisionV0, error) {
	if source.Err != nil {
		return nil, source.Err
	}
	return append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), source.Decisions...), nil
}

func codexStackMutateFirstMicrotaskForPolicyTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
	mutate func(*orquestadirectoragent.DirectorAgentMicrotaskV0),
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	out := append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), decisions...)
	for i := range out {
		if out[i].CreateMicrotask == nil {
			continue
		}
		mutate(&out[i].CreateMicrotask.Task)
		return out
	}
	return out
}

func codexStackMutateTaskForPolicyTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
	taskRef string,
	mutate func(*orquestadirectoragent.DirectorAgentMicrotaskV0),
) []orquestadirectoragent.DirectorAgentDecisionV0 {
	out := append([]orquestadirectoragent.DirectorAgentDecisionV0(nil), decisions...)
	for i := range out {
		if out[i].CreateMicrotask == nil ||
			out[i].CreateMicrotask.Task.TaskID != taskRef {
			continue
		}
		mutate(&out[i].CreateMicrotask.Task)
		return out
	}
	return out
}

func codexStackMicrotaskForPolicyTestV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
	taskID string,
) (orquestadirectoragent.DirectorAgentMicrotaskV0, bool) {
	for _, decision := range decisions {
		if decision.CreateMicrotask == nil {
			continue
		}
		task := decision.CreateMicrotask.Task
		if task.TaskID == taskID {
			return task, true
		}
	}
	return orquestadirectoragent.DirectorAgentMicrotaskV0{}, false
}
