package orquestacli

import (
	"bytes"
	"context"
	"testing"
)

func TestRunOrquestaCLIV0FalloStdoutDevuelveReasonCode(t *testing.T) {
	var stderr bytes.Buffer
	code := RunOrquestaCLIV0(context.Background(), []string{"--help"}, OrquestaCLIRunnerV0{
		Stdout: failingCLIWriterV0{},
		Stderr: &stderr,
	})
	if code != 1 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("command_output_write_failed")) ||
		bytes.Contains(stderr.Bytes(), []byte("/tmp/orquesta")) {
		t.Fatalf("stderr sin fallo redactado: %s", stderr.String())
	}
}

type failingCLIWriterV0 struct{}

func (failingCLIWriterV0) Write(_ []byte) (int, error) {
	return 0, errCLIWriterFailedV0{}
}

type errCLIWriterFailedV0 struct{}

func (errCLIWriterFailedV0) Error() string {
	return "closed private /tmp/orquesta/stdout"
}
