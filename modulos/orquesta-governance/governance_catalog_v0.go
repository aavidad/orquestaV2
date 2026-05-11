package orquestagovernance

import "strings"

const (
	GovernanceCatalogContractV0         = "GovernanceCatalogV0"
	GovernanceCatalogVersionV0          = "v0"
	GovernanceCatalogOwnerV0            = "orquesta-governance"
	GovernanceCatalogActivationPolicyV0 = "historical_entries_require_review"

	GovernanceCatalogStateEffectiveV0  = "effective"
	GovernanceCatalogStateProposedV0   = "proposed"
	GovernanceCatalogStateQuarantineV0 = "quarantine"

	GovernanceReviewStateApprovedEffectiveV0 = "approved_effective"
)

const (
	GovernanceCatalogForbiddenActivationErrorV0 GovernanceCatalogErrorV0 = "governance_catalog_forbidden_activation"
	GovernanceEntryEffectiveWithoutDecisionV0   GovernanceCatalogErrorV0 = "governance_entry_effective_without_decision"
)

type GovernanceCatalogErrorV0 string

func (e GovernanceCatalogErrorV0) Error() string {
	return string(e)
}

type GovernanceCatalogV0 struct {
	Contract            string                          `json:"contract"`
	CatalogVersion      string                          `json:"catalog_version"`
	Owner               string                          `json:"owner"`
	ActivationPolicy    string                          `json:"activation_policy"`
	SourcePolicy        GovernanceSourcePolicyV0        `json:"source_policy"`
	SecretControlPolicy GovernanceSecretControlPolicyV0 `json:"secret_control_policy"`
	Catalogs            GovernanceCatalogBlocksV0       `json:"catalogs"`
}

type GovernanceSourcePolicyV0 struct {
	DBV1IsForensicOnly                 bool `json:"dbv1_is_forensic_only"`
	LegacyIDsAreNotCanonical           bool `json:"legacy_ids_are_not_canonical"`
	HistoricalActivationRequiresReview bool `json:"historical_activation_requires_review"`
}

type GovernanceSecretControlPolicyV0 struct {
	StructuredControlsRequired bool     `json:"structured_controls_required"`
	FreeTextScanOnlyAllowed    bool     `json:"free_text_scan_only_allowed"`
	ProhibitedMaterial         []string `json:"prohibited_material"`
}

type GovernanceCatalogBlocksV0 struct {
	Effective  []GovernanceCatalogEntryV0 `json:"effective"`
	Proposed   []GovernanceCatalogEntryV0 `json:"proposed"`
	Quarantine []GovernanceCatalogEntryV0 `json:"quarantine"`
}

type GovernanceCatalogEntryV0 struct {
	Kind           string                     `json:"kind"`
	Name           string                     `json:"name"`
	Summary        string                     `json:"summary"`
	Scope          GovernanceScopeV0          `json:"scope"`
	Status         GovernanceStatusV0         `json:"status"`
	Version        GovernanceVersionV0        `json:"version"`
	Origin         GovernanceOriginV0         `json:"origin"`
	Promotion      GovernancePromotionV0      `json:"promotion"`
	SecretControls GovernanceSecretControlsV0 `json:"secret_controls"`
}

