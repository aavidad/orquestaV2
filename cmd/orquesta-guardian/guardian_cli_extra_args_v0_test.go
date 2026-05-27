package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestGuardianCLIExtraArgsV0BloqueaPosicionalesEnComandosConEfectos(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{
			name: "check-promote",
			args: []string{
				"check-promote",
				"--current-bin", "orquesta-server",
				"--skip-health",
				"secreto-posicional",
			},
		},
		{
			name: "restore-last-good",
			args: []string{
				"restore-last-good",
				"--current-bin", "orquesta-server",
				"secreto-posicional",
				"otro",
			},
		},
		{
			name: "shutdown-server",
			args: []string{
				"shutdown-server",
				"--server-addr", "127.0.0.1:1",
				"secreto-posicional",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			exitCode := runMain(tt.args, io.Discard, &stderr)
			if exitCode != 2 {
				t.Fatalf("exitCode=%d stderr=%s", exitCode, stderr.String())
			}
			got := stderr.String()
			if !strings.Contains(got, guardianConfigExtraArgsV0+":positional_args:") {
				t.Fatalf("stderr sin categoria/conteo publico: %s", got)
			}
			if strings.Contains(got, "secreto-posicional") {
				t.Fatalf("stderr eco de argumento crudo: %s", got)
			}
		})
	}
}
