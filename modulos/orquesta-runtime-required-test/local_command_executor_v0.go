package orquestaruntimerequiredtest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const outputSchemaVersionV0 = "orquesta.runtime_required_test.output.v0"

type LocalCommandExecutorV0 struct {
	ProjectWorkDir  string
	OutputDir       string
	AllowedCommands map[string]string
	Env             []string
	MaxOutputBytes  int64
	MaxArtifacts    int
}

func (executor LocalCommandExecutorV0) RunRequiredTestCommandV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	normalized := normalizeLocalCommandExecutorV0(executor)
	if err := validateLocalCommandExecutorV0(normalized); err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	tokens, err := splitCommandV0(request.TestCommand)
	if err != nil {
		return failedLocalCommandValidationResultV0(normalized, request, err.Error())
	}
	commandPath, ok := normalized.AllowedCommands[tokens[0]]
	if !ok {
		return failedLocalCommandValidationResultV0(
			normalized,
			request,
			"required_test_command_not_allowed: "+tokens[0],
		)
	}
	if commandIsShellV0(tokens[0]) || commandIsShellV0(commandPath) {
		return failedLocalCommandValidationResultV0(
			normalized,
			request,
			"required_test_command_shell_prohibited",
		)
	}

	output, runErr := runLocalCommandV0(ctx, normalized, commandPath, tokens[1:])
	status := orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0
	if runErr != nil {
		if ctx.Err() != nil {
			return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, ctx.Err()
		}
		var exitErr *exec.ExitError
		if !errors.As(runErr, &exitErr) {
			return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, runErr
		}
		status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
	}
	if status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 &&
		localCommandGoTestWithoutExecutedTestsV0(tokens, output.String()) {
		status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
		output = localCommandOutputWithDiagnosticV0(output, "required_test_go_no_tests_executed")
	}
	if status == orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0 &&
		localCommandValidationPassedWithoutScannedFilesV0(output.String()) {
		status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
		output = localCommandOutputWithDiagnosticV0(output, "required_test_validation_empty_scan")
	}
	ref, status, err := writeOutputArtifactV0(normalized, request, status, output)
	if err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
		Status:       status,
		EvidenceRefs: []string{ref},
	}, nil
}

func failedLocalCommandValidationResultV0(
	executor LocalCommandExecutorV0,
	request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0,
	message string,
) (orquestacionnucleoapp.RequiredTestCommandExecutionResultV0, error) {
	limit := executor.MaxOutputBytes
	if limit <= 0 {
		limit = 1024 * 1024
	}
	output := newOutputBufferV0(limit)
	_, _ = output.Write([]byte(strings.TrimSpace(message) + "\n"))
	ref, status, err := writeOutputArtifactV0(
		executor,
		request,
		orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0,
		*output,
	)
	if err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{
		Status:       status,
		EvidenceRefs: []string{ref},
	}, nil
}

func normalizeLocalCommandExecutorV0(executor LocalCommandExecutorV0) LocalCommandExecutorV0 {
	allowed := map[string]string{}
	for name, path := range executor.AllowedCommands {
		allowed[strings.TrimSpace(name)] = strings.TrimSpace(path)
	}
	return LocalCommandExecutorV0{
		ProjectWorkDir:  strings.TrimSpace(executor.ProjectWorkDir),
		OutputDir:       strings.TrimSpace(executor.OutputDir),
		AllowedCommands: allowed,
		Env:             append([]string(nil), executor.Env...),
		MaxOutputBytes:  executor.MaxOutputBytes,
		MaxArtifacts:    executor.MaxArtifacts,
	}
}

