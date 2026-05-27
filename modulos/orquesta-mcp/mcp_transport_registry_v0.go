package orquestamcp

import (
	"context"
	"encoding/json"
	"errors"
)

const (
	MCPTransportModeOptInV0           = "adapter_opt_in"
	MCPTransportToolUnboundV0         = MCPPublicErrTransportUnboundV0
	MCPTransportToolInputInvalidV0    = MCPPublicErrTransportInputV0
	MCPTransportPortUnavailableV0     = MCPPublicErrTransportPortV0
	MCPTransportSchemaStaleV0         = MCPPublicErrTransportSchemaV0
	MCPTransportResourceShapeV0       = "resource_payload_compacto_v0"
	MCPTransportOperatorOutputShapeV0 = "operator_tool_result_compacto_v0"
)

var ErrMCPTransportPortUnavailableV0 = errors.New(MCPTransportPortUnavailableV0)

type MCPTransportPortV0 interface {
	RegisterResourceV0(MCPTransportResourceEnvelopeV0) error
	RegisterToolV0(MCPTransportToolEnvelopeV0) error
}

type MCPTransportResourceHandlerV0 func(context.Context) (json.RawMessage, error)

type MCPTransportToolHandlerV0 func(context.Context, json.RawMessage) (json.RawMessage, error)

type MCPTransportResourceEnvelopeV0 struct {
	Name             string                        `json:"name"`
	Version          string                        `json:"version"`
	URI              string                        `json:"uri"`
	ContentType      string                        `json:"content_type"`
	SummaryKey       string                        `json:"summary_key,omitempty"`
	Shape            string                        `json:"shape"`
	Mode             string                        `json:"mode"`
	DescriptorSource MCPResourceDescriptorSourceV0 `json:"descriptor_source"`
	OutputBudget     MCPTransportOutputBudgetV0    `json:"output_budget"`
	ExecutionBudget  MCPTransportExecutionBudgetV0 `json:"execution_budget"`
	Handler          MCPTransportResourceHandlerV0 `json:"-"`
}

type MCPTransportToolEnvelopeV0 struct {
	Name            string                        `json:"name"`
	Version         string                        `json:"version"`
	ResourceURI     string                        `json:"resource_uri"`
	InputShape      string                        `json:"input_shape"`
	OutputShape     string                        `json:"output_shape"`
	Mode            string                        `json:"mode"`
	OutputBudget    MCPTransportOutputBudgetV0    `json:"output_budget"`
	ExecutionBudget MCPTransportExecutionBudgetV0 `json:"execution_budget"`
	Handler         MCPTransportToolHandlerV0     `json:"-"`
}

func RegisterMCPTransportV0(port MCPTransportPortV0, bindings MCPTransportBindingsV0) error {
	if port == nil {
		return ErrMCPTransportPortUnavailableV0
	}
	for _, resource := range MCPTransportResourcesV0() {
		if err := port.RegisterResourceV0(resource); err != nil {
			return err
		}
	}
	for _, tool := range MCPTransportToolsV0(bindings) {
		if err := port.RegisterToolV0(tool); err != nil {
			return err
		}
	}
	return nil
}

func MCPTransportResourcesV0() []MCPTransportResourceEnvelopeV0 {
	shared := MCPSharedContractsDescriptorV0()
	roadmap := MCPProjectRoadmapDescriptorV0()
	operational := MCPOperationalStatusDescriptorV0()
	functionContracts := MCPFunctionContractDescriptorV0()
	timeline := MCPWorkspaceTimelineDescriptorV0()
	bootstrap := MCPBootstrapDescriptorV0()
	workflow := MCPCoreWorkflowContractsDescriptorV0()
	operatorOps := MCPOperatorOperationsDescriptorV0()
	return []MCPTransportResourceEnvelopeV0{
		mcpTransportResourceEnvelopeV0(shared.Name, shared.Version, shared.URI, shared.ContentType, shared.SummaryKey, func() any { return NewMCPSharedContractsResourceV0() }),
		mcpTransportResourceEnvelopeV0(roadmap.Name, roadmap.Version, roadmap.URI, roadmap.ContentType, roadmap.SummaryKey, func() any { return NewMCPProjectRoadmapResourceV0() }),
		mcpTransportResourceEnvelopeV0(operational.Name, operational.Version, operational.URI, operational.ContentType, operational.SummaryKey, func() any { return NewMCPOperationalStatusResourceV0() }),
		mcpTransportResourceEnvelopeV0(functionContracts.Name, functionContracts.Version, functionContracts.URI, functionContracts.ContentType, functionContracts.SummaryKey, func() any { return NewMCPFunctionContractResourceV0() }),
		mcpTransportResourceEnvelopeV0(timeline.Name, timeline.Version, timeline.URI, timeline.ContentType, timeline.SummaryKey, func() any { return NewMCPWorkspaceTimelineResourceV0() }),
		mcpTransportResourceEnvelopeV0(bootstrap.Name, bootstrap.Version, bootstrap.URI, bootstrap.ContentType, bootstrap.SummaryKey, func() any { return NewMCPBootstrapResourceV0() }),
		mcpTransportResourceEnvelopeV0(workflow.Name, workflow.Version, workflow.URI, workflow.ContentType, workflow.SummaryKey, func() any { return NewMCPCoreWorkflowContractsResourceV0() }),
		mcpTransportResourceEnvelopeV0(operatorOps.Name, operatorOps.Version, operatorOps.URI, operatorOps.ContentType, operatorOps.SummaryKey, func() any { return NewMCPOperatorOperationsResourceV0() }),
	}
}

