package orquestadeploy

import (
	"errors"
	"strings"
)

const (
	ContainerDeployAdapterNameV0     = "ContainerDeployAdapterV0"
	ContainerDeployResultSchemaV0    = "ContainerDeployResultV0"
	ContainerDeployModeDryRunV0      = "dry_run"
	ContainerDeployStatusPreparadoV0 = "preparado"

	ErrContainerDeployTargetInvalidoV0       = "container_deploy_target_invalido"
	ErrContainerDeployResultadoInvalidoV0    = "container_deploy_resultado_invalido"
	ErrContainerDeployContenedorNoAceptadoV0 = "container_deploy_contenedor_no_aceptado"
	ErrContainerDeploySinAccionContenedorV0  = "container_deploy_sin_accion_contenedor"
	ErrContainerDeployRollbackNoReversibleV0 = "container_deploy_rollback_no_reversible"
)

type ContainerDeployAdapterV0 struct{}

type ContainerDeployResultV0 struct {
	SchemaVersion       string                      `json:"schema_version"`
	Adapter             string                      `json:"adapter"`
	Mode                string                      `json:"mode"`
	Status              string                      `json:"status"`
	RequestID           string                      `json:"request_id"`
	PlanID              string                      `json:"plan_id"`
	TargetResuelto      string                      `json:"target_resuelto"`
	OSPreparados        []ContainerDeployOSV0       `json:"os_preparados"`
	Evidencias          []ContainerDeployEvidenceV0 `json:"evidencias"`
	ValidacionEntorno   []ComprobacionV0            `json:"validacion_entorno"`
	ArtefactosPrevistos []ArtefactoPrevistoV0       `json:"artefactos_previstos"`
	Healthcheck         []ComprobacionV0            `json:"healthcheck"`
	Rollback            RollbackV0                  `json:"rollback"`
	AccionesPrevistas   []AccionPrevistaV0          `json:"acciones_previstas"`
	Advertencias        []string                    `json:"advertencias"`
}

type ContainerDeployOSV0 struct {
	OS                string   `json:"os"`
	Status            string   `json:"status"`
	Motivo            string   `json:"motivo"`
	RequisitosPrevios []string `json:"requisitos_previos"`
}

type ContainerDeployEvidenceV0 struct {
	ID          string `json:"id"`
	Descripcion string `json:"descripcion"`
}

type ContainerDeployIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type ContainerDeployAdapterV0Error struct {
	Issues []ContainerDeployIssueV0 `json:"issues"`
}

func (err ContainerDeployAdapterV0Error) Error() string {
	if len(err.Issues) == 0 {
		return ErrContainerDeployResultadoInvalidoV0
	}
	return err.Issues[0].Code
}

func PrepareContainerDeployDryRunV0(plan DeploymentPlanV0) (ContainerDeployResultV0, error) {
	return ContainerDeployAdapterV0{}.PrepareDryRun(plan)
}

func (ContainerDeployAdapterV0) PrepareDryRun(plan DeploymentPlanV0) (ContainerDeployResultV0, error) {
	if err := ValidateDeploymentPlanV0(plan); err != nil {
		return ContainerDeployResultV0{}, err
	}
	if issues := validateContainerDeployPlanV0(plan); len(issues) > 0 {
		return ContainerDeployResultV0{}, ContainerDeployAdapterV0Error{Issues: issues}
	}

	result := ContainerDeployResultV0{
		SchemaVersion:       ContainerDeployResultSchemaV0,
		Adapter:             ContainerDeployAdapterNameV0,
		Mode:                ContainerDeployModeDryRunV0,
		Status:              ContainerDeployStatusPreparadoV0,
		RequestID:           strings.TrimSpace(plan.RequestID),
		PlanID:              strings.TrimSpace(plan.PlanID),
		TargetResuelto:      "contenedor",
		OSPreparados:        containerDeployOSResultsV0(plan.MatrizOS),
		Evidencias:          containerDeployEvidencesV0(),
		ValidacionEntorno:   copyComprobacionesV0(plan.ValidacionEntorno),
		ArtefactosPrevistos: copyArtefactosPrevistosV0(plan.ArtefactosPrevistos),
		Healthcheck:         copyComprobacionesV0(plan.Healthcheck),
		Rollback:            copyRollbackV0(plan.Rollback),
		AccionesPrevistas:   copyAccionesPrevistasV0(plan.AccionesPrevistas),
		Advertencias: append(
			copyStringsV0(plan.Advertencias),
			"Dry-run contenedor: no se construyeron imagenes ni se materializaron artefactos.",
		),
	}
	if err := ValidateContainerDeployResultV0(result); err != nil {
		return ContainerDeployResultV0{}, err
	}
	return result, nil
}

