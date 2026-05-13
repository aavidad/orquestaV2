package orquestaruncontrol

type PauseRunCommandV0 struct {
	RunRef         string   `json:"run_ref"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type ResumeRunCommandV0 struct {
	RunRef         string   `json:"run_ref"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type StopRunCommandV0 struct {
	RunRef         string   `json:"run_ref"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	Forced         bool     `json:"forced,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type CancelRunCommandV0 struct {
	RunRef         string   `json:"run_ref"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	Forced         bool     `json:"forced,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type RecordRunCheckpointCommandV0 struct {
	RunRef         string   `json:"run_ref"`
	RequestedBy    string   `json:"requested_by,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

type CompleteRunControlCommandV0 struct {
	RunRef         string             `json:"run_ref"`
	TargetStatus   RunControlStatusV0 `json:"target_status"`
	RequestedBy    string             `json:"requested_by,omitempty"`
	Reason         string             `json:"reason,omitempty"`
	IdempotencyKey string             `json:"idempotency_key,omitempty"`
	EvidenceRefs   []string           `json:"evidence_refs,omitempty"`
}
