package orquestacionnucleoapp

import (
	"context"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func reviewReworkValidateRecursiveTreeBudgetsV0(
	ctx context.Context,
	request SchedulerCandidateRequestV0,
	store WorkflowTaskStorePortV0,
	treeStore WorkflowTaskByParentStorePortV0,
	parentsByRef map[string]orquestacoreworkflow.WorkflowTaskV0,
	tasks []orquestacoreworkflow.WorkflowTaskV0,
) error {
	budgetsByRoot := map[string]reviewReworkRecursiveBudgetV0{}
	for i := range tasks {
		parentRef := strings.TrimSpace(tasks[i].ParentTaskRef)
		if parentRef == "" {
			continue
		}
		parent := parentsByRef[parentRef]
		budget, err := reviewReworkRecursiveBudgetForParentV0(ctx, request.Run.RunID, store, treeStore, parent, tasks)
		if err != nil {
			return err
		}
		if budget.MaxAgents > 0 {
			tasks[i].MaxRecursiveAgents = budget.MaxAgents
		}
		if existing, ok := budgetsByRoot[budget.RootTaskRef]; ok && existing.CandidateCount >= budget.CandidateCount {
			continue
		}
		budgetsByRoot[budget.RootTaskRef] = budget
	}
	for _, budget := range budgetsByRoot {
		if budget.MaxAgents > 0 && budget.Total() > budget.MaxAgents {
			return errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.max_recursive_agents", "arbol supera max_recursive_agents")
		}
	}
	return nil
}

type reviewReworkRecursiveBudgetV0 struct {
	RootTaskRef    string
	MaxAgents      int
	PersistedCount int
	CandidateCount int
}

func (budget reviewReworkRecursiveBudgetV0) Total() int {
	return budget.PersistedCount + budget.CandidateCount
}

func reviewReworkRecursiveBudgetForParentV0(
	ctx context.Context,
	runRef string,
	store WorkflowTaskStorePortV0,
	treeStore WorkflowTaskByParentStorePortV0,
	parent orquestacoreworkflow.WorkflowTaskV0,
	candidates []orquestacoreworkflow.WorkflowTaskV0,
) (reviewReworkRecursiveBudgetV0, error) {
	root, maxAgents, err := reviewReworkRecursiveRootAndBudgetV0(ctx, runRef, store, parent)
	if err != nil {
		return reviewReworkRecursiveBudgetV0{}, err
	}
	persistedRefs, err := reviewReworkRecursivePersistedSubtreeRefsV0(ctx, runRef, treeStore, root.TaskID)
	if err != nil {
		return reviewReworkRecursiveBudgetV0{}, err
	}
	return reviewReworkRecursiveBudgetV0{
		RootTaskRef:    root.TaskID,
		MaxAgents:      maxAgents,
		PersistedCount: len(persistedRefs),
		CandidateCount: reviewReworkRecursiveCandidateSubtreeCountV0(persistedRefs, candidates),
	}, nil
}

func reviewReworkRecursiveRootAndBudgetV0(
	ctx context.Context,
	runRef string,
	store WorkflowTaskStorePortV0,
	task orquestacoreworkflow.WorkflowTaskV0,
) (orquestacoreworkflow.WorkflowTaskV0, int, error) {
	seen := map[string]bool{}
	current := task
	budget := 0
	for {
		currentRef := strings.TrimSpace(current.TaskID)
		if currentRef == "" || seen[currentRef] {
			return orquestacoreworkflow.WorkflowTaskV0{}, 0, errorV0(ErrNucleoOrquestacionInvalidoV0, "split_task.parent_task_ref", "linaje recursivo ciclico")
		}
		seen[currentRef] = true
		if current.MaxRecursiveAgents > 0 && (budget == 0 || current.MaxRecursiveAgents < budget) {
			budget = current.MaxRecursiveAgents
		}
		parentRef := strings.TrimSpace(current.ParentTaskRef)
		if parentRef == "" {
			return current, budget, nil
		}
		parents, err := store.LoadWorkflowTasksV0(ctx, runRef, []string{parentRef})
		if err != nil {
			return current, budget, nil
		}
		current = parents[0]
	}
}

func reviewReworkRecursivePersistedSubtreeRefsV0(
	ctx context.Context,
	runRef string,
	treeStore WorkflowTaskByParentStorePortV0,
	rootTaskRef string,
) (map[string]bool, error) {
	seen := map[string]bool{strings.TrimSpace(rootTaskRef): true}
	queue := []string{strings.TrimSpace(rootTaskRef)}
	for len(queue) > 0 {
		parentRef := queue[0]
		queue = queue[1:]
		children, err := treeStore.LoadWorkflowTasksByParentV0(ctx, runRef, parentRef)
		if err != nil {
			return nil, err
		}
		for _, child := range children {
			childRef := strings.TrimSpace(child.TaskID)
			if childRef == "" || seen[childRef] {
				continue
			}
			seen[childRef] = true
			queue = append(queue, childRef)
		}
	}
	return seen, nil
}

func reviewReworkRecursiveCandidateSubtreeCountV0(
	persistedRefs map[string]bool,
	candidates []orquestacoreworkflow.WorkflowTaskV0,
) int {
	candidateChildren := map[string][]string{}
	for _, candidate := range candidates {
		parentRef := strings.TrimSpace(candidate.ParentTaskRef)
		taskRef := strings.TrimSpace(candidate.TaskID)
		if parentRef == "" || taskRef == "" {
			continue
		}
		candidateChildren[parentRef] = append(candidateChildren[parentRef], taskRef)
	}
	seenCandidates := map[string]bool{}
	queue := make([]string, 0, len(persistedRefs))
	for ref := range persistedRefs {
		queue = append(queue, ref)
	}
	for len(queue) > 0 {
		parentRef := queue[0]
		queue = queue[1:]
		for _, childRef := range candidateChildren[parentRef] {
			if seenCandidates[childRef] {
				continue
			}
			seenCandidates[childRef] = true
			queue = append(queue, childRef)
		}
	}
	return len(seenCandidates)
}
