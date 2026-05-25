package orquestamcp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const mcpPublicHTTPJSONMaxBytesV0 int64 = 1 << 20
const mcpPublicExecutorErrorMaxLenV0 = 360

var mcpPublicExecutorErrorRedactionsV0 = []struct {
	pattern *regexp.Regexp
	replace string
}{
	{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._\-+/=]+`), `${1}<redacted>`},
	{regexp.MustCompile(`(?i)((?:access|refresh|id)_?token=)[^&\s]+`), `${1}<redacted>`},
	{regexp.MustCompile(`(?i)((?:api[_-]?key|token|password|secret)=)[^&\s]+`), `${1}<redacted>`},
	{regexp.MustCompile(`\bsk-[A-Za-z0-9_\-]{8,}`), `<redacted>`},
	{regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9_]{8,}`), `<redacted>`},
	{regexp.MustCompile(`/home/[^\s"']+`), `<path>`},
	{regexp.MustCompile(`/root/[^\s"']+`), `<path>`},
	{regexp.MustCompile(`/tmp/[^\s"']+`), `<path>`},
}

func decodeMCPPublicHTTPJSONV0(w http.ResponseWriter, r *http.Request, dst any) string {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, mcpPublicHTTPJSONMaxBytesV0))
	if err := decoder.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return "request_body_too_large"
		}
		return "request_body_invalido"
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		return "request_body_trailing_data"
	}
	return ""
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
	cause := publicMCPExecutorErrorSanitizeV0(err.Error())
	if cause == "" || cause == base {
		return base
	}
	return base + ": " + cause
}

func publicMCPExecutorErrorSanitizeV0(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, redaction := range mcpPublicExecutorErrorRedactionsV0 {
		value = redaction.pattern.ReplaceAllString(value, redaction.replace)
	}
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) > mcpPublicExecutorErrorMaxLenV0 {
		value = string(runes[:mcpPublicExecutorErrorMaxLenV0]) + "..."
	}
	return strings.TrimSpace(value)
}
