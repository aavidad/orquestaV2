package orquestaappdirectorservice

import (
	"context"
	"errors"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectoragent "orquesta/modulos/orquesta-director-agent"
	orquestadirectoragentworkflow "orquesta/modulos/orquesta-director-agent-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	appDirectorDecisionRecoveryEvidenceRefV0 = "evidence-ref-app-director-decision-recovery-v0"
	appDirectorDecisionRecoverySourceGroupV0 = "app-director-service"
)

type startAppDirectorDecisionRecoveryV0 struct {
	ReasonCode  string
	Field       string
	Code        string
	Detail      string
	DecisionRef string
	CommandType string
}

func startAppDirectorDecisionSourceRecoveryV0(err error) startAppDirectorDecisionRecoveryV0 {
	recovery := startAppDirectorDecisionRecoveryV0{
		ReasonCode: "director_decision_source_error",
		Field:      "director_decision_source",
		Code:       "source_error",
	}
	if err == nil {
		return recovery
	}
	recovery.Detail = err.Error()
	var budgetIssue orquestadirectoragentworkflow.DirectorAgentDecisionBatchBudgetIssueV0
	if errors.As(err, &budgetIssue) {
		recovery.Field = "director_decision_source." + strings.TrimSpace(budgetIssue.Field)
		recovery.Code = strings.TrimSpace(budgetIssue.ReasonCode)
	}
	return recovery
}

func startAppDirectorDecisionPolicyRecoveryV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
	err error,
) startAppDirectorDecisionRecoveryV0 {
	return startAppDirectorDecisionRecoveryV0{
		ReasonCode:  "director_decision_policy_blocked",
		Field:       startAppDirectorIssueFieldV0(err),
		Code:        "policy_blocked",
		DecisionRef: decision.DecisionRef,
		CommandType: decision.CommandType,
	}
}

func startAppDirectorDecisionApplyRecoveryV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
	err error,
) startAppDirectorDecisionRecoveryV0 {
	return startAppDirectorDecisionRecoveryV0{
		ReasonCode:  "director_decision_apply_error",
		Field:       startAppDirectorErrorFieldV0(err),
		Code:        "apply_error",
		DecisionRef: decision.DecisionRef,
		CommandType: decision.CommandType,
	}
}

func startAppDirectorDecisionWorkflowIssueRecoveryV0(
	decision orquestadirectoragent.DirectorAgentDecisionV0,
	issue orquestadirectoragentworkflow.DirectorAgentWorkflowIssueV0,
) startAppDirectorDecisionRecoveryV0 {
	return startAppDirectorDecisionRecoveryV0{
		ReasonCode:  "director_decision_invalid",
		Field:       "director_decision." + strings.TrimSpace(issue.Field),
		Code:        strings.TrimSpace(issue.Code),
		DecisionRef: decision.DecisionRef,
		CommandType: decision.CommandType,
	}
}

func recoverStartAppDirectorDecisionFailureV0(
	ctx context.Context,
	request StartAppDirectorRequestV0,
	ports StartAppDirectorPortsV0,
	loop orquestacionnucleoapp.ProgressiveLoopResultV0,
	recovery startAppDirectorDecisionRecoveryV0,
) (orquestacionnucleoapp.ProgressiveLoopResultV0, error) {
	if strings.TrimSpace(loop.Run.RunID) == "" {
		return loop, AppDirectorServiceIssueV0{Field: "director_decision.recovery.run"}
	}
	recovery = normalizeStartAppDirectorDecisionRecoveryV0(recovery)
	command, err := orquestacoreworkflow.NewBlockRunCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-" + recoveryBlockerIDV0(recovery),
			RunID:          loop.Run.RunID,
			IdempotencyKey: "idem-" + recoveryBlockerIDV0(recovery),
			CorrelationID:  request.CorrelationID,
			RequestedBy:    request.RequestedBy,
			OccurredAt:     request.OccurredAt,
		},
		orquestacoreworkflow.BlockRunCommandPayloadV0{
			BlockerID:    recoveryBlockerIDV0(recovery),
			ReasonCode:   recovery.ReasonCode,
			Summary:      recoverySummaryV0(recovery),
			SourceGroup:  appDirectorDecisionRecoverySourceGroupV0,
			EvidenceRefs: []string{appDirectorDecisionRecoveryEvidenceRefV0},
		},
	)
	if err != nil {
		return loop, err
	}
	commandResult, err := orquestacoreworkflow.HandleCommandV0(loop.Run, command)
	if err != nil {
		return loop, err
	}
	next, err := applyStartAppDirectorDecisionRecoveryEventsV0(loop.Run, commandResult.Events)
	if err != nil {
		return loop, err
	}
	if len(commandResult.Events) > 0 {
		if err := ports.EventSink.AppendRunEventsV0(ctx, loop.Run.RunID, commandResult.Events); err != nil {
			return loop, err
		}
		if err := ports.RunStore.SaveRunV0(ctx, next); err != nil {
			return loop, err
		}
	}
	loop.Run = next
	return loop, nil
}

