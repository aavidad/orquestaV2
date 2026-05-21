package orquestaruntime

import orquestacontext "orquesta/modulos/orquesta-context"

const AgentStartPacketSchemaVersionV0 = "agent_start_packet.v0"

const (
	AgentStartPacketInvalidoV0                      RuntimeLaunchErrorCodeV0 = "agent_start_packet_invalido"
	AgentStartPacketContextMaterializadoRequeridoV0 RuntimeLaunchErrorCodeV0 = "context_materialized_bundle_requerido"
	AgentStartPacketContextMaterializadoInvalidoV0  RuntimeLaunchErrorCodeV0 = "context_materialized_bundle_invalido"
)

type AgentStartPacketV0 struct {
	SchemaVersion string                                      `json:"schema_version"`
	RequestID     string                                      `json:"request_id"`
	CorrelationID string                                      `json:"correlation_id"`
	WorkOrderRef  string                                      `json:"work_order_ref"`
	TargetModule  string                                      `json:"target_module"`
	Phase         string                                      `json:"phase"`
	CapacityLevel string                                      `json:"capacity_level"`
	Locale        string                                      `json:"locale"`
	Task          AgentStartTaskV0                            `json:"task"`
	Context       orquestacontext.ContextMaterializedBundleV0 `json:"context"`
	DeliveryRefs  AgentStartDeliveryRefsV0                    `json:"delivery_refs"`
	Policies      []string                                    `json:"policies"`
	Issues        []RuntimeLaunchErrorV0                      `json:"issues,omitempty"`
}

type AgentStartTaskV0 struct {
	TaskRef         string   `json:"task_ref"`
	Priority        string   `json:"priority"`
	Title           string   `json:"title"`
	Objective       string   `json:"objective"`
	TargetSymbol    string   `json:"target_symbol"`
	WriteSet        []string `json:"write_set"`
	RequiredTests   []string `json:"required_tests"`
	DoneCriteria    []string `json:"done_criteria"`
	ParentTaskRef   string   `json:"parent_task_ref,omitempty"`
	CohortRef       string   `json:"cohort_ref,omitempty"`
	WaveRef         string   `json:"wave_ref,omitempty"`
	DelegationDepth int      `json:"delegation_depth,omitempty"`
	MaxChildAgents  int      `json:"max_child_agents,omitempty"`
	ChildTaskRefs   []string `json:"child_task_refs,omitempty"`
}

type AgentStartDeliveryRefsV0 struct {
	MailboxRef    string `json:"mailbox_ref"`
	AckRef        string `json:"ack_ref"`
	ReadinessRef  string `json:"readiness_ref"`
	CheckpointRef string `json:"checkpoint_ref,omitempty"`
}

func (packet AgentStartPacketV0) Valid() bool {
	return packet.SchemaVersion == AgentStartPacketSchemaVersionV0 &&
		packet.RequestID != "" &&
		packet.WorkOrderRef != "" &&
		packet.Context.Valid() &&
		len(packet.Issues) == 0
}
