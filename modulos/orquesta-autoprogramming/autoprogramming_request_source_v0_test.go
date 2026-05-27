package orquestaautoprogramming

import "testing"

func TestValidateAutoprogrammingRequestSourceV0ExigeRefsDeOrigenEnTarea(t *testing.T) {
	source := validAutoprogrammingRequestSourceV0(func(source *AutoprogrammingRequestSourceV0) {
		source.AutoprogrammingRequest.Tasks[0].ContextRefs = nil
	})

	result := ValidateAutoprogrammingRequestSourceV0(source)

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestSourceIssueV0(t, result, "source_context_ref_missing")
	if !result.RequestValidation.Accepted {
		t.Fatalf("request_validation=%+v", result.RequestValidation.Issues)
	}
}

func TestValidateAutoprogrammingRequestSourceV0RechazaSobreSinIdentidadPublica(t *testing.T) {
	result := ValidateAutoprogrammingRequestSourceV0(validAutoprogrammingRequestSourceV0(
		func(source *AutoprogrammingRequestSourceV0) {
			source.SchemaVersion = "autoprogramming_request_source.v9"
			source.SourceRef = ""
			source.SourceSurface = ""
			source.Transport = ""
			source.RequestID = ""
			source.CorrelationID = ""
			source.RequestedBy = ""
			source.PriorityScore = 0
		},
	))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	for _, code := range []string{
		"schema_version_invalid",
		"source_ref_missing",
		"source_surface_missing",
		"transport_missing",
		"request_id_missing",
		"correlation_id_missing",
		"requested_by_missing",
		"priority_score_invalid",
	} {
		assertAutoprogrammingRequestSourceIssueV0(t, result, code)
	}
}

func TestAutoprogrammingRequestSourceRefsV0IncluyeSourceRefDurable(t *testing.T) {
	source := validAutoprogrammingRequestSourceV0(nil)
	refs := AutoprogrammingRequestSourceRefsV0(source)

	if !autoprogrammingRequestSourceHasContextRefV0(refs, "source_ref:"+source.SourceRef) {
		t.Fatalf("source_ref no preservado: %v", refs)
	}
}

func validAutoprogrammingRequestSourceV0(
	mutate func(*AutoprogrammingRequestSourceV0),
) AutoprogrammingRequestSourceV0 {
	source := AutoprogrammingRequestSourceV0{
		SchemaVersion:          AutoprogrammingRequestSourceSchemaVersionV0,
		SourceRef:              "source-ref-autoprogramming-web-001",
		SourceSurface:          "web",
		Transport:              "http",
		RequestID:              "request-ref-source-web-001",
		CorrelationID:          "corr-source-web-001",
		RequestedBy:            "operator-ref-web",
		PriorityScore:          10,
		AutoprogrammingRequest: validAutoprogrammingRequestV0(nil),
	}
	for i := range source.AutoprogrammingRequest.Tasks {
		source.AutoprogrammingRequest.Tasks[i].ContextRefs = append(
			source.AutoprogrammingRequest.Tasks[i].ContextRefs,
			AutoprogrammingRequestSourceRefsV0(source)...,
		)
	}
	if mutate != nil {
		mutate(&source)
	}
	return source
}

func assertAutoprogrammingRequestSourceIssueV0(
	t *testing.T,
	result AutoprogrammingRequestSourceValidationResultV0,
	code string,
) {
	t.Helper()
	for _, issue := range result.Issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q no encontrado: %+v", code, result.Issues)
}
