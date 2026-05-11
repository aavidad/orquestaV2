package orquestadirectoragent

type DirectorAgentCompactStatsV0 struct {
	SchemaVersion string                     `json:"schema_version"`
	RunID         string                     `json:"run_id"`
	Status        string                     `json:"status"`
	CurrentPhase  string                     `json:"current_phase"`
	Totals        DirectorAgentStatsTotalsV0 `json:"totals"`
	PendingRefs   []string                   `json:"pending_refs,omitempty"`
	EvidenceRefs  []string                   `json:"evidence_refs,omitempty"`
}

type DirectorAgentStatsTotalsV0 struct {
	Phases           int `json:"phases"`
	Tasks            int `json:"tasks"`
	CapacityRequests int `json:"capacity_requests"`
	Agents           int `json:"agents"`
	Deliveries       int `json:"deliveries"`
	ReviewResults    int `json:"review_results"`
	ReworkRequests   int `json:"rework_requests"`
	ReplanDecisions  int `json:"replan_decisions"`
	ClosedTasks      int `json:"closed_tasks"`
	Validations      int `json:"validations"`
	Closures         int `json:"closures"`
}
