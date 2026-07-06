package orquestaoperatortelegram

import "strings"

type AdapterV0 struct {
	config ConfigV0
	ports  OperatorPortsV0
}

func NewAdapterV0(config ConfigV0, ports OperatorPortsV0) (AdapterV0, []IssueV0) {
	config.AuthorizedChatRefs = normalizeChatRefsV0(config.AuthorizedChatRefs)
	if issues := ValidateConfigV0(config); len(issues) > 0 {
		return AdapterV0{}, issues
	}
	return AdapterV0{config: config, ports: ports}, nil
}

func (adapter AdapterV0) HandleUpdateV0(update UpdateV0) ResponseV0 {
	if !ChatAuthorizedV0(adapter.config, update.ChatRef) {
		return ResponseV0{
			Status:  "blocked",
			Summary: "chat no autorizado",
			Issues:  []IssueV0{{Code: ErrTelegramUnauthorizedChatV0, Field: "chat_ref"}},
		}
	}
	command, issues := ParseCommandV0(update.Text)
	if len(issues) > 0 {
		return ResponseV0{Status: "invalid", Summary: "comando no reconocido", Issues: issues}
	}
	if command.Kind == CommandStopV0 && adapter.config.RequireConfirmation && !command.Confirmed {
		return ResponseV0{
			Status:      "blocked",
			Summary:     "confirmacion requerida para detener/controlar",
			CommandKind: command.Kind,
			Issues:      []IssueV0{{Code: ErrTelegramConfirmationV0, Field: "confirmation"}},
		}
	}
	summary, evidenceRefs, err := adapter.dispatchV0(command)
	if err != nil {
		return ResponseV0{
			Status:      "blocked",
			Summary:     "puerto de operador no disponible",
			CommandKind: command.Kind,
			Issues:      []IssueV0{{Code: "operator_port_error", Field: command.Kind}},
		}
	}
	return ResponseV0{
		Status:       "accepted",
		Summary:      compactPublicSummaryV0(summary),
		CommandKind:  command.Kind,
		EvidenceRefs: evidenceRefs,
	}
}

func (adapter AdapterV0) dispatchV0(command CommandV0) (string, []string, error) {
	evidenceRefs := []string{command.EvidenceRef}
	switch command.Kind {
	case CommandStatusV0:
		return adapter.ports.QueryStatusV0(command.TargetRef)
	case CommandQueueV0:
		return adapter.ports.QueryQueueV0(command.TargetRef)
	case CommandLaunchV0:
		return adapter.ports.LaunchTaskV0(command.Arguments, evidenceRefs)
	case CommandObserveV0:
		return adapter.ports.ObserveGoalV0(command.TargetRef)
	case CommandMessageV0:
		return adapter.ports.SendDirectorMessageV0(command.TargetRef, command.Arguments, evidenceRefs)
	case CommandStopV0:
		return adapter.ports.StopV0(command.TargetRef, evidenceRefs)
	default:
		return adapter.ports.HandoffV0(command.TargetRef)
	}
}

func compactPublicSummaryV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "ok"
	}
	if len([]rune(value)) <= 240 {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:240])) + "..."
}
