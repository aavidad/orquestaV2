package orquestacli

import (
	"errors"
	"strings"
	"time"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
)

const (
	CliOutputFormatTextV0 = "text"
	CliOutputFormatJSONV0 = "json"
	CliOutputFormatTSVV0  = "tsv"

	CliInputSourceStdinV0        = "stdin"
	CliInputSourceFileExplicitV0 = "file_explicit"
	CliInputSourceInlineV0       = "inline"
	CliInputSourceArgV0          = "arg"

	CliErrOpcionInvalidaV0                 = cliPublicErrOptionInvalidV0
	CliErrInputTooLargeV0                  = cliPublicErrInputTooLargeV0
	CliErrConfiguracionInvalidaV0          = cliPublicErrConfigInvalidV0
	CliErrContratoNoConfiguradoV0          = cliPublicErrContractMissingV0
	CliErrRespuestaInvalidaV0              = cliPublicErrResponseInvalidV0
	CliErrErrorTransporteV0                = cliPublicErrTransportV0
	CliErrSalidaNoSerializableV0           = cliPublicErrNotSerializableV0
	CliContractSolicitarNuevaAppV0         = "SolicitarNuevaApp"
	CliContractVersionSolicitarAppV0       = "v0"
	CliContractOperationalStatusV0         = "OperationalStatusQuery"
	CliContractVersionOperationalV0        = "v0"
	CliContractGovernanceCatalogV0         = "GovernanceCatalog"
	CliContractVersionGovernanceV0         = "v0"
	CliContractServerStatusV0              = "ServerStatus"
	CliContractVersionServerStatusV0       = "v0"
	CliContractAutoprogrammingV0           = "Autoprogramming"
	CliContractVersionAutoprogrammingV0    = "v0"
	CliTransportRESTV0                     = "rest"
	CliDefaultCommandSolicitarAppV0        = "app spec solicitar"
	CliDefaultCommandOperationalV0         = "doctor contratos"
	CliDefaultCommandGovernanceV0          = "gobernanza catalogo listar"
	CliDefaultCommandGovernanceViewV0      = "gobernanza catalogo ver"
	CliDefaultCommandServerStatusV0        = "servidor estado"
	CliDefaultCommandAutoprogPrepareV0     = "autoprogramacion preparar"
	CliDefaultCommandAutoprogStatusV0      = "autoprogramacion estado ver"
	CliDefaultCommandAutoprogObserveGoalV0 = "autoprogramacion goal observar"
	CliDefaultCommandAutoprogSuperviseV0   = "autoprogramacion supervisar"
	CliDefaultCommandAutoprogQueueV0       = "autoprogramacion cola listar"
	CliDefaultCommandAutoprogRunV0         = "autoprogramacion run ver"
	CliDefaultCommandAutoprogRunControlV0  = "autoprogramacion run controlar"
	CliDefaultRequestIDPrefixV0            = "req-cli-"
	CliDefaultErrorMessageNamespaceV0      = "orquesta_cli.errores."
)

var cliRequestIDGeneratorV0 = orquestaruntime.NewSystemRefGeneratorV0()

type CliInvocationContextV0 struct {
	Command        string        `json:"command"`
	RequestID      string        `json:"request_id"`
	CorrelationID  string        `json:"correlation_id"`
	IdempotencyKey string        `json:"idempotency_key,omitempty"`
	ServerURL      string        `json:"server_url"`
	Timeout        time.Duration `json:"timeout"`
	OutputFormat   string        `json:"output_format"`
	Locale         string        `json:"locale,omitempty"`
	DryRun         bool          `json:"dry_run,omitempty"`
	InputSource    string        `json:"input_source,omitempty"`
}

type CliOutputEnvelopeV0 struct {
	OK            bool                 `json:"ok"`
	RequestID     string               `json:"request_id"`
	CorrelationID string               `json:"correlation_id"`
	Contract      string               `json:"contract"`
	Version       string               `json:"version"`
	Data          any                  `json:"data,omitempty"`
	Errores       []CliPublicErrorV0   `json:"errores"`
	Warnings      []CliPublicWarningV0 `json:"warnings"`
	Meta          CliOutputMetaV0      `json:"meta"`
}

type CliPublicErrorV0 struct {
	Codigo      string `json:"codigo"`
	Campo       string `json:"campo,omitempty"`
	MensajeI18N string `json:"mensaje_i18n"`
	Detalle     string `json:"detalle,omitempty"`
}

type CliPublicWarningV0 struct {
	Codigo      string `json:"codigo"`
	MensajeI18N string `json:"mensaje_i18n"`
	Detalle     string `json:"detalle,omitempty"`
}

