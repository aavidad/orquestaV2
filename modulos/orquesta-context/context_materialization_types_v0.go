package orquestacontext

const ContextMaterializedBundleSchemaVersionV0 = "context_materialized_bundle.v0"

type ContextMaterializationIssueCodeV0 string

const (
	ErrContextMaterializationBundleInvalidoV0   ContextMaterializationIssueCodeV0 = "context_materialization_bundle_invalido"
	ErrContextMaterializationReaderRequeridoV0  ContextMaterializationIssueCodeV0 = "context_materialization_reader_requerido"
	ErrContextMaterializationRootInvalidoV0     ContextMaterializationIssueCodeV0 = "context_materialization_root_invalido"
	ErrContextMaterializationRefInvalidaV0      ContextMaterializationIssueCodeV0 = "context_materialization_ref_invalida"
	ErrContextMaterializationRefNoEncontradaV0  ContextMaterializationIssueCodeV0 = "context_materialization_ref_no_encontrada"
	ErrContextMaterializationTamanoV0           ContextMaterializationIssueCodeV0 = "context_materialization_tamano_invalido"
	ErrContextMaterializationDetalleProhibidoV0 ContextMaterializationIssueCodeV0 = "context_materialization_detalle_prohibido"
)

const (
	ContextMaterializationModeContentV0 = "content"
	ContextMaterializationModeRefOnlyV0 = "ref_only"
)

type ContextRefReaderV0 interface {
	ReadContextRefV0(sourceRef string, maxBytes int) (ContextRefContentV0, []ContextMaterializationIssueV0)
}

type ContextRefContentV0 struct {
	SourceRef string `json:"source_ref"`
	Content   string `json:"content"`
	Bytes     int    `json:"bytes"`
	Truncated bool   `json:"truncated"`
}

type ContextMaterializedBundleV0 struct {
	SchemaVersion        string                          `json:"schema_version"`
	BundleRef            string                          `json:"bundle_ref"`
	WorkOrderRef         string                          `json:"work_order_ref"`
	TargetModule         string                          `json:"target_module"`
	TotalBytes           int                             `json:"total_bytes"`
	Entries              []ContextMaterializedEntryV0    `json:"entries"`
	Issues               []ContextMaterializationIssueV0 `json:"issues,omitempty"`
	DirectorQuestionHint string                          `json:"director_question_hint,omitempty"`
}

type ContextMaterializedEntryV0 struct {
	EntryRef  string `json:"entry_ref"`
	Layer     string `json:"layer"`
	Kind      string `json:"kind"`
	SourceRef string `json:"source_ref"`
	Mode      string `json:"mode"`
	Content   string `json:"content,omitempty"`
	Bytes     int    `json:"bytes,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Required  bool   `json:"required"`
}

type ContextMaterializationIssueV0 struct {
	Code    ContextMaterializationIssueCodeV0 `json:"code"`
	Field   string                            `json:"field,omitempty"`
	Message string                            `json:"message,omitempty"`
}

func (materialized ContextMaterializedBundleV0) Valid() bool {
	return materialized.SchemaVersion == ContextMaterializedBundleSchemaVersionV0 &&
		materialized.BundleRef != "" &&
		len(materialized.Entries) > 0 &&
		len(materialized.Issues) == 0
}
