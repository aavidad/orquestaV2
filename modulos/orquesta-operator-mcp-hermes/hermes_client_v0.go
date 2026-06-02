package orquestaoperatormcphermes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"

	operator "orquesta/modulos/orquesta-operator-mcp"
)

const (
	defaultHermesMCPPathV0                   = "/mcp"
	defaultHermesMCPMaxRequestBytesV0  int64 = 1 << 20
	defaultHermesMCPMaxResponseBytesV0 int64 = 1 << 20
)

type HermesMCPJSONRPCClientConfigV0 struct {
	BaseURL          string
	MCPPath          string
	APIKey           string
	HTTPClient       *http.Client
	MaxRequestBytes  int64
	MaxResponseBytes int64
}

type HermesMCPJSONRPCClientV0 struct {
	endpointURL      string
	apiKey           string
	httpClient       *http.Client
	maxRequestBytes  int64
	maxResponseBytes int64
	nextID           atomic.Uint64
}

type hermesMCPJSONRPCRequestV0 struct {
	JSONRPC string                    `json:"jsonrpc"`
	ID      string                    `json:"id"`
	Method  string                    `json:"method"`
	Params  hermesMCPToolCallParamsV0 `json:"params"`
}

type hermesMCPToolCallParamsV0 struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments"`
}

type hermesMCPJSONRPCResponseV0 struct {
	JSONRPC string                   `json:"jsonrpc,omitempty"`
	ID      any                      `json:"id,omitempty"`
	Result  json.RawMessage          `json:"result,omitempty"`
	Error   *hermesMCPJSONRPCErrorV0 `json:"error,omitempty"`
}

type hermesMCPJSONRPCErrorV0 struct {
	Code    int             `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type hermesMCPToolCallResultV0 struct {
	Content []hermesMCPTextContentV0 `json:"content,omitempty"`
	IsError bool                     `json:"isError,omitempty"`
}

type hermesMCPTextContentV0 struct {
	Type     string `json:"type,omitempty"`
	Text     string `json:"text,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

func NewHermesMCPJSONRPCClientV0(
	config HermesMCPJSONRPCClientConfigV0,
) (*HermesMCPJSONRPCClientV0, error) {
	endpointURL, err := hermesEndpointURLV0(config.BaseURL, config.MCPPath)
	if err != nil {
		return nil, err
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &HermesMCPJSONRPCClientV0{
		endpointURL:      endpointURL,
		apiKey:           strings.TrimSpace(config.APIKey),
		httpClient:       httpClient,
		maxRequestBytes:  positiveInt64OrDefaultV0(config.MaxRequestBytes, defaultHermesMCPMaxRequestBytesV0),
		maxResponseBytes: positiveInt64OrDefaultV0(config.MaxResponseBytes, defaultHermesMCPMaxResponseBytesV0),
	}, nil
}

func (client *HermesMCPJSONRPCClientV0) CallToolV0(
	ctx context.Context,
	toolName string,
	input any,
	output any,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(toolName) == "" || output == nil {
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	payload, err := json.Marshal(hermesMCPJSONRPCRequestV0{
		JSONRPC: "2.0",
		ID:      client.nextRequestIDV0(),
		Method:  "tools/call",
		Params: hermesMCPToolCallParamsV0{
			Name:      strings.TrimSpace(toolName),
			Arguments: input,
		},
	})
	if err != nil || int64(len(payload)) > client.maxRequestBytes {
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpointURL, bytes.NewReader(payload))
	if err != nil {
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if client.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+client.apiKey)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return hermesHTTPStatusErrorV0(response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, client.maxResponseBytes+1))
	if err != nil {
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	if int64(len(body)) > client.maxResponseBytes {
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	return decodeHermesMCPResponseV0(body, output)
}

func (client *HermesMCPJSONRPCClientV0) nextRequestIDV0() string {
	next := client.nextID.Add(1)
	return fmt.Sprintf("hermes-mcp-call-v0-%d", next)
}

func decodeHermesMCPResponseV0(body []byte, output any) error {
	var response hermesMCPJSONRPCResponseV0
	if err := json.Unmarshal(body, &response); err != nil {
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	if response.Error != nil {
		return hermesJSONRPCPublicErrorV0(*response.Error)
	}
	if len(bytes.TrimSpace(response.Result)) == 0 {
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	if err := decodeHermesMCPToolResultV0(response.Result, output); err != nil {
		return err
	}
	return nil
}

func decodeHermesMCPToolResultV0(result json.RawMessage, output any) error {
	var toolResult hermesMCPToolCallResultV0
	if err := json.Unmarshal(result, &toolResult); err == nil && len(toolResult.Content) > 0 {
		for _, item := range toolResult.Content {
			if strings.EqualFold(strings.TrimSpace(item.Type), "text") && strings.TrimSpace(item.Text) != "" {
				if err := json.Unmarshal([]byte(item.Text), output); err != nil {
					return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
				}
				return nil
			}
		}
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	if err := json.Unmarshal(result, output); err != nil {
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
	return nil
}

func hermesJSONRPCPublicErrorV0(err hermesMCPJSONRPCErrorV0) error {
	if code := publicErrorCodeFromHermesDataV0(err.Data); code != "" {
		return operator.NewOperatorMCPPublicErrorV0(code)
	}
	if code := publicErrorCodeFromStringV0(err.Message); code != "" {
		return operator.NewOperatorMCPPublicErrorV0(code)
	}
	return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
}

func publicErrorCodeFromHermesDataV0(data json.RawMessage) string {
	if len(bytes.TrimSpace(data)) == 0 {
		return ""
	}
	var envelope struct {
		ErrorCode string `json:"error_code"`
		Code      string `json:"code"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil {
		if code := publicErrorCodeFromStringV0(envelope.ErrorCode); code != "" {
			return code
		}
		return publicErrorCodeFromStringV0(envelope.Code)
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err == nil {
		return publicErrorCodeFromStringV0(raw)
	}
	return ""
}

func publicErrorCodeFromStringV0(value string) string {
	code := strings.TrimSpace(value)
	if _, ok := operator.PublicOperatorMCPErrorCodeV0(errors.New(code)); ok {
		return code
	}
	return ""
}

func hermesHTTPStatusErrorV0(status int) error {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
	default:
		return operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPPortErrorV0)
	}
}

func hermesEndpointURLV0(baseURL string, mcpPath string) (string, error) {
	raw := strings.TrimSpace(baseURL)
	if raw == "" {
		return "", operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if path := normalizeHermesMCPPathV0(mcpPath); path != "" {
		parsed.Path = path
	} else if strings.TrimSpace(parsed.Path) == "" || parsed.Path == "/" {
		parsed.Path = defaultHermesMCPPathV0
	}
	if parsed.Path == "/api/mcp" {
		return "", operator.NewOperatorMCPPublicErrorV0(operator.ErrOperatorMCPConnectorUnavailableV0)
	}
	return parsed.String(), nil
}

func normalizeHermesMCPPathV0(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func positiveInt64OrDefaultV0(value int64, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}
