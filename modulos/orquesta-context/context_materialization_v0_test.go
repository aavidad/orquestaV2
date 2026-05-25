package orquestacontext

import (
	"encoding/json"
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
	contract := materializedEntryBySourceV0(materialized, "ProcessRuntimeConnectorV0")
	if contract.RefOnlyReason != ContextRefOnlyReasonByDesignV0 ||
		contract.RequiredRefAction != ContextRequiredRefActionAckEvidenceV0 {
		t.Fatalf("contract ref_only sin clasificacion: %+v", contract)
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

func TestMaterializeContextBundleWithSanitizerV0ConservaEvidenciaSinDatoSensible(t *testing.T) {
	root := createContextMaterializationRepoV0(t)
	writeContextFileV0(t, root, "modulos/orquesta-runtime/AGENTS.md", "token=sk-test-secret /home/alberto/private")
	store := newFileContextStoreForTestV0(t, root)
	bundle := BuildContextBundleV0(validContextBundleRequestV0())

	materialized := MaterializeContextBundleWithSanitizerV0(bundle, store, fakeContextSanitizerV0{})
	if !materialized.Valid() {
		t.Fatalf("materialized invalid: %+v", materialized.Issues)
	}
	if len(materialized.SanitizationEvidence) == 0 {
		t.Fatalf("sanitization evidence missing: %+v", materialized)
	}
	raw, err := json.Marshal(materialized)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"sk-test-secret", "/home/alberto", "token="} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("dato sensible persistido %q: %s", forbidden, string(raw))
		}
	}
}

func TestSanitizeMaterializedContextBundleV0DudaYMinimizaPorRefs(t *testing.T) {
	bundle := orquestaContextMaterializedForSanitizerTestV0("-----BEGIN PRIVATE KEY-----")

	materialized := SanitizeMaterializedContextBundleV0(bundle, reviewContextSanitizerV0{})
	if !materialized.Valid() {
		t.Fatalf("materialized invalid: %+v", materialized.Issues)
	}
	entry := materialized.Entries[0]
	if entry.Mode != ContextMaterializationModeRefOnlyV0 || entry.Content != "" || !entry.Truncated {
		t.Fatalf("expected minimal ref-only context, got %+v", entry)
	}
	if !ContextBundleRequiresSanitizationReviewV0(materialized) ||
		!strings.Contains(materialized.DirectorQuestionHint, "CONSULTA_AL_DIRECTOR") {
		t.Fatalf("review hint missing: %+v", materialized)
	}
	if entry.RefOnlyReason != ContextRefOnlyReasonSanitizationReviewV0 ||
		entry.RequiredRefAction != ContextRequiredRefActionAskDirectorV0 {
		t.Fatalf("review ref_only sin accion requerida: %+v", entry)
	}
}

func TestContextRequiredRefOnlyEntriesV0DistingueMaterializacionPendiente(t *testing.T) {
	entry := contextMaterializedRefOnlyV0(ContextBundleEntryV0{
		EntryRef:  "entry-ref-required-doc",
		Layer:     ContextLayerTaskContextV0,
		Kind:      ContextEntryDocRefV0,
		SourceRef: "docs/autoprogramacion_orquesta_pendientes_2026-05-23.md",
		Required:  true,
	})
	bundle := ContextMaterializedBundleV0{
		SchemaVersion: ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-required-doc",
		WorkOrderRef:  "task-ref-required-doc",
		TargetModule:  "orquesta-context",
		Entries:       []ContextMaterializedEntryV0{entry},
	}

	refs := ContextRequiredRefOnlyEntriesV0(bundle)
	if len(refs) != 1 ||
		refs[0].RefOnlyReason != ContextRefOnlyReasonMaterializationMissingV0 ||
		refs[0].RequiredRefAction != ContextRequiredRefActionReadLocalV0 {
		t.Fatalf("required ref_only mal clasificado: %+v", refs)
	}
}

