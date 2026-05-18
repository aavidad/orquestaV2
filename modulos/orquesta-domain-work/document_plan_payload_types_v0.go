package orquestadomainwork

type DomainDocumentPlanPayloadDefaultsV0 struct {
	PlanRef           string
	DomainRef         string
	WorkKind          string
	DocumentKind      string
	ScopeRef          string
	LanguageCode      string
	Title             string
	Objective         string
	TargetAudience    string
	EstimatedPagesMin int
	EstimatedPagesMax int
}
