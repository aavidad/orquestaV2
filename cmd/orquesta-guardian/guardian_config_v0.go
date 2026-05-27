package main

import (
	"flag"
	"io"
	"os"
	"strings"
	"time"
)

func parseGuardianConfigV0(args []string) (guardianConfigV0, error) {
	return parseGuardianConfigForCommandV0(args, true)
}

func parseGuardianShutdownConfigV0(args []string) (guardianConfigV0, error) {
	return parseGuardianConfigForCommandV0(args, false)
}

func parseGuardianConfigForCommandV0(args []string, requireCurrentBin bool) (guardianConfigV0, error) {
	repairCodex, err := strictEnvBoolOrDefaultV0("ORQUESTA_GUARDIAN_REPAIR_CODEX", false)
	if err != nil {
		return guardianConfigV0{}, err
	}
	repairCodexBroadSandbox, err := strictEnvBoolOrDefaultV0("ORQUESTA_GUARDIAN_REPAIR_CODEX_ALLOW_BROAD_SANDBOX", false)
	if err != nil {
		return guardianConfigV0{}, err
	}
	promote, err := strictEnvBoolOrDefaultV0("ORQUESTA_GUARDIAN_PROMOTE", true)
	if err != nil {
		return guardianConfigV0{}, err
	}
	skipHealth, err := strictEnvBoolOrDefaultV0("ORQUESTA_GUARDIAN_SKIP_HEALTH", false)
	if err != nil {
		return guardianConfigV0{}, err
	}
	healthTimeout, err := strictEnvDurationOrDefaultV0("ORQUESTA_GUARDIAN_HEALTH_TIMEOUT", 20*time.Second)
	if err != nil {
		return guardianConfigV0{}, err
	}
	commandTimeout, err := strictEnvDurationOrDefaultV0("ORQUESTA_GUARDIAN_COMMAND_TIMEOUT", 10*time.Minute)
	if err != nil {
		return guardianConfigV0{}, err
	}
	commandOutputMaxBytes, err := strictEnvPositiveInt64OrDefaultV0(envGuardianCommandOutputMaxBytesV0, defaultGuardianCommandOutputMaxBytesV0)
	if err != nil {
		return guardianConfigV0{}, err
	}
	artifactMaxBytes, err := strictEnvPositiveInt64OrDefaultV0("ORQUESTA_GUARDIAN_ARTIFACT_MAX_BYTES", defaultGuardianArtifactMaxBytesV0)
	if err != nil {
		return guardianConfigV0{}, err
	}
	envAllowlist, err := strictGuardianEnvAllowlistFromEnvV0()
	if err != nil {
		return guardianConfigV0{}, err
	}
	serverPID, err := strictEnvIntOrDefaultV0("ORQUESTA_GUARDIAN_SERVER_PID", 0)
	if err != nil {
		return guardianConfigV0{}, err
	}
	shutdownNow, err := strictEnvBoolOrDefaultV0("ORQUESTA_GUARDIAN_SHUTDOWN_NOW", false)
	if err != nil {
		return guardianConfigV0{}, err
	}
	shutdownForced, err := strictEnvBoolOrDefaultV0("ORQUESTA_GUARDIAN_SHUTDOWN_FORCED", false)
	if err != nil {
		return guardianConfigV0{}, err
	}
	forceAfterTimeout, err := strictEnvBoolOrDefaultV0("ORQUESTA_GUARDIAN_FORCE_AFTER_TIMEOUT", false)
	if err != nil {
		return guardianConfigV0{}, err
	}
	shutdownTimeout, err := strictEnvDurationOrDefaultV0("ORQUESTA_GUARDIAN_SHUTDOWN_TIMEOUT", 2*time.Minute)
	if err != nil {
		return guardianConfigV0{}, err
	}
	shutdownQueueLimit, err := strictEnvIntOrDefaultV0("ORQUESTA_GUARDIAN_SHUTDOWN_QUEUE_LIMIT", 500)
	if err != nil {
		return guardianConfigV0{}, err
	}
	leaseTTL, err := strictEnvDurationOrDefaultV0("ORQUESTA_GUARDIAN_LEASE_TTL", 30*time.Minute)
	if err != nil {
		return guardianConfigV0{}, err
	}
	repairMaxAttempts, err := strictEnvPositiveIntOrDefaultV0("ORQUESTA_GUARDIAN_REPAIR_MAX_ATTEMPTS", 1)
	if err != nil {
		return guardianConfigV0{}, err
	}
	flags := flag.NewFlagSet("orquesta-guardian", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var tests repeatedStringFlagV0
	var repairWriteSet repeatedStringFlagV0
	var repairRequiredTests repeatedStringFlagV0
	var repairRetryEvidence repeatedStringFlagV0
	var commandEffectEvidence repeatedStringFlagV0
	var skipHealthEvidence repeatedStringFlagV0
	config := guardianConfigV0{
		ProjectDir:                    envOrDefaultV0("ORQUESTA_GUARDIAN_PROJECT_DIR", "."),
		StateDir:                      envOrDefaultV0("ORQUESTA_GUARDIAN_STATE_DIR", ".orquesta-guardian"),
		CurrentBin:                    strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_CURRENT_BIN")),
		CandidateBin:                  strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_CANDIDATE_BIN")),
		LastGoodBin:                   strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_LAST_GOOD_BIN")),
		ArtifactRoot:                  strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_ARTIFACT_ROOT")),
		BuildCommand:                  strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_BUILD_COMMAND")),
		RepairCommand:                 strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_COMMAND")),
		CommandEffectEvidenceRefs:     splitEnvListV0(os.Getenv(envGuardianCommandEffectEvidenceRefsV0)),
		SkipHealthEvidenceRefs:        splitEnvListV0(os.Getenv("ORQUESTA_GUARDIAN_SKIP_HEALTH_EVIDENCE_REFS")),
		RepairCodex:                   repairCodex,
		RepairCodexSandbox:            envOrDefaultV0("ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX", "workspace-write"),
		RepairCodexEffort:             envOrDefaultV0("ORQUESTA_GUARDIAN_REPAIR_CODEX_REASONING_EFFORT", "medium"),
		RepairCodexRuntimeDir:         strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_RUNTIME_DIR")),
		RepairCodexWorktreeRef:        strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_WORKTREE_REF")),
		RepairCodexBranchRef:          strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_BRANCH_REF")),
		RepairCodexRunRef:             strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_RUN_REF")),
		RepairCodexPromotionRef:       strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_PROMOTION_REF")),
		RepairCodexAllowBroadSandbox:  repairCodexBroadSandbox,
		RepairCodexSandboxEvidenceRef: strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX_EVIDENCE_REF")),
		RepairMaxAttempts:             repairMaxAttempts,
		RepairRetryEvidenceRefs:       splitEnvListV0(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_RETRY_EVIDENCE_REFS")),
		Promote:                       promote,
		SkipHealth:                    skipHealth,
		HealthTimeout:                 healthTimeout,
		CommandTimeout:                commandTimeout,
		CommandOutputMaxBytes:         commandOutputMaxBytes,
		ArtifactMaxBytes:              artifactMaxBytes,
		EnvAllowlist:                  envAllowlist,
		CandidateAddr:                 strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_CANDIDATE_ADDR")),
		ServerAddr:                    strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_SERVER_ADDR")),
		ServerPID:                     serverPID,
		AttemptRef:                    strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_ATTEMPT_REF")),
		PromotionRef:                  strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_PROMOTION_REF")),
		RunRef:                        strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_RUN_REF")),
		WorktreeRef:                   strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_WORKTREE_REF")),
		BranchRef:                     strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_BRANCH_REF")),
		ShutdownRef:                   strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_SHUTDOWN_REF")),
		LeaseTTL:                      leaseTTL,
		ShutdownNow:                   shutdownNow,
		ShutdownForced:                shutdownForced,
		ForceAfterTimeout:             forceAfterTimeout,
		ShutdownEscalationEvidenceRef: strings.TrimSpace(
			os.Getenv("ORQUESTA_GUARDIAN_SHUTDOWN_ESCALATION_EVIDENCE_REF"),
		),
		ShutdownTimeout:    shutdownTimeout,
		ShutdownQueueLimit: shutdownQueueLimit,
		OccurredAt:         time.Now().UTC(),
	}
	tests = append(tests, splitEnvCommandsV0(os.Getenv("ORQUESTA_GUARDIAN_TEST_COMMANDS"))...)
	repairWriteSet = append(repairWriteSet, splitEnvListV0(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_WRITE_SET"))...)
	repairRequiredTests = append(repairRequiredTests, splitEnvCommandsV0(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_REQUIRED_TESTS"))...)
	repairRetryEvidence = append(repairRetryEvidence, config.RepairRetryEvidenceRefs...)
	commandEffectEvidence = append(commandEffectEvidence, config.CommandEffectEvidenceRefs...)
	skipHealthEvidence = append(skipHealthEvidence, config.SkipHealthEvidenceRefs...)
	flags.StringVar(&config.ProjectDir, "project-dir", config.ProjectDir, "directorio del repo")
	flags.StringVar(&config.StateDir, "state-dir", config.StateDir, "directorio de estado del guardian")
	flags.StringVar(&config.CurrentBin, "current-bin", config.CurrentBin, "binario vivo a promocionar")
	flags.StringVar(&config.CandidateBin, "candidate-bin", config.CandidateBin, "binario candidato")
	flags.StringVar(&config.LastGoodBin, "last-good-bin", config.LastGoodBin, "backup last_good")
	flags.StringVar(&config.ArtifactRoot, "artifact-root", config.ArtifactRoot, "raiz declarada para artefactos binarios")
	flags.StringVar(&config.BuildCommand, "build-command", config.BuildCommand, "comando shell de build")
	flags.Var(&tests, "test-command", "comando shell de test; repetible")
	flags.StringVar(&config.RepairCommand, "repair-command", config.RepairCommand, "comando shell externo de reparacion opt-in")
	flags.Var(&commandEffectEvidence, "command-effect-evidence-ref", "evidence ref para comando no canonico o efecto externo")
	flags.Var(&skipHealthEvidence, "skip-health-evidence-ref", "evidence ref break-glass para saltar healthcheck")
	flags.BoolVar(&config.RepairCodex, "repair-codex", config.RepairCodex, "lanzar un agente Codex externo opt-in si falla candidato")
	flags.StringVar(&config.RepairCodexSandbox, "repair-codex-sandbox", config.RepairCodexSandbox, "sandbox para agente Codex reparador")
	flags.StringVar(&config.RepairCodexEffort, "repair-codex-reasoning-effort", config.RepairCodexEffort, "reasoning effort del reparador Codex")
	flags.StringVar(&config.RepairCodexRuntimeDir, "repair-codex-runtime-dir", config.RepairCodexRuntimeDir, "runtime dir del reparador Codex")
	flags.Var(&repairWriteSet, "repair-codex-write-set", "write-set cerrado del reparador Codex; repetible o separado por comas")
	flags.Var(&repairRequiredTests, "repair-codex-required-test", "test requerido del reparador Codex; repetible")
	flags.StringVar(&config.RepairCodexWorktreeRef, "repair-codex-worktree-ref", config.RepairCodexWorktreeRef, "worktree_ref opaca del reparador Codex")
	flags.StringVar(&config.RepairCodexBranchRef, "repair-codex-branch-ref", config.RepairCodexBranchRef, "branch_ref opaca del reparador Codex")
	flags.StringVar(&config.RepairCodexRunRef, "repair-codex-run-ref", config.RepairCodexRunRef, "run_ref opaca del reparador Codex")
	flags.StringVar(&config.RepairCodexPromotionRef, "repair-codex-promotion-ref", config.RepairCodexPromotionRef, "promotion_ref opaca del reparador Codex")
	flags.BoolVar(&config.RepairCodexAllowBroadSandbox, "repair-codex-allow-broad-sandbox", config.RepairCodexAllowBroadSandbox, "opt-in auditado para sandbox amplio")
	flags.StringVar(&config.RepairCodexSandboxEvidenceRef, "repair-codex-sandbox-evidence-ref", config.RepairCodexSandboxEvidenceRef, "evidence ref para sandbox amplio")
	flags.IntVar(&config.RepairMaxAttempts, "repair-max-attempts", config.RepairMaxAttempts, "presupuesto maximo de reparaciones por intento/promocion")
	flags.Var(&repairRetryEvidence, "repair-retry-evidence-ref", "evidence ref que autoriza nuevo intento sobre el mismo failure packet")
	flags.BoolVar(&config.Promote, "promote", config.Promote, "promocionar candidato si pasa")
	flags.BoolVar(&config.SkipHealth, "skip-health", config.SkipHealth, "saltar healthcheck del candidato")
	flags.DurationVar(&config.HealthTimeout, "health-timeout", config.HealthTimeout, "timeout de healthcheck")
	flags.DurationVar(&config.CommandTimeout, "command-timeout", config.CommandTimeout, "timeout por comando shell")
	flags.Int64Var(&config.CommandOutputMaxBytes, "command-output-max-bytes", config.CommandOutputMaxBytes, "limite de bytes redactados por comando")
	flags.Int64Var(&config.ArtifactMaxBytes, "artifact-max-bytes", config.ArtifactMaxBytes, "limite de bytes por artefacto binario")
	flags.StringVar(&config.CandidateAddr, "candidate-addr", config.CandidateAddr, "addr temporal para candidato")
	flags.StringVar(&config.ServerAddr, "server-addr", config.ServerAddr, "addr del servidor vivo para shutdown cooperativo")
	flags.IntVar(&config.ServerPID, "server-pid", config.ServerPID, "pid del servidor vivo para senalizar tras shutdown_ready")
	flags.StringVar(&config.AttemptRef, "attempt-ref", config.AttemptRef, "ref opaca del intento de guardian")
	flags.StringVar(&config.PromotionRef, "promotion-ref", config.PromotionRef, "ref opaca de promocion")
	flags.StringVar(&config.ShutdownRef, "shutdown-ref", config.ShutdownRef, "ref opaca de apagado")
	flags.DurationVar(&config.LeaseTTL, "lease-ttl", config.LeaseTTL, "ttl del lease del guardian")
	flags.BoolVar(&config.ShutdownNow, "now", config.ShutdownNow, "apagado forzado inmediato, sin esperar shutdown_ready")
	flags.BoolVar(&config.ShutdownForced, "shutdown-forced", config.ShutdownForced, "usar forced=true en shutdown")
	flags.BoolVar(&config.ForceAfterTimeout, "force-after-timeout", config.ForceAfterTimeout, "break-glass opt-in: forzar shutdown al vencer el timeout cooperativo")
	flags.StringVar(&config.ShutdownEscalationEvidenceRef, "shutdown-escalation-evidence-ref", config.ShutdownEscalationEvidenceRef, "evidence ref publica para escalado break-glass")
	flags.DurationVar(&config.ShutdownTimeout, "shutdown-timeout", config.ShutdownTimeout, "timeout total de shutdown cooperativo")
	flags.IntVar(&config.ShutdownQueueLimit, "shutdown-queue-limit", config.ShutdownQueueLimit, "limite de runs por peticion shutdown")
	if err := flags.Parse(args); err != nil {
		return guardianConfigV0{}, guardianConfigParseErrorV0{Code: guardianConfigInvalidFlagV0}
	}
	if err := rejectGuardianExtraArgsV0(flags); err != nil {
		return guardianConfigV0{}, err
	}
	config.TestCommands = compactStringsV0(tests)
	config.RepairCodexWriteSet = compactStringsV0(expandGuardianListValuesV0(repairWriteSet))
	config.RepairCodexRequiredTests = compactStringsV0(repairRequiredTests)
	config.RepairRetryEvidenceRefs = compactStringsV0(expandGuardianListValuesV0(repairRetryEvidence))
	config.CommandEffectEvidenceRefs = compactStringsV0(commandEffectEvidence)
	config.SkipHealthEvidenceRefs = compactStringsV0(expandGuardianListValuesV0(skipHealthEvidence))
	if err := validateParsedGuardianConfigV0(config); err != nil {
		return guardianConfigV0{}, err
	}
	return normalizeGuardianConfigForCommandV0(config, requireCurrentBin)
}
