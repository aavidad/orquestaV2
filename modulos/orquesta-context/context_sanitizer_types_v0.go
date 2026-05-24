package orquestacontext

const ContextSanitizationEvidenceSchemaVersionV0 = "context_sanitization_evidence.v0"

type ContextSanitizationStatusV0 string

const (
	ContextSanitizationStatusCleanV0          ContextSanitizationStatusV0 = "clean"
	ContextSanitizationStatusSanitizedV0      ContextSanitizationStatusV0 = "sanitized"
	ContextSanitizationStatusReviewRequiredV0 ContextSanitizationStatusV0 = "review_required"
	ContextSanitizationStatusBlockedV0        ContextSanitizationStatusV0 = "blocked"
)

type ContextSanitizerPortV0 interface {
	SanitizeContextEntryV0(ContextSanitizationRequestV0) ContextSanitizationResultV0
}

type ContextSanitizationRequestV0 struct {
	BundleRef    string `json:"bundle_ref"`
	WorkOrderRef string `json:"work_order_ref"`
	TargetModule string `json:"target_module"`
	EntryRef     string `json:"entry_ref"`
	SourceRef    string `json:"source_ref"`
	Content      string `json:"content"`
	Bytes        int    `json:"bytes"`
}

type ContextSanitizationResultV0 struct {
	Status   ContextSanitizationStatusV0     `json:"status"`
	Content  string                          `json:"content,omitempty"`
	Evidence ContextSanitizationEvidenceV0   `json:"evidence"`
	Issues   []ContextMaterializationIssueV0 `json:"issues,omitempty"`
}

type ContextSanitizationEvidenceV0 struct {
	SchemaVersion    string                      `json:"schema_version"`
	EvidenceRef      string                      `json:"evidence_ref"`
	BundleRef        string                      `json:"bundle_ref"`
	WorkOrderRef     string                      `json:"work_order_ref"`
	TargetModule     string                      `json:"target_module"`
	EntryRef         string                      `json:"entry_ref"`
	SourceRef        string                      `json:"source_ref"`
	SanitizerRef     string                      `json:"sanitizer_ref"`
	Status           ContextSanitizationStatusV0 `json:"status"`
	Categories       []string                    `json:"categories,omitempty"`
	ReplacementCount int                         `json:"replacement_count,omitempty"`
	ReviewRequired   bool                        `json:"review_required,omitempty"`
}