func validateLocalCommandExecutorV0(executor LocalCommandExecutorV0) error {
	if err := validateRuntimeDirV0("project_work_dir", executor.ProjectWorkDir, false); err != nil {
		return err
	}
	if err := validateRuntimeDirV0("output_dir", executor.OutputDir, true); err != nil {
		return err
	}
	if len(executor.AllowedCommands) == 0 {
		return fmt.Errorf("required_test_allowed_commands_required")
	}
	for name, path := range executor.AllowedCommands {
		if name == "" || strings.ContainsAny(name, `/\`) {
			return fmt.Errorf("required_test_command_name_invalid: %s", name)
		}
		if commandIsShellV0(name) {
			return fmt.Errorf("required_test_command_shell_prohibited")
		}
		if !filepath.IsAbs(path) || pathHasCredentialMarkerV0(path) || commandIsShellV0(path) {
			return fmt.Errorf("required_test_command_path_invalid: %s", name)
		}
	}
	for i, item := range executor.Env {
		if !envEntryAllowedV0(item) {
			return fmt.Errorf("required_test_env_invalid: env[%d]", i)
		}
	}
	return nil
}

func validateRuntimeDirV0(field string, path string, create bool) error {
	if path == "" || !filepath.IsAbs(path) || pathHasCredentialMarkerV0(path) {
		return fmt.Errorf("required_test_%s_invalid", field)
	}
	if create {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return fmt.Errorf("required_test_%s_create: %w", field, err)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("required_test_%s_stat: %w", field, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("required_test_%s_not_dir", field)
	}
	return nil
}

func runLocalCommandV0(
	ctx context.Context,
	executor LocalCommandExecutorV0,
	commandPath string,
	args []string,
) (outputBufferV0, error) {
	limit := executor.MaxOutputBytes
	if limit <= 0 {
		limit = 1024 * 1024
	}
	output := newOutputBufferV0(limit)
	cmd := exec.CommandContext(ctx, commandPath, args...)
	cmd.Dir = executor.ProjectWorkDir
	cmd.Env = isolatedLocalCommandEnvV0(executor.Env)
	cmd.Stdout = output
	cmd.Stderr = output
	err := cmd.Run()
	return *output, err
}

func localCommandGoTestWithoutExecutedTestsV0(tokens []string, output string) bool {
	if len(tokens) < 2 ||
		filepath.Base(strings.TrimSpace(tokens[0])) != "go" ||
		strings.TrimSpace(tokens[1]) != "test" {
		return false
	}
	seenNoTests := false
	seenExecutedTests := false
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "[no test files]") ||
			strings.Contains(trimmed, "[no tests to run]") {
			seenNoTests = true
			continue
		}
		if strings.HasPrefix(trimmed, "ok ") || strings.HasPrefix(trimmed, "ok\t") {
			seenExecutedTests = true
		}
	}
	return seenNoTests && !seenExecutedTests
}

func localCommandValidationPassedWithoutScannedFilesV0(output string) bool {
	compact := strings.NewReplacer(" ", "", "\t", "", "\r", "").Replace(strings.ToLower(output))
	if compact == "" {
		return false
	}
	hasPassingValidation := strings.Contains(compact, `"status":"pass"`) ||
		strings.Contains(compact, `"status":"passed"`) ||
		strings.Contains(compact, "status:pass") ||
		strings.Contains(compact, "status:passed") ||
		strings.Contains(compact, "status=pass") ||
		strings.Contains(compact, "status=passed")
	hasEmptyScan := strings.Contains(compact, `"files_scanned":0`) ||
		strings.Contains(compact, "files_scanned:0") ||
		strings.Contains(compact, "files_scanned=0")
	return hasPassingValidation && hasEmptyScan
}

func localCommandOutputWithDiagnosticV0(output outputBufferV0, message string) outputBufferV0 {
	limit := output.limit
	if limit <= 0 {
		limit = 1024 * 1024
	}
	next := newOutputBufferV0(limit)
	_, _ = next.Write([]byte(output.String()))
	_, _ = next.Write([]byte("\n" + strings.TrimSpace(message) + "\n"))
	return *next
}

func isolatedLocalCommandEnvV0(env []string) []string {
	if len(env) == 0 {
		return []string{}
	}
	return append([]string(nil), env...)
}

func outputArtifactRefV0(request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(request.RunRef),
		strings.TrimSpace(request.TaskRef),
		strings.TrimSpace(request.TestCommand),
		strings.TrimSpace(request.CorrelationID),
	}, "\x00")))
	return "required-test-output-v0/" + hex.EncodeToString(hash[:])[:24] + ".log"
}
