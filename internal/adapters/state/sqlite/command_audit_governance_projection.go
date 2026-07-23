package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"orquesta/internal/ports"
)

var _ ports.CommandGovernanceAuditReader = (*Repository)(nil)

type commandGovernanceRoot struct {
	kind                     ports.CommandGovernanceFactKind
	ref, authorization, goal string
	planGeneration           int64
	anchor                   string
}

func (repository *Repository) ProjectCommandGovernance(
	ctx context.Context,
	invocationRef string,
) (ports.CommandGovernanceProjection, error) {
	if repository == nil || repository.db == nil || ctx == nil ||
		strings.TrimSpace(invocationRef) != invocationRef || invocationRef == "" {
		return ports.CommandGovernanceProjection{}, errors.New("sqlite.command_audit_projection_invalid")
	}
	projection, root, err := repository.commandGovernanceRoot(ctx, invocationRef)
	if err != nil {
		return ports.CommandGovernanceProjection{}, err
	}
	projection.Facts = []ports.CommandGovernanceFact{
		{Kind: ports.CommandGovernanceAuthorization, Ref: root.authorization, CausalRef: projection.InvocationRef},
		{Kind: root.kind, Ref: root.ref, CausalRef: root.authorization},
	}
	if err := repository.appendCommandEffectFacts(ctx, root, &projection); err != nil {
		return ports.CommandGovernanceProjection{}, err
	}
	if err := repository.appendCommandTerminalFacts(ctx, root, &projection); err != nil {
		return ports.CommandGovernanceProjection{}, err
	}
	return projection, nil
}

func (repository *Repository) commandGovernanceRoot(
	ctx context.Context,
	invocationRef string,
) (ports.CommandGovernanceProjection, commandGovernanceRoot, error) {
	var projection ports.CommandGovernanceProjection
	var root commandGovernanceRoot
	var kind string
	err := repository.db.QueryRowContext(ctx, `
SELECT invocation.ref,outcome.ref,invocation.command_id,invocation.request_ref,
 invocation.principal_ref,invocation.project_ref,outcome.status,outcome.error_code,
 CASE invocation.command_id
  WHEN 'orquesta.director.plan.propose' THEN director.authorization_receipt_ref
  WHEN 'orquesta.effects.decide' THEN approval.authorization_receipt_ref
  WHEN 'orquesta.goals.control' THEN control.authorization_receipt_ref
  WHEN 'orquesta.council.round.open' THEN round.authorization_receipt_ref
  WHEN 'orquesta.council.skip' THEN skip.authorization_receipt_ref END,
 CASE invocation.command_id
  WHEN 'orquesta.director.plan.propose' THEN 'director_decision'
  WHEN 'orquesta.effects.decide' THEN 'effect_decision'
  WHEN 'orquesta.goals.control' THEN 'closure_control'
  WHEN 'orquesta.council.round.open' THEN 'council_round'
  WHEN 'orquesta.council.skip' THEN 'council_skip' END,
 CASE invocation.command_id
  WHEN 'orquesta.director.plan.propose' THEN director.ref
  WHEN 'orquesta.effects.decide' THEN approval.ref
  WHEN 'orquesta.goals.control' THEN control.ref
  WHEN 'orquesta.council.round.open' THEN round.ref
  WHEN 'orquesta.council.skip' THEN skip.ref END,
 CASE invocation.command_id
  WHEN 'orquesta.director.plan.propose' THEN director.goal_ref
  WHEN 'orquesta.effects.decide' THEN approval.goal_ref
  WHEN 'orquesta.goals.control' THEN control.goal_ref
  WHEN 'orquesta.council.round.open' THEN round.goal_ref
  WHEN 'orquesta.council.skip' THEN skip.goal_ref END,
 CASE invocation.command_id
  WHEN 'orquesta.director.plan.propose' THEN director.applied_plan_generation
  WHEN 'orquesta.effects.decide' THEN approval.plan_generation
  WHEN 'orquesta.goals.control' THEN control.plan_generation
  WHEN 'orquesta.council.round.open' THEN round.plan_generation
  WHEN 'orquesta.council.skip' THEN skip.plan_generation END,
 CASE invocation.command_id
  WHEN 'orquesta.effects.decide' THEN approval.intent_ref
  WHEN 'orquesta.council.round.open' THEN round.subject_digest
  ELSE '' END
FROM command_invocations invocation
JOIN command_outcomes outcome ON outcome.command_invocation_ref=invocation.ref
LEFT JOIN director_decisions director ON invocation.command_id='orquesta.director.plan.propose'
 AND director.request_ref=invocation.request_ref AND director.principal_ref=invocation.principal_ref
 AND director.project_ref=invocation.project_ref
LEFT JOIN effect_approvals approval ON invocation.command_id='orquesta.effects.decide'
 AND approval.request_ref=invocation.request_ref AND approval.decided_by_ref=invocation.principal_ref
 AND approval.project_ref=invocation.project_ref
LEFT JOIN controls control ON invocation.command_id='orquesta.goals.control'
 AND control.request_ref=invocation.request_ref AND control.principal_ref=invocation.principal_ref
 AND control.project_ref=invocation.project_ref
LEFT JOIN council_rounds round ON invocation.command_id='orquesta.council.round.open'
 AND round.request_ref=invocation.request_ref AND round.opened_by_ref=invocation.principal_ref
 AND round.project_ref=invocation.project_ref
LEFT JOIN council_skips skip ON invocation.command_id='orquesta.council.skip'
 AND skip.request_ref=invocation.request_ref AND skip.principal_ref=invocation.principal_ref
 AND skip.project_ref=invocation.project_ref
WHERE invocation.ref=? AND outcome.status='completed'
 AND invocation.command_id IN ('orquesta.director.plan.propose','orquesta.effects.decide',
  'orquesta.goals.control','orquesta.council.round.open','orquesta.council.skip')
 AND EXISTS(
  SELECT 1 FROM authorization_receipts authorization
  WHERE authorization.ref=CASE invocation.command_id
   WHEN 'orquesta.director.plan.propose' THEN director.authorization_receipt_ref
   WHEN 'orquesta.effects.decide' THEN approval.authorization_receipt_ref
   WHEN 'orquesta.goals.control' THEN control.authorization_receipt_ref
   WHEN 'orquesta.council.round.open' THEN round.authorization_receipt_ref
   WHEN 'orquesta.council.skip' THEN skip.authorization_receipt_ref END
   AND authorization.principal_ref=invocation.principal_ref
   AND authorization.project_ref=invocation.project_ref
 )`,
		invocationRef,
	).Scan(
		&projection.InvocationRef, &projection.OutcomeRef, &projection.CommandID,
		&projection.RequestRef, &projection.PrincipalRef, &projection.ProjectRef,
		&projection.OutcomeStatus, &projection.ErrorCode, &root.authorization, &kind,
		&root.ref, &root.goal, &root.planGeneration, &root.anchor,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return projection, root, ports.ErrCommandAuditProjectionNotFound
	}
	if err != nil {
		return projection, root, fmt.Errorf("sqlite.command_audit_projection: %w", err)
	}
	root.kind = ports.CommandGovernanceFactKind(kind)
	projection.AuthorizationReceiptRef, projection.FactKind, projection.FactRef =
		root.authorization, root.kind, root.ref
	if root.authorization == "" || root.ref == "" || root.goal == "" || root.planGeneration < 1 {
		return projection, root, errors.New("sqlite.command_audit_projection_cross_link_invalid")
	}
	return projection, root, nil
}

