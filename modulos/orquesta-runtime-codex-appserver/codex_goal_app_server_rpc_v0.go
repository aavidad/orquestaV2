package orquestaruntimecodexappserver

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

func codexAppServerRPCPayloadV0(method string, params interface{}) (string, error) {
	var b strings.Builder
	for _, request := range []map[string]interface{}{
		codexAppServerInitializeRequestV0(),
		codexAppServerInitializedNotificationV0(),
		{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  method,
			"params":  params,
		},
	} {
		line, err := codexAppServerRPCMessageLineV0(request)
		if err != nil {
			return "", err
		}
		b.WriteString(line)
	}
	return b.String(), nil
}

func codexAppServerInitializeRequestV0() map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"clientInfo": map[string]string{
				"name":    "orquesta-server",
				"version": "0",
			},
			"capabilities": map[string]interface{}{
				"experimentalApi": true,
			},
		},
	}
}

func codexAppServerInitializedNotificationV0() map[string]interface{} {
	return map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
		"params":  map[string]interface{}{},
	}
}

func codexAppServerRPCMessageLineV0(request map[string]interface{}) (string, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

func decodeCodexAppServerRPCResponseReaderV0(stdout io.Reader, responseID int, out interface{}) error {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(
		make([]byte, 0, codexAppServerCommandProtocolInitialResponseLineBufferBytesV0),
		codexAppServerCommandProtocolDefaultMaxResponseLineBytesV0,
	)
	return codexAppServerScannerResponseErrorV0(
		decodeCodexAppServerRPCResponseScannerV0(scanner, responseID, out),
		codexAppServerCommandResponseTooLargeIssueCodeV0,
	)
}

func decodeCodexAppServerRPCResponseScannerV0(scanner *bufio.Scanner, responseID int, out interface{}) error {
	var lastErr error
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var response serverCodexAppServerRPCResponseV0
		if err := json.Unmarshal(line, &response); err != nil {
			lastErr = err
			continue
		}
		if response.ID != responseID {
			continue
		}
		if response.Error != nil {
			return codexAppServerRPCErrorV0(response.Error.Code, response.Error.Message)
		}
		if out == nil {
			return nil
		}
		if len(response.Result) == 0 {
			return errors.New("codex_app_server_empty_result")
		}
		if err := json.Unmarshal(response.Result, out); err != nil {
			return err
		}
		sanitizeCodexAppServerRPCDecodedOutV0(out)
		return nil
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("codex_app_server_response_missing")
}

type serverCodexAppServerRPCResponseV0 struct {
	ID     int             `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func codexAppServerRPCErrorV0(code int, message string) error {
	return codexAppServerCallErrorV0{
		Code: codexAppServerRPCIssueCodeV0(code),
		Err:  fmt.Errorf("codex_app_server_rpc_error: code=%d message=%s", code, strings.TrimSpace(message)),
	}
}

func codexAppServerRPCIssueCodeV0(code int) string {
	switch code {
	case -32700:
		return "codex_app_server_rpc_parse_error"
	case -32600:
		return "codex_app_server_rpc_invalid_request"
	case -32601:
		return "codex_app_server_rpc_method_not_found"
	case -32602:
		return "codex_app_server_rpc_invalid_params"
	case -32603:
		return "codex_app_server_rpc_internal_error"
	default:
		return "codex_app_server_rpc_error"
	}
}
