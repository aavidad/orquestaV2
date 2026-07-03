package orquestaserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	escalationDirectorDecisionStopV0     = "stop"
	escalationDirectorDecisionReviewOKV0 = "review_ok"
	escalationDirectorDecisionDeferV0    = "defer"

	escalationDirectorDefaultTimeoutV0   = 120 * time.Second
	escalationDirectorDefaultMaxPerDayV0 = 8

	escalationDirectorStopReasonV0       = "escalation_director_stop"
	escalationDirectorRequestedByV0      = "orquesta-server-escalation-director"
	escalationDirectorStopEvidenceV0     = "evidence-ref-escalation-director-stop"
	escalationDirectorReviewOKEvidenceV0 = "evidence-ref-escalation-director-review-ok"
)

var escalationDirectorDefaultCommandV0 = []string{"claude", "-p"}

// escalationDirectorEscalatableCodesV0 lista los codigos de anomalia que el
// bucle determinista no sabe resolver solo: cualquier codigo fuera de esta
// lista ya tiene accion automatica propia y no se escala.
var escalationDirectorEscalatableCodesV0 = map[string]struct{}{
	idleSelfImprovementGoalCompletedWithoutResultReasonV0: {},
	idleSelfImprovementGoalSelfReportTaskMismatchReasonV0: {},
	"partial_artifacts_written":                           {},
}

type escalationDirectorDecisionV0 struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// escalateGoalObservationAnomaliesV0 invoca un director externo por eventos
// (por defecto `claude -p`) SOLO cuando la observacion publica anomalias sin
// accion automatica. La decision queda acotada a stop cooperativo, marcar
// revisado o diferir; con presupuesto diario e idempotencia por firma para no
// quemar cuota en bucle.
func (runtime *RuntimeV0) escalateGoalObservationAnomaliesV0(
	ctx context.Context,
	result orquestagoal.GoalWorkObserveActiveResultV0,
	now time.Time,
) {
	if runtime == nil || runtime.tracker == nil || !runtime.config.EscalationDirectorEnabled {
		return
	}
	pending := escalationDirectorPendingIssuesV0(result)
	if len(pending) == 0 {
		return
	}
	signature := escalationDirectorSignatureV0(pending)
	state := runtime.tracker.SnapshotV0()
	if state.EscalationDirectorLastSignature == signature {
		return
	}
	day := now.UTC().Format("2006-01-02")
	used := state.EscalationDirectorInvocationsToday
	if state.EscalationDirectorDay != day {
		used = 0
	}
	maxPerDay := runtime.config.EscalationDirectorMaxPerDay
	if maxPerDay <= 0 {
		maxPerDay = escalationDirectorDefaultMaxPerDayV0
	}
	if used >= maxPerDay {
		runtime.persistEscalationDirectorOutcomeV0(ctx, day, used, signature, "budget_exhausted", "", now)
		runtime.auditEventV0(ctx, "escalation_director_budget_exhausted", "skipped", "", map[string]interface{}{
			"signature":   signature,
			"invocations": used,
			"max_per_day": maxPerDay,
		})
		return
	}
	decision, err := runtime.runEscalationDirectorCommandV0(ctx, escalationDirectorPromptV0(pending))
	if err != nil {
		runtime.persistEscalationDirectorOutcomeV0(ctx, day, used+1, signature, "error", err.Error(), now)
		runtime.auditEventV0(ctx, "escalation_director_error", "error", err.Error(), map[string]interface{}{
			"signature": signature,
		})
		return
	}
	runtime.applyEscalationDirectorDecisionV0(ctx, decision, pending)
	runtime.persistEscalationDirectorOutcomeV0(ctx, day, used+1, signature, decision.Decision, decision.Reason, now)
	runtime.auditEventV0(ctx, "escalation_director_decision", "ok", decision.Decision, map[string]interface{}{
		"signature": signature,
		"reason":    decision.Reason,
		"issues":    len(pending),
	})
}

