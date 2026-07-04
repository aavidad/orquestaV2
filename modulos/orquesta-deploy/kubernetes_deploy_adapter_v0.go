package orquestadeploy

import "strings"

const (
	KubernetesDeployAdapterNameV0     = "KubernetesDeployAdapterV0"
	KubernetesDeployResultSchemaV0    = "KubernetesDeployResultV0"
	KubernetesDeployModeDryRunV0      = "dry_run"
	KubernetesDeployStatusPreparadoV0 = "preparado"

	ErrKubernetesDeployTargetInvalidoV0       = "kubernetes_deploy_target_invalido"
	ErrKubernetesDeployResultadoInvalidoV0    = "kubernetes_deploy_resultado_invalido"
	ErrKubernetesDeployRollbackNoReversibleV0 = "kubernetes_deploy_rollback_no_reversible"
	ErrKubernetesDeploySinPublicacionV0       = "kubernetes_deploy_sin_publicacion"
	ErrKubernetesDeployMatrizClienteV0        = "kubernetes_deploy_matriz_cliente_invalida"
)

type KubernetesDeployAdapterV0 struct{}

type KubernetesDeployResultV0 struct {
	SchemaVersion       string                       `json:"schema_version"`
	Adapter             string                       `json:"adapter"`
	Mode                string                       `json:"mode"`
	Status              string                       `json:"status"`
	RequestID           string                       `json:"request_id"`
	PlanID              string                       `json:"plan_id"`
	TargetResuelto      string                       `json:"target_resuelto"`
	OSPreparados        []KubernetesDeployOSV0       `json:"os_preparados"`
	Evidencias          []KubernetesDeployEvidenceV0 `json:"evidencias"`
	ValidacionEntorno   []ComprobacionV0             `json:"validacion_entorno"`
	ArtefactosPrevistos []ArtefactoPrevistoV0        `json:"artefactos_previstos"`
	Healthcheck         []ComprobacionV0             `json:"healthcheck"`
	Rollback            RollbackV0                   `json:"rollback"`
	AccionesPrevistas   []AccionPrevistaV0           `json:"acciones_previstas"`
	Advertencias        []string                     `json:"advertencias"`
}

type KubernetesDeployOSV0 struct {
	OS                string   `json:"os"`
	Status            string   `json:"status"`
	Motivo            string   `json:"motivo"`
	RequisitosPrevios []string `json:"requisitos_previos"`
}

type KubernetesDeployEvidenceV0 struct {
	ID          string `json:"id"`
	Descripcion string `json:"descripcion"`
}

type KubernetesDeployIssueV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type KubernetesDeployAdapterV0Error struct {
	Issues []KubernetesDeployIssueV0 `json:"issues"`
}

func (err KubernetesDeployAdapterV0Error) Error() string {
	if len(err.Issues) == 0 {
		return ErrKubernetesDeployResultadoInvalidoV0
	}
	return err.Issues[0].Code
}

func PrepareKubernetesDeployDryRunV0(plan DeploymentPlanV0) (KubernetesDeployResultV0, error) {
	return KubernetesDeployAdapterV0{}.PrepareDryRun(plan)
}

func (KubernetesDeployAdapterV0) PrepareDryRun(plan DeploymentPlanV0) (KubernetesDeployResultV0, error) {
	if err := ValidateDeploymentPlanV0(plan); err != nil {
		return KubernetesDeployResultV0{}, err
	}
	if issues := validateKubernetesDeployPlanV0(plan); len(issues) > 0 {
		return KubernetesDeployResultV0{}, KubernetesDeployAdapterV0Error{Issues: issues}
	}

	result := KubernetesDeployResultV0{
		SchemaVersion:       KubernetesDeployResultSchemaV0,
		Adapter:             KubernetesDeployAdapterNameV0,
		Mode:                KubernetesDeployModeDryRunV0,
		Status:              KubernetesDeployStatusPreparadoV0,
		RequestID:           strings.TrimSpace(plan.RequestID),
		PlanID:              strings.TrimSpace(plan.PlanID),
		TargetResuelto:      "kubernetes",
		OSPreparados:        kubernetesDeployOSResultsV0(plan.MatrizOS),
		Evidencias:          kubernetesDeployEvidencesV0(),
		ValidacionEntorno:   copyComprobacionesV0(plan.ValidacionEntorno),
		ArtefactosPrevistos: copyArtefactosPrevistosV0(plan.ArtefactosPrevistos),
		Healthcheck:         copyComprobacionesV0(plan.Healthcheck),
		Rollback:            copyRollbackV0(plan.Rollback),
		AccionesPrevistas:   copyAccionesPrevistasV0(plan.AccionesPrevistas),
		Advertencias: append(
			copyStringsV0(plan.Advertencias),
			"Dry-run kubernetes: no se escribieron manifests, no se leyo kubeconfig y no se contacto cluster alguno.",
		),
	}
	if err := ValidateKubernetesDeployResultV0(result); err != nil {
		return KubernetesDeployResultV0{}, err
	}
	return result, nil
}

