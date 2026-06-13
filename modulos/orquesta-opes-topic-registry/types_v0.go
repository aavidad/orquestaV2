package orquestaopestopicregistry

import "context"

const (
	TopicRegistryUpdateResultSchemaV0 = "opes_topic_registry_update_result.v0"

	TopicRegistryUpdateStatusAppliedV0 = "applied"
	TopicRegistryUpdateStatusInvalidV0 = "invalid"
	TopicRegistryUpdateStatusFailedV0  = "failed"

	TopicRegistryActionUpdateV0  = "update"
	TopicRegistryActionReleaseV0 = "release"

	DefaultTopicRegistryAgentIDV0 = "orquesta-opes-topic-registry"

	ErrTopicRegistryToolPathRequiredV0 = "opes_topic_registry_tool_path_required"
	ErrTopicRegistryCourseIDRequiredV0 = "opes_topic_registry_course_id_required"
	ErrTopicRegistryTopicIDRequiredV0  = "opes_topic_registry_topic_id_required"
	ErrTopicRegistryActionInvalidV0    = "opes_topic_registry_action_invalid"
	ErrTopicRegistryRunnerRequiredV0   = "opes_topic_registry_runner_required"
	ErrTopicRegistryCommandFailedV0    = "opes_topic_registry_command_failed"
)

type TopicRegistryUpdateRequestV0 struct {
	ToolPath     string
	Action       string
	CourseID     string
	TopicID      string
	AgentID      string
	Status       string
	Summary      string
	Done         string
	Pending      string
	EvidenceRefs []string
	Force        bool
}

type TopicRegistryUpdateResultV0 struct {
	SchemaVersion string                 `json:"schema_version"`
	Status        string                 `json:"status"`
	Action        string                 `json:"action,omitempty"`
	CourseID      string                 `json:"course_id,omitempty"`
	TopicID       string                 `json:"topic_id,omitempty"`
	CommandArgs   []string               `json:"command_args,omitempty"`
	EvidenceRefs  []string               `json:"evidence_refs,omitempty"`
	Issues        []TopicRegistryIssueV0 `json:"issues,omitempty"`
}

type TopicRegistryIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type TopicRegistryCommandInvocationV0 struct {
	ToolPath string
	Args     []string
}

type TopicRegistryCommandResultV0 struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type TopicRegistryCommandRunnerPortV0 interface {
	RunTopicRegistryCommandV0(
		context.Context,
		TopicRegistryCommandInvocationV0,
	) (TopicRegistryCommandResultV0, error)
}
