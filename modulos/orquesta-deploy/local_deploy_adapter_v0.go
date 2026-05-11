package orquestadeploy

import (
	"errors"
	"strings"
)

const (
	LocalDeployAdapterNameV0     = "LocalDeployAdapterV0"
	LocalDeployResultSchemaV0    = "LocalDeployResultV0"
	LocalDeployModeDryRunV0      = "dry_run"
	LocalDeployStatusPreparadoV0 = "preparado"

	ErrLocalDeployTargetNoLocalV0        = "local_deploy_target_no_local"
	ErrLocalDeployResultadoInvalidoV0    = "local_deploy_resultado_invalido"
	ErrLocalDeployRollbackNoReversibleV0 = "local_deploy_rollback_no_reversible"
	ErrLocalDeployContenedorNoPuroV0     = "local_deploy_contenedor_no_puro"
)

type LocalDeployAdapterV0 struct{}

type LocalDeployResultV0 struct {
	SchemaVersion       string                  `json:"schema_version"`
	Adapter             string                  `json:"adapter"`
	Mode                string                  `json:"mode"`
	Status              string                  `json:"status"`
	RequestID           string                  `json:"request_id"`
	PlanID              string                  `json:"plan_id"`
	TargetResuelto      string                  `json:"target_resuelto"`
	OSPreparados        []LocalDeployOSV0       `json:"os_preparados"`
	Evidencias          []LocalDeployEvidenceV0 `json:"evidencias"`
	ValidacionEntorno   []ComprobacionV0        `json:"validacion_entorno"`
	ArtefactosPrevistos []ArtefactoPrevistoV0   `json:"artefactos_previstos"`
	Healthcheck         []ComprobacionV0        `json:"healthcheck"`
	Rollback            RollbackV0              `json:"rollback"`
	AccionesPrevistas   []AccionPrevistaV0      `json:"acciones_previstas"`
	Advertencias        []string                `json:"advertencias"`
}

type LocalDeployOSV0 struct {
	OS                string   `json:"os"`
	Status            string   `json:"status"`
	Motivo            string   `json:"motivo"`
	RequisitosPrevios []string `json:"requisitos_previos"`
}

type LocalDeployEvidenceV0 struct {
	ID          string `json:"id"`
	Descripcion string `json:"descripcion"`
}

type LocalDeployIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type LocalDeployAdapterV0Error struct {
	Issues []LocalDeployIssueV0 `json:"issues"`
}

func (err LocalDeployAdapterV0Error) Error() string {
	if len(err.Issues) == 0 {
		return ErrLocalDeployResultadoInvalidoV0
	}
	return err.Issues[0].Code
}

func PrepareLocalDeployDryRunV0(plan DeploymentPlanV0) (LocalDeployResultV0, error) {
	return LocalDeployAdapterV0{}.PrepareDryRun(plan)
}

func (LocalDeployAdapterV0) PrepareDryRun(plan DeploymentPlanV0) (LocalDeployResultV0, error) {
	if err := ValidateDeploymentPlanV0(plan); err != nil {
		return LocalDeployResultV0{}, err
	}
	if issues := validateLocalDeployPlanV0(plan); len(issues) > 0 {
		return LocalDeployResultV0{}, LocalDeployAdapterV0Error{Issues: issues}
	}

	result := LocalDeployResultV0{
		SchemaVersion:       LocalDeployResultSchemaV0,
		Adapter:             LocalDeployAdapterNameV0,
		Mode:                LocalDeployModeDryRunV0,
		Status:              LocalDeployStatusPreparadoV0,
		RequestID:           strings.TrimSpace(plan.RequestID),
		PlanID:              strings.TrimSpace(plan.PlanID),
		TargetResuelto:      "local",
		OSPreparados:        localDeployOSResultsV0(plan.MatrizOS),
		Evidencias:          localDeployEvidencesV0(),
		ValidacionEntorno:   copyComprobacionesV0(plan.ValidacionEntorno),
		ArtefactosPrevistos: copyArtefactosPrevistosV0(plan.ArtefactosPrevistos),
		Healthcheck:         copyComprobacionesV0(plan.Healthcheck),
		Rollback:            copyRollbackV0(plan.Rollback),
		AccionesPrevistas:   copyAccionesPrevistasV0(plan.AccionesPrevistas),
		Advertencias:        append(copyStringsV0(plan.Advertencias), "Dry-run local: no se materializaron artefactos ni cambios de entorno."),
	}
	if err := ValidateLocalDeployResultV0(result); err != nil {
		return LocalDeployResultV0{}, err
	}
	return result, nil
}

