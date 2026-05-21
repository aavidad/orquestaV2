package orquestaappchange

import "testing"

func TestPrepareAppChangeRequestV0NormalizaEnvelopePublico(t *testing.T) {
	request := PrepareAppChangeRequestV0(AppChangeRequestV0{
		ChangeRef:  "change-ref-001",
		RunRef:     " run-ref-001 ",
		UserIntent: " aplicar cambio ",
		AcceptanceCriteria: []string{
			" criterio ",
			"criterio",
			"",
		},
		RequiredTests: []string{
			" go test ./... ",
			"go test ./...",
			"",
		},
	})

	if request.SchemaVersion != AppChangeRequestSchemaV0 ||
		request.RequestID != "change-ref-001" ||
		request.CorrelationID != "change-ref-001" ||
		request.RunRef != "run-ref-001" ||
		len(request.AcceptanceCriteria) != 1 ||
		len(request.RequiredTests) != 1 ||
		request.RequiredTests[0] != "go test ./..." {
		t.Fatalf("request normalizada inesperada: %+v", request)
	}
}

func TestValidateAppChangeRequestV0ExponeValidacionSinPersistir(t *testing.T) {
	issues := ValidateAppChangeRequestV0(AppChangeRequestV0{
		RunRef:    "run-ref-001",
		ChangeRef: "change-ref-001",
	})

	if len(issues) != 1 || issues[0].Code != ErrAppChangeIntentRequiredV0 {
		t.Fatalf("issues=%+v", issues)
	}
}
