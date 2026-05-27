package main

import (
	"path/filepath"
	"strings"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestAutoprogrammingPromotionGuardianRepairCodexConfigFromEnvV0(t *testing.T) {
	t.Setenv(envServerAutoprogrammingPromotionEnabledV0, "1")
	t.Setenv(envServerAutoprogrammingPromotionGuardianEnabledV0, "1")
	t.Setenv(envServerAutoprogrammingPromotionGuardianRepairCodexV0, "1")
	t.Setenv(envServerAutoprogrammingPromotionGuardianRepairCodexWriteSetV0, "cmd/orquesta-guardian,cmd/orquesta-server")
	t.Setenv(envServerAutoprogrammingPromotionGuardianRepairCodexRequiredTestsV0, "go test -count=1 ./cmd/orquesta-guardian")
	t.Setenv(envServerAutoprogrammingPromotionGuardianRepairCodexWorktreeRefV0, "worktree-ref-orquesta-server-idle-self-improvement")
	t.Setenv(envServerAutoprogrammingPromotionGuardianRepairCodexBranchRefV0, "branch-ref-orquesta-server-idle-self-improvement")
	t.Setenv(envServerAutoprogrammingPromotionGuardianRepairCodexSandboxV0, "workspace-write")
	t.Setenv(envServerAutoprogrammingPromotionGuardianRepairCodexReasoningV0, "medium")
	t.Setenv(envServerAutoprogrammingPromotionGuardianRepairCodexRuntimeDirV0, "guardian-repair-runtime")

	enabled := autoprogrammingPromotionConfigFromEnvV0(orquestaserver.ConfigV0{
		ProjectWorkDir: t.TempDir(),
		StateDir:       filepath.Join(t.TempDir(), "state"),
	})
	port, ok := enabled.Port.(serverAutoprogrammingPromotionPortV0)
	if !ok || !port.Guardian.RepairCodex ||
		len(port.Guardian.RepairCodexWriteSet) != 2 ||
		port.Guardian.RepairCodexWorktreeRef != "worktree-ref-orquesta-server-idle-self-improvement" ||
		port.Guardian.RepairCodexBranchRef != "branch-ref-orquesta-server-idle-self-improvement" ||
		port.Guardian.RepairCodexSandbox != "workspace-write" ||
		port.Guardian.RepairCodexReasoning != "medium" {
		t.Fatalf("guardian repair codex=%+v ok=%v", port.Guardian, ok)
	}
}

func TestAutoprogrammingPromotionGuardianRepairCodexEnvV0(t *testing.T) {
	request := autoprogrammingPromotionGuardianRequestV0(
		serverAutoprogrammingPromotionPortV0{
			ProjectWorkDir: t.TempDir(),
			Guardian: serverAutoprogrammingPromotionGuardianV0{
				RepairCodex:              true,
				RepairCodexWriteSet:      []string{"cmd/orquesta-guardian"},
				RepairCodexRequiredTests: []string{"go test -count=1 ./cmd/orquesta-guardian"},
				RepairCodexSandbox:       "workspace-write",
				RepairCodexReasoning:     "medium",
			},
		},
		orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0{
			PromotionRef: "promotion-ref-guardian-repair",
			RunRef:       "run-ref-guardian-repair",
			WorktreeRef:  "worktree-ref-orquesta-server-idle-self-improvement",
			BranchRef:    "branch-ref-orquesta-server-idle-self-improvement",
			WriteSet:     []string{"cmd/orquesta-server"},
		},
	)
	env := strings.Join(autoprogrammingPromotionGuardianEnvV0(nil, request), "\n")
	for _, want := range []string{
		"ORQUESTA_GUARDIAN_REPAIR_CODEX=1",
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_WRITE_SET=cmd/orquesta-guardian,cmd/orquesta-server",
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_REQUIRED_TESTS=go test -count=1 ./cmd/orquesta-guardian",
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_WORKTREE_REF=worktree-ref-orquesta-server-idle-self-improvement",
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_BRANCH_REF=branch-ref-orquesta-server-idle-self-improvement",
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_RUN_REF=run-ref-guardian-repair",
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_PROMOTION_REF=promotion-ref-guardian-repair",
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX=workspace-write",
		"ORQUESTA_GUARDIAN_REPAIR_CODEX_REASONING_EFFORT=medium",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("env falta %q: %s", want, env)
		}
	}
	if strings.Contains(env, "refs/heads/") || strings.Contains(env, ".git") {
		t.Fatalf("env convierte refs opacas: %s", env)
	}
}
