package orquestaoperatormcp

type OperatorMCPStatusPortV0 interface {
	QueryOperatorStatusV0(OperatorStatusQueryV0) (OperatorMCPStatusResultV0, error)
}

type OperatorMCPBurstPortV0 interface {
	RequestOperatorSupervisedBurstV0(OperatorSupervisedBurstRequestV0) (OperatorMCPBurstResultV0, error)
}

type OperatorMCPOutboxPortV0 interface {
	ListOperatorPendingOutboxV0(OperatorPendingOutboxQueryV0) (OperatorMCPOutboxResultV0, error)
}

type OperatorMCPDirectedQueryPortV0 interface {
	RaiseOperatorDirectedQueryV0(OperatorDirectedQueryV0) (OperatorMCPDirectedQueryResultV0, error)
}

type OperatorMCPConnectorV0 interface {
	OperatorMCPStatusPortV0
	OperatorMCPBurstPortV0
	OperatorMCPOutboxPortV0
	OperatorMCPDirectedQueryPortV0
}

type OperatorMCPStatusResultV0 struct {
	Status       string   `json:"status"`
	Summary      string   `json:"summary,omitempty"`
	Sections     []string `json:"sections,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type OperatorMCPBurstResultV0 struct {
	BurstRef      string   `json:"burst_ref"`
	ExecutedSteps int      `json:"executed_steps"`
	FinalAction   string   `json:"final_action,omitempty"`
	TraceRefs     []string `json:"trace_refs,omitempty"`
}

type OperatorMCPOutboxItemV0 struct {
	MessageRef string `json:"message_ref"`
	Kind       string `json:"kind,omitempty"`
	TargetRef  string `json:"target_ref,omitempty"`
}

type OperatorMCPOutboxResultV0 struct {
	WatermarkRef string                    `json:"watermark_ref,omitempty"`
	PendingCount int                       `json:"pending_count"`
	Items        []OperatorMCPOutboxItemV0 `json:"items,omitempty"`
	EvidenceRefs []string                  `json:"evidence_refs,omitempty"`
}

type OperatorMCPDirectedQueryResultV0 struct {
	Accepted   bool     `json:"accepted"`
	AnswerRef  string   `json:"answer_ref,omitempty"`
	NextAction string   `json:"next_action,omitempty"`
	TraceRefs  []string `json:"trace_refs,omitempty"`
}
