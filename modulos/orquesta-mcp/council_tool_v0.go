package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	MCPCouncilToolNameV0    = "orquesta.council.convene.v0"
	MCPCouncilToolVersionV0 = "v0"
	MCPCouncilResourceURIV0 = "orquesta://contracts/council-convene/v0"

	MCPCouncilEstadoOKV0    = "ok"
	MCPCouncilEstadoErrorV0 = "error"

	MCPCouncilActionAssignV0 = "assign"
	MCPCouncilActionDecideV0 = "decide"

	MCPCouncilErrPortUnavailableV0 = "council_port_unavailable"
	MCPCouncilErrActionRequiredV0  = "council_action_required"
	MCPCouncilErrConveneFailedV0   = "council_convene_failed"
)

// MCPCouncilPublicErrorClassifierV0 lo aporta el stack para traducir el error
// TIPADO del dominio a un codigo publico. La superficie MCP nunca vuelca
// `err.Error()`: seria filtrar detalle interno al exterior.
type MCPCouncilPublicErrorClassifierV0 func(error) (code string, field string, ok bool)

type MCPCouncilMemberV0 struct {
	MemberRef       string  `json:"member_ref"`
	FamilyRef       string  `json:"family_ref,omitempty"`
	BudgetRemaining float64 `json:"budget_remaining"`
	CapabilityRank  int     `json:"capability_rank,omitempty"`
}

