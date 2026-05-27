package orquestacli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	orquestacore "orquesta/modulos/orquesta-core"
)

const (
	FunctionContractCliListEndpointV0       = "/api/v0/core/function-contracts/list"
	FunctionContractCliViewEndpointV0       = "/api/v0/core/function-contracts/view"
	FunctionContractCliCorrelationHeaderV0  = CliCorrelationHeaderV0
	FunctionContractCliContractV0           = "FunctionContract"
	FunctionContractCliContractVersionV0    = "v0"
	FunctionContractCliDefaultListCommandV0 = "contratos funcion listar"
	FunctionContractCliDefaultViewCommandV0 = "contratos funcion ver"
	FunctionContractCliDefaultRegCommandV0  = "contratos funcion registrar"

	FunctionContractCliErrRegistrarBloqueadoV0 = "registrar_function_contract_bloqueado"
)

type FunctionContractCliClientV0 struct {
	BaseURL      string
	ListEndpoint string
	ViewEndpoint string
	Timeout      time.Duration
	HTTPClient   *http.Client
}

type ListarFunctionContractsRequestV0 struct {
	RequestID     string                        `json:"request_id"`
	CorrelationID string                        `json:"correlation_id"`
	Filtros       FunctionContractFiltrosV0     `json:"filtros"`
	Page          FunctionContractPageRequestV0 `json:"page"`
}

type FunctionContractFiltrosV0 struct {
	Modulo          string `json:"modulo,omitempty"`
	Estado          string `json:"estado,omitempty"`
	ArchivoObjetivo string `json:"archivo_objetivo,omitempty"`
	SimboloObjetivo string `json:"simbolo_objetivo,omitempty"`
}

type FunctionContractPageRequestV0 struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type ListarFunctionContractsResultV0 struct {
	Items      []FunctionContractResumenV0 `json:"items"`
	NextCursor string                      `json:"next_cursor,omitempty"`
	Warnings   []string                    `json:"warnings"`
}

type FunctionContractResumenV0 struct {
	FunctionContractRef string `json:"function_contract_ref"`
	Titulo              string `json:"titulo,omitempty"`
	Modulo              string `json:"modulo,omitempty"`
	ArchivoObjetivo     string `json:"archivo_objetivo,omitempty"`
	SimboloObjetivo     string `json:"simbolo_objetivo,omitempty"`
	Estado              string `json:"estado,omitempty"`
	Version             int    `json:"version,omitempty"`
}

type VerFunctionContractRequestV0 struct {
	RequestID           string `json:"request_id"`
	CorrelationID       string `json:"correlation_id"`
	FunctionContractRef string `json:"function_contract_ref"`
	Version             int    `json:"version,omitempty"`
}

type VerFunctionContractResultV0 struct {
	FunctionContract orquestacore.FunctionContractV0 `json:"function_contract"`
	Warnings         []string                        `json:"warnings"`
}

type functionContractHTTPErrorV0 struct {
	Errores []orquestacore.FunctionContractErrorV0 `json:"errores"`
}

func NewFunctionContractCliClientV0(serverURL string, timeout time.Duration) (*FunctionContractCliClientV0, error) {
	config, err := newCLIRESTClientConfigV0(serverURL, timeout, FunctionContractCliListEndpointV0)
	if err != nil {
		return nil, err
	}
	return &FunctionContractCliClientV0{
		BaseURL:      config.BaseURL,
		ListEndpoint: config.Endpoint,
		ViewEndpoint: FunctionContractCliViewEndpointV0,
		Timeout:      config.Timeout,
		HTTPClient:   newCLILoopbackHTTPClientV0(config.Timeout),
	}, nil
}

func (client *FunctionContractCliClientV0) ListarFunctionContracts(ctx context.Context, inv CliInvocationContextV0, req ListarFunctionContractsRequestV0) CliOutputEnvelopeV0 {
	start := time.Now()
	inv = prepareFunctionContractInvocationV0(inv, client, FunctionContractCliDefaultListCommandV0)
	if errs := ValidateCliInvocationContextV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, FunctionContractCliContractV0, FunctionContractCliContractVersionV0, errs, cliMetaV0(start, 0, false))
	}
	baseURL, err := normalizeServerURLV0(inv.ServerURL)
	if err != nil {
		return clientErrorFunctionContractEnvelopeV0(inv, err, start)
	}

	req.RequestID = inv.RequestID
	req.CorrelationID = inv.CorrelationID
	payload, err := json.Marshal(cleanListarFunctionContractsRequestV0(req))
	if err != nil {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "request_no_serializable", 0, false, start)
	}
	return client.doFunctionContractRequestV0(ctx, inv, baseURL, listEndpointFunctionContractV0(client), payload, start, decodeFunctionContractListResponseV0)
}

