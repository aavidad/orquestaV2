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
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s_http_%d", context, response.StatusCode)
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
	mediaType := strings.TrimSpace(strings.ToLower(strings.Split(value, ";")[0]))
	return mediaType == "application/json"
}
