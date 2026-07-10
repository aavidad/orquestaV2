package orquestadocumentextraction

const (
	DocumentExtractionIRSchemaVersionV0       = "document_extraction_ir.v0"
	DocumentExtractionContractSchemaVersionV0 = "document_extraction_contract.v0"
	DocumentExtractionReceiptSchemaVersionV0  = "document_extraction_receipt.v0"
)

type DocumentValueStateV0 string

const (
	DocumentValueStatePresentV0        DocumentValueStateV0 = "present"
	DocumentValueStateNullExplicitV0   DocumentValueStateV0 = "null_explicit"
	DocumentValueStateAbsentV0         DocumentValueStateV0 = "absent"
	DocumentValueStateUnreadableV0     DocumentValueStateV0 = "unreadable"
	DocumentValueStateNotApplicableV0  DocumentValueStateV0 = "not_applicable"
	DocumentValueStateProposedV0       DocumentValueStateV0 = "proposed"
	DocumentValueStateHumanCorrectedV0 DocumentValueStateV0 = "human_corrected"
)

type DocumentDataHandlingModeV0 string

const (
	DocumentDataHandlingLocalV0 DocumentDataHandlingModeV0 = "local"
	DocumentDataHandlingCloudV0 DocumentDataHandlingModeV0 = "cloud"
)

type DocumentFieldDataTypeV0 string

const (
	DocumentFieldDataTypeStringV0   DocumentFieldDataTypeV0 = "string"
	DocumentFieldDataTypeNumberV0   DocumentFieldDataTypeV0 = "number"
	DocumentFieldDataTypeBooleanV0  DocumentFieldDataTypeV0 = "boolean"
	DocumentFieldDataTypeDateV0     DocumentFieldDataTypeV0 = "date"
	DocumentFieldDataTypeCurrencyV0 DocumentFieldDataTypeV0 = "currency"
	DocumentFieldDataTypeObjectV0   DocumentFieldDataTypeV0 = "object"
	DocumentFieldDataTypeArrayV0    DocumentFieldDataTypeV0 = "array"
)

type DocumentLanguageHintV0 struct {
	LanguageCode string   `json:"language_code"`
	Confidence   *float64 `json:"confidence,omitempty"`
}

type DocumentBoundingBoxV0 struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type DocumentPointV0 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type DocumentProvenanceV0 struct {
	SourceRefs     []string `json:"source_refs,omitempty"`
	AdapterRef     string   `json:"adapter_ref,omitempty"`
	AdapterVersion string   `json:"adapter_version,omitempty"`
	TransformRefs  []string `json:"transform_refs,omitempty"`
}

type DocumentTransformV0 struct {
	TransformRef string `json:"transform_ref"`
	Kind         string `json:"kind"`
	Version      string `json:"version"`
	ConfigHash   string `json:"config_hash"`
	InputHash    string `json:"input_hash"`
	OutputHash   string `json:"output_hash"`
}

type DocumentV0 struct {
	IRVersion     string                   `json:"ir_version"`
	DocumentRef   string                   `json:"document_ref"`
	SourceRef     string                   `json:"source_ref"`
	ContentHash   string                   `json:"content_hash"`
	MediaKind     string                   `json:"media_kind"`
	DetectedKind  string                   `json:"detected_kind,omitempty"`
	PageCount     int                      `json:"page_count"`
	LanguageHints []DocumentLanguageHintV0 `json:"language_hints,omitempty"`
	Pages         []DocumentPageV0         `json:"pages"`
}

type DocumentPageV0 struct {
	PageRef        string                `json:"page_ref"`
	Index          int                   `json:"index"`
	Width          float64               `json:"width"`
	Height         float64               `json:"height"`
	ImageHash      string                `json:"image_hash,omitempty"`
	TransformChain []DocumentTransformV0 `json:"transform_chain,omitempty"`
	Blocks         []DocumentBlockV0     `json:"blocks,omitempty"`
}

