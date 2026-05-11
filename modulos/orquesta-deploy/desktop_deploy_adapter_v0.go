package orquestadeploy

import (
	"errors"
	"strings"
)

const (
	DesktopDeployAdapterNameV0     = "DesktopDeployAdapterV0"
	DesktopDeployResultSchemaV0    = "DesktopDeployResultV0"
	DesktopDeployModeDryRunV0      = "dry_run"
	DesktopDeployStatusPreparadoV0 = "preparado"

	ErrDesktopDeployTargetInvalidoV0       = "desktop_deploy_target_invalido"
	ErrDesktopDeployResultadoInvalidoV0    = "desktop_deploy_resultado_invalido"
	ErrDesktopDeployRollbackNoReversibleV0 = "desktop_deploy_rollback_no_reversible"
	ErrDesktopDeploySinPaqueteV0           = "desktop_deploy_sin_paquete"
	ErrDesktopDeployContenedorNoPuroV0     = "desktop_deploy_contenedor_no_puro"
)

type DesktopDeployAdapterV0 struct{}

type DesktopDeployResultV0 struct {
	SchemaVersion       string                    `json:"schema_version"`
	Adapter             string                    `json:"adapter"`
	Mode                string                    `json:"mode"`
	Status              string                    `json:"status"`
	RequestID           string                    `json:"request_id"`
	PlanID              string                    `json:"plan_id"`
	TargetResuelto      string                    `json:"target_resuelto"`
	OSPreparados        []DesktopDeployOSV0       `json:"os_preparados"`
	Evidencias          []DesktopDeployEvidenceV0 `json:"evidencias"`
	ValidacionEntorno   []ComprobacionV0          `json:"validacion_entorno"`
	ArtefactosPrevistos []ArtefactoPrevistoV0     `json:"artefactos_previstos"`
	Healthcheck         []ComprobacionV0          `json:"healthcheck"`
	Rollback            RollbackV0                `json:"rollback"`
	AccionesPrevistas   []AccionPrevistaV0        `json:"acciones_previstas"`
	Advertencias        []string                  `json:"advertencias"`
}

type DesktopDeployOSV0 struct {
	OS                string   `json:"os"`
	Status            string   `json:"status"`
	Motivo            string   `json:"motivo"`
	RequisitosPrevios []string `json:"requisitos_previos"`
}

type DesktopDeployEvidenceV0 struct {
	ID          string `json:"id"`
	Descripcion string `json:"descripcion"`
}

type DesktopDeployIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type DesktopDeployAdapterV0Error struct {
	Issues []DesktopDeployIssueV0 `json:"issues"`
}

func (err DesktopDeployAdapterV0Error) Error() string {
	if len(err.Issues) == 0 {
		return ErrDesktopDeployResultadoInvalidoV0
	}
	return err.Issues[0].Code
}

func PrepareDesktopDeployDryRunV0(plan DeploymentPlanV0) (DesktopDeployResultV0, error) {
	return DesktopDeployAdapterV0{}.PrepareDryRun(plan)
}

func (DesktopDeployAdapterV0) PrepareDryRun(plan DeploymentPlanV0) (DesktopDeployResultV0, error) {
	if err := ValidateDeploymentPlanV0(plan); err != nil {
		return DesktopDeployResultV0{}, err
	}
	if issues := validateDesktopDeployPlanV0(plan); len(issues) > 0 {
		return DesktopDeployResultV0{}, DesktopDeployAdapterV0Error{Issues: issues}
	}

	result := DesktopDeployResultV0{
		SchemaVersion:       DesktopDeployResultSchemaV0,
		Adapter:             DesktopDeployAdapterNameV0,
		Mode:                DesktopDeployModeDryRunV0,
		Status:              DesktopDeployStatusPreparadoV0,
		RequestID:           strings.TrimSpace(plan.RequestID),
		PlanID:              strings.TrimSpace(plan.PlanID),
		TargetResuelto:      "desktop",
		OSPreparados:        desktopDeployOSResultsV0(plan.MatrizOS),
		Evidencias:          desktopDeployEvidencesV0(),
		ValidacionEntorno:   copyComprobacionesV0(plan.ValidacionEntorno),
		ArtefactosPrevistos: copyArtefactosPrevistosV0(plan.ArtefactosPrevistos),
		Healthcheck:         copyComprobacionesV0(plan.Healthcheck),
		Rollback:            copyRollbackV0(plan.Rollback),
		AccionesPrevistas:   copyAccionesPrevistasV0(plan.AccionesPrevistas),
		Advertencias: append(
			copyStringsV0(plan.Advertencias),
			"Dry-run desktop: no se empaqueto binario, no se firmo y no se escribieron artefactos reales.",
		),
	}
	if err := ValidateDesktopDeployResultV0(result); err != nil {
		return DesktopDeployResultV0{}, err
	}
	return result, nil
}

