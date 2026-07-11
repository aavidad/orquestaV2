package orquestagoal

import "context"

const (
	GoalWorkSpecSchemaV0                     = "orquesta_goal_work_spec.v0"
	GoalWorkLaunchReceiptSchemaV0            = "orquesta_goal_launch_receipt.v0"
	GoalWorkResultSchemaV0                   = "orquesta_goal_work_result.v0"
	GoalWorkResultRepairReceiptSchemaV0      = "orquesta_goal_work_result_repair_receipt.v0"
	GoalWorkStateSchemaV0                    = "orquesta_goal_work_state.v0"
	GoalWorkRunMarkerSchemaV0                = "orquesta_goal_work_run_marker.v0"
	GoalRequiredTestAttestationSchemaV0      = "orquesta_goal_required_test_attestation.v0"
	GoalRequiredTestFinalSnapshotSchemaV0    = "orquesta_goal_required_test_final_snapshot.v0"
	GoalRequiredTestAttestationClaimSchemaV0 = "orquesta_goal_required_test_attestation_claim.v0"

	GoalDirectorKindRuntimeGoalV0 = "runtime_goal"
	GoalDirectorKindCodexGoalV0   = "codex_goal"

	GoalRuleEnforcementAdvisoryV0 = "advisory"
	GoalRuleEnforcementHardV0     = "hard"

	GoalStatusAcceptedV0 = "accepted"
	GoalStatusInvalidV0  = "invalid"
	GoalStatusRunningV0  = "running"
	GoalStatusCompleteV0 = "complete"
	GoalStatusBlockedV0  = "blocked"
)

const (
	GoalMaterializedArtifactStatusValidV0          = "valid"
	GoalMaterializedArtifactStatusInvalidV0        = "invalid"
	GoalMaterializedArtifactStatusPartialV0        = "partial"
	GoalMaterializedArtifactStatusNonPublishableV0 = "non_publishable"
)

const (
	ErrGoalRefRequiredV0                                   = "goal_ref_required"
	ErrGoalRefInvalidV0                                    = "goal_ref_invalid"
	ErrGoalObjectiveRequiredV0                             = "goal_objective_required"
	ErrGoalDirectorRequiredV0                              = "goal_director_kind_required"
	ErrGoalDirectorInvalidV0                               = "goal_director_kind_invalid"
	ErrGoalWriteSetRequiredV0                              = "goal_write_set_required"
	ErrGoalWriteSetInvalidV0                               = "goal_write_set_invalid"
	ErrGoalRefFieldInvalidV0                               = "goal_ref_field_invalid"
	ErrGoalRuleInvalidV0                                   = "goal_rule_invalid"
	ErrGoalSpecLimitExceededV0                             = "goal_spec_limit_exceeded"
	ErrGoalStatusInvalidV0                                 = "goal_status_invalid"
	ErrGoalClosureInvalidV0                                = "goal_closure_invalid"
	ErrGoalArtifactPathScopeV0                             = "goal_artifact_path_out_of_scope"
	ErrGoalMaterializedArtifactInvalidV0                   = "goal_materialized_artifact_invalid"
	ErrGoalChecklistIncompleteV0                           = "goal_checklist_incomplete"
	ErrGoalReworkPlanRequiredV0                            = "goal_rework_plan_required"
	ErrGoalResultJSONInvalidV0                             = "goal_result_json_invalid"
	ErrGoalRequiredTestAttestationMissingV0                = "goal_required_test_attestation_missing"
	ErrGoalRequiredTestAttestationFailedV0                 = "goal_required_test_attestation_failed"
	ErrGoalRequiredTestAttestorInfrastructureFailedV0      = "goal_required_test_attestor_infrastructure_failed"
	ErrGoalRequiredTestAttestationMismatchV0               = "goal_required_test_attestation_mismatch"
	ErrGoalRequiredTestAttestorUntrustedV0                 = "goal_required_test_attestor_untrusted"
	ErrGoalRequiredTestAttestationClaimedV0                = "goal_required_test_attestation_claimed"
	ErrGoalRequiredTestSnapshotMissingV0                   = "goal_required_test_final_snapshot_missing"
	ErrGoalRequiredTestSnapshotMismatchV0                  = "goal_required_test_final_snapshot_mismatch"
	ErrGoalRequiredAcceptanceCriterionAttestationMissingV0 = "goal_required_acceptance_criterion_attestation_missing"
)