type DocumentBlockV0 struct {
	BlockRef     string                 `json:"block_ref"`
	Kind         string                 `json:"kind"`
	ReadingOrder int                    `json:"reading_order"`
	BoundingBox  *DocumentBoundingBoxV0 `json:"bbox,omitempty"`
	Polygon      []DocumentPointV0      `json:"polygon,omitempty"`
	Confidence   *float64               `json:"confidence,omitempty"`
	Provenance   DocumentProvenanceV0   `json:"provenance,omitempty"`
	Spans        []DocumentSpanV0       `json:"spans,omitempty"`
	Tables       []DocumentTableV0      `json:"tables,omitempty"`
}

type DocumentSpanV0 struct {
	SpanRef     string                 `json:"span_ref"`
	TextRaw     string                 `json:"text_raw"`
	BoundingBox *DocumentBoundingBoxV0 `json:"bbox,omitempty"`
	Polygon     []DocumentPointV0      `json:"polygon,omitempty"`
	Confidence  *float64               `json:"confidence,omitempty"`
	Provenance  DocumentProvenanceV0   `json:"provenance,omitempty"`
}

type DocumentTableV0 struct {
	TableRef    string                 `json:"table_ref"`
	BoundingBox *DocumentBoundingBoxV0 `json:"bbox,omitempty"`
	Polygon     []DocumentPointV0      `json:"polygon,omitempty"`
	Headers     []string               `json:"headers,omitempty"`
	Rows        []DocumentTableRowV0   `json:"rows,omitempty"`
	Confidence  *float64               `json:"confidence,omitempty"`
	Provenance  DocumentProvenanceV0   `json:"provenance,omitempty"`
}

type DocumentTableRowV0 struct {
	RowRef string                `json:"row_ref"`
	Cells  []DocumentTableCellV0 `json:"cells,omitempty"`
}

type DocumentTableCellV0 struct {
	CellRef     string                 `json:"cell_ref"`
	TextRaw     string                 `json:"text_raw"`
	BoundingBox *DocumentBoundingBoxV0 `json:"bbox,omitempty"`
	Confidence  *float64               `json:"confidence,omitempty"`
	Provenance  DocumentProvenanceV0   `json:"provenance,omitempty"`
}

type DocumentNormalizedValueV0 struct {
	Value               string `json:"value"`
	NormalizationLocale string `json:"normalization_locale"`
	RuleRef             string `json:"rule_ref"`
	ValidatorVersion    string `json:"validator_version"`
}

type DocumentFieldCandidateV0 struct {
	CandidateRef    string                     `json:"candidate_ref"`
	FieldRef        string                     `json:"field_ref"`
	OccurrenceRef   string                     `json:"occurrence_ref"`
	ValueRaw        string                     `json:"value_raw"`
	ValueNormalized *DocumentNormalizedValueV0 `json:"value_normalized,omitempty"`
	ValueState      DocumentValueStateV0       `json:"value_state"`
	DataTypeHint    DocumentFieldDataTypeV0    `json:"datatype_hint,omitempty"`
	Confidence      *float64                   `json:"confidence,omitempty"`
	EvidenceRefs    []string                   `json:"evidence_refs,omitempty"`
	SourceRefs      []string                   `json:"source_refs,omitempty"`
	ValidationRefs  []string                   `json:"validation_refs,omitempty"`
}

type DocumentEvidenceV0 struct {
	EvidenceRef string                 `json:"evidence_ref"`
	DocumentRef string                 `json:"document_ref"`
	PageRef     string                 `json:"page_ref"`
	BlockRef    string                 `json:"block_ref,omitempty"`
	SpanRefs    []string               `json:"span_refs,omitempty"`
	BoundingBox *DocumentBoundingBoxV0 `json:"bbox,omitempty"`
	Polygon     []DocumentPointV0      `json:"polygon,omitempty"`
	SourceRefs  []string               `json:"source_refs,omitempty"`
}

