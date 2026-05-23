package orquestaoperatormcp

import (
	"fmt"
	"strings"
)

type OperatorMCPSimulatedConnectorConfigV0 struct {
	StatusConnectorRef string
	BurstConnectorRef  string
	OutboxConnectorRef string
	QueryConnectorRef  string
	Status             string
	Summary            string
	OutboxItems        []OperatorMCPOutboxItemV0
}

type OperatorMCPSimulatedConnectorV0 struct {
	config OperatorMCPSimulatedConnectorConfigV0
}

var _ OperatorMCPConnectorV0 = OperatorMCPSimulatedConnectorV0{}

func NewOperatorMCPSimulatedConnectorV0(
	config OperatorMCPSimulatedConnectorConfigV0,
) OperatorMCPSimulatedConnectorV0 {
	return OperatorMCPSimulatedConnectorV0{config: normalizeSimulatedOperatorConfigV0(config)}
}

func (connector OperatorMCPSimulatedConnectorV0) QueryOperatorStatusV0(
	input OperatorStatusQueryV0,
) (OperatorMCPStatusResultV0, error) {
	if err := connector.requireConnectorRefV0("status", input.StatusConnectorRef); err != nil {
		return OperatorMCPStatusResultV0{}, err
	}
	return OperatorMCPStatusResultV0{
		Status:       connector.config.Status,
		Summary:      connector.config.Summary,
		Sections:     normalizeOperatorSectionsV0(input.IncludeSections),
		EvidenceRefs: []string{"evidence-ref-operator-status-simulated"},
	}, nil
}

func (connector OperatorMCPSimulatedConnectorV0) RequestOperatorSupervisedBurstV0(
	input OperatorSupervisedBurstRequestV0,
) (OperatorMCPBurstResultV0, error) {
	if err := connector.requireConnectorRefV0("burst", input.BurstConnectorRef); err != nil {
		return OperatorMCPBurstResultV0{}, err
	}
	steps := input.MaxSteps
	if steps > 2 {
		steps = 2
	}
	return OperatorMCPBurstResultV0{
		BurstRef:      "burst-ref-" + input.RequestRef,
		ExecutedSteps: steps,
		FinalAction:   "simulated_supervision_ready",
		TraceRefs:     []string{"trace-ref-" + input.RunRef},
	}, nil
}

func (connector OperatorMCPSimulatedConnectorV0) ListOperatorPendingOutboxV0(
	input OperatorPendingOutboxQueryV0,
) (OperatorMCPOutboxResultV0, error) {
	if err := connector.requireConnectorRefV0("outbox", input.OutboxConnectorRef); err != nil {
		return OperatorMCPOutboxResultV0{}, err
	}
	kinds := normalizeOperatorKindsV0(input.IncludeKinds)
	items := filterOperatorOutboxItemsV0(connector.config.OutboxItems, kinds)
	return OperatorMCPOutboxResultV0{
		WatermarkRef: "watermark-ref-" + input.SubjectRef,
		PendingCount: len(items),
		Items:        limitOperatorOutboxItemsV0(items, input.Limit),
		EvidenceRefs: []string{"evidence-ref-operator-outbox-simulated"},
	}, nil
}

func (connector OperatorMCPSimulatedConnectorV0) RaiseOperatorDirectedQueryV0(
	input OperatorDirectedQueryV0,
) (OperatorMCPDirectedQueryResultV0, error) {
	if err := connector.requireConnectorRefV0("query", input.QueryConnectorRef); err != nil {
		return OperatorMCPDirectedQueryResultV0{}, err
	}
	return OperatorMCPDirectedQueryResultV0{
		Accepted:   true,
		AnswerRef:  "answer-ref-" + input.QueryRef,
		NextAction: "await_external_operator_answer",
		TraceRefs:  []string{"trace-ref-" + input.TargetRef},
	}, nil
}

func (connector OperatorMCPSimulatedConnectorV0) requireConnectorRefV0(kind string, ref string) error {
	expected := connectorRefForKindV0(connector.config, kind)
	if normalizeOperatorConnectorRefAliasV0(ref) != normalizeOperatorConnectorRefAliasV0(expected) {
		return fmt.Errorf("operator_mcp_connector_unavailable")
	}
	return nil
}

func normalizeSimulatedOperatorConfigV0(
	config OperatorMCPSimulatedConnectorConfigV0,
) OperatorMCPSimulatedConnectorConfigV0 {
	if strings.TrimSpace(config.StatusConnectorRef) == "" {
		config.StatusConnectorRef = "status-connector-ref-simulated"
	}
	if strings.TrimSpace(config.BurstConnectorRef) == "" {
		config.BurstConnectorRef = "burst-connector-ref-simulated"
	}
	if strings.TrimSpace(config.OutboxConnectorRef) == "" {
		config.OutboxConnectorRef = "outbox-connector-ref-simulated"
	}
	if strings.TrimSpace(config.QueryConnectorRef) == "" {
		config.QueryConnectorRef = "query-connector-ref-simulated"
	}
	if strings.TrimSpace(config.Status) == "" {
		config.Status = "simulated"
	}
	if strings.TrimSpace(config.Summary) == "" {
		config.Summary = "simulated operator connector"
	}
	if len(config.OutboxItems) == 0 {
		config.OutboxItems = []OperatorMCPOutboxItemV0{{
			MessageRef: "message-ref-simulated-001",
			Kind:       "director_question",
			TargetRef:  "target-ref-simulated-001",
		}}
	}
	config.OutboxItems = normalizeOperatorOutboxItemsV0(config.OutboxItems)
	return config
}

func connectorRefForKindV0(config OperatorMCPSimulatedConnectorConfigV0, kind string) string {
	switch kind {
	case "status":
		return config.StatusConnectorRef
	case "burst":
		return config.BurstConnectorRef
	case "outbox":
		return config.OutboxConnectorRef
	default:
		return config.QueryConnectorRef
	}
}

func normalizeOperatorConnectorRefAliasV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, "state", "status")
	value = strings.ReplaceAll(value, "estado", "status")
	value = strings.ReplaceAll(value, "consulta", "query")
	return value
}
