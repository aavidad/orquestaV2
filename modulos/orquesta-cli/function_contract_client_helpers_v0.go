package orquestacli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
)

const (
	functionContractErrIncompletoV0           = "function_contract_incompleto"
	functionContractErrConsultaNoDisponibleV0 = "consulta_function_contract_no_disponible"
	functionContractErrOperacionNoPromovidaV0 = "operacion_no_promovida"
)

func prepareFunctionContractInvocationV0(inv CliInvocationContextV0, client *FunctionContractCliClientV0, command string) CliInvocationContextV0 {
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = command
	}
	timeout := effectiveTimeoutV0(inv.Timeout, clientTimeoutFunctionContractV0(client))
	inv.Timeout = timeout
	if inv.ServerURL == "" && client != nil {
		inv.ServerURL = client.BaseURL
	}
	return inv
}

func cleanListarFunctionContractsRequestV0(req ListarFunctionContractsRequestV0) ListarFunctionContractsRequestV0 {
	req.RequestID = strings.TrimSpace(req.RequestID)
	req.CorrelationID = strings.TrimSpace(req.CorrelationID)
	req.Filtros.Modulo = strings.TrimSpace(req.Filtros.Modulo)
	req.Filtros.Estado = strings.TrimSpace(req.Filtros.Estado)
	req.Filtros.ArchivoObjetivo = strings.TrimSpace(req.Filtros.ArchivoObjetivo)
	req.Filtros.SimboloObjetivo = strings.TrimSpace(req.Filtros.SimboloObjetivo)
	req.Page.Cursor = strings.TrimSpace(req.Page.Cursor)
	return req
}

func clientErrorFunctionContractEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	var cliErr CliClientErrorV0
	if errors.As(err, &cliErr) {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, cliErr.Code, cliErr.Field, cliErr.Detail, cliErr.StatusCode, cliErr.Retryable, start)
	}
	return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrConfiguracionInvalidaV0, "server_url", "server_url_invalida", 0, false, start)
}

func transportFunctionContractEnvelopeV0(inv CliInvocationContextV0, err error, start time.Time) CliOutputEnvelopeV0 {
	detail := "request_failed"
	if isTimeoutErrorV0(err) {
		detail = "timeout"
	}
	return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "transporte", detail, 0, true, start)
}

func cliSingleFunctionContractErrorEnvelopeV0(inv CliInvocationContextV0, code, field, detail string, statusCode int, retryable bool, start time.Time) CliOutputEnvelopeV0 {
	return NewCliOutputErrorEnvelopeV0(
		inv,
		FunctionContractCliContractV0,
		FunctionContractCliContractVersionV0,
		[]CliPublicErrorV0{NewCliPublicErrorV0(code, field, detail)},
		cliMetaV0(start, statusCode, retryable),
	)
}

func listEndpointFunctionContractV0(client *FunctionContractCliClientV0) string {
	if client != nil && strings.TrimSpace(client.ListEndpoint) != "" {
		return client.ListEndpoint
	}
	return FunctionContractCliListEndpointV0
}

func viewEndpointFunctionContractV0(client *FunctionContractCliClientV0) string {
	if client != nil && strings.TrimSpace(client.ViewEndpoint) != "" {
		return client.ViewEndpoint
	}
	return FunctionContractCliViewEndpointV0
}

func httpClientFunctionContractV0(client *FunctionContractCliClientV0, timeout time.Duration) *http.Client {
	if client == nil {
		return httpClientFromConfigV0(cliRESTClientConfigV0{}, timeout)
	}
	return httpClientFromConfigV0(cliRESTClientConfigV0{HTTPClient: client.HTTPClient}, timeout)
}

func clientTimeoutFunctionContractV0(client *FunctionContractCliClientV0) time.Duration {
	if client == nil {
		return clientTimeoutFromConfigV0(cliRESTClientConfigV0{})
	}
	return clientTimeoutFromConfigV0(cliRESTClientConfigV0{Timeout: client.Timeout})
}

func decodeFunctionContractIssuesV0(body io.Reader) ([]CliPublicErrorV0, bool) {
	var out functionContractHTTPErrorV0
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return nil, false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, false
	}
	errs := make([]CliPublicErrorV0, 0, len(out.Errores))
	for _, issue := range out.Errores {
		errs = append(errs, functionContractIssueToCliErrorV0(issue))
	}
	return errs, true
}

func functionContractIssueToCliErrorV0(issue orquestacore.FunctionContractErrorV0) CliPublicErrorV0 {
	code := strings.TrimSpace(issue.Code)
	if code == "" || !isKnownFunctionContractIssueCliV0(code) {
		code = functionContractErrConsultaNoDisponibleV0
	}
	return CliPublicErrorV0{
		Codigo:      code,
		Campo:       strings.TrimSpace(issue.Field),
		MensajeI18N: "orquesta_core.function_contract.errores." + code,
		Detalle:     "",
	}
}

func isKnownFunctionContractIssueCliV0(code string) bool {
	switch strings.TrimSpace(code) {
	case functionContractErrIncompletoV0,
		"archivo_objetivo_fuera_de_write_set",
		"write_set_vacio",
		"cambio_fuera_de_write_set",
		"dependencia_prohibida",
		"tests_obligatorios_ausentes",
		"formato_entrega_no_soportado",
		"evidencia_insuficiente",
		"estado_no_ejecutable",
		functionContractErrConsultaNoDisponibleV0,
		"filtro_no_soportado",
		"cursor_invalido",
		functionContractErrOperacionNoPromovidaV0,
		"function_contract_no_encontrado",
		FunctionContractCliErrRegistrarBloqueadoV0:
		return true
	default:
		return false
	}
}

func decodeVerFunctionContractResultV0(raw []byte) (VerFunctionContractResultV0, error) {
	var result VerFunctionContractResultV0
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return VerFunctionContractResultV0{}, errors.New("json_invalido")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return VerFunctionContractResultV0{}, errors.New("json_invalido")
	}
	if !validFunctionContractV0(result.FunctionContract) {
		return VerFunctionContractResultV0{}, errors.New("function_contract_invalido")
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	return result, nil
}

func validFunctionContractSummaryV0(item FunctionContractResumenV0) bool {
	if strings.TrimSpace(item.FunctionContractRef) == "" {
		return false
	}
	if item.Version < 0 {
		return false
	}
	return true
}

func validFunctionContractV0(contract orquestacore.FunctionContractV0) bool {
	return strings.TrimSpace(contract.Titulo) != "" &&
		strings.TrimSpace(contract.Objetivo) != "" &&
		strings.TrimSpace(contract.ArchivoObjetivo) != "" &&
		strings.TrimSpace(contract.SimboloObjetivo) != "" &&
		len(contract.WriteSet) > 0 &&
		len(contract.TestsObligatorios) > 0 &&
		strings.TrimSpace(contract.FormatoEntrega) != "" &&
		len(contract.CriterioCierre) > 0 &&
		strings.TrimSpace(contract.Estado) != "" &&
		contract.Version > 0
}
