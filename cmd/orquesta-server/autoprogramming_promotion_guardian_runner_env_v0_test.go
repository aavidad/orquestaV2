package main

import (
	"context"
	"strings"
	"testing"
)

const guardianRunnerEnvSuccessJSONV0 = `{"schema_version":"orquesta_guardian_result.v0","status":"candidate_promoted","promoted":true,"redaction_level":"refs_only","freshness":"2026-05-27T10:00:00Z","counters":{"evidence_refs":1},"evidence_refs":["evidence-ref-guardian-runner-env"]}`

func TestAutoprogrammingPromotionGuardianRunnerEnvV0NoHeredarCredencialesPorDefecto(t *testing.T) {
	t.Setenv("HOME", "/sensitive/home")
	t.Setenv("ORQUESTA_CODEX_HOME", "/sensitive/codex-home")
	t.Setenv("ORQUESTA_CODEX_CODE_HOME", "/sensitive/code-home")
	t.Setenv("ORQUESTA_SERVER_CONTROL_TOKEN", "secret-control-token")

	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: `if [ -n "$HOME$ORQUESTA_CODEX_HOME$ORQUESTA_CODEX_CODE_HOME$ORQUESTA_SERVER_CONTROL_TOKEN" ]; then printf '%s\n' '{"schema_version":"orquesta_guardian_result.v0","status":"candidate_failed","redaction_level":"refs_only","freshness":"2026-05-27T10:00:00Z","counters":{"evidence_refs":1},"evidence_refs":["evidence-ref-guardian-runner-env-leaked"]}'; exit 1; fi; printf '%s\n' '` + guardianRunnerEnvSuccessJSONV0 + `'`,
	}
	result, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{
			ProjectDir:   t.TempDir(),
			StateDir:     t.TempDir(),
			EvidenceRefs: []string{"evidence-ref-home-opt-in"},
			PromotionRef: "promotion-ref-env-opt-in",
			WorktreeRef:  "worktree-ref-env-opt-in",
			BranchRef:    "branch-ref-env-opt-in",
		},
	)
	if err != nil || result.Status != "candidate_promoted" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestAutoprogrammingPromotionGuardianRunnerEnvV0PermiteOptInExplicito(t *testing.T) {
	t.Setenv("HOME", "/explicit/home")

	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command:      `if [ "$HOME" != "/explicit/home" ]; then exit 1; fi; printf '%s\n' '` + guardianRunnerEnvSuccessJSONV0 + `'`,
		EnvAllowlist: []string{"PATH", "HOME"},
	}
	result, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{
			ProjectDir:   t.TempDir(),
			StateDir:     t.TempDir(),
			EvidenceRefs: []string{"evidence-ref-runner-home-opt-in"},
		},
	)
	if err != nil || len(result.EvidenceRefs) != 1 ||
		result.EvidenceRefs[0] != "evidence-ref-guardian-runner-env" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestAutoprogrammingPromotionGuardianRunnerEnvV0ConservaSoloAllowlistYVarsGuardian(t *testing.T) {
	parent := []string{
		"PATH=/bin",
		"HOME=/sensitive/home",
		"ORQUESTA_CODEX_HOME=/sensitive/codex-home",
		"SECRET_TOKEN=secret",
	}
	env := autoprogrammingPromotionGuardianRunnerEnvV0(
		parent,
		serverAutoprogrammingPromotionGuardianRequestV0{
			ProjectDir:  "/project",
			StateDir:    "/state",
			WorktreeRef: "worktree-ref-orquesta-server-idle-self-improvement",
			BranchRef:   "branch-ref-orquesta-server-idle-self-improvement",
		},
		nil,
	)
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{
		"HOME=/sensitive/home",
		"ORQUESTA_CODEX_HOME=/sensitive/codex-home",
		"SECRET_TOKEN=secret",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("env hereda secreto %q: %s", forbidden, joined)
		}
	}
	for _, want := range []string{
		"PATH=/bin",
		"ORQUESTA_GUARDIAN_RUNNER_ENV_POLICY=minimal_allowlist",
		"ORQUESTA_GUARDIAN_WORKTREE_REF=worktree-ref-orquesta-server-idle-self-improvement",
		"ORQUESTA_GUARDIAN_BRANCH_REF=branch-ref-orquesta-server-idle-self-improvement",
		"GOCACHE=/state/guardian-runner-cache/go-build",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("env no contiene %q: %s", want, joined)
		}
	}
}

func TestAutoprogrammingPromotionGuardianRunnerEnvV0NoHeredaVarsGuardianDelPadre(t *testing.T) {
	env := autoprogrammingPromotionGuardianRunnerEnvV0(
		[]string{
			"PATH=/bin",
			"ORQUESTA_GUARDIAN_BUILD_COMMAND=rm -rf forbidden",
			"ORQUESTA_GUARDIAN_WORKTREE_REF=worktree-ref-forbidden",
		},
		serverAutoprogrammingPromotionGuardianRequestV0{
			StateDir:     "/state",
			BuildCommand: "go test ./...",
			WorktreeRef:  "worktree-ref-owned",
		},
		[]string{"PATH", "ORQUESTA_GUARDIAN_BUILD_COMMAND", "ORQUESTA_GUARDIAN_WORKTREE_REF"},
	)
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{
		"ORQUESTA_GUARDIAN_BUILD_COMMAND=rm -rf forbidden",
		"ORQUESTA_GUARDIAN_WORKTREE_REF=worktree-ref-forbidden",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("env hereda var guardian del padre %q: %s", forbidden, joined)
		}
	}
	for _, want := range []string{
		"ORQUESTA_GUARDIAN_BUILD_COMMAND=go test ./...",
		"ORQUESTA_GUARDIAN_WORKTREE_REF=worktree-ref-owned",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("env no contiene %q: %s", want, joined)
		}
	}
}
