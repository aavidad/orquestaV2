package orquestaruntimeollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const DefaultOllamaBaseURLV0 = "http://localhost:11434"

type OllamaModelManagerV0 struct {
	BaseURL        string
	HTTPClient     *http.Client
	BearerToken    string
	DefaultPullTTL time.Duration
}

func NewOllamaModelManagerV0(baseURL string, client *http.Client) OllamaModelManagerV0 {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultOllamaBaseURLV0
	}
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	return OllamaModelManagerV0{
		BaseURL:    strings.TrimSpace(baseURL),
		HTTPClient: client,
	}
}

func (m OllamaModelManagerV0) ListRuntimeModelsV0(
	ctx context.Context,
	req orquestaruntime.RuntimeModelListRequestV0,
) (orquestaruntime.RuntimeModelListResultV0, error) {
	baseURL := m.baseURL()
	var payload ollamaTagsResponseV0
	if err := m.doJSONV0(ctx, http.MethodGet, baseURL, "/api/tags", nil, &payload); err != nil {
		return orquestaruntime.RuntimeModelListResultV0{}, err
	}
	models := make([]orquestaruntime.RuntimeModelInfoV0, 0, len(payload.Models))
	for _, model := range payload.Models {
		models = append(models, ollamaModelInfoV0(model, "available"))
	}
	return orquestaruntime.RuntimeModelListResultV0{
		ProviderRef: firstNonBlankV0(req.ProviderRef, "ollama"),
		EndpointRef: req.EndpointRef,
		BaseURL:     redactOllamaBaseURLForResultV0(baseURL),
		Models:      models,
		Evidence: []orquestaruntime.RuntimeModelEvidence{
			{Kind: "ollama_endpoint", Ref: "api/tags"},
		},
	}, nil
}

func (m OllamaModelManagerV0) RuntimeModelStatusV0(
	ctx context.Context,
	req orquestaruntime.RuntimeModelListRequestV0,
) (orquestaruntime.RuntimeModelListResultV0, error) {
	baseURL := m.baseURL()
	var payload ollamaProcessResponseV0
	if err := m.doJSONV0(ctx, http.MethodGet, baseURL, "/api/ps", nil, &payload); err != nil {
		return orquestaruntime.RuntimeModelListResultV0{}, err
	}
	models := make([]orquestaruntime.RuntimeModelInfoV0, 0, len(payload.Models))
	for _, model := range payload.Models {
		models = append(models, ollamaModelInfoV0(model, "running"))
	}
	return orquestaruntime.RuntimeModelListResultV0{
		ProviderRef: firstNonBlankV0(req.ProviderRef, "ollama"),
		EndpointRef: req.EndpointRef,
		BaseURL:     redactOllamaBaseURLForResultV0(baseURL),
		Models:      models,
		Evidence: []orquestaruntime.RuntimeModelEvidence{
			{Kind: "ollama_endpoint", Ref: "api/ps"},
		},
	}, nil
}

func (m OllamaModelManagerV0) PullRuntimeModelV0(
	ctx context.Context,
	req orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return orquestaruntime.RuntimeModelActionResultV0{}, fmt.Errorf("ollama model requerido")
	}
	baseURL := m.baseURL()
	var payload ollamaActionResponseV0
	body := map[string]any{"model": model, "stream": false}
	if err := m.doJSONV0(ctx, http.MethodPost, baseURL, "/api/pull", body, &payload); err != nil {
		return orquestaruntime.RuntimeModelActionResultV0{}, err
	}
	return m.actionResultV0(req, baseURL, model, firstNonBlankV0(payload.Status, "pulled"), "api/pull"), nil
}

func (m OllamaModelManagerV0) ServeRuntimeModelV0(
	ctx context.Context,
	req orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	return m.keepAliveActionV0(ctx, req, firstNonBlankV0(req.KeepAlive, "-1"), "served")
}

func (m OllamaModelManagerV0) StopRuntimeModelV0(
	ctx context.Context,
	req orquestaruntime.RuntimeModelActionRequestV0,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	return m.keepAliveActionV0(ctx, req, "0", "stopped")
}

