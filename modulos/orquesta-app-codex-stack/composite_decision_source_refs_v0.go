package orquestaappcodexstack

import (
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func validateCompositeDirectorDecisionRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) error {
	votes := compositeStringSetV0(run.Votes)
	accepted := compositeStringSetV0(run.Decisions)
	answers := compositeStringSetV0(run.DirectorAnswers)
	contracts := compositeStringSetV0(run.FunctionContracts)
	for _, decision := range decisions {
		switch {
		case decision.AnswerQuestion != nil:
			compositeAddRefV0(answers, decision.AnswerQuestion.AnswerID)
		case decision.RequestVote != nil:
			compositeAddRefV0(votes, decision.RequestVote.VoteRequestID)
		case decision.AcceptDecision != nil:
			if !votes[strings.TrimSpace(decision.AcceptDecision.VoteRef)] {
				return fmt.Errorf("director_decisions invalidas: accept_decision.vote_ref sin request_vote previo")
			}
			compositeAddRefV0(accepted, decision.AcceptDecision.DecisionRef)
		case decision.PublishContract != nil:
			decisionRef := strings.TrimSpace(decision.PublishContract.DecisionRef)
			if !accepted[decisionRef] && !answers[decisionRef] {
				return fmt.Errorf("director_decisions invalidas: publish_function_contract.decision_ref sin accept_decision previo")
			}
			compositeAddRefV0(contracts, decision.PublishContract.ContractRef)
		case decision.CreateMicrotask != nil:
			if err := validateCompositeMicrotaskContractRefsV0(contracts, decision.CreateMicrotask.Task); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCompositeMicrotaskContractRefsV0(
	contracts map[string]bool,
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
) error {
	for _, ref := range task.FunctionContractRefs {
		if !contracts[strings.TrimSpace(ref.ContractRef)] {
			return fmt.Errorf("director_decisions invalidas: create_microtask.function_contract_refs sin publish_function_contract previo")
		}
	}
	return nil
}

func compositeProgrammingMicrotasksV0(
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) []orquestadirectoragent.DirectorAgentMicrotaskV0 {
	tasks := []orquestadirectoragent.DirectorAgentMicrotaskV0{}
	for _, decision := range decisions {
		if decision.CommandType != orquestadirectoragent.DirectorAgentCommandCreateMicrotaskV0 ||
			decision.CreateMicrotask == nil {
			continue
		}
		task := decision.CreateMicrotask.Task
		if strings.TrimSpace(task.PhaseID) != string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0) {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func compositeStringSetV0(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		compositeAddRefV0(out, value)
	}
	return out
}

func compositeAddRefV0(values map[string]bool, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		values[value] = true
	}
}
