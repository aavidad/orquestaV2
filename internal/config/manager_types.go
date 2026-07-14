package config

import "time"

// ManagerOptions binds the neutral configuration service to one document
// store and one immutable bootstrap snapshot.
type ManagerOptions struct {
	Store       DocumentStore
	Active      Snapshot
	Environment map[string]string
	SourcePath  string
	// ReservedPaths are opaque storage paths which desired runtime paths must
	// not overlap. The concrete DocumentStore composition owns this list.
	ReservedPaths []string
	Now           func() time.Time
}

// ConfigKeyView is one registry-ordered, redacted comparison between active
// bootstrap state and desired persisted state.
type ConfigKeyView struct {
	Key             Key    `json:"key"`
	ActiveValue     any    `json:"active_value"`
	DesiredValue    any    `json:"desired_value"`
	Source          Source `json:"source"`
	Type            string `json:"type"`
	Sensitive       bool   `json:"sensitive"`
	Scope           string `json:"scope"`
	RestartRequired bool   `json:"restart_required"`
	PendingRestart  bool   `json:"pending_restart"`
}

// ConfigView describes current source identity and desired versus active
// configuration without exposing credential references.
type ConfigView struct {
	SourceRevision     Revision        `json:"source_revision"`
	ActiveHash         string          `json:"active_hash"`
	DesiredHash        string          `json:"desired_hash"`
	PendingRestart     bool            `json:"pending_restart"`
	PendingRestartKeys []Key           `json:"pending_restart_keys"`
	Keys               []ConfigKeyView `json:"keys"`
}

// Change sets one canonical explicit value or removes it from the human TOML.
type Change struct {
	Key   Key  `json:"key"`
	Value any  `json:"value,omitempty"`
	Unset bool `json:"unset,omitempty"`
}

// UpdateRequest is one confirmed, expected-revision configuration mutation.
type UpdateRequest struct {
	ActorRef         string   `json:"actor_ref"`
	RequestRef       string   `json:"request_ref"`
	ExpectedRevision Revision `json:"expected_revision"`
	Confirm          bool     `json:"confirm"`
	Changes          []Change `json:"changes"`
}

// UpdateResult returns the post-commit view and immutable store receipt.
type UpdateResult struct {
	View     ConfigView    `json:"view"`
	Receipt  ChangeReceipt `json:"receipt"`
	Replayed bool          `json:"replayed"`
}

// DoctorMode is a non-mutating registry proposal classification.
type DoctorMode string

const (
	DoctorReuse   DoctorMode = "reuse"
	DoctorReplace DoctorMode = "replace"
	DoctorNew     DoctorMode = "new"
)

// DoctorRequest contains candidate registry changes. Proposals deliberately
// carry no Mode; Doctor derives it from target and retirement shape.
type DoctorRequest struct {
	Proposals []DoctorProposal `json:"proposals"`
}

// DoctorProposal describes a possible reuse, replacement or new canonical
// key without mutating the registry.
type DoctorProposal struct {
	Key                 Key    `json:"key"`
	TargetKey           Key    `json:"target_key,omitempty"`
	SemanticRef         string `json:"semantic_ref"`
	Alias               string `json:"alias,omitempty"`
	RemoveAfterRevision string `json:"remove_after_revision,omitempty"`
	GoName              string `json:"go_name,omitempty"`
	EnvAlias            string `json:"env_alias,omitempty"`
	Type                string `json:"type,omitempty"`
	Scope               string `json:"scope,omitempty"`
}

// DoctorDecision is one accepted classification.
type DoctorDecision struct {
	DoctorProposal
	Mode DoctorMode `json:"mode"`
}

// DoctorConflictCode is stable diagnostic data for registry tooling.
type DoctorConflictCode string

const (
	DoctorConflictShape      DoctorConflictCode = "doctor_shape_conflict"
	DoctorConflictKey        DoctorConflictCode = "doctor_key_conflict"
	DoctorConflictTarget     DoctorConflictCode = "doctor_target_conflict"
	DoctorConflictSemantic   DoctorConflictCode = "doctor_semantic_conflict"
	DoctorConflictGoName     DoctorConflictCode = "doctor_go_name_conflict"
	DoctorConflictEnvAlias   DoctorConflictCode = "doctor_environment_conflict"
	DoctorConflictAlias      DoctorConflictCode = "doctor_alias_conflict"
	DoctorConflictRetirement DoctorConflictCode = "doctor_retirement_conflict"
	DoctorConflictType       DoctorConflictCode = "doctor_type_conflict"
	DoctorConflictScope      DoctorConflictCode = "doctor_scope_conflict"
)

// DoctorConflict identifies one proposal field that collides with canonical
// registry state or an earlier accepted proposal.
type DoctorConflict struct {
	ProposalIndex int                `json:"proposal_index"`
	Key           Key                `json:"key"`
	Field         string             `json:"field"`
	Code          DoctorConflictCode `json:"code"`
	ExistingKey   Key                `json:"existing_key,omitempty"`
}

// DoctorReport preserves request order for accepted decisions and conflicts.
type DoctorReport struct {
	Accepted  []DoctorDecision `json:"accepted"`
	Conflicts []DoctorConflict `json:"conflicts"`
}
