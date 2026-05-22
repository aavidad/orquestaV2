package orquestamcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
)

const (
	MCPNuevaAppToolCorrelationHeaderV0 = "X-Correlation-ID"
	MCPNuevaAppToolDefaultTimeoutV0    = 10 * time.Second
)

type MCPNuevaAppToolExecutorV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

func NewMCPNuevaAppToolExecutorV0(serverURL string, timeout time.Duration) (*MCPNuevaAppToolExecutorV0, error) {
	baseURL, err := normalizeMCPServerURLV0(serverURL)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = MCPNuevaAppToolDefaultTimeoutV0
	}
	return &MCPNuevaAppToolExecutorV0{
		BaseURL:  baseURL,
		Endpoint: orquestafactoryhttp.AppSpecHTTPPathV0,
		Timeout:  timeout,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (executor *MCPNuevaAppToolExecutorV0) Execute(ctx context.Context, input MCPNuevaAppToolInputV0) (MCPNuevaAppToolResultV0, error) {
	req, correlationID := ToAppSpecRequestV0(input)
	payload, err := json.Marshal(req)
	if err != nil {
		return MCPNuevaAppToolResultV0{}, fmt.Errorf("mcp nueva app marshal request: %w", err)
	}

	timeout := MCPNuevaAppToolDefaultTimeoutV0
	if executor != nil && executor.Timeout > 0 {
		timeout = executor.Timeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, joinMCPEndpointV0(baseURLMCPV0(executor), endpointMCPV0(executor)), bytes.NewReader(payload))
	if err != nil {
		return MCPNuevaAppToolResultV0{}, fmt.Errorf("mcp nueva app request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set(MCPNuevaAppToolCorrelationHeaderV0, correlationID)

	resp, err := httpClientMCPV0(executor, timeout).Do(httpReq)
	if err != nil {
		if isTimeoutMCPV0(err) {
			return MCPNuevaAppToolResultV0{}, fmt.Errorf("mcp nueva app timeout: %w", err)
		}
		return MCPNuevaAppToolResultV0{}, fmt.Errorf("mcp nueva app transport: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var out orquestafactoryhttp.AppSpecHTTPErrorResponseV0
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return MCPNuevaAppToolResultV0{}, fmt.Errorf("mcp nueva app decode 400: %w", err)
		}
		return NewMCPNuevaAppErrorResultV0(req.RequestID, responseCorrelationIDMCPV0(resp, correlationID), out.Errores), nil
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return MCPNuevaAppToolResultV0{}, fmt.Errorf("mcp nueva app status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out orquestafactoryhttp.AppSpecHTTPResponseV0
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return MCPNuevaAppToolResultV0{}, fmt.Errorf("mcp nueva app decode ok: %w", err)
	}
	if strings.TrimSpace(out.AppSpec.SchemaVersion) == "" || strings.TrimSpace(out.Backlog.SchemaVersion) == "" {
		return MCPNuevaAppToolResultV0{}, fmt.Errorf("mcp nueva app response incompleta")
	}
	return NewMCPNuevaAppOKResultV0(out.AppSpec, out.Backlog, responseCorrelationIDMCPV0(resp, correlationID)), nil
}

func normalizeMCPServerURLV0(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("server_url requerida")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("server_url invalida")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("server_url no debe contener credenciales")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("server_url debe usar http o https")
	}
	if strings.TrimSpace(parsed.Host) == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("server_url debe ser base http")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func baseURLMCPV0(executor *MCPNuevaAppToolExecutorV0) string {
	if executor == nil {
		return ""
	}
	return strings.TrimSpace(executor.BaseURL)
}

func endpointMCPV0(executor *MCPNuevaAppToolExecutorV0) string {
	if executor != nil && strings.TrimSpace(executor.Endpoint) != "" {
		return strings.TrimSpace(executor.Endpoint)
	}
	return orquestafactoryhttp.AppSpecHTTPPathV0
}

func joinMCPEndpointV0(baseURL, endpoint string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func httpClientMCPV0(executor *MCPNuevaAppToolExecutorV0, timeout time.Duration) *http.Client {
	if executor != nil && executor.HTTPClient != nil {
		return executor.HTTPClient
	}
	return &http.Client{Timeout: timeout}
}

func responseCorrelationIDMCPV0(resp *http.Response, fallback string) string {
	if resp != nil {
		if header := strings.TrimSpace(resp.Header.Get(MCPNuevaAppToolCorrelationHeaderV0)); header != "" {
			return header
		}
	}
	return strings.TrimSpace(fallback)
}

func isTimeoutMCPV0(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "context deadline exceeded")
}
