package orquestapresentationextraction

import (
	"context"

	document "orquesta/modulos/orquesta-document-extraction"
)

type PresentationSourcePortV0 interface {
	AdapterIdentityV0() PresentationAdapterIdentityV0
	ResolvePresentationV0(context.Context, string) (PresentationSourceMaterialV0, error)
}

// PresentationDocumentProjectorPortV0 is an external adapter for PPTX/ODP.
// It returns the common document IR: one slide is one ordered page.
type PresentationDocumentProjectorPortV0 interface {
	AdapterIdentityV0() PresentationAdapterIdentityV0
	ProjectPresentationDocumentV0(context.Context, PresentationSourceMaterialV0) (document.DocumentV0, error)
}

type PresentationExtractionPortsV0 struct {
	Source    PresentationSourcePortV0
	Projector PresentationDocumentProjectorPortV0
}
