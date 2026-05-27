package orquestaautoprogramming

const (
	AutoprogrammingPromotionGuardianReceiptSchemaVersionV0 = "promotion_guardian_receipt.v0"

	AutoprogrammingPromotionGuardianReceiptCandidateVerifiedV0           = "candidate_verified"
	AutoprogrammingPromotionGuardianReceiptCandidatePromotedV0           = "candidate_promoted"
	AutoprogrammingPromotionGuardianReceiptCandidatePromotedBreakglassV0 = "candidate_promoted_breakglass"
	AutoprogrammingPromotionGuardianReceiptPromotionBlockedV0            = "promotion_blocked"
	AutoprogrammingPromotionGuardianReceiptLastGoodRestoredV0            = "last_good_restored"
	AutoprogrammingPromotionGuardianReceiptResultInvalidV0               = "guardian_result_invalid"
)

type AutoprogrammingPromotionGuardianReceiptV0 struct {
	SchemaVersion         string   `json:"schema_version"`
	ReceiptRef            string   `json:"receipt_ref"`
	Status                string   `json:"status"`
	PromotionRef          string   `json:"promotion_ref,omitempty"`
	RunRef                string   `json:"run_ref,omitempty"`
	ProjectRef            string   `json:"project_ref,omitempty"`
	AppRef                string   `json:"app_ref,omitempty"`
	RepoRef               string   `json:"repo_ref,omitempty"`
	WorktreeRef           string   `json:"worktree_ref,omitempty"`
	BranchRef             string   `json:"branch_ref,omitempty"`
	GuardianAttemptRef    string   `json:"guardian_attempt_ref,omitempty"`
	GuardianResultStatus  string   `json:"guardian_result_status,omitempty"`
	GuardianResultPhase   string   `json:"guardian_result_phase,omitempty"`
	GuardianPromote       bool     `json:"guardian_promote"`
	GuardianPromoted      bool     `json:"guardian_promoted,omitempty"`
	GuardianRestored      bool     `json:"guardian_restored,omitempty"`
	ManifestRef           string   `json:"manifest_ref,omitempty"`
	RepairPacketRef       string   `json:"repair_packet_ref,omitempty"`
	CandidateHash         string   `json:"candidate_hash,omitempty"`
	CandidateSizeBytes    int64    `json:"candidate_size_bytes,omitempty"`
	StagingEffectStatus   string   `json:"staging_effect_status,omitempty"`
	StagingCommitRef      string   `json:"staging_commit_ref,omitempty"`
	StagingCommitShortRef string   `json:"staging_commit_short_ref,omitempty"`
	StagingChangedPaths   []string `json:"staging_changed_paths,omitempty"`
	Retryable             bool     `json:"retryable,omitempty"`
	ReasonCodes           []string `json:"reason_codes,omitempty"`
	EvidenceRefs          []string `json:"evidence_refs,omitempty"`
}
