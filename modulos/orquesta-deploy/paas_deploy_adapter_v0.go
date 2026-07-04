package orquestadeploy

import "strings"

const (
	PaaSDeployAdapterNameV0     = "PaaSDeployAdapterV0"
	PaaSDeployResultSchemaV0    = "PaaSDeployResultV0"
	PaaSDeployModeDryRunV0      = "dry_run"
	PaaSDeployStatusPreparadoV0 = "preparado"

	ErrPaaSDeployTargetInvalidoV0       = "paas_deploy_target_invalido"
	ErrPaaSDeployResultadoInvalidoV0    = "paas_deploy_resultado_invalido"
	ErrPaaSDeployRollbackNoReversibleV0 = "paas_deploy_rollback_no_reversible"
	ErrPaaSDeployProveedorNoDeclaradoV0 = "paas_deploy_proveedor_no_declarado"
	ErrPaaSDeploySinPublicacionV0       = "paas_deploy_sin_publicacion"
	ErrPaaSDeployMatrizClienteV0        = "paas_deploy_matriz_cliente_invalida"
	ErrPaaSDeployContenedorNoPuroV0     = "paas_deploy_contenedor_no_puro"
)

type PaaSDeployAdapterV0 struct{}

type PaaSDeployResultV0 struct {
	SchemaVersion       string                 `json:"schema_version"`
	Adapter             string                 `json:"adapter"`
	Mode                string                 `json:"mode"`
	Status              string                 `json:"status"`
	RequestID           string                 `json:"request_id"`
	PlanID              string                 `json:"plan_id"`
	TargetResuelto      string                 `json:"target_resuelto"`
	OSPreparados        []PaaSDeployOSV0       `json:"os_preparados"`
	Evidencias          []PaaSDeployEvidenceV0 `json:"evidencias"`
	ValidacionEntorno   []ComprobacionV0       `json:"validacion_entorno"`
	ArtefactosPrevistos []ArtefactoPrevistoV0  `json:"artefactos_previstos"`
	Healthcheck         []ComprobacionV0       `json:"healthcheck"`
	Rollback            RollbackV0             `json:"rollback"`
	AccionesPrevistas   []AccionPrevistaV0     `json:"acciones_previstas"`
	Advertencias        []string               `json:"advertencias"`
}

type PaaSDeployOSV0 struct {
	OS                string   `json:"os"`
	Status            string   `json:"status"`
	Motivo            string   `json:"motivo"`
	RequisitosPrevios []string `json:"requisitos_previos"`
}

type PaaSDeployEvidenceV0 struct {
	ID          string `json:"id"`
	Descripcion string `json:"descripcion"`
}

type PaaSDeployIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type PaaSDeployAdapterV0Error struct {
	Issues []PaaSDeployIssueV0 `json:"issues"`
}

func (err PaaSDeployAdapterV0Error) Error() string {
	if len(err.Issues) == 0 {
		return ErrPaaSDeployResultadoInvalidoV0
	}
	return err.Issues[0].Code
}

func PreparePaaSDeployDryRunV0(plan DeploymentPlanV0) (PaaSDeployResultV0, error) {
	return PaaSDeployAdapterV0{}.PrepareDryRun(plan)
}

func (PaaSDeployAdapterV0) PrepareDryRun(plan DeploymentPlanV0) (PaaSDeployResultV0, error) {
	if err := ValidateDeploymentPlanV0(plan); err != nil {
		return PaaSDeployResultV0{}, err
	}
	if issues := validatePaaSDeployPlanV0(plan); len(issues) > 0 {
		return PaaSDeployResultV0{}, PaaSDeployAdapterV0Error{Issues: issues}
	}

	result := PaaSDeployResultV0{
		SchemaVersion:       PaaSDeployResultSchemaV0,
		Adapter:             PaaSDeployAdapterNameV0,
		Mode:                PaaSDeployModeDryRunV0,
		Status:              PaaSDeployStatusPreparadoV0,
		RequestID:           strings.TrimSpace(plan.RequestID),
		PlanID:              strings.TrimSpace(plan.PlanID),
		TargetResuelto:      "paas",
		OSPreparados:        paasDeployOSResultsV0(plan.MatrizOS),
		Evidencias:          paasDeployEvidencesV0(),
		ValidacionEntorno:   copyComprobacionesV0(plan.ValidacionEntorno),
		ArtefactosPrevistos: copyArtefactosPrevistosV0(plan.ArtefactosPrevistos),
		Healthcheck:         copyComprobacionesV0(plan.Healthcheck),
		Rollback:            copyRollbackV0(plan.Rollback),
		AccionesPrevistas:   copyAccionesPrevistasV0(plan.AccionesPrevistas),
		Advertencias: append(
			copyStringsV0(plan.Advertencias),
			"Dry-run paas: no se eligio proveedor, runtime, region ni cuenta real.",
		),
	}
	if err := ValidatePaaSDeployResultV0(result); err != nil {
		return PaaSDeployResultV0{}, err
	}
	return result, nil
}

