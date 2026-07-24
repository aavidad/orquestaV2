package mcpinterface

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	commandcore "orquesta/internal/commands"
	"orquesta/internal/interfaces/httpapi"
)

func TestV22GoalsGetExpandedProjectionHasHTTPMCPParity(t *testing.T) {
	data := json.RawMessage(`{
		"goal":{"goal_ref":"goal:public","project_ref":"project:v20","state":"running","revision":4,"plan_generation":2,"app_spec_generation":1,"spec_hash":"sha256:spec","work_item_count":1},
		"execution_count":1,
		"artifact_count":0,
		"work_items":[{"work_item_ref":"work-item:public","state":"running","revision":2,"parent_work_item_ref":"","dependency_refs":[],"handoff_required":false,"execution_ref":"execution:public","paused":false,"cancel_requested":false,"interrupt_code":""}],
		"executions":[{"execution_ref":"execution:public","work_item_ref":"work-item:public","attempt_no":1,"max_attempts":3,"replaces_execution_ref":"","plan_generation":2,"app_spec_generation":1,"state":"running","purpose":"work","failure_code":"","recipient_mailbox_retired":false}],
		"attestations":[],
		"reviews":[],
		"controls":[],
		"integration_receipts":[],
		"mailbox_receipts":[{"message_ref":"message:public","state":"acknowledged","source_principal_ref":"principal:child","source_execution_ref":"execution:child","recipient_principal_ref":"principal:parent","recipient_execution_ref":"execution:public","admission_ref":"admission:public","consumption_ref":"consumption:public","acknowledgement_ref":"acknowledgement:public","outcome":"acknowledged"}]
	}`)
	executor := &v20Executor{result: commandcore.Result{Data: data, AuditRef: "audit:v22-public"}}
	provider := v20Identity{principal: v20Principal(t)}

	httpHandler, err := httpapi.New(httpapi.Config{Dispatcher: executor, Identity: provider})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(httpHandler)
	t.Cleanup(httpServer.Close)
	envelope := []byte(`{"version":"1","request_ref":"request:v22-http","project_ref":"project:v20","payload":{"goal_ref":"goal:public"}}`)
	response, err := http.Post(httpServer.URL+"/api/v1/commands/orquesta.goals.get", "application/json", bytes.NewReader(envelope))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var httpResult commandcore.Result
	if err := json.NewDecoder(response.Body).Decode(&httpResult); err != nil {
		t.Fatal(err)
	}

	mcpServer := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "orquesta-v22", Version: "test"}, nil)
	if err := registerCommandToolsForTest(t, mcpServer, executor, provider, "es"); err != nil {
		t.Fatal(err)
	}
	handler := sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server { return mcpServer }, &sdkmcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	mcpHTTP := httptest.NewServer(handler)
	t.Cleanup(mcpHTTP.Close)
	session := connectOfficialClient(t, mcpHTTP.URL)
	called := callTool(t, session, "orquesta.goals.get", map[string]any{
		"version": "1", "request_ref": "request:v22-mcp", "project_ref": "project:v20",
		"payload": map[string]any{"goal_ref": "goal:public"},
	})
	var output CommandToolOutput
	decodeStructured(t, called, &output)

	var httpData, mcpData any
	if err := json.Unmarshal(httpResult.Data, &httpData); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(output.Result.Data, &mcpData); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(httpData, mcpData) {
		t.Fatalf("http data=%v mcp data=%v", httpData, mcpData)
	}
	httpResult.RequestRef = ""
	output.Result.RequestRef = ""
	httpResult.Data = nil
	output.Result.Data = nil
	if !reflect.DeepEqual(httpResult, output.Result) {
		t.Fatalf("http=%+v mcp=%+v", httpResult, output.Result)
	}
	var projected struct {
		Goal struct {
			AppSpecGeneration uint64 `json:"app_spec_generation"`
		} `json:"goal"`
		WorkItems  []json.RawMessage `json:"work_items"`
		Executions []struct {
			AttemptNo   uint64 `json:"attempt_no"`
			FailureCode string `json:"failure_code"`
		} `json:"executions"`
		MailboxReceipts []struct {
			Admission       string `json:"admission_ref"`
			Consumption     string `json:"consumption_ref"`
			Acknowledgement string `json:"acknowledgement_ref"`
		} `json:"mailbox_receipts"`
	}
	if err := json.Unmarshal(data, &projected); err != nil {
		t.Fatal(err)
	}
	if projected.Goal.AppSpecGeneration != 1 || len(projected.WorkItems) != 1 ||
		len(projected.Executions) != 1 || projected.Executions[0].AttemptNo != 1 ||
		len(projected.MailboxReceipts) != 1 || projected.MailboxReceipts[0].Admission == "" ||
		projected.MailboxReceipts[0].Consumption == "" || projected.MailboxReceipts[0].Acknowledgement == "" {
		t.Fatalf("projection=%+v", projected)
	}
}
