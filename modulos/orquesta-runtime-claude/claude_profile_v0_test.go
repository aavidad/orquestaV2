package orquestaruntimeclaude

import (
	"path/filepath"
	"testing"
)

func TestValidateClaudeConnectorProfileV0AceptaOptIn(t *testing.T) {
	root := t.TempDir()
	profile := ClaudeConnectorProfileV0{
		SchemaVersion:  ClaudeConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "claude"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
		Model:          "sonnet",
		PermissionMode: "dontAsk",
		OutputFormat:   "text",
	}
	if issues := ValidateClaudeConnectorProfileV0(profile); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateClaudeConnectorProfileV0RechazaArgsPropios(t *testing.T) {
	root := t.TempDir()
	profile := ClaudeConnectorProfileV0{
		SchemaVersion:  ClaudeConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "claude"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
		ExtraArgs:      []string{"--print"},
	}
	if issues := ValidateClaudeConnectorProfileV0(profile); len(issues) == 0 {
		t.Fatalf("esperaba issue")
	}
}

func TestValidateClaudeConnectorProfileV0AceptaRuntimeDentroDelProyecto(t *testing.T) {
	root := t.TempDir()
	profile := ClaudeConnectorProfileV0{
		SchemaVersion:  ClaudeConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "claude"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "project", ".orquesta-runtime", "agent"),
	}

	if issues := ValidateClaudeConnectorProfileV0(profile); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateClaudeConnectorProfileV0RechazaRuntimeQueContieneProyecto(t *testing.T) {
	root := t.TempDir()
	profile := ClaudeConnectorProfileV0{
		SchemaVersion:  ClaudeConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "claude"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: root,
	}

	issues := ValidateClaudeConnectorProfileV0(profile)
	if len(issues) == 0 {
		t.Fatalf("esperaba issue")
	}
	found := false
	for _, issue := range issues {
		if string(issue.Code) == string(ClaudeConnectorPathInvalidV0) && issue.Field == "runtime_work_dir" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateClaudeConnectorProfileV0RechazaPlacementInconsistente(t *testing.T) {
	root := t.TempDir()
	profile := ClaudeConnectorProfileV0{
		SchemaVersion:           ClaudeConnectorProfileSchemaVersionV0,
		OptIn:                   true,
		CommandPath:             filepath.Join(root, "claude"),
		ProjectWorkDir:          filepath.Join(root, "project"),
		RuntimeWorkDir:          filepath.Join(root, "runtime"),
		RuntimeWorkDirPlacement: ClaudeRuntimeWorkDirInsideProjectV0,
	}

	issues := ValidateClaudeConnectorProfileV0(profile)
	if len(issues) == 0 {
		t.Fatalf("esperaba issue")
	}
	found := false
	for _, issue := range issues {
		if string(issue.Code) == string(ClaudeConnectorValueInvalidV0) && issue.Field == "runtime_work_dir_placement" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues=%+v", issues)
	}
}
