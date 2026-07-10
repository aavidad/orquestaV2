package orquestapresentationextraction

import (
	"context"
	"testing"

	document "orquesta/modulos/orquesta-document-extraction"
)

func TestExtractPresentationV0ProjectsOrderedSlidesToDocumentIR(t *testing.T) {
	identity := PresentationAdapterIdentityV0{AdapterRef: "fake", Version: "1"}
	result, err := ExtractPresentationV0(context.Background(), PresentationExtractionRequestV0{PresentationRef: "presentation:1", ConfigurationRef: "config:1", ConfigurationHash: "sha256:config"}, PresentationExtractionPortsV0{
		Source:    presentationSourceFakeV0{identity: identity, source: PresentationSourceMaterialV0{PresentationRef: "presentation:1", SourceRef: "source:1", Format: PresentationFormatODPV0, ContentHash: "sha256:presentation", SnapshotRef: "snapshot:1"}},
		Projector: presentationProjectorFakeV0{identity: identity, doc: document.DocumentV0{IRVersion: document.DocumentExtractionIRSchemaVersionV0, DocumentRef: "presentation:1", SourceRef: "source:1", ContentHash: "sha256:presentation", MediaKind: "presentation", PageCount: 2, Pages: []document.DocumentPageV0{{PageRef: "slide:1", Index: 0}, {PageRef: "slide:2", Index: 1}}}},
	})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(result.Receipt.SlidePageRefs) != 2 || result.Receipt.Format != PresentationFormatODPV0 {
		t.Fatalf("receipt=%+v", result.Receipt)
	}
}

func TestExtractPresentationV0RejectsUnorderedSlides(t *testing.T) {
	identity := PresentationAdapterIdentityV0{AdapterRef: "fake", Version: "1"}
	_, err := ExtractPresentationV0(context.Background(), PresentationExtractionRequestV0{PresentationRef: "presentation:1", ConfigurationRef: "config:1", ConfigurationHash: "hash"}, PresentationExtractionPortsV0{Source: presentationSourceFakeV0{identity: identity, source: PresentationSourceMaterialV0{PresentationRef: "presentation:1", SourceRef: "source:1", Format: PresentationFormatPPTXV0, ContentHash: "hash", SnapshotRef: "snapshot:1"}}, Projector: presentationProjectorFakeV0{identity: identity, doc: document.DocumentV0{IRVersion: document.DocumentExtractionIRSchemaVersionV0, DocumentRef: "presentation:1", ContentHash: "hash", MediaKind: "presentation", PageCount: 1, Pages: []document.DocumentPageV0{{PageRef: "slide:1", Index: 2}}}}})
	if err == nil || err.Error() != "presentation_slide_order_invalid" {
		t.Fatalf("err=%v", err)
	}
}

type presentationSourceFakeV0 struct {
	identity PresentationAdapterIdentityV0
	source   PresentationSourceMaterialV0
}

func (fake presentationSourceFakeV0) AdapterIdentityV0() PresentationAdapterIdentityV0 {
	return fake.identity
}
func (fake presentationSourceFakeV0) ResolvePresentationV0(context.Context, string) (PresentationSourceMaterialV0, error) {
	return fake.source, nil
}

type presentationProjectorFakeV0 struct {
	identity PresentationAdapterIdentityV0
	doc      document.DocumentV0
}

func (fake presentationProjectorFakeV0) AdapterIdentityV0() PresentationAdapterIdentityV0 {
	return fake.identity
}
func (fake presentationProjectorFakeV0) ProjectPresentationDocumentV0(context.Context, PresentationSourceMaterialV0) (document.DocumentV0, error) {
	return fake.doc, nil
}
