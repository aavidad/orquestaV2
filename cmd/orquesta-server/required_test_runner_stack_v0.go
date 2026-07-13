package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaserver "orquesta/modulos/orquesta-server"
)

// serverRequiredTestRunnerV0 keeps ownership of the executor descriptors while
// exposing the existing required-test runner port to the application stack.
type serverRequiredTestRunnerV0 struct {
	orquestacionnucleoapp.RequiredTestRunnerV0
	executor orquestaruntimerequiredtest.LocalCommandExecutorV0
}

func (runner *serverRequiredTestRunnerV0) RunRequiredTestsV0(
	ctx context.Context,
	request orquestacionnucleoapp.RequiredTestExecutionRequestV0,
) (orquestacionnucleoapp.RequiredTestExecutionResultV0, error) {
	if runner == nil {
		return orquestacionnucleoapp.RequiredTestExecutionResultV0{}, fmt.Errorf("required_test_runner_unavailable")
	}
	return runner.RequiredTestRunnerV0.RunRequiredTestsV0(ctx, request)
}

func (runner *serverRequiredTestRunnerV0) Close() error {
	if runner == nil {
		return nil
	}
	return runner.executor.Close()
}

var _ orquestacionnucleoapp.RequiredTestRunnerPortV0 = (*serverRequiredTestRunnerV0)(nil)

func requiredTestRunnerFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
	evidenceWriter orquestacionnucleoapp.RequiredTestEvidenceWriterPortV0,
	projectConfigs ...serverProjectConfigFileV0,
) (orquestacionnucleoapp.RequiredTestRunnerPortV0, error) {
	projectConfig := serverProjectConfigFileV0{}
	if len(projectConfigs) > 0 {
		projectConfig = projectConfigs[0]
	} else {
		projectConfig = projectConfigFromServerConfigBestEffortV0(serverConfig)
	}
	if !requiredTestRunnerEnabledFromProjectConfigV0(projectConfig) {
		return nil, nil
	}
	if evidenceWriter == nil {
		return nil, fmt.Errorf("required_test_evidence_writer_required")
	}
	allowed, err := requiredTestAllowedCommandsFromProjectConfigV0(projectConfig)
	if err != nil {
		return nil, err
	}
	absOutputDir, err := requiredTestOutputDirFromProjectConfigV0(serverConfig, projectConfig)
	if err != nil {
		return nil, err
	}
	env, err := requiredTestEnvFromProjectConfigV0(projectConfig, allowed, absOutputDir)
	if err != nil {
		return nil, err
	}
	executor, err := orquestaruntimerequiredtest.NewLocalCommandExecutorV0(orquestaruntimerequiredtest.LocalCommandExecutorV0{
		ProjectWorkDir:  serverConfig.ProjectWorkDir,
		OutputDir:       absOutputDir,
		AllowedCommands: allowed,
		Env:             env,
		MaxOutputBytes:  int64(intProjectConfigOrEnvOrDefaultV0(envRequiredTestMaxOutputBytesV0, projectConfig.RequiredTestRunner.MaxOutputBytes, 1024*1024)),
		MaxArtifacts:    intProjectConfigOrEnvOrDefaultV0(envRequiredTestOutputMaxArtifactsV0, projectConfig.RequiredTestRunner.MaxArtifacts, 200),
	})
	if err != nil {
		return nil, err
	}
	coreRunner := orquestacionnucleoapp.RequiredTestRunnerV0{
		Executor:       executor,
		EvidenceWriter: evidenceWriter,
	}
	return &serverRequiredTestRunnerV0{
		RequiredTestRunnerV0: coreRunner,
		executor:             executor,
	}, nil
}

func requiredTestRunnerEnabledFromProjectConfigV0(projectConfig serverProjectConfigFileV0) bool {
	if value, ok := os.LookupEnv(envRequiredTestRunnerEnabledV0); ok {
		return strings.TrimSpace(value) == "1"
	}
	return projectConfig.RequiredTestRunner.Enabled != nil && *projectConfig.RequiredTestRunner.Enabled
}

