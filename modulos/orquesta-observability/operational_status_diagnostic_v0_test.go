package orquestaobservability

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateDiagnosticoCompactoV0Valido(t *testing.T) {
	diagnostic := validDiagnosticoCompactoV0()

	if err := ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		t.Fatalf("diagnostic should validate: %v", err)
	}
	if diagnostic.Privacy.ContainsSecret ||
		diagnostic.Privacy.ContainsTranscript ||
		diagnostic.Privacy.ContainsPrompt ||
		diagnostic.Privacy.ContainsCompletion ||
		diagnostic.Privacy.ContainsConnectionDetail {
		t.Fatalf("privacy flags should be false: %+v", diagnostic.Privacy)
	}
}

func TestDecodeDiagnosticoCompactoV0JSONEstricto(t *testing.T) {
	diagnostic := validDiagnosticoCompactoV0()
	data, err := json.Marshal(diagnostic)
	if err != nil {
		t.Fatalf("marshal diagnostic: %v", err)
	}
	data = []byte(strings.Replace(string(data), `"privacy":`, `"sql":"select * from audit_log","privacy":`, 1))

	_, err = DecodeDiagnosticoCompactoV0(data)
	assertOperationalStatusIssueV0(t, err, ErrOperationalStatusQueryInvalidaV0)
}

func TestValidateDiagnosticoCompactoV0PrivacyFalse(t *testing.T) {
	tests := []struct {
		name string
		edit func(*DiagnosticoCompactoV0)
		code string
	}{
		{
			name: "secret false obligatorio",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Privacy.ContainsSecret = true
			},
			code: ErrSecretoDetectadoV0,
		},
		{
			name: "transcript false obligatorio",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Privacy.ContainsTranscript = true
			},
			code: ErrTranscriptNoPermitidoV0,
		},
		{
			name: "prompt false obligatorio",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Privacy.ContainsPrompt = true
			},
			code: ErrOperationalStatusQueryInvalidaV0,
		},
		{
			name: "completion false obligatorio",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Privacy.ContainsCompletion = true
			},
			code: ErrOperationalStatusQueryInvalidaV0,
		},
		{
			name: "connection detail false obligatorio",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Privacy.ContainsConnectionDetail = true
			},
			code: ErrOperationalStatusQueryInvalidaV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostic := validDiagnosticoCompactoV0()
			test.edit(&diagnostic)
			assertOperationalStatusIssueV0(t, ValidateDiagnosticoCompactoV0(diagnostic), test.code)
		})
	}
}

func TestValidateDiagnosticoCompactoV0ContenidoProhibido(t *testing.T) {
	tests := []struct {
		name string
		edit func(*DiagnosticoCompactoV0)
		code string
	}{
		{
			name: "secreto en contador",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Contadores["access_token_count"] = 3
			},
			code: ErrSecretoDetectadoV0,
		},
		{
			name: "transcript en actividad",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.ActividadReciente[0].Summary = "transcript completo detectado"
			},
			code: ErrTranscriptNoPermitidoV0,
		},
		{
			name: "prompt en bloqueo",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Bloqueos[0].Summary = "prompt completo no permitido"
			},
			code: ErrOperationalStatusQueryInvalidaV0,
		},
		{
			name: "sql en referencia",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Referencias[0].TargetRef = "event_sql_20260504_000001"
			},
			code: ErrOperationalStatusQueryInvalidaV0,
		},
		{
			name: "completion en progreso",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Progreso.Summary = "completion completa no permitida"
			},
			code: ErrOperationalStatusQueryInvalidaV0,
		},
		{
			name: "dsn en warning",
			edit: func(diagnostic *DiagnosticoCompactoV0) {
				diagnostic.Warnings[0].Summary = "dsn oculto"
			},
			code: ErrOperationalStatusQueryInvalidaV0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostic := validDiagnosticoCompactoV0()
			test.edit(&diagnostic)
			assertOperationalStatusIssueV0(t, ValidateDiagnosticoCompactoV0(diagnostic), test.code)
		})
	}
}

func TestValidateDiagnosticoCompactoV0PermiteVocabularioOperativoOpaco(t *testing.T) {
	diagnostic := validDiagnosticoCompactoV0()
	diagnostic.Progreso.Summary = "runtime provider model como contexto opaco"
	diagnostic.Warnings[0].Summary = "provider model not_available sin valor concreto"
	diagnostic.Referencias[0].TargetRef = "token_policy_ref-runtime-redaction-v0"
	diagnostic.Privacy.RedactionLevel = DiagnosticoPrivacyRedactionMetadataOnlyV0

	if err := ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		t.Fatalf("diagnostic should allow opaque operational vocabulary: %v", err)
	}
}

func TestValidateDiagnosticoCompactoV0RedactionLevelVerificable(t *testing.T) {
	diagnostic := validDiagnosticoCompactoV0()
	diagnostic.Privacy.RedactionLevel = "raw"

	assertOperationalStatusIssueV0(t, ValidateDiagnosticoCompactoV0(diagnostic), ErrOperationalStatusQueryInvalidaV0)
}

func TestValidateDiagnosticoCompactoV0ListasCompactas(t *testing.T) {
	diagnostic := validDiagnosticoCompactoV0()
	diagnostic.ActividadReciente = make([]DiagnosticoActividadV0, maxOperationalListItemsV0+1)
	for index := range diagnostic.ActividadReciente {
		diagnostic.ActividadReciente[index] = DiagnosticoActividadV0{
			ActivityRef: "activity_20260504_" + leftPadOperationalStatusTestV0(index),
			OccurredAt:  "2026-05-04T10:00:00Z",
			Area:        "runtime",
			Summary:     "Cambio compacto observado",
			EventRef:    "event_20260504_" + leftPadOperationalStatusTestV0(index),
		}
	}

	assertOperationalStatusIssueV0(t, ValidateDiagnosticoCompactoV0(diagnostic), ErrConsultaDemasiadoAmpliaV0)
}
