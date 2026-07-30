// Tipos del contrato de raíces históricas; se separan para mantener cada
// responsabilidad de prueba en un fichero manejable.
package orquesta_test

const legacySourceRootsPath = "product/traceability/legacy_source_roots_2026-07-30.json"

type legacyRootsDocument struct {
	DocumentKind                    string                  `json:"document_kind"`
	SchemaVersion                   int                     `json:"schema_version"`
	AssembledAt                     string                  `json:"assembled_at"`
	Closed                          bool                    `json:"closed"`
	ClosureReason                   string                  `json:"closure_reason"`
	SourceUniverseStatement         string                  `json:"source_universe_statement"`
	LogicalPathPolicy               legacyRootsPathPolicy   `json:"logical_path_policy"`
	Vocabularies                    legacyRootsVocabularies `json:"vocabularies"`
	ObservationMethod               legacyRootsMethod       `json:"observation_method"`
	ObservationBatches              []legacyRootsBatch      `json:"observation_batches"`
	UnaccreditedHistoricalReports   []legacyRootsReport     `json:"unaccredited_historical_reports"`
	IgnoredEntriesAdvisory          legacyRootsIgnored      `json:"ignored_entries_advisory"`
	RootIDsSHA256                   string                  `json:"root_ids_sha256"`
	Roots                           []legacyRootsRoot       `json:"roots"`
	GitFacts                        []legacyRootsGitFact    `json:"git_facts"`
	FileFacts                       []legacyRootsFileFact   `json:"file_facts"`
	Collections                     []legacyRootsCollection `json:"collections"`
	V1V22Tags                       []legacyRootsTag        `json:"v1_v22_tags"`
	GitObjectReconciliation         []legacyRootsReconcile  `json:"git_object_reconciliation"`
	UnresolvedSourceDecisionRootIDs []string                `json:"unresolved_source_decision_root_ids"`
	PendingObservationBatchIDs      []string                `json:"pending_observation_batch_ids"`
	BlockingPhysicalCensusRootIDs   []string                `json:"blocking_physical_census_root_ids"`
}

type legacyRootsPathPolicy struct {
	RootRefs            []string `json:"root_refs"`
	LocalMappingTracked bool     `json:"local_mapping_tracked"`
	ProfileIDsPublished bool     `json:"profile_ids_published"`
	Rule                string   `json:"rule"`
}

type legacyRootsVocabularies struct {
	RootClasses        []string `json:"root_classes"`
	ScopeKinds         []string `json:"scope_kinds"`
	Natures            []string `json:"natures"`
	SourceDecisions    []string `json:"source_decisions"`
	GitCensusStatuses  []string `json:"git_census_statuses"`
	PhysicalStatuses   []string `json:"physical_census_statuses"`
	GitObjectCoverages []string `json:"git_object_coverages"`
}

type legacyRootsMethod struct {
	GitVersion      string                    `json:"git_version"`
	Commands        []string                  `json:"commands"`
	Semantics       map[string]string         `json:"semantics"`
	ComparedRefSets []legacyRootsComparedRefs `json:"compared_ref_sets"`
}

type legacyRootsBatch struct {
	ID              string                 `json:"id"`
	Kind            string                 `json:"kind"`
	StartedAt       *string                `json:"started_at"`
	FinishedAt      *string                `json:"finished_at"`
	TimePrecision   string                 `json:"time_precision"`
	Status          string                 `json:"status"`
	RawCensusSHA256 *string                `json:"raw_census_sha256"`
	Reason          string                 `json:"reason"`
	Metrics         map[string]int         `json:"metrics"`
	WorktreeCounts  []legacyRootsWorktrees `json:"worktree_counts"`
}

type legacyRootsWorktrees struct {
	CommonGitDirRef            string `json:"common_git_dir_ref"`
	TotalCount                 int    `json:"total_count"`
	IncludesMain               bool   `json:"includes_main"`
	AdditionalMemberCount      int    `json:"additional_member_count"`
	DirtyTotalCount            *int   `json:"dirty_total_count"`
	DirtyAdditionalMemberCount *int   `json:"dirty_additional_member_count"`
}

