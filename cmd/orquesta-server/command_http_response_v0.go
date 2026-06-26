package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const commandHTTPResponseMaxBytesV0 int64 = 1 << 20

func readCommandHTTPResponseBodyV0(response *http.Response, context string) ([]byte, error) {
	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("%s_response_missing", context)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, commandHTTPResponseMaxBytesV0+1))
	if err != nil {
		return nil, fmt.Errorf("%s_response_read_error", context)
	}
	if int64(len(body)) > commandHTTPResponseMaxBytesV0 {
		return nil, fmt.Errorf("%s_response_too_large", context)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, commandHTTPStatusErrorV0(context, response.StatusCode, response.Header.Get("Content-Type"), body)
	}
	if !commandHTTPContentTypeIsJSONV0(response.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("%s_response_content_type", context)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("%s_response_invalid_json", context)
	}
	if decoder.Decode(&payload) != io.EOF {
		return nil, fmt.Errorf("%s_response_trailing_data", context)
	}
	return body, nil
}

func commandHTTPContentTypeIsJSONV0(value string) bool {
	if strings.TrimSpace(value) == "" {
		return true
	}
	mediaType := strings.TrimSpace(strings.ToLower(strings.Split(value, ";")[0]))
	return mediaType == "application/json" || mediaType == "text/plain"
}

func commandHTTPStatusErrorV0(context string, status int, contentType string, body []byte) error {
	code := fmt.Sprintf("%s_http_%d", strings.TrimSpace(context), status)
	if detail := commandHTTPPublicErrorDetailV0(contentType, body); detail != "" {
		return fmt.Errorf("%s:%s", code, detail)
	}
	return fmt.Errorf("%s", code)
}

func commandHTTPPublicErrorDetailV0(contentType string, body []byte) string {
	if len(body) == 0 ||
		!commandHTTPContentTypeIsJSONV0(contentType) ||
		!json.Valid(body) {
		return ""
	}
	var decoded struct {
		ErroresPublicos []struct {
			Code string `json:"code"`
		} `json:"errores_publicos"`
		Issues []struct {
			Code string `json:"code"`
		} `json:"issues"`
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return ""
	}
	codes := make([]string, 0, len(decoded.ErroresPublicos)+len(decoded.Issues)+len(decoded.Diagnostics)+1)
	for _, issue := range decoded.ErroresPublicos {
		codes = append(codes, compactCommandHTTPErrorCodeV0(issue.Code))
	}
	for _, issue := range decoded.Issues {
		codes = append(codes, compactCommandHTTPErrorCodeV0(issue.Code))
	}
	for _, diagnostic := range decoded.Diagnostics {
		codes = append(codes, compactCommandHTTPErrorCodeV0(diagnostic.Code))
	}
	codes = append(codes, compactCommandHTTPErrorCodeV0(decoded.Code))
	return strings.Join(compactCommandHTTPErrorCodesV0(codes), ",")
}

func compactCommandHTTPErrorCodeV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastSep := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_',
			r == '-',
			r == '.',
			r == ':':
			b.WriteRune(r)
			lastSep = false
		default:
			if !lastSep {
				b.WriteByte('-')
				lastSep = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func compactCommandHTTPErrorCodesV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
