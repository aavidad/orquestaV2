package orquestadirector

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const ErrDirectorBootstrapInvalidoV0 = "director_bootstrap_invalido"

type BootstrapProyectoDesdeAppSpecCommandV0 struct {
	IdempotencyKey string                                    `json:"idempotency_key"`
	AppSpec        orquestafactory.AppSpecV0                 `json:"app_spec"`
	Backlog        orquestafactory.BacklogInicialPropuestoV0 `json:"backlog"`
	RequestedBy    string                                    `json:"requested_by,omitempty"`
	OccurredAt     string                                    `json:"occurred_at"`
	RequestID      string                                    `json:"request_id,omitempty"`
	CorrelationID  string                                    `json:"correlation_id,omitempty"`
}

type BootstrapProyectoDesdeAppSpecResultV0 struct {
	RegistroAceptado BootstrapRegistroAceptadoV0                       `json:"registro_aceptado"`
	ProjectRef       string                                            `json:"project_ref"`
	AppSpecRef       string                                            `json:"app_spec_ref"`
	StartRunCommand  orquestacoreworkflow.OrchestrationCommandV0       `json:"start_run_command"`
	WorkflowResult   orquestacoreworkflow.OrchestrationCommandResultV0 `json:"workflow_result"`
}

type BootstrapRegistroAceptadoV0 struct {
	RegistroID       string   `json:"registro_id"`
	ProjectRef       string   `json:"project_ref"`
	AppSpecRef       string   `json:"app_spec_ref"`
	Estado           string   `json:"estado"`
	EventosDominio   int      `json:"eventos_dominio"`
	Warnings         []string `json:"warnings"`
	FasesIniciales   int      `json:"fases_iniciales"`
	Microtareas      int      `json:"microtareas"`
	Contratos        int      `json:"contratos"`
	RequestID        string   `json:"request_id,omitempty"`
	CorrelationID    string   `json:"correlation_id,omitempty"`
	BootstrapVersion string   `json:"bootstrap_version"`
}

