// Este fichero declara únicamente los contratos JSON de entrada y salida sellada.
package main

import "encoding/json"

const (
	expectedSourceSHA256      = "0feffd7e493bb643bd6dbfa393d8480688f73f7ef452dbc73fbd0e23ede02044"
	expectedRootIDsSHA256     = "sha256:c4a93e4679a045ac68d835accb7bfd0ff0b6cb918bc2a7c9ab1248d3803291ec"
	expectedPendingSHA256     = "sha256:2d2b81b27bfccad5a0b3a6dfabdec299a516dc3b31400f2cb2ecd6948c1e7834"
	sourceDigestDomain        = "orquesta.legacy-source-roots.file-bytes.v1"
	pendingDigestDomain       = "orquesta.legacy-blocking-root-ids.ordered-nul.v1"
	membersDigestDomain       = "orquesta.legacy-collection-members.canonical-json-lf.v1"
	subjectDigestDomain       = "orquesta.legacy-physical-subjects.ordered-canonical-json-lf.v1"
	maxSourceBytes        int = 1 << 20
	maxUniverseBytes      int = 1 << 20
)

type sourceDocument struct {
	DocumentKind                    string             `json:"document_kind"`
	SchemaVersion                   int                `json:"schema_version"`
	AssembledAt                     json.RawMessage    `json:"assembled_at"`
	Closed                          bool               `json:"closed"`
	ClosureReason                   json.RawMessage    `json:"closure_reason"`
	SourceUniverseStatement         json.RawMessage    `json:"source_universe_statement"`
	LogicalPathPolicy               json.RawMessage    `json:"logical_path_policy"`
	Vocabularies                    json.RawMessage    `json:"vocabularies"`
	ObservationMethod               json.RawMessage    `json:"observation_method"`
	ObservationBatches              json.RawMessage    `json:"observation_batches"`
	UnaccreditedHistoricalReports   json.RawMessage    `json:"unaccredited_historical_reports"`
	IgnoredEntriesAdvisory          json.RawMessage    `json:"ignored_entries_advisory"`
	RootIDsSHA256                   string             `json:"root_ids_sha256"`
	Roots                           []sourceRoot       `json:"roots"`
	GitFacts                        json.RawMessage    `json:"git_facts"`
	FileFacts                       json.RawMessage    `json:"file_facts"`
	Collections                     []sourceCollection `json:"collections"`
	V1V22Tags                       json.RawMessage    `json:"v1_v22_tags"`
	GitObjectReconciliation         json.RawMessage    `json:"git_object_reconciliation"`
	UnresolvedSourceDecisionRootIDs json.RawMessage    `json:"unresolved_source_decision_root_ids"`
	PendingObservationBatchIDs      json.RawMessage    `json:"pending_observation_batch_ids"`
	BlockingPhysicalCensusRootIDs   []string           `json:"blocking_physical_census_root_ids"`
}

type sourceRoot struct {
	ID                     string          `json:"id"`
	LogicalRoot            string          `json:"logical_root"`
	RelativePath           string          `json:"relative_path"`
	ExistsAtObservation    bool            `json:"exists_at_observation"`
	Class                  string          `json:"class"`
	ScopeKind              string          `json:"scope_kind"`
	Nature                 string          `json:"nature"`
	SourceDecision         string          `json:"source_decision"`
	SourceDecisionReason   string          `json:"source_decision_reason"`
	PhysicalCensusRequired bool            `json:"physical_census_required"`
	GitObjectCoverage      string          `json:"git_object_coverage"`
	EvidenceRefs           json.RawMessage `json:"evidence_refs"`
	ObservationBatchID     string          `json:"observation_batch_id"`
	GitCensusStatus        string          `json:"git_census_status"`
	GitCensusStatusReason  string          `json:"git_census_status_reason"`
	PhysicalCensusStatus   string          `json:"physical_census_status"`
	PhysicalStatusReason   string          `json:"physical_census_status_reason"`
}

type sourceCollection struct {
	RootID             string         `json:"root_id"`
	MembersSHA256      string         `json:"members_sha256"`
	Members            []sourceMember `json:"members"`
	ObservationBatchID string         `json:"observation_batch_id"`
}

type sourceMember struct {
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

type subjectRef struct {
	Kind             string `json:"kind"`
	RootID           string `json:"root_id,omitempty"`
	CollectionRootID string `json:"collection_root_id,omitempty"`
	PathAlias        string `json:"path_alias,omitempty"`
}

type expansionDocument struct {
	DocumentKind              string           `json:"document_kind"`
	SchemaVersion             int              `json:"schema_version"`
	SourceV3                  sourceSeal       `json:"source_v3"`
	DigestContracts           digestContracts  `json:"digest_contracts"`
	LogicalReferenceSetSHA256 string           `json:"logical_reference_set_sha256"`
	SubjectSetSHA256          string           `json:"subject_set_sha256"`
	Counts                    expansionCounts  `json:"counts"`
	Decisions                 decisionCounts   `json:"decisions"`
	MembershipSeals           []membershipSeal `json:"membership_seals"`
	Subjects                  []subjectRef     `json:"subjects"`
}

type sourceSeal struct {
	DocumentKind  string `json:"document_kind"`
	SchemaVersion int    `json:"schema_version"`
	BytesSHA256   string `json:"bytes_sha256"`
}

type digestContracts struct {
	SourceV3Bytes     digestContract `json:"source_v3_bytes"`
	LogicalReferences digestContract `json:"logical_references"`
	CollectionMembers digestContract `json:"collection_members"`
	Subjects          digestContract `json:"subjects"`
}

type digestContract struct {
	Algorithm      string `json:"algorithm"`
	Domain         string `json:"domain"`
	DomainInDigest bool   `json:"domain_in_digest"`
}

type expansionCounts struct {
	SourceRoots          int `json:"source_roots"`
	LogicalReferences    int `json:"logical_references"`
	SimpleReferences     int `json:"simple_references"`
	CollectionReferences int `json:"collection_references"`
	CollectionMembers    int `json:"collection_members"`
	PhysicalSubjects     int `json:"physical_subjects"`
	HistoricalPresent    int `json:"historical_present"`
	HistoricalAbsent     int `json:"historical_absent"`
}

type decisionCounts struct {
	Include int `json:"incluir"`
	Exclude int `json:"excluir"`
	Pending int `json:"pendiente"`
}

type membershipSeal struct {
	RootID        string `json:"root_id"`
	MemberCount   int    `json:"member_count"`
	MembersSHA256 string `json:"members_sha256"`
}