func ValidateLocalDeployResultV0(result LocalDeployResultV0) error {
	var issues []LocalDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, LocalDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(result.SchemaVersion) != LocalDeployResultSchemaV0 {
		add(ErrLocalDeployResultadoInvalidoV0, "schema_version")
	}
	if strings.TrimSpace(result.Adapter) != LocalDeployAdapterNameV0 {
		add(ErrLocalDeployResultadoInvalidoV0, "adapter")
	}
	if strings.TrimSpace(result.Mode) != LocalDeployModeDryRunV0 {
		add(ErrLocalDeployResultadoInvalidoV0, "mode")
	}
	if strings.TrimSpace(result.Status) != LocalDeployStatusPreparadoV0 {
		add(ErrLocalDeployResultadoInvalidoV0, "status")
	}
	if strings.TrimSpace(result.PlanID) == "" {
		add(ErrLocalDeployResultadoInvalidoV0, "plan_id")
	}
	if strings.TrimSpace(result.TargetResuelto) != "local" {
		add(ErrLocalDeployTargetNoLocalV0, "target_resuelto")
	}
	if !localDeployOSExactosV0(result.OSPreparados) {
		add(ErrMatrizOSIncompletaV0, "os_preparados")
	}
	if len(result.Evidencias) == 0 {
		add(ErrLocalDeployResultadoInvalidoV0, "evidencias")
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
		add(ErrLocalDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if algunaAccionRequiereContenedorV0(result.AccionesPrevistas) || algunaAccionRequiereContenedorV0(result.Rollback.PasosPrevistos) {
		add(ErrLocalDeployContenedorNoPuroV0, "acciones_previstas")
	}
	if len(issues) > 0 {
		return LocalDeployAdapterV0Error{Issues: issues}
	}
	return nil
}

func validateLocalDeployPlanV0(plan DeploymentPlanV0) []LocalDeployIssueV0 {
	var issues []LocalDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, LocalDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(plan.DeployTarget) != "local" || strings.TrimSpace(plan.TargetResuelto) != "local" {
		add(ErrLocalDeployTargetNoLocalV0, "deploy_target")
	}
	for _, row := range plan.MatrizOS {
		if !aplicaTrueV0(row.Aplica) {
			add(ErrMatrizOSIncompletaV0, "matriz_os."+strings.TrimSpace(row.OS)+".aplica")
		}
		if strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			add(ErrMatrizOSIncompletaV0, "matriz_os."+strings.TrimSpace(row.OS))
		}
	}
	if !plan.Rollback.Reversible {
		add(ErrLocalDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if algunaAccionRequiereContenedorV0(plan.AccionesPrevistas) || algunaAccionRequiereContenedorV0(plan.Rollback.PasosPrevistos) {
		add(ErrLocalDeployContenedorNoPuroV0, "acciones_previstas")
	}
	return issues
}

func localDeployOSResultsV0(rows []FilaOSV0) []LocalDeployOSV0 {
	results := make([]LocalDeployOSV0, 0, len(rows))
	for _, row := range rows {
		results = append(results, LocalDeployOSV0{
			OS:                strings.TrimSpace(row.OS),
			Status:            LocalDeployStatusPreparadoV0,
			Motivo:            strings.TrimSpace(row.Motivo),
			RequisitosPrevios: copyStringsV0(row.RequisitosPrevios),
		})
	}
	return results
}

func localDeployEvidencesV0() []LocalDeployEvidenceV0 {
	return []LocalDeployEvidenceV0{
		{ID: "deployment_plan_validado", Descripcion: "DeploymentPlanV0 validado antes de preparar el resultado local."},
		{ID: "target_local_confirmado", Descripcion: "Target local confirmado sin seleccionar proveedor ni instalador global."},
		{ID: "dry_run_sin_efectos", Descripcion: "Preparacion declarativa sin materializar artefactos ni cambiar entorno."},
		{ID: "rollback_reversible_confirmado", Descripcion: "Rollback reversible confirmado para el modo dry-run v0."},
	}
}

func localDeployOSExactosV0(rows []LocalDeployOSV0) bool {
	if len(rows) != 3 {
		return false
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Status) != LocalDeployStatusPreparadoV0 || strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			return false
		}
		values = append(values, row.OS)
	}
	return osExactosV0(values)
}

func aplicaTrueV0(value any) bool {
	accepted, ok := value.(bool)
	return ok && accepted
}

func copyRollbackV0(value RollbackV0) RollbackV0 {
	return RollbackV0{
		Reversible:     value.Reversible,
		Motivo:         strings.TrimSpace(value.Motivo),
		PasosPrevistos: copyAccionesPrevistasV0(value.PasosPrevistos),
	}
}

func copyComprobacionesV0(values []ComprobacionV0) []ComprobacionV0 {
	copied := make([]ComprobacionV0, 0, len(values))
	for _, value := range values {
		copied = append(copied, ComprobacionV0{
			ID:          strings.TrimSpace(value.ID),
			Descripcion: strings.TrimSpace(value.Descripcion),
			Obligatoria: value.Obligatoria,
		})
	}
	return copied
}

func copyArtefactosPrevistosV0(values []ArtefactoPrevistoV0) []ArtefactoPrevistoV0 {
	copied := make([]ArtefactoPrevistoV0, 0, len(values))
	for _, value := range values {
		copied = append(copied, ArtefactoPrevistoV0{
			ID:          strings.TrimSpace(value.ID),
			Tipo:        strings.TrimSpace(value.Tipo),
			Descripcion: strings.TrimSpace(value.Descripcion),
			RutaLogica:  strings.TrimSpace(value.RutaLogica),
		})
	}
	return copied
}

func copyAccionesPrevistasV0(values []AccionPrevistaV0) []AccionPrevistaV0 {
	copied := make([]AccionPrevistaV0, 0, len(values))
	for _, value := range values {
		copied = append(copied, AccionPrevistaV0{
			Orden:              value.Orden,
			Tipo:               strings.TrimSpace(value.Tipo),
			Descripcion:        strings.TrimSpace(value.Descripcion),
			RequiereContenedor: value.RequiereContenedor,
		})
	}
	return copied
}

func copyStringsV0(values []string) []string {
	copied := make([]string, 0, len(values))
	for _, value := range values {
		copied = append(copied, strings.TrimSpace(value))
	}
	return copied
}

func HasLocalDeployIssueV0(err error, code string) bool {
	var localErr LocalDeployAdapterV0Error
	if !errors.As(err, &localErr) {
		return false
	}
	for _, issue := range localErr.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
