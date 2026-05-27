package orquestagovernance

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const governancePublicHTTPJSONMaxBytesV0 int64 = 256 << 10

func decodeGovernancePublicHTTPJSONV0(w http.ResponseWriter, r *http.Request, dst any) string {
	if !governancePublicHTTPContentTypeAllowsJSONV0(r.Header.Get("Content-Type")) {
		return "request_content_type_invalido"
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, governancePublicHTTPJSONMaxBytesV0))
	decoder.DisallowUnknownFields()
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

func governancePublicHTTPContentTypeAllowsJSONV0(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return true
	}
	mediaType := strings.TrimSpace(strings.Split(value, ";")[0])
	return mediaType == "application/json" || strings.HasSuffix(mediaType, "+json")
}

func governancePublicCorrelationIDFromHeaderV0(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
}

func firstNonEmptyGovernancePublicV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
