package orquestaappcodexstack

import (
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type CodexStackAgentRuntimeDetailConfigV0 struct {
	ReceiptStore   orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	RuntimeWorkDir string
}

type CodexStackAgentRuntimeDetailRequestV0 struct {
	RequestID    string `json:"request_id"`
	RunRef       string `json:"run_ref"`
	AgentRef     string `json:"agent_ref,omitempty"`
	IncludeLogs  bool   `json:"include_logs,omitempty"`
	MaxFileBytes int64  `json:"max_file_bytes,omitempty"`
}

type CodexStackAgentRuntimeDetailResponseV0 struct {
	Estado string                                   `json:"estado"`
	RunRef string                                   `json:"run_ref,omitempty"`
	Agents []CodexStackAgentRuntimeDetailAgentV0    `json:"agents"`
	Issues []CodexStackAgentRuntimeDetailIssueV0    `json:"issues,omitempty"`
	Files  []CodexStackAgentRuntimeDetailFileV0     `json:"files,omitempty"`
	Skills CodexStackAgentRuntimeDetailSkillHintsV0 `json:"skills,omitempty"`
}

type CodexStackAgentRuntimeDetailAgentV0 struct {
	RunRef        string                                                `json:"run_ref"`
	AgentRef      string                                                `json:"agent_ref"`
	RuntimeRef    string                                                `json:"runtime_ref,omitempty"`
	DescriptorRef string                                                `json:"descriptor_ref,omitempty"`
	Task          CodexStackAgentRuntimeDetailTaskV0                    `json:"task"`
	Descriptor    orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 `json:"-"`
	AgentPacket   orquestaruntime.AgentStartPacketV0                    `json:"-"`
	Ack           map[string]any                                        `json:"-"`
	PromptText    string                                                `json:"-"`
	Skills        CodexStackAgentRuntimeDetailSkillHintsV0              `json:"skills"`
	Files         []CodexStackAgentRuntimeDetailFileV0                  `json:"files"`
}

type CodexStackAgentRuntimeDetailTaskV0 struct {
	TaskRef            string   `json:"task_ref,omitempty"`
	Title              string   `json:"title,omitempty"`
	ObjectiveSummary   string   `json:"objective_summary,omitempty"`
	WriteSet           []string `json:"write_set,omitempty"`
	RequiredTests      []string `json:"required_tests,omitempty"`
	ContextRefsCount   int      `json:"context_refs_count,omitempty"`
	RequiredTestsCount int      `json:"required_tests_count,omitempty"`
	WriteSetCount      int      `json:"write_set_count,omitempty"`
}

type CodexStackAgentRuntimeDetailSkillHintsV0 struct {
	CavemanRequested bool     `json:"caveman_requested"`
	CompactProtocol  bool     `json:"compact_protocol"`
	Policies         []string `json:"policies,omitempty"`
	CapacityLevel    string   `json:"capacity_level,omitempty"`
	Phase            string   `json:"phase,omitempty"`
	MaxChildAgents   int      `json:"max_child_agents,omitempty"`
}

type CodexStackAgentRuntimeDetailFileV0 struct {
	Name           string         `json:"name"`
	FileKind       string         `json:"file_kind"`
	Exists         bool           `json:"exists"`
	SizeBytes      int64          `json:"size_bytes,omitempty"`
	Truncated      bool           `json:"truncated,omitempty"`
	RedactionLevel string         `json:"redaction_level"`
	ReasonCodes    []string       `json:"reason_codes,omitempty"`
	ContentRef     string         `json:"content_ref,omitempty"`
	SHA256         string         `json:"sha256,omitempty"`
	Preview        string         `json:"preview,omitempty"`
	Extracts       map[string]any `json:"extracts,omitempty"`
	Content        string         `json:"-"`
}

type CodexStackAgentRuntimeDetailIssueV0 struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
