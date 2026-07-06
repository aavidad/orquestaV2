package orquestaoperatortelegram

import "strings"

func ParseCommandV0(text string) (CommandV0, []IssueV0) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "/") {
		text = strings.TrimPrefix(text, "/")
	}
	if text == "" {
		return CommandV0{}, []IssueV0{{Code: ErrTelegramCommandInvalidV0, Field: "text"}}
	}
	fields := strings.Fields(text)
	verb := strings.ToLower(strings.TrimPrefix(fields[0], "orquesta_"))
	var rest string
	if len(fields) > 1 {
		rest = strings.TrimSpace(strings.TrimPrefix(text, fields[0]))
	}
	command := CommandV0{Arguments: rest}
	switch verb {
	case "estado", "status":
		command.Kind = CommandStatusV0
		command.TargetRef = firstFieldV0(rest)
	case "cola", "queue":
		command.Kind = CommandQueueV0
		command.TargetRef = firstFieldV0(rest)
	case "lanzar", "launch", "launch_task", "task":
		command.Kind = CommandLaunchV0
		command.Arguments = strings.TrimSpace(rest)
	case "observar", "observe", "observe_goal", "goal":
		command.Kind = CommandObserveV0
		command.TargetRef = firstFieldV0(rest)
	case "mensaje", "msg", "director", "director_message":
		command.Kind = CommandMessageV0
		command.TargetRef = firstFieldV0(rest)
		command.Arguments = strings.TrimSpace(strings.TrimPrefix(rest, command.TargetRef))
	case "detener", "stop", "control":
		command.Kind = CommandStopV0
		command.TargetRef = firstFieldV0(rest)
		command.Confirmed = hasConfirmationV0(rest)
	case "handoff", "resumen":
		command.Kind = CommandHandoffV0
		command.TargetRef = firstFieldV0(rest)
	default:
		return CommandV0{}, []IssueV0{{Code: ErrTelegramCommandInvalidV0, Field: "command"}}
	}
	command.EvidenceRef = "evidence-ref-telegram-command-" + command.Kind
	return command, nil
}

func firstFieldV0(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func hasConfirmationV0(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "confirm=") || strings.Contains(value, "confirmar")
}
