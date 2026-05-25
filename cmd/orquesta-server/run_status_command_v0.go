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
	_, _ = stdout.Write(body)
	return 0
}

func postRunStatusBodyV0(addr string, input orquestamcp.MCPDirectorStatsToolInputV0) ([]byte, error) {
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(input); err != nil {
		return nil, fmt.Errorf("run_status_request_invalid")
	}
	endpoint := strings.TrimRight(strings.TrimSpace(addr), "/")
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "http://" + endpoint
	}
	request, err := http.NewRequest(http.MethodPost, endpoint+orquestamcp.MCPDirectorStatsHTTPPathV0, &payload)
	if err != nil {
		return nil, fmt.Errorf("run_status_request_invalid")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Correlation-ID", strings.TrimSpace(input.RunRef))
	client := http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return readCommandHTTPResponseBodyV0(response, "run_status")
}