type GovernanceScopeV0 struct {
	Level   string   `json:"level"`
	Modules []string `json:"modules,omitempty"`
	Roles   []string `json:"roles,omitempty"`
	Phases  []string `json:"phases,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

type GovernanceStatusV0 struct {
	CatalogState string `json:"catalog_state"`
	ReviewState  string `json:"review_state"`
}

type GovernanceVersionV0 struct {
	ContractVersion       string `json:"contract_version"`
	DocumentVersion       string `json:"document_version"`
	SourceVersionEvidence string `json:"source_version_evidence"`
}

type GovernanceOriginV0 struct {
	SourceType           string `json:"source_type"`
	SourceRef            string `json:"source_ref"`
	ForensicInventoryRef string `json:"forensic_inventory_ref"`
	LegacyPublicIDPolicy string `json:"legacy_public_id_policy"`
}

type GovernancePromotionV0 struct {
	Criterion                string  `json:"criterion"`
	DecisionRef              *string `json:"decision_ref"`
	PromotedAt               *string `json:"promoted_at"`
	HistoricalReviewRequired bool    `json:"historical_review_required"`
}

type GovernanceSecretControlsV0 struct {
	ContainsSecretMaterial   bool     `json:"contains_secret_material"`
	ReviewMethod             string   `json:"review_method"`
	StructuredMarkersChecked []string `json:"structured_markers_checked"`
}

type GovernanceCatalogQueryV0 struct {
	Module string
	Role   string
	Phase  string
	Tags   []string
}

type GovernanceCatalogQueryResultV0 struct {
	Effective []GovernanceCatalogEntryV0  `json:"effective"`
	Counters  GovernanceCatalogCountersV0 `json:"counters"`
}

type GovernanceCatalogCountersV0 struct {
	Effective  int `json:"effective"`
	Proposed   int `json:"proposed"`
	Quarantine int `json:"quarantine"`
}

func QueryEffectiveGovernanceCatalogV0(catalog GovernanceCatalogV0, query GovernanceCatalogQueryV0) (GovernanceCatalogQueryResultV0, error) {
	result := GovernanceCatalogQueryResultV0{}

	if err := ValidateGovernanceCatalogPlacementV0(catalog); err != nil {
		return result, err
	}

	for _, entry := range catalog.Catalogs.Effective {
		if governanceEntryMatchesQueryV0(entry, query) {
			result.Effective = append(result.Effective, entry)
		}
	}
	for _, entry := range catalog.Catalogs.Proposed {
		if governanceEntryMatchesQueryV0(entry, query) {
			result.Counters.Proposed++
		}
	}
	for _, entry := range catalog.Catalogs.Quarantine {
		if governanceEntryMatchesQueryV0(entry, query) {
			result.Counters.Quarantine++
		}
	}

	result.Counters.Effective = len(result.Effective)
	return result, nil
}

func ValidateGovernanceCatalogPlacementV0(catalog GovernanceCatalogV0) error {
	for _, entry := range catalog.Catalogs.Effective {
		if err := ValidateEffectiveGovernanceEntryV0(entry); err != nil {
			return err
		}
	}
	for _, entry := range catalog.Catalogs.Proposed {
		if entry.Status.CatalogState != GovernanceCatalogStateProposedV0 {
			return GovernanceCatalogForbiddenActivationErrorV0
		}
	}
	for _, entry := range catalog.Catalogs.Quarantine {
		if entry.Status.CatalogState != GovernanceCatalogStateQuarantineV0 {
			return GovernanceCatalogForbiddenActivationErrorV0
		}
	}
	return nil
}

func ValidateEffectiveGovernanceEntryV0(entry GovernanceCatalogEntryV0) error {
	if entry.Status.CatalogState != GovernanceCatalogStateEffectiveV0 ||
		entry.Status.ReviewState != GovernanceReviewStateApprovedEffectiveV0 {
		return GovernanceCatalogForbiddenActivationErrorV0
	}
	if governanceStringPointerIsBlankV0(entry.Promotion.DecisionRef) ||
		governanceStringPointerIsBlankV0(entry.Promotion.PromotedAt) {
		return GovernanceEntryEffectiveWithoutDecisionV0
	}
	return nil
}

func governanceStringPointerIsBlankV0(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}

func governanceEntryMatchesQueryV0(entry GovernanceCatalogEntryV0, query GovernanceCatalogQueryV0) bool {
	return governanceScopeValueMatchesV0(query.Module, entry.Scope.Modules) &&
		governanceScopeValueMatchesV0(query.Role, entry.Scope.Roles) &&
		governanceScopeValueMatchesV0(query.Phase, entry.Scope.Phases) &&
		governanceTagsMatchV0(query.Tags, entry.Scope.Tags)
}

func governanceScopeValueMatchesV0(filter string, values []string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" || len(values) == 0 {
		return true
	}
	for _, value := range values {
		if value == filter {
			return true
		}
	}
	return false
}

func governanceTagsMatchV0(filters []string, tags []string) bool {
	required := governanceCleanTagsV0(filters)
	if len(required) == 0 {
		return true
	}
	if len(tags) == 0 {
		return false
	}

	available := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			available[tag] = struct{}{}
		}
	}
	for _, tag := range required {
		if _, ok := available[tag]; !ok {
			return false
		}
	}
	return true
}

func governanceCleanTagsV0(tags []string) []string {
	clean := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		clean = append(clean, tag)
	}
	return clean
}
