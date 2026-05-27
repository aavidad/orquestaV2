package orquestacli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const cliRESTResponseMaxBytesV0 int64 = 1 << 20

func readCLIRESTResponseBodyV0(resp *http.Response) ([]byte, string) {
	return readCLIRESTResponseBodyForCommandV0(resp, "")
}

func readCLIRESTResponseBodyForCommandV0(resp *http.Response, command string) ([]byte, string) {
	if resp == nil || resp.Body == nil {
		return nil, "response_missing"
	}
	maxBytes := cliRESTResponseMaxBytesForCommandV0(command)
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "body_no_legible"
	}
	if int64(len(body)) > maxBytes {
		return nil, "response_body_too_large"
	}
	if !cliRESTContentTypeIsJSONV0(resp.Header.Get("Content-Type")) {
		return nil, "response_content_type"
	}
	return body, ""
}

func decodeCLIRESTJSONBodyV0(resp *http.Response, target any) string {
	return decodeCLIRESTJSONBodyForCommandV0(resp, "", target)
}

func decodeCLIRESTJSONBodyForCommandV0(resp *http.Response, command string, target any) string {
	body, detail := readCLIRESTResponseBodyForCommandV0(resp, command)
	if detail != "" {
		return detail
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return "json_invalido"
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return "response_trailing_data"
	}
	return ""
}

func cliRESTNo2xxDetailForCommandV0(resp *http.Response, command string) string {
	_, detail := readCLIRESTResponseBodyForCommandV0(resp, command)
	if detail == "response_body_too_large" {
		return "status_no_2xx_response_body_too_large"
	}
	return "status_no_2xx"
}

func cliRESTResponseMaxBytesForCommandV0(command string) int64 {
	switch strings.TrimSpace(command) {
	case CliDefaultCommandServerStatusV0:
		return 256 << 10
	case CliDefaultCommandSolicitarAppV0,
		BootstrapAppSpecCliDefaultCommandV0,
		FunctionContractCliDefaultListCommandV0,
		FunctionContractCliDefaultViewCommandV0,
		CliDefaultCommandOperationalV0,
		CliDefaultCommandGovernanceV0,
		CliDefaultCommandAutoprogPrepareV0,
		CliDefaultCommandAutoprogQueueV0,
		CliDefaultCommandAutoprogStatusV0,
		CliDefaultCommandAutoprogRunV0,
		CliDefaultCommandAutoprogSuperviseV0,
		CliDefaultCommandAutoprogRunControlV0:
		return cliRESTResponseMaxBytesV0
	default:
		return cliRESTResponseMaxBytesV0
	}
}

func cliRESTContentTypeIsJSONV0(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return mediaType == "application/json" || mediaType == "text/plain"
}
