package orquestaruntimerequiredtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

// goalRequiredTestCommandRequiresRaceCGOV0 recognizes only the Go race flag in
// the command portion. Arguments passed to a test binary must not change the
// attestor's toolchain policy.
func goalRequiredTestCommandRequiresRaceCGOV0(command string, commandPath string) bool {
	tokens, err := splitCommandV0(command)
	if err != nil || len(tokens) < 3 || filepath.Base(strings.TrimSpace(commandPath)) != "go" || tokens[1] != "test" {
		return false
	}
	for _, token := range tokens[2:] {
		if token == "-args" || token == "--" {
			return false
		}
		if token == "-race" {
			return true
		}
	}
	return false
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) requiredTestUsesRaceCGOV0(command string) (bool, error) {
	_, _, race, err := adapter.requiredTestRaceCGOCommandV0(command)
	return race, err
}

// requiredTestRaceCGOCommandV0 resolves the command exactly as the frozen
// required test will run. In particular, a Go alias is part of the attested
// command identity: a race probe must not silently substitute another
// allowlisted Go binary.
func (adapter *LocalGoalRequiredTestAttestationAdapterV0) requiredTestRaceCGOCommandV0(command string) (string, string, bool, error) {
	tokens, err := splitCommandV0(command)
	if err != nil {
		return "", "", false, err
	}
	if len(tokens) == 0 {
		return "", "", false, fmt.Errorf("goal_required_test_command_invalid_before_launch")
	}
	commandPath, ok := adapter.config.AllowedCommands[tokens[0]]
	if !ok {
		return "", "", false, fmt.Errorf("goal_required_test_command_not_allowed_before_launch: %s", tokens[0])
	}
	return tokens[0], commandPath, goalRequiredTestCommandRequiresRaceCGOV0(command, commandPath), nil
}

// runRaceCGOProbeV0 checks the only CGO-enabled execution path in a freshly
// created dependency-free module. It never executes inside the checked-out
// project and its output receipt survives a failed probe.
func (adapter *LocalGoalRequiredTestAttestationAdapterV0) runRaceCGOProbeV0(
	ctx context.Context,
	correlationID string,
	goCommand string,
	goPath string,
) (result orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, resultErr error) {
	if strings.TrimSpace(goCommand) == "" || filepath.Base(strings.TrimSpace(goPath)) != "go" || adapter.config.AllowedCommands[goCommand] != goPath {
		return result, fmt.Errorf("goal_required_test_race_cgo_go_unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	root := filepath.Join(adapter.config.RuntimeRoot, "race-cgo-probes")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return result, err
	}
	runDir, err := os.MkdirTemp(root, localGoalAttestationHashV0(correlationID)+"-")
	if err != nil {
		return result, err
	}
	defer func() {
		if cleanupErr := removeGoalRequiredTestExecutionDirV0(runDir); resultErr == nil && cleanupErr != nil {
			result = orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}
			resultErr = fmt.Errorf("goal_required_test_execution_cleanup_failed: %w", cleanupErr)
		}
	}()
	for name, content := range map[string]string{
		"go.mod":                 "module orquesta.invalid/racecgo\n\ngo 1.22\n",
		"race_cgo_probe_test.go": "package racecgo\n\nimport \"testing\"\n\nfunc TestRaceCGOProbe(t *testing.T) {}\n",
	} {
		if err := os.WriteFile(filepath.Join(runDir, name), []byte(content), 0o600); err != nil {
			return result, err
		}
	}
	for _, dir := range []string{filepath.Join(runDir, "tmp"), filepath.Join(runDir, "go-cache"), filepath.Join(runDir, "go-path"), filepath.Join(runDir, "go-mod-cache")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return result, err
		}
	}
	outputDir := filepath.Join(adapter.config.RuntimeRoot, "evidence", "race-cgo-probes")
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		return result, err
	}
	executor := LocalCommandExecutorV0{
		ProjectWorkDir: runDir, OutputDir: outputDir, AllowedCommands: adapter.config.AllowedCommands,
		Env:            adapter.hermeticEnvironmentForRaceCGOV0(runDir, filepath.Join(runDir, "go-mod-cache")),
		MaxOutputBytes: adapter.config.MaxOutputBytes, MaxArtifacts: adapter.config.MaxArtifacts,
	}
	executionContext, cancel := context.WithTimeout(ctx, adapter.config.MaxRuntime)
	defer cancel()
	output, runErr := runLocalCommandV0(executionContext, executor, goPath, []string{"test", "-race", "-count=1", "."})
	status := orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0
	if runErr != nil {
		status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
	}
	ref, status, err := writeOutputArtifactV0(executor, orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0{
		RunRef: "goal-required-test-race-cgo-probe", TaskRef: "race-cgo-probe", TestCommand: goCommand + " test -race -count=1 .", CorrelationID: correlationID,
	}, status, output)
	if err != nil {
		return result, err
	}
	return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{Status: status, EvidenceRefs: []string{ref}}, nil
}
