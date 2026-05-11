package orquestacoreworkflow

import "strings"

const ErrAppSpecRunDraftInvalidoV0 = "appspec_run_draft_invalido"

var forbiddenAppSpecRunDraftFragmentsV0 = []string{
	"secret",
	"secreto",
	"token",
	"password",
	"credential",
	"credencial",
	"api_key",
	"oauth",
	"db",
	"database",
	"sql",
	"dsn",
	"connection",
	"conexion",
	"table",
	"tabla",
	"runtime",
	"provider",
	"proveedor",
	"home",
	"tmux",
	"docker",
	"http",
	"cli",
	"mcp",
}

type AppSpecRunDraftV0 struct {
	RunID          string `json:"run_id"`
	ProjectRef     string `json:"project_ref"`
	AppSpecRef     string `json:"app_spec_ref"`
	RequestedBy    string `json:"requested_by,omitempty"`
	CorrelationID  string `json:"correlation_id,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
	OccurredAt     string `json:"occurred_at"`
}

func StartRunFromAppSpecV0(draft AppSpecRunDraftV0) (OrchestrationCommandV0, error) {
	normalized, err := NormalizeAppSpecRunDraftV0(draft)
	if err != nil {
		return OrchestrationCommandV0{}, err
	}
	return NewStartRunCommandV0(appSpecRunDraftCommandMetaV0(normalized), StartRunCommandPayloadV0{
		ProjectRef: normalized.ProjectRef,
		AppSpecRef: normalized.AppSpecRef,
	})
}

func NormalizeAppSpecRunDraftV0(draft AppSpecRunDraftV0) (AppSpecRunDraftV0, error) {
	normalized := normalizeAppSpecRunDraftV0(draft)
	if err := ValidateAppSpecRunDraftV0(normalized); err != nil {
		return AppSpecRunDraftV0{}, err
	}
	return normalized, nil
}

func ValidateAppSpecRunDraftV0(draft AppSpecRunDraftV0) error {
	if strings.TrimSpace(draft.RunID) == "" {
		return commandErrorV0(ErrAppSpecRunDraftInvalidoV0, "run_id")
	}
	if strings.TrimSpace(draft.ProjectRef) == "" {
		return commandErrorV0(ErrAppSpecRunDraftInvalidoV0, "project_ref")
	}
	if strings.TrimSpace(draft.AppSpecRef) == "" {
		return commandErrorV0(ErrAppSpecRunDraftInvalidoV0, "app_spec_ref")
	}
	if strings.TrimSpace(draft.IdempotencyKey) == "" {
		return commandErrorV0(ErrIdempotencyKeyRequeridaV0, "idempotency_key")
	}
	if strings.TrimSpace(draft.OccurredAt) == "" {
		return commandErrorV0(ErrAppSpecRunDraftInvalidoV0, "occurred_at")
	}
	if appSpecRunDraftHasForbiddenDetailV0(draft) {
		return commandErrorV0(ErrDetalleProhibidoV0, "draft")
	}
	return nil
}

func normalizeAppSpecRunDraftV0(draft AppSpecRunDraftV0) AppSpecRunDraftV0 {
	return AppSpecRunDraftV0{
		RunID:          strings.TrimSpace(draft.RunID),
		ProjectRef:     strings.TrimSpace(draft.ProjectRef),
		AppSpecRef:     strings.TrimSpace(draft.AppSpecRef),
		RequestedBy:    strings.TrimSpace(draft.RequestedBy),
		CorrelationID:  strings.TrimSpace(draft.CorrelationID),
		IdempotencyKey: strings.TrimSpace(draft.IdempotencyKey),
		OccurredAt:     strings.TrimSpace(draft.OccurredAt),
	}
}

func appSpecRunDraftCommandMetaV0(draft AppSpecRunDraftV0) OrchestrationCommandMetaV0 {
	return OrchestrationCommandMetaV0{
		CommandID:      "cmd:start_run_from_appspec:v0:" + draft.RunID,
		RunID:          draft.RunID,
		IdempotencyKey: draft.IdempotencyKey,
		CorrelationID:  draft.CorrelationID,
		RequestedBy:    draft.RequestedBy,
		OccurredAt:     draft.OccurredAt,
	}
}

func appSpecRunDraftHasForbiddenDetailV0(draft AppSpecRunDraftV0) bool {
	values := []string{
		draft.RunID,
		draft.ProjectRef,
		draft.AppSpecRef,
		draft.RequestedBy,
		draft.CorrelationID,
		draft.IdempotencyKey,
	}
	for _, value := range values {
		if appSpecRunDraftTextHasForbiddenDetailV0(value) {
			return true
		}
	}
	return false
}

func appSpecRunDraftTextHasForbiddenDetailV0(value string) bool {
	lower := strings.ToLower(value)
	for _, fragment := range forbiddenAppSpecRunDraftFragmentsV0 {
		if containsForbiddenFragmentV0(lower, fragment) {
			return true
		}
	}
	return false
}
