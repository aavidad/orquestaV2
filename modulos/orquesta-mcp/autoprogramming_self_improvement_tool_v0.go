package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
)

const (
	MCPAutoprogrammingSelfImprovementToolNameV0    = "orquesta.autoprogramming.self_improvement.propose.v0"
	MCPAutoprogrammingSelfImprovementToolVersionV0 = "v0"
	MCPAutoprogrammingSelfImprovementResourceURIV0 = "orquesta://contracts/autoprogramming-self-improvement/v0"
	MCPAutoprogrammingSelfImprovementEstadoOKV0    = "ok"
	MCPAutoprogrammingSelfImprovementEstadoErrorV0 = "error"
)

type MCPAutoprogrammingSelfImprovementToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPAutoprogrammingSelfImprovementToolInputV0 struct {
	RequestID      string                                                           `json:"request_id,omitempty"`
	CorrelationID  string                                                           `json:"correlation_id,omitempty"`
	AutoPrepareRun bool                                                             `json:"auto_prepare_run,omitempty"`
	OperatorAdvice []MCPAutoprogrammingOperatorAdviceV0                             `json:"operator_advice,omitempty"`
	Proposal       orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0 `json:"proposal"`
}

func (input *MCPAutoprogrammingSelfImprovementToolInputV0) UnmarshalJSON(raw []byte) error {
	type alias MCPAutoprogrammingSelfImprovementToolInputV0
	var envelope struct {
		alias
		OperatorAdvice json.RawMessage `json:"operator_advice"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	*input = MCPAutoprogrammingSelfImprovementToolInputV0(envelope.alias)
	if len(envelope.OperatorAdvice) == 0 || strings.TrimSpace(string(envelope.OperatorAdvice)) == "null" {
		return nil
	}
	var advice mcpAutoprogrammingOperatorAdviceListV0
	if err := json.Unmarshal(envelope.OperatorAdvice, &advice); err != nil {
		return err
	}
	input.OperatorAdvice = []MCPAutoprogrammingOperatorAdviceV0(advice)
	return nil
}

type MCPAutoprogrammingSelfImprovementToolResultV0 struct {
	Estado                 string                                            `json:"estado"`
	RequestID              string                                            `json:"request_id,omitempty"`
	CorrelationID          string                                            `json:"correlation_id,omitempty"`
	Accepted               bool                                              `json:"accepted"`
	Background             bool                                              `json:"background"`
	PriorityScore          int                                               `json:"priority_score"`
	AutoprogrammingRequest *orquestaautoprogramming.AutoprogrammingRequestV0 `json:"autoprogramming_request,omitempty"`
	PrepareRun             *MCPAutoprogrammingPrepareRunToolInputV0          `json:"prepare_run,omitempty"`
	PreparedRun            *MCPAutoprogrammingPrepareRunToolResultV0         `json:"prepared_run,omitempty"`
	OperatorAdvice         []MCPAutoprogrammingOperatorAdviceV0              `json:"operator_advice,omitempty"`
	NextActions            []string                                          `json:"next_actions,omitempty"`
	Errores                []MCPValidationIssueV0                            `json:"errores_publicos,omitempty"`
}

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

type MCPAutoprogrammingSelfImprovementToolExecutorV0 struct {
	PrepareRun MCPTransportAutoprogrammingPrepareRunExecutorV0
}

func NewMCPAutoprogrammingSelfImprovementToolExecutorV0(
	prepareRun MCPTransportAutoprogrammingPrepareRunExecutorV0,
) MCPAutoprogrammingSelfImprovementToolExecutorV0 {
	return MCPAutoprogrammingSelfImprovementToolExecutorV0{PrepareRun: prepareRun}
}

func MCPAutoprogrammingSelfImprovementDescriptorV0() MCPAutoprogrammingSelfImprovementToolDescriptorV0 {
	return MCPAutoprogrammingSelfImprovementToolDescriptorV0{
		Name:        MCPAutoprogrammingSelfImprovementToolNameV0,
		Version:     MCPAutoprogrammingSelfImprovementToolVersionV0,
		InputSchema: "envelope:{request_id?,correlation_id?,auto_prepare_run?,operator_advice?,proposal:AutoprogrammingSelfImprovementProposalV0}",
		Output:      "ok:{autoprogramming_request,priority_score,prepare_run,prepared_run?,operator_advice?,next_actions}|error:{errores_publicos,operator_advice?,next_actions}",
		ResourceURI: MCPAutoprogrammingSelfImprovementResourceURIV0,
		Invariantes: []string{
			"convierte fallos observados en automejora secundaria",
			"solo prepara run si auto_prepare_run y executor inyectado existen",
			"prioridad baja por defecto para no bloquear trabajo principal",
			"operator_advice es observacion no bloqueante y no decide runtime",
			"hexagonal: solo refs opacas, evidencia y write-set propio",
		},
	}
}

func (executor MCPAutoprogrammingSelfImprovementToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAutoprogrammingSelfImprovementToolInputV0,
) (MCPAutoprogrammingSelfImprovementToolResultV0, error) {
	_ = ctx
	proposal := input.Proposal
	if strings.TrimSpace(proposal.RequestRef) == "" {
		proposal.RequestRef = firstNonEmptyMCPV0(input.RequestID, input.CorrelationID)
	}
	operatorAdvice := normalizeMCPAutoprogrammingOperatorAdviceV0(
		input.OperatorAdvice,
		firstNonEmptyMCPV0(proposal.RequestRef, input.RequestID, input.CorrelationID),
	)
	proposal = withMCPAutoprogrammingOperatorAdviceV0(proposal, operatorAdvice)
	result := orquestaautoprogramming.BuildAutoprogrammingSelfImprovementRequestV0(proposal)
	out := MCPAutoprogrammingSelfImprovementToolResultV0{
		Estado:         MCPAutoprogrammingSelfImprovementEstadoOKV0,
		RequestID:      strings.TrimSpace(result.Request.RequestRef),
		CorrelationID:  firstNonEmptyMCPV0(input.CorrelationID, input.RequestID, proposal.RequestRef),
		Accepted:       result.Accepted,
		Background:     result.Background,
		PriorityScore:  result.PriorityScore,
		OperatorAdvice: operatorAdvice,
		NextActions:    compactStringsMCPV0(result.NextActions),
	}
	if len(out.OperatorAdvice) > 0 {
		out.NextActions = compactStringsMCPV0(append(out.NextActions, "operator_advice_recorded_non_blocking"))
	}
	if !result.Accepted {
		out.Estado = MCPAutoprogrammingSelfImprovementEstadoErrorV0
		out.RequestID = strings.TrimSpace(proposal.RequestRef)
		out.Errores = selfImprovementIssuesMCPV0(result.Issues)
		return out, nil
	}
	out.AutoprogrammingRequest = &result.Request
	out.PrepareRun = &MCPAutoprogrammingPrepareRunToolInputV0{
		RequestID:              strings.TrimSpace(result.Request.RequestRef),
		CorrelationID:          out.CorrelationID,
		RequestedBy:            firstNonEmptyMCPV0(proposal.ObservedBy, "orquesta-autoprogramming-self-improvement"),
		AutoprogrammingRequest: result.Request,
		PriorityScore:          result.PriorityScore,
	}
	executor.maybePrepareSelfImprovementRunV0(ctx, input, &out)
	return out, nil
}

func (executor MCPAutoprogrammingSelfImprovementToolExecutorV0) maybePrepareSelfImprovementRunV0(
	ctx context.Context,
	input MCPAutoprogrammingSelfImprovementToolInputV0,
	out *MCPAutoprogrammingSelfImprovementToolResultV0,
) {
	if !input.AutoPrepareRun || out == nil || out.PrepareRun == nil {
		return
	}
	if executor.PrepareRun == nil {
		out.NextActions = compactStringsMCPV0(append(out.NextActions,
			"repair_configure_autoprogramming_prepare_run_executor",
			"post_/api/v0/autoprogramming/prepare-run_with_prepare_run",
		))
		out.Errores = append(out.Errores, MCPValidationIssueV0{
			Code:    "autoprogramming_prepare_run_port_unavailable",
			Field:   "auto_prepare_run",
			Message: "prepare-run executor no configurado",
		})
		return
	}
	prepared, err := executor.PrepareRun.Execute(ctx, *out.PrepareRun)
	if err != nil {
		prepared = NewMCPAutoprogrammingPrepareRunErrorResultV0(
			*out.PrepareRun,
			"autoprogramming_prepare_run_error",
			"auto_prepare_run",
			err.Error(),
		)
	}
	out.PreparedRun = &prepared
	if prepared.Estado == MCPAutoprogrammingPrepareRunEstadoErrorV0 {
		out.NextActions = compactStringsMCPV0(append(out.NextActions,
			"repair_autoprogramming_prepare_run_result",
		))
		return
	}
	out.NextActions = compactStringsMCPV0(append(out.NextActions,
		"supervise_prepared_run_by_run_ref",
	))
}

func selfImprovementIssuesMCPV0(
	issues []orquestaautoprogramming.AutoprogrammingRequestIssueV0,
) []MCPValidationIssueV0 {
	out := make([]MCPValidationIssueV0, 0, len(issues))
	for _, issue := range issues {
		out = append(out, MCPValidationIssueV0{
			Code:    strings.TrimSpace(issue.Code),
			Field:   strings.TrimSpace(issue.Field),
			Message: strings.TrimSpace(issue.Message),
		})
	}
	if out == nil {
		return []MCPValidationIssueV0{}
	}
	return out
}

func withMCPAutoprogrammingOperatorAdviceV0(
	proposal orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0,
	advice []MCPAutoprogrammingOperatorAdviceV0,
) orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0 {
	for _, item := range advice {
		for _, ref := range compactStringsMCPV0([]string{item.AdviceRef, item.OperatorRef, item.TargetRef, item.RunRef, item.TaskRef}) {
			proposal.ContextRefs = append(proposal.ContextRefs, "operator_advice_ref:"+ref)
		}
		for _, ref := range item.EvidenceRefs {
			proposal.ContextRefs = append(proposal.ContextRefs, "operator_advice_evidence_ref:"+ref)
		}
		if item.Message != "" || item.Action != "" {
			proposal.CompactRules = append(proposal.CompactRules,
				"operator_advice_non_blocking:"+firstNonEmptyMCPV0(item.Action, "advise")+":"+item.Message,
			)
		}
	}
	return proposal
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