func (m OllamaModelManagerV0) keepAliveActionV0(
	ctx context.Context,
	req orquestaruntime.RuntimeModelActionRequestV0,
	keepAlive string,
	fallbackStatus string,
) (orquestaruntime.RuntimeModelActionResultV0, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		return orquestaruntime.RuntimeModelActionResultV0{}, fmt.Errorf("ollama model requerido")
	}
	baseURL := m.baseURL()
	var payload ollamaActionResponseV0
	body := map[string]any{
		"model":      model,
		"keep_alive": keepAlive,
		"stream":     false,
	}
	if err := m.doJSONV0(ctx, http.MethodPost, baseURL, "/api/generate", body, &payload); err != nil {
		return orquestaruntime.RuntimeModelActionResultV0{}, err
	}
	return m.actionResultV0(req, baseURL, model, firstNonBlankV0(payload.Status, fallbackStatus), "api/generate"), nil
}

func (m OllamaModelManagerV0) actionResultV0(
	req orquestaruntime.RuntimeModelActionRequestV0,
	baseURL string,
	model string,
	status string,
	endpoint string,
) orquestaruntime.RuntimeModelActionResultV0 {
	return orquestaruntime.RuntimeModelActionResultV0{
		ProviderRef: firstNonBlankV0(req.ProviderRef, "ollama"),
		EndpointRef: req.EndpointRef,
		BaseURL:     redactOllamaBaseURLForResultV0(baseURL),
		Model:       model,
		Accepted:    true,
		Status:      status,
		Evidence: []orquestaruntime.RuntimeModelEvidence{
			{Kind: "ollama_endpoint", Ref: endpoint},
		},
	}
}

func (m OllamaModelManagerV0) doJSONV0(
	ctx context.Context,
	method string,
	baseURL string,
	apiPath string,
	body any,
	out any,
) error {
	u, err := joinOllamaURLV0(baseURL, apiPath)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Accept", "application/json")
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(m.BearerToken) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(m.BearerToken))
	}
	client := m.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ollama %s %s: status %d", method, apiPath, resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (m OllamaModelManagerV0) baseURL() string {
	if strings.TrimSpace(m.BaseURL) != "" {
		return strings.TrimSpace(m.BaseURL)
	}
	return DefaultOllamaBaseURLV0
}

type ollamaTagsResponseV0 struct {
	Models []ollamaModelV0 `json:"models"`
}

type ollamaProcessResponseV0 struct {
	Models []ollamaModelV0 `json:"models"`
}

type ollamaActionResponseV0 struct {
	Status string `json:"status"`
}

type ollamaModelV0 struct {
	Name       string             `json:"name"`
	Model      string             `json:"model"`
	ModifiedAt string             `json:"modified_at"`
	Size       int64              `json:"size"`
	SizeVRAM   int64              `json:"size_vram"`
	Digest     string             `json:"digest"`
	ExpiresAt  string             `json:"expires_at"`
	Details    ollamaModelDetails `json:"details"`
}

type ollamaModelDetails struct {
	Family          string `json:"family"`
	ParameterSize   string `json:"parameter_size"`
	Quantization    string `json:"quantization_level"`
	ParentModel     string `json:"parent_model"`
	Format          string `json:"format"`
	Template        string `json:"template"`
	System          string `json:"system"`
	License         any    `json:"license"`
	ModifiedAt      string `json:"modified_at"`
	GeneralArch     string `json:"general.architecture"`
	GeneralFileType string `json:"general.file_type"`
}

func ollamaModelInfoV0(model ollamaModelV0, status string) orquestaruntime.RuntimeModelInfoV0 {
	name := firstNonBlankV0(model.Name, model.Model)
	return orquestaruntime.RuntimeModelInfoV0{
		Name:       name,
		Provider:   "ollama",
		Family:     firstNonBlankV0(model.Details.Family, model.Details.GeneralArch),
		Parameter:  model.Details.ParameterSize,
		Quant:      firstNonBlankV0(model.Details.Quantization, model.Details.GeneralFileType),
		SizeBytes:  model.Size,
		SizeVRAM:   model.SizeVRAM,
		Digest:     model.Digest,
		ModifiedAt: firstNonBlankV0(model.ModifiedAt, model.Details.ModifiedAt),
		ExpiresAt:  model.ExpiresAt,
		Status:     status,
	}
}

func joinOllamaURLV0(baseURL string, apiPath string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("ollama base_url invalida")
	}
	parsed.Path = path.Join(strings.TrimRight(parsed.Path, "/"), apiPath)
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func redactOllamaBaseURLForResultV0(baseURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func firstNonBlankV0(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
