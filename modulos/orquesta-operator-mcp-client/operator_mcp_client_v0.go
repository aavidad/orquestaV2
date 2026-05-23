package orquestaoperatormcpclient

import (
	"context"
	"strings"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

type GenericMCPClientV0 interface {
	CallToolV0(ctx context.Context, toolName string, input any, output any) error
}

type OperatorMCPClientToolNamesV0 struct {
	Status        string
	Burst         string
	Outbox        string
	DirectedQuery string
}

type OperatorMCPClientConnectorRefsV0 struct {
	Status        string
	Burst         string
	Outbox        string
	DirectedQuery string
}

type OperatorMCPClientConfigV0 struct {
	Client        GenericMCPClientV0
	ToolNames     OperatorMCPClientToolNamesV0
	ConnectorRefs OperatorMCPClientConnectorRefsV0
}

type OperatorMCPClientConnectorV0 struct {
	client        GenericMCPClientV0
	toolNames     OperatorMCPClientToolNamesV0
	connectorRefs OperatorMCPClientConnectorRefsV0
}

type operatorMCPClientToolResultV0 struct {
	Estado        string                                     `json:"estado"`
	Status        *operator.OperatorMCPStatusResultV0        `json:"status,omitempty"`
	Burst         *operator.OperatorMCPBurstResultV0         `json:"burst,omitempty"`
	Outbox        *operator.OperatorMCPOutboxResultV0        `json:"outbox,omitempty"`
	DirectedQuery *operator.OperatorMCPDirectedQueryResultV0 `json:"directed_query,omitempty"`
	ErrorCode     string                                     `json:"error_code,omitempty"`
}

var _ operator.OperatorMCPConnectorV0 = OperatorMCPClientConnectorV0{}

func NewOperatorMCPClientConnectorV0(
	config OperatorMCPClientConfigV0,
) OperatorMCPClientConnectorV0 {
	return OperatorMCPClientConnectorV0{
		client:        config.Client,
		toolNames:     normalizeToolNamesV0(config.ToolNames),
		connectorRefs: normalizeConnectorRefsV0(config.ConnectorRefs),
	}
}

func (connector OperatorMCPClientConnectorV0) QueryOperatorStatusV0(
	input operator.OperatorStatusQueryV0,
) (operator.OperatorMCPStatusResultV0, error) {
	input.StatusConnectorRef = configuredRefV0(connector.connectorRefs.Status, input.StatusConnectorRef)
	result, err := connector.callToolV0(connector.toolNames.Status, input)
	if err != nil {
		return operator.OperatorMCPStatusResultV0{}, err
	}
	if err := result.publicErrorV0(); err != nil {
		return operator.OperatorMCPStatusResultV0{}, err
	}
	if result.Status == nil {
		return operator.OperatorMCPStatusResultV0{}, publicClientErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	return *result.Status, nil
}

func (connector OperatorMCPClientConnectorV0) RequestOperatorSupervisedBurstV0(
	input operator.OperatorSupervisedBurstRequestV0,
) (operator.OperatorMCPBurstResultV0, error) {
	input.BurstConnectorRef = configuredRefV0(connector.connectorRefs.Burst, input.BurstConnectorRef)
	result, err := connector.callToolV0(connector.toolNames.Burst, input)
	if err != nil {
		return operator.OperatorMCPBurstResultV0{}, err
	}
	if err := result.publicErrorV0(); err != nil {
		return operator.OperatorMCPBurstResultV0{}, err
	}
	if result.Burst == nil {
		return operator.OperatorMCPBurstResultV0{}, publicClientErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	return *result.Burst, nil
}

func (connector OperatorMCPClientConnectorV0) ListOperatorPendingOutboxV0(
	input operator.OperatorPendingOutboxQueryV0,
) (operator.OperatorMCPOutboxResultV0, error) {
	input.OutboxConnectorRef = configuredRefV0(connector.connectorRefs.Outbox, input.OutboxConnectorRef)
	result, err := connector.callToolV0(connector.toolNames.Outbox, input)
	if err != nil {
		return operator.OperatorMCPOutboxResultV0{}, err
	}
	if err := result.publicErrorV0(); err != nil {
		return operator.OperatorMCPOutboxResultV0{}, err
	}
	if result.Outbox == nil {
		return operator.OperatorMCPOutboxResultV0{}, publicClientErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	return *result.Outbox, nil
}

func (connector OperatorMCPClientConnectorV0) RaiseOperatorDirectedQueryV0(
	input operator.OperatorDirectedQueryV0,
) (operator.OperatorMCPDirectedQueryResultV0, error) {
	input.QueryConnectorRef = configuredRefV0(connector.connectorRefs.DirectedQuery, input.QueryConnectorRef)
	result, err := connector.callToolV0(connector.toolNames.DirectedQuery, input)
	if err != nil {
		return operator.OperatorMCPDirectedQueryResultV0{}, err
	}
	if err := result.publicErrorV0(); err != nil {
		return operator.OperatorMCPDirectedQueryResultV0{}, err
	}
	if result.DirectedQuery == nil {
		return operator.OperatorMCPDirectedQueryResultV0{}, publicClientErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	return *result.DirectedQuery, nil
}

func (connector OperatorMCPClientConnectorV0) callToolV0(
	toolName string,
	input any,
) (operatorMCPClientToolResultV0, error) {
	if connector.client == nil {
		return operatorMCPClientToolResultV0{}, publicClientErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
	}
	var result operatorMCPClientToolResultV0
	if err := connector.client.CallToolV0(context.Background(), toolName, input, &result); err != nil {
		if code, ok := operator.PublicOperatorMCPErrorCodeV0(err); ok {
			return operatorMCPClientToolResultV0{}, publicClientErrorV0(code)
		}
		return operatorMCPClientToolResultV0{}, publicClientErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	return result, nil
}

func (result operatorMCPClientToolResultV0) publicErrorV0() error {
	if strings.EqualFold(strings.TrimSpace(result.Estado), "ok") {
		return nil
	}
	if code := normalizePublicErrorCodeV0(result.ErrorCode); code != "" {
		return publicClientErrorV0(code)
	}
	return publicClientErrorV0(operator.ErrOperatorMCPPortErrorV0)
}

func normalizeToolNamesV0(names OperatorMCPClientToolNamesV0) OperatorMCPClientToolNamesV0 {
	names.Status = defaultStringV0(names.Status, operator.OperatorMCPStatusToolNameV0)
	names.Burst = defaultStringV0(names.Burst, operator.OperatorMCPBurstToolNameV0)
	names.Outbox = defaultStringV0(names.Outbox, operator.OperatorMCPOutboxToolNameV0)
	names.DirectedQuery = defaultStringV0(names.DirectedQuery, operator.OperatorMCPDirectedQueryToolV0)
	return names
}

func normalizeConnectorRefsV0(refs OperatorMCPClientConnectorRefsV0) OperatorMCPClientConnectorRefsV0 {
	refs.Status = strings.TrimSpace(refs.Status)
	refs.Burst = strings.TrimSpace(refs.Burst)
	refs.Outbox = strings.TrimSpace(refs.Outbox)
	refs.DirectedQuery = strings.TrimSpace(refs.DirectedQuery)
	return refs
}

func configuredRefV0(configured string, input string) string {
	if configured != "" {
		return configured
	}
	return input
}

func defaultStringV0(value string, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func normalizePublicErrorCodeV0(code string) string {
	switch strings.TrimSpace(code) {
	case operator.ErrOperatorMCPRequiredFieldV0:
		return operator.ErrOperatorMCPRequiredFieldV0
	case operator.ErrOperatorMCPOpaqueRefV0:
		return operator.ErrOperatorMCPOpaqueRefV0
	case operator.ErrOperatorMCPBudgetInvalidV0:
		return operator.ErrOperatorMCPBudgetInvalidV0
	case operator.ErrOperatorMCPLimitInvalidV0:
		return operator.ErrOperatorMCPLimitInvalidV0
	case operator.ErrOperatorMCPQuestionInvalidV0:
		return operator.ErrOperatorMCPQuestionInvalidV0
	case operator.ErrOperatorMCPSectionInvalidV0:
		return operator.ErrOperatorMCPSectionInvalidV0
	case operator.ErrOperatorMCPPortUnavailableV0:
		return operator.ErrOperatorMCPPortUnavailableV0
	case operator.ErrOperatorMCPPortErrorV0:
		return operator.ErrOperatorMCPPortErrorV0
	case operator.ErrOperatorMCPConnectorUnavailableV0:
		return operator.ErrOperatorMCPConnectorUnavailableV0
	default:
		return ""
	}
}

func publicClientErrorV0(code string) error {
	return operator.NewOperatorMCPPublicErrorV0(code)
}
