package orquestacli

import (
	"bytes"
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
	BootstrapAppSpecCliLegacyEndpointV0     = "/api/v0/director/bootstrap/appspec"
	BootstrapAppSpecCliPreferredEndpointV0  = "/api/v0/apps/director"
	BootstrapAppSpecCliCorrelationHeaderV0  = CliCorrelationHeaderV0
	BootstrapAppSpecCliContractV0           = "BootstrapProyectoDesdeAppSpec"
	BootstrapAppSpecCliContractVersionV0    = "v0"
	BootstrapAppSpecCliDefaultCommandV0     = "app spec bootstrap"
	BootstrapAppSpecCliQuarantineDetailV0   = "bootstrap_appspec_legacy_route_en_cuarentena_use_api_v0_apps_director"
	BootstrapAppSpecCliQuarantineEvidenceV0 = "T86_bootstrap_appspec_legacy_route_quarantine"
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
	config, err := newBootstrapAppSpecLegacyClientConfigV0(serverURL, timeout)
	if err != nil {
		return nil, err
	}
	return &BootstrapAppSpecCliClientV0{
		BaseURL:    config.BaseURL,
		Endpoint:   config.Endpoint,
		Timeout:    config.Timeout,
		HTTPClient: newCLILoopbackHTTPClientV0(config.Timeout),
	}, nil
}

func (client *BootstrapAppSpecCliClientV0) BootstrapProyectoDesdeAppSpec(ctx context.Context, inv CliInvocationContextV0, cmd orquestadirector.BootstrapProyectoDesdeAppSpecCommandV0) CliOutputEnvelopeV0 {
	_ = ctx
	_ = cmd
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
	return bootstrapAppSpecQuarantineEnvelopeV0(inv, start)
}

func decodeBootstrapAppSpecResponseV0(resp *http.Response, inv CliInvocationContextV0, start time.Time) CliOutputEnvelopeV0 {
	if resp.StatusCode == http.StatusBadRequest {
		raw, detail := readCLIRESTResponseBodyForCommandV0(resp, inv.Command)
		if detail != "" {
			return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
		}
		errs, ok := decodeBootstrapAppSpecIssuesV0(bytes.NewReader(raw))
		if !ok || len(errs) == 0 {
			return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "errores", "respuesta_400_sin_errores_publicos", resp.StatusCode, false, start)
		}
		return NewCliOutputErrorEnvelopeV0(inv, BootstrapAppSpecCliContractV0, BootstrapAppSpecCliContractVersionV0, errs, cliMetaV0(start, resp.StatusCode, false))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode > 299 {
		return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrErrorTransporteV0, "status_code", cliRESTNo2xxDetailForCommandV0(resp, inv.Command), resp.StatusCode, retryableStatusV0(resp.StatusCode), start)
	}

	var result orquestadirector.BootstrapProyectoDesdeAppSpecResultV0
	if detail := decodeCLIRESTJSONBodyForCommandV0(resp, inv.Command, &result); detail != "" {
		return cliSingleBootstrapAppSpecErrorEnvelopeV0(inv, CliErrRespuestaInvalidaV0, "body", detail, resp.StatusCode, false, start)
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
