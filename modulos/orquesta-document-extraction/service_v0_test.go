package orquestadocumentextraction_test

import (
	"context"
	"testing"

	orquestadocumentextraction "orquesta/modulos/orquesta-document-extraction"
	orquestadocumentextractionfake "orquesta/modulos/orquesta-document-extraction-fake"
)

func TestExtractDocumentV0AcceptsEvidencedFieldsAndRecordsReproducibleReceipt(t *testing.T) {
	request, ports, exporter, receiptStore := documentExtractionFixtureV0(true)
	result, err := orquestadocumentextraction.ExtractDocumentV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("ExtractDocumentV0: %v", err)
	}
	if len(result.AcceptedFields) != 1 || result.AcceptedFields[0].ValueRaw != "Ana Example" {
		t.Fatalf("accepted fields = %#v", result.AcceptedFields)
	}
	if len(result.AcceptedFields[0].EvidenceRefs) != 1 || result.AcceptedFields[0].EvidenceRefs[0] != "evidence:1" {
		t.Fatalf("candidate evidence = %#v", result.AcceptedFields[0].EvidenceRefs)
	}
	if exporter.CallCountV0() != 1 {
		t.Fatalf("export calls = %d, want 1", exporter.CallCountV0())
	}
	if result.Receipt.ReceiptRef == "" || result.Receipt.ContentHash != "sha256:source" || result.Receipt.ConfigurationHash != "sha256:config" || result.Receipt.NormalizationLocale != "es-ES" || result.Receipt.OutputLocale != "en-GB" {
		t.Fatalf("receipt = %#v", result.Receipt)
	}
	if result.Receipt.Status != "completed" || len(result.Receipt.Adapters) != 9 {
		t.Fatalf("receipt status/adapters = %#v", result.Receipt)
	}
	stored, ok := receiptStore.LastReceiptV0()
	if !ok || stored.ConfigurationRef != "config:local:v1" {
		t.Fatalf("stored receipt = %#v", stored)
	}
}

func TestExtractDocumentV0PreservesCandidateButDoesNotExportWithoutEvidence(t *testing.T) {
	request, ports, exporter, _ := documentExtractionFixtureV0(false)
	result, err := orquestadocumentextraction.ExtractDocumentV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("ExtractDocumentV0: %v", err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].ValueRaw != "Ana Example" {
		t.Fatalf("candidates = %#v", result.Candidates)
	}
	if len(result.AcceptedFields) != 0 || exporter.CallCountV0() != 0 {
		t.Fatalf("accepted=%d exports=%d", len(result.AcceptedFields), exporter.CallCountV0())
	}
	if len(result.Issues) != 1 || result.Issues[0].Code != "document_candidate_evidence_required" {
		t.Fatalf("issues = %#v", result.Issues)
	}
	if result.Receipt.Status != "completed_with_issues" {
		t.Fatalf("receipt status = %q", result.Receipt.Status)
	}
}

func TestExtractDocumentV0RejectsEvidenceWithoutPageAndSpatialAnchor(t *testing.T) {
	request, ports, exporter, _ := documentExtractionFixtureV0(true)
	ports.EvidenceLocator = orquestadocumentextractionfake.StaticDocumentEvidenceLocatorV0{
		Identity: orquestadocumentextraction.DocumentAdapterIdentityV0{AdapterRef: "evidence", Version: "1"},
		EvidenceByCandidate: map[string][]orquestadocumentextraction.DocumentEvidenceV0{
			"candidate:1": {{EvidenceRef: "evidence:invalid", DocumentRef: "document:1"}},
		},
	}
	result, err := orquestadocumentextraction.ExtractDocumentV0(context.Background(), request, ports)
	if err != nil {
		t.Fatalf("ExtractDocumentV0: %v", err)
	}
	if len(result.AcceptedFields) != 0 || exporter.CallCountV0() != 0 || len(result.Issues) != 1 || result.Issues[0].Code != "document_candidate_evidence_required" {
		t.Fatalf("result = %#v, exports=%d", result, exporter.CallCountV0())
	}
}

func TestDocumentSchemasRemainGenericThroughOpaqueFieldRefs(t *testing.T) {
	person, err := orquestadocumentextraction.NewPersonDocumentSchemaV0(orquestadocumentextraction.DocumentSchemaV0{SchemaRef: "schema:person", DomainRef: "domain:external", EntityKind: "person", SchemaVersion: "1", Fields: []orquestadocumentextraction.DocumentSchemaFieldV0{{FieldRef: "field:external-person-value", DataType: orquestadocumentextraction.DocumentFieldDataTypeStringV0}}})
	if err != nil || person.Schema.Fields[0].FieldRef != "field:external-person-value" {
		t.Fatalf("person schema = %#v, %v", person, err)
	}
	invoice, err := orquestadocumentextraction.NewInvoiceDocumentSchemaV0(orquestadocumentextraction.DocumentSchemaV0{SchemaRef: "schema:invoice", DomainRef: "domain:external", EntityKind: "invoice", SchemaVersion: "1", Fields: []orquestadocumentextraction.DocumentSchemaFieldV0{{FieldRef: "field:external-invoice-value", DataType: orquestadocumentextraction.DocumentFieldDataTypeCurrencyV0}}})
	if err != nil || invoice.Schema.Fields[0].FieldRef != "field:external-invoice-value" {
		t.Fatalf("invoice schema = %#v, %v", invoice, err)
	}
}