type DocumentSchemaFieldV0 struct {
	FieldRef string                  `json:"field_ref"`
	DataType DocumentFieldDataTypeV0 `json:"data_type"`
	Required bool                    `json:"required"`
}

type DocumentSchemaV0 struct {
	SchemaRef     string                  `json:"schema_ref"`
	DomainRef     string                  `json:"domain_ref"`
	EntityKind    string                  `json:"entity_kind"`
	SchemaVersion string                  `json:"schema_version"`
	Fields        []DocumentSchemaFieldV0 `json:"fields"`
}

type PersonDocumentSchemaV0 struct {
	Schema DocumentSchemaV0 `json:"schema"`
}

type InvoiceDocumentSchemaV0 struct {
	Schema DocumentSchemaV0 `json:"schema"`
}

type DocumentLocationV0 struct {
	LocationRef string   `json:"location_ref"`
	PageRef     string   `json:"page_ref"`
	BlockRefs   []string `json:"block_refs,omitempty"`
	TableRefs   []string `json:"table_refs,omitempty"`
}

type DocumentSchemaSliceV0 struct {
	SliceRef            string                  `json:"slice_ref"`
	SchemaRef           string                  `json:"schema_ref"`
	DocumentRef         string                  `json:"document_ref"`
	Location            DocumentLocationV0      `json:"location"`
	Fields              []DocumentSchemaFieldV0 `json:"fields"`
	DocumentLanguage    string                  `json:"document_language,omitempty"`
	NormalizationLocale string                  `json:"normalization_locale"`
}

type DocumentAdapterIdentityV0 struct {
	AdapterRef   string `json:"adapter_ref"`
	Version      string `json:"version"`
	BuildHash    string `json:"build_hash,omitempty"`
	ModelRef     string `json:"model_ref,omitempty"`
	ModelVersion string `json:"model_version,omitempty"`
}

type DocumentConnectorRegistrationV0 struct {
	ConnectorRef     string                     `json:"connector_ref"`
	Mode             DocumentDataHandlingModeV0 `json:"mode"`
	CapabilityRefs   []string                   `json:"capability_refs"`
	ConfigurationRef string                     `json:"configuration_ref"`
	AdapterIdentity  DocumentAdapterIdentityV0  `json:"adapter_identity"`
}

type DocumentConnectorRegistryV0 struct {
	RegistryRef string                            `json:"registry_ref"`
	Connectors  []DocumentConnectorRegistrationV0 `json:"connectors"`
}

type DocumentExtractionPolicyV0 struct {
	PolicyRef        string                     `json:"policy_ref"`
	DataHandlingMode DocumentDataHandlingModeV0 `json:"data_handling_mode"`
	CloudOptIn       bool                       `json:"cloud_opt_in"`
	RegionRef        string                     `json:"region_ref,omitempty"`
	RetentionRef     string                     `json:"retention_ref,omitempty"`
	DeletionRef      string                     `json:"deletion_ref,omitempty"`
	EncryptionRef    string                     `json:"encryption_ref,omitempty"`
	AuditRef         string                     `json:"audit_ref,omitempty"`
}

type DocumentSourceMaterialV0 struct {
	DocumentRef string `json:"document_ref"`
	SourceRef   string `json:"source_ref"`
	ContentHash string `json:"content_hash"`
	MediaKind   string `json:"media_kind"`
	Bytes       []byte `json:"-"`
}

type DocumentNormalizedMaterialV0 struct {
	Source     DocumentSourceMaterialV0 `json:"source"`
	Transforms []DocumentTransformV0    `json:"transforms,omitempty"`
}

type DocumentValidationResultV0 struct {
	Candidate           DocumentFieldCandidateV0 `json:"candidate"`
	Accepted            bool                     `json:"accepted"`
	RequiresHumanReview bool                     `json:"requires_human_review"`
	ReasonRef           string                   `json:"reason_ref,omitempty"`
	ValidationRefs      []string                 `json:"validation_refs,omitempty"`
}

