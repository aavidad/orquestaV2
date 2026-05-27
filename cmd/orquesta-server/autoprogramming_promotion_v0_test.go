package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestAutoprogrammingPromotionConfigFromEnvV0OptInYRefsOpacasV0(t *testing.T) {
	projectDir := t.TempDir()
	stateDir := filepath.Join(t.TempDir(), "state")
	config := orquestaserver.ConfigV0{ProjectWorkDir: projectDir, StateDir: stateDir}

	disabled := autoprogrammingPromotionConfigFromEnvV0(config)
	if disabled.Enabled || disabled.Port != nil {
		t.Fatalf("disabled=%+v", disabled)
	}

	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_ENABLED", "1")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_REPO_REF", "repo-ref-autoprogramming-test")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_APP_REF", "app-ref-autoprogramming-test")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_COMMIT_MESSAGE", "test: promote staging")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_ENABLED", "1")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_COMMAND", "guardian-check")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_CURRENT_BIN", "orquesta-server-current")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_ARTIFACT_ROOT", "guardian-artifacts")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_ARTIFACT_MAX_BYTES", "1024")
	t.Setenv("ORQUESTA_SERVER_AUTOPROGRAMMING_PROMOTION_GUARDIAN_SKIP_HEALTH_EVIDENCE_REFS", "evidence-ref-skip-health")
	enabled := autoprogrammingPromotionConfigFromEnvV0(config)
	if !enabled.Enabled ||
		enabled.Port == nil ||
		enabled.RepoRef != "repo-ref-autoprogramming-test" ||
		enabled.AppRef != "app-ref-autoprogramming-test" ||
		enabled.CommitMessage != "test: promote staging" {
		t.Fatalf("enabled=%+v", enabled)
	}
	port, ok := enabled.Port.(serverAutoprogrammingPromotionPortV0)
	if !ok || !port.Guardian.Enabled || port.Guardian.Command != "guardian-check" ||
		port.Guardian.CurrentBin != "orquesta-server-current" ||
		port.Guardian.ArtifactRoot != "guardian-artifacts" ||
		port.Guardian.ArtifactMaxBytes != "1024" ||
		len(port.Guardian.SkipHealthEvidenceRefs) != 1 ||
		port.Guardian.Runner == nil {
		t.Fatalf("guardian=%+v ok=%v", port.Guardian, ok)
	}
}

func TestAutoprogrammingPromotionGuardianDistingueConfigInvalidaV0(t *testing.T) {
	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: "printf 'orquesta-guardian: guardian_config_invalid_duration\\n' >&2; exit 2",
	}
	_, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{ProjectDir: t.TempDir()},
	)
	if err == nil || !strings.Contains(err.Error(), "guardian_config_invalid_duration") {
		t.Fatalf("err=%v", err)
	}
}

func TestAutoprogrammingPromotionGuardianDistinguePathPolicyInvalidaV0(t *testing.T) {
	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: "printf 'orquesta-guardian: guardian_config_path_outside_allowed_root\\n' >&2; exit 2",
	}
	_, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{ProjectDir: t.TempDir()},
	)
	if err == nil || !strings.Contains(err.Error(), "guardian_config_path_outside_allowed_root") {
		t.Fatalf("err=%v", err)
	}
}

func TestAutoprogrammingPromotionGuardianEnvV0ConservaRefsOpacas(t *testing.T) {
	env := autoprogrammingPromotionGuardianEnvV0(nil, serverAutoprogrammingPromotionGuardianRequestV0{
		ProjectDir:  "/tmp/project",
		WorktreeRef: "worktree-ref-orquesta-server-idle-self-improvement",
		BranchRef:   "branch-ref-orquesta-server-idle-self-improvement",
	})
	joined := strings.Join(env, "\n")
	for _, want := range []string{
		"ORQUESTA_GUARDIAN_WORKTREE_REF=worktree-ref-orquesta-server-idle-self-improvement",
		"ORQUESTA_GUARDIAN_BRANCH_REF=branch-ref-orquesta-server-idle-self-improvement",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("env no conserva ref %q: %s", want, joined)
		}
	}
	if strings.Contains(joined, "refs/heads/") || strings.Contains(joined, ".git") {
		t.Fatalf("env convierte refs opacas en git/path: %s", joined)
	}
}

