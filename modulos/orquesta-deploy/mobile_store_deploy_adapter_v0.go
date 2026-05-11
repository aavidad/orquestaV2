package orquestadeploy

import (
	"errors"
	"strings"
)

const (
	MobileStoreDeployAdapterNameV0     = "MobileStoreDeployAdapterV0"
	MobileStoreDeployResultSchemaV0    = "MobileStoreDeployResultV0"
	MobileStoreDeployModeDryRunV0      = "dry_run"
	MobileStoreDeployStatusPreparadoV0 = "preparado"

	ErrMobileStoreDeployTargetInvalidoV0       = "mobile_store_deploy_target_invalido"
	ErrMobileStoreDeployResultadoInvalidoV0    = "mobile_store_deploy_resultado_invalido"
	ErrMobileStoreDeployRollbackNoReversibleV0 = "mobile_store_deploy_rollback_no_reversible"
	ErrMobileStoreDeployMatrizInvalidaV0       = "mobile_store_deploy_matriz_invalida"
	ErrMobileStoreDeploySinMetadataV0          = "mobile_store_deploy_sin_metadata"
	ErrMobileStoreDeploySinDecisionV0          = "mobile_store_deploy_sin_decision"
	ErrMobileStoreDeployContenedorNoPuroV0     = "mobile_store_deploy_contenedor_no_puro"
)

type MobileStoreDeployAdapterV0 struct{}

type MobileStoreDeployResultV0 struct {
	SchemaVersion       string                        `json:"schema_version"`
	Adapter             string                        `json:"adapter"`
	Mode                string                        `json:"mode"`
	Status              string                        `json:"status"`
	RequestID           string                        `json:"request_id"`
	PlanID              string                        `json:"plan_id"`
	TargetResuelto      string                        `json:"target_resuelto"`
	OSPreparados        []MobileStoreDeployOSV0       `json:"os_preparados"`
	Evidencias          []MobileStoreDeployEvidenceV0 `json:"evidencias"`
	ValidacionEntorno   []ComprobacionV0              `json:"validacion_entorno"`
	ArtefactosPrevistos []ArtefactoPrevistoV0         `json:"artefactos_previstos"`
	Healthcheck         []ComprobacionV0              `json:"healthcheck"`
	Rollback            RollbackV0                    `json:"rollback"`
	AccionesPrevistas   []AccionPrevistaV0            `json:"acciones_previstas"`
	Advertencias        []string                      `json:"advertencias"`
}

type MobileStoreDeployOSV0 struct {
	OS                string   `json:"os"`
	Status            string   `json:"status"`
	Motivo            string   `json:"motivo"`
	RequisitosPrevios []string `json:"requisitos_previos"`
}

type MobileStoreDeployEvidenceV0 struct {
	ID          string `json:"id"`
	Descripcion string `json:"descripcion"`
}

type MobileStoreDeployIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type MobileStoreDeployAdapterV0Error struct {
	Issues []MobileStoreDeployIssueV0 `json:"issues"`
}

func (err MobileStoreDeployAdapterV0Error) Error() string {
	if len(err.Issues) == 0 {
		return ErrMobileStoreDeployResultadoInvalidoV0
	}
	return err.Issues[0].Code
}

func PrepareMobileStoreDeployDryRunV0(plan DeploymentPlanV0) (MobileStoreDeployResultV0, error) {
	return MobileStoreDeployAdapterV0{}.PrepareDryRun(plan)
}

