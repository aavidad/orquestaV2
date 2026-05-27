package orquestamcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const mcpHTTPResponseMaxBytesV0 int64 = 1 << 20

func decodeMCPHTTPJSONResponseV0(resp *http.Response, target any) error {
	body, err := readMCPHTTPResponseBodyV0(resp)
	if err != nil {
		return err
	}
	if !mcpHTTPContentTypeIsJSONV0(resp.Header.Get("Content-Type")) {
		return fmt.Errorf("mcp_response_content_type")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("mcp_response_invalid_json")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("mcp_response_trailing_data")
	}
	return nil
}

func readMCPHTTPResponseBodyV0(resp *http.Response) ([]byte, error) {
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("mcp_response_missing")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, mcpHTTPResponseMaxBytesV0+1))
	if err != nil {
		return nil, fmt.Errorf("mcp_response_read_error")
	}
	if int64(len(body)) > mcpHTTPResponseMaxBytesV0 {
		return nil, fmt.Errorf("mcp_response_too_large")
	}
	return body, nil
}

func discardMCPHTTPResponseBodyV0(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, mcpHTTPResponseMaxBytesV0))
}

func mcpHTTPContentTypeIsJSONV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return mediaType == "application/json" || mediaType == "text/plain"
}
