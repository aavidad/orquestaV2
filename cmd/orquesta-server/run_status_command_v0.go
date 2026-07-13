package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func runStatusCommandV0(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("run-status", flag.ContinueOnError)
	flags.SetOutput(stderr)
	runRef := flags.String("run-ref", "", "ref opaca del run")
	progress := flags.Bool("progress", false, "incluir progreso observado de agentes")
	usage := flags.Bool("usage", false, "incluir uso agregado de agentes")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*runRef) == "" {
		_, _ = fmt.Fprintln(stderr, "run-status: run_ref requerido")
		return 2
	}
	config, err := serverConfigFromEnvV0()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "run-status: %v\n", err)
		return 1
	}
	defer releaseServerProjectConfigSnapshotV0(config)
	state, err := loadStateV0(config)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "run-status: %v\n", err)
		return 1
	}
	body, err := postRunStatusBodyV0(state.Addr, orquestamcp.MCPDirectorStatsToolInputV0{
		RunRef:               strings.TrimSpace(*runRef),
		IncludeProcessRefs:   true,
		IncludeAgentProgress: *progress,
		IncludeAgentUsage:    *usage,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "run-status: %v\n", err)
		return 1
	}
	var result orquestamcp.MCPDirectorStatsToolResultV0
	if err := json.Unmarshal(body, &result); err != nil {
		_, _ = fmt.Fprintf(stderr, "run-status: run_status_response_invalid_json\n")
		return 1
	}
	if err := writeCommandPublicOutputV0(
		stdout,
		"run-status",
		result.Estado,
		commandPublicFreshnessLiveV0,
		"use_director_stats_tool_for_canonical_run_state",
		commandPublicRunStatusPayloadV0(result),
	); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "run-status", "stdout", "json_encode", err)
	}
	return 0
}

func postRunStatusBodyV0(addr string, input orquestamcp.MCPDirectorStatsToolInputV0) ([]byte, error) {
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(input); err != nil {
		return nil, fmt.Errorf("run_status_request_invalid")
	}
	baseURL, err := commandRESTBaseURLFromAddrV0(addr)
	if err != nil {
		return nil, fmt.Errorf("run_status_request_invalid")
	}
	target, err := commandRESTEndpointURLV0(baseURL, orquestamcp.MCPDirectorStatsHTTPPathV0)
	if err != nil {
		return nil, fmt.Errorf("run_status_request_invalid")
	}
	request, err := http.NewRequest(http.MethodPost, target, &payload)
	if err != nil {
		return nil, fmt.Errorf("run_status_request_invalid")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", strings.TrimSpace(input.RunRef))
	client := commandHTTPClientWithRedirectPolicyV0(5*time.Second, baseURL)
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return readCommandHTTPResponseBodyV0(response, "run_status")
}
