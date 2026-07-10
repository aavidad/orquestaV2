package orquestadocumentextraction

import "context"

type DocumentSourcePortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	ResolveDocumentV0(context.Context, string) (DocumentSourceMaterialV0, error)
}

type DocumentNormalizerPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	NormalizeDocumentV0(context.Context, DocumentSourceMaterialV0, DocumentExtractionPolicyV0) (DocumentNormalizedMaterialV0, error)
}

type DocumentParserPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	ParseDocumentV0(context.Context, DocumentNormalizedMaterialV0) (DocumentV0, error)
}

type DocumentLocalizerPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	LocateDocumentSchemaV0(context.Context, DocumentV0, DocumentSchemaV0) ([]DocumentLocationV0, error)
}

type DocumentSchemaExtractorPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	ExtractDocumentSchemaSliceV0(context.Context, DocumentV0, DocumentSchemaSliceV0) ([]DocumentFieldCandidateV0, error)
}

type DocumentEvidenceLocatorPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	LocateDocumentEvidenceV0(context.Context, DocumentV0, DocumentFieldCandidateV0) ([]DocumentEvidenceV0, error)
}

type DocumentValidatorPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	ValidateDocumentFieldV0(context.Context, DocumentSchemaFieldV0, DocumentFieldCandidateV0, string) (DocumentValidationResultV0, error)
}

type DocumentHumanReviewPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	ReviewDocumentFieldV0(context.Context, DocumentSchemaFieldV0, DocumentFieldCandidateV0) (DocumentHumanReviewResultV0, error)
}

type DocumentExporterPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	ExportDocumentFieldsV0(context.Context, DocumentExportRequestV0) (DocumentExportArtifactV0, error)
}

type DocumentReceiptStorePortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	StoreDocumentExtractionReceiptV0(context.Context, DocumentExtractionReceiptV0) (string, error)
}

type DocumentToolOperationStorePortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	SubmitDocumentToolCommandV0(context.Context, DocumentToolCommandV0) (DocumentToolOperationV0, error)
	LoadDocumentToolOperationV0(context.Context, string) (DocumentToolOperationV0, error)
}

type DocumentToolCapabilityRegistryPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	DiscoverDocumentToolCapabilitiesV0(context.Context, DocumentToolDiscoveryRequestV0) ([]DocumentToolCapabilityDescriptorV0, error)
}

type DocumentToolBundleExporterPortV0 interface {
	AdapterIdentityV0() DocumentAdapterIdentityV0
	ExportDocumentExtractionBundleV0(context.Context, DocumentExtractionBundleDescriptorV0) (DocumentExtractionBundleDescriptorV0, error)
}

type DocumentExtractionPortsV0 struct {
	Source          DocumentSourcePortV0
	Normalizer      DocumentNormalizerPortV0
	Parser          DocumentParserPortV0
	Localizer       DocumentLocalizerPortV0
	SchemaExtractor DocumentSchemaExtractorPortV0
	EvidenceLocator DocumentEvidenceLocatorPortV0
	Validator       DocumentValidatorPortV0
	HumanReview     DocumentHumanReviewPortV0
	Exporters       []DocumentExporterPortV0
	ReceiptStore    DocumentReceiptStorePortV0
}

type DocumentExtractionRequestV0 struct {
	DocumentRef         string                     `json:"document_ref"`
	Schema              DocumentSchemaV0           `json:"schema"`
	Policy              DocumentExtractionPolicyV0 `json:"policy"`
	ConfigurationRef    string                     `json:"configuration_ref"`
	ConfigurationHash   string                     `json:"configuration_hash"`
	DocumentLanguage    string                     `json:"document_language,omitempty"`
	NormalizationLocale string                     `json:"normalization_locale"`
	OutputLocale        string                     `json:"output_locale"`
	MaxFieldsPerSlice   int                        `json:"max_fields_per_slice,omitempty"`
	RequireHumanReview  bool                       `json:"require_human_review"`
}

type DocumentExportRequestV0 struct {
	DocumentRef    string                     `json:"document_ref"`
	SchemaRef      string                     `json:"schema_ref"`
	DestinationRef string                     `json:"destination_ref"`
	OutputLocale   string                     `json:"output_locale"`
	Fields         []DocumentFieldCandidateV0 `json:"fields"`
	Evidence       []DocumentEvidenceV0       `json:"evidence"`
}

type DocumentExtractionResultV0 struct {
	Document       DocumentV0                  `json:"document"`
	SchemaSlices   []DocumentSchemaSliceV0     `json:"schema_slices"`
	Candidates     []DocumentFieldCandidateV0  `json:"candidates"`
	AcceptedFields []DocumentFieldCandidateV0  `json:"accepted_fields"`
	Evidence       []DocumentEvidenceV0        `json:"evidence"`
	Artifacts      []DocumentExportArtifactV0  `json:"artifacts"`
	Receipt        DocumentExtractionReceiptV0 `json:"receipt"`
	Issues         []DocumentExtractionIssueV0 `json:"issues,omitempty"`
}

type DocumentToolDiscoveryRequestV0 struct {
	AttachModes          []DocumentToolAttachModeV0 `json:"attach_modes,omitempty"`
	RequiredCapabilities []string                   `json:"required_capabilities,omitempty"`
	Policy               DocumentExtractionPolicyV0 `json:"policy"`
}
