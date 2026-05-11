package orquestadeploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const (
	DeploymentPlanSchemaV0 = "DeploymentPlanV0"
	AppSpecSchemaV0        = "AppSpecV0"

	ErrDeploymentPlanJSONInvalidoV0      = "deployment_plan_json_invalido"
	ErrDeploymentPlanSchemaNoSoportadoV0 = "deployment_plan_schema_no_soportado"
	ErrDeployTargetRequeridoV0           = "deploy_target_requerido"
	ErrDeployTargetNoSoportadoV0         = "deploy_target_no_soportado"
	ErrMatrizOSIncompletaV0              = "matriz_os_incompleta"
	ErrContenedorNoAceptadoV0            = "contenedor_no_aceptado"
	ErrValidacionEntornoIncompletaV0     = "validacion_entorno_incompleta"
	ErrHealthcheckIncompletoV0           = "healthcheck_incompleto"
	ErrRollbackIncompletoV0              = "rollback_incompleto"
	ErrArtefactosPrevistosIncompletosV0  = "artefactos_previstos_incompletos"
	ErrScriptSinContratoV0               = "script_sin_contrato"
)

type DeploymentPlanV0 struct {
	SchemaVersion        string                `json:"schema_version"`
	RequestID            string                `json:"request_id"`
	AppSpecID            string                `json:"app_spec_id"`
	AppSpecVersion       string                `json:"app_spec_version"`
	DeployTarget         string                `json:"deploy_target"`
	SistemasOperativos   []string              `json:"sistemas_operativos"`
	AceptacionContenedor any                   `json:"aceptacion_contenedor"`
	Restricciones        []RestriccionV0       `json:"restricciones"`
	PlanID               string                `json:"plan_id"`
	TargetResuelto       string                `json:"target_resuelto"`
	MatrizOS             []FilaOSV0            `json:"matriz_os"`
	ValidacionEntorno    []ComprobacionV0      `json:"validacion_entorno"`
	ArtefactosPrevistos  []ArtefactoPrevistoV0 `json:"artefactos_previstos"`
	Healthcheck          []ComprobacionV0      `json:"healthcheck"`
	Rollback             RollbackV0            `json:"rollback"`
	AccionesPrevistas    []AccionPrevistaV0    `json:"acciones_previstas"`
	Advertencias         []string              `json:"advertencias"`
}

type RestriccionV0 struct {
	Clave string `json:"clave"`
	Valor string `json:"valor"`
}

type FilaOSV0 struct {
	OS                string   `json:"os"`
	Aplica            any      `json:"aplica"`
	Motivo            string   `json:"motivo"`
	RequisitosPrevios []string `json:"requisitos_previos"`
}

type ComprobacionV0 struct {
	ID          string `json:"id"`
	Descripcion string `json:"descripcion"`
	Obligatoria bool   `json:"obligatoria"`
}

type ArtefactoPrevistoV0 struct {
	ID          string `json:"id"`
	Tipo        string `json:"tipo"`
	Descripcion string `json:"descripcion"`
	RutaLogica  string `json:"ruta_logica"`
}

type RollbackV0 struct {
	Reversible     bool               `json:"reversible"`
	Motivo         string             `json:"motivo"`
	PasosPrevistos []AccionPrevistaV0 `json:"pasos_previstos"`
}

type AccionPrevistaV0 struct {
	Orden              int    `json:"orden"`
	Tipo               string `json:"tipo"`
	Descripcion        string `json:"descripcion"`
	RequiereContenedor bool   `json:"requiere_contenedor"`
}

type DeploymentPlanIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type DeploymentPlanContractV0Error struct {
	Issues []DeploymentPlanIssueV0 `json:"issues"`
}

func (err DeploymentPlanContractV0Error) Error() string {
	if len(err.Issues) == 0 {
		return "deployment_plan_invalido"
	}
	return err.Issues[0].Code
}

func DecodeDeploymentPlanV0(data []byte) (DeploymentPlanV0, error) {
	var plan DeploymentPlanV0
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return DeploymentPlanV0{}, DeploymentPlanContractV0Error{
			Issues: []DeploymentPlanIssueV0{{Code: ErrDeploymentPlanJSONInvalidoV0}},
		}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return DeploymentPlanV0{}, DeploymentPlanContractV0Error{
			Issues: []DeploymentPlanIssueV0{{Code: ErrDeploymentPlanJSONInvalidoV0}},
		}
	}
	if err := ValidateDeploymentPlanV0(plan); err != nil {
		return DeploymentPlanV0{}, err
	}
	return plan, nil
}

