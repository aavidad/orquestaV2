package orquestacontext

const (
	ContextBundleRequestSchemaVersionV0 = "context_bundle_request.v0"
	ContextBundleSchemaVersionV0        = "context_bundle.v0"
)

type ContextBundleIssueCodeV0 string

const (
	ErrContextBundleSchemaNoSoportadoV0 ContextBundleIssueCodeV0 = "context_bundle_schema_no_soportado"
	ErrContextBundleCampoRequeridoV0    ContextBundleIssueCodeV0 = "context_bundle_campo_requerido"
	ErrContextBundleModuloInvalidoV0    ContextBundleIssueCodeV0 = "context_bundle_modulo_invalido"
	ErrContextBundleFaseNoSoportadaV0   ContextBundleIssueCodeV0 = "context_bundle_fase_no_soportada"
	ErrContextBundleCapacidadInvalidaV0 ContextBundleIssueCodeV0 = "context_bundle_capacidad_invalida"
	ErrContextBundleDetalleProhibidoV0  ContextBundleIssueCodeV0 = "context_bundle_detalle_prohibido"
	ErrContextBundleTamanoInvalidoV0    ContextBundleIssueCodeV0 = "context_bundle_tamano_invalido"
)

const (
	ContextLayerCommonRulesV0     = "common_rules"
	ContextLayerModuleContextV0   = "module_context"
	ContextLayerPhaseContextV0    = "phase_context"
	ContextLayerTaskContextV0     = "task_context"
	ContextLayerContractContextV0 = "contract_context"
	ContextLayerEvidenceContextV0 = "evidence_context"
)

const (
	ContextEntryRuleRefV0     = "rule_ref"
	ContextEntryDocRefV0      = "doc_ref"
	ContextEntryReadRefV0     = "read_ref"
	ContextEntryWriteRefV0    = "write_ref"
	ContextEntryContractRefV0 = "contract_ref"
	ContextEntryEvidenceRefV0 = "evidence_ref"
)

type ContextBundleRequestV0 struct {
	SchemaVersion   string   `json:"schema_version"`
	BundleRef       string   `json:"bundle_ref"`
	WorkOrderRef    string   `json:"work_order_ref"`
	TargetModule    string   `json:"target_module"`
	Phase           string   `json:"phase"`
	TaskKind        string   `json:"task_kind"`
	Objective       string   `json:"objective"`
	CapacityLevel   string   `json:"capacity_level"`
	ReadSet         []string `json:"read_set,omitempty"`
	WriteSet        []string `json:"write_set,omitempty"`
	ContractRefs    []string `json:"contract_refs,omitempty"`
	CrossModuleRefs []string `json:"cross_module_refs,omitempty"`
	EvidenceRefs    []string `json:"evidence_refs,omitempty"`
	MaxEntries      int      `json:"max_entries,omitempty"`
	MaxTotalBytes   int      `json:"max_total_bytes,omitempty"`
}

type ContextBundleV0 struct {
	SchemaVersion string                 `json:"schema_version"`
	BundleRef     string                 `json:"bundle_ref"`
	WorkOrderRef  string                 `json:"work_order_ref"`
	TargetModule  string                 `json:"target_module"`
	Phase         string                 `json:"phase"`
	CapacityLevel string                 `json:"capacity_level"`
	Summary       ContextBundleSummaryV0 `json:"summary"`
	Limits        ContextBundleLimitsV0  `json:"limits"`
	Entries       []ContextBundleEntryV0 `json:"entries"`
	Issues        []ContextBundleIssueV0 `json:"issues,omitempty"`
}

type ContextBundleSummaryV0 struct {
	TaskKind               string `json:"task_kind"`
	Objective              string `json:"objective"`
	ContextPolicy          string `json:"context_policy"`
	DirectorQuestionPolicy string `json:"director_question_policy"`
}

type ContextBundleLimitsV0 struct {
	MaxEntries    int `json:"max_entries"`
	MaxTotalBytes int `json:"max_total_bytes"`
}

type ContextBundleEntryV0 struct {
	EntryRef  string `json:"entry_ref"`
	Layer     string `json:"layer"`
	Kind      string `json:"kind"`
	SourceRef string `json:"source_ref"`
	Reason    string `json:"reason"`
	Required  bool   `json:"required"`
	MaxBytes  int    `json:"max_bytes"`
}

type ContextBundleIssueV0 struct {
	Code    ContextBundleIssueCodeV0 `json:"code"`
	Field   string                   `json:"field,omitempty"`
	Message string                   `json:"message,omitempty"`
}

func (bundle ContextBundleV0) Valid() bool {
	return len(bundle.Issues) == 0 &&
		bundle.SchemaVersion == ContextBundleSchemaVersionV0 &&
		bundle.BundleRef != "" &&
		len(bundle.Entries) > 0
}