func commandIntentScope(root commandGovernanceRoot) (string, []any) {
	switch root.kind {
	case ports.CommandGovernanceDirectorDecision:
		return `intent.goal_ref=? AND intent.plan_generation=? AND intent.authority_receipt_ref=?`,
			[]any{root.goal, root.planGeneration, root.authorization}
	case ports.CommandGovernanceClosureControl:
		return `EXISTS(SELECT 1 FROM outbox action WHERE action.ref=intent.action_ref AND action.control_ref=?)`,
			[]any{root.ref}
	case ports.CommandGovernanceEffectDecision:
		return `intent.ref=?`, []any{root.anchor}
	case ports.CommandGovernanceCouncilRound:
		return `EXISTS(SELECT 1 FROM executions execution WHERE execution.ref=intent.execution_ref
 AND execution.council_subject_digest=?)`, []any{root.anchor}
	case ports.CommandGovernanceCouncilSkip:
		return `intent.council_skip_ref=?`, []any{root.ref}
	default:
		return "0", nil
	}
}

func (repository *Repository) appendCommandEffectFacts(
	ctx context.Context,
	root commandGovernanceRoot,
	projection *ports.CommandGovernanceProjection,
) error {
	scope, args := commandIntentScope(root)
	rows, err := repository.db.QueryContext(ctx, `
SELECT intent.ref,attempt.ref,receipt.ref
FROM effect_intents intent
LEFT JOIN effect_attempts attempt ON attempt.intent_ref=intent.ref
LEFT JOIN effect_receipts receipt ON receipt.attempt_ref=attempt.ref
WHERE `+scope+`
ORDER BY intent.created_at,intent.ref,attempt.started_at,attempt.ref,receipt.confirmed_at,receipt.ref`, args...)
	if err != nil {
		return fmt.Errorf("sqlite.command_audit_projection_effects: %w", err)
	}
	defer rows.Close()
	seen := make(map[string]struct{})
	for rows.Next() {
		var intent string
		var attempt, receipt sql.NullString
		if err := rows.Scan(&intent, &attempt, &receipt); err != nil {
			return err
		}
		appendGovernanceFact(projection, seen, ports.CommandGovernanceEffectIntent, intent, root.ref)
		if attempt.Valid {
			appendGovernanceFact(projection, seen, ports.CommandGovernanceEffectAttempt, attempt.String, intent)
		}
		if receipt.Valid {
			appendGovernanceFact(projection, seen, ports.CommandGovernanceEffectReceipt, receipt.String, attempt.String)
		}
	}
	return rows.Err()
}

