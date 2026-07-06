package main

import (
	"path/filepath"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
)

func TestServerGoalRequiredTestDependencyGraphFromGoListOutputV0(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	output := "orquesta/modulos/orquesta-director-tick-input|" + filepath.Join(root, "modulos/orquesta-director-tick-input") + "|\n" +
		"orquesta/modulos/orquesta-orchestration-core|" + filepath.Join(root, "modulos/orquesta-orchestration-core") + "|orquesta/modulos/orquesta-director-tick-input\tfmt\n" +
		"fmt|/usr/local/go/src/fmt|\n"

	graph, err := serverGoalRequiredTestDependencyGraphFromGoListOutputV0(root, output)
	if err != nil {
		t.Fatalf("graph: %v", err)
	}
	result := orquestaappcodexstack.ResolveGoalRequiredTestDependencyCommandsV0(
		orquestaappcodexstack.GoalRequiredTestDependencyRequestV0{
			WriteSet: []string{"modulos/orquesta-director-tick-input/tick_input_usecase_v0.go"},
		},
		graph,
	)
	if !serverGoalRequiredTestCommandContainsForTestV0(result.Commands, "./modulos/orquesta-orchestration-core") {
		t.Fatalf("commands=%v, falta orchestration-core", result.Commands)
	}
}

func TestServerGoalRequiredTestDependencyGraphFromGoListOutputV0RequiereRoot(t *testing.T) {
	_, err := serverGoalRequiredTestDependencyGraphFromGoListOutputV0("", "orquesta/modulos/a|/tmp/a|")
	if err == nil {
		t.Fatal("expected root error")
	}
}

func serverGoalRequiredTestCommandContainsForTestV0(commands []string, value string) bool {
	for _, command := range commands {
		if command == "go test -count=1 "+value {
			return true
		}
	}
	return false
}
