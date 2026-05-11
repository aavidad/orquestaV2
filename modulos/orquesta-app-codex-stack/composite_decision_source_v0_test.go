package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"testing"

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
