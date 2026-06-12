package orquestaappcodexstack

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
)

const (
	PrivacyFilterSidecarHTTPRequestSchemaVersionV0  = "privacy_filter_sidecar_request.v0"
	PrivacyFilterSidecarHTTPResponseSchemaVersionV0 = "privacy_filter_sidecar_response.v0"

	defaultPrivacyFilterSidecarHTTPTimeoutV0          = 2 * time.Second
	defaultPrivacyFilterSidecarHTTPMaxResponseBytesV0 = int64(64 << 10)
)

type PrivacyFilterSidecarHTTPConfigV0 struct {
	Enabled          bool
	EndpointURL      string
	HTTPClient       *http.Client
	Timeout          time.Duration
	MaxResponseBytes int64
	SidecarRef       string
	AdapterRef       string
	TransportRef     string
	EvidenceRef      string
}

type PrivacyFilterSidecarHTTPPublicConfigV0 struct {
	Enabled                 bool   `json:"enabled"`
	LocalEndpointConfigured bool   `json:"local_endpoint_configured"`
	SidecarRef              string `json:"sidecar_ref,omitempty"`
	AdapterRef              string `json:"adapter_ref,omitempty"`
	TransportRef            string `json:"transport_ref,omitempty"`
	EvidenceRef             string `json:"evidence_ref,omitempty"`
}

type PrivacyFilterSidecarHTTPRequestV0 struct {
	SchemaVersion string `json:"schema_version"`
	RequestRef    string `json:"request_ref,omitempty"`
	EntryRef      string `json:"entry_ref,omitempty"`
	SourceRef     string `json:"source_ref,omitempty"`
	Payload       string `json:"payload"`
}

type PrivacyFilterSidecarHTTPResponseV0 struct {
	SchemaVersion    string   `json:"schema_version,omitempty"`
	Content          string   `json:"content,omitempty"`
	Sanitized        bool     `json:"sanitized,omitempty"`
	ReviewRequired   bool     `json:"review_required,omitempty"`
	EvidenceRefs     []string `json:"evidence_refs,omitempty"`
	Categories       []string `json:"categories,omitempty"`
	ReplacementCount int      `json:"replacement_count,omitempty"`
}

type PrivacyFilterSidecarHTTPPortV0 struct {
	config      PrivacyFilterSidecarHTTPConfigV0
	endpointURL string
	client      *http.Client
}

func NewPrivacyFilterSidecarHTTPPortV0(
	config PrivacyFilterSidecarHTTPConfigV0,
) (PrivacyFilterSidecarHTTPPortV0, error) {
	config.EndpointURL = strings.TrimSpace(config.EndpointURL)
	if !config.Enabled {
		return PrivacyFilterSidecarHTTPPortV0{config: config}, nil
	}
	endpoint, err := validatePrivacyFilterSidecarLocalEndpointV0(config.EndpointURL)
	if err != nil {
		return PrivacyFilterSidecarHTTPPortV0{}, err
	}
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return PrivacyFilterSidecarHTTPPortV0{
		config:      config,
		endpointURL: endpoint.String(),
		client:      client,
	}, nil
}

func RedactedPrivacyFilterSidecarHTTPConfigV0(
	config PrivacyFilterSidecarHTTPConfigV0,
) PrivacyFilterSidecarHTTPPublicConfigV0 {
	return PrivacyFilterSidecarHTTPPublicConfigV0{
		Enabled:                 config.Enabled,
		LocalEndpointConfigured: strings.TrimSpace(config.EndpointURL) != "",
		SidecarRef:              localSensitiveDataSafeRefValueV0(config.SidecarRef, "sidecar-ref"),
		AdapterRef:              localSensitiveDataSafeRefValueV0(config.AdapterRef, "adapter-ref"),
		TransportRef:            localSensitiveDataSafeRefValueV0(config.TransportRef, "transport-ref"),
		EvidenceRef:             localSensitiveDataSafeRefValueV0(config.EvidenceRef, "evidence-ref"),
	}
}

