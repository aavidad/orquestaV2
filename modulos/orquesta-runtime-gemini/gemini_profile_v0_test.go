package orquestaruntimegemini

import (
	"path/filepath"
	"testing"
)

func TestValidateGeminiConnectorProfileV0AceptaOptInSeguro(t *testing.T) {
	root := t.TempDir()
	profile := GeminiConnectorProfileV0{
		SchemaVersion:  GeminiConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "gemini"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
		Model:          "gemini-2.5-pro",
		ApprovalMode:   "auto_edit",
		OutputFormat:   "text",
	}
	if issues := ValidateGeminiConnectorProfileV0(profile); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateGeminiConnectorProfileV0RechazaPromptEnExtraArgs(t *testing.T) {
	root := t.TempDir()
	profile := GeminiConnectorProfileV0{
		SchemaVersion:  GeminiConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "gemini"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "runtime"),
		ExtraArgs:      []string{"--prompt"},
	}
	issues := ValidateGeminiConnectorProfileV0(profile)
	if len(issues) != 1 || issues[0].Field != "extra_args[0]" {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateGeminiConnectorProfileV0AceptaRuntimeDentroDelProyecto(t *testing.T) {
	root := t.TempDir()
	profile := GeminiConnectorProfileV0{
		SchemaVersion:  GeminiConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "gemini"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: filepath.Join(root, "project", ".orquesta-runtime", "agent"),
		ApprovalMode:   "auto_edit",
	}

	if issues := ValidateGeminiConnectorProfileV0(profile); len(issues) != 0 {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateGeminiConnectorProfileV0RechazaRuntimeQueContieneProyecto(t *testing.T) {
	root := t.TempDir()
	profile := GeminiConnectorProfileV0{
		SchemaVersion:  GeminiConnectorProfileSchemaVersionV0,
		OptIn:          true,
		CommandPath:    filepath.Join(root, "gemini"),
		ProjectWorkDir: filepath.Join(root, "project"),
		RuntimeWorkDir: root,
		ApprovalMode:   "auto_edit",
	}

	issues := ValidateGeminiConnectorProfileV0(profile)
	if len(issues) == 0 {
		t.Fatalf("esperaba issue")
	}
	found := false
	for _, issue := range issues {
		if string(issue.Code) == string(GeminiConnectorPathInvalidV0) && issue.Field == "runtime_work_dir" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues=%+v", issues)
	}
}

func TestValidateGeminiConnectorProfileV0RechazaPlacementInconsistente(t *testing.T) {
	root := t.TempDir()
	profile := GeminiConnectorProfileV0{
		SchemaVersion:           GeminiConnectorProfileSchemaVersionV0,
		OptIn:                   true,
		CommandPath:             filepath.Join(root, "gemini"),
		ProjectWorkDir:          filepath.Join(root, "project"),
		RuntimeWorkDir:          filepath.Join(root, "runtime"),
		RuntimeWorkDirPlacement: GeminiRuntimeWorkDirInsideProjectV0,
		ApprovalMode:            "auto_edit",
	}

	issues := ValidateGeminiConnectorProfileV0(profile)
	if len(issues) == 0 {
		t.Fatalf("esperaba issue")
	}
	found := false
	for _, issue := range issues {
		if string(issue.Code) == string(GeminiConnectorValueInvalidV0) && issue.Field == "runtime_work_dir_placement" {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues=%+v", issues)
	}
}
