package orquestaruntimerequiredtest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarails "orquesta/modulos/orquesta-rails"
)

const defaultOutputArtifactRetentionV0 = 200

func writeOutputArtifactV0(
	executor LocalCommandExecutorV0,
	request orquestacionnucleoapp.RequiredTestCommandExecutionRequestV0,
	status orquestacionnucleoapp.RequiredTestEvidenceStatusV0,
	output outputBufferV0,
) (string, orquestacionnucleoapp.RequiredTestEvidenceStatusV0, error) {
	ref := outputArtifactRefV0(request)
	path := filepath.Join(executor.OutputDir, strings.TrimPrefix(ref, "required-test-output-v0/"))
	body, applied, blocked := redactRequiredTestOutputV0(output.String())
	command, commandApplied, commandBlocked := redactRequiredTestOutputV0(strings.TrimSpace(request.TestCommand))
	if blocked || commandBlocked {
		status = orquestacionnucleoapp.RequiredTestEvidenceStatusFailedV0
		body = "required_test_output_blocked_sensitive_unredactable\n"
	}
	content := strings.Join([]string{
		"schema_version=" + outputSchemaVersionV0,
		"test_command=" + command,
		"status=" + string(status),
		fmt.Sprintf("truncated=%t", output.Truncated()),
		"output_redacted=true",
		fmt.Sprintf("redaction_applied=%t", applied || commandApplied || blocked || commandBlocked),
		fmt.Sprintf("max_output_bytes=%d", normalizedMaxOutputBytesV0(executor.MaxOutputBytes)),
		fmt.Sprintf("retention_max_artifacts=%d", normalizedMaxArtifactsV0(executor.MaxArtifacts)),
		"",
		body,
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return "", "", err
	}
	if err := applyOutputArtifactRetentionV0(executor.OutputDir, path, executor.MaxArtifacts); err != nil {
		return "", "", err
	}
	return ref, status, nil
}

func redactRequiredTestOutputV0(value string) (string, bool, bool) {
	if !utf8.ValidString(value) {
		return "", true, true
	}
	redacted, changed := orquestarails.RedactOperationalTextForFieldV0(
		"required_test_output",
		"stdout_stderr",
		value,
	)
	if strings.Contains(strings.ToLower(redacted), "-----begin ") {
		return "", true, true
	}
	return redacted, changed, false
}

func applyOutputArtifactRetentionV0(outputDir string, currentPath string, maxArtifacts int) error {
	limit := normalizedMaxArtifactsV0(maxArtifacts)
	if limit <= 0 {
		return nil
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}
	files := make([]outputArtifactFileV0, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		path := filepath.Join(outputDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, outputArtifactFileV0{Path: path, Name: entry.Name(), UnixNano: info.ModTime().UnixNano()})
	}
	if len(files) <= limit {
		return nil
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].UnixNano == files[j].UnixNano {
			return files[i].Name < files[j].Name
		}
		return files[i].UnixNano < files[j].UnixNano
	})
	currentClean := filepath.Clean(currentPath)
	for _, file := range files[:len(files)-limit] {
		if filepath.Clean(file.Path) == currentClean {
			continue
		}
		if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func normalizedMaxOutputBytesV0(value int64) int64 {
	if value <= 0 {
		return 1024 * 1024
	}
	return value
}

func normalizedMaxArtifactsV0(value int) int {
	if value <= 0 {
		return defaultOutputArtifactRetentionV0
	}
	return value
}

type outputArtifactFileV0 struct {
	Path     string
	Name     string
	UnixNano int64
}
