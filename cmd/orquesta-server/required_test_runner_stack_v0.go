package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimerequiredtest "orquesta/modulos/orquesta-runtime-required-test"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func requiredTestRunnerFromEnvV0(
	serverConfig orquestaserver.ConfigV0,
	evidenceWriter orquestacionnucleoapp.RequiredTestEvidenceWriterPortV0,
) (orquestacionnucleoapp.RequiredTestRunnerPortV0, error) {
	if strings.TrimSpace(os.Getenv(envRequiredTestRunnerEnabledV0)) != "1" {
		return nil, nil
	}
	if evidenceWriter == nil {
		return nil, fmt.Errorf("required_test_evidence_writer_required")
	}
	allowed, err := requiredTestAllowedCommandsFromEnvV0()
	if err != nil {
		return nil, err
	}
	absOutputDir, err := requiredTestOutputDirFromEnvV0(serverConfig)
	if err != nil {
		return nil, err
	}
	env, err := requiredTestEnvFromEnvV0(allowed, absOutputDir)
	if err != nil {
		return nil, err
	}
	return orquestacionnucleoapp.RequiredTestRunnerV0{
		Executor: orquestaruntimerequiredtest.LocalCommandExecutorV0{
			ProjectWorkDir:  serverConfig.ProjectWorkDir,
			OutputDir:       absOutputDir,
			AllowedCommands: allowed,
			Env:             env,
			MaxOutputBytes:  int64(intEnvOrDefaultV0(envRequiredTestMaxOutputBytesV0, 1024*1024)),
			MaxArtifacts:    intEnvOrDefaultV0(envRequiredTestOutputMaxArtifactsV0, 200),
		},
		EvidenceWriter: evidenceWriter,
	}, nil
}

func requiredTestAllowedCommandsFromEnvV0() (map[string]string, error) {
	allowed := map[string]string{}
	if goCommand := strings.TrimSpace(os.Getenv(envRequiredTestGoCommandV0)); goCommand != "" {
		abs, err := filepath.Abs(goCommand)
		if err != nil {
			return nil, fmt.Errorf("required_test_go_command_invalid")
		}
		allowed["go"] = abs
	}
	for _, item := range strings.Split(os.Getenv(envRequiredTestAllowedCommandsV0), ",") {
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

func requiredTestOutputDirFromEnvV0(serverConfig orquestaserver.ConfigV0) (string, error) {
	outputDir := strings.TrimSpace(os.Getenv(envRequiredTestOutputDirV0))
	if outputDir == "" {
		outputDir = filepath.Join(serverConfig.StateDir, "required-test-output")
	}
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("required_test_output_dir_invalid")
	}
	return absOutputDir, nil
}

func requiredTestEnvFromEnvV0(
	allowed map[string]string,
	outputDir string,
) ([]string, error) {
	env := []string(nil)
	provided := map[string]bool{}
	for _, item := range strings.Split(os.Getenv(envRequiredTestEnvV0), ",") {
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
		goEnv := map[string]string{
			"GOCACHE":    filepath.Join(outputDir, "go-build-cache"),
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
