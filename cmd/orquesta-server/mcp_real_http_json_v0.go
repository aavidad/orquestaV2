package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

const (
	serverPublicHTTPJSONControlMaxBytesV0 int64 = 256 << 10
	serverPublicHTTPJSONMCPMaxBytesV0     int64 = mcpJSONRPCMaxBodyBytesV0
)

type serverPublicHTTPJSONProfileV0 string

const (
	serverPublicHTTPJSONProfileControlV0 serverPublicHTTPJSONProfileV0 = "control_plane"
	serverPublicHTTPJSONProfileMCPV0     serverPublicHTTPJSONProfileV0 = "mcp_json_rpc"
)

func decodeMCPJSONRPCRequestV0(w http.ResponseWriter, r *http.Request, request *mcpJSONRPCRequestV0) string {
	if !mcpJSONRPCAcceptAllowsJSONV0(r.Header.Get("Accept")) {
		return serverPublicErrAcceptInvalidV0
	}
	raw, code := decodeServerPublicHTTPJSONRawV0(w, r, serverPublicHTTPJSONProfileMCPV0)
	if code != "" {
		return code
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "mcp_request_must_be_object"
	}
	switch trimmed[0] {
	case '[':
		return "mcp_batch_unsupported"
	case '{':
	default:
		return "mcp_request_must_be_object"
	}
	if err := json.Unmarshal(raw, request); err != nil {
		return "mcp_json_invalid"
	}
	return ""
}

func decodeServerPublicHTTPJSONRawV0(
	w http.ResponseWriter,
	r *http.Request,
	profile serverPublicHTTPJSONProfileV0,
) (json.RawMessage, string) {
	if !serverPublicHTTPContentTypeAllowsJSONV0(r.Header.Get("Content-Type")) {
		return nil, serverPublicErrContentTypeInvalidV0
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, serverPublicHTTPJSONMaxBytesV0(profile)))
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, serverPublicErrBodyTooLargeV0
		}
		return nil, "mcp_json_invalid"
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, serverPublicErrBodyTrailingDataV0
	}
	return raw, ""
}

func mcpJSONRPCContentTypeAllowsJSONV0(value string) bool {
	return serverPublicHTTPContentTypeAllowsJSONV0(value)
}

func mcpJSONRPCAcceptAllowsJSONV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	for _, part := range strings.Split(value, ",") {
		mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(part))
		if err != nil {
			mediaType = strings.TrimSpace(strings.Split(part, ";")[0])
		}
		if strings.TrimSpace(params["q"]) == "0" || strings.TrimSpace(params["q"]) == "0.0" {
			continue
		}
		mediaType = strings.ToLower(mediaType)
		if mediaType == "*/*" ||
			mediaType == "application/*" ||
			mediaType == "application/json" ||
			strings.HasSuffix(mediaType, "+json") {
			return true
		}
	}
	return false
}

func serverPublicHTTPJSONMaxBytesV0(profile serverPublicHTTPJSONProfileV0) int64 {
	switch profile {
	case serverPublicHTTPJSONProfileMCPV0:
		return serverPublicHTTPJSONMCPMaxBytesV0
	default:
		return serverPublicHTTPJSONControlMaxBytesV0
	}
}

func serverPublicHTTPContentTypeAllowsJSONV0(value string) bool {
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