func (client *FunctionContractCliClientV0) VerFunctionContract(ctx context.Context, inv CliInvocationContextV0, req VerFunctionContractRequestV0) CliOutputEnvelopeV0 {
	start := time.Now()
	inv = prepareFunctionContractInvocationV0(inv, client, FunctionContractCliDefaultViewCommandV0)
	if errs := ValidateCliInvocationContextV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, FunctionContractCliContractV0, FunctionContractCliContractVersionV0, errs, cliMetaV0(start, 0, false))
	}
	baseURL, err := normalizeServerURLV0(inv.ServerURL)
	if err != nil {
		return clientErrorFunctionContractEnvelopeV0(inv, err, start)
	}

	req.RequestID = inv.RequestID
	req.CorrelationID = inv.CorrelationID
	req.FunctionContractRef = strings.TrimSpace(req.FunctionContractRef)
	payload, err := json.Marshal(req)
	if err != nil {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "request_no_serializable", 0, false, start)
	}
	return client.doFunctionContractRequestV0(ctx, inv, baseURL, viewEndpointFunctionContractV0(client), payload, start, decodeFunctionContractViewResponseV0)
}

func (client *FunctionContractCliClientV0) RegistrarFunctionContract(ctx context.Context, inv CliInvocationContextV0, contract orquestacore.FunctionContractV0) CliOutputEnvelopeV0 {
	_ = ctx
	_ = client
	_ = contract
	start := time.Now()
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = FunctionContractCliDefaultRegCommandV0
	}
	if errs := ValidateCliInvocationContextV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, FunctionContractCliContractV0, FunctionContractCliContractVersionV0, errs, cliMetaV0(start, 0, false))
	}
	return cliSingleFunctionContractErrorEnvelopeV0(inv, FunctionContractCliErrRegistrarBloqueadoV0, "registrar", "operacion_bloqueada_por_contrato", 0, false, start)
}

func (client *FunctionContractCliClientV0) doFunctionContractRequestV0(ctx context.Context, inv CliInvocationContextV0, baseURL string, endpoint string, payload []byte, start time.Time, decode func(*http.Response, CliInvocationContextV0, time.Time) CliOutputEnvelopeV0) CliOutputEnvelopeV0 {
	ctx, cancel := context.WithTimeout(ctx, inv.Timeout)
	defer cancel()

	httpReq, err := prepareCLIRESTRequestV0(ctx, baseURL, endpoint, inv.CorrelationID, payload)
	if err != nil {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "server_url", "request_http_invalida", 0, true, start)
	}
	resp, err := httpClientFunctionContractV0(client, inv.Timeout).Do(httpReq)
	if err != nil {
		return transportFunctionContractEnvelopeV0(inv, err, start)
	}
	defer resp.Body.Close()

	return decode(resp, inv, start)
}

func decodeFunctionContractListResponseV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	if errEnv, handled := decodeFunctionContractErrorStatusV0(resp, inv, start); handled {
		return errEnv
	}
	raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
	if detail != "" {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
	}
	result, err := decodeListarFunctionContractsResultV0(raw)
	if err != nil {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", err.Error(), resp.StatusCode, false, start)
	}
	return NewCliOutputOKEnvelopeV0(inv, FunctionContractCliContractV0, FunctionContractCliContractVersionV0, result, cliMetaV0(start, resp.StatusCode, false))
}

func decodeFunctionContractViewResponseV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	if errEnv, handled := decodeFunctionContractErrorStatusV0(resp, inv, start); handled {
		return errEnv
	}
	raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
	if detail != "" {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
	}
	result, err := decodeVerFunctionContractResultV0(raw)
	if err != nil {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", err.Error(), resp.StatusCode, false, start)
	}
	return NewCliOutputOKEnvelopeV0(inv, FunctionContractCliContractV0, FunctionContractCliContractVersionV0, result, cliMetaV0(start, resp.StatusCode, false))
}

func decodeFunctionContractErrorStatusV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) (CliOutputEnvelopeV0, bool) {
	if resp.StatusCode == http.StatusBadRequest {
		raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
		if detail != "" {
			return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start), true
		}
		errs, ok := decodeFunctionContractIssuesV0(bytes.NewReader(raw))
		if !ok || len(errs) == 0 {
			return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "errores", "respuesta_400_sin_errores_publicos", resp.StatusCode, false, start), true
		}
		return NewCliOutputErrorEnvelopeV0(inv, FunctionContractCliContractV0, FunctionContractCliContractVersionV0, errs, cliMetaV0(start, resp.StatusCode, false)), true
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return cliSingleFunctionContractErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "status_code", cliRESTNo2xxDetailForCommandV0(resp, inv.Command), resp.StatusCode, retryableStatusV0(resp.StatusCode), start), true
	}
	return CliOutputEnvelopeV0{}, false
}

func decodeListarFunctionContractsResultV0(raw []byte) (ListarFunctionContractsResultV0, error) {
	var result ListarFunctionContractsResultV0
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return ListarFunctionContractsResultV0{}, errors.New("json_invalido")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ListarFunctionContractsResultV0{}, errors.New("json_invalido")
	}
	if result.Items == nil {
		return ListarFunctionContractsResultV0{}, errors.New("items_requeridos")
	}
	for _, item := range result.Items {
		if !validFunctionContractSummaryV0(item) {
			return ListarFunctionContractsResultV0{}, errors.New("items_invalidos")
		}
	}
	if result.Warnings == nil {
		result.Warnings = []string{}
	}
	return result, nil
}