func (repository *Repository) appendCommandTerminalFacts(
	ctx context.Context,
	root commandGovernanceRoot,
	projection *ports.CommandGovernanceProjection,
) error {
	scope, args := commandIntentScope(root)
	seen := make(map[string]struct{}, len(projection.Facts))
	for _, fact := range projection.Facts {
		seen[string(fact.Kind)+"\x00"+fact.Ref] = struct{}{}
	}
	rootEvent := ""
	switch root.kind {
	case ports.CommandGovernanceDirectorDecision:
		rootEvent = "event:director-plan-applied:" + root.ref
	case ports.CommandGovernanceClosureControl:
		rootEvent = "event:control:" + root.ref
	}
	if rootEvent != "" {
		var ref string
		if err := repository.db.QueryRowContext(ctx, `SELECT ref FROM events WHERE ref=?`, rootEvent).Scan(&ref); err != nil {
			return fmt.Errorf("sqlite.command_audit_projection_root_event: %w", err)
		}
		appendGovernanceFact(projection, seen, ports.CommandGovernanceCausalEvent, ref, root.ref)
	}
	var terminalEvent string
	err := repository.db.QueryRowContext(ctx, `
SELECT event.ref
FROM goals goal
JOIN events event ON event.goal_ref=goal.ref
 AND event.ref='event:goal-' || goal.state || ':' || goal.ref
 AND event.kind='goal.' || goal.state AND event.occurred_at=goal.closed_at
WHERE goal.ref=? AND goal.state IN ('succeeded','failed','canceled')
 AND goal.closed_at IS NOT NULL`, root.goal).Scan(&terminalEvent)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("sqlite.command_audit_projection_terminal_goal: %w", err)
	}
	if err == nil {
		appendGovernanceFact(
			projection, seen, ports.CommandGovernanceTerminalEvent, terminalEvent, root.ref,
		)
	}
	rows, err := repository.db.QueryContext(ctx, `
SELECT DISTINCT event.ref,event.kind,receipt.ref
FROM effect_intents intent
JOIN effect_attempts attempt ON attempt.intent_ref=intent.ref
JOIN effect_receipts receipt ON receipt.attempt_ref=attempt.ref
JOIN events event ON event.goal_ref=receipt.goal_ref
 AND ifnull(event.work_item_ref,'')=receipt.work_item_ref
 AND ifnull(event.execution_ref,'')=receipt.execution_ref
 AND event.occurred_at=receipt.confirmed_at
WHERE `+scope+` ORDER BY event.ref`, args...)
	if err != nil {
		return fmt.Errorf("sqlite.command_audit_projection_events: %w", err)
	}
	for rows.Next() {
		var ref, eventKind, receipt string
		if err := rows.Scan(&ref, &eventKind, &receipt); err != nil {
			rows.Close()
			return err
		}
		kind := ports.CommandGovernanceCausalEvent
		if eventKind == "goal.succeeded" || eventKind == "goal.failed" || eventKind == "goal.canceled" {
			kind = ports.CommandGovernanceTerminalEvent
		}
		appendGovernanceFact(projection, seen, kind, ref, receipt)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	rows, err = repository.db.QueryContext(ctx, `
SELECT attestation.ref,attestation.effect_receipt_ref
FROM attestations attestation
JOIN effect_intents intent ON intent.ref=attestation.effect_intent_ref
JOIN effect_attempts attempt ON attempt.ref=attestation.effect_attempt_ref AND attempt.intent_ref=intent.ref
JOIN effect_receipts receipt ON receipt.ref=attestation.effect_receipt_ref
 AND receipt.attempt_ref=attempt.ref AND receipt.intent_ref=intent.ref
WHERE attestation.kind='required_tests' AND `+scope+`
ORDER BY attestation.finished_at,attestation.ref`, args...)
	if err != nil {
		return fmt.Errorf("sqlite.command_audit_projection_attestations: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ref, receipt string
		if err := rows.Scan(&ref, &receipt); err != nil {
			return err
		}
		appendGovernanceFact(projection, seen, ports.CommandGovernanceAttestation, ref, receipt)
	}
	return rows.Err()
}

func appendGovernanceFact(
	projection *ports.CommandGovernanceProjection,
	seen map[string]struct{},
	kind ports.CommandGovernanceFactKind,
	ref, causalRef string,
) {
	if ref == "" || causalRef == "" {
		return
	}
	key := string(kind) + "\x00" + ref
	if _, exists := seen[key]; exists {
		return
	}
	seen[key] = struct{}{}
	projection.Facts = append(projection.Facts, ports.CommandGovernanceFact{
		Kind: kind, Ref: ref, CausalRef: causalRef,
	})
}
