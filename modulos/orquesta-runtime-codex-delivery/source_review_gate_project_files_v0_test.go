package orquestaruntimecodexdelivery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func TestCodexReviewGateProjectFileEvidenceV0CuentaLineasReales(t *testing.T) {
	projectDir := t.TempDir()
	writeProjectFileForReviewGateTestV0(t, projectDir, "internal/api/handler.go", "package api\nfunc Handler() {}\n")
	descriptor := CodexReceiptDescriptorV0{ProjectWorkDir: projectDir}
	ack := orquestaruntimecodex.CodexAgentAckV0{
		Files: orquestaruntimecodex.EvidenceListV0{"internal/api/handler.go"},
	}

	evidence, err := (CodexReviewGateProjectFileEvidenceV0{}).
		BuildCodexReviewGateFileEvidenceV0(context.Background(), descriptor, ack)
	if err != nil {
		t.Fatalf("BuildCodexReviewGateFileEvidenceV0: %v", err)
	}
	if len(evidence.Issues) != 0 {
		t.Fatalf("issues=%+v", evidence.Issues)
	}
	if len(evidence.Files) != 1 || evidence.Files[0].Path != "internal/api/handler.go" ||
		evidence.Files[0].LineCount != 2 {
		t.Fatalf("files=%+v", evidence.Files)
	}
}

func TestCodexReviewGateProjectFileEvidenceV0MarcaFicheroAusente(t *testing.T) {
	descriptor := CodexReceiptDescriptorV0{ProjectWorkDir: t.TempDir()}
	ack := orquestaruntimecodex.CodexAgentAckV0{
		Files: orquestaruntimecodex.EvidenceListV0{"internal/api/missing.go"},
	}

	evidence, err := (CodexReviewGateProjectFileEvidenceV0{}).
		BuildCodexReviewGateFileEvidenceV0(context.Background(), descriptor, ack)
	if err != nil {
		t.Fatalf("BuildCodexReviewGateFileEvidenceV0: %v", err)
	}
	if len(evidence.Files) != 1 || evidence.Files[0].Path != "internal/api/missing.go" {
		t.Fatalf("files=%+v", evidence.Files)
	}
	if len(evidence.Issues) != 1 || evidence.Issues[0].Code != "delivery_file_missing" {
		t.Fatalf("issues=%+v", evidence.Issues)
	}
}

func writeProjectFileForReviewGateTestV0(
	t *testing.T,
	root string,
	rel string,
	content string,
) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