type BootstrapProyectoDesdeAppSpecErrorV0 struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Field         string `json:"field,omitempty"`
	Retryable     bool   `json:"retryable"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (err BootstrapProyectoDesdeAppSpecErrorV0) Error() string {
	return err.Code
}

func BootstrapProyectoDesdeAppSpecV0(
	cmd BootstrapProyectoDesdeAppSpecCommandV0,
) (BootstrapProyectoDesdeAppSpecResultV0, error) {
	normalized, err := normalizeBootstrapCommandV0(cmd)
	if err != nil {
		return BootstrapProyectoDesdeAppSpecResultV0{}, err
	}
	accepted, err := orquestacore.RegistrarProyectoDesdeAppSpecV0(coreRegisterCommandV0(normalized))
	if err != nil {
		return BootstrapProyectoDesdeAppSpecResultV0{}, err
	}
	startRun, err := orquestacoreworkflow.StartRunFromAppSpecV0(workflowDraftV0(normalized, accepted))
	if err != nil {
		return BootstrapProyectoDesdeAppSpecResultV0{}, err
	}
	workflowResult, err := orquestacoreworkflow.HandleCommandV0(orquestacoreworkflow.OrchestrationRunV0{}, startRun)
	if err != nil {
		return BootstrapProyectoDesdeAppSpecResultV0{}, err
	}
	return bootstrapResultV0(normalized, accepted, startRun, workflowResult), nil
}

func normalizeBootstrapCommandV0(
	cmd BootstrapProyectoDesdeAppSpecCommandV0,
) (BootstrapProyectoDesdeAppSpecCommandV0, error) {
	normalized := cmd
	normalized.IdempotencyKey = strings.TrimSpace(cmd.IdempotencyKey)
	normalized.RequestedBy = strings.TrimSpace(cmd.RequestedBy)
	normalized.OccurredAt = strings.TrimSpace(cmd.OccurredAt)
	normalized.RequestID = strings.TrimSpace(cmd.RequestID)
	normalized.CorrelationID = strings.TrimSpace(cmd.CorrelationID)
	if normalized.IdempotencyKey == "" {
		return BootstrapProyectoDesdeAppSpecCommandV0{}, bootstrapErrorV0(
			"idempotency_key requerida",
			"idempotency_key",
			normalized.CorrelationID,
		)
	}
	if normalized.OccurredAt == "" {
		return BootstrapProyectoDesdeAppSpecCommandV0{}, bootstrapErrorV0(
			"occurred_at requerido",
			"occurred_at",
			normalized.CorrelationID,
		)
	}
	return normalized, nil
}

func coreRegisterCommandV0(
	cmd BootstrapProyectoDesdeAppSpecCommandV0,
) orquestacore.RegistrarProyectoDesdeAppSpecCommandV0 {
	return orquestacore.RegistrarProyectoDesdeAppSpecCommandV0{
		IdempotencyKey: cmd.IdempotencyKey,
		AppSpec:        cmd.AppSpec,
		AppSpecVersion: orquestacore.RegistrarProyectoDesdeAppSpecVersionV0,
		Backlog:        cmd.Backlog,
		BacklogVersion: orquestacore.RegistrarProyectoDesdeAppSpecVersionV0,
		Origen:         "orquesta-director",
		CorrelationID:  cmd.CorrelationID,
		RequestID:      firstNonEmptyV0(cmd.RequestID, cmd.AppSpec.RequestID),
		SolicitadoEn:   cmd.OccurredAt,
	}
}

func workflowDraftV0(
	cmd BootstrapProyectoDesdeAppSpecCommandV0,
	accepted orquestacore.RegistroProyectoAceptadoV0,
) orquestacoreworkflow.AppSpecRunDraftV0 {
	plan := accepted.ProyectoPlanBorrador
	return orquestacoreworkflow.AppSpecRunDraftV0{
		RunID:          opaqueRefV0("run", accepted.RegistroID),
		ProjectRef:     opaqueRefV0("project", plan.ProyectoIDPropuesto),
		AppSpecRef:     opaqueRefV0("appspec", plan.AppSpecRef),
		RequestedBy:    cmd.RequestedBy,
		CorrelationID:  cmd.CorrelationID,
		IdempotencyKey: cmd.IdempotencyKey,
		OccurredAt:     cmd.OccurredAt,
	}
}

func bootstrapResultV0(
	cmd BootstrapProyectoDesdeAppSpecCommandV0,
	accepted orquestacore.RegistroProyectoAceptadoV0,
	startRun orquestacoreworkflow.OrchestrationCommandV0,
	workflowResult orquestacoreworkflow.OrchestrationCommandResultV0,
) BootstrapProyectoDesdeAppSpecResultV0 {
	plan := accepted.ProyectoPlanBorrador
	projectRef := opaqueRefV0("project", plan.ProyectoIDPropuesto)
	appSpecRef := opaqueRefV0("appspec", plan.AppSpecRef)
	return BootstrapProyectoDesdeAppSpecResultV0{
		RegistroAceptado: compactAcceptedV0(cmd, accepted),
		ProjectRef:       projectRef,
		AppSpecRef:       appSpecRef,
		StartRunCommand:  startRun,
		WorkflowResult:   workflowResult,
	}
}

func compactAcceptedV0(
	cmd BootstrapProyectoDesdeAppSpecCommandV0,
	accepted orquestacore.RegistroProyectoAceptadoV0,
) BootstrapRegistroAceptadoV0 {
	plan := accepted.ProyectoPlanBorrador
	return BootstrapRegistroAceptadoV0{
		RegistroID:       accepted.RegistroID,
		ProjectRef:       opaqueRefV0("project", plan.ProyectoIDPropuesto),
		AppSpecRef:       opaqueRefV0("appspec", plan.AppSpecRef),
		Estado:           plan.Estado,
		EventosDominio:   len(accepted.EventosDominio),
		Warnings:         append([]string{}, accepted.Warnings...),
		FasesIniciales:   len(plan.FasesIniciales),
		Microtareas:      len(plan.Microtareas),
		Contratos:        len(plan.ContratosRequeridos),
		RequestID:        firstNonEmptyV0(cmd.RequestID, plan.RequestID),
		CorrelationID:    cmd.CorrelationID,
		BootstrapVersion: "v0",
	}
}

func opaqueRefV0(prefix string, value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return prefix + "ref_" + hex.EncodeToString(sum[:])[:16]
}

func bootstrapErrorV0(message string, field string, correlationID string) BootstrapProyectoDesdeAppSpecErrorV0 {
	return BootstrapProyectoDesdeAppSpecErrorV0{
		Code:          ErrDirectorBootstrapInvalidoV0,
		Message:       message,
		Field:         field,
		Retryable:     false,
		CorrelationID: strings.TrimSpace(correlationID),
	}
}

func firstNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