func normalizeStartAppDirectorDecisionRecoveryV0(
	recovery startAppDirectorDecisionRecoveryV0,
) startAppDirectorDecisionRecoveryV0 {
	recovery.ReasonCode = strings.TrimSpace(recovery.ReasonCode)
	recovery.Field = strings.TrimSpace(recovery.Field)
	recovery.Code = strings.TrimSpace(recovery.Code)
	recovery.Detail = strings.TrimSpace(recovery.Detail)
	recovery.DecisionRef = strings.TrimSpace(recovery.DecisionRef)
	recovery.CommandType = strings.TrimSpace(recovery.CommandType)
	if recovery.ReasonCode == "" {
		recovery.ReasonCode = "director_decision_blocked"
	}
	if recovery.Field == "" {
		recovery.Field = "director_decision"
	}
	if recovery.Code == "" {
		recovery.Code = "blocked"
	}
	return recovery
}

func applyStartAppDirectorDecisionRecoveryEventsV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	events []orquestacoreworkflow.OrchestrationEventV0,
) (orquestacoreworkflow.OrchestrationRunV0, error) {
	next := run
	for _, event := range events {
		applied, err := orquestacoreworkflow.ApplyEventV0(next, event)
		if err != nil {
			return run, err
		}
		next = applied
	}
	return next, nil
}

func recoveryBlockerIDV0(recovery startAppDirectorDecisionRecoveryV0) string {
	parts := []string{
		"app-director-decision",
		recovery.ReasonCode,
		recovery.Field,
		recovery.DecisionRef,
	}
	return compactRecoveryRefV0(strings.Join(compactStartAppDirectorStringsV0(parts), "-"))
}

func recoverySummaryV0(recovery startAppDirectorDecisionRecoveryV0) string {
	parts := []string{
		"Corregir decision del director",
		"code=" + recovery.Code,
		"field=" + recovery.Field,
	}
	if recovery.DecisionRef != "" {
		parts = append(parts, "decision_ref="+recovery.DecisionRef)
	}
	if recovery.CommandType != "" {
		parts = append(parts, "command_type="+recovery.CommandType)
	}
	if recovery.Detail != "" {
		parts = append(parts, "detail="+recovery.Detail)
	}
	return compactRecoverySummaryV0(strings.Join(parts, "; "))
}

func startAppDirectorIssueFieldV0(err error) string {
	var issue AppDirectorServiceIssueV0
	if errors.As(err, &issue) {
		return issue.Field
	}
	return "director_decision"
}

func startAppDirectorErrorFieldV0(err error) string {
	var commandErr orquestacoreworkflow.OrchestrationCommandErrorV0
	if errors.As(err, &commandErr) && strings.TrimSpace(commandErr.Field) != "" {
		return "director_decision." + strings.TrimSpace(commandErr.Field)
	}
	return "director_decision"
}

func compactRecoveryRefV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "app-director-decision-blocked"
	}
	if len(out) > 180 {
		return strings.Trim(out[:180], "-")
	}
	return out
}

func compactRecoverySummaryV0(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= 600 {
		return value
	}
	return strings.TrimSpace(value[:600])
}
