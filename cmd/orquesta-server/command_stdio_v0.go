package main

import (
	"encoding/json"
	"io"

	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func writeCommandJSONOutputV0(stdout io.Writer, payload any) error {
	return json.NewEncoder(stdout).Encode(payload)
}

func reportCommandStdioWriteFailureV0(stderr io.Writer, command, stream, stage string, err error) int {
	if stderr == nil {
		return 1
	}
	failure := orquestaobservability.NewCommandStdioWriteFailureV0(command, stream, stage, err)
	_ = json.NewEncoder(stderr).Encode(failure)
	return 1
}

func writeCommandTextOutputV0(stdout io.Writer, text string) error {
	_, err := io.WriteString(stdout, text)
	return err
}