func mcpTransportResourceEnvelopeV0(
	name string,
	version string,
	uri string,
	contentType string,
	summaryKey string,
	payload func() any,
) MCPTransportResourceEnvelopeV0 {
	return MCPTransportResourceEnvelopeV0{
		Name:             name,
		Version:          version,
		URI:              uri,
		ContentType:      contentType,
		SummaryKey:       summaryKey,
		Shape:            MCPTransportResourceShapeV0,
		Mode:             MCPTransportModeOptInV0,
		DescriptorSource: mcpResourceDescriptorSourceForResourceV0(name),
		OutputBudget:     MCPTransportResourceOutputBudgetV0(MCPTransportOutputFreshnessStaticV0),
		ExecutionBudget:  MCPTransportResourceExecutionBudgetV0(),
		Handler: func(context.Context) (json.RawMessage, error) {
			return json.Marshal(payload())
		},
	}
}

func mcpTransportToolEnvelopeV0(
	name string,
	version string,
	resourceURI string,
	inputShape string,
	outputShape string,
	handler MCPTransportToolHandlerV0,
) MCPTransportToolEnvelopeV0 {
	return MCPTransportToolEnvelopeV0{
		Name:            name,
		Version:         version,
		ResourceURI:     resourceURI,
		InputShape:      inputShape,
		OutputShape:     outputShape,
		Mode:            MCPTransportModeOptInV0,
		OutputBudget:    MCPTransportToolOutputBudgetV0(MCPTransportOutputFreshnessLiveV0),
		ExecutionBudget: MCPTransportToolExecutionBudgetV0(MCPTransportExecutionProfileControlPlaneMutationV0),
		Handler:         handler,
	}
}

func mcpOperatorTransportToolV0(name string, port any) MCPTransportToolEnvelopeV0 {
	return mcpTransportToolEnvelopeV0(
		name,
		"v0",
		MCPOperatorOperationsResourceURIV0,
		"operator_tool_input_compacto_v0",
		MCPTransportOperatorOutputShapeV0,
		mcpOperatorTransportHandlerV0(name, port),
	)
}

func mcpNuevaAppTransportHandlerV0(port MCPTransportNuevaAppExecutorV0) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPNuevaAppToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPNuevaAppToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func mcpArrancarDirectorAppTransportHandlerV0(
	port MCPTransportArrancarDirectorAppExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPArrancarDirectorAppToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if port == nil {
			return mcpTransportToolErrorPayloadV0(MCPArrancarDirectorAppToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := port.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func mcpBootstrapTransportHandlerV0(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input MCPBootstrapToolInputV0
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, err
	}
	return json.Marshal(ExecuteMCPBootstrapToolV0(input))
}

func mcpCoreWorkflowTransportHandlerV0(_ context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input MCPCoreWorkflowCommandToolInputV0
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, err
	}
	return json.Marshal(ExecuteMCPCoreWorkflowCommandToolV0(input))
}

func mcpHumanDirectorWorkReviewPlanTransportHandlerV0(
	executor MCPHumanDirectorWorkReviewPlanToolExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPHumanDirectorWorkReviewPlanToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func mcpAutoprogrammingSelfImprovementTransportHandlerV0(
	executor MCPAutoprogrammingSelfImprovementToolExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPAutoprogrammingSelfImprovementToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}