const (
	GoalWorkSpecMaxStringBytesV0        = 32 * 1024
	GoalWorkSpecMaxCommandBytesV0       = 16 * 1024
	GoalWorkIssueDetailMaxBytesV0       = 4 * 1024
	GoalWorkSpecMaxListItemsV0          = 512
	GoalWorkSpecMaxProjectedJSONBytesV0 = 256 * 1024
)

type GoalWorkSpecV0 struct {
	SchemaVersion            string                   `json:"schema_version"`
	GoalRef                  string                   `json:"goal_ref"`
	RequestRef               string                   `json:"request_ref,omitempty"`
	RunRef                   string                   `json:"run_ref,omitempty"`
	ImplementerAgentRef      string                   `json:"implementer_agent_ref,omitempty"`
	ImplementerCredentialRef string                   `json:"implementer_credential_ref,omitempty"`
	ProjectRef               string                   `json:"project_ref,omitempty"`
	DomainRef                string                   `json:"domain_ref,omitempty"`
	WorkKind                 string                   `json:"work_kind,omitempty"`
	WorkProfileKind          string                   `json:"work_profile_kind,omitempty"`
	Objective                string                   `json:"objective"`
	DirectorKind             string                   `json:"director_kind"`
	ContextRefs              []GoalContextRefV0       `json:"context_refs,omitempty"`
	RuleRefs                 []GoalRuleRefV0          `json:"rule_refs,omitempty"`
	SkillRefs                []string                 `json:"skill_refs,omitempty"`
	WriteSet                 []GoalWriteScopeV0       `json:"write_set,omitempty"`
	WriteSetSHA256           string                   `json:"write_set_sha256,omitempty"`
	RequiredTests            []GoalRequiredTestV0     `json:"required_tests,omitempty"`
	AcceptanceCriteria       []string                 `json:"acceptance_criteria,omitempty"`
	ArtifactContracts        []GoalArtifactContractV0 `json:"artifact_contracts,omitempty"`
	EvidenceRefs             []string                 `json:"evidence_refs,omitempty"`
	Budget                   GoalBudgetV0             `json:"budget,omitempty"`
	ClosurePolicy            GoalClosurePolicyV0      `json:"closure_policy,omitempty"`
	ReworkPolicy             GoalReworkPolicyV0       `json:"rework_policy,omitempty"`
}

type GoalContextRefV0 struct {
	Kind     string `json:"kind,omitempty"`
	Ref      string `json:"ref"`
	Purpose  string `json:"purpose,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type GoalRuleRefV0 struct {
	Kind        string `json:"kind,omitempty"`
	Ref         string `json:"ref"`
	Enforcement string `json:"enforcement,omitempty"`
}

type GoalWriteScopeV0 struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose,omitempty"`
}

type GoalRequiredTestV0 struct {
	TestRef                string   `json:"test_ref"`
	CommandRef             string   `json:"command_ref,omitempty"`
	Command                string   `json:"command,omitempty"`
	CommandSHA256          string   `json:"command_sha256,omitempty"`
	DefinitionSHA256       string   `json:"definition_sha256,omitempty"`
	AcceptanceCriteria     []string `json:"acceptance_criteria,omitempty"`
	AcceptanceCriteriaRefs []string `json:"acceptance_criteria_refs,omitempty"`
	EvidenceRefs           []string `json:"evidence_refs,omitempty"`
}