func ValidateDeploymentPlanV0(plan DeploymentPlanV0) error {
	var issues []DeploymentPlanIssueV0
	add := func(code, field string) {
		issues = append(issues, DeploymentPlanIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(plan.SchemaVersion) != DeploymentPlanSchemaV0 {
		add(ErrDeploymentPlanSchemaNoSoportadoV0, "schema_version")
	}
	if strings.TrimSpace(plan.DeployTarget) == "" {
		add(ErrDeployTargetRequeridoV0, "deploy_target")
	} else if !deploymentTargetSoportadoV0(plan.DeployTarget) {
		add(ErrDeployTargetNoSoportadoV0, "deploy_target")
	}
	if !osExactosV0(plan.SistemasOperativos) {
		add(ErrMatrizOSIncompletaV0, "sistemas_operativos")
	}
	if !matrizOSExactaV0(plan.MatrizOS) {
		add(ErrMatrizOSIncompletaV0, "matriz_os")
	}
	if deploymentTargetConcretoV0(plan.DeployTarget) && strings.TrimSpace(plan.DeployTarget) != strings.TrimSpace(plan.TargetResuelto) {
		add(ErrDeployTargetNoSoportadoV0, "target_resuelto")
	}
	if len(plan.ValidacionEntorno) == 0 {
		add(ErrValidacionEntornoIncompletaV0, "validacion_entorno")
	}
	if len(plan.Healthcheck) == 0 {
		add(ErrHealthcheckIncompletoV0, "healthcheck")
	}
	if strings.TrimSpace(plan.Rollback.Motivo) == "" || len(plan.Rollback.PasosPrevistos) == 0 {
		add(ErrRollbackIncompletoV0, "rollback")
	}
	if len(plan.ArtefactosPrevistos) == 0 {
		add(ErrArtefactosPrevistosIncompletosV0, "artefactos_previstos")
	}
	if !accionesDeclarativasV0(plan.AccionesPrevistas) {
		add(ErrScriptSinContratoV0, "acciones_previstas")
	}
	if !accionesDeclarativasV0(plan.Rollback.PasosPrevistos) {
		add(ErrScriptSinContratoV0, "rollback.pasos_previstos")
	}
	if plan.DeployTarget == "contenedor" {
		if aceptacionContenedorFalseV0(plan.AceptacionContenedor) {
			add(ErrContenedorNoAceptadoV0, "aceptacion_contenedor")
		}
		if !algunaAccionRequiereContenedorV0(plan.AccionesPrevistas) {
			add(ErrContenedorNoAceptadoV0, "acciones_previstas")
		}
	}
	if plan.DeployTarget == "local" && aceptacionContenedorFalseV0(plan.AceptacionContenedor) {
		if algunaAccionRequiereContenedorV0(plan.AccionesPrevistas) || algunaAccionRequiereContenedorV0(plan.Rollback.PasosPrevistos) {
			add(ErrContenedorNoAceptadoV0, "acciones_previstas")
		}
	}
	if len(issues) > 0 {
		return DeploymentPlanContractV0Error{Issues: issues}
	}
	return nil
}

func deploymentTargetSoportadoV0(target string) bool {
	switch strings.TrimSpace(target) {
	case "sin_preferencia", "local", "contenedor", "paas", "serverless", "kubernetes", "desktop", "mobile_store":
		return true
	default:
		return false
	}
}

func deploymentTargetConcretoV0(target string) bool {
	return deploymentTargetSoportadoV0(target) && strings.TrimSpace(target) != "sin_preferencia"
}

func osExactosV0(values []string) bool {
	if len(values) != 3 {
		return false
	}
	seen := map[string]bool{}
	for _, value := range values {
		seen[strings.TrimSpace(value)] = true
	}
	return seen["linux"] && seen["darwin"] && seen["windows"] && len(seen) == 3
}

func matrizOSExactaV0(rows []FilaOSV0) bool {
	if len(rows) != 3 {
		return false
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		values = append(values, row.OS)
	}
	return osExactosV0(values)
}

func accionesDeclarativasV0(actions []AccionPrevistaV0) bool {
	for _, action := range actions {
		switch strings.TrimSpace(action.Tipo) {
		case "validar_entorno", "preparar_artefactos", "publicar_declarativo", "verificar_salud", "registrar_rollback", "solicitar_decision":
		default:
			return false
		}
	}
	return true
}

func algunaAccionRequiereContenedorV0(actions []AccionPrevistaV0) bool {
	for _, action := range actions {
		if action.RequiereContenedor {
			return true
		}
	}
	return false
}

func aceptacionContenedorFalseV0(value any) bool {
	accepted, ok := value.(bool)
	return ok && !accepted
}

func HasDeploymentPlanIssueV0(err error, code string) bool {
	var contractErr DeploymentPlanContractV0Error
	if !errors.As(err, &contractErr) {
		return false
	}
	for _, issue := range contractErr.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
