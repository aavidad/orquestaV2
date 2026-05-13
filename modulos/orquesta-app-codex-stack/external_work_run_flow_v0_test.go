package orquestaappcodexstack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestCodexStackV0ExternalWorkRunCreaRunSinDirectorInicial(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())

	result := postExternalWorkRunStackV0(t, stack)
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if string(run.CurrentPhase) != "programacion" ||
		len(run.DirectorQuestions) != 1 ||
		len(run.StartedAgents) != 0 {
		t.Fatalf("run=%+v", run)
	}

	ranking := postRunQueuePriorityStackV0(t, stack, orquestamcp.MCPRunQueuePriorityToolInputV0{
		Action:   "rank",
		QueueRef: DefaultRunQueueRefV0,
	})
	if len(ranking.Ranked) != 1 ||
		ranking.Ranked[0].RunRef != result.RunRef ||
		ranking.Ranked[0].AppRef != "opes" {
		t.Fatalf("ranking=%+v result=%+v", ranking, result)
	}
}

func postExternalWorkRunStackV0(
	t *testing.T,
	stack StackV0,
) orquestamcp.MCPExternalWorkRunToolResultV0 {
	t.Helper()
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:     "req-external-work-stack-001",
		CorrelationID: "corr-external-work-stack-001",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "opes-job-job-ref-001",
			AppRef:     "opes",
			UserIntent: "Resolver trabajo externo de OPES.",
			AcceptanceCriteria: []string{
				"markdown valido",
			},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     "job-ref-001",
				WorkKind:   "draft_content_block",
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "topic_id",
					Value: "topic-ref-001",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/external-work/run", body)
	req.Header.Set("Content-Type", "application/json")
	stack.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("external work run status=%d body=%s", rec.Code, rec.Body.String())
	}
	var result orquestamcp.MCPExternalWorkRunToolResultV0
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode external work run: %v", err)
	}
	if result.Estado != orquestamcp.MCPExternalWorkRunEstadoOKV0 {
		t.Fatalf("result=%+v", result)
	}
	return result
}
