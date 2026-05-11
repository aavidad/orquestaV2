package orquestacli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirector "orquesta/modulos/orquesta-director"
)

const (
	BootstrapAppSpecCliEndpointV0          = "/api/v0/director/bootstrap/appspec"
	BootstrapAppSpecCliCorrelationHeaderV0 = CliCorrelationHeaderV0
	BootstrapAppSpecCliContractV0          = "BootstrapProyectoDesdeAppSpec"
	BootstrapAppSpecCliContractVersionV0   = "v0"
	BootstrapAppSpecCliDefaultCommandV0    = "app spec bootstrap"
)

type BootstrapAppSpecCliClientV0 struct {
	BaseURL    string
	Endpoint   string
	Timeout    time.Duration
	HTTPClient *http.Client
}

type bootstrapAppSpecHTTPErrorV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
	Errores []struct {
		Code    string `json:"code"`
		Codigo  string `json:"codigo"`
		Field   string `json:"field"`
		Campo   string `json:"campo"`
		Message string `json:"message"`
	} `json:"errores,omitempty"`
}

func NewBootstrapAppSpecCliClientV0(serverURL string, timeout time.Duration) (*BootstrapAppSpecCliClientV0, error) {
	config, err := newCLIRESTClientConfigV0(serverURL, timeout, BootstrapAppSpecCliEndpointV0)
	if err != nil {
		return nil, err
	}
	return &BootstrapAppSpecCliClientV0{
		BaseURL:  config.BaseURL,
		Endpoint: config.Endpoint,
		Timeout:  config.Timeout,
		HTTPClient: &http.Client{
			Timeout: config.Timeout,
		},
	}, nil
}

func (client *BootstrapAppSpecCliClientV0) BootstrapProyectoDesdeAppSpec(ctx context.Context, inv CliInvocationContextV0, cmd orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0) CliOutputEnvelopeV0 {
	start := time.Now()
	inv = NormalizeCliInvocationContextV0(inv)
	if inv.Command == "" {
		inv.Command = BootstrapAppSpecCliDefaultCommandV0
	}
	timeout := effectiveTimeoutV0(inv.Timeout, clientTimeoutBootstrapAppSpecV0(client))
	inv.Timeout = timeout
	if inv.ServerURL == "" && client != nil {
		inv.ServerURL = client.BaseURL
	}

	if errs := validateBootstrapAppSpecInvocationV0(inv); len(errs) > 0 {
		return NewCliOutputErrorEnvelopeV0(inv, BootstrapAppSpecCliContractV0, BootstrapAppSpecCliContractVersionV0, errs, cliMetaV0(start, 0, false))
	}
	baseURL, err := normalizeServerURLV0(inv.ServerURL)
	if err != nil {
		return clientErrorBootstrapAppSpecEnvelopeV0(inv, err, start)
	}

	cmd.RequestID = inv.RequestID
	cmd.CorrelationID = inv.CorrelationID
	if strings.TrimSpace(inv.IdempotencyKey) != "" {
		cmd.IdempotencyKey = inv.IdempotencyKey
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "request_no_serializable", 0, false, start)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	httpReq, err := prepareCLIRESTRequestV0(ctx, baseURL, endpointBootstrapAppSpecV0(client), inv.CorrelationID, payload)
	if err != nil {
		return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "server_url", "request_http_invalida", 0, true, start)
	}
	resp, err := httpClientBootstrapAppSpecV0(client, timeout).Do(httpReq)
	if err != nil {
		return transportBootstrapAppSpecEnvelopeV0(inv, err, start)
	}
	defer resp.Body.Close()

	return decodeBootstrapAppSpecResponseV0(resp, inv, start)
}

func decodeBootstrapAppSpecResponseV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	if resp.StatusCode == http.StatusBadRequest {
		errs, ok := decodeBootstrapAppSpecIssuesV0(resp.Body)
		if !ok || len(errs) == 0 {
			return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "errores", "respuesta_400_sin_errores_publicos", resp.StatusCode, false, start)
		}
		return NewCliOutputErrorEnvelopeV0(inv, BootstrapAppSpecCliContractV0, BootstrapAppSpecCliContractVersionV0, errs, cliMetaV0(start, resp.StatusCode, false))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "status_code", "status_no_2xx", resp.StatusCode, retryableStatusV0(resp.StatusCode), start)
	}

	var result orquestadirector.BootstrapProyectoDesdeAppSpecResultV0
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", "json_invalido", resp.StatusCode, false, start)
	}
	if !validBootstrapAppSpecResultV0(result) {
		return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "data", "bootstrap_result_invalido", resp.StatusCode, false, start)
	}
	return NewCliOutputOKEnvelopeV0(inv, BootstrapAppSpecCliContractV0, BootstrapAppSpecCliContractVersionV0, result, cliMetaV0(start, resp.StatusCode, false))
}

func decodeBootstrapAppSpecIssuesV0(body io.Reader) ([]CliPublicErrorV0, bool) {
	var out bootstrapAppSpecHTTPErrorV0
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		return nil, false
	}
	errs := make([]CliPublicErrorV0, 0, len(out.Errores)+1)
	if strings.TrimSpace(out.Code) != "" {
		errs = append(errs, bootstrapAppSpecIssueToCliErrorV0(out.Code, out.Field, out.Message))
	}
	for _, issue := range out.Errores {
		code := firstNonEmptyBootstrapAppSpecV0(issue.Code, issue.Codigo)
		field := firstNonEmptyBootstrapAppSpecV0(issue.Field, issue.Campo)
		errs = append(errs, bootstrapAppSpecIssueToCliErrorV0(code, field, issue.Message))
	}
	return errs, true
}

func validBootstrapAppSpecResultV0(result orquestadirector.BootstrapProyectoDesdeAppSpecResultV0) bool {
	return strings.TrimSpace(result.ProjectRef) != "" &&
		strings.TrimSpace(result.AppSpecRef) != "" &&
		strings.TrimSpace(result.RegistroAceptado.RegistroID) != "" &&
		strings.TrimSpace(result.RegistroAceptado.BootstrapVersion) == "v0" &&
		result.StartRunCommand.CommandType == orquestacoreworkflow.OrchestrationCommandStartRunV0 &&
		len(result.WorkflowResult.Events) > 0 &&
		result.WorkflowResult.Events[0].EventType == orquestacoreworkflow.OrchestrationEventRunStartedV0
}