func ValidatePaaSDeployResultV0(result PaaSDeployResultV0) error {
	var issues []PaaSDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, PaaSDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(result.SchemaVersion) != PaaSDeployResultSchemaV0 {
		add(ErrPaaSDeployResultadoInvalidoV0, "schema_version")
	}
	if strings.TrimSpace(result.Adapter) != PaaSDeployAdapterNameV0 {
		add(ErrPaaSDeployResultadoInvalidoV0, "adapter")
	}
	if strings.TrimSpace(result.Mode) != PaaSDeployModeDryRunV0 {
		add(ErrPaaSDeployResultadoInvalidoV0, "mode")
	}
	if strings.TrimSpace(result.Status) != PaaSDeployStatusPreparadoV0 {
		add(ErrPaaSDeployResultadoInvalidoV0, "status")
	}
	if strings.TrimSpace(result.PlanID) == "" {
		add(ErrPaaSDeployResultadoInvalidoV0, "plan_id")
	}
	if strings.TrimSpace(result.TargetResuelto) != "paas" {
		add(ErrPaaSDeployTargetInvalidoV0, "target_resuelto")
	}
	if !paasDeployOSExactosV0(result.OSPreparados) || !paasMatrixHasClientRowsV0(result.OSPreparados) {
		add(ErrPaaSDeployMatrizClienteV0, "os_preparados")
	}
	if len(result.Evidencias) == 0 {
		add(ErrPaaSDeployResultadoInvalidoV0, "evidencias")
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
		add(ErrPaaSDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !algunaAccionTipoV0(result.AccionesPrevistas, "publicar_declarativo") {
		add(ErrPaaSDeploySinPublicacionV0, "acciones_previstas")
	}
	if algunaAccionRequiereContenedorV0(result.AccionesPrevistas) || algunaAccionRequiereContenedorV0(result.Rollback.PasosPrevistos) {
		add(ErrPaaSDeployContenedorNoPuroV0, "acciones_previstas")
	}
	if len(issues) > 0 {
		return PaaSDeployAdapterV0Error{Issues: issues}
	}
	return nil
}

func validatePaaSDeployPlanV0(plan DeploymentPlanV0) []PaaSDeployIssueV0 {
	var issues []PaaSDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, PaaSDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(plan.DeployTarget) != "paas" || strings.TrimSpace(plan.TargetResuelto) != "paas" {
		add(ErrPaaSDeployTargetInvalidoV0, "deploy_target")
	}
	for _, row := range plan.MatrizOS {
		if strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			add(ErrMatrizOSIncompletaV0, "matriz_os."+strings.TrimSpace(row.OS))
		}
	}
	if !paasMatrixHasClientRowsV0(paasDeployOSResultsV0(plan.MatrizOS)) {
		add(ErrPaaSDeployMatrizClienteV0, "matriz_os")
	}
	if !plan.Rollback.Reversible {
		add(ErrPaaSDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !paasHasProviderRestrictionV0(plan.Restricciones) {
		add(ErrPaaSDeployProveedorNoDeclaradoV0, "restricciones")
	}
	if !algunaAccionTipoV0(plan.AccionesPrevistas, "publicar_declarativo") {
		add(ErrPaaSDeploySinPublicacionV0, "acciones_previstas")
	}
	if algunaAccionRequiereContenedorV0(plan.AccionesPrevistas) || algunaAccionRequiereContenedorV0(plan.Rollback.PasosPrevistos) {
		add(ErrPaaSDeployContenedorNoPuroV0, "acciones_previstas")
	}
	return issues
}

func paasDeployOSResultsV0(rows []FilaOSV0) []PaaSDeployOSV0 {
	results := make([]PaaSDeployOSV0, 0, len(rows))
	for _, row := range rows {
		results = append(results, PaaSDeployOSV0{
			OS:                strings.TrimSpace(row.OS),
			Status:            PaaSDeployStatusPreparadoV0,
			Motivo:            strings.TrimSpace(row.Motivo),
			RequisitosPrevios: copyStringsV0(row.RequisitosPrevios),
		})
	}
	return results
}

func paasDeployEvidencesV0() []PaaSDeployEvidenceV0 {
	return []PaaSDeployEvidenceV0{
		{ID: "deployment_plan_validado", Descripcion: "DeploymentPlanV0 validado antes de preparar el resultado paas."},
		{ID: "target_paas_confirmado", Descripcion: "Target paas confirmado sin elegir proveedor, cuenta, region ni runtime gestionado."},
		{ID: "proveedor_opaco_declarado", Descripcion: "El plan conserva una restriccion opaca de proveedor pendiente o externo."},
		{ID: "dry_run_sin_efectos", Descripcion: "Preparacion declarativa sin SDK, IaC, credenciales ni despliegue real."},
	}
}

func paasDeployOSExactosV0(rows []PaaSDeployOSV0) bool {
	if len(rows) != 3 {
		return false
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Status) != PaaSDeployStatusPreparadoV0 || strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			return false
		}
		values = append(values, row.OS)
	}
	return osExactosV0(values)
}

func paasMatrixHasClientRowsV0(rows []PaaSDeployOSV0) bool {
	clientRows := 0
	for _, row := range rows {
		if strings.TrimSpace(row.OS) != "linux" && strings.Contains(strings.ToLower(strings.TrimSpace(row.Motivo)), "cliente") {
			clientRows++
		}
	}
	return clientRows >= 2
}

func paasHasProviderRestrictionV0(restrictions []RestriccionV0) bool {
	for _, restriction := range restrictions {
		key := strings.ToLower(strings.TrimSpace(restriction.Clave))
		value := strings.ToLower(strings.TrimSpace(restriction.Valor))
		if strings.Contains(key, "plataforma") && (strings.Contains(value, "opaco") || strings.Contains(value, "pendiente")) {
			return true
		}
	}
	return false
}
