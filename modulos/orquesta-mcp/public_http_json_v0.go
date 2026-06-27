package orquestamcp

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

const (
	mcpPublicHTTPJSONControlMaxBytesV0         int64 = 256 << 10
	mcpPublicHTTPJSONAutoprogrammingMaxBytesV0 int64 = 512 << 10
	mcpPublicHTTPJSONRPCMaxBytesV0             int64 = 1 << 20
	mcpPublicHTTPJSONDomainWorkMaxBytesV0      int64 = 2 << 20
)

type mcpPublicHTTPJSONProfileV0 string

type mcpPublicHTTPJSONUnknownFieldsPolicyV0 string

const (
	mcpPublicHTTPJSONProfileControlV0         mcpPublicHTTPJSONProfileV0 = "control_plane"
	mcpPublicHTTPJSONProfileAutoprogrammingV0 mcpPublicHTTPJSONProfileV0 = "autoprogramming"
	mcpPublicHTTPJSONProfileJSONRPCV0         mcpPublicHTTPJSONProfileV0 = "mcp_json_rpc"
	mcpPublicHTTPJSONProfileDomainWorkV0      mcpPublicHTTPJSONProfileV0 = "domain_work"

	mcpPublicHTTPJSONUnknownFieldsLegacyV0 mcpPublicHTTPJSONUnknownFieldsPolicyV0 = "legacy_accept_unknown_fields"
)

func decodeMCPPublicHTTPJSONV0(w http.ResponseWriter, r *http.Request, dst any) string {
	return decodeMCPPublicHTTPJSONProfileV0(w, r, dst, mcpPublicHTTPJSONProfileControlV0)
}

func decodeMCPPublicHTTPJSONProfileV0(
	w http.ResponseWriter,
	r *http.Request,
	dst any,
	profile mcpPublicHTTPJSONProfileV0,
) string {
	if !mcpPublicHTTPContentTypeAllowsJSONV0(r.Header.Get("Content-Type")) {
		return MCPPublicErrContentTypeInvalidV0
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, mcpPublicHTTPJSONMaxBytesV0(profile)))
	if err := decoder.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return MCPPublicErrBodyTooLargeV0
		}
		return MCPPublicErrBodyInvalidV0
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return MCPPublicErrBodyTrailingDataV0
	}
	return ""
}

func mcpPublicHTTPJSONUnknownFieldsPolicyForProfileV0(profile mcpPublicHTTPJSONProfileV0) mcpPublicHTTPJSONUnknownFieldsPolicyV0 {
	switch profile {
	default:
		return mcpPublicHTTPJSONUnknownFieldsLegacyV0
	}
}

func mcpPublicHTTPJSONMaxBytesV0(profile mcpPublicHTTPJSONProfileV0) int64 {
	switch profile {
	case mcpPublicHTTPJSONProfileAutoprogrammingV0:
		return mcpPublicHTTPJSONAutoprogrammingMaxBytesV0
	case mcpPublicHTTPJSONProfileJSONRPCV0:
		return mcpPublicHTTPJSONRPCMaxBytesV0
	case mcpPublicHTTPJSONProfileDomainWorkV0:
		return mcpPublicHTTPJSONDomainWorkMaxBytesV0
	default:
		return mcpPublicHTTPJSONControlMaxBytesV0
	}
}

func mcpPublicHTTPContentTypeAllowsJSONV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(value, ";")[0])
	}
	mediaType = strings.ToLower(mediaType)
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func publicMCPExecutorErrorMessageV0(fallback string) string {
	if fallback == "" {
		return "executor_error"
	}
	return fallback
}

func publicMCPExecutorErrorMessageFromErrorV0(fallback string, err error) string {
	base := publicMCPExecutorErrorMessageV0(fallback)
	if err == nil {
		return base
	}
	if MCPPublicErrorCodeKnownV0(err.Error()) {
		return base + ": " + strings.TrimSpace(err.Error())
	}
	return base
}

func publicMCPErrorMessageFromTextV0(fallback string, message string) string {
	message = strings.TrimSpace(message)
	if MCPPublicErrorCodeKnownV0(message) {
		return message
	}
	return publicMCPExecutorErrorMessageV0(fallback)
}
