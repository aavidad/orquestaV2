package orquestacoreworkflow_test

import (
	"fmt"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestRailRecordConcurrencyGateExternalMatrixV0(t *testing.T) {
	cases := []struct {
		name         string
		runRef       string
		gateRef      string
		planRef      string
		summary      string
		evidenceRefs []string
	}{
		{
			name:         "secrets-policy-reference",
			runRef:       "run-ref-rail-secrets-policy",
			gateRef:      "gate-ref-secrets-policy",
			planRef:      "plan-ref-secrets-policy",
			summary:      "secrets_policy references_only credential docs",
			evidenceRefs: []string{"evidence-ref-secrets-policy-references-only"},
		},
		{
			name:         "completion-run-ref",
			runRef:       "request-ref-app-completion-loop-rail",
			gateRef:      "gate-ref-app-completion-loop",
			planRef:      "plan-ref-app-completion-loop",
			summary:      "completion loop ref opaca",
			evidenceRefs: []string{"evidence-ref-completion-loop"},
		},
		{
			name:         "runtime-provider-model-git",
			runRef:       "run-ref-runtime-provider-model-git",
			gateRef:      "gate-ref-runtime-provider-model-git",
			planRef:      "plan-ref-runtime-provider-model-git",
			summary:      "runtime provider model git refs opacas",
			evidenceRefs: []string{"evidence-ref-runtime-provider-model-git"},
		},
		{
			name:         "oauth-token-budget",
			runRef:       "run-ref-oauth-token-budget",
			gateRef:      "gate-ref-oauth-token-budget",
			planRef:      "plan-ref-oauth-token-budget",
			summary:      "oauth token budget como texto operativo",
			evidenceRefs: []string{"evidence-ref-access-token-policy"},
		},
		{
			name:         "prompt-transcript-reference",
			runRef:       "run-ref-prompt-transcript-reference",
			gateRef:      "gate-ref-prompt-transcript-reference",
			planRef:      "plan-ref-prompt-transcript-reference",
			summary:      "prompt transcript references are opaque here",
			evidenceRefs: []string{"evidence-ref-prompt-transcript"},
		},
	}
	for _, term := range railLegacyForbiddenLikeTermsV0() {
		refPart := railRefPartV0(term)
		cases = append(cases, struct {
			name         string
			runRef       string
			gateRef      string
			planRef      string
			summary      string
			evidenceRefs []string
		}{
			name:         "legacy-term-" + refPart,
			runRef:       "run-ref-rail-" + refPart,
			gateRef:      "gate-ref-rail-" + refPart,
			planRef:      "plan-ref-rail-" + refPart,
			summary:      "rail matrix term " + term + " as opaque operational text",
			evidenceRefs: []string{"evidence-ref-rail-" + refPart, "evidence-ref-" + term},
		})
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := railProgramacionRunV0(t, tc.runRef)
			payload := orquestacoreworkflow.RecordConcurrencyGateCommandPayloadV0{
				RunRef:           tc.runRef,
				GateRef:          tc.gateRef,
				PlanRef:          tc.planRef,
				SubjectClaimRefs: []string{"claim-ref-ready-rail"},
				ReadyClaimRefs:   []string{"claim-ref-ready-rail"},
				Decision:         orquestacoreworkflow.ConcurrencyGateDecisionAllowRequestAgentV0,
				Summary:          tc.summary,
				EvidenceRefs:     tc.evidenceRefs,
			}
			command, err := orquestacoreworkflow.NewRecordConcurrencyGateCommandV0(
				railCommandMetaV0(tc.runRef, "cmd-gate-"+tc.name, "idem-gate-"+tc.name),
				payload,
			)
			if err != nil {
				t.Fatalf("NewRecordConcurrencyGateCommandV0: %v", err)
			}
			result, err := orquestacoreworkflow.HandleCommandV0(run, command)
			if err != nil {
				t.Fatalf("HandleCommandV0: %v", err)
			}
			if len(result.Events) != 1 || result.Events[0].EventType != orquestacoreworkflow.OrchestrationEventConcurrencyGateRecordedV0 {
				t.Fatalf("events=%+v", result.Events)
			}
			next, err := orquestacoreworkflow.ApplyEventV0(run, result.Events[0])
			if err != nil {
				t.Fatalf("ApplyEventV0: %v", err)
			}
			if issues := orquestacoreworkflow.ValidateOrchestrationRunV0(next); len(issues) != 0 {
				t.Fatalf("run issues=%+v", issues)
			}
		})
	}
}

func railLegacyForbiddenLikeTermsV0() []string {
	return []string{
		"secret",
		"secreto",
		"token",
		"password",
		"credential",
		"credencial",
		"api_key",
		"api-key",
		"access_token",
		"access-token",
		"refresh_token",
		"refresh-token",
		"client_secret",
		"client-secret",
		"oauth",
		"transcript",
		"prompt",
		"completion",
		"raw_text",
		"full_text",
		"runtime",
		"sql",
		"dsn",
		"connection",
		"conexion",
		"table",
		"tabla",
		"provider",
		"proveedor",
		"model",
		"home",
		"tmux",
		"docker",
		"git",
	}
}

func railRefPartV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("_", "-", " ", "-", "/", "-", "\\", "-")
	return replacer.Replace(value)
}

func railProgramacionRunV0(t *testing.T, runRef string) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	start, err := orquestacoreworkflow.NewStartRunCommandV0(
		railCommandMetaV0(runRef, "cmd-start-"+runRef, "idem-start-"+runRef),
		orquestacoreworkflow.StartRunCommandPayloadV0{
			ProjectRef: "project-ref-rail",
			AppSpecRef: "appspec-ref-rail",
		},
	)
	if err != nil {
		t.Fatalf("NewStartRunCommandV0: %v", err)
	}
	run := railApplySingleEventV0(t, orquestacoreworkflow.OrchestrationRunV0{}, start)
	open, err := orquestacoreworkflow.NewOpenPhaseCommandV0(
		railCommandMetaV0(runRef, "cmd-open-programacion-"+runRef, "idem-open-programacion-"+runRef),
		orquestacoreworkflow.OpenPhaseCommandPayloadV0{PhaseID: string(orquestacoreworkflow.OrchestrationPhaseProgramacionV0)},
	)
	if err != nil {
		t.Fatalf("NewOpenPhaseCommandV0: %v", err)
	}
	return railApplySingleEventV0(t, run, open)
}

func railApplySingleEventV0(
	t *testing.T,
	run orquestacoreworkflow.OrchestrationRunV0,
	command orquestacoreworkflow.OrchestrationCommandV0,
) orquestacoreworkflow.OrchestrationRunV0 {
	t.Helper()
	result, err := orquestacoreworkflow.HandleCommandV0(run, command)
	if err != nil {
		t.Fatalf("HandleCommandV0(%s): %v", command.CommandType, err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events=%d, want 1", len(result.Events))
	}
	next, err := orquestacoreworkflow.ApplyEventV0(run, result.Events[0])
	if err != nil {
		t.Fatalf("ApplyEventV0(%s): %v", result.Events[0].EventType, err)
	}
	return next
}

func railCommandMetaV0(runRef string, commandID string, idempotencyKey string) orquestacoreworkflow.OrchestrationCommandMetaV0 {
	return orquestacoreworkflow.OrchestrationCommandMetaV0{
		CommandID:      commandID,
		RunID:          runRef,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  fmt.Sprintf("corr-%s", commandID),
		RequestedBy:    "rail-test",
		OccurredAt:     "2026-05-23T17:00:00Z",
	}
}
