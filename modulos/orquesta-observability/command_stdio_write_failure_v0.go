package orquestaobservability

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	CommandStdioWriteFailureSchemaVersionV0 = "orquesta_command_stdio_write_failure.v0"
	CommandOutputWriteFailedV0              = "command_output_write_failed"
	CommandErrorWriteFailedV0               = "command_error_write_failed"
)

type CommandStdioWriteFailureV0 struct {
	SchemaVersion string `json:"schema_version"`
	ReasonCode    string `json:"reason_code"`
	Command       string `json:"command"`
	Stream        string `json:"stream"`
	Stage         string `json:"stage"`
	ErrorRef      string `json:"error_ref,omitempty"`
	Redacted      bool   `json:"redacted"`
}

func NewCommandStdioWriteFailureV0(command, stream, stage string, err error) CommandStdioWriteFailureV0 {
	stream = commandStdioTokenV0(stream, "stdout")
	reason := CommandOutputWriteFailedV0
	if stream == "stderr" {
		reason = CommandErrorWriteFailedV0
	}
	return CommandStdioWriteFailureV0{
		SchemaVersion: CommandStdioWriteFailureSchemaVersionV0,
		ReasonCode:    reason,
		Command:       commandStdioTokenV0(command, "command"),
		Stream:        stream,
		Stage:         commandStdioTokenV0(stage, "write"),
		ErrorRef:      commandStdioErrorRefV0(err),
		Redacted:      true,
	}
}

func commandStdioTokenV0(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return fallback
	}
	token := b.String()
	if len(token) > 80 {
		return token[:80]
	}
	return token
}

func commandStdioErrorRefV0(err error) string {
	if err == nil {
		return ""
	}
	errorText := err.Error()
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		errorText = unwrapped.Error()
	}
	digest := sha256.Sum256([]byte(errorText))
	return "stdio-write-error-ref-" + hex.EncodeToString(digest[:])[:16]
}
