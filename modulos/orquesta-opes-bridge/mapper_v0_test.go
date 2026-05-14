package orquestaopesbridge

import (
	"encoding/json"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaopesconnector "orquesta/modulos/orquesta-opes-connector"
)

func TestBuildExternalWorkRunRequestV0MapeaSummarizeTopic(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestWithContextV0(orquestaopesconnector.ExternalJobV0{
		ID:             "job-ref-summary-001",
		Type:           "summarize_topic",
		Status:         "pending",
		ExecutionMode:  "external",
		CorrelationID:  "corr-ref-summary-001",
		IdempotencyKey: "idem-summary-001",
		RequestedBy:    "opes",
		PayloadJSON: `{
			"program_id":"program-ref-001",
			"topic_id":"topic-ref-001",
			"official_order":90,
			"quality_criteria":["derivar solo del temario contrastado","no inventar"]
		}`,
	}, JobRunConfigV0{PriorityScore: 80}, JobContextV0{
		TopicBlocks: []orquestaopesconnector.TopicBlockV0{{
			ID:         "block-ref-001",
			StableID:   "stable-ref-001",
			Title:      "Bloque 1",
			Markdown:   "Texto del bloque.",
			SourceRefs: []string{"source-ref-001"},
		}},
	})

	if !ok {
		t.Fatalf("request no construida")
	}
	work := req.AppChangeRequest.ExternalWork
	if req.ProjectRef != "opes" ||
		req.AppChangeRequest.AppRef != "opes" ||
		req.AppChangeRequest.ChangeRef != "opes-job-job-ref-summary-001" ||
		req.AppChangeRequest.AllowedWriteSet[0] != "external/opes/summarize_topic/job-ref-summary-001" ||
		work == nil ||
		work.JobRef != "job-ref-summary-001" ||
		work.WorkKind != "summarize_topic" ||
		!fieldValueForTestV0(work.InputFields, "expected_artifact_type", "topic_summary") ||
		!fieldValueForTestV0(work.InputFields, "context_budget_profile", "large") ||
		!fieldValuesForTestV0(work.InputFields, "quality_criteria", []string{"derivar solo del temario contrastado", "no inventar"}) ||
		!fieldJSONForTestV0(work.InputFields, "topic_blocks") ||
		!containsStringForTestV0(work.WorkRefs, "opes-topic_id-topic-ref-001") {
		t.Fatalf("request=%+v work=%+v", req, work)
	}
}

func TestBuildExternalWorkRunRequestV0MapeaExpansionComoLarge(t *testing.T) {
	req, ok := BuildExternalWorkRunRequestV0(orquestaopesconnector.ExternalJobV0{
		ID:   "job-ref-expansion-001",
		Type: "expand_topic_from_summary",
		PayloadJSON: `{
			"topic_id":"topic-ref-001",
			"summary_payload_json":{"markdown":"Resumen"},
			"output_contract":["artifact_type=topic_expansion_package"]
		}`,
	}, JobRunConfigV0{})

	if !ok {
		t.Fatalf("request no construida")
	}
	fields := req.AppChangeRequest.ExternalWork.InputFields
	if req.AppChangeRequest.AllowedWriteSet[0] != "external/opes/expand_topic_from_summary/job-ref-expansion-001" ||
		!fieldValueForTestV0(fields, "expected_artifact_type", "topic_expansion_package") ||
		!fieldValueForTestV0(fields, "context_budget_profile", "large") ||
		!fieldValuesForTestV0(fields, "required_document_variants", []string{
			"tema_grande",
			"tema_mediano",
			"resumen",
			"esquema_repaso",
			"plan_visuales",
		}) ||
		!fieldJSONForTestV0(fields, "summary_payload_json") {
		t.Fatalf("request=%+v", req)
	}
	if !containsStringForTestV0(req.AppChangeRequest.AcceptanceCriteria, "incluir resumen/memoria de repaso derivado del tema desarrollado") {
		t.Fatalf("criteria=%+v", req.AppChangeRequest.AcceptanceCriteria)
	}
}

func fieldValueForTestV0(fields []orquestadomainwork.DomainWorkFieldV0, name string, value string) bool {
	for _, field := range fields {
		if field.Name == name && field.Value == value {
			return true
		}
	}
	return false
}

func fieldValuesForTestV0(fields []orquestadomainwork.DomainWorkFieldV0, name string, values []string) bool {
	for _, field := range fields {
		if field.Name != name || len(field.Values) != len(values) {
			continue
		}
		ok := true
		for index := range values {
			if field.Values[index] != values[index] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func fieldJSONForTestV0(fields []orquestadomainwork.DomainWorkFieldV0, name string) bool {
	for _, field := range fields {
		if field.Name == name && json.Valid(field.ValueJSON) {
			return true
		}
	}
	return false
}

func containsStringForTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