type GoalArtifactContractV0 struct {
	ArtifactRef  string   `json:"artifact_ref"`
	ArtifactType string   `json:"artifact_type"`
	Required     bool     `json:"required,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type GoalBudgetV0 struct {
	TokenBudget       int `json:"token_budget,omitempty"`
	MaxRuntimeSeconds int `json:"max_runtime_seconds,omitempty"`
	MaxReworkGoals    int `json:"max_rework_goals,omitempty"`
	MaxSubgoals       int `json:"max_subgoals,omitempty"`
}

type GoalClosurePolicyV0 struct {
	RequireRequiredTests                      bool     `json:"require_required_tests,omitempty"`
	RequireArtifacts                          bool     `json:"require_artifacts,omitempty"`
	RequireArtifactPaths                      bool     `json:"require_artifact_paths,omitempty"`
	RequireMaterializedArtifacts              bool     `json:"require_materialized_artifacts,omitempty"`
	RequireChecklist                          bool     `json:"require_checklist,omitempty"`
	RequireReworkPlanForPartialArtifacts      bool     `json:"require_rework_plan_for_partial_artifacts,omitempty"`
	RequireDomainReceipt                      bool     `json:"require_domain_receipt,omitempty"`
	RequireIndependentRequiredTestAttestation bool     `json:"require_independent_required_test_attestation,omitempty"`
	RequiredAttestorTrustPolicyRef            string   `json:"required_attestor_trust_policy_ref,omitempty"`
	RequiredAcceptanceCriteriaRefs            []string `json:"required_acceptance_criteria_refs,omitempty"`
	RequiredEvidenceRefs                      []string `json:"required_evidence_refs,omitempty"`
}

type GoalReworkPolicyV0 struct {
	PreferNewGoal     bool `json:"prefer_new_goal,omitempty"`
	MaxReworkGoals    int  `json:"max_rework_goals,omitempty"`
	PreserveArtifacts bool `json:"preserve_artifacts,omitempty"`
}

type GoalWorkIssueV0 struct {
	Code   string `json:"code"`
	Field  string `json:"field,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type GoalLaunchReceiptV0 struct {
	SchemaVersion   string              `json:"schema_version"`
	Status          string              `json:"status"`
	GoalRef         string              `json:"goal_ref,omitempty"`
	ExternalGoalRef string              `json:"external_goal_ref,omitempty"`
	ContextBudget   GoalContextBudgetV0 `json:"context_budget,omitempty"`
	EvidenceRefs    []string            `json:"evidence_refs,omitempty"`
	Issues          []GoalWorkIssueV0   `json:"issues,omitempty"`
}

type GoalContextBudgetV0 struct {
	ContextBudgetTotalBytes  int64  `json:"context_budget_total_bytes,omitempty"`
	StaticPromptBytes        int64  `json:"static_prompt_bytes,omitempty"`
	QueriedContextBytes      int64  `json:"queried_context_bytes,omitempty"`
	MaterializedContextBytes int64  `json:"materialized_context_bytes,omitempty"`
	DynamicContextBytes      int64  `json:"dynamic_context_bytes,omitempty"`
	CodeContextCacheStatus   string `json:"code_context_cache_status,omitempty"`
}

type GoalWorkResultV0 struct {
	SchemaVersion         string                         `json:"schema_version"`
	Status                string                         `json:"status"`
	GoalRef               string                         `json:"goal_ref,omitempty"`
	ExternalGoalRef       string                         `json:"external_goal_ref,omitempty"`
	Summary               string                         `json:"summary,omitempty"`
	ContextBudget         GoalContextBudgetV0            `json:"context_budget,omitempty"`
	ArtifactRefs          []string                       `json:"artifact_refs,omitempty"`
	ArtifactPaths         []string                       `json:"artifact_paths,omitempty"`
	MaterializedArtifacts []GoalMaterializedArtifactV0   `json:"materialized_artifacts,omitempty"`
	Checklist             GoalWorkChecklistV0            `json:"checklist,omitempty"`
	RequiredTestResults   []GoalRequiredTestResultV0     `json:"required_test_results,omitempty"`
	DomainReceiptRefs     []string                       `json:"domain_receipt_refs,omitempty"`
	ReworkPlanRefs        []string                       `json:"rework_plan_refs,omitempty"`
	EvidenceRefs          []string                       `json:"evidence_refs,omitempty"`
	Issues                []GoalWorkIssueV0              `json:"issues,omitempty"`
	RepairReceipt         *GoalWorkResultRepairReceiptV0 `json:"repair_receipt,omitempty"`
}