type DocumentHumanReviewResultV0 struct {
	Candidate   DocumentFieldCandidateV0 `json:"candidate"`
	Accepted    bool                     `json:"accepted"`
	DecisionRef string                   `json:"decision_ref"`
	ReasonRef   string                   `json:"reason_ref,omitempty"`
}

type DocumentExportArtifactV0 struct {
	ArtifactRef    string `json:"artifact_ref"`
	DestinationRef string `json:"destination_ref"`
	Format         string `json:"format"`
	ContentHash    string `json:"content_hash"`
	Bytes          []byte `json:"-"`
}

type DocumentExtractionReceiptV0 struct {
	SchemaVersion       string                      `json:"schema_version"`
	ReceiptRef          string                      `json:"receipt_ref"`
	ContractVersion     string                      `json:"contract_version"`
	IRVersion           string                      `json:"ir_version"`
	DocumentRef         string                      `json:"document_ref"`
	ContentHash         string                      `json:"content_hash"`
	SchemaRefs          []string                    `json:"schema_refs"`
	SchemaSliceRefs     []string                    `json:"schema_slice_refs"`
	PolicyRef           string                      `json:"policy_ref"`
	ConfigurationRef    string                      `json:"configuration_ref"`
	ConfigurationHash   string                      `json:"configuration_hash"`
	DocumentLanguage    string                      `json:"document_language,omitempty"`
	NormalizationLocale string                      `json:"normalization_locale"`
	OutputLocale        string                      `json:"output_locale"`
	Adapters            []DocumentAdapterIdentityV0 `json:"adapters"`
	TransformRefs       []string                    `json:"transform_refs,omitempty"`
	EvidenceRefs        []string                    `json:"evidence_refs"`
	ArtifactRefs        []string                    `json:"artifact_refs"`
	Status              string                      `json:"status"`
}

type DocumentExtractionIssueV0 struct {
	Code         string `json:"code"`
	CandidateRef string `json:"candidate_ref,omitempty"`
	DetailRef    string `json:"detail_ref,omitempty"`
}

type DocumentToolSurfaceV0 string

const (
	DocumentToolSurfaceInspectV0 DocumentToolSurfaceV0 = "documents.inspect.v0"
	DocumentToolSurfaceExtractV0 DocumentToolSurfaceV0 = "documents.extract.v0"
	DocumentToolSurfaceStatusV0  DocumentToolSurfaceV0 = "documents.status.v0"
	DocumentToolSurfaceReviewV0  DocumentToolSurfaceV0 = "documents.review.v0"
	DocumentToolSurfaceExportV0  DocumentToolSurfaceV0 = "documents.export.v0"
)

type DocumentToolAttachModeV0 string

const (
	DocumentToolAttachModeEmbeddedModuleV0  DocumentToolAttachModeV0 = "embedded_module"
	DocumentToolAttachModeLocalSidecarV0    DocumentToolAttachModeV0 = "local_sidecar"
	DocumentToolAttachModeRemoteConnectorV0 DocumentToolAttachModeV0 = "remote_connector"
)

type DocumentToolOperationStateV0 string

const (
	DocumentToolOperationQueuedV0    DocumentToolOperationStateV0 = "queued"
	DocumentToolOperationRunningV0   DocumentToolOperationStateV0 = "running"
	DocumentToolOperationReviewV0    DocumentToolOperationStateV0 = "awaiting_review"
	DocumentToolOperationCompletedV0 DocumentToolOperationStateV0 = "completed"
	DocumentToolOperationFailedV0    DocumentToolOperationStateV0 = "failed"
)

type DocumentToolEffectProfileV0 struct {
	EffectRef              string                     `json:"effect_ref"`
	PermissionRefs         []string                   `json:"permission_refs"`
	DataHandlingMode       DocumentDataHandlingModeV0 `json:"data_handling_mode"`
	ExternalEffectsAllowed bool                       `json:"external_effects_allowed"`
}