type legacyRootsReport struct {
	ID                      string `json:"id"`
	ObservationBatchID      string `json:"observation_batch_id"`
	SubjectRootID           string `json:"subject_root_id"`
	ReportedTotalCount      int    `json:"reported_total_count"`
	IncludesMain            bool   `json:"includes_main"`
	ReportedDirtyTotalCount int    `json:"reported_dirty_total_count"`
	EvidenceStatus          string `json:"evidence_status"`
	Reason                  string `json:"reason"`
}

type legacyRootsIgnored struct {
	ObservationBatchID string `json:"observation_batch_id"`
	Status             string `json:"status"`
	AffectedRootCount  int    `json:"affected_root_count"`
	IgnoredEntryCount  int    `json:"ignored_entry_count"`
	Reason             string `json:"reason"`
}

type legacyRootsComparedRefs struct {
	LeftRootID  string `json:"left_root_id"`
	RightRootID string `json:"right_root_id"`
	Namespace   string `json:"namespace"`
}

type legacyRootsRoot struct {
	ID                     string   `json:"id"`
	LogicalRoot            string   `json:"logical_root"`
	RelativePath           string   `json:"relative_path"`
	ExistsAtObservation    bool     `json:"exists_at_observation"`
	Class                  string   `json:"class"`
	ScopeKind              string   `json:"scope_kind"`
	Nature                 string   `json:"nature"`
	SourceDecision         string   `json:"source_decision"`
	SourceDecisionReason   string   `json:"source_decision_reason"`
	ObservationBatchID     string   `json:"observation_batch_id"`
	GitCensusStatus        string   `json:"git_census_status"`
	GitCensusStatusReason  string   `json:"git_census_status_reason"`
	PhysicalCensusStatus   string   `json:"physical_census_status"`
	PhysicalStatusReason   string   `json:"physical_census_status_reason"`
	PhysicalCensusRequired bool     `json:"physical_census_required"`
	GitObjectCoverage      string   `json:"git_object_coverage"`
	EvidenceRefs           []string `json:"evidence_refs"`
}

type legacyRootsGitFact struct {
	RootID                string `json:"root_id"`
	CommonGitDirRef       string `json:"common_git_dir_ref"`
	HeadOID               string `json:"head_oid"`
	RefCount              *int   `json:"ref_count"`
	CommitCount           *int   `json:"commit_count"`
	ReachableOnlyFromRoot *int   `json:"reachable_only_from_root"`
	DirtyEntryCount       *int   `json:"dirty_entry_count"`
	Prunable              bool   `json:"prunable"`
	ObservationBatchID    string `json:"observation_batch_id"`
}

type legacyRootsFileFact struct {
	RootID             string `json:"root_id"`
	SHA256             string `json:"sha256"`
	ByteSize           int64  `json:"byte_size"`
	Verification       string `json:"verification"`
	ObservationBatchID string `json:"observation_batch_id"`
}

type legacyRootsCollection struct {
	RootID             string              `json:"root_id"`
	MembersSHA256      string              `json:"members_sha256"`
	Members            []legacyRootsMember `json:"members"`
	ObservationBatchID string              `json:"observation_batch_id"`
}

type legacyRootsMember struct {
	PathAlias           string  `json:"path_alias"`
	ExistsAtObservation bool    `json:"exists_at_observation"`
	Nature              string  `json:"nature"`
	HeadOID             *string `json:"head_oid"`
	CommonGitDirRef     *string `json:"common_git_dir_ref"`
	DirtyEntryCount     *int    `json:"dirty_entry_count"`
	Prunable            *bool   `json:"prunable"`
	SHA256              *string `json:"sha256"`
	ByteSize            *int64  `json:"byte_size"`
}

type legacyRootsTag struct {
	Name               string   `json:"name"`
	TagObjectOID       string   `json:"tag_object_oid"`
	CommitOID          string   `json:"commit_oid"`
	ObservedRootIDs    []string `json:"observed_root_ids"`
	ObservationBatchID string   `json:"observation_batch_id"`
}

type legacyRootsReconcile struct {
	RootID                    string `json:"root_id"`
	ReachableOnlyFromRoot     int    `json:"reachable_only_from_root"`
	ContainedInCurrentProduct bool   `json:"contained_in_current_product"`
	EvidenceStatus            string `json:"evidence_status"`
	Reason                    string `json:"reason"`
	ObservationBatchID        string `json:"observation_batch_id"`
}
