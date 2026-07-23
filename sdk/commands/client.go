// Package commands provides a registry-independent HTTP command client.
package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Result is the public transport envelope. It intentionally duplicates the
// wire contract instead of exposing Orquesta's internal command package.
type Result struct {
	CommandID      string          `json:"command_id"`
	CommandVersion string          `json:"command_version"`
	RequestRef     string          `json:"request_ref"`
	Data           json.RawMessage `json:"data,omitempty"`
	Failure        *Failure        `json:"failure,omitempty"`
	AuditRef       string          `json:"audit_ref,omitempty"`
}

type Failure struct {
	Code       string `json:"code"`
	MessageKey string `json:"message_key"`
}

const (
	CodeInvalidRequest  = "invalid_request"
	CodeUnauthenticated = "unauthenticated"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeUnavailable     = "unavailable"
	CodeInternal        = "internal"
)

// ErrRedirectRejected reports a server redirect that the command client has
// intentionally not followed. Commands never replay requests to a new target.
var ErrRedirectRejected = errors.New("commandsdk.redirect_rejected")

type Request struct {
	CommandID           string
	Version             string
	RequestRef          string
	ProjectRef          string
	ClaimedExecutionRef string
	Payload             json.RawMessage
}

type Config struct {
	BaseURL          string
	HTTPClient       *http.Client
	MaxResponseBytes int64
}
type Client struct {
	baseURL string
	http    *http.Client
	maximum int64
}

func New(config Config) (*Client, error) {
	baseURL, ok := canonicalBaseURL(config.BaseURL)
	if !ok || config.HTTPClient == nil || config.MaxResponseBytes <= 0 {
		return nil, errors.New("commandsdk.config_invalid")
	}
	httpClient := *config.HTTPClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return ErrRedirectRejected
	}
	return &Client{baseURL: baseURL, http: &httpClient, maximum: config.MaxResponseBytes}, nil
}

func (client *Client) Invoke(ctx context.Context, request Request) (Result, error) {
	if client == nil || ctx == nil {
		return Result{}, errors.New("commandsdk.unavailable")
	}
	if !validCommandID(request.CommandID) || !validVersion(request.Version) ||
		!validRef(request.RequestRef) || !validRef(request.ProjectRef) ||
		(request.ClaimedExecutionRef != "" && !validRef(request.ClaimedExecutionRef)) ||
		!json.Valid(request.Payload) {
		return Result{}, errors.New("commandsdk.command_invalid")
	}
	body, err := json.Marshal(struct {
		Version             string          `json:"version"`
		RequestRef          string          `json:"request_ref"`
		ProjectRef          string          `json:"project_ref"`
		ClaimedExecutionRef string          `json:"claimed_execution_ref,omitempty"`
		Payload             json.RawMessage `json:"payload"`
	}{request.Version, request.RequestRef, request.ProjectRef, request.ClaimedExecutionRef, request.Payload})
	if err != nil {
		return Result{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/api/v1/commands/"+url.PathEscape(request.CommandID), bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(httpRequest)
	if err != nil {
		if errors.Is(err, ErrRedirectRejected) {
			return Result{}, ErrRedirectRejected
		}
		return Result{}, err
	}
	defer response.Body.Close()
	encoded, err := io.ReadAll(io.LimitReader(response.Body, client.maximum+1))
	if err != nil || int64(len(encoded)) > client.maximum {
		return Result{}, errors.New("commandsdk.response_invalid")
	}
	var result Result
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return Result{}, errors.New("commandsdk.response_invalid")
	}
	if result.CommandID != request.CommandID || result.CommandVersion != request.Version ||
		result.RequestRef != request.RequestRef || response.StatusCode != resultHTTPStatus(result.Failure) {
		return Result{}, errors.New("commandsdk.response_invalid")
	}
	return result, nil
}

func canonicalBaseURL(raw string) (string, bool) {
	if raw == "" || strings.TrimSpace(raw) != raw || strings.Contains(raw, "#") {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.Opaque != "" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") || (parsed.RawPath != "" && parsed.RawPath != "/") {
		return "", false
	}
	hostname := parsed.Hostname()
	if !validHostname(hostname) || !validPort(parsed) {
		return "", false
	}
	if parsed.Scheme == "http" && !isLoopbackHost(hostname) {
		return "", false
	}
	parsed.Path, parsed.RawPath = "", ""
	return parsed.String(), true
}

func validHostname(hostname string) bool {
	if hostname == "" || len(hostname) > 253 {
		return false
	}
	if net.ParseIP(hostname) != nil {
		return true
	}
	name := strings.TrimSuffix(hostname, ".")
	if name == "" {
		return false
	}
	for _, label := range strings.Split(name, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for index := range label {
			character := label[index]
			if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
				(character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}
	return true
}

func validPort(parsed *url.URL) bool {
	port := parsed.Port()
	if port == "" {
		return !strings.HasSuffix(parsed.Host, ":") && !strings.HasSuffix(parsed.Host, "]:")
	}
	number, err := strconv.Atoi(port)
	return err == nil && number >= 1 && number <= 65535
}

func isLoopbackHost(hostname string) bool {
	if strings.EqualFold(hostname, "localhost") {
		return true
	}
	address := net.ParseIP(hostname)
	return address != nil && address.IsLoopback()
}

func validCommandID(value string) bool {
	const prefix = "orquesta."
	if len(value) <= len(prefix) || len(value) > 255 || !strings.HasPrefix(value, prefix) {
		return false
	}
	for _, segment := range strings.Split(strings.TrimPrefix(value, prefix), ".") {
		if len(segment) == 0 || segment[0] < 'a' || segment[0] > 'z' {
			return false
		}
		for index := 1; index < len(segment); index++ {
			character := segment[index]
			if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '_' {
				return false
			}
		}
	}
	return true
}

func validVersion(value string) bool {
	if value == "" || len(value) > 10 || value[0] < '1' || value[0] > '9' {
		return false
	}
	for index := 1; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func validRef(value string) bool {
	if value == "" || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func resultHTTPStatus(failure *Failure) int {
	if failure == nil {
		return http.StatusOK
	}
	switch failure.Code {
	case CodeInvalidRequest:
		return http.StatusBadRequest
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeUnavailable:
		return http.StatusServiceUnavailable
	case CodeInternal:
		return http.StatusInternalServerError
	default:
		return 0
	}
}
