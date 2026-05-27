package orquestamcp

import (
	"encoding/json"
	"strings"
)

type MCPAutoprogrammingOperatorAdviceV0 struct {
	AdviceRef    string   `json:"advice_ref,omitempty"`
	OperatorRef  string   `json:"operator_ref,omitempty"`
	TargetRef    string   `json:"target_ref,omitempty"`
	SubjectRef   string   `json:"subject_ref,omitempty"`
	Ref          string   `json:"ref,omitempty"`
	Target       string   `json:"target,omitempty"`
	RunRef       string   `json:"run_ref,omitempty"`
	Run          string   `json:"run,omitempty"`
	TaskRef      string   `json:"task_ref,omitempty"`
	Task         string   `json:"task,omitempty"`
	Action       string   `json:"action,omitempty"`
	Kind         string   `json:"kind,omitempty"`
	Message      string   `json:"message,omitempty"`
	Advice       string   `json:"advice,omitempty"`
	Text         string   `json:"text,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	NonBlocking  bool     `json:"non_blocking"`
}

type mcpAutoprogrammingOperatorAdviceListV0 []MCPAutoprogrammingOperatorAdviceV0

func (advice *mcpAutoprogrammingOperatorAdviceListV0) UnmarshalJSON(raw []byte) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		*advice = nil
		return nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return err
		}
		if strings.TrimSpace(text) == "" {
			*advice = nil
			return nil
		}
		*advice = []MCPAutoprogrammingOperatorAdviceV0{{Message: strings.TrimSpace(text)}}
		return nil
	}
	if strings.HasPrefix(trimmed, "{") {
		var item MCPAutoprogrammingOperatorAdviceV0
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		*advice = []MCPAutoprogrammingOperatorAdviceV0{item}
		return nil
	}
	var items []MCPAutoprogrammingOperatorAdviceV0
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}
	*advice = items
	return nil
}

func (advice mcpAutoprogrammingOperatorAdviceListV0) normalizedMCPV0(defaultTargetRef string) []MCPAutoprogrammingOperatorAdviceV0 {
	return normalizeMCPAutoprogrammingOperatorAdviceV0([]MCPAutoprogrammingOperatorAdviceV0(advice), defaultTargetRef)
}

func normalizeMCPAutoprogrammingOperatorAdviceV0(
	input []MCPAutoprogrammingOperatorAdviceV0,
	defaultTargetRef string,
) []MCPAutoprogrammingOperatorAdviceV0 {
	out := make([]MCPAutoprogrammingOperatorAdviceV0, 0, len(input))
	for _, item := range input {
		rawAction := firstNonEmptyMCPV0(item.Action, item.Kind)
		advice := MCPAutoprogrammingOperatorAdviceV0{
			AdviceRef:   strings.TrimSpace(item.AdviceRef),
			OperatorRef: strings.TrimSpace(item.OperatorRef),
			TargetRef: firstNonEmptyMCPV0(
				item.TargetRef,
				item.SubjectRef,
				item.Target,
				item.Ref,
				item.RunRef,
				item.Run,
				item.TaskRef,
				item.Task,
				defaultTargetRef,
			),
			RunRef:       firstNonEmptyMCPV0(item.RunRef, item.Run),
			TaskRef:      firstNonEmptyMCPV0(item.TaskRef, item.Task),
			Action:       normalizeMCPAutoprogrammingAdviceActionV0(rawAction),
			Message:      firstNonEmptyMCPV0(item.Message, item.Advice, item.Text),
			EvidenceRefs: compactStringsMCPV0(item.EvidenceRefs),
			NonBlocking:  true,
		}
		if advice.AdviceRef == "" &&
			advice.TargetRef == "" &&
			strings.TrimSpace(rawAction) == "" &&
			advice.Message == "" &&
			len(advice.EvidenceRefs) == 0 {
			continue
		}
		out = append(out, advice)
	}
	if out == nil {
		return []MCPAutoprogrammingOperatorAdviceV0{}
	}
	return out
}

func normalizeMCPAutoprogrammingAdviceActionV0(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "", "advice", "advise", "suggest", "recommend", "recommendation", "consejo", "aconsejar":
		return "advise"
	case "observe", "watch", "status", "mirar", "observar":
		return "observe"
	case "review", "revise", "revisar":
		return "review"
	case "pause", "stop", "block", "hold", "pausar", "parar", "bloquear":
		return "advise"
	default:
		return strings.TrimSpace(action)
	}
}
