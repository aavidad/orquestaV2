package orquestadomainwork

const (
	DomainDocumentPlanSchemaV0       = "domain_document_plan.v0"
	DomainDocumentPlanArtifactTypeV0 = "document_plan"

	DomainWorkKindPlanDocumentV0 = "plan_documento"
	DomainWorkKindPlanTopicV0    = "plan_tema"
	DomainWorkKindPlanSyllabusV0 = "plan_temario"

	ErrDomainDocumentPlanRefRequiredV0          = "domain_document_plan_ref_required"
	ErrDomainDocumentPlanRefInvalidV0           = "domain_document_plan_ref_invalid"
	ErrDomainDocumentPlanWorkKindRequiredV0     = "domain_document_plan_work_kind_required"
	ErrDomainDocumentPlanWorkKindInvalidV0      = "domain_document_plan_work_kind_invalid"
	ErrDomainDocumentPlanDocumentKindRequiredV0 = "domain_document_plan_document_kind_required"
	ErrDomainDocumentPlanTitleRequiredV0        = "domain_document_plan_title_required"
	ErrDomainDocumentPlanObjectiveRequiredV0    = "domain_document_plan_objective_required"
	ErrDomainDocumentPlanLanguageRequiredV0     = "domain_document_plan_language_required"
	ErrDomainDocumentPlanSectionsRequiredV0     = "domain_document_plan_sections_required"
	ErrDomainDocumentPlanDeliverablesRequiredV0 = "domain_document_plan_deliverables_required"
	ErrDomainDocumentPlanRangeInvalidV0         = "domain_document_plan_range_invalid"
)

type DomainDocumentPlanV0 struct {
	SchemaVersion     string                            `json:"schema_version"`
	PlanRef           string                            `json:"plan_ref"`
	DomainRef         string                            `json:"domain_ref"`
	WorkKind          string                            `json:"work_kind"`
	DocumentKind      string                            `json:"document_kind"`
	ScopeRef          string                            `json:"scope_ref,omitempty"`
	LanguageCode      string                            `json:"language_code"`
	Title             string                            `json:"title"`
	Objective         string                            `json:"objective"`
	TargetAudience    string                            `json:"target_audience,omitempty"`
	EstimatedPagesMin int                               `json:"estimated_pages_min,omitempty"`
	EstimatedPagesMax int                               `json:"estimated_pages_max,omitempty"`
	Sections          []DomainDocumentPlanSectionV0     `json:"sections"`
	Visuals           []DomainDocumentPlanVisualV0      `json:"visuals,omitempty"`
	ReviewSteps       []DomainDocumentPlanReviewV0      `json:"review_steps,omitempty"`
	Deliverables      []DomainDocumentPlanDeliverableV0 `json:"deliverables"`
	QualityCriteria   []string                          `json:"quality_criteria,omitempty"`
	Constraints       []string                          `json:"constraints,omitempty"`
	SourceRefs        []string                          `json:"source_refs,omitempty"`
	EvidenceRefs      []string                          `json:"evidence_refs,omitempty"`
}

type PlanTemaV0 = DomainDocumentPlanV0
type PlanTemarioV0 = DomainDocumentPlanV0

type DomainDocumentPlanSectionV0 struct {
	SectionRef         string   `json:"section_ref"`
	ParentRef          string   `json:"parent_ref,omitempty"`
	Order              int      `json:"order"`
	Title              string   `json:"title"`
	Objective          string   `json:"objective"`
	WorkKind           string   `json:"work_kind"`
	DependsOn          []string `json:"depends_on,omitempty"`
	TargetWordsMin     int      `json:"target_words_min,omitempty"`
	TargetWordsMax     int      `json:"target_words_max,omitempty"`
	RequiredElements   []string `json:"required_elements,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	SourceRefs         []string `json:"source_refs,omitempty"`
}

type DomainDocumentPlanVisualV0 struct {
	VisualRef          string   `json:"visual_ref"`
	VisualType         string   `json:"visual_type"`
	PlacementRef       string   `json:"placement_ref,omitempty"`
	Objective          string   `json:"objective"`
	WorkKind           string   `json:"work_kind"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	SourceRefs         []string `json:"source_refs,omitempty"`
}

type DomainDocumentPlanReviewV0 struct {
	ReviewRef          string   `json:"review_ref"`
	Order              int      `json:"order"`
	WorkKind           string   `json:"work_kind"`
	Objective          string   `json:"objective"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
}

type DomainDocumentPlanDeliverableV0 struct {
	DeliverableRef string `json:"deliverable_ref"`
	ArtifactType   string `json:"artifact_type"`
	Title          string `json:"title"`
	Required       bool   `json:"required"`
}
