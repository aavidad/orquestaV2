package orquestapresentationextraction

import (
	"context"
	"fmt"
	"strings"
)

func ExtractPresentationV0(ctx context.Context, request PresentationExtractionRequestV0, ports PresentationExtractionPortsV0) (PresentationExtractionResultV0, error) {
	if blankPresentationValueV0(request.PresentationRef) || blankPresentationValueV0(request.ConfigurationRef) || blankPresentationValueV0(request.ConfigurationHash) {
		return PresentationExtractionResultV0{}, fmt.Errorf("presentation_extraction_request_invalid")
	}
	if ports.Source == nil || ports.Projector == nil {
		return PresentationExtractionResultV0{}, fmt.Errorf("presentation_extraction_port_required")
	}
	source, err := ports.Source.ResolvePresentationV0(ctx, request.PresentationRef)
	if err != nil {
		return PresentationExtractionResultV0{}, err
	}
	if source.PresentationRef != request.PresentationRef || !validPresentationFormatV0(source.Format) || blankPresentationValueV0(source.SourceRef) || blankPresentationValueV0(source.ContentHash) || blankPresentationValueV0(source.SnapshotRef) {
		return PresentationExtractionResultV0{}, fmt.Errorf("presentation_source_material_invalid")
	}
	doc, err := ports.Projector.ProjectPresentationDocumentV0(ctx, source)
	if err != nil {
		return PresentationExtractionResultV0{}, err
	}
	if doc.IRVersion != "document_extraction_ir.v0" || doc.DocumentRef != source.PresentationRef || doc.ContentHash != source.ContentHash || doc.PageCount == 0 || doc.PageCount != len(doc.Pages) || doc.MediaKind != "presentation" {
		return PresentationExtractionResultV0{}, fmt.Errorf("presentation_document_projection_invalid")
	}
	receipt := PresentationExtractionReceiptV0{SchemaVersion: PresentationExtractionContractSchemaVersionV0, PresentationRef: source.PresentationRef, SourceRef: source.SourceRef, Format: source.Format, ContentHash: source.ContentHash, SnapshotRef: source.SnapshotRef, ConfigurationRef: request.ConfigurationRef, ConfigurationHash: request.ConfigurationHash, Adapters: []PresentationAdapterIdentityV0{ports.Source.AdapterIdentityV0(), ports.Projector.AdapterIdentityV0()}}
	for index, page := range doc.Pages {
		if page.PageRef == "" || page.Index != index {
			return PresentationExtractionResultV0{}, fmt.Errorf("presentation_slide_order_invalid")
		}
		receipt.SlidePageRefs = append(receipt.SlidePageRefs, page.PageRef)
	}
	return PresentationExtractionResultV0{Document: doc, Receipt: receipt}, nil
}

func validPresentationFormatV0(format PresentationFormatV0) bool {
	return format == PresentationFormatPPTXV0 || format == PresentationFormatODPV0
}
func blankPresentationValueV0(value string) bool { return strings.TrimSpace(value) == "" }