func ValidateDesktopDeployResultV0(result DesktopDeployResultV0) error {
	var issues []DesktopDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, DesktopDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(result.SchemaVersion) != DesktopDeployResultSchemaV0 {
		add(ErrDesktopDeployResultadoInvalidoV0, "schema_version")
	}
	if strings.TrimSpace(result.Adapter) != DesktopDeployAdapterNameV0 {
		add(ErrDesktopDeployResultadoInvalidoV0, "adapter")
	}
	if strings.TrimSpace(result.Mode) != DesktopDeployModeDryRunV0 {
		add(ErrDesktopDeployResultadoInvalidoV0, "mode")
	}
	if strings.TrimSpace(result.Status) != DesktopDeployStatusPreparadoV0 {
		add(ErrDesktopDeployResultadoInvalidoV0, "status")
	}
	if strings.TrimSpace(result.PlanID) == "" {
		add(ErrDesktopDeployResultadoInvalidoV0, "plan_id")
	}
	if strings.TrimSpace(result.TargetResuelto) != "desktop" {
		add(ErrDesktopDeployTargetInvalidoV0, "target_resuelto")
	}
	if !desktopDeployOSExactosV0(result.OSPreparados) {
		add(ErrMatrizOSIncompletaV0, "os_preparados")
	}
	if len(result.Evidencias) == 0 {
		add(ErrDesktopDeployResultadoInvalidoV0, "evidencias")
	}
	if len(result.ValidacionEntorno) == 0 {
		add(ErrValidacionEntornoIncompletaV0, "validacion_entorno")
	}
	if len(result.ArtefactosPrevistos) == 0 || !desktopHasPackageArtifactV0(result.ArtefactosPrevistos) {
		add(ErrDesktopDeploySinPaqueteV0, "artefactos_previstos")
	}
	if len(result.Healthcheck) == 0 {
		add(ErrHealthcheckIncompletoV0, "healthcheck")
	}
	if !result.Rollback.Reversible {
		add(ErrDesktopDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if algunaAccionRequiereContenedorV0(result.AccionesPrevistas) || algunaAccionRequiereContenedorV0(result.Rollback.PasosPrevistos) {
		add(ErrDesktopDeployContenedorNoPuroV0, "acciones_previstas")
	}
	if len(issues) > 0 {
		return DesktopDeployAdapterV0Error{Issues: issues}
	}
	return nil
}

func validateDesktopDeployPlanV0(plan DeploymentPlanV0) []DesktopDeployIssueV0 {
	var issues []DesktopDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, DesktopDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(plan.DeployTarget) != "desktop" || strings.TrimSpace(plan.TargetResuelto) != "desktop" {
		add(ErrDesktopDeployTargetInvalidoV0, "deploy_target")
	}
	for _, row := range plan.MatrizOS {
		if !aplicaTrueV0(row.Aplica) || strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			add(ErrMatrizOSIncompletaV0, "matriz_os."+strings.TrimSpace(row.OS))
		}
	}
	if !plan.Rollback.Reversible {
		add(ErrDesktopDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !desktopHasPackageArtifactV0(plan.ArtefactosPrevistos) {
		add(ErrDesktopDeploySinPaqueteV0, "artefactos_previstos")
	}
	if algunaAccionRequiereContenedorV0(plan.AccionesPrevistas) || algunaAccionRequiereContenedorV0(plan.Rollback.PasosPrevistos) {
		add(ErrDesktopDeployContenedorNoPuroV0, "acciones_previstas")
	}
	return issues
}

func desktopDeployOSResultsV0(rows []FilaOSV0) []DesktopDeployOSV0 {
	results := make([]DesktopDeployOSV0, 0, len(rows))
	for _, row := range rows {
		results = append(results, DesktopDeployOSV0{
			OS:                strings.TrimSpace(row.OS),
			Status:            DesktopDeployStatusPreparadoV0,
			Motivo:            strings.TrimSpace(row.Motivo),
			RequisitosPrevios: copyStringsV0(row.RequisitosPrevios),
		})
	}
	return results
}

func desktopDeployEvidencesV0() []DesktopDeployEvidenceV0 {
	return []DesktopDeployEvidenceV0{
		{ID: "deployment_plan_validado", Descripcion: "DeploymentPlanV0 validado antes de preparar el resultado desktop."},
		{ID: "target_desktop_confirmado", Descripcion: "Target desktop confirmado para linux, darwin y windows sin imponer proveedor ni runtime."},
		{ID: "paquete_desktop_previsto", Descripcion: "El plan declara artefactos de empaquetado desktop sin crear instaladores ni bundles reales."},
		{ID: "dry_run_sin_efectos", Descripcion: "Preparacion declarativa sin empaquetar binarios, firmar ni tocar filesystem productivo."},
	}
}

func desktopDeployOSExactosV0(rows []DesktopDeployOSV0) bool {
	if len(rows) != 3 {
		return false
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Status) != DesktopDeployStatusPreparadoV0 || strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			return false
		}
		values = append(values, row.OS)
	}
	return osExactosV0(values)
}

func desktopHasPackageArtifactV0(artifacts []ArtefactoPrevistoV0) bool {
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Tipo) == "paquete_desktop" {
			return true
		}
	}
	return false
}

func HasDesktopDeployIssueV0(err error, code string) bool {
	var desktopErr DesktopDeployAdapterV0Error
	if !errors.As(err, &desktopErr) {
		return false
	}
	for _, issue := range desktopErr.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
