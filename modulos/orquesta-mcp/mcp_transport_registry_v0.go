package orquestamcp

import (
	"context"
	"encoding/json"
	"errors"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

const (
	MCPTransportModeOptInV0           = "adapter_opt_in"
	MCPTransportToolUnboundV0         = "mcp_transport_tool_unbound"
	MCPTransportPortUnavailableV0     = "mcp_transport_port_unavailable"
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
	Name        string                        `json:"name"`
	Version     string                        `json:"version"`
	URI         string                        `json:"uri"`
	ContentType string                        `json:"content_type"`
	SummaryKey  string                        `json:"summary_key,omitempty"`
	Shape       string                        `json:"shape"`
	Mode        string                        `json:"mode"`
	Handler     MCPTransportResourceHandlerV0 `json:"-"`
}

type MCPTransportToolEnvelopeV0 struct {
	Name        string                    `json:"name"`
	Version     string                    `json:"version"`
	ResourceURI string                    `json:"resource_uri"`
	InputShape  string                    `json:"input_shape"`
	OutputShape string                    `json:"output_shape"`
	Mode        string                    `json:"mode"`
	Handler     MCPTransportToolHandlerV0 `json:"-"`
}

type MCPTransportBindingsV0 struct {
	NuevaApp             MCPTransportNuevaAppExecutorV0
	ArrancarDirector     MCPTransportArrancarDirectorAppExecutorV0
	RequestAppChange     MCPTransportRequestAppChangeExecutorV0
	EjecutarOrquestacion MCPTransportEjecutarOrquestacionAppExecutorV0
	DirectorDecision     MCPTransportDirectorAgentDecisionExecutorV0
	DirectorStats        MCPTransportDirectorStatsExecutorV0
	RunControl           MCPTransportRunControlExecutorV0
	RunQueuePriority     MCPTransportRunQueuePriorityExecutorV0
	OperatorStatus       operator.OperatorMCPStatusPortV0
	OperatorBurst        operator.OperatorMCPBurstPortV0
	OperatorOutbox       operator.OperatorMCPOutboxPortV0
	OperatorQuery        operator.OperatorMCPDirectedQueryPortV0
}

type MCPTransportNuevaAppExecutorV0 interface {
	Execute(context.Context, MCPNuevaAppToolInputV0) (MCPNuevaAppToolResultV0, error)
}

type MCPTransportArrancarDirectorAppExecutorV0 interface {
	Execute(
		context.Context,
		MCPArrancarDirectorAppToolInputV0,
	) (MCPArrancarDirectorAppToolResultV0, error)
}

type MCPTransportRequestAppChangeExecutorV0 interface {
	Execute(
		context.Context,
		MCPRequestAppChangeToolInputV0,
	) (MCPRequestAppChangeToolResultV0, error)
}

type MCPTransportEjecutarOrquestacionAppExecutorV0 interface {
	Execute(
		context.Context,
		MCPEjecutarOrquestacionAppToolInputV0,
	) (MCPEjecutarOrquestacionAppToolResultV0, error)
}

type MCPTransportDirectorStatsExecutorV0 interface {
	Execute(
		context.Context,
		MCPDirectorStatsToolInputV0,
	) (MCPDirectorStatsToolResultV0, error)
}

type MCPTransportRunControlExecutorV0 interface {
	Execute(
		context.Context,
		MCPRunControlToolInputV0,
	) (MCPRunControlToolResultV0, error)
}

type MCPTransportRunQueuePriorityExecutorV0 interface {
	Execute(
		context.Context,
		MCPRunQueuePriorityToolInputV0,
	) (MCPRunQueuePriorityToolResultV0, error)
}

type MCPTransportToolErrorV0 struct {
	Estado    string `json:"estado"`
	Tool      string `json:"tool"`
	ErrorCode string `json:"error_code"`
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
	bootstrap := MCPBootstrapDescriptorV0()
	workflow := MCPCoreWorkflowContractsDescriptorV0()
	operatorOps := MCPOperatorOperationsDescriptorV0()
	return []MCPTransportResourceEnvelopeV0{
		mcpTransportResourceEnvelopeV0(shared.Name, shared.Version, shared.URI, shared.ContentType, shared.SummaryKey, func() any { return NewMCPSharedContractsResourceV0() }),
		mcpTransportResourceEnvelopeV0(roadmap.Name, roadmap.Version, roadmap.URI, roadmap.ContentType, roadmap.SummaryKey, func() any { return NewMCPProjectRoadmapResourceV0() }),
		mcpTransportResourceEnvelopeV0(operational.Name, operational.Version, operational.URI, operational.ContentType, operational.SummaryKey, func() any { return NewMCPOperationalStatusResourceV0() }),
		mcpTransportResourceEnvelopeV0(bootstrap.Name, bootstrap.Version, bootstrap.URI, bootstrap.ContentType, bootstrap.SummaryKey, func() any { return NewMCPBootstrapResourceV0() }),
		mcpTransportResourceEnvelopeV0(workflow.Name, workflow.Version, workflow.URI, workflow.ContentType, workflow.SummaryKey, func() any { return NewMCPCoreWorkflowContractsResourceV0() }),
		mcpTransportResourceEnvelopeV0(operatorOps.Name, operatorOps.Version, operatorOps.URI, operatorOps.ContentType, operatorOps.SummaryKey, func() any { return NewMCPOperatorOperationsResourceV0() }),
	}
}

func MCPTransportToolsV0(bindings MCPTransportBindingsV0) []MCPTransportToolEnvelopeV0 {
	nueva := MCPNuevaAppDescriptorV0()
	director := MCPArrancarDirectorAppDescriptorV0()
	change := MCPRequestAppChangeDescriptorV0()
	decision := MCPDirectorAgentDecisionDescriptorV0()
	stats := MCPDirectorStatsDescriptorV0()
	preparar := MCPPrepararOrquestacionAppDescriptorV0()
	ejecutar := MCPEjecutarOrquestacionAppDescriptorV0()
	autoprogramming := MCPAutoprogrammingValidateRequestDescriptorV0()
	bootstrap := MCPBootstrapToolDescriptorV0Value()
	workflow := MCPCoreWorkflowCommandToolDescriptorV0Value()
	runControl := MCPRunControlDescriptorV0()
	runQueue := MCPRunQueuePriorityDescriptorV0()
	return []MCPTransportToolEnvelopeV0{
		mcpTransportToolEnvelopeV0(nueva.Name, nueva.Version, nueva.ResourceURI, nueva.InputSchema, nueva.Output, mcpNuevaAppTransportHandlerV0(bindings.NuevaApp)),
		mcpTransportToolEnvelopeV0(director.Name, director.Version, director.ResourceURI, director.InputSchema, director.Output, mcpArrancarDirectorAppTransportHandlerV0(bindings.ArrancarDirector)),
		mcpTransportToolEnvelopeV0(change.Name, change.Version, change.ResourceURI, change.InputSchema, change.Output, mcpRequestAppChangeTransportHandlerV0(bindings.RequestAppChange)),
		mcpTransportToolEnvelopeV0(decision.Name, decision.Version, decision.ResourceURI, decision.InputSchema, decision.Output, mcpDirectorAgentDecisionTransportHandlerV0(bindings.DirectorDecision)),
		mcpTransportToolEnvelopeV0(stats.Name, stats.Version, stats.ResourceURI, stats.InputSchema, stats.Output, mcpDirectorStatsTransportHandlerV0(bindings.DirectorStats)),
		mcpTransportToolEnvelopeV0(preparar.Name, preparar.Version, preparar.ResourceURI, preparar.InputSchema, preparar.Output, mcpPrepararOrquestacionAppTransportHandlerV0),
		mcpTransportToolEnvelopeV0(ejecutar.Name, ejecutar.Version, ejecutar.ResourceURI, ejecutar.InputSchema, ejecutar.Output, mcpEjecutarOrquestacionAppTransportHandlerV0(bindings.EjecutarOrquestacion)),
		mcpTransportToolEnvelopeV0(autoprogramming.Name, autoprogramming.Version, autoprogramming.ResourceURI, autoprogramming.InputSchema, autoprogramming.Output, mcpAutoprogrammingValidateRequestTransportHandlerV0(MCPAutoprogrammingValidateRequestToolExecutorV0{})),
		mcpTransportToolEnvelopeV0(bootstrap.Name, bootstrap.Version, bootstrap.ResourceURI, bootstrap.InputSchema, bootstrap.Output, mcpBootstrapTransportHandlerV0),
		mcpTransportToolEnvelopeV0(workflow.Name, workflow.Version, workflow.ResourceURI, workflow.InputSchema, workflow.Output, mcpCoreWorkflowTransportHandlerV0),
		mcpTransportToolEnvelopeV0(runControl.Name, runControl.Version, runControl.ResourceURI, runControl.InputSchema, runControl.Output, mcpRunControlTransportHandlerV0(bindings.RunControl)),
		mcpTransportToolEnvelopeV0(runQueue.Name, runQueue.Version, runQueue.ResourceURI, runQueue.InputSchema, runQueue.Output, mcpRunQueuePriorityTransportHandlerV0(bindings.RunQueuePriority)),
		mcpOperatorTransportToolV0(operator.OperatorMCPStatusToolNameV0, bindings.OperatorStatus),
		mcpOperatorTransportToolV0(operator.OperatorMCPBurstToolNameV0, bindings.OperatorBurst),
		mcpOperatorTransportToolV0(operator.OperatorMCPOutboxToolNameV0, bindings.OperatorOutbox),
		mcpOperatorTransportToolV0(operator.OperatorMCPDirectedQueryToolV0, bindings.OperatorQuery),
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
		Name:        name,
		Version:     version,
		URI:         uri,
		ContentType: contentType,
		SummaryKey:  summaryKey,
		Shape:       MCPTransportResourceShapeV0,
		Mode:        MCPTransportModeOptInV0,
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
		Name:        name,
		Version:     version,
		ResourceURI: resourceURI,
		InputShape:  inputShape,
		OutputShape: outputShape,
		Mode:        MCPTransportModeOptInV0,
		Handler:     handler,
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