// MCPCouncilOverrideV0 es la superficie por la que el operador FUERZA un rol. Su
// requisito literal: "yo puedo forzar que sea uno u otro, pero para eso tengo que
// tener algun medio de hacerlo". Este es el medio.
type MCPCouncilOverrideV0 struct {
	Role      string `json:"role"`
	MemberRef string `json:"member_ref"`
	ForcedBy  string `json:"forced_by,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type MCPCouncilBallotV0 struct {
	MemberRef string `json:"member_ref"`
	Vote      string `json:"vote"`
	Reason    string `json:"reason,omitempty"`
}

type MCPCouncilToolInputV0 struct {
	SchemaVersion    string                 `json:"schema_version,omitempty"`
	RequestRef       string                 `json:"request_ref,omitempty"`
	CorrelationID    string                 `json:"correlation_id,omitempty"`
	Action           string                 `json:"action"`
	CouncilRef       string                 `json:"council_ref,omitempty"`
	AuthorRef        string                 `json:"author_ref,omitempty"`
	SecurityCritical bool                   `json:"security_critical,omitempty"`
	Members          []MCPCouncilMemberV0   `json:"members,omitempty"`
	Overrides        []MCPCouncilOverrideV0 `json:"overrides,omitempty"`
	Ballots          []MCPCouncilBallotV0   `json:"ballots,omitempty"`
}

type MCPCouncilSeatV0 struct {
	Role               string `json:"role"`
	MemberRef          string `json:"member_ref"`
	FamilyRef          string `json:"family_ref,omitempty"`
	Material           string `json:"material"`
	Forced             bool   `json:"forced,omitempty"`
	ForcedBy           string `json:"forced_by,omitempty"`
	ForcedReason       string `json:"forced_reason,omitempty"`
	AutomaticMemberRef string `json:"automatic_member_ref,omitempty"`
	BudgetWarning      string `json:"budget_warning,omitempty"`
}

type MCPCouncilToolResultV0 struct {
	Estado          string                                 `json:"estado"`
	RequestRef      string                                 `json:"request_ref,omitempty"`
	CorrelationID   string                                 `json:"correlation_id,omitempty"`
	CouncilRef      string                                 `json:"council_ref,omitempty"`
	AuthorRef       string                                 `json:"author_ref,omitempty"`
	Seats           []MCPCouncilSeatV0                     `json:"seats,omitempty"`
	Warnings        []string                               `json:"warnings,omitempty"`
	Outcome         string                                 `json:"outcome,omitempty"`
	Approvals       int                                    `json:"approvals,omitempty"`
	Reworks         int                                    `json:"reworks,omitempty"`
	Blocks          int                                    `json:"blocks,omitempty"`
	Total           int                                    `json:"total,omitempty"`
	Rationale       string                                 `json:"rationale,omitempty"`
	ErroresPublicos []MCPToolCapabilitiesListPublicErrorV0 `json:"errores_publicos,omitempty"`
}

type MCPCouncilPortV0 interface {
	ConveneCouncilV0(context.Context, MCPCouncilToolInputV0) (MCPCouncilToolResultV0, error)
}

type MCPCouncilToolExecutorV0 struct {
	Council           MCPCouncilPortV0
	ClassifyPublicErr MCPCouncilPublicErrorClassifierV0
}

func MCPCouncilDescriptorV0() MCPToolCapabilitiesListToolDescriptorV0 {
	return MCPToolCapabilitiesListToolDescriptorV0{
		Name:        MCPCouncilToolNameV0,
		Version:     MCPCouncilToolVersionV0,
		InputSchema: "council_convene:{schema_version?,request_ref?,correlation_id?,action,council_ref?,author_ref?,security_critical?,members?[],overrides?[],ballots?[]}",
		Output:      "ok:{estado,council_ref?,author_ref?,seats?[]{role,member_ref,material,forced,automatic_member_ref,budget_warning},warnings?[],outcome?,approvals?,reworks?,blocks?,total?,rationale?}|error:{estado,errores_publicos[]{code,field?}}",
		ResourceURI: MCPCouncilResourceURIV0,
		Invariantes: []string{
			"los roles se asignan en caliente por presupuesto; ningun modelo tiene rol fijo",
			"el override manual del operador gana a la asignacion automatica y deja evidencia",
			"el override no puede hacer que el autor se revise ni poner de adversario a su familia",
			"un miembro con varios roles conserva una sola voz",
			"umbral de dos tercios; el empate es rework; el veto de seguridad no lo levanta ninguna mayoria",
			"puerto sin cablear se delata antes de validar la entrada",
		},
	}
}

func (executor MCPCouncilToolExecutorV0) Execute(
	ctx context.Context,
	input MCPCouncilToolInputV0,
) (MCPCouncilToolResultV0, error) {
	input = normalizeMCPCouncilInputV0(input)
	if executor.Council == nil {
		return newMCPCouncilErrorV0(input, MCPCouncilErrPortUnavailableV0, "council"), nil
	}
	if input.Action == "" {
		return newMCPCouncilErrorV0(input, MCPCouncilErrActionRequiredV0, "action"), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := executor.Council.ConveneCouncilV0(ctx, input)
	if err != nil {
		code, field := MCPCouncilErrConveneFailedV0, "action"
		if executor.ClassifyPublicErr != nil {
			if clasificado, campo, ok := executor.ClassifyPublicErr(err); ok {
				code, field = clasificado, campo
			}
		}
		return newMCPCouncilErrorV0(input, code, field), nil
	}
	result.Estado = MCPCouncilEstadoOKV0
	result.RequestRef = input.RequestRef
	result.CorrelationID = input.CorrelationID
	return result, nil
}

func mcpCouncilTransportHandlerV0(executor MCPTransportCouncilExecutorV0) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPCouncilToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if executor == nil {
			return mcpTransportToolErrorPayloadV0(MCPCouncilToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func normalizeMCPCouncilInputV0(input MCPCouncilToolInputV0) MCPCouncilToolInputV0 {
	input.SchemaVersion = strings.TrimSpace(input.SchemaVersion)
	input.RequestRef = strings.TrimSpace(input.RequestRef)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.Action = strings.TrimSpace(input.Action)
	input.CouncilRef = strings.TrimSpace(input.CouncilRef)
	input.AuthorRef = strings.TrimSpace(input.AuthorRef)
	return input
}

func newMCPCouncilErrorV0(input MCPCouncilToolInputV0, code string, field string) MCPCouncilToolResultV0 {
	return MCPCouncilToolResultV0{
		Estado:          MCPCouncilEstadoErrorV0,
		RequestRef:      input.RequestRef,
		CorrelationID:   input.CorrelationID,
		CouncilRef:      input.CouncilRef,
		ErroresPublicos: []MCPToolCapabilitiesListPublicErrorV0{{Code: code, Field: field}},
	}
}
