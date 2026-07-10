package orquestaruntimecodex

import (
	"strings"
	"testing"
)

func TestBuildCodexWrapperScriptV0NoMaterializaModeloVacio(t *testing.T) {
	profile := CodexConnectorProfileV0{CommandPath: "/bin/codex", ProjectWorkDir: "/project", RuntimeWorkDir: "/runtime"}
	if script := BuildCodexWrapperScriptV0(profile); strings.Contains(script, "-m ") {
		t.Fatalf("wrapper materializa modelo vacio: %s", script)
	}
}

func TestBuildCodexWrapperScriptV0NoMaterializaEsfuerzoVacio(t *testing.T) {
	profile := CodexConnectorProfileV0{CommandPath: "/bin/codex", ProjectWorkDir: "/project", RuntimeWorkDir: "/runtime"}
	if script := BuildCodexWrapperScriptV0(profile); strings.Contains(script, "model_reasoning_effort") {
		t.Fatalf("wrapper materializa esfuerzo vacio: %s", script)
	}
}