// GoalWorkResultJSONDecodeV0 is the neutral outcome of reading a durable goal
// result. Disposition tells adapters whether the input was canonical,
// recoverably repaired, or structurally impossible to use.
type GoalWorkResultJSONDecodeV0 struct {
	Result        GoalWorkResultV0               `json:"result"`
	Disposition   string                         `json:"disposition"`
	Issues        []GoalWorkIssueV0              `json:"issues,omitempty"`
	RepairReceipt *GoalWorkResultRepairReceiptV0 `json:"repair_receipt,omitempty"`
}

type GoalWorkResultRepairReceiptV0 struct {
	SchemaVersion         string                                 `json:"schema_version"`
	OriginalSchemaVersion string                                 `json:"original_schema_version,omitempty"`
	OriginalRefHash       string                                 `json:"original_ref_hash"`
	Transformations       []GoalWorkResultRepairTransformationV0 `json:"transformations,omitempty"`
	EvidenceRefs          []string                               `json:"evidence_refs,omitempty"`
}

type GoalWorkResultRepairTransformationV0 struct {
	Field string `json:"field"`
	Kind  string `json:"kind"`
}

type GoalMaterializedArtifactV0 struct {
	ArtifactRef  string            `json:"artifact_ref,omitempty"`
	Path         string            `json:"path,omitempty"`
	ArtifactType string            `json:"artifact_type,omitempty"`
	Scope        string            `json:"scope,omitempty"`
	Status       string            `json:"status,omitempty"`
	EvidenceRefs []string          `json:"evidence_refs,omitempty"`
	Issues       []GoalWorkIssueV0 `json:"issues,omitempty"`
}

type GoalWorkChecklistV0 struct {
	ExpectedRefs  []string `json:"expected_refs,omitempty"`
	CompletedRefs []string `json:"completed_refs,omitempty"`
	MissingRefs   []string `json:"missing_refs,omitempty"`
	EvidenceRefs  []string `json:"evidence_refs,omitempty"`
}

type GoalWorkStateV0 struct {
	SchemaVersion   string                   `json:"schema_version"`
	StoreVersion    uint64                   `json:"store_version,omitempty"`
	RunRef          string                   `json:"run_ref"`
	GoalRef         string                   `json:"goal_ref"`
	ExternalGoalRef string                   `json:"external_goal_ref,omitempty"`
	Status          string                   `json:"status"`
	Spec            GoalWorkSpecV0           `json:"spec"`
	LaunchReceipt   GoalLaunchReceiptV0      `json:"launch_receipt"`
	ContextBudget   GoalContextBudgetV0      `json:"context_budget,omitempty"`
	LastResult      *GoalWorkResultV0        `json:"last_result,omitempty"`
	LastClosure     *GoalClosureValidationV0 `json:"last_closure,omitempty"`
	EvidenceRefs    []string                 `json:"evidence_refs,omitempty"`
}

type GoalWorkRunMarkerV0 struct {
	SchemaVersion   string               `json:"schema_version"`
	RunRef          string               `json:"run_ref"`
	GoalRef         string               `json:"goal_ref,omitempty"`
	ExternalGoalRef string               `json:"external_goal_ref,omitempty"`
	DirectorKind    string               `json:"director_kind,omitempty"`
	Status          string               `json:"status,omitempty"`
	Spec            *GoalWorkSpecV0      `json:"spec,omitempty"`
	LaunchReceipt   *GoalLaunchReceiptV0 `json:"launch_receipt,omitempty"`
	ContextBudget   GoalContextBudgetV0  `json:"context_budget,omitempty"`
	EvidenceRefs    []string             `json:"evidence_refs,omitempty"`
}

type GoalWorkStateListRequestV0 struct {
	RunRefs    []string `json:"run_refs,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	ActiveOnly bool     `json:"active_only,omitempty"`
	MaxItems   int      `json:"max_items,omitempty"`
}

type GoalWorkRunMarkerListRequestV0 struct {
	RunRefs    []string `json:"run_refs,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	ActiveOnly bool     `json:"active_only,omitempty"`
	MaxItems   int      `json:"max_items,omitempty"`
}

