package orquestacontext

import (
	"strings"
	"testing"
)

func TestMaterializeContextBundleV0ConFilesystemExplicito(t *testing.T) {
	root := createContextMaterializationRepoV0(t)
	store := newFileContextStoreForTestV0(t, root)
	bundle := BuildContextBundleV0(validContextBundleRequestV0())

	materialized := MaterializeContextBundleV0(bundle, store)
	if !materialized.Valid() {
		t.Fatalf("materialized invalid: %+v", materialized.Issues)
	}
	requireMaterializedModeV0(t, materialized, "modulos/orquesta-runtime/AGENTS.md", ContextMaterializationModeContentV0)
	requireMaterializedModeV0(t, materialized, "process_runtime_connector_types_v0.go", ContextMaterializationModeContentV0)
	requireMaterializedModeV0(t, materialized, "process_runtime_connector_v0.go", ContextMaterializationModeRefOnlyV0)
	requireMaterializedModeV0(t, materialized, "ProcessRuntimeConnectorV0", ContextMaterializationModeRefOnlyV0)
	if materialized.TotalBytes <= 0 || materialized.TotalBytes > bundle.Limits.MaxTotalBytes {
		t.Fatalf("total_bytes=%d limits=%+v", materialized.TotalBytes, bundle.Limits)
	}
	if !strings.Contains(materialized.DirectorQuestionHint, "CONSULTA_AL_DIRECTOR") {
		t.Fatalf("director hint missing: %q", materialized.DirectorQuestionHint)
	}
}

func TestMaterializeContextBundleV0TruncaEntradasGrandes(t *testing.T) {
	root := createContextMaterializationRepoV0(t)
	writeContextFileV0(t, root, "modulos/orquesta-runtime/AGENTS.md", strings.Repeat("regla\n", 900))
	store := newFileContextStoreForTestV0(t, root)
	bundle := BuildContextBundleV0(validContextBundleRequestV0())

	materialized := MaterializeContextBundleV0(bundle, store)
	if !materialized.Valid() {
		t.Fatalf("materialized invalid: %+v", materialized.Issues)
	}
	entry := materializedEntryBySourceV0(materialized, "modulos/orquesta-runtime/AGENTS.md")
	if !entry.Truncated || entry.Bytes != defaultContextBundleEntryBytesV0 {
		t.Fatalf("expected truncated AGENTS entry, got %+v", entry)
	}
}

func TestMaterializeContextBundleV0RechazaReaderAusente(t *testing.T) {
	bundle := BuildContextBundleV0(validContextBundleRequestV0())
	materialized := MaterializeContextBundleV0(bundle, nil)

	requireMaterializationIssueV0(t, materialized.Issues, ErrContextMaterializationReaderRequeridoV0)
}