func escalationDirectorPendingIssuesV0(
	result orquestagoal.GoalWorkObserveActiveResultV0,
) []orquestagoal.GoalWorkObserveActiveIssueV0 {
	pending := []orquestagoal.GoalWorkObserveActiveIssueV0{}
	seen := map[string]struct{}{}
	for _, issue := range result.Issues {
		pending = appendEscalationDirectorIssueV0(pending, seen, issue)
	}
	for _, observation := range result.Observations {
		runRef := strings.TrimSpace(observation.State.RunRef)
		goalRef := strings.TrimSpace(observation.State.GoalRef)
		if goalRef == "" {
			goalRef = strings.TrimSpace(observation.Result.GoalRef)
		}
		for _, issue := range observation.Result.Issues {
			pending = appendEscalationDirectorIssueV0(pending, seen, orquestagoal.GoalWorkObserveActiveIssueV0{
				RunRef:  runRef,
				GoalRef: goalRef,
				Code:    issue.Code,
				Field:   issue.Field,
				Message: issue.Detail,
			})
		}
		for _, issue := range observation.Closure.Issues {
			pending = appendEscalationDirectorIssueV0(pending, seen, orquestagoal.GoalWorkObserveActiveIssueV0{
				RunRef:  runRef,
				GoalRef: goalRef,
				Code:    issue.Code,
				Field:   issue.Field,
				Message: issue.Detail,
			})
		}
	}
	return pending
}

func appendEscalationDirectorIssueV0(
	pending []orquestagoal.GoalWorkObserveActiveIssueV0,
	seen map[string]struct{},
	issue orquestagoal.GoalWorkObserveActiveIssueV0,
) []orquestagoal.GoalWorkObserveActiveIssueV0 {
	issue.Code = strings.TrimSpace(issue.Code)
	if !escalationDirectorIssueEscalatableV0(issue.Code) {
		return pending
	}
	issue.RunRef = strings.TrimSpace(issue.RunRef)
	issue.GoalRef = strings.TrimSpace(issue.GoalRef)
	issue.Field = strings.TrimSpace(issue.Field)
	issue.Message = strings.TrimSpace(issue.Message)
	key := issue.RunRef + "|" + issue.GoalRef + "|" + issue.Code + "|" + issue.Field
	if _, ok := seen[key]; ok {
		return pending
	}
	seen[key] = struct{}{}
	return append(pending, issue)
}

func escalationDirectorIssueEscalatableV0(code string) bool {
	code = strings.TrimSpace(code)
	if _, ok := escalationDirectorEscalatableCodesV0[code]; ok {
		return true
	}
	return strings.HasPrefix(code, "review_")
}

func escalationDirectorSignatureV0(issues []orquestagoal.GoalWorkObserveActiveIssueV0) string {
	markers := make([]string, 0, len(issues))
	for _, issue := range issues {
		markers = append(markers, strings.TrimSpace(issue.RunRef)+"|"+strings.TrimSpace(issue.Code))
	}
	sort.Strings(markers)
	sum := sha256.Sum256([]byte(strings.Join(markers, "\n")))
	return fmt.Sprintf("sha256:%x", sum[:])
}

func escalationDirectorPromptV0(issues []orquestagoal.GoalWorkObserveActiveIssueV0) string {
	var builder strings.Builder
	builder.WriteString("Eres el director de escalada de Orquesta. La observacion de goals ")
	builder.WriteString("ha publicado anomalias sin accion automatica. Decide UNA accion global ")
	builder.WriteString("y responde SOLO un objeto JSON de la forma ")
	builder.WriteString(`{"decision":"stop|review_ok|defer","reason":"..."}` + "\n")
	builder.WriteString("- stop: pedir stop cooperativo del goal (anomalia grave en curso).\n")
	builder.WriteString("- review_ok: la evidencia indica falsa alarma o cierre ya valido; marcar revisado.\n")
	builder.WriteString("- defer: falta contexto; dejar para el siguiente tick o el operador.\n")
	builder.WriteString("Anomalias:\n")
	for _, issue := range issues {
		fmt.Fprintf(&builder, "- code=%s run_ref=%s goal_ref=%s field=%s detalle=%s\n",
			issue.Code, issue.RunRef, issue.GoalRef, issue.Field, issue.Message)
	}
	return builder.String()
}

