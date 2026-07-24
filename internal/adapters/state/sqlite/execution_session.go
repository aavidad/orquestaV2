package sqlite

import (
	"context"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

var _ application.ExecutionSessionAuthoritySource = (*Repository)(nil)

type executionAuthorityMatch struct {
	authority ports.ExecutionSessionAuthority
	active    bool
}

// ClassifyExecutionPrincipal derives identity only from durable execution
// tuples. It deliberately scans inactive and foreign-project executions too:
// found and active are separate security facts.
func (repository *Repository) ClassifyExecutionPrincipal(
	ctx context.Context,
	principal identity.Principal,
	projectRef goal.ProjectRef,
) (bool, bool, goal.ExecutionRef, error) {
	if ctx == nil || identity.ValidatePrincipal(principal) != nil ||
		principal.Kind != identity.PrincipalKindService || projectRef.String() == "" {
		return false, false, goal.ExecutionRef{}, errors.New("sqlite.execution_principal_invalid")
	}
	match, found, err := repository.readExecutionAuthority(ctx, goal.ExecutionRef{}, principal)
	if err != nil || !found {
		return false, false, goal.ExecutionRef{}, err
	}
	return true, match.active && match.authority.Request.ProjectRef == projectRef,
		match.authority.Request.ExecutionRef, nil
}

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
	match, found, err := repository.readExecutionAuthority(ctx, executionRef, principal)
	if err != nil || !found || !match.active || match.authority.ServicePrincipal != principal ||
		match.authority.Request.ProjectRef != projectRef {
		return goal.ExecutionRef{}, stateError(application.StateNotFound, errors.New("sqlite.execution_authority_not_found"))
	}
	return executionRef, nil
}

func (repository *Repository) ExecutionSessionAuthority(
	ctx context.Context,
	executionRef goal.ExecutionRef,
	authenticationMethod string,
) (ports.ExecutionSessionAuthority, error) {
	if ctx == nil || executionRef.String() == "" || authenticationMethod == "" {
		return ports.ExecutionSessionAuthority{}, stateError(application.StateNotFound, errors.New("sqlite.execution_authority_not_found"))
	}
	match, found, err := repository.readExecutionAuthority(
		ctx, executionRef, identity.Principal{Method: authenticationMethod},
	)
	if err != nil {
		return ports.ExecutionSessionAuthority{}, err
	}
	if !found || !match.active {
		return ports.ExecutionSessionAuthority{}, stateError(application.StateNotFound, errors.New("sqlite.execution_authority_not_found"))
	}
	return match.authority, nil
}

func (repository *Repository) readExecutionAuthority(
	ctx context.Context,
	executionRef goal.ExecutionRef,
	principal identity.Principal,
) (executionAuthorityMatch, bool, error) {
	database, err := repository.database()
	if err != nil {
		return executionAuthorityMatch{}, false, err
	}
	return findExecutionAuthority(ctx, database, executionRef, principal)
}

func findExecutionAuthority(
	ctx context.Context,
	source queryer,
	executionRef goal.ExecutionRef,
	principal identity.Principal,
) (executionAuthorityMatch, bool, error) {
	rows, err := source.QueryContext(ctx, `
SELECT g.project_ref,e.goal_ref,e.work_item_ref,e.ref,COALESCE(e.replaces_execution_ref,''),
 e.attempt_no,e.plan_generation,e.app_spec_generation,e.spec_hash,
 CASE WHEN g.state='running' AND
  (e.state IN ('dispatching','running') OR (e.state='succeeded' AND EXISTS (
   SELECT 1 FROM outbox action WHERE action.goal_ref=e.goal_ref AND action.execution_ref=e.ref
    AND action.kind='admit_mailbox' AND action.completed_at IS NULL
    AND action.retired_at IS NULL AND action.quarantined_at IS NULL)))
  AND NOT EXISTS (SELECT 1 FROM executions successor WHERE successor.replaces_execution_ref=e.ref)
 THEN 1 ELSE 0 END
FROM executions e JOIN goals g ON g.ref=e.goal_ref
JOIN work_items w ON w.goal_ref=e.goal_ref AND w.ref=e.work_item_ref
WHERE (?='' OR e.ref=?)`, executionRef.String(), executionRef.String())
	if err != nil {
		return executionAuthorityMatch{}, false, mapDatabaseError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var projectValue, goalValue, itemValue, executionValue, replacementValue, specHash string
		var attempt, planGeneration, appSpecGeneration, active int64
		if err := rows.Scan(&projectValue, &goalValue, &itemValue, &executionValue, &replacementValue,
			&attempt, &planGeneration, &appSpecGeneration, &specHash, &active); err != nil {
			return executionAuthorityMatch{}, false, mapDatabaseError(err)
		}
		projectRef, projectErr := goal.NewProjectRef(projectValue)
		goalRef, goalErr := goal.NewGoalRef(goalValue)
		itemRef, itemErr := goal.NewWorkItemRef(itemValue)
		persistedRef, executionErr := goal.NewExecutionRef(executionValue)
		replacementRef, replacementErr := goal.ExecutionRef{}, error(nil)
		if replacementValue != "" {
			replacementRef, replacementErr = goal.NewExecutionRef(replacementValue)
		}
		request := ports.ExecutionSessionEnsureRequest{
			ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: itemRef, ExecutionRef: persistedRef,
			ExecutionAttempt: uint64(attempt), ReplacesExecutionRef: replacementRef,
			PlanGeneration:    goal.PlanGeneration(planGeneration),
			AppSpecGeneration: goal.AppSpecGeneration(appSpecGeneration), SpecHash: specHash,
		}
		authority, deriveErr := application.DeriveExecutionSessionAuthority(request, principal.Method)
		if projectErr != nil || goalErr != nil || itemErr != nil || executionErr != nil ||
			replacementErr != nil || deriveErr != nil {
			return executionAuthorityMatch{}, false, invalid(errors.New("sqlite.execution_authority_invalid"))
		}
		if executionRef.String() != "" || authority.ServicePrincipal == principal {
			return executionAuthorityMatch{authority: authority, active: active == 1}, true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return executionAuthorityMatch{}, false, mapDatabaseError(err)
	}
	return executionAuthorityMatch{}, false, nil
}
