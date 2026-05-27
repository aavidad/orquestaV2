package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"orquesta/modulos/orquesta-app-gateway/inprocesshttp"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	operator "orquesta/modulos/orquesta-operator-mcp"
)

const mcpRealSmokeConfirmEnvV0 = "ORQUESTA_MCP_REAL_SMOKE_CONFIRM"

type mcpRealSmokeSummaryV0 struct {
	SchemaVersion     string `json:"schema_version"`
	Status            string `json:"status"`
	ResourceRead      bool   `json:"resource_read"`
	SelfImprovement   bool   `json:"self_improvement"`
	OperatorErrorCode string `json:"operator_error_code"`
}

type mcpJSONRPCRawResponseV0 struct {
	JSONRPC string             `json:"jsonrpc"`
	ID      json.RawMessage    `json:"id,omitempty"`
	Result  json.RawMessage    `json:"result,omitempty"`
	Error   *mcpJSONRPCErrorV0 `json:"error,omitempty"`
}

func mcpRealSmokeCommandV0(stdout io.Writer, stderr io.Writer) int {
	if os.Getenv(mcpRealSmokeConfirmEnvV0) != "1" {
		_, _ = fmt.Fprintf(stderr, "%s=1 requerido para smoke MCP real opt-in\n", mcpRealSmokeConfirmEnvV0)
		return 2
	}
	handler, err := newMCPRealHTTPHandlerV0(orquestamcp.MCPTransportBindingsV0{})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp smoke: %v\n", err)
		return 1
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return mcpRealSmokeInProcessCommandV0(stdout, stderr, handler)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 2 * time.Second}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			_, _ = fmt.Fprintf(stderr, "mcp smoke serve: %v\n", err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	summary, err := runMCPRealSmokeV0(ctx, "http://"+listener.Addr().String()+mcpRealHTTPPathV0)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp smoke: %v\n", err)
		return 1
	}
	if err := writeCommandPublicOutputV0(
		stdout,
		"mcp-real-smoke",
		summary.Status,
		commandPublicFreshnessLiveV0,
		"local_smoke_summary_only",
		summary,
	); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "mcp-real-smoke", "stdout", "json_encode", err)
	}
	return 0
}

func runMCPRealSmokeV0(ctx context.Context, endpoint string) (mcpRealSmokeSummaryV0, error) {
	return runMCPRealSmokeWithClientV0(ctx, commandHTTPClientWithRedirectPolicyV0(5*time.Second, endpoint), endpoint)
}

func runMCPRealSmokeWithHandlerV0(
	ctx context.Context,
	handler http.Handler,
) (mcpRealSmokeSummaryV0, error) {
	return runMCPRealSmokeWithClientV0(
		ctx,
		http.Client{
			Timeout:       5 * time.Second,
			Transport:     inprocesshttp.TransportV0{Handler: handler},
			CheckRedirect: commandHTTPRedirectPolicyV0("http://orquesta-mcp.local"),
		},
		"http://orquesta-mcp.local"+mcpRealHTTPPathV0,
	)
}

func runMCPRealSmokeWithClientV0(
	ctx context.Context,
	client http.Client,
	endpoint string,
) (mcpRealSmokeSummaryV0, error) {
	if err := mcpSmokeReadOperatorResourceV0(ctx, client, endpoint); err != nil {
		return mcpRealSmokeSummaryV0{}, err
	}
	if err := mcpSmokeSelfImprovementV0(ctx, client, endpoint); err != nil {
		return mcpRealSmokeSummaryV0{}, err
	}
	code, err := mcpSmokeOperatorErrorV0(ctx, client, endpoint)
	if err != nil {
		return mcpRealSmokeSummaryV0{}, err
	}
	return mcpRealSmokeSummaryV0{
		SchemaVersion:     "mcp_real_transport_smoke.v0",
		Status:            "completed",
		ResourceRead:      true,
		SelfImprovement:   true,
		OperatorErrorCode: code,
	}, nil
}

func mcpRealSmokeInProcessCommandV0(stdout io.Writer, stderr io.Writer, handler http.Handler) int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	summary, err := runMCPRealSmokeWithHandlerV0(ctx, handler)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "mcp smoke: %v\n", err)
		return 1
	}
	if err := writeCommandPublicOutputV0(
		stdout,
		"mcp-real-smoke",
		summary.Status,
		commandPublicFreshnessSnapshotV0,
		"inprocess_smoke_summary_only",
		summary,
	); err != nil {
		return reportCommandStdioWriteFailureV0(stderr, "mcp-real-smoke", "stdout", "json_encode", err)
	}
	return 0
}

func mcpSmokeReadOperatorResourceV0(ctx context.Context, client http.Client, endpoint string) error {
	var result mcpResourceReadResultV0
	if err := callMCPJSONRPCV0(ctx, client, endpoint, "resources/read", mcpResourceReadParamsV0{
		URI: orquestamcp.MCPOperatorOperationsResourceURIV0,
	}, &result); err != nil {
		return err
	}
	if len(result.Contents) != 1 || !strings.Contains(result.Contents[0].Text, "orquesta.operator") {
		return fmt.Errorf("mcp resource read inesperado")
	}
	return nil
}

