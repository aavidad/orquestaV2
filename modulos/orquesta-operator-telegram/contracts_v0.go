package orquestaoperatortelegram

const (
	SchemaVersionV0 = "operator_telegram.v0"

	CommandStatusV0  = "status"
	CommandQueueV0   = "queue"
	CommandLaunchV0  = "launch_task"
	CommandObserveV0 = "observe_goal"
	CommandMessageV0 = "director_message"
	CommandStopV0    = "stop"
	CommandHandoffV0 = "handoff"

	ErrTelegramUnauthorizedChatV0 = "telegram_unauthorized_chat"
	ErrTelegramCommandInvalidV0   = "telegram_command_invalid"
	ErrTelegramConfirmationV0     = "telegram_confirmation_required"
	ErrTelegramLinkMissingV0      = "telegram_inodo_bot_link_missing"
)

type ConfigV0 struct {
	Enabled             bool
	BotLinkRef          string
	TokenConfigured     bool
	AuthorizedChatRefs  []string
	RequireConfirmation bool
}

type UpdateV0 struct {
	UpdateRef string
	ChatRef   string
	Text      string
}

type CommandV0 struct {
	Kind        string
	TargetRef   string
	Arguments   string
	Confirmed   bool
	EvidenceRef string
}

type CommandDescriptorV0 struct {
	Kind                 string
	Aliases              []string
	TargetRefRequired    bool
	ArgumentsRequired    bool
	ConfirmationRequired bool
}

func CommandCatalogV0() []CommandDescriptorV0 {
	return []CommandDescriptorV0{
		{Kind: CommandStatusV0, Aliases: []string{"estado", "status"}},
		{Kind: CommandQueueV0, Aliases: []string{"cola", "queue"}},
		{Kind: CommandLaunchV0, Aliases: []string{"lanzar", "launch", "launch_task", "task"}, ArgumentsRequired: true},
		{Kind: CommandObserveV0, Aliases: []string{"observar", "observe", "observe_goal", "goal"}, TargetRefRequired: true},
		{Kind: CommandMessageV0, Aliases: []string{"mensaje", "msg", "director", "director_message"}, TargetRefRequired: true, ArgumentsRequired: true},
		{Kind: CommandStopV0, Aliases: []string{"detener", "stop", "control"}, TargetRefRequired: true, ConfirmationRequired: true},
		{Kind: CommandHandoffV0, Aliases: []string{"handoff", "resumen"}},
	}
}

type ResponseV0 struct {
	Status       string    `json:"status"`
	Summary      string    `json:"summary"`
	CommandKind  string    `json:"command_kind,omitempty"`
	EvidenceRefs []string  `json:"evidence_refs,omitempty"`
	Issues       []IssueV0 `json:"issues,omitempty"`
}

type IssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type OperatorPortsV0 interface {
	QueryStatusV0(targetRef string) (string, []string, error)
	QueryQueueV0(targetRef string) (string, []string, error)
	LaunchTaskV0(taskSpec string, evidenceRefs []string) (string, []string, error)
	ObserveGoalV0(goalRef string) (string, []string, error)
	SendDirectorMessageV0(targetRef string, body string, evidenceRefs []string) (string, []string, error)
	StopV0(targetRef string, evidenceRefs []string) (string, []string, error)
	HandoffV0(targetRef string) (string, []string, error)
}
