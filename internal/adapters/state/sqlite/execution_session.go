package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

var _ application.ExecutionSessionAuthoritySource = (*Repository)(nil)

// ResolveExecution structurally satisfies commands.ExecutionAuthorityResolver
// without importing the commands layer.
func (repository *Repository) ResolveExecution(
	ctx context.Context,
	principal identity.Principal,
	projectRef goal.ProjectRef,
	executionRef goal.ExecutionRef,
) (goal.ExecutionRef, error) {
	if ctx == nil || identity.ValidatePrincipal(principal) != nil ||
		principal.Kind != identity.PrincipalKindService || projectRef.String() == "" ||
		executionRef.String() == "" {
		return goal.ExecutionRef{}, stateError(application.StateNotFound, errors.New("sqlite.execution_authority_not_found"))
	}
	authority, err := repository.executionSessionAuthority(ctx, executionRef, principal.Method, projectRef)
	if err != nil || authority.ServicePrincipal != principal ||
		authority.Request.ProjectRef != projectRef || authority.Request.ExecutionRef != executionRef {
		return goal.ExecutionRef{}, stateError(application.StateNotFound, errors.New("sqlite.execution_authority_not_found"))
	}
	return executionRef, nil
}

func (repository *Repository) ExecutionSessionAuthority(
	ctx context.Context,
	executionRef goal.ExecutionRef,
	authenticationMethod string,
) (ports.ExecutionSessionAuthority, error) {
	return repository.executionSessionAuthority(ctx, executionRef, authenticationMethod, goal.ProjectRef{})
}

func (repository *Repository) executionSessionAuthority(
	ctx context.Context,
	executionRef goal.ExecutionRef,
	authenticationMethod string,
	expectedProject goal.ProjectRef,
) (ports.ExecutionSessionAuthority, error) {
	if ctx == nil || executionRef.String() == "" || authenticationMethod == "" {
		return ports.ExecutionSessionAuthority{}, stateError(application.StateNotFound, errors.New("sqlite.execution_authority_not_found"))
	}
	transaction, err := beginReadTransaction(ctx, repository)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, err
	}
	defer func() { _ = transaction.Rollback() }()

	var projectValue, goalValue, itemValue, executionValue, replacementValue, specHash string
	var attempt, planGeneration, appSpecGeneration int64
	err = transaction.QueryRowContext(ctx, `
SELECT g.project_ref,e.goal_ref,e.work_item_ref,e.ref,COALESCE(e.replaces_execution_ref,''),
       e.attempt_no,e.plan_generation,e.app_spec_generation,e.spec_hash
FROM executions e
JOIN goals g ON g.ref=e.goal_ref
JOIN work_items w ON w.goal_ref=e.goal_ref AND w.ref=e.work_item_ref
WHERE e.ref=? AND g.state='running' AND
 (e.state IN ('dispatching','running') OR
  (e.state='succeeded' AND EXISTS (
    SELECT 1 FROM outbox action
    WHERE action.goal_ref=e.goal_ref AND action.execution_ref=e.ref
      AND action.kind='admit_mailbox' AND action.completed_at IS NULL
      AND action.retired_at IS NULL AND action.quarantined_at IS NULL)))
  AND NOT EXISTS (
    SELECT 1 FROM executions successor
    WHERE successor.replaces_execution_ref=e.ref
  )`,
		executionRef.String(),
	).Scan(
		&projectValue, &goalValue, &itemValue, &executionValue, &replacementValue,
		&attempt, &planGeneration, &appSpecGeneration, &specHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.ExecutionSessionAuthority{}, stateError(application.StateNotFound, err)
	}
	if err != nil {
		return ports.ExecutionSessionAuthority{}, mapDatabaseError(err)
	}
	projectRef, err := goal.NewProjectRef(projectValue)
	if err != nil || expectedProject.String() != "" && projectRef != expectedProject {
		return ports.ExecutionSessionAuthority{}, stateError(application.StateNotFound, errors.New("sqlite.execution_authority_not_found"))
	}
	goalRef, goalErr := goal.NewGoalRef(goalValue)
	workItemRef, itemErr := goal.NewWorkItemRef(itemValue)
	persistedExecutionRef, executionErr := goal.NewExecutionRef(executionValue)
	var replacementRef goal.ExecutionRef
	var replacementErr error
	if replacementValue != "" {
		replacementRef, replacementErr = goal.NewExecutionRef(replacementValue)
	}
	if goalErr != nil || itemErr != nil || executionErr != nil || replacementErr != nil ||
		attempt <= 0 || planGeneration <= 0 || appSpecGeneration <= 0 {
		return ports.ExecutionSessionAuthority{}, invalid(errors.New("sqlite.execution_authority_invalid"))
	}
	request := ports.ExecutionSessionEnsureRequest{
		ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: workItemRef,
		ExecutionRef: persistedExecutionRef, ExecutionAttempt: uint64(attempt),
		ReplacesExecutionRef: replacementRef, PlanGeneration: goal.PlanGeneration(planGeneration),
		AppSpecGeneration: goal.AppSpecGeneration(appSpecGeneration), SpecHash: specHash,
	}
	authority, err := application.DeriveExecutionSessionAuthority(request, authenticationMethod)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, invalid(errors.New("sqlite.execution_authority_invalid"))
	}
	if err := commit(transaction); err != nil {
		return ports.ExecutionSessionAuthority{}, err
	}
	return authority, nil
}
