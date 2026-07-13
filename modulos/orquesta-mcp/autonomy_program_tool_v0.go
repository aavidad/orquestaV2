package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	MCPAutonomyProgramToolNameV0    = "orquesta.autonomy.program.v0"
	MCPAutonomyProgramToolVersionV0 = "v0"
	MCPAutonomyProgramResourceURIV0 = "orquesta://contracts/autonomy-program/v0"

	MCPAutonomyProgramEstadoOKV0    = "ok"
	MCPAutonomyProgramEstadoErrorV0 = "error"

	MCPAutonomyProgramActionSaveV0     = "save"
	MCPAutonomyProgramActionObserveV0  = "observe"
	MCPAutonomyProgramActionFrontierV0 = "frontier"
	MCPAutonomyProgramActionRecoverV0  = "recover"

	MCPAutonomyProgramErrPortUnavailableV0 = "autonomy_program_port_unavailable"
	MCPAutonomyProgramErrActionRequiredV0  = "autonomy_program_action_required"
	MCPAutonomyProgramErrFailedV0          = "autonomy_program_failed"
)

type MCPAutonomyProgramToolInputV0 struct {
	SchemaVersion string          `json:"schema_version,omitempty"`
	RequestRef    string          `json:"request_ref,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Action        string          `json:"action"`
	ProjectRef    string          `json:"project_ref,omitempty"`
	RootRef       string          `json:"root_ref,omitempty"`
	ProgramRef    string          `json:"program_ref,omitempty"`
	Program       json.RawMessage `json:"program,omitempty"`
}

type MCPAutonomyProgramToolResultV0 struct {
	Estado          string                                 `json:"estado"`
	RequestRef      string                                 `json:"request_ref,omitempty"`
	CorrelationID   string                                 `json:"correlation_id,omitempty"`
	ProgramRef      string                                 `json:"program_ref,omitempty"`
	Status          string                                 `json:"status,omitempty"`
	NodeCount       int                                    `json:"node_count,omitempty"`
	LaunchRefs      []string                               `json:"launch_refs,omitempty"`
	RecoveryRefs    []string                               `json:"recovery_refs,omitempty"`
	Program         json.RawMessage                        `json:"program,omitempty"`
	ErroresPublicos []MCPToolCapabilitiesListPublicErrorV0 `json:"errores_publicos,omitempty"`
}

// MCPAutonomyProgramPortV0 lo implementa el servidor sobre el dominio real. La
// tool no conoce el grafo: pide guardar, observar, calcular la frontera o
// recuperar.
type MCPAutonomyProgramPortV0 interface {
	RunAutonomyProgramActionV0(context.Context, MCPAutonomyProgramToolInputV0) (MCPAutonomyProgramToolResultV0, error)
}

type MCPAutonomyProgramToolExecutorV0 struct {
	Program MCPAutonomyProgramPortV0
}

func MCPAutonomyProgramDescriptorV0() MCPToolCapabilitiesListToolDescriptorV0 {
	return MCPToolCapabilitiesListToolDescriptorV0{
		Name:        MCPAutonomyProgramToolNameV0,
		Version:     MCPAutonomyProgramToolVersionV0,
		InputSchema: "autonomy_program:{schema_version?,request_ref?,correlation_id?,action,project_ref?,root_ref?,program_ref?,program?}",
		Output:      "ok:{estado,program_ref?,status?,node_count?,launch_refs?[],recovery_refs?[],program?}|error:{estado,errores_publicos[]{code,field?}}",
		ResourceURI: MCPAutonomyProgramResourceURIV0,
		Invariantes: []string{
			"el avance del programa se persiste con CompareAndSwap: dos planificadores no se pisan",
			"save acepta un reintento identico y choca si la topologia cambio",
			"la frontera solo lanza nodos con dependencias satisfechas",
			"puerto sin cablear se delata antes de validar la entrada",
		},
	}
}

func (executor MCPAutonomyProgramToolExecutorV0) Execute(
	ctx context.Context,
	input MCPAutonomyProgramToolInputV0,
) (MCPAutonomyProgramToolResultV0, error) {
	input.Action = strings.TrimSpace(input.Action)
	if executor.Program == nil {
		return newMCPAutonomyProgramErrorV0(input, MCPAutonomyProgramErrPortUnavailableV0, "program"), nil
	}
	if input.Action == "" {
		return newMCPAutonomyProgramErrorV0(input, MCPAutonomyProgramErrActionRequiredV0, "action"), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := executor.Program.RunAutonomyProgramActionV0(ctx, input)
	if err != nil {
		return newMCPAutonomyProgramErrorV0(input, MCPAutonomyProgramErrFailedV0, "action"), nil
	}
	result.Estado = MCPAutonomyProgramEstadoOKV0
	result.RequestRef = input.RequestRef
	result.CorrelationID = input.CorrelationID
	return result, nil
}

func mcpAutonomyProgramTransportHandlerV0(executor MCPTransportAutonomyProgramExecutorV0) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPAutonomyProgramToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if executor == nil {
			return mcpTransportToolErrorPayloadV0(MCPAutonomyProgramToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func newMCPAutonomyProgramErrorV0(
	input MCPAutonomyProgramToolInputV0,
	code string,
	field string,
) MCPAutonomyProgramToolResultV0 {
	return MCPAutonomyProgramToolResultV0{
		Estado:          MCPAutonomyProgramEstadoErrorV0,
		RequestRef:      input.RequestRef,
		CorrelationID:   input.CorrelationID,
		ProgramRef:      input.ProgramRef,
		ErroresPublicos: []MCPToolCapabilitiesListPublicErrorV0{{Code: code, Field: field}},
	}
}
