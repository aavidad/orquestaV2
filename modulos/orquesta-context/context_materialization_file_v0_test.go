package orquestacontext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileContextRefStoreV0RequiereRootExplicito(t *testing.T) {
	store, issues := NewFileContextRefStoreV0("")
	if store != nil {
		t.Fatalf("store=%+v want nil", store)
	}
	requireMaterializationIssueV0(t, issues, ErrContextMaterializationRootInvalidoV0)
}

func TestFileContextRefStoreV0BloqueaTraversal(t *testing.T) {
	store := newFileContextStoreForTestV0(t, t.TempDir())
	_, issues := store.ReadContextRefV0("modulos/orquesta-runtime/../secret.txt", 100)

	requireMaterializationIssueV0(t, issues, ErrContextMaterializationRefInvalidaV0)
}

func TestFileContextRefStoreV0ReportaRefNoEncontrada(t *testing.T) {
	store := newFileContextStoreForTestV0(t, t.TempDir())
	_, issues := store.ReadContextRefV0("modulos/orquesta-runtime/AGENTS.md", 100)

	requireMaterializationIssueV0(t, issues, ErrContextMaterializationRefNoEncontradaV0)
}

func createContextMaterializationRepoV0(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"modulos/orquesta-runtime/AGENTS.md":                             "reglas runtime\n",
		"modulos/orquesta-runtime/README.md":                             "runtime\n",
		"modulos/orquesta-runtime/docs/contratos.md":                     "contratos\n",
		"modulos/orquesta-runtime/docs/tareas.md":                        "tareas\n",
		"modulos/orquesta-runtime/docs/pruebas.md":                       "pruebas\n",
		"modulos/orquesta-runtime/process_runtime_connector_types_v0.go": "package orquestaruntime\n",
	}
	for path, content := range files {
		writeContextFileV0(t, root, path, content)
	}
	return root
}

func writeContextFileV0(t *testing.T, root string, path string, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o700); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func newFileContextStoreForTestV0(t *testing.T, root string) *FileContextRefStoreV0 {
	t.Helper()
	store, issues := NewFileContextRefStoreV0(root)
	if len(issues) != 0 {
		t.Fatalf("new file context store issues=%+v", issues)
	}
	return store
}

func requireMaterializedModeV0(
	t *testing.T,
	materialized ContextMaterializedBundleV0,
	sourceRef string,
	mode string,
) {
	t.Helper()
	entry := materializedEntryBySourceV0(materialized, sourceRef)
	if entry.Mode != mode {
		t.Fatalf("entry %q mode=%q want %q entry=%+v", sourceRef, entry.Mode, mode, entry)
	}
}

func materializedEntryBySourceV0(
	materialized ContextMaterializedBundleV0,
	sourceRef string,
) ContextMaterializedEntryV0 {
	for _, entry := range materialized.Entries {
		if entry.SourceRef == sourceRef {
			return entry
		}
	}
	return ContextMaterializedEntryV0{}
}

func requireMaterializationIssueV0(
	t *testing.T,
	issues []ContextMaterializationIssueV0,
	code ContextMaterializationIssueCodeV0,
) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q not found in %+v", code, issues)
}