func mcpSmokeSelfImprovementV0(ctx context.Context, client http.Client, endpoint string) error {
	var result mcpToolCallResultV0
	arguments, err := mcpMarshalMCPRealSmokeV0(orquestamcp.MCPAutoprogrammingSelfImprovementToolInputV0{
		RequestID: "request-ref-mcp-real-smoke-self-improvement-001",
		Proposal:  mcpRealSmokeProposalV0(),
	})
	if err != nil {
		return err
	}
	if err := callMCPJSONRPCV0(ctx, client, endpoint, "tools/call", mcpToolCallParamsV0{
		Name:      orquestamcp.MCPAutoprogrammingSelfImprovementToolNameV0,
		Arguments: arguments,
	}, &result); err != nil {
		return err
	}
	var output orquestamcp.MCPAutoprogrammingSelfImprovementToolResultV0
	if err := decodeMCPToolTextV0(result, &output); err != nil {
		return err
	}
	if output.Estado != orquestamcp.MCPAutoprogrammingSelfImprovementEstadoOKV0 ||
		!output.Accepted ||
		output.PrepareRun == nil ||
		output.PreparedRun != nil {
		return fmt.Errorf("mcp self_improvement inesperado")
	}
	return nil
}

func mcpSmokeOperatorErrorV0(ctx context.Context, client http.Client, endpoint string) (string, error) {
	var result mcpToolCallResultV0
	arguments, err := mcpMarshalMCPRealSmokeV0(operator.OperatorStatusQueryV0{
		RequestRef:         "request-ref-mcp-real-smoke-status-001",
		SubjectRef:         "run-ref-mcp-real-smoke-001",
		StatusConnectorRef: "status-connector-ref-mcp-real-smoke-001",
		IncludeSections:    []string{"summary"},
	})
	if err != nil {
		return "", err
	}
	if err := callMCPJSONRPCV0(ctx, client, endpoint, "tools/call", mcpToolCallParamsV0{
		Name:      operator.OperatorMCPStatusToolNameV0,
		Arguments: arguments,
	}, &result); err != nil {
		return "", err
	}
	var output orquestamcp.MCPOperatorToolResultV0
	if err := decodeMCPToolTextV0(result, &output); err != nil {
		return "", err
	}
	if output.Estado != orquestamcp.MCPOperatorToolEstadoErrorV0 ||
		output.ErrorCode != operator.ErrOperatorMCPPortUnavailableV0 ||
		!result.IsError {
		return "", fmt.Errorf("mcp operator error inesperado")
	}
	return output.ErrorCode, nil
}

func callMCPJSONRPCV0(
	ctx context.Context,
	client http.Client,
	endpoint string,
	method string,
	params any,
	output any,
) error {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": mcpJSONRPCVersionV0,
		"id":      "request-ref-mcp-real-smoke-jsonrpc",
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBody, err := readCommandHTTPResponseBodyV0(response, "mcp")
	if err != nil {
		return err
	}
	var rpc mcpJSONRPCRawResponseV0
	if err := json.Unmarshal(responseBody, &rpc); err != nil {
		return err
	}
	if rpc.Error != nil {
		return fmt.Errorf("mcp rpc error %s", rpc.Error.Message)
	}
	return json.Unmarshal(rpc.Result, output)
}

func decodeMCPToolTextV0(result mcpToolCallResultV0, output any) error {
	if len(result.Content) != 1 || strings.TrimSpace(result.Content[0].Text) == "" {
		return fmt.Errorf("mcp tool content vacio")
	}
	return json.Unmarshal([]byte(result.Content[0].Text), output)
}

func mcpMarshalMCPRealSmokeV0(value any) (json.RawMessage, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("internal_invariant:mcp_smoke_arguments_json")
	}
	return payload, nil
}

func mcpRealSmokeProposalV0() orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0 {
	return orquestaautoprogramming.AutoprogrammingSelfImprovementProposalV0{
		ProjectRef:         "project-ref-orquesta",
		WorktreeRef:        "worktree-ref-orquesta-server-idle-self-improvement",
		WorktreeIsolated:   true,
		BranchRef:          "branch-ref-orquesta-server-idle-self-improvement",
		ObservedBy:         "director",
		SourceRunRef:       "run-ref-mcp-real-smoke-source-001",
		SourceTaskRef:      "task-ref-self-improvement-dfc25bcca8c4",
		FailureKind:        "mcp_transport_smoke",
		FailureSummary:     "Smoke temporal valida transporte MCP real opt-in.",
		SuggestedArea:      "mcp",
		SuggestedWriteSet:  []string{"cmd/orquesta-server/mcp_real_transport_v0.go"},
		RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-mcp ./cmd/orquesta-server"},
		AcceptanceCriteria: []string{"smoke MCP real opt-in ejecuta lectura y propuesta no destructiva"},
		CompactRules:       []string{"no leer stores ni runtime desde transporte"},
		EvidenceRefs:       []string{"evidence-ref-mcp-real-smoke-001"},
	}
}