type CliOutputMetaV0 struct {
	Transporte string `json:"transporte"`
	DuracionMS int64  `json:"duracion_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	Retryable  bool   `json:"retryable"`
}

type CliClientErrorV0 struct {
	Code       string
	Field      string
	Detail     string
	StatusCode int
	Retryable  bool
}

func (err CliClientErrorV0) Error() string {
	return err.Code
}

func NormalizeCliInvocationContextV0(inv CliInvocationContextV0) CliInvocationContextV0 {
	inv.Command = strings.TrimSpace(inv.Command)
	inv.RequestID = strings.TrimSpace(inv.RequestID)
	inv.CorrelationID = strings.TrimSpace(inv.CorrelationID)
	inv.IdempotencyKey = strings.TrimSpace(inv.IdempotencyKey)
	inv.ServerURL = strings.TrimSpace(inv.ServerURL)
	inv.OutputFormat = strings.TrimSpace(inv.OutputFormat)
	inv.Locale = strings.TrimSpace(inv.Locale)
	inv.InputSource = strings.TrimSpace(inv.InputSource)

	if inv.RequestID == "" {
		inv.RequestID = newCliRequestIDV0()
	}
	if inv.CorrelationID == "" {
		inv.CorrelationID = inv.RequestID
	}
	if inv.OutputFormat == "" {
		inv.OutputFormat = CliOutputFormatJSONV0
	}
	if inv.InputSource == "" {
		inv.InputSource = CliInputSourceArgV0
	}
	return inv
}

func ValidateCliInvocationContextV0(inv CliInvocationContextV0) []CliPublicErrorV0 {
	var errs []CliPublicErrorV0
	switch inv.OutputFormat {
	case CliOutputFormatTextV0, CliOutputFormatJSONV0, CliOutputFormatTSVV0:
	default:
		errs = append(errs, NewCliPublicErrorV0(CliErrOpcionInvalidaV0, "output_format", "output_format_debe_ser_text_json_o_tsv"))
	}
	switch inv.InputSource {
	case "", CliInputSourceStdinV0, CliInputSourceFileExplicitV0, CliInputSourceInlineV0, CliInputSourceArgV0:
	default:
		errs = append(errs, NewCliPublicErrorV0(CliErrOpcionInvalidaV0, "input_source", "input_source_debe_ser_stdin_file_explicit_inline_o_arg"))
	}
	return errs
}

func NewCliOutputOKEnvelopeV0(inv CliInvocationContextV0, contract string, version string, data any, meta CliOutputMetaV0) CliOutputEnvelopeV0 {
	return CliOutputEnvelopeV0{
		OK:            true,
		RequestID:     strings.TrimSpace(inv.RequestID),
		CorrelationID: strings.TrimSpace(inv.CorrelationID),
		Contract:      strings.TrimSpace(contract),
		Version:       strings.TrimSpace(version),
		Data:          data,
		Errores:       []CliPublicErrorV0{},
		Warnings:      []CliPublicWarningV0{},
		Meta:          meta,
	}
}

func NewCliOutputErrorEnvelopeV0(inv CliInvocationContextV0, contract string, version string, errs []CliPublicErrorV0, meta CliOutputMetaV0) CliOutputEnvelopeV0 {
	if errs == nil {
		errs = []CliPublicErrorV0{}
	}
	return CliOutputEnvelopeV0{
		OK:            false,
		RequestID:     strings.TrimSpace(inv.RequestID),
		CorrelationID: strings.TrimSpace(inv.CorrelationID),
		Contract:      strings.TrimSpace(contract),
		Version:       strings.TrimSpace(version),
		Errores:       errs,
		Warnings:      []CliPublicWarningV0{},
		Meta:          meta,
	}
}

func NewCliPublicErrorV0(code, field, detail string) CliPublicErrorV0 {
	code = strings.TrimSpace(code)
	return CliPublicErrorV0{
		Codigo:      code,
		Campo:       strings.TrimSpace(field),
		MensajeI18N: CliDefaultErrorMessageNamespaceV0 + code,
		Detalle:     strings.TrimSpace(detail),
	}
}

func NewCliClientErrorV0(code, field, detail string, statusCode int, retryable bool) CliClientErrorV0 {
	return CliClientErrorV0{
		Code:       strings.TrimSpace(code),
		Field:      strings.TrimSpace(field),
		Detail:     strings.TrimSpace(detail),
		StatusCode: statusCode,
		Retryable:  retryable,
	}
}

func IsCliClientErrorCodeV0(err error, code string) bool {
	var cliErr CliClientErrorV0
	if errors.As(err, &cliErr) {
		return cliErr.Code == code
	}
	return false
}

func cliMetaV0(start time.Time, statusCode int, retryable bool) CliOutputMetaV0 {
	duration := time.Since(start).Milliseconds()
	if duration < 0 {
		duration = 0
	}
	return CliOutputMetaV0{
		Transporte: CliTransportRESTV0,
		DuracionMS: duration,
		StatusCode: statusCode,
		Retryable:  retryable,
	}
}

func newCliRequestIDV0() string {
	ref, _ := cliRequestIDGeneratorV0.NextRefV0(CliDefaultRequestIDPrefixV0, "client-mutation")
	return ref
}