func (MobileStoreDeployAdapterV0) PrepareDryRun(plan DeploymentPlanV0) (MobileStoreDeployResultV0, error) {
	if err := ValidateDeploymentPlanV0(plan); err != nil {
		return MobileStoreDeployResultV0{}, err
	}
	if issues := validateMobileStoreDeployPlanV0(plan); len(issues) > 0 {
		return MobileStoreDeployResultV0{}, MobileStoreDeployAdapterV0Error{Issues: issues}
	}

	result := MobileStoreDeployResultV0{
		SchemaVersion:       MobileStoreDeployResultSchemaV0,
		Adapter:             MobileStoreDeployAdapterNameV0,
		Mode:                MobileStoreDeployModeDryRunV0,
		Status:              MobileStoreDeployStatusPreparadoV0,
		RequestID:           strings.TrimSpace(plan.RequestID),
		PlanID:              strings.TrimSpace(plan.PlanID),
		TargetResuelto:      "mobile_store",
		OSPreparados:        mobileStoreDeployOSResultsV0(plan.MatrizOS),
		Evidencias:          mobileStoreDeployEvidencesV0(),
		ValidacionEntorno:   copyComprobacionesV0(plan.ValidacionEntorno),
		ArtefactosPrevistos: copyArtefactosPrevistosV0(plan.ArtefactosPrevistos),
		Healthcheck:         copyComprobacionesV0(plan.Healthcheck),
		Rollback:            copyRollbackV0(plan.Rollback),
		AccionesPrevistas:   copyAccionesPrevistasV0(plan.AccionesPrevistas),
		Advertencias: append(
			copyStringsV0(plan.Advertencias),
			"Dry-run mobile_store: no se firmo, no se notarizo y no se subio paquete a ninguna tienda.",
		),
	}
	if err := ValidateMobileStoreDeployResultV0(result); err != nil {
		return MobileStoreDeployResultV0{}, err
	}
	return result, nil
}

