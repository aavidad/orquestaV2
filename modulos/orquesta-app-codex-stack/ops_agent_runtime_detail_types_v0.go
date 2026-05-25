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
	RunRef      string                                                `json:"run_ref"`
	AgentRef    string                                                `json:"agent_ref"`
	RuntimeDir  string                                                `json:"runtime_dir,omitempty"`
	Descriptor  orquestaruntimecodexdelivery.CodexReceiptDescriptorV0 `json:"descriptor"`
	AgentPacket orquestaruntime.AgentStartPacketV0                    `json:"agent_packet"`
	Ack         map[string]any                                        `json:"ack,omitempty"`
	PromptText  string                                                `json:"prompt_text,omitempty"`
	Skills      CodexStackAgentRuntimeDetailSkillHintsV0              `json:"skills"`
	Files       []CodexStackAgentRuntimeDetailFileV0                  `json:"files"`
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
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Exists    bool   `json:"exists"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
}

type CodexStackAgentRuntimeDetailIssueV0 struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}