func TestSanitizeMaterializedContextBundleV0NoPersisteSourceRefSensible(t *testing.T) {
	bundle := orquestaContextMaterializedForSanitizerTestV0("contenido publico")
	bundle.Entries[0].SourceRef = "/home/alberto/private/token=sk-secret-local"

	materialized := SanitizeMaterializedContextBundleV0(bundle, fakeContextSanitizerV0{})
	if !materialized.Valid() {
		t.Fatalf("materialized invalid: %+v", materialized.Issues)
	}
	raw, err := json.Marshal(materialized)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	lower := strings.ToLower(string(raw))
	for _, forbidden := range []string{"/home/alberto", "sk-secret-local", "token="} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("metadata sensible persistida %q: %s", forbidden, string(raw))
		}
	}
}

func TestSanitizeMaterializedContextBundleV0NoPersisteEntryRefSensible(t *testing.T) {
	bundle := orquestaContextMaterializedForSanitizerTestV0("contenido publico")
	bundle.Entries[0].EntryRef = "entry-ref-access_token=sk-secret-local"

	materialized := SanitizeMaterializedContextBundleV0(bundle, fakeContextSanitizerV0{})
	if len(materialized.SanitizationEvidence) == 0 {
		t.Fatalf("sanitization evidence missing: %+v", materialized)
	}
	raw, err := json.Marshal(materialized)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{"access_token", "sk-secret-local", "token="} {
		if strings.Contains(strings.ToLower(string(raw)), forbidden) {
			t.Fatalf("sensitive entry ref fragment leaked %q: %s", forbidden, string(raw))
		}
	}
}

func TestSanitizeMaterializedContextBundleV0PermiteRefsDePoliticaOpaca(t *testing.T) {
	bundle := orquestaContextMaterializedForSanitizerTestV0("contenido publico")
	bundle.Entries[0].EntryRef = "entry-ref-prompt-policy"
	bundle.Entries[0].SourceRef = "modulos/orquesta-context/transcript-policy.md"

	materialized := SanitizeMaterializedContextBundleV0(bundle, fakeContextSanitizerV0{})
	if !materialized.Valid() {
		t.Fatalf("materialized invalid: %+v", materialized.Issues)
	}
	raw, err := json.Marshal(materialized)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	serialized := string(raw)
	for _, want := range []string{"entry-ref-prompt-policy", "transcript-policy.md"} {
		if !strings.Contains(serialized, want) {
			t.Fatalf("ref opaca no preservada %q: %s", want, serialized)
		}
	}
}

type fakeContextSanitizerV0 struct{}

func (fakeContextSanitizerV0) SanitizeContextEntryV0(
	request ContextSanitizationRequestV0,
) ContextSanitizationResultV0 {
	return ContextSanitizationResultV0{
		Status:  ContextSanitizationStatusSanitizedV0,
		Content: "sanitized-ref-context-entry-001",
		Evidence: ContextSanitizationEvidenceV0{
			SanitizerRef:     "sanitizer-ref-test",
			Categories:       []string{"credential_value", "private_path"},
			ReplacementCount: 2,
		},
	}
}

type reviewContextSanitizerV0 struct{}

func (reviewContextSanitizerV0) SanitizeContextEntryV0(
	request ContextSanitizationRequestV0,
) ContextSanitizationResultV0 {
	return ContextSanitizationResultV0{
		Status: ContextSanitizationStatusReviewRequiredV0,
		Evidence: ContextSanitizationEvidenceV0{
			SanitizerRef:   "sanitizer-ref-review",
			Categories:     []string{"non_public_context"},
			ReviewRequired: true,
		},
	}
}

func orquestaContextMaterializedForSanitizerTestV0(content string) ContextMaterializedBundleV0 {
	return ContextMaterializedBundleV0{
		SchemaVersion: ContextMaterializedBundleSchemaVersionV0,
		BundleRef:     "bundle-ref-sanitizer-001",
		WorkOrderRef:  "task-ref-sanitizer-001",
		TargetModule:  "orquesta-context",
		TotalBytes:    len(content),
		Entries: []ContextMaterializedEntryV0{{
			EntryRef:  "entry-ref-sanitizer-001",
			Layer:     ContextLayerTaskContextV0,
			Kind:      ContextEntryDocRefV0,
			SourceRef: "modulos/orquesta-context/AGENTS.md",
			Mode:      ContextMaterializationModeContentV0,
			Content:   content,
			Bytes:     len(content),
			Required:  true,
		}},
	}
}
