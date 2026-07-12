package orquestatoolcapabilityfile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

func TestToolCapabilityFileCatalogV0ListaFiltraYOrdena(t *testing.T) {
	root := t.TempDir()
	writeToolCapabilityCatalogTestV0(t, root, []toolcapability.CapabilityManifestV0{
		toolCapabilityCatalogManifestTestV0("manifest-z", "cap-z", "tool-z", "en"),
		toolCapabilityCatalogManifestTestV0("manifest-a2", "cap-a", "tool-a", "es"),
		toolCapabilityCatalogManifestTestV0("manifest-a1", "cap-a", "tool-a", "en", "es"),
	})
	catalog, err := NewToolCapabilityFileCatalogV0(root)
	if err != nil {
		t.Fatalf("new catalog: %v", err)
	}
	manifests, err := catalog.ListCapabilityManifestsV0(context.Background(), toolcapability.CapabilityManifestFilterV0{
		CapabilityRef: "cap-a",
		Locale:        "es",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(manifests) != 2 || manifests[0].ManifestRef != "manifest-a1" || manifests[1].ManifestRef != "manifest-a2" {
		t.Fatalf("manifests=%+v", manifests)
	}
}

func TestToolCapabilityFileCatalogV0FicheroAusenteEsCatalogoVacio(t *testing.T) {
	catalog, err := NewToolCapabilityFileCatalogV0(t.TempDir())
	if err != nil {
		t.Fatalf("new catalog: %v", err)
	}
	manifests, err := catalog.ListCapabilityManifestsV0(context.Background(), toolcapability.CapabilityManifestFilterV0{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(manifests) != 0 {
		t.Fatalf("manifests=%+v", manifests)
	}
}

func TestToolCapabilityFileCatalogV0RechazaRootRelativoOSymlinks(t *testing.T) {
	if _, err := NewToolCapabilityFileCatalogV0("relative-root"); err == nil {
		t.Fatalf("expected relative root error")
	}
	parent := t.TempDir()
	realRoot := filepath.Join(parent, "real")
	if err := os.Mkdir(realRoot, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	linkRoot := filepath.Join(parent, "link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if _, err := NewToolCapabilityFileCatalogV0(linkRoot); err == nil || !strings.Contains(err.Error(), "symlinks") {
		t.Fatalf("expected symlink root error, got %v", err)
	}
}

func TestToolCapabilityFileCatalogV0RechazaFicheroSymlinkOGrande(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.json")
	if err := os.WriteFile(target, []byte(`{"schema_version":"tool_capability_file_catalog.v0","manifests":[]}`), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(root, toolCapabilityCatalogFileNameV0)); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	catalog, err := NewToolCapabilityFileCatalogV0(root)
	if err != nil {
		t.Fatalf("new catalog: %v", err)
	}
	if _, err := catalog.ListCapabilityManifestsV0(context.Background(), toolcapability.CapabilityManifestFilterV0{}); err == nil {
		t.Fatalf("expected symlink file error")
	}
}

func writeToolCapabilityCatalogTestV0(t *testing.T, root string, manifests []toolcapability.CapabilityManifestV0) {
	t.Helper()
	payload, err := json.Marshal(toolCapabilityFileCatalogEnvelopeV0{
		SchemaVersion: toolCapabilityCatalogSchemaV0,
		Manifests:     manifests,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, toolCapabilityCatalogFileNameV0), payload, 0o600); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
}

func toolCapabilityCatalogManifestTestV0(manifestRef, capabilityRef, toolRef string, locales ...string) toolcapability.CapabilityManifestV0 {
	return toolcapability.CapabilityManifestV0{
		SchemaVersion:    toolcapability.CapabilityManifestSchemaV0,
		ManifestRef:      manifestRef,
		CapabilityRef:    capabilityRef,
		ToolRef:          toolRef,
		Version:          "1.0.0",
		InputSchemaRef:   "input-schema-ref",
		OutputSchemaRef:  "output-schema-ref",
		Locales:          locales,
		EffectProfile:    toolcapability.CapabilityEffectProfileV0{ProfileRef: "effect-profile-ref"},
		Idempotency:      toolcapability.CapabilityIdempotencyV0{KeySchemaRef: "idempotency-key-schema-ref", ScopeRef: "idempotency-scope-ref"},
		ConfigSchemaRef:  "config-schema-ref",
		TestRefs:         []string{"test-ref"},
		ReceiptSchemaRef: "receipt-schema-ref",
		IntegrationModes: []string{toolcapability.IntegrationModeEmbeddedModuleV0},
	}
}
