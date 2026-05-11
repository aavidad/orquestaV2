package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
)

func TestCodexStackV0ExponeControlYPrioridadPorAPISinAcoplarCore(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	stack := mustBuildCodexStackForTestV0(t, runtime)
	director := postDirectorAPIV0(t, stack)

	control := postRunControlStackV0(t, stack, orquestamcp.MCPRunControlToolInputV0{
		Action:      "pause",
		RunRef:      director.RunRef,
		RequestedBy: "director",
		Reason:      "prioridad humana menor",
	})
	if control.Status != string(orquestaruncontrol.RunControlStatusPausedV0) {
		t.Fatalf("control=%+v", control)
	}
	state, err := stack.Stores.RunControl.ReadRunControlStateV0(
		context.Background(),
		orquestaruncontrol.RunControlReadRequestV0{RunRef: director.RunRef},
	)
	if err != nil {
		t.Fatalf("ReadRunControlStateV0: %v", err)
	}
	if state.Status != orquestaruncontrol.RunControlStatusPausedV0 {
		t.Fatalf("state=%+v", state)
	}

	priority := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:        "set_priority",
		RunRef:        director.RunRef,
		AppRef:        "agenda-api-web",
		PriorityScore: 90,
		RequestedBy:   "director",
	})
	if priority.Updated == nil || priority.Updated.PriorityScore != 90 {
		t.Fatalf("priority=%+v", priority)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action: "rank",
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != director.RunRef ||
		ranking.Ranked[0].PriorityScore != 90 {
		t.Fatalf("ranking=%+v", ranking)
	}
}

func postRunControlStackV0(
	t *testing.T,
	stack StackV0,
	input orquestamcp.MCPRunControlToolInputV0,
) orquestamcp.MCPRunControlToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode control: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/control", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("control status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunControlToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode control: %v", err)
	}
	return result
}

func postRunQueuePriorityStackV0(
	t *testing.T,
	stack StackV0,
	input orquestamcp.MCPRunQueuePriorityToolInputV0,
) orquestamcp.MCPRunQueuePriorityToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(input); err != nil {
		t.Fatalf("encode priority: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/runs/queue/priority", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("priority status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPRunQueuePriorityToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode priority: %v", err)
	}
	return result
}