func ValidateKubernetesDeployResultV0(result KubernetesDeployResultV0) error {
	var issues []KubernetesDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, KubernetesDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(result.SchemaVersion) != KubernetesDeployResultSchemaV0 {
		add(ErrKubernetesDeployResultadoInvalidoV0, "schema_version")
	}
	if strings.TrimSpace(result.Adapter) != KubernetesDeployAdapterNameV0 {
		add(ErrKubernetesDeployResultadoInvalidoV0, "adapter")
	}
	if strings.TrimSpace(result.Mode) != KubernetesDeployModeDryRunV0 {
		add(ErrKubernetesDeployResultadoInvalidoV0, "mode")
	}
	if strings.TrimSpace(result.Status) != KubernetesDeployStatusPreparadoV0 {
		add(ErrKubernetesDeployResultadoInvalidoV0, "status")
	}
	if strings.TrimSpace(result.PlanID) == "" {
		add(ErrKubernetesDeployResultadoInvalidoV0, "plan_id")
	}
	if strings.TrimSpace(result.TargetResuelto) != "kubernetes" {
		add(ErrKubernetesDeployTargetInvalidoV0, "target_resuelto")
	}
	if !kubernetesDeployOSExactosV0(result.OSPreparados) {
		add(ErrMatrizOSIncompletaV0, "os_preparados")
	}
	if len(result.Evidencias) == 0 {
		add(ErrKubernetesDeployResultadoInvalidoV0, "evidencias")
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
		add(ErrKubernetesDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !algunaAccionTipoV0(result.AccionesPrevistas, "publicar_declarativo") {
		add(ErrKubernetesDeploySinPublicacionV0, "acciones_previstas")
	}
	if !kubernetesMatrixHasClientRowsV0(result.OSPreparados) {
		add(ErrKubernetesDeployMatrizClienteV0, "os_preparados")
	}
	if len(issues) > 0 {
		return KubernetesDeployAdapterV0Error{Issues: issues}
	}
	return nil
}

func validateKubernetesDeployPlanV0(plan DeploymentPlanV0) []KubernetesDeployIssueV0 {
	var issues []KubernetesDeployIssueV0
	add := func(code, field string) {
		issues = append(issues, KubernetesDeployIssueV0{Code: code, Field: field})
	}

	if strings.TrimSpace(plan.DeployTarget) != "kubernetes" || strings.TrimSpace(plan.TargetResuelto) != "kubernetes" {
		add(ErrKubernetesDeployTargetInvalidoV0, "deploy_target")
	}
	for _, row := range plan.MatrizOS {
		if strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			add(ErrMatrizOSIncompletaV0, "matriz_os."+strings.TrimSpace(row.OS))
		}
	}
	if !plan.Rollback.Reversible {
		add(ErrKubernetesDeployRollbackNoReversibleV0, "rollback.reversible")
	}
	if !algunaAccionTipoV0(plan.AccionesPrevistas, "publicar_declarativo") {
		add(ErrKubernetesDeploySinPublicacionV0, "acciones_previstas")
	}
	if !kubernetesMatrixHasClientRowsV0(kubernetesDeployOSResultsV0(plan.MatrizOS)) {
		add(ErrKubernetesDeployMatrizClienteV0, "matriz_os")
	}
	return issues
}

func kubernetesDeployOSResultsV0(rows []FilaOSV0) []KubernetesDeployOSV0 {
	results := make([]KubernetesDeployOSV0, 0, len(rows))
	for _, row := range rows {
		results = append(results, KubernetesDeployOSV0{
			OS:                strings.TrimSpace(row.OS),
			Status:            KubernetesDeployStatusPreparadoV0,
			Motivo:            strings.TrimSpace(row.Motivo),
			RequisitosPrevios: copyStringsV0(row.RequisitosPrevios),
		})
	}
	return results
}

func kubernetesDeployEvidencesV0() []KubernetesDeployEvidenceV0 {
	return []KubernetesDeployEvidenceV0{
		{ID: "deployment_plan_validado", Descripcion: "DeploymentPlanV0 validado antes de preparar el resultado kubernetes."},
		{ID: "target_kubernetes_confirmado", Descripcion: "Target kubernetes confirmado sin elegir proveedor, cluster ni namespace reales."},
		{ID: "publicacion_declarativa_presente", Descripcion: "El plan conserva al menos una accion declarativa de publicacion sin aplicar recursos reales."},
		{ID: "dry_run_sin_cluster", Descripcion: "Preparacion declarativa sin kubeconfig, cliente real, manifests ni acceso a cluster."},
	}
}

func kubernetesDeployOSExactosV0(rows []KubernetesDeployOSV0) bool {
	if len(rows) != 3 {
		return false
	}
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Status) != KubernetesDeployStatusPreparadoV0 || strings.TrimSpace(row.Motivo) == "" || len(row.RequisitosPrevios) == 0 {
			return false
		}
		values = append(values, row.OS)
	}
	return osExactosV0(values)
}

func kubernetesMatrixHasClientRowsV0(rows []KubernetesDeployOSV0) bool {
	clientRows := 0
	for _, row := range rows {
		if strings.Contains(strings.ToLower(strings.TrimSpace(row.Motivo)), "cliente") {
			clientRows++
		}
	}
	return clientRows >= 2
}

func algunaAccionTipoV0(actions []AccionPrevistaV0, want string) bool {
	for _, action := range actions {
		if strings.TrimSpace(action.Tipo) == want {
			return true
		}
	}
	return false
}