func TestAutoprogrammingPromotionGuardianConsumeResultadoEstructuradoV0(t *testing.T) {
	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: `printf '%s\n' '{"schema_version":"orquesta_guardian_result.v0","status":"candidate_promoted","promoted":true,"redaction_level":"refs_only","freshness":"2026-05-27T10:00:00Z","counters":{"evidence_refs":2},"evidence_refs":["evidence-ref-guardian-candidate-promoted","evidence-ref-extra"]}'`,
	}
	result, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{ProjectDir: t.TempDir()},
	)
	if err != nil ||
		result.Status != "candidate_promoted" ||
		!result.Promoted ||
		len(result.EvidenceRefs) != 2 ||
		result.EvidenceRefs[0] != "evidence-ref-guardian-candidate-promoted" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestAutoprogrammingPromotionGuardianConsumeVerificacionSinPromoteV0(t *testing.T) {
	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: `printf '%s\n' '{"schema_version":"orquesta_guardian_result.v0","status":"candidate_promoted","redaction_level":"refs_only","freshness":"2026-05-27T10:00:00Z","config_effective":{"promote":false,"skip_health":true,"repair_codex":false,"force_after_timeout":false,"shutdown_forced":false,"health_timeout_ms":1000,"command_timeout_ms":1000,"shutdown_timeout_ms":1000,"command_output_max_bytes":1024,"artifact_max_bytes":1024,"shutdown_queue_limit":1},"counters":{"evidence_refs":1},"evidence_refs":["evidence-ref-guardian-candidate-verified"]}'`,
	}
	result, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{ProjectDir: t.TempDir()},
	)
	if err != nil || !result.VerifiedOnly || result.Status != "candidate_promoted" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestAutoprogrammingPromotionGuardianBloqueaResultadoSensibleV0(t *testing.T) {
	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: `printf '%s\n' '{"schema_version":"orquesta_guardian_result.v0","status":"candidate_promoted","promoted":true,"project_dir":"/tmp/private","redaction_level":"refs_only","freshness":"2026-05-27T10:00:00Z","counters":{"evidence_refs":1},"evidence_refs":["evidence-ref-guardian-candidate-promoted"]}'`,
	}
	_, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{ProjectDir: t.TempDir()},
	)
	if err == nil || !strings.Contains(err.Error(), "guardian_result_invalid") {
		t.Fatalf("err=%v", err)
	}
}

func TestAutoprogrammingPromotionGuardianPropagaEstadoYEvidenciaV0(t *testing.T) {
	runner := shellAutoprogrammingPromotionGuardianRunnerV0{
		Command: `printf '%s\n' '{"schema_version":"orquesta_guardian_result.v0","status":"candidate_failed","phase":"test-1","redaction_level":"refs_only","freshness":"2026-05-27T10:00:00Z","counters":{"failed_count":1,"evidence_refs":1},"evidence_refs":["evidence-ref-guardian-candidate-failed"]}'; exit 1`,
	}
	result, err := runner.CheckAutoprogrammingPromotionGuardianV0(
		context.Background(),
		serverAutoprogrammingPromotionGuardianRequestV0{ProjectDir: t.TempDir()},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "candidate_failed") ||
		len(result.EvidenceRefs) != 1 ||
		result.EvidenceRefs[0] != "evidence-ref-guardian-candidate-failed" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestAutoprogrammingPromotionGuardianBlockedEffectV0(t *testing.T) {
	effect := autoprogrammingPromotionGuardianBlockedEffectV0(
		orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
			PromotionRef: "promotion-ref-guardian-test",
			RunRef:       "run-ref-guardian-test",
			WorktreeRef:  "worktree-ref-guardian-test",
			BranchRef:    "branch-ref-guardian-test",
			Status:       orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
			EvidenceRefs: []string{"evidence-ref-worktree"},
		},
		orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0{
			PromotionRef: "promotion-ref-guardian-test",
			RunRef:       "run-ref-guardian-test",
			WorktreeRef:  "worktree-ref-guardian-test",
			BranchRef:    "branch-ref-guardian-test",
		},
		serverAutoprogrammingPromotionGuardianResultV0{
			Status:       "candidate_failed",
			EvidenceRefs: []string{"evidence-ref-guardian"},
		},
		errors.New("guardian_candidate_failed"),
	)
	if effect.Status != orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0 ||
		!effect.Retryable ||
		!containsStringV0(effect.EvidenceRefs, "evidence-ref-guardian") ||
		len(effect.PromotionGuardianReceipts) != 1 ||
		effect.PromotionGuardianReceipts[0].Status !=
			orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptPromotionBlockedV0 ||
		len(effect.Issues) != 1 ||
		effect.Issues[0].Code != "guardian_candidate_failed" {
		t.Fatalf("effect=%+v", effect)
	}
}

