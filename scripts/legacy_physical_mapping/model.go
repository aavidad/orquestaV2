// Este fichero declara contratos JSON compactos sin identidades físicas.
package main

const (
	expectedV3SHA         = "sha256:0feffd7e493bb643bd6dbfa393d8480688f73f7ef452dbc73fbd0e23ede02044"
	expectedUniverseSHA   = "sha256:b81478cef265fbb3970925cd09d450b23cd42196858e271c7b26a229fa106f1e"
	expectedReferences    = "sha256:2d2b81b27bfccad5a0b3a6dfabdec299a516dc3b31400f2cb2ecd6948c1e7834"
	expectedSubjects      = "sha256:405f5b68f0e886a40bb69751fbb62f60567a15d6879393e9f42c4d1b5a458cc4"
	contextDigestDomain   = "orquesta.legacy-private-mapping-context.v1"
	candidateDigestDomain = "orquesta.legacy-private-mapping-semantic-integrity.v1"
	maxBaseBytes          = 1 << 20
	maxCandidateBytes     = 2 << 20
	maxVerdictBytes       = 16 << 10
	maxJSONDepth          = 12
	maxJSONTokens         = 50000
	maxArrayItems         = 1024
	maxStringBytes        = 256
)

type subjectRef struct {
	Kind             string `json:"kind"`
	RootID           string `json:"root_id,omitempty"`
	CollectionRootID string `json:"collection_root_id,omitempty"`
	PathAlias        string `json:"path_alias,omitempty"`
}
type sourceDocument struct {
	Roots                         []sourceRoot       `json:"roots"`
	Collections                   []sourceCollection `json:"collections"`
	BlockingPhysicalCensusRootIDs []string           `json:"blocking_physical_census_root_ids"`
}
type sourceRoot struct {
	ID                  string `json:"id"`
	ExistsAtObservation bool   `json:"exists_at_observation"`
	ScopeKind           string `json:"scope_kind"`
	SourceDecision      string `json:"source_decision"`
}
type sourceCollection struct {
	RootID  string         `json:"root_id"`
	Members []sourceMember `json:"members"`
}
type sourceMember struct {
	PathAlias           string `json:"path_alias"`
	ExistsAtObservation bool   `json:"exists_at_observation"`
}
type universeDocument struct {
	Subjects []subjectRef `json:"subjects"`
}
type mappingContext struct {
	ViewRef             string `json:"view_ref"`
	ViewEvidenceSHA256  string `json:"view_evidence_sha256"`
	ViewState           string `json:"view_state"`
	FenceRef            string `json:"fence_ref"`
	FenceGeneration     uint64 `json:"fence_generation"`
	FenceEvidenceSHA256 string `json:"fence_evidence_sha256"`
	FenceState          string `json:"fence_state"`
	PolicyRef           string `json:"policy_ref"`
	PolicySHA256        string `json:"policy_sha256"`
	ConfigurationSHA256 string `json:"configuration_sha256"`
	AttemptRef          string `json:"attempt_ref"`
	WindowStartedAt     string `json:"window_started_at"`
	WindowEndedAt       string `json:"window_ended_at"`
}
type mappingObservation struct {
	Subject               subjectRef `json:"subject"`
	ExistsAtV3Observation bool       `json:"exists_at_v3_observation"`
	PresentInStableView   bool       `json:"present_in_stable_view"`
	ObservedType          string     `json:"observed_type"`
	ObservationOutcome    string     `json:"observation_outcome"`
	ObservedAt            string     `json:"observed_at"`
	IdentityRef           string     `json:"identity_ref,omitempty"`
	ReopenEvidenceSHA256  string     `json:"reopen_evidence_sha256,omitempty"`
	BindingRef            string     `json:"binding_ref,omitempty"`
	AbsenceEvidenceRef    string     `json:"absence_evidence_ref,omitempty"`
	AbsenceEvidenceSHA256 string     `json:"absence_evidence_sha256,omitempty"`
}
type ownershipDeclaration struct {
	BindingRef         string       `json:"binding_ref"`
	OwnerSubject       subjectRef   `json:"owner_subject"`
	AliasSubjects      []subjectRef `json:"alias_subjects"`
	OverlapSubjects    []subjectRef `json:"overlap_subjects"`
	PrunedFromSubjects []subjectRef `json:"pruned_from_subjects"`
}
type mappingCandidate struct {
	DocumentKind              string                 `json:"document_kind"`
	SchemaVersion             int                    `json:"schema_version"`
	SourceV3BytesSHA256       string                 `json:"source_v3_bytes_sha256"`
	UniverseBytesSHA256       string                 `json:"universe_bytes_sha256"`
	LogicalReferenceSetSHA256 string                 `json:"logical_reference_set_sha256"`
	SubjectSetSHA256          string                 `json:"subject_set_sha256"`
	Context                   mappingContext         `json:"context"`
	ContextSHA256             string                 `json:"context_sha256"`
	Observations              []mappingObservation   `json:"observations"`
	Ownership                 []ownershipDeclaration `json:"ownership"`
	SemanticIntegritySHA256   string                 `json:"semantic_integrity_sha256"`
}
type verdictCounts struct {
	LogicalReferences int `json:"logical_references"`
	SimpleReferences  int `json:"simple_references"`
	Collections       int `json:"collection_references"`
	CollectionMembers int `json:"collection_members"`
	Subjects          int `json:"physical_subjects"`
	HistoricalPresent int `json:"historical_present"`
	HistoricalAbsent  int `json:"historical_absent"`
	StablePresent     int `json:"stable_present"`
	StableAbsent      int `json:"stable_absent"`
	Regular           int `json:"regular"`
	Directories       int `json:"directories"`
	Bindings          int `json:"bindings"`
	Aliases           int `json:"aliases"`
	Overlaps          int `json:"overlaps"`
	Prunes            int `json:"prunes"`
}
type validationVerdict struct {
	DocumentKind                     string        `json:"document_kind"`
	SchemaVersion                    int           `json:"schema_version"`
	Decision                         string        `json:"decision"`
	Authority                        string        `json:"authority"`
	SourceV3BytesSHA256              string        `json:"source_v3_bytes_sha256"`
	UniverseBytesSHA256              string        `json:"universe_bytes_sha256"`
	CandidateBytesSHA256             string        `json:"candidate_bytes_sha256"`
	CandidateSemanticIntegritySHA256 string        `json:"candidate_semantic_integrity_sha256"`
	ContextSHA256                    string        `json:"context_sha256"`
	Counts                           verdictCounts `json:"counts"`
}