func (port PrivacyFilterSidecarHTTPPortV0) FilterEgressPayloadV0(
	request PrivacyFilterSidecarRequestV0,
) PrivacyFilterSidecarResultV0 {
	if !port.config.Enabled || strings.TrimSpace(port.endpointURL) == "" {
		return PrivacyFilterSidecarResultV0{
			Content:      request.Payload,
			EvidenceRefs: port.defaultEvidenceRefsV0(request),
			Categories:   []string{"privacy_filter_sidecar_disabled"},
		}
	}

	body := bytes.Buffer{}
	err := json.NewEncoder(&body).Encode(PrivacyFilterSidecarHTTPRequestV0{
		SchemaVersion: PrivacyFilterSidecarHTTPRequestSchemaVersionV0,
		RequestRef:    request.RequestRef,
		EntryRef:      request.EntryRef,
		SourceRef:     request.SourceRef,
		Payload:       request.Payload,
	})
	if err != nil {
		return port.errorResultV0(request, "privacy_filter_sidecar_request_encode_error")
	}

	ctx := context.Background()
	cancel := func() {}
	timeout := port.config.Timeout
	if timeout == 0 && (port.client == nil || port.client.Timeout == 0) {
		timeout = defaultPrivacyFilterSidecarHTTPTimeoutV0
	}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, port.endpointURL, &body)
	if err != nil {
		return port.errorResultV0(request, "privacy_filter_sidecar_request_build_error")
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := port.client.Do(httpRequest)
	if err != nil {
		return port.errorResultV0(request, "privacy_filter_sidecar_request_error")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return port.errorResultV0(request, fmt.Sprintf("privacy_filter_sidecar_http_%d", response.StatusCode))
	}

	limit := port.config.MaxResponseBytes
	if limit <= 0 {
		limit = defaultPrivacyFilterSidecarHTTPMaxResponseBytesV0
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return port.errorResultV0(request, "privacy_filter_sidecar_response_read_error")
	}
	if int64(len(raw)) > limit {
		return port.errorResultV0(request, "privacy_filter_sidecar_response_too_large")
	}

	var decoded PrivacyFilterSidecarHTTPResponseV0
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return port.errorResultV0(request, "privacy_filter_sidecar_response_decode_error")
	}
	return port.resultFromHTTPResponseV0(request, decoded)
}

func (port PrivacyFilterSidecarHTTPPortV0) resultFromHTTPResponseV0(
	request PrivacyFilterSidecarRequestV0,
	response PrivacyFilterSidecarHTTPResponseV0,
) PrivacyFilterSidecarResultV0 {
	content := response.Content
	reviewRequired := response.ReviewRequired
	if strings.TrimSpace(content) == "" && response.Sanitized {
		reviewRequired = true
	}
	if strings.TrimSpace(content) == "" && !reviewRequired {
		content = request.Payload
	}
	categories := compactPrivacyFilterSidecarCategoriesV0(response.Categories)
	if len(categories) == 0 {
		categories = []string{"privacy_filter_sidecar_checked"}
	}
	return PrivacyFilterSidecarResultV0{
		Content:          content,
		Sanitized:        response.Sanitized || content != request.Payload,
		ReviewRequired:   reviewRequired,
		EvidenceRefs:     port.safeEvidenceRefsV0(request, response.EvidenceRefs),
		Categories:       categories,
		ReplacementCount: response.ReplacementCount,
	}
}

func (port PrivacyFilterSidecarHTTPPortV0) errorResultV0(
	request PrivacyFilterSidecarRequestV0,
	category string,
) PrivacyFilterSidecarResultV0 {
	return PrivacyFilterSidecarResultV0{
		ReviewRequired: true,
		EvidenceRefs:   port.defaultEvidenceRefsV0(request),
		Categories:     compactPrivacyFilterSidecarCategoriesV0([]string{"privacy_filter_sidecar_error", category}),
	}
}

func (port PrivacyFilterSidecarHTTPPortV0) safeEvidenceRefsV0(
	request PrivacyFilterSidecarRequestV0,
	refs []string,
) []string {
	if len(refs) == 0 {
		return port.defaultEvidenceRefsV0(request)
	}
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = localSensitiveDataSafeRefValueV0(ref, "evidence-ref")
		if strings.TrimSpace(ref) != "" {
			out = append(out, ref)
		}
	}
	if len(out) == 0 {
		return port.defaultEvidenceRefsV0(request)
	}
	return out
}

func (port PrivacyFilterSidecarHTTPPortV0) defaultEvidenceRefsV0(
	request PrivacyFilterSidecarRequestV0,
) []string {
	base := firstCodexStackStringV0(
		port.config.EvidenceRef,
		port.config.SidecarRef,
		"evidence-ref-privacy-filter-sidecar",
	)
	return []string{localSensitiveDataSafeRefValueV0(
		base+"-"+safeContextSanitizerPartV0(request.EntryRef),
		"evidence-ref",
	)}
}

func validatePrivacyFilterSidecarLocalEndpointV0(raw string) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("privacy_filter_sidecar_endpoint_invalid: %w", err)
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return nil, fmt.Errorf("privacy_filter_sidecar_endpoint_requires_http")
	}
	if endpoint.Host == "" || !privacyFilterSidecarHostIsLocalV0(endpoint.Hostname()) {
		return nil, fmt.Errorf("privacy_filter_sidecar_endpoint_not_local")
	}
	return endpoint, nil
}

func privacyFilterSidecarHostIsLocalV0(host string) bool {
	host = strings.Trim(strings.ToLower(strings.TrimSpace(host)), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func compactPrivacyFilterSidecarCategoriesV0(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if privacyFilterSidecarCategoryHasForbiddenDetailV0(value) {
			value = "privacy-filter-category-" + safeContextSanitizerPartV0(value)
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func privacyFilterSidecarCategoryHasForbiddenDetailV0(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"://", "/home/", "\\home\\", "/users/", "\\users\\", "c:\\users\\",
		"$home", "~/", "sk-", "bearer ", "token=", "prompt=", "completion=",
		"transcript=", "access_token=", "refresh_token=", "api_key=", "api-key=",
		"password=", "secret=",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