func TestAutoprogrammingPromotionGuardianReceiptPromocionCausalV0(t *testing.T) {
	effect := autoprogrammingPromotionEffectWithGuardianV0(
		orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
			Status:         orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
			PromotionRef:   "promotion-ref-guardian-causal",
			RunRef:         "run-ref-guardian-causal",
			ProjectRef:     "project-ref-guardian-causal",
			WorktreeRef:    "worktree-ref-guardian-causal",
			BranchRef:      "branch-ref-guardian-causal",
			CommitRef:      "commit-ref-guardian-causal",
			CommitShortRef: "commit-short-ref-guardian-causal",
			ChangedPaths:   []string{"cmd/orquesta-server/autoprogramming_promotion_guardian_v0.go"},
			EvidenceRefs:   []string{"evidence-ref-worktree"},
		},
		orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0{
			PromotionRef: "promotion-ref-guardian-causal",
			RunRef:       "run-ref-guardian-causal",
			ProjectRef:   "project-ref-guardian-causal",
			WorktreeRef:  "worktree-ref-guardian-causal",
			BranchRef:    "branch-ref-guardian-causal",
		},
		serverAutoprogrammingPromotionGuardianResultV0{
			Status:             "candidate_promoted",
			Phase:              "promote",
			Promote:            true,
			Promoted:           true,
			ManifestRef:        "guardian-manifest-ref-001",
			ProjectRef:         "project-ref-guardian-causal",
			AppRef:             "app-ref-guardian-causal",
			RepoRef:            "repo-ref-guardian-causal",
			PromotionRef:       "promotion-ref-guardian-causal",
			RunRef:             "run-ref-guardian-causal",
			WorktreeRef:        "worktree-ref-guardian-causal",
			BranchRef:          "branch-ref-guardian-causal",
			CandidateHash:      "sha256:abc123",
			CandidateSizeBytes: 42,
			EvidenceRefs:       []string{"evidence-ref-guardian-candidate-promoted"},
		},
	)
	if effect.Status != orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0 ||
		len(effect.PromotionGuardianReceipts) != 1 {
		t.Fatalf("effect=%+v", effect)
	}
	receipt := effect.PromotionGuardianReceipts[0]
	if receipt.SchemaVersion != orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptSchemaVersionV0 ||
		receipt.Status != orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidatePromotedV0 ||
		receipt.ReceiptRef == "" ||
		receipt.GuardianAttemptRef == "" ||
		receipt.CandidateHash != "sha256:abc123" ||
		receipt.CandidateSizeBytes != 42 ||
		receipt.StagingEffectStatus != orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0 ||
		receipt.ManifestRef != "guardian-manifest-ref-001" {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestAutoprogrammingPromotionGuardianReceiptVerificadoNoPromueveActivoV0(t *testing.T) {
	effect := autoprogrammingPromotionEffectWithGuardianV0(
		orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
			Status:       orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0,
			PromotionRef: "promotion-ref-guardian-verified",
			RunRef:       "run-ref-guardian-verified",
			WorktreeRef:  "worktree-ref-guardian-verified",
			BranchRef:    "branch-ref-guardian-verified",
			EvidenceRefs: []string{"evidence-ref-worktree"},
		},
		orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0{
			PromotionRef: "promotion-ref-guardian-verified",
			RunRef:       "run-ref-guardian-verified",
			WorktreeRef:  "worktree-ref-guardian-verified",
			BranchRef:    "branch-ref-guardian-verified",
		},
		serverAutoprogrammingPromotionGuardianResultV0{
			Status:       "candidate_promoted",
			Phase:        "verified",
			Promote:      false,
			VerifiedOnly: true,
			EvidenceRefs: []string{"evidence-ref-guardian-candidate-verified"},
		},
	)
	if effect.Status != orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0 ||
		!effect.Retryable ||
		len(effect.PromotionGuardianReceipts) != 1 ||
		effect.PromotionGuardianReceipts[0].Status !=
			orquestaautoprogramming.AutoprogrammingPromotionGuardianReceiptCandidateVerifiedV0 ||
		len(effect.Issues) != 1 ||
		effect.Issues[0].Code != "guardian_candidate_verified_without_promotion" {
		t.Fatalf("effect=%+v", effect)
	}
}
