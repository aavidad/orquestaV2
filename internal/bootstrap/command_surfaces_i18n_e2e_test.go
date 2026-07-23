package bootstrap

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/i18n"
	mcpinterface "orquesta/internal/interfaces/mcp"
)

func TestRealHTTPMCPCLIAndI18NParityEndToEnd(t *testing.T) {
	spanish := exerciseRealI18NSurfaces(t, "es")
	english := exerciseRealI18NSurfaces(t, "en")
	for _, evidence := range []realI18NSurfaceEvidence{spanish, english} {
		if !reflect.DeepEqual(evidence.successHTTP, evidence.successMCP) ||
			!reflect.DeepEqual(evidence.successHTTP, evidence.successCLI) {
			t.Fatalf("%s success HTTP=%+v MCP=%+v CLI=%+v",
				evidence.locale, evidence.successHTTP, evidence.successMCP, evidence.successCLI)
		}
		if !reflect.DeepEqual(evidence.failureHTTP, evidence.failureMCP) ||
			!reflect.DeepEqual(evidence.failureHTTP, evidence.failureCLI) {
			t.Fatalf("%s failure HTTP=%+v MCP=%+v CLI=%+v",
				evidence.locale, evidence.failureHTTP, evidence.failureMCP, evidence.failureCLI)
		}
		failure := evidence.failureHTTP.Failure
		if failure == nil || failure.Code != commandcore.CodeNotFound ||
			failure.MessageKey != "error.not_found" ||
			evidence.failureHTTP.CommandID != "orquesta.goals.get" ||
			evidence.failureHTTP.CommandVersion != "1" ||
			evidence.failureHTTP.RequestRef != "request:i18n-failure" {
			t.Fatalf("%s unstable failure envelope=%+v", evidence.locale, evidence.failureHTTP)
		}
	}
	if !reflect.DeepEqual(spanish.successHTTP, english.successHTTP) ||
		!reflect.DeepEqual(spanish.failureHTTP, english.failureHTTP) {
		t.Fatalf("locale changed machine envelopes: es=%+v/%+v en=%+v/%+v",
			spanish.successHTTP, spanish.failureHTTP, english.successHTTP, english.failureHTTP)
	}
	if spanish.instructions == english.instructions ||
		spanish.description == english.description ||
		spanish.commandHelp == english.commandHelp {
		t.Fatalf("human metadata did not vary: es=%+v en=%+v", spanish, english)
	}
	catalog, err := i18n.LoadBundled()
	if err != nil {
		t.Fatal(err)
	}
	for _, evidence := range []realI18NSurfaceEvidence{spanish, english} {
		wantInstructions, err := catalog.Text(evidence.locale, "server.instructions")
		if err != nil {
			t.Fatal(err)
		}
		wantDescription, err := catalog.Text(evidence.locale, "command.system.status.description")
		if err != nil {
			t.Fatal(err)
		}
		wantHelp, err := catalog.Text(evidence.locale, commandUsageKey)
		if err != nil {
			t.Fatal(err)
		}
		if evidence.instructions != wantInstructions || evidence.description != wantDescription ||
			evidence.commandHelp != wantHelp {
			t.Fatalf("%s human metadata=%+v want=%q/%q/%q",
				evidence.locale, evidence, wantInstructions, wantDescription, wantHelp)
		}
	}
}

type realI18NSurfaceEvidence struct {
	locale       string
	instructions string
	description  string
	commandHelp  string
	successHTTP  commandcore.Result
	successMCP   commandcore.Result
	successCLI   commandcore.Result
	failureHTTP  commandcore.Result
	failureMCP   commandcore.Result
	failureCLI   commandcore.Result
}

func exerciseRealI18NSurfaces(t *testing.T, locale string) realI18NSurfaceEvidence {
	t.Helper()
	root := t.TempDir()
	runtime, err := Build(context.Background(), Options{
		ConfigPath:   writeI18NTestConfig(t, root, locale),
		Version:      "i18n-surface-" + locale,
		AgentFactory: countingFactory(new(atomic.Int64)),
	})
	if err != nil {
		t.Fatalf("%s build: %v", locale, err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("%s start: %v", locale, err)
	}
	defer shutdownRuntime(t, runtime)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "i18n-surface-" + locale, Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdkmcp.StreamableClientTransport{
		Endpoint: runtime.MCPURL(), HTTPClient: authorizedHTTPClient(t, root+"/secrets/local-owner.token"),
	}, nil)
	if err != nil {
		t.Fatalf("%s MCP connect: %v", locale, err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("%s list tools: %v", locale, err)
	}
	var description string
	for _, tool := range tools.Tools {
		if tool.Name == "orquesta.system.status" {
			description = tool.Description
			break
		}
	}
	if description == "" {
		t.Fatalf("%s system status tool missing", locale)
	}

	const successRef = "request:i18n-success"
	successHTTP := postRuntimeCommand(t, runtime, root, "orquesta.system.status",
		commandEnvelope(t, successRef, "project:default", "", map[string]any{}))
	successMCPCall := callMCPTool(t, ctx, session, "orquesta.system.status", map[string]any{
		"version": "1", "request_ref": successRef, "project_ref": "project:default",
		"payload": map[string]any{},
	})
	var successMCP mcpinterface.CommandToolOutput
	decodeMCPOutput(t, successMCPCall, &successMCP)
	successCLI := decodeSDKResult(t, invokeRealCommandCLIRequest(
		t, runtime, root, locale, successRef, "{}", "system", "status",
	))

	const failureRef = "request:i18n-failure"
	failurePayload := map[string]any{"goal_ref": "goal:missing"}
	failureHTTP := postRuntimeCommand(t, runtime, root, "orquesta.goals.get",
		commandEnvelope(t, failureRef, "project:default", "", failurePayload))
	failureMCPCall := callMCPTool(t, ctx, session, "orquesta.goals.get", map[string]any{
		"version": "1", "request_ref": failureRef, "project_ref": "project:default",
		"payload": failurePayload,
	})
	var failureMCP mcpinterface.CommandToolOutput
	decodeMCPOutput(t, failureMCPCall, &failureMCP)
	failureCLI := decodeSDKResult(t, invokeRealCommandCLIRequest(
		t, runtime, root, locale, failureRef, `{"goal_ref":"goal:missing"}`, "goals", "get",
	))

	return realI18NSurfaceEvidence{
		locale: locale, instructions: session.InitializeResult().Instructions,
		description: description, commandHelp: invokeRealCommandCLIHelp(t, locale),
		successHTTP: successHTTP, successMCP: successMCP.Result, successCLI: successCLI,
		failureHTTP: failureHTTP, failureMCP: failureMCP.Result, failureCLI: failureCLI,
	}
}

func writeI18NTestConfig(t *testing.T, root, locale string) string {
	t.Helper()
	path := writeTestConfig(t, root)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(file, "\n[api]\nlocale = %q\n", locale); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
