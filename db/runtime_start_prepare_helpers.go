package db

import (
	"fmt"
	"orquesta/runtimeagente"
	"strings"
)

func connectorConfigFromDBConector(conector *Conector) runtimeagente.ConnectorConfig {
	if conector == nil {
		return runtimeagente.ConnectorConfig{}
	}
	return runtimeagente.ConnectorConfig{
		Slug:         strings.TrimSpace(conector.Slug),
		Nombre:       strings.TrimSpace(conector.Nombre),
		Transporte:   strings.TrimSpace(conector.Transporte),
		Comando:      strings.TrimSpace(conector.Comando),
		ArgsJSON:     strings.TrimSpace(conector.ArgsJSON),
		EnvJSON:      strings.TrimSpace(conector.EnvJSON),
		MetadataJSON: strings.TrimSpace(conector.MetadataJSON),
		Activo:       conector.Activo,
	}
}

func prepararResumeStartRuntime(agenteRef string, proyecto *Proyecto, conector *Conector, resume runtimeagente.ResumeContext, perfilTarea, modelo, razonamiento string, skipBootstrap bool, excludeOrderID int64) (string, runtimeagente.ResumeContext, *bootstrapRuntimeData, error) {
	var bootstrap *bootstrapRuntimeData
	if canonical, err := CanonicalizeAgentName(agenteRef); err == nil {
		agenteRef = canonical
	}
	if skipBootstrap {
		resume = runtimeagente.ResumeContext{
			Branch: strings.TrimSpace(resume.Branch),
			CWD:    strings.TrimSpace(resume.CWD),
		}
		resume = SanitizeResumeContextForProject(resume, proyecto)
		resume.CWD = RutaTrabajoPreferidaAgenteProyecto(strings.TrimSpace(agenteRef), proyecto, strings.TrimSpace(resume.CWD))
		if strings.TrimSpace(resume.CWD) == "" {
			resume.CWD = strings.TrimSpace(proyecto.RutaAbs)
		}
	} else {
		var err error
		resume, bootstrap, err = prepararResumeBootstrapRuntime(strings.TrimSpace(agenteRef), proyecto, resume, excludeOrderID)
		if err != nil {
			return "", runtimeagente.ResumeContext{}, nil, err
		}
	}
	resume.ResumePayloadJSON = MergeResumePayloadPerfilEjecucion(
		resume.ResumePayloadJSON,
		strings.TrimSpace(perfilTarea),
		strings.TrimSpace(modelo),
		strings.TrimSpace(razonamiento),
	)
	resume = runtimeagente.SanitizarResumeParaConector(connectorConfigFromDBConector(conector), resume)
	if strings.TrimSpace(resume.ExternalSessionID) == "" &&
		strings.TrimSpace(resume.ResumenContinuidad) == "" &&
		strings.TrimSpace(resume.ResumePayloadJSON) == "" &&
		(strings.TrimSpace(perfilTarea) != "" || strings.TrimSpace(modelo) != "" || strings.TrimSpace(razonamiento) != "") {
		resume.ResumePayloadJSON = MergeResumePayloadPerfilEjecucion(
			"",
			strings.TrimSpace(perfilTarea),
			strings.TrimSpace(modelo),
			strings.TrimSpace(razonamiento),
		)
	}
	return strings.TrimSpace(agenteRef), resume, bootstrap, nil
}

func construirLaunchPlanStartRuntime(agente *Agente, proyecto *Proyecto, conector *Conector, agenteRef string, perfilTarea, modelo, razonamiento string, resume runtimeagente.ResumeContext, tareaID *int64) (*runtimeagente.LaunchPlan, error) {
	conectorRuntime := connectorConfigFromDBConector(conector)
	plan, err := runtimeagente.DefaultRegistry().Prepare(runtimeagente.LaunchRequest{
		Agente:       nombreAgenteRuntime(strings.TrimSpace(agenteRef), strings.TrimSpace(agente.Nombre)),
		Rol:          strings.TrimSpace(agente.Rol),
		ProyectoSlug: strings.TrimSpace(proyecto.Slug),
		ProyectoRuta: strings.TrimSpace(proyecto.RutaAbs),
		Modelo:       strings.TrimSpace(modelo),
		Razonamiento: strings.TrimSpace(razonamiento),
		PerfilTarea:  strings.TrimSpace(perfilTarea),
		Conector:     conectorRuntime,
		Resume:       resume,
		TareaID:      tareaID,
	})
	if err != nil {
		return nil, err
	}
	plan.BootstrapPrompt, err = BuildLaunchBootstrapPromptForContext(agente, proyecto, plan)
	if err != nil {
		return nil, err
	}
	metadataJSONNormalizada, err := runtimeagente.NormalizeConnectorMetadataJSON(conectorRuntime)
	if err != nil {
		return nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
	}
	if err := runtimeagente.ApplyLaunchPromptMetadata(plan, metadataJSONNormalizada); err != nil {
		return nil, fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
	}
	return plan, nil
}
