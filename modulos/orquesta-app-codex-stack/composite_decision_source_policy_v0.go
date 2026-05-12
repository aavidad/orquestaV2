package orquestaappcodexstack

import (
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
)

func validateCompositeDirectorDecisionBatchV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	requestKind string,
	objectiveHints []string,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) error {
	if err := validateCompositeDirectorDecisionRefsV0(run, decisions); err != nil {
		return err
	}
	tasks := compositeProgrammingMicrotasksV0(decisions)
	if err := validateCompositeProgrammingMicrotaskAnchorsV0(
		requestKind,
		objectiveHints,
		tasks,
	); err != nil {
		return err
	}
	if len(tasks) == 0 || !compositeLooksLikeGoAppPlanV0(tasks) {
		return nil
	}
	if !compositeTasksRequireGoTestAllV0(tasks) {
		return fmt.Errorf("director_decisions invalidas: app Go sin required_tests go test ./...")
	}
	if !compositeTasksCoverWriteSetV0(tasks, "go.mod") {
		return fmt.Errorf("director_decisions invalidas: app Go sin microtarea para go.mod")
	}
	if !compositeTasksCoverCmdEntrypointV0(tasks) {
		return fmt.Errorf("director_decisions invalidas: app Go sin microtarea para cmd/server")
	}
	if !compositeTasksDependOnBootstrapV0(tasks) {
		return fmt.Errorf("director_decisions invalidas: app Go sin depends_on hacia bootstrap")
	}
	return nil
}

func validateCompositeDirectorDecisionRefsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	decisions []orquestadirectoragent.DirectorAgentDecisionV0,
) error {
	votes := compositeStringSetV0(run.Votes)
	accepted := compositeStringSetV0(run.Decisions)
	contracts := compositeStringSetV0(run.FunctionContracts)
	for _, decision := range decisions {
		switch {
		case decision.RequestVote != nil:
			compositeAddRefV0(votes, decision.RequestVote.VoteRequestID)
		case decision.AcceptDecision != nil:
			if !votes[strings.TrimSpace(decision.AcceptDecision.VoteRef)] {
				return fmt.Errorf("director_decisions invalidas: accept_decision.vote_ref sin request_vote previo")
			}
			compositeAddRefV0(accepted, decision.AcceptDecision.DecisionRef)
		case decision.PublishContract != nil:
			if !accepted[strings.TrimSpace(decision.PublishContract.DecisionRef)] {
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

func compositeLooksLikeGoAppPlanV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	for _, task := range tasks {
		if compositeTaskMentionsAnyV0(task, "go test", "go.mod", "cmd/server") {
			return true
		}
		for _, path := range task.WriteSet {
			normalized := compositeNormalizePathTokenV0(path)
			if normalized == "go.mod" ||
				strings.HasPrefix(normalized, "cmd/") ||
				strings.HasPrefix(normalized, "internal/") {
				return true
			}
		}
	}
	return false
}

func compositeTasksRequireGoTestAllV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	for _, task := range tasks {
		for _, test := range task.RequiredTests {
			if strings.Contains(strings.ToLower(strings.TrimSpace(test)), "go test ./...") {
				return true
			}
		}
	}
	return false
}

func compositeTasksCoverWriteSetV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
	want string,
) bool {
	want = compositeNormalizePathTokenV0(want)
	for _, task := range tasks {
		for _, path := range task.WriteSet {
			if compositeNormalizePathTokenV0(path) == want {
				return true
			}
		}
	}
	return false
}

func compositeTasksCoverCmdEntrypointV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	for _, task := range tasks {
		for _, path := range task.WriteSet {
			normalized := compositeNormalizePathTokenV0(path)
			if normalized == "cmd" ||
				normalized == "cmd/**" ||
				normalized == "cmd/server" ||
				strings.HasPrefix(normalized, "cmd/") {
				return true
			}
		}
	}
	return false
}

func compositeTasksDependOnBootstrapV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) bool {
	bootstrapIDs := compositeBootstrapTaskIDsV0(tasks)
	if len(bootstrapIDs) == 0 {
		return false
	}
	for _, task := range tasks {
		if compositeTaskCoversGoModV0(task) || !compositeTaskNeedsGoBootstrapV0(task) {
			continue
		}
		if !compositeTaskDependsOnAnyV0(task, bootstrapIDs) {
			return false
		}
	}
	return true
}

func compositeBootstrapTaskIDsV0(
	tasks []orquestadirectoragent.DirectorAgentMicrotaskV0,
) map[string]bool {
	ids := map[string]bool{}
	for _, task := range tasks {
		if compositeTaskCoversGoModV0(task) {
			ids[strings.TrimSpace(task.TaskID)] = true
		}
	}
	return ids
}

func compositeTaskCoversGoModV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	for _, path := range task.WriteSet {
		if compositeNormalizePathTokenV0(path) == "go.mod" {
			return true
		}
	}
	return false
}

func compositeTaskNeedsGoBootstrapV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	if compositeTaskRequiresGoTestAllV0(task) {
		return true
	}
	for _, path := range task.WriteSet {
		normalized := compositeNormalizePathTokenV0(path)
		if strings.HasPrefix(normalized, "cmd/") ||
			strings.HasPrefix(normalized, "internal/") {
			return true
		}
	}
	return false
}

func compositeTaskRequiresGoTestAllV0(task orquestadirectoragent.DirectorAgentMicrotaskV0) bool {
	for _, test := range task.RequiredTests {
		if strings.Contains(strings.ToLower(strings.TrimSpace(test)), "go test ./...") {
			return true
		}
	}
	return false
}

func compositeTaskDependsOnAnyV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
	refs map[string]bool,
) bool {
	for _, dep := range task.DependsOn {
		if refs[strings.TrimSpace(dep)] {
			return true
		}
	}
	return false
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

func compositeTaskMentionsAnyV0(
	task orquestadirectoragent.DirectorAgentMicrotaskV0,
	needles ...string,
) bool {
	text := strings.ToLower(strings.Join(append(
		append(
			append([]string{task.Title, task.Summary}, task.AcceptanceCriteria...),
			task.RequiredTests...,
		),
		task.WriteSet...,
	), "\n"))
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func compositeNormalizePathTokenV0(path string) string {
	path = strings.TrimSpace(strings.ToLower(path))
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimSuffix(path, "/")
	return path
}
