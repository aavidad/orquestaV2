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

func TestCodexStackV0ExternalWorkRunAceptaContratoAmplioV0(t *testing.T) {
	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	body := bytes.NewBuffer(nil)
	err := json.NewEncoder(body).Encode(orquestamcp.MCPExternalWorkRunToolInputV0{
		RequestID:     "req-external-work-wide-001",
		CorrelationID: "corr-external-work-wide-001",
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "opes-job-job-ref-wide-001",
			AppRef:     "opes",
			UserIntent: "Resolver trabajo externo amplio de OPES.",
			AcceptanceCriteria: []string{
				"devolver artifact_type=topic_expansion_package",
				"payload_json valido y trazable",
				"sin placeholders",
				"sin leer internals de OPES",
				"entrega en fichero unico bajo allowed_write_set",
				"paquete apto para tema_grande con capitulos y bloques trazables",
				"conservar base para tema_mediano sin perder autores normativa ni procedimientos",
				"incluir resumen/memoria de repaso derivado del tema desarrollado",
				"incluir esquema de examen y plan de visuales cuando aporten valor",
			},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     "job-ref-wide-001",
				WorkKind:   "expand_topic_from_summary",
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "expected_artifact",
					Value: "topic_expansion_package",
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
	run, err := stack.Stores.RunStore.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if len(run.DirectorQuestions) != 1 {
		t.Fatalf("director_questions=%v", run.DirectorQuestions)
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
