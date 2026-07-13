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
	registry        *allowedCommandIdentityRegistryV0
}

// NewLocalCommandExecutorV0 accepts configured symlinks only at this public
// factory boundary. It resolves each one once to a canonical regular target;
// the registry then freezes that target and never follows the symlink again.
// The lower-level identity registry deliberately rejects symlinks.
func NewLocalCommandExecutorV0(executor LocalCommandExecutorV0) (LocalCommandExecutorV0, error) {
	normalized := normalizeLocalCommandExecutorV0(executor)
	allowed, err := canonicalAllowedCommandsV0(normalized.AllowedCommands)
	if err != nil {
		return LocalCommandExecutorV0{}, err
	}
	normalized.AllowedCommands = allowed
	registry, err := newAllowedCommandIdentityRegistryV0(normalized.AllowedCommands)
	if err != nil {
		return LocalCommandExecutorV0{}, err
	}
	normalized.AllowedCommands = nil
	normalized.registry = registry
	if err := validateLocalCommandExecutorV0(normalized); err != nil {
		_ = registry.Close()
		return LocalCommandExecutorV0{}, err
	}
	return normalized, nil
}

func canonicalAllowedCommandsV0(allowed map[string]string) (map[string]string, error) {
	canonical := make(map[string]string, len(allowed))
	for alias, path := range allowed {
		alias = strings.TrimSpace(alias)
		path = strings.TrimSpace(path)
		if _, duplicate := canonical[alias]; duplicate {
			return nil, fmt.Errorf("required_test_command_alias_duplicate: %s", alias)
		}
		if filepath.IsAbs(path) {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil, fmt.Errorf("required_test_command_identity_invalid: %s", alias)
			}
			path = resolved
		}
		canonical[alias] = path
	}
	return canonical, nil
}

func localCommandExecutorWithRegistryV0(executor LocalCommandExecutorV0, registry *allowedCommandIdentityRegistryV0) LocalCommandExecutorV0 {
	executor = normalizeLocalCommandExecutorV0(executor)
	executor.AllowedCommands = nil
	executor.registry = registry
	return executor
}

func (executor *LocalCommandExecutorV0) Close() error {
	if executor == nil || executor.registry == nil {
		return nil
	}
	return executor.registry.Close()
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
	normalized := executor
	if err := validateLocalCommandExecutorV0(normalized); err != nil {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	resolved, err := normalized.registry.resolve(request.TestCommand)
	if err != nil {
		message := err.Error()
		if resolutionErr := commandAllowlistResolutionErrorV0Of(err); resolutionErr != nil {
			switch resolutionErr.Failure {
			case commandAllowlistNotAllowedFailureV0:
				message = "required_test_command_not_allowed: " + resolutionErr.Command
			case commandAllowlistShellFailureV0:
				message = "required_test_command_shell_prohibited"
			}
		}
		return failedLocalCommandValidationResultV0(normalized, request, message)
	}

	output, runErr := runLocalCommandV0(ctx, normalized, resolved, resolved.Tokens[1:])
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
		localCommandGoTestWithoutExecutedTestsV0(resolved.Tokens, output.String()) {
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
		output,
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
	return LocalCommandExecutorV0{
		ProjectWorkDir:  strings.TrimSpace(executor.ProjectWorkDir),
		OutputDir:       strings.TrimSpace(executor.OutputDir),
		AllowedCommands: executor.AllowedCommands,
		Env:             append([]string(nil), executor.Env...),
		MaxOutputBytes:  executor.MaxOutputBytes,
		MaxArtifacts:    executor.MaxArtifacts,
		registry:        executor.registry,
	}
}

func validateLocalCommandExecutorV0(executor LocalCommandExecutorV0) error {
	if err := validateRuntimeDirV0("project_work_dir", executor.ProjectWorkDir, false); err != nil {
		return err
	}
	if err := validateRuntimeDirV0("output_dir", executor.OutputDir, true); err != nil {
		return err
	}
	if executor.registry == nil {
		return fmt.Errorf("required_test_command_identity_invalid")
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
	resolved commandAllowlistResolutionV0,
	args []string,
) (*outputBufferV0, error) {
	limit := executor.MaxOutputBytes
	if limit <= 0 {
		limit = 1024 * 1024
	}
	output := newOutputBufferV0(limit)
	if resolved.executionFile == nil || resolved.identity == nil {
		return output, fmt.Errorf("required_test_command_identity_invalid")
	}
	defer resolved.executionFile.Close()
	cmd := execCommandContextFromResolutionV0(ctx, resolved, args...)
	cmd.Dir = executor.ProjectWorkDir
	cmd.Env = isolatedLocalCommandEnvForResolutionV0(executor.Env, resolved)
	cmd.Stdout = output
	cmd.Stderr = output
	err := cmd.Run()
	return output, err
}

func execCommandContextFromResolutionV0(ctx context.Context, resolved commandAllowlistResolutionV0, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "/proc/self/fd/3", args...)
	cmd.Args = append([]string{resolved.identity.alias}, args...)
	cmd.ExtraFiles = []*os.File{resolved.executionFile}
	return cmd
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

func localCommandOutputWithDiagnosticV0(output *outputBufferV0, message string) *outputBufferV0 {
	limit := output.limit
	if limit <= 0 {
		limit = 1024 * 1024
	}
	next := newOutputBufferV0(limit)
	_, _ = next.Write([]byte(output.String()))
	_, _ = next.Write([]byte("\n" + strings.TrimSpace(message) + "\n"))
	return next
}

func isolatedLocalCommandEnvV0(env []string) []string {
	if len(env) == 0 {
		return []string{}
	}
	return append([]string(nil), env...)
}

func isolatedLocalCommandEnvForResolutionV0(env []string, resolved commandAllowlistResolutionV0) []string {
	result := isolatedLocalCommandEnvV0(env)
	if resolved.identity == nil || filepath.Base(strings.TrimSpace(resolved.identity.path)) != "go" {
		return result
	}
	for _, item := range result {
		if key, _, ok := strings.Cut(item, "="); ok && strings.EqualFold(strings.TrimSpace(key), "GOROOT") {
			return result
		}
	}
	// A sealed memfd deliberately hides the on-disk executable pathname from
	// the child. Trimmed Go distributions need that pathname to recover GOROOT,
	// so derive it from the already admitted canonical Go binary, never from
	// ambient environment.
	goRoot := filepath.Dir(filepath.Dir(resolved.identity.path))
	if info, err := os.Stat(goRoot); err == nil && info.IsDir() {
		result = append(result, "GOROOT="+goRoot)
	}
	return result
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