func (runtime *RuntimeV0) runEscalationDirectorCommandV0(
	ctx context.Context,
	prompt string,
) (escalationDirectorDecisionV0, error) {
	command := runtime.config.EscalationDirectorCommand
	if len(command) == 0 {
		command = escalationDirectorDefaultCommandV0
	}
	timeout := runtime.config.EscalationDirectorTimeout
	if timeout <= 0 {
		timeout = escalationDirectorDefaultTimeoutV0
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := append(append([]string{}, command[1:]...), prompt)
	cmd := exec.CommandContext(runCtx, command[0], args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return escalationDirectorDecisionV0{}, fmt.Errorf(
			"escalation director command: %w (stderr: %s)",
			err,
			strings.TrimSpace(stderr.String()),
		)
	}
	return parseEscalationDirectorDecisionV0(stdout.String())
}

func parseEscalationDirectorDecisionV0(output string) (escalationDirectorDecisionV0, error) {
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start < 0 || end <= start {
		return escalationDirectorDecisionV0{}, fmt.Errorf("escalation director: sin objeto JSON en la salida")
	}
	var decision escalationDirectorDecisionV0
	if err := json.Unmarshal([]byte(output[start:end+1]), &decision); err != nil {
		return escalationDirectorDecisionV0{}, fmt.Errorf("escalation director: JSON invalido: %w", err)
	}
	decision.Decision = strings.ToLower(strings.TrimSpace(decision.Decision))
	switch decision.Decision {
	case escalationDirectorDecisionStopV0,
		escalationDirectorDecisionReviewOKV0,
		escalationDirectorDecisionDeferV0:
		return decision, nil
	}
	return decision, fmt.Errorf("escalation director: decision desconocida %q", decision.Decision)
}

func (runtime *RuntimeV0) applyEscalationDirectorDecisionV0(
	ctx context.Context,
	decision escalationDirectorDecisionV0,
	issues []orquestagoal.GoalWorkObserveActiveIssueV0,
) {
	switch decision.Decision {
	case escalationDirectorDecisionStopV0:
		if runtime.goalStopper == nil || len(issues) == 0 {
			return
		}
		issue := issues[0]
		_, _ = runtime.goalStopper.RequestGoalCooperativeStopV0(ctx, GoalCooperativeStopRequestV0{
			RunRef:            strings.TrimSpace(issue.RunRef),
			GoalRef:           strings.TrimSpace(issue.GoalRef),
			Reason:            escalationDirectorStopReasonV0,
			RecommendedAction: idleSelfImprovementGoalReviewReplanRecommendedActionV0,
			RequestedBy:       escalationDirectorRequestedByV0,
			IdempotencyKey:    "idem-escalation-stop-" + serverGoalProgressSafeRefPartV0(issue.RunRef),
			EvidenceRefs:      escalationDirectorStopEvidenceRefsV0(issue),
		})
	case escalationDirectorDecisionReviewOKV0:
		if runtime.goalStateStore == nil {
			return
		}
		seen := map[string]struct{}{}
		for _, issue := range issues {
			runRef := strings.TrimSpace(issue.RunRef)
			if runRef == "" {
				continue
			}
			if _, ok := seen[runRef]; ok {
				continue
			}
			seen[runRef] = struct{}{}
			state, err := runtime.goalStateStore.LoadGoalWorkStateV0(ctx, runRef)
			if err != nil {
				continue
			}
			state.EvidenceRefs = compactConfigStringsV0(append(
				state.EvidenceRefs,
				escalationDirectorReviewOKEvidenceV0,
			))
			_ = runtime.goalStateStore.SaveGoalWorkStateV0(ctx, state)
		}
	}
}

func escalationDirectorStopEvidenceRefsV0(
	issue orquestagoal.GoalWorkObserveActiveIssueV0,
) []string {
	refs := []string{escalationDirectorStopEvidenceV0}
	if code := serverGoalProgressSafeRefPartV0(issue.Code); code != "unknown" {
		refs = append(refs, "evidence-ref-escalation-director-issue:"+code)
	}
	if field := serverGoalProgressSafeRefPartV0(issue.Field); field != "unknown" {
		refs = append(refs, "evidence-ref-escalation-director-field:"+field)
	}
	return compactConfigStringsV0(refs)
}

func (runtime *RuntimeV0) persistEscalationDirectorOutcomeV0(
	ctx context.Context,
	day string,
	invocations int,
	signature string,
	decision string,
	reason string,
	now time.Time,
) {
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkEscalationDirectorV0(day, invocations, signature, decision, reason, now),
		"escalation_director",
	)
}

func (tracker *StatusTrackerV0) MarkEscalationDirectorV0(
	day string,
	invocations int,
	signature string,
	decision string,
	reason string,
	now time.Time,
) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.EscalationDirectorDay = day
		state.EscalationDirectorInvocationsToday = invocations
		state.EscalationDirectorLastSignature = signature
		state.EscalationDirectorLastDecision = decision
		state.EscalationDirectorLastReason = reason
		state.EscalationDirectorLastAt = formatTimeV0(now)
	})
}
