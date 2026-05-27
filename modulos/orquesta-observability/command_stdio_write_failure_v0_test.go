package orquestaobservability

import (
	"errors"
	"strings"
	"testing"
)

func TestNewCommandStdioWriteFailureV0RedactaErrorDeStdout(t *testing.T) {
	failure := NewCommandStdioWriteFailureV0(
		"codex-wave-status",
		"stdout",
		"json_encode",
		errors.New("write /home/alberto/private/stdout.log: broken pipe"),
	)

	if failure.SchemaVersion != CommandStdioWriteFailureSchemaVersionV0 ||
		failure.ReasonCode != CommandOutputWriteFailedV0 ||
		failure.Command != "codex-wave-status" ||
		failure.Stream != "stdout" ||
		failure.Stage != "json_encode" ||
		!failure.Redacted ||
		!strings.HasPrefix(failure.ErrorRef, "stdio-write-error-ref-") {
		t.Fatalf("failure=%+v", failure)
	}
	if strings.Contains(failure.ErrorRef, "home") || strings.Contains(failure.ErrorRef, "stdout.log") {
		t.Fatalf("error_ref filtra detalle local: %s", failure.ErrorRef)
	}
}

func TestNewCommandStdioWriteFailureV0ClasificaStderr(t *testing.T) {
	failure := NewCommandStdioWriteFailureV0("run-status", "stderr", "error_line", errors.New("closed"))
	if failure.ReasonCode != CommandErrorWriteFailedV0 || failure.Stream != "stderr" {
		t.Fatalf("failure=%+v", failure)
	}
}
