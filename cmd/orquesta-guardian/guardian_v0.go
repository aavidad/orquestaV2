package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	guardianResultSchemaVersionV0       = "orquesta_guardian_result.v0"
	guardianRepairPacketSchemaVersionV0 = "orquesta_guardian_repair_packet.v0"

	guardianStatusPromotedV0        = "candidate_promoted"
	guardianStatusCandidateFailedV0 = "candidate_failed"
	guardianStatusRestoredV0        = "last_good_restored"
	guardianStatusShutdownReadyV0   = "shutdown_ready"
)

type guardianConfigV0 struct {
	ProjectDir            string
	StateDir              string
	CurrentBin            string
	CandidateBin          string
	LastGoodBin           string
	BuildCommand          string
	TestCommands          []string
	RepairCommand         string
	RepairCodex           bool
	RepairCodexSandbox    string
	RepairCodexEffort     string
	RepairCodexRuntimeDir string
	Promote               bool
	SkipHealth            bool
	HealthTimeout         time.Duration
	CommandTimeout        time.Duration
	CandidateAddr         string
	ServerAddr            string
	ServerPID             int
	ShutdownNow           bool
	ShutdownForced        bool
	ForceAfterTimeout     bool
	ShutdownTimeout       time.Duration
	ShutdownQueueLimit    int
	OccurredAt            time.Time
	ManifestPath          string
	RepairPacketPath      string
}

type guardianResultV0 struct {
	SchemaVersion    string                    `json:"schema_version"`
	Status           string                    `json:"status"`
	Phase            string                    `json:"phase,omitempty"`
	Promoted         bool                      `json:"promoted,omitempty"`
	Restored         bool                      `json:"restored,omitempty"`
	RepairStarted    bool                      `json:"repair_started,omitempty"`
	ManifestPath     string                    `json:"manifest_path,omitempty"`
	RepairPacketPath string                    `json:"repair_packet_path,omitempty"`
	ProjectDir       string                    `json:"project_dir,omitempty"`
	CurrentBin       string                    `json:"current_bin,omitempty"`
	CandidateBin     string                    `json:"candidate_bin,omitempty"`
	LastGoodBin      string                    `json:"last_good_bin,omitempty"`
	Commands         []guardianCommandResultV0 `json:"commands,omitempty"`
	EvidenceRefs     []string                  `json:"evidence_refs,omitempty"`
	Message          string                    `json:"message,omitempty"`
	Shutdown         *guardianShutdownResultV0 `json:"shutdown,omitempty"`
}

type guardianShutdownResultV0 struct {
	Estado                  string `json:"estado,omitempty"`
	Status                  string `json:"status,omitempty"`
	ShutdownReady           bool   `json:"shutdown_ready,omitempty"`
	RunsRequested           int    `json:"runs_requested,omitempty"`
	RunsStopped             int    `json:"runs_stopped,omitempty"`
	AgentsInFlight          int    `json:"agents_in_flight,omitempty"`
	CheckpointsPending      int    `json:"checkpoints_pending,omitempty"`
	CheckpointAgentsPending int    `json:"checkpoint_agents_pending,omitempty"`
}

