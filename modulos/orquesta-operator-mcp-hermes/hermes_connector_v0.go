package orquestaoperatormcphermes

import (
	"net/http"
	"time"

	operator "orquesta/modulos/orquesta-operator-mcp"
	operatorclient "orquesta/modulos/orquesta-operator-mcp-client"
)

type HermesOperatorMCPConfigV0 struct {
	BaseURL          string
	MCPPath          string
	APIKey           string
	ToolNames        operatorclient.OperatorMCPClientToolNamesV0
	ConnectorRefs    operatorclient.OperatorMCPClientConnectorRefsV0
	Timeout          time.Duration
	HTTPClient       *http.Client
	MaxRequestBytes  int64
	MaxResponseBytes int64
}

func NewHermesOperatorMCPConnectorV0(
	config HermesOperatorMCPConfigV0,
) (operator.OperatorMCPConnectorV0, error) {
	client, err := NewHermesMCPJSONRPCClientV0(HermesMCPJSONRPCClientConfigV0{
		BaseURL:          config.BaseURL,
		MCPPath:          config.MCPPath,
		APIKey:           config.APIKey,
		HTTPClient:       config.HTTPClient,
		MaxRequestBytes:  config.MaxRequestBytes,
		MaxResponseBytes: config.MaxResponseBytes,
	})
	if err != nil {
		return nil, err
	}
	connector := operatorclient.NewOperatorMCPClientConnectorV0(operatorclient.OperatorMCPClientConfigV0{
		Client:        client,
		ToolNames:     config.ToolNames,
		ConnectorRefs: config.ConnectorRefs,
		Timeout:       config.Timeout,
	})
	return connector, nil
}
