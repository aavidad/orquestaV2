package orquestaruntimerequiredtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

// PreflightGoalRequiredTestAttestationV0 verifies the configured external
// toolchain before the server makes an implementer available.
func (adapter *LocalGoalRequiredTestAttestationAdapterV0) PreflightGoalRequiredTestAttestationV0(ctx context.Context) error {
	if adapter == nil {
		return fmt.Errorf("goal_required_test_attestation_adapter_unavailable")
	}
	result, err := adapter.runPreflightV0(ctx, "startup", "goal-required-test-attestation-startup")
	if err != nil {
		return err
	}
	if result.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 {
		return fmt.Errorf("goal_required_test_attestation_preflight_failed")
	}
	return nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) runPreflightV0(
	ctx context.Context,
	correlationID string,
	scope string,
) (orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(adapter.config.PreflightCommands) == 0 {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, fmt.Errorf("goal_required_test_preflight_required")
	}
	snapshotEvidenceRef, snapshotValid, err := adapter.observeDependencySnapshotV0()
	if err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	if !snapshotValid {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
			Status: orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0, EvidenceRefs: []string{snapshotEvidenceRef},
		}, nil
	}
	runDir := filepath.Join(adapter.config.RuntimeRoot, "preflight", localGoalAttestationHashV0(correlationID))
	outputDir := filepath.Join(adapter.config.RuntimeRoot, "evidence", "preflight")
	for _, dir := range []string{runDir, outputDir, filepath.Join(runDir, "tmp"), filepath.Join(runDir, "go-cache"), filepath.Join(runDir, "go-path")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
		}
	}
	result := orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{Status: orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0}
	if snapshotEvidenceRef != "" {
		result.EvidenceRefs = append(result.EvidenceRefs, snapshotEvidenceRef)
	}
	for index, command := range adapter.config.PreflightCommands {
		request := orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0{
			RunRef: "goal-required-test-preflight", TaskRef: scope,
			TestCommand: command, CorrelationID: fmt.Sprintf("%s-%d", correlationID, index),
		}
		execution, err := adapter.runHermeticCommandV0(ctx, runDir, outputDir, request)
		if err != nil {
			return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
		}
		result.EvidenceRefs = append(result.EvidenceRefs, execution.EvidenceRefs...)
		if execution.Status != orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 {
			result.Status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
			return result, nil
		}
	}
	return result, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) runHermeticCommandV0(
	ctx context.Context,
	runDir string,
	outputDir string,
	request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, error) {
	executor := LocalCommandExecutorV0{
		ProjectWorkDir: adapter.config.ProjectWorkDir, OutputDir: outputDir,
		AllowedCommands: adapter.config.AllowedCommands, Env: adapter.hermeticEnvironmentV0(runDir),
		MaxOutputBytes: adapter.config.MaxOutputBytes, MaxArtifacts: adapter.config.MaxArtifacts,
	}
	tokens, err := splitCommandV0(request.TestCommand)
	if err != nil {
		return failedLocalCommandValidationResultV0(executor, request, err.Error())
	}
	commandPath, ok := executor.AllowedCommands[tokens[0]]
	if !ok || commandIsShellV0(tokens[0]) || commandIsShellV0(commandPath) {
		return failedLocalCommandValidationResultV0(executor, request, "required_test_preflight_command_not_allowed")
	}
	executionContext, cancel := context.WithTimeout(ctx, adapter.config.MaxRuntime)
	defer cancel()
	output, runErr := runLocalCommandV0(executionContext, executor, commandPath, tokens[1:])
	status := orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0
	if runErr != nil {
		status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
	}
	ref, status, err := writeOutputArtifactV0(executor, request, status, output)
	if err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{Status: status, EvidenceRefs: []string{ref}}, nil
}

func (adapter *LocalGoalRequiredTestAttestationAdapterV0) hermeticEnvironmentV0(runDir string) []string {
	modCache := adapter.config.DependencySnapshotPath
	if modCache == "" {
		modCache = filepath.Join(runDir, "go-mod-cache")
	}
	pathDirs := map[string]struct{}{}
	for _, commandPath := range adapter.config.AllowedCommands {
		pathDirs[filepath.Dir(commandPath)] = struct{}{}
	}
	pathDirs[filepath.Dir(adapter.config.GitCommandPath)] = struct{}{}
	orderedPathDirs := make([]string, 0, len(pathDirs))
	for dir := range pathDirs {
		orderedPathDirs = append(orderedPathDirs, dir)
	}
	sort.Strings(orderedPathDirs)
	return []string{
		"CGO_ENABLED=0", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GONOSUMDB=*", "GOTOOLCHAIN=local",
		"GOCACHE=" + filepath.Join(runDir, "go-cache"), "GOMODCACHE=" + modCache,
		"GOPATH=" + filepath.Join(runDir, "go-path"), "GOTMPDIR=" + filepath.Join(runDir, "tmp"),
		"TMPDIR=" + filepath.Join(runDir, "tmp"), "PATH=" + strings.Join(orderedPathDirs, string(os.PathListSeparator)),
	}
}
