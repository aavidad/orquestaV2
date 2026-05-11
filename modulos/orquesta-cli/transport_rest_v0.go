package orquestacli

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	CliCorrelationHeaderV0  = "X-Correlation-ID"
	CliDefaultTimeoutRESTV0 = 10 * time.Second
)

type cliRESTClientConfigV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

func newCLIRESTClientConfigV0(serverURL string, timeout time.Duration, endpoint string) (cliRESTClientConfigV0, error) {
	baseURL, err := normalizeServerURLV0(serverURL)
	if err != nil {
		return cliRESTClientConfigV0{}, err
	}
	return cliRESTClientConfigV0{
		BaseURL:  baseURL,
		Endpoint: strings.TrimSpace(endpoint),
		Timeout:  effectiveTimeoutV0(timeout, 0),
	}, nil
}

func prepareCLIRESTRequestV0(ctx context.Context, baseURL string, endpoint string, correlationID string, payload []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinEndpointV0(baseURL, endpoint), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(CliCorrelationHeaderV0, strings.TrimSpace(correlationID))
	return req, nil
}

func httpClientFromConfigV0(config cliRESTClientConfigV0, timeout time.Duration) *http.Client {
	if config.HTTPClient != nil {
		return config.HTTPClient
	}
	return &http.Client{Timeout: timeout}
}

func clientTimeoutFromConfigV0(config cliRESTClientConfigV0) time.Duration {
	return config.Timeout
}

func effectiveTimeoutV0(primary, fallback time.Duration) time.Duration {
	if primary > 0 {
		return primary
	}
	if fallback > 0 {
		return fallback
	}
	return CliDefaultTimeoutRESTV0
}

func normalizeServerURLV0(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", NewCliClientErrorV0(CliErrConfiguracionInvalidaV0, "server_url", "server_url_requerida", 0, false)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", NewCliClientErrorV0(CliErrConfiguracionInvalidaV0, "server_url", "server_url_invalida", 0, false)
	}
	if parsed.User != nil {
		return "", NewCliClientErrorV0(CliErrConfiguracionInvalidaV0, "server_url", "server_url_no_debe_contener_credenciales", 0, false)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", NewCliClientErrorV0(CliErrConfiguracionInvalidaV0, "server_url", "server_url_debe_usar_http_o_https", 0, false)
	}
	if strings.TrimSpace(parsed.Host) == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", NewCliClientErrorV0(CliErrConfiguracionInvalidaV0, "server_url", "server_url_debe_ser_base_http", 0, false)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func joinEndpointV0(baseURL, endpoint string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(endpoint, "/")
}

func retryableStatusV0(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func isTimeoutErrorV0(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