func ValidateMobileStoreDeployResultV0(result MobileStoreDeployResultV0) error {
	var issues []MobileStoreDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, MobileStoreDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(result.SchemaVersion) != MobileStoreDeployResultSchemaV0 {
		add(ErrMobileStoreDeployResultadoInvalidoV0, "schema_version")
	}
	if strings.TrimSpace(result.Adapter) != MobileStoreDeployAdapterNameV0 {
		add(ErrMobileStoreDeployResultadoInvalidoV0, "adapter")
	}
	if strings.TrimSpace(result.Mode) != MobileStoreDeployModeDryRunV0 {
		add(ErrMobileStoreDeployResultadoInvalidoV0, "mode")
	}
	if strings.TrimSpace(result.Status) != MobileStoreDeployStatusPreparadoV0 {
		add(ErrMobileStoreDeployResultadoInvalidoV0, "status")
	}
	if strings.TrimSpace(result.PlanID) == "" {
		add(ErrMobileStoreDeployResultadoInvalidoV0, "plan_id")
	}
	if strings.TrimSpace(result.TargetResuelto) != "mobile_store" {
		add(ErrMobileStoreDeployTargetInvalidoV0, "target_resuelto")
	}
	if !mobileStoreDeployOSExactosV0(result.OSPreparados) || !mobileStoreHasDarwinApplicableV0(result.OSPreparados) || !mobileStoreHasClientRowsV0(result.OSPreparados) {
		add(ErrMobileStoreDeployMatrizInvalidaV0, "os_preparados")
	}
	if len(result.Evidencias) == 0 {
		add(ErrMobileStoreDeployResultadoInvalidoV0, "evidencias")
	}
	if len(result.ValidacionEntorno) == 0 {
		add(ErrValidacionEntornoIncompletaV0, "validacion_entorno")
	}
	if len(result.ArtefactosPrevistos) == 0 || !mobileStoreHasMetadataArtifactV0(result.ArtefactosPrevistos) {
		add(ErrMobileStoreDeploySinMetadataV0, "artefactos_previstos")
	}
	if len(result.Healthcheck) == 0 {
		add(ErrHealthcheckIncompletoV0, "healthcheck")
	}
	if !result.Rollback.Reversible {
		add(ErrMobileStoreDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !algunaAccionTipoV0(result.AccionesPrevistas, "solicitar_decision") {
		add(ErrMobileStoreDeploySinDecisionV0, "acciones_previstas")
	}
	if algunaAccionRequiereContenedorV0(result.AccionesPrevistas) || algunaAccionRequiereContenedorV0(result.Rollback.PasosPrevistos) {
		add(ErrMobileStoreDeployContenedorNoPuroV0, "acciones_previstas")
	}
	if len(issues) > 0 {
		return MobileStoreDeployAdapterV0Error{Issues: issues}
	}
	return nil
}

func validateMobileStoreDeployPlanV0(plan DeploymentPlanV0) []MobileStoreDeployIssueV0 {
	var issues []MobileStoreDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, MobileStoreDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(plan.DeployTarget) != "mobile_store" || strings.TrimSpace(plan.TargetResuelto) != "mobile_store" {
		add(ErrMobileStoreDeployTargetInvalidoV0, "deploy_target")
	}
	rows := mobileStoreDeployOSResultsV0(plan.MatrizOS)
	if !mobileStoreHasDarwinApplicableV0(rows) || !mobileStoreHasClientRowsV0(rows) {
		add(ErrMobileStoreDeployMatrizInvalidaV0, "matriz_os")
	}
	for _, row := range plan.MatrizOS {
		if strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			add(ErrMatrizOSIncompletaV0, "matriz_os."+strings.TrimSpace(row.OS))
		}
	}
	if !plan.Rollback.Reversible {
		add(ErrMobileStoreDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !mobileStoreHasMetadataArtifactV0(plan.ArtefactosPrevistos) {
		add(ErrMobileStoreDeploySinMetadataV0, "artefactos_previstos")
	}
	if !algunaAccionTipoV0(plan.AccionesPrevistas, "solicitar_decision") {
		add(ErrMobileStoreDeploySinDecisionV0, "acciones_previstas")
	}
	if algunaAccionRequiereContenedorV0(plan.AccionesPrevistas) || algunaAccionRequiereContenedorV0(plan.Rollback.PasosPrevistos) {
		add(ErrMobileStoreDeployContenedorNoPuroV0, "acciones_previstas")
	}
	return issues
}

func mobileStoreDeployOSResultsV0(rows []FilaOSV0) []MobileStoreDeployOSV0 {
	results := make([]MobileStoreDeployOSV0, 0, len(rows))
	for _, row := range rows {
		results = append(results, MobileStoreDeployOSV0{
			OS:                strings.TrimSpace(row.OS),
			Status:            MobileStoreDeployStatusPreparadoV0,
			Motivo:            strings.TrimSpace(row.Motivo),
			RequisitosPrevios: copyStringsV0(row.RequisitosPrevios),
		})
	}
	return results
}

func mobileStoreDeployEvidencesV0() []MobileStoreDeployEvidenceV0 {
	return []MobileStoreDeployEvidenceV0{
		{ID: "deployment_plan_validado", Descripcion: "DeploymentPlanV0 validado antes de preparar el resultado mobile_store."},
		{ID: "target_mobile_store_confirmado", Descripcion: "Target mobile_store confirmado sin elegir tienda concreta ni proveedor externo."},
		{ID: "metadata_publicacion_presente", Descripcion: "El plan declara metadata de publicacion y artefactos previstos sin firmar ni subir paquetes."},
		{ID: "dry_run_sin_efectos", Descripcion: "Preparacion declarativa sin notarizacion, sin credenciales y sin integracion real de tienda."},
	}
}

func mobileStoreDeployOSExactosV0(rows []MobileStoreDeployOSV0) bool {
	if len(rows) != 3 {
		return false
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Status) != MobileStoreDeployStatusPreparadoV0 || strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			return false
		}
		values = append(values, row.OS)
	}
	return osExactosV0(values)
}

func mobileStoreHasDarwinApplicableV0(rows []MobileStoreDeployOSV0) bool {
	for _, row := range rows {
		if strings.TrimSpace(row.OS) == "darwin" && strings.Contains(strings.ToLower(strings.TrimSpace(row.Motivo)), "aplica") {
			return true
		}
	}
	return false
}

func mobileStoreHasClientRowsV0(rows []MobileStoreDeployOSV0) bool {
	clientRows := 0
	for _, row := range rows {
		if strings.TrimSpace(row.OS) != "darwin" && strings.Contains(strings.ToLower(strings.TrimSpace(row.Motivo)), "cliente") {
			clientRows++
		}
	}
	return clientRows >= 2
}

func mobileStoreHasMetadataArtifactV0(artifacts []ArtefactoPrevistoV0) bool {
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Tipo) == "metadata_publicacion" {
			return true
		}
	}
	return false
}

func HasMobileStoreDeployIssueV0(err error, code string) bool {
	var mobileErr MobileStoreDeployAdapterV0Error
	if !errors.As(err, &mobileErr) {
		return false
	}
	for _, issue := range mobileErr.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