type guardianCommandResultV0 struct {
	Phase      string `json:"phase"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	OutputPath string `json:"output_path,omitempty"`
	Error      string `json:"error,omitempty"`
}

type guardianRepairPacketV0 struct {
	SchemaVersion string                    `json:"schema_version"`
	FailurePhase  string                    `json:"failure_phase"`
	Summary       string                    `json:"summary"`
	ProjectDir    string                    `json:"project_dir"`
	CurrentBin    string                    `json:"current_bin"`
	CandidateBin  string                    `json:"candidate_bin"`
	LastGoodBin   string                    `json:"last_good_bin"`
	Commands      []guardianCommandResultV0 `json:"commands,omitempty"`
	EvidenceRefs  []string                  `json:"evidence_refs,omitempty"`
	CreatedAt     string                    `json:"created_at"`
}

type repeatedStringFlagV0 []string

func (values *repeatedStringFlagV0) String() string {
	return strings.Join(*values, "\n")
}

func (values *repeatedStringFlagV0) Set(value string) error {
	value = strings.TrimSpace(value)
	if value != "" {
		*values = append(*values, value)
	}
	return nil
}

func runMain(args []string, stdout io.Writer, stderr io.Writer) int {
	command := "check-promote"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command = args[0]
		args = args[1:]
	}
	switch command {
	case "check-promote":
		config, err := parseGuardianConfigV0(args)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-guardian: %v\n", err)
			return 2
		}
		result := runGuardianCheckPromoteV0(context.Background(), config)
		_ = json.NewEncoder(stdout).Encode(result)
		if result.Status != guardianStatusPromotedV0 {
			return 1
		}
		return 0
	case "restore-last-good":
		config, err := parseGuardianConfigV0(args)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-guardian: %v\n", err)
			return 2
		}
		result := restoreLastGoodCommandV0(config)
		_ = json.NewEncoder(stdout).Encode(result)
		if !result.Restored {
			return 1
		}
		return 0
	case "shutdown-server":
		config, err := parseGuardianConfigV0(args)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "orquesta-guardian: %v\n", err)
			return 2
		}
		result := runGuardianShutdownServerV0(context.Background(), config)
		_ = json.NewEncoder(stdout).Encode(result)
		if result.Status != guardianStatusShutdownReadyV0 {
			return 1
		}
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "comando no soportado: %s\n", command)
		return 2
	}
}

func parseGuardianConfigV0(args []string) (guardianConfigV0, error) {
	flags := flag.NewFlagSet("orquesta-guardian", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var tests repeatedStringFlagV0
	config := guardianConfigV0{
		ProjectDir:            envOrDefaultV0("ORQUESTA_GUARDIAN_PROJECT_DIR", "."),
		StateDir:              envOrDefaultV0("ORQUESTA_GUARDIAN_STATE_DIR", ".orquesta-guardian"),
		CurrentBin:            strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_CURRENT_BIN")),
		CandidateBin:          strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_CANDIDATE_BIN")),
		LastGoodBin:           strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_LAST_GOOD_BIN")),
		BuildCommand:          strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_BUILD_COMMAND")),
		RepairCommand:         strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_COMMAND")),
		RepairCodex:           envBoolOrDefaultV0("ORQUESTA_GUARDIAN_REPAIR_CODEX", false),
		RepairCodexSandbox:    envOrDefaultV0("ORQUESTA_GUARDIAN_REPAIR_CODEX_SANDBOX", "danger-full-access"),
		RepairCodexEffort:     envOrDefaultV0("ORQUESTA_GUARDIAN_REPAIR_CODEX_REASONING_EFFORT", "high"),
		RepairCodexRuntimeDir: strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_REPAIR_CODEX_RUNTIME_DIR")),
		Promote:               envBoolOrDefaultV0("ORQUESTA_GUARDIAN_PROMOTE", true),
		SkipHealth:            envBoolOrDefaultV0("ORQUESTA_GUARDIAN_SKIP_HEALTH", false),
		HealthTimeout:         envDurationOrDefaultV0("ORQUESTA_GUARDIAN_HEALTH_TIMEOUT", 20*time.Second),
		CommandTimeout:        envDurationOrDefaultV0("ORQUESTA_GUARDIAN_COMMAND_TIMEOUT", 10*time.Minute),
		CandidateAddr:         strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_CANDIDATE_ADDR")),
		ServerAddr:            strings.TrimSpace(os.Getenv("ORQUESTA_GUARDIAN_SERVER_ADDR")),
		ServerPID:             envIntOrDefaultV0("ORQUESTA_GUARDIAN_SERVER_PID", 0),
		ShutdownNow:           envBoolOrDefaultV0("ORQUESTA_GUARDIAN_SHUTDOWN_NOW", false),
		ShutdownForced:        envBoolOrDefaultV0("ORQUESTA_GUARDIAN_SHUTDOWN_FORCED", false),
		ForceAfterTimeout:     envBoolOrDefaultV0("ORQUESTA_GUARDIAN_FORCE_AFTER_TIMEOUT", true),
		ShutdownTimeout:       envDurationOrDefaultV0("ORQUESTA_GUARDIAN_SHUTDOWN_TIMEOUT", 2*time.Minute),
		ShutdownQueueLimit:    envIntOrDefaultV0("ORQUESTA_GUARDIAN_SHUTDOWN_QUEUE_LIMIT", 500),
		OccurredAt:            time.Now().UTC(),
	}
	tests = append(tests, splitEnvCommandsV0(os.Getenv("ORQUESTA_GUARDIAN_TEST_COMMANDS"))...)
	flags.StringVar(&config.ProjectDir, "project-dir", config.ProjectDir, "directorio del repo")
	flags.StringVar(&config.StateDir, "state-dir", config.StateDir, "directorio de estado del guardian")
	flags.StringVar(&config.CurrentBin, "current-bin", config.CurrentBin, "binario vivo a promocionar")
	flags.StringVar(&config.CandidateBin, "candidate-bin", config.CandidateBin, "binario candidato")
	flags.StringVar(&config.LastGoodBin, "last-good-bin", config.LastGoodBin, "backup last_good")
	flags.StringVar(&config.BuildCommand, "build-command", config.BuildCommand, "comando shell de build")
	flags.Var(&tests, "test-command", "comando shell de test; repetible")
	flags.StringVar(&config.RepairCommand, "repair-command", config.RepairCommand, "comando shell externo de reparacion opt-in")
	flags.BoolVar(&config.RepairCodex, "repair-codex", config.RepairCodex, "lanzar un agente Codex externo opt-in si falla candidato")
	flags.StringVar(&config.RepairCodexSandbox, "repair-codex-sandbox", config.RepairCodexSandbox, "sandbox para agente Codex reparador")
	flags.StringVar(&config.RepairCodexEffort, "repair-codex-reasoning-effort", config.RepairCodexEffort, "reasoning effort del reparador Codex")
	flags.StringVar(&config.RepairCodexRuntimeDir, "repair-codex-runtime-dir", config.RepairCodexRuntimeDir, "runtime dir del reparador Codex")
	flags.BoolVar(&config.Promote, "promote", config.Promote, "promocionar candidato si pasa")
	flags.BoolVar(&config.SkipHealth, "skip-health", config.SkipHealth, "saltar healthcheck del candidato")
	flags.DurationVar(&config.HealthTimeout, "health-timeout", config.HealthTimeout, "timeout de healthcheck")
	flags.DurationVar(&config.CommandTimeout, "command-timeout", config.CommandTimeout, "timeout por comando shell")
	flags.StringVar(&config.CandidateAddr, "candidate-addr", config.CandidateAddr, "addr temporal para candidato")
	flags.StringVar(&config.ServerAddr, "server-addr", config.ServerAddr, "addr del servidor vivo para shutdown cooperativo")
	flags.IntVar(&config.ServerPID, "server-pid", config.ServerPID, "pid del servidor vivo para senalizar tras shutdown_ready")
	flags.BoolVar(&config.ShutdownNow, "now", config.ShutdownNow, "apagado forzado inmediato, sin esperar shutdown_ready")
	flags.BoolVar(&config.ShutdownForced, "shutdown-forced", config.ShutdownForced, "usar forced=true en shutdown")
	flags.BoolVar(&config.ForceAfterTimeout, "force-after-timeout", config.ForceAfterTimeout, "forzar shutdown al vencer el timeout cooperativo")
	flags.DurationVar(&config.ShutdownTimeout, "shutdown-timeout", config.ShutdownTimeout, "timeout total de shutdown cooperativo")
	flags.IntVar(&config.ShutdownQueueLimit, "shutdown-queue-limit", config.ShutdownQueueLimit, "limite de runs por peticion shutdown")
	if err := flags.Parse(args); err != nil {
		return guardianConfigV0{}, err
	}
	config.TestCommands = compactStringsV0(tests)
	return normalizeGuardianConfigV0(config)
}

func normalizeGuardianConfigV0(config guardianConfigV0) (guardianConfigV0, error) {
	projectDir, err := filepath.Abs(strings.TrimSpace(config.ProjectDir))
	if err != nil {
		return guardianConfigV0{}, err
	}
	config.ProjectDir = projectDir
	config.StateDir = absPathFromBaseV0(config.ProjectDir, config.StateDir)
	if config.CurrentBin == "" {
		return guardianConfigV0{}, errors.New("current-bin requerido")
	}
	config.CurrentBin = absPathFromBaseV0(config.ProjectDir, config.CurrentBin)
	if config.CandidateBin == "" {
		config.CandidateBin = filepath.Join(config.StateDir, "candidate", filepath.Base(config.CurrentBin))
	}
	config.CandidateBin = absPathFromBaseV0(config.ProjectDir, config.CandidateBin)
	if config.LastGoodBin == "" {
		config.LastGoodBin = filepath.Join(config.StateDir, "last_good", filepath.Base(config.CurrentBin))
	}
	config.LastGoodBin = absPathFromBaseV0(config.ProjectDir, config.LastGoodBin)
	if config.RepairCodexRuntimeDir == "" {
		config.RepairCodexRuntimeDir = filepath.Join(config.StateDir, "repair-codex-runtime")
	}
	config.RepairCodexRuntimeDir = absPathFromBaseV0(config.ProjectDir, config.RepairCodexRuntimeDir)
	if config.BuildCommand == "" {
		config.BuildCommand = "go build -o " + shellQuoteV0(config.CandidateBin) + " ./cmd/orquesta-server"
	}
	if config.HealthTimeout <= 0 {
		config.HealthTimeout = 20 * time.Second
	}
	if config.CommandTimeout <= 0 {
		config.CommandTimeout = 10 * time.Minute
	}
	if config.ShutdownNow {
		config.ShutdownForced = true
	}
	stamp := config.OccurredAt.Format("20060102T150405Z")
	config.ManifestPath = filepath.Join(config.StateDir, "manifests", "guardian-"+stamp+".json")
	config.RepairPacketPath = filepath.Join(config.StateDir, "repair", "repair-"+stamp+".json")
	return config, nil
}

func runGuardianCheckPromoteV0(ctx context.Context, config guardianConfigV0) guardianResultV0 {
	result := baseGuardianResultV0(config)
	if err := ensureGuardianDirsV0(config); err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Phase = "prepare"
		result.Message = err.Error()
		writeGuardianManifestV0(config, result)
		return result
	}
	build := runGuardianShellCommandV0(ctx, config, "build", expandGuardianCommandV0(config.BuildCommand, config))
	result.Commands = append(result.Commands, build)
	if build.ExitCode != 0 {
		return failGuardianCandidateV0(ctx, config, result, "build", "candidate build failed")
	}
	for index, testCommand := range config.TestCommands {
		phase := "test-" + strconv.Itoa(index+1)
		test := runGuardianShellCommandV0(ctx, config, phase, expandGuardianCommandV0(testCommand, config))
		result.Commands = append(result.Commands, test)
		if test.ExitCode != 0 {
			return failGuardianCandidateV0(ctx, config, result, phase, "candidate required tests failed")
		}
	}
	if !config.SkipHealth {
		health := runGuardianCandidateHealthcheckV0(ctx, config)
		result.Commands = append(result.Commands, health)
		if health.ExitCode != 0 {
			return failGuardianCandidateV0(ctx, config, result, "healthcheck", "candidate healthcheck failed")
		}
	}
	if !config.Promote {
		result.Status = guardianStatusPromotedV0
		result.Phase = "verified"
		result.Message = "candidate verified without promotion"
		result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-candidate-verified")
		writeGuardianManifestV0(config, result)
		return result
	}
	if err := promoteGuardianCandidateV0(config); err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Phase = "promote"
		result.Message = err.Error()
		writeGuardianManifestV0(config, result)
		return result
	}
	result.Status = guardianStatusPromotedV0
	result.Phase = "promote"
	result.Promoted = true
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-candidate-promoted")
	result.Message = "candidate promoted"
	writeGuardianManifestV0(config, result)
	return result
}

func failGuardianCandidateV0(
	ctx context.Context,
	config guardianConfigV0,
	result guardianResultV0,
	phase string,
	message string,
) guardianResultV0 {
	result.Status = guardianStatusCandidateFailedV0
	result.Phase = phase
	result.Message = message
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-candidate-failed")
	packetPath, err := writeGuardianRepairPacketV0(config, result)
	if err == nil {
		result.RepairPacketPath = packetPath
	}
	if packetPath != "" && (strings.TrimSpace(config.RepairCommand) != "" || config.RepairCodex) {
		repair := runGuardianRepairCommandV0(ctx, config, phase, packetPath)
		result.Commands = append(result.Commands, repair)
		result.RepairStarted = true
	}
	writeGuardianManifestV0(config, result)
	return result
}

func baseGuardianResultV0(config guardianConfigV0) guardianResultV0 {
	return guardianResultV0{
		SchemaVersion: guardianResultSchemaVersionV0,
		ProjectDir:    config.ProjectDir,
		CurrentBin:    config.CurrentBin,
		CandidateBin:  config.CandidateBin,
		LastGoodBin:   config.LastGoodBin,
		ManifestPath:  config.ManifestPath,
		EvidenceRefs: []string{
			"evidence-ref-guardian-breakglass",
		},
	}
}

func ensureGuardianDirsV0(config guardianConfigV0) error {
	for _, dir := range []string{
		config.StateDir,
		filepath.Dir(config.CandidateBin),
		filepath.Dir(config.LastGoodBin),
		filepath.Dir(config.ManifestPath),
		filepath.Dir(config.RepairPacketPath),
		filepath.Join(config.StateDir, "logs"),
		config.RepairCodexRuntimeDir,
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

func runGuardianShellCommandV0(
	ctx context.Context,
	config guardianConfigV0,
	phase string,
	command string,
) guardianCommandResultV0 {
	start := time.Now()
	outputPath := filepath.Join(config.StateDir, "logs", sanitizeFilenamePartV0(phase)+"-"+start.UTC().Format("20060102T150405.000000000Z")+".log")
	commandCtx, cancel := context.WithTimeout(ctx, config.CommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, "sh", "-c", command)
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(commandCtx, "cmd", "/C", command)
	}
	cmd.Dir = config.ProjectDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	_ = os.WriteFile(outputPath, out.Bytes(), 0o600)
	return guardianCommandResultV0{
		Phase:      phase,
		Command:    command,
		ExitCode:   commandExitCodeV0(err),
		DurationMS: time.Since(start).Milliseconds(),
		OutputPath: outputPath,
		Error:      commandErrorStringV0(err, commandCtx.Err()),
	}
}

func runGuardianCandidateHealthcheckV0(ctx context.Context, config guardianConfigV0) guardianCommandResultV0 {
	start := time.Now()
	outputPath := filepath.Join(config.StateDir, "logs", "healthcheck-"+start.UTC().Format("20060102T150405.000000000Z")+".log")
	addr := strings.TrimSpace(config.CandidateAddr)
	if addr == "" {
		var err error
		addr, err = freeLocalAddrV0()
		if err != nil {
			_ = os.WriteFile(outputPath, []byte(err.Error()), 0o600)
			return guardianCommandResultV0{Phase: "healthcheck", Command: "candidate healthcheck", ExitCode: 1, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath, Error: err.Error()}
		}
	}
	candidateState := filepath.Join(config.StateDir, "candidate-state")
	candidateRuntime := filepath.Join(config.StateDir, "candidate-runtime")
	_ = os.MkdirAll(candidateState, 0o700)
	_ = os.MkdirAll(candidateRuntime, 0o700)
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, config.CandidateBin, "run")
	cmd.Dir = config.ProjectDir
	cmd.Env = append(os.Environ(),
		"ORQUESTA_SERVER_ADDR="+addr,
		"ORQUESTA_SERVER_STATE_DIR="+candidateState,
		"ORQUESTA_SERVER_RUNTIME_WORKDIR="+candidateRuntime,
		"ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0",
		"ORQUESTA_OPES_BRIDGE_DRY_RUN=1",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		_ = os.WriteFile(outputPath, []byte(err.Error()), 0o600)
		return guardianCommandResultV0{Phase: "healthcheck", Command: config.CandidateBin + " run", ExitCode: 1, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath, Error: err.Error()}
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	err := waitHealthURLV0(ctx, "http://"+addr+"/healthz", config.HealthTimeout)
	stopGuardianProcessV0(cmd, waitDone)
	_ = os.WriteFile(outputPath, out.Bytes(), 0o600)
	if err != nil {
		return guardianCommandResultV0{Phase: "healthcheck", Command: config.CandidateBin + " run", ExitCode: 1, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath, Error: err.Error()}
	}
	return guardianCommandResultV0{Phase: "healthcheck", Command: config.CandidateBin + " run", ExitCode: 0, DurationMS: time.Since(start).Milliseconds(), OutputPath: outputPath}
}

func waitHealthURLV0(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := http.Client{Timeout: 500 * time.Millisecond}
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		response, err := client.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("healthcheck_timeout")
}

func stopGuardianProcessV0(cmd *exec.Cmd, waitDone <-chan error) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Signal(os.Interrupt)
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		<-waitDone
	}
}

func promoteGuardianCandidateV0(config guardianConfigV0) error {
	if _, err := os.Stat(config.CandidateBin); err != nil {
		return fmt.Errorf("candidate_bin_unavailable: %w", err)
	}
	if _, err := os.Stat(config.CurrentBin); err == nil {
		if err := copyFileAtomicV0(config.CurrentBin, config.LastGoodBin, 0o700); err != nil {
			return err
		}
	}
	return copyFileAtomicV0(config.CandidateBin, config.CurrentBin, 0o700)
}

func restoreLastGoodCommandV0(config guardianConfigV0) guardianResultV0 {
	result := baseGuardianResultV0(config)
	if err := ensureGuardianDirsV0(config); err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Phase = "restore"
		result.Message = err.Error()
		writeGuardianManifestV0(config, result)
		return result
	}
	if err := copyFileAtomicV0(config.LastGoodBin, config.CurrentBin, 0o700); err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Phase = "restore"
		result.Message = err.Error()
		writeGuardianManifestV0(config, result)
		return result
	}
	result.Status = guardianStatusRestoredV0
	result.Phase = "restore"
	result.Restored = true
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-last-good-restored")
	result.Message = "last_good restored"
	writeGuardianManifestV0(config, result)
	return result
}

func runGuardianShutdownServerV0(ctx context.Context, config guardianConfigV0) guardianResultV0 {
	result := baseGuardianResultV0(config)
	result.Phase = "shutdown"
	if strings.TrimSpace(config.ServerAddr) == "" {
		result.Status = guardianStatusCandidateFailedV0
		result.Message = "server_addr_required"
		writeGuardianManifestV0(config, result)
		return result
	}
	if err := ensureGuardianDirsV0(config); err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Message = err.Error()
		writeGuardianManifestV0(config, result)
		return result
	}
	shutdown, err := waitGuardianServerShutdownReadyV0(ctx, config)
	result.Shutdown = &shutdown
	if err != nil {
		result.Status = guardianStatusCandidateFailedV0
		result.Message = err.Error()
		writeGuardianManifestV0(config, result)
		return result
	}
	if config.ServerPID > 0 {
		if err := signalGuardianProcessV0(config.ServerPID); err != nil {
			result.Status = guardianStatusCandidateFailedV0
			result.Message = err.Error()
			writeGuardianManifestV0(config, result)
			return result
		}
	}
	result.Status = guardianStatusShutdownReadyV0
	result.EvidenceRefs = append(result.EvidenceRefs, "evidence-ref-guardian-shutdown-ready")
	if config.ShutdownNow {
		result.Message = "server forced shutdown requested"
	} else if result.Shutdown != nil && !result.Shutdown.ShutdownReady {
		result.Message = "server forced shutdown requested after cooperative timeout"
	} else {
		result.Message = "server shutdown ready"
	}
	writeGuardianManifestV0(config, result)
	return result
}

func waitGuardianServerShutdownReadyV0(
	ctx context.Context,
	config guardianConfigV0,
) (guardianShutdownResultV0, error) {
	if config.ShutdownNow {
		return requestGuardianServerShutdownV0(config)
	}
	timeout := config.ShutdownTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	var last guardianShutdownResultV0
	var lastErr error
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		shutdown, err := requestGuardianServerShutdownV0(config)
		if err == nil {
			last = shutdown
			if shutdown.ShutdownReady {
				return shutdown, nil
			}
		} else {
			lastErr = err
		}
		time.Sleep(time.Second)
	}
	if lastErr != nil {
		return last, lastErr
	}
	if config.ForceAfterTimeout {
		forced := config
		forced.ShutdownForced = true
		forced.ShutdownNow = true
		shutdown, err := requestGuardianServerShutdownV0(forced)
		if err != nil {
			return last, err
		}
		if shutdown.ShutdownReady || config.ServerPID > 0 {
			return shutdown, nil
		}
		return shutdown, fmt.Errorf(
			"shutdown_forced_but_not_ready status=%s runs=%d/%d agents_in_flight=%d",
			shutdown.Status,
			shutdown.RunsStopped,
			shutdown.RunsRequested,
			shutdown.AgentsInFlight,
		)
	}
	return last, fmt.Errorf(
		"shutdown_not_ready status=%s runs=%d/%d agents_in_flight=%d checkpoints=%d checkpoint_agents=%d",
		last.Status,
		last.RunsStopped,
		last.RunsRequested,
		last.AgentsInFlight,
		last.CheckpointsPending,
		last.CheckpointAgentsPending,
	)
}

func requestGuardianServerShutdownV0(config guardianConfigV0) (guardianShutdownResultV0, error) {
	now := time.Now().UTC()
	payload := map[string]any{
		"forced":            config.ShutdownForced,
		"requested_by":      "orquesta-guardian",
		"reason":            guardianShutdownReasonV0(config),
		"idempotency_key":   "idem-orquesta-guardian-shutdown",
		"queue_limit":       config.ShutdownQueueLimit,
		"max_ticks":         8,
		"max_runs_per_tick": 4,
		"occurred_at":       now.Format(time.RFC3339),
		"evidence_refs":     []string{"evidence-ref-guardian-controlled-shutdown"},
	}
	if !config.ShutdownForced {
		payload["checkpoint_deadline_at"] = config.OccurredAt.Add(config.ShutdownTimeout).Format(time.RFC3339)
	}
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(payload); err != nil {
		return guardianShutdownResultV0{}, err
	}
	client := http.Client{Timeout: 30 * time.Second}
	response, err := client.Post(
		"http://"+strings.TrimSpace(config.ServerAddr)+"/api/v0/server/shutdown",
		"application/json",
		body,
	)
	if err != nil {
		return guardianShutdownResultV0{}, err
	}
	defer response.Body.Close()
	var result guardianShutdownResultV0
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return guardianShutdownResultV0{}, err
	}
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("shutdown_http_%d_%s", response.StatusCode, result.Status)
	}
	return result, nil
}

func guardianShutdownReasonV0(config guardianConfigV0) string {
	if config.ShutdownNow {
		return "guardian forced shutdown now"
	}
	return "guardian controlled shutdown"
}

func signalGuardianProcessV0(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(os.Interrupt)
}

func writeGuardianRepairPacketV0(config guardianConfigV0, result guardianResultV0) (string, error) {
	packet := guardianRepairPacketV0{
		SchemaVersion: guardianRepairPacketSchemaVersionV0,
		FailurePhase:  result.Phase,
		Summary:       result.Message,
		ProjectDir:    config.ProjectDir,
		CurrentBin:    config.CurrentBin,
		CandidateBin:  config.CandidateBin,
		LastGoodBin:   config.LastGoodBin,
		Commands:      append([]guardianCommandResultV0(nil), result.Commands...),
		EvidenceRefs:  append([]string(nil), result.EvidenceRefs...),
		CreatedAt:     config.OccurredAt.Format(time.RFC3339),
	}
	if err := writeJSONFileV0(config.RepairPacketPath, packet); err != nil {
		return "", err
	}
	return config.RepairPacketPath, nil
}

func runGuardianRepairCommandV0(
	ctx context.Context,
	config guardianConfigV0,
	phase string,
	packetPath string,
) guardianCommandResultV0 {
	command := expandGuardianCommandV0(config.RepairCommand, config)
	if command == "" && config.RepairCodex {
		var err error
		command, err = guardianCodexRepairCommandV0(config, packetPath)
		if err != nil {
			return guardianCommandResultV0{
				Phase:    "repair",
				Command:  "guardian codex repair command",
				ExitCode: 1,
				Error:    err.Error(),
			}
		}
	}
	commandCtx, cancel := context.WithTimeout(ctx, config.CommandTimeout)
	defer cancel()
	start := time.Now()
	outputPath := filepath.Join(config.StateDir, "logs", "repair-"+start.UTC().Format("20060102T150405.000000000Z")+".log")
	cmd := exec.CommandContext(commandCtx, "sh", "-c", command)
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(commandCtx, "cmd", "/C", command)
	}
	cmd.Dir = config.ProjectDir
	cmd.Env = append(os.Environ(),
		"ORQUESTA_GUARDIAN_REPAIR_PACKET="+packetPath,
		"ORQUESTA_GUARDIAN_FAILURE_PHASE="+phase,
		"ORQUESTA_GUARDIAN_PROJECT_DIR="+config.ProjectDir,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	_ = os.WriteFile(outputPath, out.Bytes(), 0o600)
	return guardianCommandResultV0{
		Phase:      "repair",
		Command:    command,
		ExitCode:   commandExitCodeV0(err),
		DurationMS: time.Since(start).Milliseconds(),
		OutputPath: outputPath,
		Error:      commandErrorStringV0(err, commandCtx.Err()),
	}
}

func guardianCodexRepairCommandV0(config guardianConfigV0, packetPath string) (string, error) {
	promptPath, err := writeGuardianCodexRepairPromptV0(config, packetPath)
	if err != nil {
		return "", err
	}
	waveRef := "guardian-repair-" + config.OccurredAt.Format("20060102T150405Z")
	return strings.Join([]string{
		shellQuoteV0(config.CurrentBin),
		"codex-launch-wave",
		"--agents 1",
		"--wave-ref " + shellQuoteV0(waveRef),
		"--project-dir " + shellQuoteV0(config.ProjectDir),
		"--runtime-dir " + shellQuoteV0(filepath.Join(config.RepairCodexRuntimeDir, waveRef)),
		"--sandbox " + shellQuoteV0(config.RepairCodexSandbox),
		"--approval-policy never",
		"--reasoning-effort " + shellQuoteV0(config.RepairCodexEffort),
		"--prompt-file " + shellQuoteV0(promptPath),
	}, " "), nil
}

func writeGuardianCodexRepairPromptV0(config guardianConfigV0, packetPath string) (string, error) {
	promptPath := strings.TrimSuffix(packetPath, filepath.Ext(packetPath)) + ".md"
	content := strings.Join([]string{
		"# Reparacion break-glass de Orquesta",
		"",
		"Trabaja en el repo indicado y repara solo el fallo descrito en el paquete.",
		"No borres trabajo de otros agentes. No hagas reset destructivo. Mantener arquitectura hexagonal.",
		"Si hay que ampliar permisos o rails para que funcione, documenta deuda futura y conserva evidencia.",
		"",
		"Paquete JSON de reparacion:",
		packetPath,
		"",
		"Objetivo:",
		"- Arreglar build/tests/healthcheck del candidato de Orquesta.",
		"- Ejecutar las pruebas relevantes indicadas en el paquete.",
		"- Dejar ACK/handoff compacto con cambios, pruebas y bloqueos.",
		"",
	}, "\n")
	if err := os.WriteFile(promptPath, []byte(content), 0o600); err != nil {
		return "", err
	}
	return promptPath, nil
}

func writeGuardianManifestV0(config guardianConfigV0, result guardianResultV0) {
	if result.ManifestPath == "" {
		result.ManifestPath = config.ManifestPath
	}
	_ = writeJSONFileV0(config.ManifestPath, result)
}

func writeJSONFileV0(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func copyFileAtomicV0(src string, dst string, mode os.FileMode) error {
	body, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	tmp := dst + ".tmp-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.WriteFile(tmp, body, mode); err != nil {
		return err
	}
	if err := os.Chmod(tmp, mode); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func commandExitCodeV0(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func commandErrorStringV0(err error, ctxErr error) string {
	if ctxErr != nil {
		return ctxErr.Error()
	}
	if err != nil {
		return err.Error()
	}
	return ""
}

func freeLocalAddrV0() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	return listener.Addr().String(), nil
}

func expandGuardianCommandV0(command string, config guardianConfigV0) string {
	replacer := strings.NewReplacer(
		"{project_dir}", shellQuoteV0(config.ProjectDir),
		"{state_dir}", shellQuoteV0(config.StateDir),
		"{current_bin}", shellQuoteV0(config.CurrentBin),
		"{candidate_bin}", shellQuoteV0(config.CandidateBin),
		"{last_good_bin}", shellQuoteV0(config.LastGoodBin),
		"{repair_packet}", shellQuoteV0(config.RepairPacketPath),
	)
	return replacer.Replace(command)
}

func envOrDefaultV0(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBoolOrDefaultV0(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envDurationOrDefaultV0(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envIntOrDefaultV0(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitEnvCommandsV0(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return compactStringsV0(strings.Split(value, "\n"))
}

func absPathFromBaseV0(base string, path string) string {
	path = strings.TrimSpace(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(base, path)
}

func sanitizeFilenamePartV0(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
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
		return "command"
	}
	return out
}

func shellQuoteV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func compactStringsV0(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
