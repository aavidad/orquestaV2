package orquestaopesbridge

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestOPESValidateTopicPackageV1FinalPackageBloqueaRAGSinSourceVariantYVisualPerdido(t *testing.T) {
	packageDir := t.TempDir()
	writeFinalPackageFixtureV0(t, packageDir, false)

	output, err := runOPESFinalPackageValidatorV0(t, packageDir)
	if err == nil {
		t.Fatalf("validator ok inesperado output=%s", output)
	}
	for _, want := range []string{
		"rag_chunks_missing_source_variant=1",
		"visual_manifest_ref_not_in_html=html_final/img/tema_001_visual.webp",
		"visual_manifest_ref_not_in_html=html_ampliado/img/tema_001_visual.webp",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output=%s falta %s", output, want)
		}
	}
}

func TestOPESValidateTopicPackageV1FinalPackageAceptaRAGMetadataYVisualPreservado(t *testing.T) {
	packageDir := t.TempDir()
	writeFinalPackageFixtureV0(t, packageDir, true)

	output, err := runOPESFinalPackageValidatorV0(t, packageDir)
	if err != nil {
		t.Fatalf("validator err=%v output=%s", err, output)
	}
	if !strings.Contains(output, "OK final_package=") {
		t.Fatalf("output=%s", output)
	}
}

func runOPESFinalPackageValidatorV0(t *testing.T, packageDir string) (string, error) {
	t.Helper()
	cmd := exec.Command(
		"python3",
		filepath.Join("scripts", "opes_validate_topic_package_v1.py"),
		"--final-package-dir",
		packageDir,
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeFinalPackageFixtureV0(t *testing.T, packageDir string, valid bool) {
	t.Helper()
	writeFileV0(t, filepath.Join(packageDir, "manifest_cierre.json"), `{
  "schema": "opes_final_package_evidence_manifest.v0",
  "status": "listo_para_revision_operador",
  "package_ref": "package-ref-001",
  "manifest_ref": "manifest-cierre-ref-001"
}`)
	writeFileV0(t, filepath.Join(packageDir, "rag", "manifest.json"), `{
  "schema": "opes.rag_manifest.v2",
  "course_id": "course-ref-001",
  "chunks": "rag/corpus/chunks.jsonl",
  "summary": "rag/corpus/summary.json",
  "status": "pass"
}`)
	writeFileV0(t, filepath.Join(packageDir, "rag", "corpus", "summary.json"), `{
  "schema": "opes.rag_summary.v2",
  "course_id": "course-ref-001",
  "sources": [
    {
      "path": "html_final/tema_001.html",
      "source_variant": "html_final",
      "topic_id": "tema_001"
    }
  ]
}`)
	chunk := `{"chunk_id":"chunk-ref-001","course_id":"course-ref-001","source_path":"html_final/tema_001.html","text":"Texto del tema."}`
	if valid {
		chunk = `{"chunk_id":"chunk-ref-001","course_id":"course-ref-001","source_variant":"html_final","source_path":"html_final/tema_001.html","text":"Texto del tema."}`
	}
	writeFileV0(t, filepath.Join(packageDir, "rag", "corpus", "chunks.jsonl"), chunk+"\n")

	writeFileV0(t, filepath.Join(packageDir, "08_assets", "visuals_manifest.json"), `{
  "schema_version": "opes.visuals_manifest.v1",
  "summary": {
    "asset_count": 1,
    "pending_count": 0
  },
  "assets": [
    {
      "topic": 1,
      "file": "img/tema_001_visual.webp",
      "variants": [
        "html_final/img/tema_001_visual.webp",
        "html_ampliado/img/tema_001_visual.webp"
      ],
      "status": "raster_profesional",
      "alt": "Visual didactico del tema 01."
    }
  ]
}`)
	writeFileV0(t, filepath.Join(packageDir, "html_final", "img", "tema_001_visual.webp"), "fake-webp")
	writeFileV0(t, filepath.Join(packageDir, "html_ampliado", "img", "tema_001_visual.webp"), "fake-webp")

	htmlBody := "<html><body><h1>Tema 001</h1><p>Texto sin visual.</p></body></html>"
	if valid {
		htmlBody = `<html><body><h1>Tema 001</h1><figure><img src="img/tema_001_visual.webp" alt="Visual didactico"></figure></body></html>`
	}
	writeFileV0(t, filepath.Join(packageDir, "html_final", "tema_001.html"), htmlBody)
	writeFileV0(t, filepath.Join(packageDir, "html_ampliado", "tema_001.html"), htmlBody)
}

func writeFileV0(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