func ValidateContainerDeployResultV0(result ContainerDeployResultV0) error {
	var issues []ContainerDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, ContainerDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(result.SchemaVersion) != ContainerDeployResultSchemaV0 {
		add(ErrContainerDeployResultadoInvalidoV0, "schema_version")
	}
	if strings.TrimSpace(result.Adapter) != ContainerDeployAdapterNameV0 {
		add(ErrContainerDeployResultadoInvalidoV0, "adapter")
	}
	if strings.TrimSpace(result.Mode) != ContainerDeployModeDryRunV0 {
		add(ErrContainerDeployResultadoInvalidoV0, "mode")
	}
	if strings.TrimSpace(result.Status) != ContainerDeployStatusPreparadoV0 {
		add(ErrContainerDeployResultadoInvalidoV0, "status")
	}
	if strings.TrimSpace(result.PlanID) == "" {
		add(ErrContainerDeployResultadoInvalidoV0, "plan_id")
	}
	if strings.TrimSpace(result.TargetResuelto) != "contenedor" {
		add(ErrContainerDeployTargetInvalidoV0, "target_resuelto")
	}
	if !containerDeployOSExactosV0(result.OSPreparados) {
		add(ErrMatrizOSIncompletaV0, "os_preparados")
	}
	if len(result.Evidencias) == 0 {
		add(ErrContainerDeployResultadoInvalidoV0, "evidencias")
	}
	if len(result.ValidacionEntorno) == 0 {
		add(ErrValidacionEntornoIncompletaV0, "validacion_entorno")
	}
	if len(result.ArtefactosPrevistos) == 0 {
		add(ErrArtefactosPrevistosIncompletosV0, "artefactos_previstos")
	}
	if len(result.Healthcheck) == 0 {
		add(ErrHealthcheckIncompletoV0, "healthcheck")
	}
	if !result.Rollback.Reversible {
		add(ErrContainerDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !algunaAccionRequiereContenedorV0(result.AccionesPrevistas) {
		add(ErrContainerDeploySinAccionContenedorV0, "acciones_previstas")
	}
	if len(issues) > 0 {
		return ContainerDeployAdapterV0Error{Issues: issues}
	}
	return nil
}

func validateContainerDeployPlanV0(plan DeploymentPlanV0) []ContainerDeployIssueV0 {
	var issues []ContainerDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, ContainerDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(plan.DeployTarget) != "contenedor" || strings.TrimSpace(plan.TargetResuelto) != "contenedor" {
		add(ErrContainerDeployTargetInvalidoV0, "deploy_target")
	}
	if aceptacionContenedorFalseV0(plan.AceptacionContenedor) {
		add(ErrContainerDeployContenedorNoAceptadoV0, "aceptacion_contenedor")
	}
	for _, row := range plan.MatrizOS {
		if strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			add(ErrMatrizOSIncompletaV0, "matriz_os."+strings.TrimSpace(row.OS))
		}
	}
	if !plan.Rollback.Reversible {
		add(ErrContainerDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !algunaAccionRequiereContenedorV0(plan.AccionesPrevistas) {
		add(ErrContainerDeploySinAccionContenedorV0, "acciones_previstas")
	}
	return issues
}

func containerDeployOSResultsV0(rows []FilaOSV0) []ContainerDeployOSV0 {
	results := make([]ContainerDeployOSV0, 0, len(rows))
	for _, row := range rows {
		results = append(results, ContainerDeployOSV0{
			OS:                strings.TrimSpace(row.OS),
			Status:            ContainerDeployStatusPreparadoV0,
			Motivo:            strings.TrimSpace(row.Motivo),
			RequisitosPrevios: copyStringsV0(row.RequisitosPrevios),
		})
	}
	return results
}

func containerDeployEvidencesV0() []ContainerDeployEvidenceV0 {
	return []ContainerDeployEvidenceV0{
		{ID: "deployment_plan_validado", Descripcion: "DeploymentPlanV0 validado antes de preparar el resultado de contenedor."},
		{ID: "target_contenedor_confirmado", Descripcion: "Target contenedor confirmado con aceptacion explicita y alcance dry-run."},
		{ID: "accion_contenedor_presente", Descripcion: "El plan declara al menos una accion que requiere contenedor sin ejecutar runtime real."},
		{ID: "dry_run_sin_efectos", Descripcion: "Preparacion declarativa sin construir imagenes, publicar artefactos ni tocar el entorno."},
	}
}

func containerDeployOSExactosV0(rows []ContainerDeployOSV0) bool {
	if len(rows) != 3 {
		return false
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Status) != ContainerDeployStatusPreparadoV0 || strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			return false
		}
		values = append(values, row.OS)
	}
	return osExactosV0(values)
}

func HasContainerDeployIssueV0(err error, code string) bool {
	var containerErr ContainerDeployAdapterV0Error
	if !errors.As(err, &containerErr) {
		return false
	}
	for _, issue := range containerErr.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
