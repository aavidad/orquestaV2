package orquestacli

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

type failingCliEntropyV0 struct{}

func (failingCliEntropyV0) Read([]byte) (int, error) {
	return 0, errors.New("entropy unavailable")
}

func TestNormalizeCliInvocationContextV0GeneraRequestYCorrelationID(t *testing.T) {
	inv := NormalizeCliInvocationContextV0(CliInvocationContextV0{})

	if !strings.HasPrefix(inv.RequestID, CliDefaultRequestIDPrefixV0) {
		t.Fatalf("request_id=%q", inv.RequestID)
	}
	if inv.CorrelationID != inv.RequestID {
		t.Fatalf("correlation_id=%q request_id=%q", inv.CorrelationID, inv.RequestID)
	}
	if inv.OutputFormat != CliOutputFormatJSONV0 || inv.InputSource != CliInputSourceArgV0 {
		t.Fatalf("defaults inesperados: output=%q input=%q", inv.OutputFormat, inv.InputSource)
	}
}

func TestNewCliRequestIDV0FallbackDegradadoObservableV0(t *testing.T) {
	previous := cliRequestIDGeneratorV0
	cliRequestIDGeneratorV0 = orquestaruntime.NewRefGeneratorV0(
		orquestaruntime.ClockFuncV0(func() time.Time {
			return time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
		}),
		failingCliEntropyV0{},
	)
	defer func() { cliRequestIDGeneratorV0 = previous }()

	ref := newCliRequestIDV0()
	if !strings.HasPrefix(ref, CliDefaultRequestIDPrefixV0+"client-mutation-degraded-") {
		t.Fatalf("request_id fallback no observable: %q", ref)
	}
}

func TestCliOutputEnvelopeV0JSONEstable(t *testing.T) {
	inv := CliInvocationContextV0{RequestID: "req-golden", CorrelationID: "corr-golden"}
	data := goldenSolicitarNuevaAppDataV0{
		AppSpec: goldenSchemaV0{SchemaVersion: "app_spec.v0"},
		Backlog: goldenSchemaV0{SchemaVersion: "backlog_inicial_propuesto.v0"},
	}

	cases := []struct {
		name string
		env  CliOutputEnvelopeV0
		want string
	}{
		{
			name: "exito",
			env: NewCliOutputOKEnvelopeV0(inv, CliContractSolicitarNuevaAppV0, CliContractVersionSolicitarAppV0, data, CliOutputMetaV0{
				Transporte: CliTransportRESTV0,
				DuracionMS: 7,
				StatusCode: 200,
				Retryable:  false,
			}),
			want: `{
  "ok": true,
  "request_id": "req-golden",
  "correlation_id": "corr-golden",
  "contract": "SolicitarNuevaApp",
  "version": "v0",
  "data": {
    "app_spec": {
      "schema_version": "app_spec.v0"
    },
    "backlog": {
      "schema_version": "backlog_inicial_propuesto.v0"
    }
  },
  "errores": [],
  "warnings": [],
  "meta": {
    "transporte": "rest",
    "duracion_ms": 7,
    "status_code": 200,
    "retryable": false
  }
}`,
		},
		{
			name: "validacion",
			env: NewCliOutputErrorEnvelopeV0(inv, CliContractSolicitarNuevaAppV0, CliContractVersionSolicitarAppV0, []CliPublicErrorV0{{
				Codigo:      "idioma_invalido",
				Campo:       "locale",
				MensajeI18N: "orquesta_factory.errores.idioma_invalido",
				Detalle:     "locale BCP 47 invalido",
			}}, CliOutputMetaV0{
				Transporte: CliTransportRESTV0,
				DuracionMS: 3,
				StatusCode: 400,
				Retryable:  false,
			}),
			want: `{
  "ok": false,
  "request_id": "req-golden",
  "correlation_id": "corr-golden",
  "contract": "SolicitarNuevaApp",
  "version": "v0",
  "errores": [
    {
      "codigo": "idioma_invalido",
      "campo": "locale",
      "mensaje_i18n": "orquesta_factory.errores.idioma_invalido",
      "detalle": "locale BCP 47 invalido"
    }
  ],
  "warnings": [],
  "meta": {
    "transporte": "rest",
    "duracion_ms": 3,
    "status_code": 400,
    "retryable": false
  }
}`,
		},
		{
			name: "transporte",
			env: NewCliOutputErrorEnvelopeV0(inv, CliContractSolicitarNuevaAppV0, CliContractVersionSolicitarAppV0, []CliPublicErrorV0{
				NewCliPublicErrorV0(CliErrErrorTransporteV0, "status_code", "status_no_2xx"),
			}, CliOutputMetaV0{
				Transporte: CliTransportRESTV0,
				DuracionMS: 5,
				StatusCode: 500,
				Retryable:  true,
			}),
			want: `{
  "ok": false,
  "request_id": "req-golden",
  "correlation_id": "corr-golden",
  "contract": "SolicitarNuevaApp",
  "version": "v0",
  "errores": [
    {
      "codigo": "error_transporte",
      "campo": "status_code",
      "mensaje_i18n": "orquesta_cli.errores.error_transporte",
      "detalle": "status_no_2xx"
    }
  ],
  "warnings": [],
  "meta": {
    "transporte": "rest",
    "duracion_ms": 5,
    "status_code": 500,
    "retryable": true
  }
}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.MarshalIndent(tt.env, "", "  ")
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("golden mismatch\nwant:\n%s\n\ngot:\n%s", tt.want, string(got))
			}
		})
	}
}

type goldenSolicitarNuevaAppDataV0 struct {
	AppSpec goldenSchemaV0 `json:"app_spec"`
	Backlog goldenSchemaV0 `json:"backlog"`
}

type goldenSchemaV0 struct {
	SchemaVersion string `json:"schema_version"`
}