func TestDocumentCloudPolicyRequiresExplicitPrivacyControls(t *testing.T) {
	err := orquestadocumentextraction.ValidateDocumentExtractionPolicyV0(orquestadocumentextraction.DocumentExtractionPolicyV0{DataHandlingMode: orquestadocumentextraction.DocumentDataHandlingCloudV0, CloudOptIn: true, PolicyRef: "policy:cloud"})
	if err == nil {
		t.Fatal("cloud policy without privacy controls accepted")
	}
	err = orquestadocumentextraction.ValidateDocumentExtractionPolicyV0(orquestadocumentextraction.DocumentExtractionPolicyV0{DataHandlingMode: orquestadocumentextraction.DocumentDataHandlingCloudV0, CloudOptIn: true, PolicyRef: "policy:cloud", RegionRef: "region:eu", RetentionRef: "retention:1", DeletionRef: "deletion:1", EncryptionRef: "encryption:1", AuditRef: "audit:1"})
	if err != nil {
		t.Fatalf("complete cloud policy: %v", err)
	}
}

func documentExtractionFixtureV0(withEvidence bool) (orquestadocumentextraction.DocumentExtractionRequestV0, orquestadocumentextraction.DocumentExtractionPortsV0, *orquestadocumentextractionfake.RecordingDocumentExporterV0, *orquestadocumentextractionfake.InMemoryDocumentReceiptStoreV0) {
	identity := func(ref string) orquestadocumentextraction.DocumentAdapterIdentityV0 {
		return orquestadocumentextraction.DocumentAdapterIdentityV0{AdapterRef: ref, Version: "1", BuildHash: "sha256:" + ref}
	}
	document := orquestadocumentextraction.DocumentV0{IRVersion: orquestadocumentextraction.DocumentExtractionIRSchemaVersionV0, DocumentRef: "document:1", SourceRef: "source:1", ContentHash: "sha256:source", MediaKind: "application/pdf", PageCount: 1, Pages: []orquestadocumentextraction.DocumentPageV0{{PageRef: "page:1", Index: 0, Width: 100, Height: 100, Blocks: []orquestadocumentextraction.DocumentBlockV0{{BlockRef: "block:1", Kind: "paragraph", Spans: []orquestadocumentextraction.DocumentSpanV0{{SpanRef: "span:1", TextRaw: "Ana Example"}}}}}}}
	request := orquestadocumentextraction.DocumentExtractionRequestV0{DocumentRef: "document:1", Schema: orquestadocumentextraction.DocumentSchemaV0{SchemaRef: "schema:person", DomainRef: "domain:external", EntityKind: "person", SchemaVersion: "1", Fields: []orquestadocumentextraction.DocumentSchemaFieldV0{{FieldRef: "field:external-person-value", DataType: orquestadocumentextraction.DocumentFieldDataTypeStringV0}}}, ConfigurationRef: "config:local:v1", ConfigurationHash: "sha256:config", DocumentLanguage: "es", NormalizationLocale: "es-ES", OutputLocale: "en-GB"}
	candidate := orquestadocumentextraction.DocumentFieldCandidateV0{CandidateRef: "candidate:1", FieldRef: "field:external-person-value", OccurrenceRef: "occurrence:1", ValueRaw: "Ana Example", ValueState: orquestadocumentextraction.DocumentValueStatePresentV0, DataTypeHint: orquestadocumentextraction.DocumentFieldDataTypeStringV0}
	evidence := map[string][]orquestadocumentextraction.DocumentEvidenceV0{}
	if withEvidence {
		evidence[candidate.CandidateRef] = []orquestadocumentextraction.DocumentEvidenceV0{{EvidenceRef: "evidence:1", DocumentRef: "document:1", PageRef: "page:1", BlockRef: "block:1", SpanRefs: []string{"span:1"}}}
	}
	exporter := &orquestadocumentextractionfake.RecordingDocumentExporterV0{Identity: identity("exporter"), Artifact: orquestadocumentextraction.DocumentExportArtifactV0{ArtifactRef: "artifact:1", DestinationRef: "destination:1", Format: "test", ContentHash: "sha256:artifact"}}
	receiptStore := &orquestadocumentextractionfake.InMemoryDocumentReceiptStoreV0{Identity: identity("receipt")}
	ports := orquestadocumentextraction.DocumentExtractionPortsV0{Source: orquestadocumentextractionfake.StaticDocumentSourceV0{Identity: identity("source"), Material: orquestadocumentextraction.DocumentSourceMaterialV0{DocumentRef: "document:1", SourceRef: "source:1", ContentHash: "sha256:source", MediaKind: "application/pdf"}}, Normalizer: orquestadocumentextractionfake.PassthroughDocumentNormalizerV0{Identity: identity("normalizer")}, Parser: orquestadocumentextractionfake.StaticDocumentParserV0{Identity: identity("parser"), Document: document}, Localizer: orquestadocumentextractionfake.StaticDocumentLocalizerV0{Identity: identity("localizer"), Locations: []orquestadocumentextraction.DocumentLocationV0{{LocationRef: "location:1", PageRef: "page:1", BlockRefs: []string{"block:1"}}}}, SchemaExtractor: orquestadocumentextractionfake.StaticDocumentSchemaExtractorV0{Identity: identity("extractor"), CandidatesBySlice: map[string][]orquestadocumentextraction.DocumentFieldCandidateV0{"schema:person:location:0:fields:0": {candidate}}}, EvidenceLocator: orquestadocumentextractionfake.StaticDocumentEvidenceLocatorV0{Identity: identity("evidence"), EvidenceByCandidate: evidence}, Validator: orquestadocumentextractionfake.AcceptingDocumentValidatorV0{Identity: identity("validator")}, Exporters: []orquestadocumentextraction.DocumentExporterPortV0{exporter}, ReceiptStore: receiptStore}
	return request, ports, exporter, receiptStore
}