func requiredTestAllowedCommandsFromProjectConfigV0(projectConfig serverProjectConfigFileV0) (map[string]string, error) {
	allowed := map[string]string{}
	runtime := projectConfig.RequiredTestRunner
	if goCommand := stringProjectConfigOrEnvOrDefaultV0(envRequiredTestGoCommandV0, runtime.GoCommand, ""); goCommand != "" {
		abs, err := filepath.Abs(goCommand)
		if err != nil {
			return nil, fmt.Errorf("required_test_go_command_invalid")
		}
		allowed["go"] = abs
	}
	for _, item := range requiredTestAllowedCommandEntriesFromProjectConfigV0(runtime) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		name, path, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("required_test_allowed_commands_invalid")
		}
		abs, err := filepath.Abs(strings.TrimSpace(path))
		if err != nil {
			return nil, fmt.Errorf("required_test_allowed_commands_invalid")
		}
		allowed[strings.TrimSpace(name)] = abs
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("required_test_allowed_commands_required")
	}
	return allowed, nil
}

func requiredTestAllowedCommandEntriesFromProjectConfigV0(runtime serverProjectConfigRequiredTestRunnerV0) []string {
	if raw := strings.TrimSpace(os.Getenv(envRequiredTestAllowedCommandsV0)); raw != "" {
		return strings.Split(raw, ",")
	}
	keys := make([]string, 0, len(runtime.AllowedCommands))
	for name := range runtime.AllowedCommands {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	entries := make([]string, 0, len(keys))
	for _, name := range keys {
		entries = append(entries, name+"="+runtime.AllowedCommands[name])
	}
	return entries
}

func requiredTestOutputDirFromProjectConfigV0(
	serverConfig orquestaserver.ConfigV0,
	projectConfig serverProjectConfigFileV0,
) (string, error) {
	outputDir := stringProjectConfigOrEnvOrDefaultV0(
		envRequiredTestOutputDirV0,
		projectConfig.RequiredTestRunner.OutputDir,
		"",
	)
	if outputDir == "" {
		outputDir = filepath.Join(serverConfig.StateDir, "required-test-output")
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("required_test_output_dir_invalid")
	}
	return absOutputDir, nil
}

func requiredTestEnvFromProjectConfigV0(
	projectConfig serverProjectConfigFileV0,
	allowed map[string]string,
	outputDir string,
) ([]string, error) {
	env := []string(nil)
	provided := map[string]bool{}
	for _, item := range requiredTestEnvEntriesFromProjectConfigV0(projectConfig.RequiredTestRunner) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, _, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("required_test_env_invalid")
		}
		provided[strings.ToUpper(strings.TrimSpace(key))] = true
		env = append(env, item)
	}
	if _, ok := allowed["go"]; ok {
		goCacheDir := requiredTestGoCacheDirV0(outputDir)
		goEnv := map[string]string{
			"GOCACHE":    goCacheDir,
			"GOTMPDIR":   filepath.Join(outputDir, "go-tmp"),
			"GOPATH":     filepath.Join(outputDir, "go-path"),
			"GOMODCACHE": filepath.Join(outputDir, "go-mod-cache"),
		}
		for _, key := range []string{"GOCACHE", "GOTMPDIR", "GOPATH", "GOMODCACHE"} {
			if provided[key] {
				continue
			}
			dir := goEnv[key]
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return nil, fmt.Errorf("required_test_go_env_unavailable")
			}
			env = append(env, key+"="+dir)
		}
	}
	return env, nil
}

func requiredTestEnvEntriesFromProjectConfigV0(runtime serverProjectConfigRequiredTestRunnerV0) []string {
	if raw := strings.TrimSpace(os.Getenv(envRequiredTestEnvV0)); raw != "" {
		return strings.Split(raw, ",")
	}
	keys := make([]string, 0, len(runtime.Environment))
	for key := range runtime.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]string, 0, len(keys))
	for _, key := range keys {
		entries = append(entries, key+"="+runtime.Environment[key])
	}
	return entries
}

func requiredTestGoCacheDirV0(outputDir string) string {
	parent := strings.TrimSpace(os.Getenv("GOCACHE"))
	if requiredTestReusableGoCacheDirV0(parent) {
		return parent
	}
	return filepath.Join(outputDir, "go-build-cache")
}

func requiredTestReusableGoCacheDirV0(path string) bool {
	if path == "" ||
		!filepath.IsAbs(path) ||
		strings.ContainsAny(path, "\n\r") ||
		requiredTestPathHasCredentialMarkerV0(path) {
		return false
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return false
	}
	probe, err := os.CreateTemp(path, ".orquesta-required-test-gocache-probe-*")
	if err != nil {
		return false
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return true
}

func requiredTestPathHasCredentialMarkerV0(value string) bool {
	low := strings.ToLower(strings.TrimSpace(value))
	for _, marker := range []string{"$home", "${home}", "%userprofile%", "oauth", "token", "secret", "api_key", "credential", "password"} {
		if strings.Contains(low, marker) {
			return true
		}
	}
	return false
}
