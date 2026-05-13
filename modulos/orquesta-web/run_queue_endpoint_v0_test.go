package orquestaweb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestRunQueueWebEndpointV0GETConsultaRanking(t *testing.T) {
	client := &fakeRunQueueClientV0{
		result: WebRunQueueViewModelV0{
			SchemaVersion: "web_run_queue_panel.v0",
			Estado:        WebRunQueueEstadoOKV0,
			Action:        WebRunQueueActionRankV0,
			Ranked: []WebRunQueueCandidateV0{{
				Rank:          1,
				RunRef:        "run-ref-web-queue-001",
				AppRef:        "app-ref-web-queue-001",
				PriorityScore: 90,
				StatsHref:     "/director-stats?include_agent_progress=true&run_ref=run-ref-web-queue-001",
			}},
		},
	}
	endpoint := NewRunQueueWebEndpointV0(client)
	req := httptest.NewRequest(http.MethodGet, "/run-queue?action=rank&queue_ref=global&limit=3", nil)
	rec := httptest.NewRecorder()

	endpoint.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page WebRunQueuePageV0
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if client.query.Action != WebRunQueueActionRankV0 ||
		client.query.QueueRef != "global" ||
		client.query.Limit != 3 ||
		page.Refresh.Href == "" ||
		len(page.ViewModel.Ranked) != 1 ||
		page.ViewModel.Ranked[0].StatsHref != "/director-stats?include_agent_progress=true&run_ref=run-ref-web-queue-001" {
		t.Fatalf("query=%+v page=%+v", client.query, page)
	}
}

func TestRunQueueWebEndpointV0POSTSetPriority(t *testing.T) {
	client := &fakeRunQueueClientV0{
		result: WebRunQueueViewModelV0{
			SchemaVersion: "web_run_queue_panel.v0",
			Estado:        WebRunQueueEstadoOKV0,
			Action:        WebRunQueueActionSetV0,
			Updated: &WebRunQueueCandidateV0{
				RunRef:        "run-ref-web-queue-002",
				AppRef:        "app-ref-web-queue-002",
				PriorityScore: 120,
			},
		},
	}
	values := url.Values{}
	values.Set("action", WebRunQueueActionSetV0)
	values.Set("run_ref", "run-ref-web-queue-002")
	values.Set("app_ref", "app-ref-web-queue-002")
	values.Set("priority_score", "120")
	req := httptest.NewRequest(http.MethodPost, "/run-queue", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	NewRunQueueWebEndpointV0(client).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if client.query.Action != WebRunQueueActionSetV0 ||
		client.query.RunRef != "run-ref-web-queue-002" ||
		client.query.PriorityScore != 120 {
		t.Fatalf("query=%+v", client.query)
	}
}

func TestWebRunQueueViewModelV0IncluyeHrefStatsPorRun(t *testing.T) {
	vm := NewWebRunQueueViewModelV0("es", orquestamcp.MCPRunQueuePriorityToolResultV0{
		Estado: WebRunQueueEstadoOKV0,
		Action: WebRunQueueActionRankV0,
		Ranked: []orquestamcp.MCPRunQueueRankedCandidateCompactV0{{
			Rank:          1,
			RunRef:        "run-ref-web-queue-003",
			AppRef:        "app-ref-web-queue-003",
			PriorityScore: 90,
		}},
	})

	if len(vm.Ranked) != 1 ||
		vm.Ranked[0].StatsHref != "/director-stats?include_agent_progress=true&run_ref=run-ref-web-queue-003" {
		t.Fatalf("vm=%+v", vm)
	}
}

type fakeRunQueueClientV0 struct {
	query  WebRunQueueQueryV0
	result WebRunQueueViewModelV0
}

func (client *fakeRunQueueClientV0) ConsultarRunQueue(
	_ context.Context,
	query WebRunQueueQueryV0,
) (WebRunQueueViewModelV0, error) {
	client.query = query
	return client.result, nil
}
