package orquestapresentationextraction

import document "orquesta/modulos/orquesta-document-extraction"

const PresentationExtractionContractSchemaVersionV0 = "presentation_extraction_contract.v0"

type PresentationFormatV0 string

const (
	PresentationFormatPPTXV0 PresentationFormatV0 = "pptx"
	PresentationFormatODPV0  PresentationFormatV0 = "odp"
)

type PresentationAdapterIdentityV0 struct {
	AdapterRef string `json:"adapter_ref"`
	Version    string `json:"version"`
}

type PresentationSourceMaterialV0 struct {
	PresentationRef string               `json:"presentation_ref"`
	SourceRef       string               `json:"source_ref"`
	Format          PresentationFormatV0 `json:"format"`
	ContentHash     string               `json:"content_hash"`
	SnapshotRef     string               `json:"snapshot_ref"`
}

type PresentationExtractionRequestV0 struct {
	PresentationRef   string `json:"presentation_ref"`
	ConfigurationRef  string `json:"configuration_ref"`
	ConfigurationHash string `json:"configuration_hash"`
}

type PresentationExtractionReceiptV0 struct {
	SchemaVersion     string                          `json:"schema_version"`
	PresentationRef   string                          `json:"presentation_ref"`
	SourceRef         string                          `json:"source_ref"`
	Format            PresentationFormatV0            `json:"format"`
	ContentHash       string                          `json:"content_hash"`
	SnapshotRef       string                          `json:"snapshot_ref"`
	ConfigurationRef  string                          `json:"configuration_ref"`
	ConfigurationHash string                          `json:"configuration_hash"`
	SlidePageRefs     []string                        `json:"slide_page_refs"`
	Adapters          []PresentationAdapterIdentityV0 `json:"adapters"`
}

type PresentationExtractionResultV0 struct {
	Document document.DocumentV0             `json:"document"`
	Receipt  PresentationExtractionReceiptV0 `json:"receipt"`
}
