package orquestadeploy

import (
	"strconv"
	"strings"
)

func deploymentPlanRestrictionsV0(target string, values []string) []RestriccionV0 {
	out := make([]RestriccionV0, 0, len(values)+1)
	for index, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, RestriccionV0{Clave: "restriccion_" + deploymentIndexV0(index+1), Valor: trimmed})
		}
	}
	if target == "paas" {
		out = append(out, RestriccionV0{Clave: "plataforma", Valor: "proveedor_opaco_pendiente"})
	}
	return out
}

func deploymentPlanOSMatrixV0(target string) []FilaOSV0 {
	return []FilaOSV0{
		deploymentPlanOSRowV0("linux", target),
		deploymentPlanOSRowV0("darwin", target),
		deploymentPlanOSRowV0("windows", target),
	}
}

func deploymentPlanOSRowV0(osName string, target string) FilaOSV0 {
	motivo := "cliente dry-run disponible para validar artefactos declarativos"
	if target == "desktop" {
		motivo = "aplica para empaquetado desktop declarativo"
	}
	if target == "mobile_store" && osName == "darwin" {
		motivo = "aplica para metadatos y revision mobile_store declarativa"
	}
	if target == "local" {
		motivo = "cliente local dry-run sin tocar filesystem productivo"
	}
	return FilaOSV0{
		OS:                osName,
		Aplica:            true,
		Motivo:            motivo,
		RequisitosPrevios: []string{"contrato DeploymentPlanV0 validado", "dry-run sin secretos"},
	}
}

func deploymentPlanChecksV0(prefix string) []ComprobacionV0 {
	return []ComprobacionV0{
		{ID: prefix + "_contrato", Descripcion: "Validar DeploymentPlanV0 y refs opacas.", Obligatoria: true},
		{ID: prefix + "_sin_efectos", Descripcion: "Confirmar que el dry-run no ejecuta efectos externos.", Obligatoria: true},
	}
}

func deploymentPlanArtifactsV0(target string) []ArtefactoPrevistoV0 {
	artifacts := []ArtefactoPrevistoV0{
		{ID: "deployment_plan", Tipo: "deployment_plan", Descripcion: "Plan declarativo de deploy.", RutaLogica: "deploy/" + target + "/deployment_plan"},
		{ID: "dry_run_evidence", Tipo: "dry_run_evidence", Descripcion: "Evidencia compacta de dry-run.", RutaLogica: "deploy/" + target + "/dry_run_evidence"},
	}
	switch target {
	case "desktop":
		artifacts = append(artifacts, ArtefactoPrevistoV0{ID: "desktop_package", Tipo: "paquete_desktop", Descripcion: "Paquete desktop previsto sin construir.", RutaLogica: "deploy/desktop/package"})
	case "mobile_store":
		artifacts = append(artifacts, ArtefactoPrevistoV0{ID: "mobile_metadata", Tipo: "metadata_publicacion", Descripcion: "Metadata de tienda prevista sin publicar.", RutaLogica: "deploy/mobile_store/metadata"})
	}
	return artifacts
}

func deploymentPlanRollbackV0(target string) RollbackV0 {
	return RollbackV0{
		Reversible: true,
		Motivo:     "Dry-run reversible: no se aplica despliegue real.",
		PasosPrevistos: []AccionPrevistaV0{
			{Orden: 10, Tipo: "registrar_rollback", Descripcion: "Registrar plan de rollback declarativo para " + target + ".", RequiereContenedor: false},
		},
	}
}

func deploymentPlanActionsV0(target string) []AccionPrevistaV0 {
	actions := []AccionPrevistaV0{
		{Orden: 10, Tipo: "validar_entorno", Descripcion: "Validar entorno declarado para " + target + ".", RequiereContenedor: false},
		{Orden: 20, Tipo: "preparar_artefactos", Descripcion: "Preparar artefactos logicos sin escribir archivos reales.", RequiereContenedor: target == "contenedor"},
		{Orden: 30, Tipo: "publicar_declarativo", Descripcion: "Simular publicacion declarativa sin proveedor externo.", RequiereContenedor: target == "contenedor"},
		{Orden: 40, Tipo: "verificar_salud", Descripcion: "Verificar healthcheck declarativo.", RequiereContenedor: false},
	}
	if target == "mobile_store" {
		actions = append(actions, AccionPrevistaV0{Orden: 50, Tipo: "solicitar_decision", Descripcion: "Pedir decision externa antes de publicar en tienda.", RequiereContenedor: false})
	}
	return actions
}

func deploymentPlanRefV0(appSpecID string, target string) string {
	base := deploymentSlugV0(firstDeploymentValueV0(appSpecID, "app-spec"))
	return base + "-" + deploymentSlugV0(target) + "-dry-run"
}

func deploymentSlugV0(value string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			continue
		}
		if r == '-' || r == '_' {
			out.WriteRune('-')
		}
	}
	if out.Len() == 0 {
		return "ref"
	}
	return out.String()
}

func firstDeploymentValueV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" && trimmed != "sin_preferencia" {
			return trimmed
		}
	}
	return ""
}

func deploymentIndexV0(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