type GoalRequiredTestResultV0 struct {
	TestRef      string   `json:"test_ref"`
	Status       string   `json:"status"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

// GoalRequiredTestAttestationV0 is an immutable receipt emitted by an actor
// causally distinct from the implementation goal. Commands remain references
// and hashes: neither a result summary nor free text is closure authority.
type GoalRequiredTestAttestationV0 struct {
	SchemaVersion          string               `json:"schema_version"`
	AttestationRef         string               `json:"attestation_ref"`
	RunRef                 string               `json:"run_ref"`
	GoalRef                string               `json:"goal_ref"`
	FinalSnapshotRef       string               `json:"final_snapshot_ref"`
	CheckoutRef            string               `json:"checkout_ref"`
	RevisionRef            string               `json:"revision_ref"`
	WriteSetSHA256         string               `json:"write_set_sha256"`
	TestRef                string               `json:"test_ref"`
	CommandRef             string               `json:"command_ref,omitempty"`
	CommandSHA256          string               `json:"command_sha256"`
	DefinitionSHA256       string               `json:"definition_sha256"`
	Status                 string               `json:"status"`
	FailureCode            string               `json:"failure_code,omitempty"`
	ImplementerAgentRef    string               `json:"implementer_agent_ref"`
	AttestorAgentRef       string               `json:"attestor_agent_ref"`
	AttestorCredentialRef  string               `json:"attestor_credential_ref"`
	StartedAt              string               `json:"started_at"`
	FinishedAt             string               `json:"finished_at"`
	IsolatedEnvironmentRef string               `json:"isolated_environment_ref"`
	ExitCode               int                  `json:"exit_code"`
	HashesBefore           []GoalAttestedHashV0 `json:"hashes_before,omitempty"`
	HashesAfter            []GoalAttestedHashV0 `json:"hashes_after,omitempty"`
	EvidenceRefs           []string             `json:"evidence_refs,omitempty"`
}

type GoalAttestedHashV0 struct {
	Ref    string `json:"ref"`
	SHA256 string `json:"sha256"`
}

// GoalRequiredTestFinalSnapshotV0 is captured by a trusted adapter after the
// implementation delivery and before any independent required test starts.
type GoalRequiredTestFinalSnapshotV0 struct {
	SchemaVersion  string               `json:"schema_version"`
	SnapshotRef    string               `json:"snapshot_ref"`
	RunRef         string               `json:"run_ref"`
	GoalRef        string               `json:"goal_ref"`
	CheckoutRef    string               `json:"checkout_ref"`
	RevisionRef    string               `json:"revision_ref"`
	WriteSetSHA256 string               `json:"write_set_sha256"`
	Hashes         []GoalAttestedHashV0 `json:"hashes"`
	ObservedAt     string               `json:"observed_at"`
	EvidenceRefs   []string             `json:"evidence_refs"`
}

type GoalRequiredTestFinalSnapshotRequestV0 struct {
	RunRef         string             `json:"run_ref"`
	GoalRef        string             `json:"goal_ref"`
	WriteSet       []GoalWriteScopeV0 `json:"write_set"`
	WriteSetSHA256 string             `json:"write_set_sha256"`
}

type GoalRequiredTestAttestationRequestV0 struct {
	RunRef                   string                          `json:"run_ref"`
	GoalRef                  string                          `json:"goal_ref"`
	ImplementerAgentRef      string                          `json:"implementer_agent_ref"`
	ImplementerCredentialRef string                          `json:"implementer_credential_ref"`
	AttestorTrustPolicyRef   string                          `json:"attestor_trust_policy_ref"`
	FinalSnapshot            GoalRequiredTestFinalSnapshotV0 `json:"final_snapshot"`
	RequiredTests            []GoalRequiredTestV0            `json:"required_tests"`
}

type GoalRequiredTestAttestationQueryV0 struct {
	RunRef      string `json:"run_ref"`
	GoalRef     string `json:"goal_ref"`
	RevisionRef string `json:"revision_ref,omitempty"`
}

const (
	GoalRequiredTestAttestationClaimStatusPendingV0   = "pending"
	GoalRequiredTestAttestationClaimStatusCompletedV0 = "completed"
	GoalRequiredTestAttestationClaimStatusFailedV0    = "failed"
)

type GoalRequiredTestAttestationClaimV0 struct {
	SchemaVersion    string `json:"schema_version"`
	ClaimRef         string `json:"claim_ref"`
	RunRef           string `json:"run_ref"`
	GoalRef          string `json:"goal_ref"`
	RevisionRef      string `json:"revision_ref"`
	TestRef          string `json:"test_ref"`
	DefinitionSHA256 string `json:"definition_sha256"`
	Status           string `json:"status"`
	AttestationRef   string `json:"attestation_ref,omitempty"`
	ClaimedAt        string `json:"claimed_at"`
	CompletedAt      string `json:"completed_at,omitempty"`
	FailedAt         string `json:"failed_at,omitempty"`
	FailureCode      string `json:"failure_code,omitempty"`
}

type GoalRequiredTestAttestationClaimRequestV0 struct {
	RunRef           string `json:"run_ref"`
	GoalRef          string `json:"goal_ref"`
	RevisionRef      string `json:"revision_ref"`
	TestRef          string `json:"test_ref"`
	DefinitionSHA256 string `json:"definition_sha256"`
}

type GoalRequiredTestAttestationClaimResultV0 struct {
	Claim    GoalRequiredTestAttestationClaimV0 `json:"claim"`
	Acquired bool                               `json:"acquired"`
}

type GoalObservationRequestV0 struct {
	GoalRef         string `json:"goal_ref"`
	ExternalGoalRef string `json:"external_goal_ref,omitempty"`
}

type GoalClosureValidationV0 struct {
	Status                   string                                   `json:"status"`
	Accepted                 bool                                     `json:"accepted,omitempty"`
	NeedsRework              bool                                     `json:"needs_rework,omitempty"`
	EvidenceRefs             []string                                 `json:"evidence_refs,omitempty"`
	AttestationVerifications []GoalRequiredTestIdentityVerificationV0 `json:"attestation_verifications,omitempty"`
	Issues                   []GoalWorkIssueV0                        `json:"issues,omitempty"`
}

type GoalRequiredTestIdentityVerificationRequestV0 struct {
	RunRef                   string `json:"run_ref"`
	GoalRef                  string `json:"goal_ref"`
	RevisionRef              string `json:"revision_ref"`
	AttestationRef           string `json:"attestation_ref"`
	TestRef                  string `json:"test_ref"`
	ImplementerAgentRef      string `json:"implementer_agent_ref"`
	ImplementerCredentialRef string `json:"implementer_credential_ref"`
	AttestorAgentRef         string `json:"attestor_agent_ref"`
	AttestorCredentialRef    string `json:"attestor_credential_ref"`
	RequiredTrustPolicyRef   string `json:"required_trust_policy_ref"`
}

// GoalRequiredTestIdentityVerificationV0 is safe operator evidence emitted by
// a trusted adapter after it verifies both credentials and their independence.
type GoalRequiredTestIdentityVerificationV0 struct {
	AttestationRef          string   `json:"attestation_ref"`
	TestRef                 string   `json:"test_ref"`
	Verified                bool     `json:"verified"`
	Independent             bool     `json:"independent"`
	ImplementerPrincipalRef string   `json:"implementer_principal_ref"`
	AttestorPrincipalRef    string   `json:"attestor_principal_ref"`
	AttestorCredentialRef   string   `json:"attestor_credential_ref"`
	TrustPolicyRef          string   `json:"trust_policy_ref"`
	EvidenceRefs            []string `json:"evidence_refs"`
}

type GoalWorkLauncherPortV0 interface {
	LaunchGoalWorkV0(context.Context, GoalWorkSpecV0) (GoalLaunchReceiptV0, error)
}

type GoalWorkObservationPortV0 interface {
	ObserveGoalWorkV0(context.Context, GoalObservationRequestV0) (GoalWorkResultV0, error)
}

type GoalWorkClosureValidatorPortV0 interface {
	ValidateGoalWorkClosureV0(context.Context, GoalWorkSpecV0, GoalWorkResultV0) (GoalClosureValidationV0, error)
}

type GoalRequiredTestAttestorPortV0 interface {
	AttestGoalRequiredTestsV0(context.Context, GoalRequiredTestAttestationRequestV0) ([]GoalRequiredTestAttestationV0, error)
}

type GoalRequiredTestSpecBinderPortV0 interface {
	BindGoalRequiredTestSpecV0(context.Context, GoalWorkSpecV0) (GoalWorkSpecV0, error)
}

type GoalRequiredTestFinalSnapshotObserverPortV0 interface {
	CaptureGoalRequiredTestFinalSnapshotV0(context.Context, GoalRequiredTestFinalSnapshotRequestV0) (GoalRequiredTestFinalSnapshotV0, error)
}

type GoalRequiredTestIdentityVerifierPortV0 interface {
	VerifyGoalRequiredTestIdentityV0(context.Context, GoalRequiredTestIdentityVerificationRequestV0) (GoalRequiredTestIdentityVerificationV0, error)
}

type GoalRequiredTestAttestationReaderPortV0 interface {
	ListGoalRequiredTestAttestationsV0(context.Context, GoalRequiredTestAttestationQueryV0) ([]GoalRequiredTestAttestationV0, error)
}

type GoalRequiredTestFinalSnapshotReaderPortV0 interface {
	LoadGoalRequiredTestFinalSnapshotV0(context.Context, string, string) (GoalRequiredTestFinalSnapshotV0, error)
}

type GoalRequiredTestAttestationStorePortV0 interface {
	GoalRequiredTestAttestationReaderPortV0
	GoalRequiredTestFinalSnapshotReaderPortV0
	SaveGoalRequiredTestAttestationV0(context.Context, GoalRequiredTestAttestationV0) error
	FreezeGoalRequiredTestFinalSnapshotV0(context.Context, GoalRequiredTestFinalSnapshotV0) (GoalRequiredTestFinalSnapshotV0, error)
	AcquireGoalRequiredTestAttestationClaimV0(context.Context, GoalRequiredTestAttestationClaimRequestV0) (GoalRequiredTestAttestationClaimResultV0, error)
	CompleteGoalRequiredTestAttestationClaimV0(context.Context, GoalRequiredTestAttestationClaimV0, GoalRequiredTestAttestationV0) error
	FailGoalRequiredTestAttestationClaimV0(context.Context, GoalRequiredTestAttestationClaimV0, string) (GoalRequiredTestAttestationClaimV0, error)
}

type GoalWorkStateStorePortV0 interface {
	SaveGoalWorkStateV0(context.Context, GoalWorkStateV0) error
	LoadGoalWorkStateV0(context.Context, string) (GoalWorkStateV0, error)
}

type GoalWorkStateCASStorePortV0 interface {
	CompareAndSwapGoalWorkStateV0(context.Context, uint64, GoalWorkStateV0) (GoalWorkStateV0, error)
}

type GoalWorkStateCASConflictErrorV0 struct {
	RunRef          string
	ExpectedVersion uint64
	CurrentVersion  uint64
}

func (err GoalWorkStateCASConflictErrorV0) Error() string {
	return "goal_work_state_cas_conflict"
}

type GoalWorkRunMarkerStorePortV0 interface {
	SaveGoalWorkRunMarkerV0(context.Context, GoalWorkRunMarkerV0) error
	LoadGoalWorkRunMarkerV0(context.Context, string) (GoalWorkRunMarkerV0, error)
}

type GoalWorkRunMarkerListPortV0 interface {
	ListGoalWorkRunMarkersV0(context.Context, GoalWorkRunMarkerListRequestV0) ([]GoalWorkRunMarkerV0, error)
}

type GoalWorkStateListPortV0 interface {
	ListGoalWorkStatesV0(context.Context, GoalWorkStateListRequestV0) ([]GoalWorkStateV0, error)
}
