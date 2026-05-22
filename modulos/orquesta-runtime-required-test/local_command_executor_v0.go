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
	"sync"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const outputSchemaVersionV0 = "orquesta.runtime_required_test.output.v0"

type LocalCommandExecutorV0 struct {
	ProjectWorkDir  string
	OutputDir       string
	AllowedCommands map[string]string
	Env             []string
	MaxOutputBytes  int64
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
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, err
	}
	commandPath, ok := normalized.AllowedCommands[tokens[0]]
	if !ok {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, fmt.Errorf("required_test_command_not_allowed: %s", tokens[0])
	}
	if commandIsShellV0(tokens[0]) || commandIsShellV0(commandPath) {
		return orquestacionnucleoapp.RequiredTestCommandExecutionResultV0{}, fmt.Errorf("required_test_command_shell_prohibited")
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
	ref, err := writeOutputArtifactV0(normalized, request, status, output)
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

func isolatedLocalCommandEnvV0(env []string) []string {
	if len(env) == 0 {
		return []string{}
	}
	return append([]string(nil), env...)
}

func writeOutputArtifactV0(
	executor LocalCommandExecutorV0,
	request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0,
	status orquestacionnucleoapp.RequiredTestEvidenceStatusV0,
	output outputBufferV0,
) (string, error) {
	ref := outputArtifactRefV0(request)
	path := filepath.Join(executor.OutputDir, strings.TrimPrefix(ref, "required-test-output-v0/"))
	content := strings.Join([]string{
		"schema_version=" + outputSchemaVersionV0,
		"test_command=" + strings.TrimSpace(request.TestCommand),
		"status=" + string(status),
		fmt.Sprintf("truncated=%t", output.Truncated()),
		"",
		output.String(),
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", err
	}
	return ref, nil
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

func splitCommandV0(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("required_test_command_required")
	}
	tokens := make([]string, 0)
	var current strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if current.Len() == 0 {
			return
		}
		tokens = append(tokens, current.String())
		current.Reset()
	}
	for _, r := range command {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t':
			flush()
		case ';', '&', '|', '<', '>', '`':
			return nil, fmt.Errorf("required_test_command_shell_syntax_prohibited")
		default:
			current.WriteRune(r)
		}
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("required_test_command_unclosed_token")
	}
	flush()
	if len(tokens) == 0 {
		return nil, fmt.Errorf("required_test_command_required")
	}
	return tokens, nil
}

func commandIsShellV0(command string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(command)))
	switch base {
	case "sh", "bash", "dash", "zsh", "fish", "cmd", "cmd.exe", "powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return true
	default:
		return false
	}
}

func envEntryAllowedV0(item string) bool {
	key, value, ok := strings.Cut(item, "=")
	if !ok || strings.TrimSpace(key) == "" || pathHasCredentialMarkerV0(key) || pathHasCredentialMarkerV0(value) {
		return false
	}
	if strings.ContainsAny(value, `/\`) {
		return envPathValueAllowedV0(key, value)
	}
	return true
}

func envPathValueAllowedV0(key string, value string) bool {
	switch strings.ToUpper(strings.TrimSpace(key)) {
	case "PATH":
		return envPathListAllowedV0(value)
	case "GOCACHE", "GOMODCACHE", "GOPATH", "GOTMPDIR", "TMPDIR":
		return filepath.IsAbs(value) && !pathHasCredentialMarkerV0(value)
	default:
		return false
	}
}

func envPathListAllowedV0(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, item := range filepath.SplitList(value) {
		if item == "" || !filepath.IsAbs(item) || pathHasCredentialMarkerV0(item) {
			return false
		}
	}
	return true
}

func pathHasCredentialMarkerV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	for _, marker := range []string{"$home", "${home}", "%userprofile%", "oauth", "token", "secret", "api_key", "credential", "password"} {
		if strings.Contains(low, marker) {
			return true
		}
	}
	return false
}

type outputBufferV0 struct {
	mu        sync.Mutex
	limit     int64
	used      int64
	truncated bool
	builder   strings.Builder
}

func newOutputBufferV0(limit int64) *outputBufferV0 {
	return &outputBufferV0{limit: limit}
}

func (buffer *outputBufferV0) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	if buffer.limit <= 0 || buffer.used >= buffer.limit {
		buffer.truncated = true
		return len(data), nil
	}
	remaining := buffer.limit - buffer.used
	writeLen := int64(len(data))
	if writeLen > remaining {
		writeLen = remaining
		buffer.truncated = true
	}
	buffer.builder.Write(data[:int(writeLen)])
	buffer.used += writeLen
	return len(data), nil
}

func (buffer *outputBufferV0) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.builder.String()
}

func (buffer *outputBufferV0) Truncated() bool {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.truncated
}
