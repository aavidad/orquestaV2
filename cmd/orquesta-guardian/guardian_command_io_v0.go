package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	envGuardianCommandOutputMaxBytesV0     = "ORQUESTA_GUARDIAN_COMMAND_OUTPUT_MAX_BYTES"
	envGuardianCommandEnvAllowlistV0       = "ORQUESTA_GUARDIAN_COMMAND_ENV_ALLOWLIST"
	defaultGuardianCommandOutputMaxBytesV0 = int64(64 * 1024)
	guardianOutputTruncatedReasonV0        = "guardian_command_output_budget_exceeded"
)

type guardianOutputCaptureV0 struct {
	config    guardianConfigV0
	tail      bytes.Buffer
	seenBytes int64
	truncated bool
}

func newGuardianOutputCaptureV0(config guardianConfigV0) *guardianOutputCaptureV0 {
	return &guardianOutputCaptureV0{config: config}
}

func (capture *guardianOutputCaptureV0) Write(payload []byte) (int, error) {
	if len(payload) == 0 {
		return 0, nil
	}
	capture.seenBytes += int64(len(payload))
	redacted := []byte(redactGuardianDiagnosticV0(capture.config, string(payload)))
	capture.tail.Write(redacted)
	maxBytes := guardianOutputMaxBytesV0(capture.config)
	if int64(capture.tail.Len()) > maxBytes {
		capture.truncated = true
		body := capture.tail.Bytes()
		body = body[int64(len(body))-maxBytes:]
		capture.tail.Reset()
		capture.tail.Write(body)
	}
	return len(payload), nil
}

func (capture *guardianOutputCaptureV0) BytesSeen() int64 {
	return capture.seenBytes
}

func (capture *guardianOutputCaptureV0) Truncated() bool {
	return capture.truncated
}

func (capture *guardianOutputCaptureV0) ReasonCode() string {
	if capture.truncated {
		return guardianOutputTruncatedReasonV0
	}
	return ""
}

func (capture *guardianOutputCaptureV0) WriteLog(path string) error {
	return writeGuardianRedactedLogV0(capture.config, path, capture.tail.String())
}

func writeGuardianRedactedLogV0(config guardianConfigV0, path string, content string) error {
	return writeGuardianTextFileV0(path, redactGuardianDiagnosticV0(config, content))
}

func guardianOutputMaxBytesV0(config guardianConfigV0) int64 {
	if config.CommandOutputMaxBytes > 0 {
		return config.CommandOutputMaxBytes
	}
	return defaultGuardianCommandOutputMaxBytesV0
}

func defaultGuardianEnvAllowlistV0() []string {
	if runtime.GOOS == "windows" {
		return []string{"PATH", "TEMP", "TMP", "SystemRoot", "ComSpec", "PATHEXT", "WINDIR"}
	}
	return []string{"PATH", "TMPDIR", "TEMP", "TMP"}
}

func guardianCommandEnvV0(config guardianConfigV0, explicit []string) []string {
	allow := map[string]bool{}
	for _, key := range config.EnvAllowlist {
		key = strings.TrimSpace(key)
		if key != "" {
			allow[key] = true
		}
	}
	env := make([]string, 0, len(allow)+len(explicit)+4)
	for _, item := range os.Environ() {
		key, _, ok := strings.Cut(item, "=")
		if ok && allow[key] {
			env = append(env, item)
		}
	}
	env = append(env,
		"ORQUESTA_GUARDIAN_ENV_POLICY=minimal_allowlist",
		"ORQUESTA_GUARDIAN_OUTPUT_POLICY=tail_redacted",
		"GOCACHE="+filepath.Join(config.StateDir, "cache", "go-build"),
		"XDG_CACHE_HOME="+filepath.Join(config.StateDir, "cache"),
	)
	env = append(env, explicit...)
	return env
}

func redactGuardianDiagnosticV0(config guardianConfigV0, value string) string {
	if value == "" {
		return ""
	}
	out := value
	for _, item := range []string{
		config.ProjectDir,
		config.StateDir,
		config.CurrentBin,
		config.CandidateBin,
		config.LastGoodBin,
		config.RepairCodexRuntimeDir,
	} {
		out = redactLiteralV0(out, item, "[REDACTED_PATH]")
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = redactLiteralV0(out, home, "[REDACTED_HOME]")
	}
	for _, pattern := range guardianSecretPatternsV0() {
		out = pattern.ReplaceAllString(out, "$1[REDACTED]")
	}
	out = guardianURLCredentialPatternV0().ReplaceAllString(out, "$1[REDACTED]@")
	out = guardianPrivateKeyPatternV0().ReplaceAllString(out, "[REDACTED_PRIVATE_KEY]")
	return out
}

func redactLiteralV0(value string, literal string, replacement string) string {
	literal = strings.TrimSpace(literal)
	if literal == "" {
		return value
	}
	value = strings.ReplaceAll(value, literal, replacement)
	if filepath.Separator != '/' {
		value = strings.ReplaceAll(value, filepath.ToSlash(literal), replacement)
	}
	return value
}

func guardianSecretPatternsV0() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(api[_-]?key\s*[:=]\s*)([^\s"'<>]+)`),
		regexp.MustCompile(`(?i)\b(client[_-]?secret\s*[:=]\s*)([^\s"'<>]+)`),
		regexp.MustCompile(`(?i)\b(access[_-]?token\s*[:=]\s*)([^\s"'<>]+)`),
		regexp.MustCompile(`(?i)\b(refresh[_-]?token\s*[:=]\s*)([^\s"'<>]+)`),
		regexp.MustCompile(`(?i)\b(password\s*[:=]\s*)([^\s"'<>]+)`),
		regexp.MustCompile(`(?i)\b(dsn\s*[:=]\s*)([^\s"'<>]+)`),
		regexp.MustCompile(`(?i)\b(authorization\s*[:=]\s*bearer\s+)([^\s"'<>]+)`),
		regexp.MustCompile(`(?i)\b(bearer\s+)([A-Za-z0-9._~+/=-]{12,})`),
	}
}

func guardianURLCredentialPatternV0() *regexp.Regexp {
	return regexp.MustCompile(`([a-z][a-z0-9+.-]*://)[^/\s:@]+:[^/\s@]+@`)
}

func guardianPrivateKeyPatternV0() *regexp.Regexp {
	return regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)
}