type DocumentToolBudgetV0 struct {
	BudgetRef          string `json:"budget_ref"`
	MaxPages           int    `json:"max_pages,omitempty"`
	MaxBytes           int64  `json:"max_bytes,omitempty"`
	MaxDurationSeconds int    `json:"max_duration_seconds,omitempty"`
	MaxCostMinor       int64  `json:"max_cost_minor,omitempty"`
}

type DocumentToolCapabilityDescriptorV0 struct {
	CapabilityRef          string                      `json:"capability_ref"`
	Version                string                      `json:"version"`
	SurfaceRefs            []DocumentToolSurfaceV0     `json:"surface_refs"`
	RequiredCapabilityRefs []string                    `json:"required_capability_refs"`
	PermissionProfile      DocumentToolEffectProfileV0 `json:"permission_profile"`
	AttachModes            []DocumentToolAttachModeV0  `json:"attach_modes"`
	ConnectorRegistryRef   string                      `json:"connector_registry_ref,omitempty"`
	ReceiptSchemaVersion   string                      `json:"receipt_schema_version"`
}

// DocumentExtractionBundleDescriptorV0 is a temporary, extraction-specific
// attachment contract. It deliberately does not duplicate the generic
// ToolBundle/AttachPlan SDK, which will consume this descriptor later.
type DocumentExtractionBundleDescriptorV0 struct {
	BundleRef        string                             `json:"bundle_ref"`
	Capability       DocumentToolCapabilityDescriptorV0 `json:"capability"`
	AttachMode       DocumentToolAttachModeV0           `json:"attach_mode"`
	ModuleRef        string                             `json:"module_ref,omitempty"`
	ConnectorRef     string                             `json:"connector_ref,omitempty"`
	ConfigurationRef string                             `json:"configuration_ref"`
	I18NRef          string                             `json:"i18n_ref"`
	TestArtifactRefs []string                           `json:"test_artifact_refs"`
	ArtifactHashes   []DocumentArtifactHashV0           `json:"artifact_hashes"`
	ReceiptRef       string                             `json:"receipt_ref,omitempty"`
}

type DocumentArtifactHashV0 struct {
	ArtifactRef string `json:"artifact_ref"`
	Hash        string `json:"hash"`
}

type DocumentToolCommandV0 struct {
	CommandRef       string                      `json:"command_ref"`
	Surface          DocumentToolSurfaceV0       `json:"surface"`
	IdempotencyKey   string                      `json:"idempotency_key"`
	DocumentRef      string                      `json:"document_ref,omitempty"`
	SchemaRef        string                      `json:"schema_ref,omitempty"`
	OperationRef     string                      `json:"operation_ref,omitempty"`
	ReviewRef        string                      `json:"review_ref,omitempty"`
	ExportRef        string                      `json:"export_ref,omitempty"`
	ConfigurationRef string                      `json:"configuration_ref"`
	PolicyRef        string                      `json:"policy_ref,omitempty"`
	Budget           DocumentToolBudgetV0        `json:"budget"`
	EffectProfile    DocumentToolEffectProfileV0 `json:"effect_profile"`
}

type DocumentToolPublicErrorV0 struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key"`
	Retryable  bool   `json:"retryable"`
}

const (
	DocumentToolPublicErrorInvalidCommandV0 = "document_tool_invalid_command"
	DocumentToolPublicErrorNotFoundV0       = "document_tool_operation_not_found"
	DocumentToolPublicErrorUnavailableV0    = "document_tool_unavailable"
)

type DocumentToolOperationV0 struct {
	OperationRef   string                       `json:"operation_ref"`
	CommandRef     string                       `json:"command_ref"`
	Surface        DocumentToolSurfaceV0        `json:"surface"`
	IdempotencyKey string                       `json:"idempotency_key"`
	State          DocumentToolOperationStateV0 `json:"state"`
	ResultRef      string                       `json:"result_ref,omitempty"`
	ReceiptRef     string                       `json:"receipt_ref,omitempty"`
	PublicError    *DocumentToolPublicErrorV0   `json:"public_error,omitempty"`
}
