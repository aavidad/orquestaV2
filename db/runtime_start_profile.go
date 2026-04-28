package db

import (
	"fmt"
	"orquesta/runtimeagente"
	"strings"
)

func resolverPerfilModeloStartRuntime(agenteRef string, proyecto *Proyecto, conector *Conector, resume runtimeagente.ResumeContext, perfilTarea, modelo, razonamiento string) (string, string, string, error) {
	perfilSolicitado := strings.TrimSpace(perfilTarea)
	modeloSolicitado := strings.TrimSpace(modelo)
	razonamientoSolicitado := strings.TrimSpace(razonamiento)
	perfilPersistido, modeloPersistido, razonamientoPersistido := ResumePayloadPerfilEjecucion(resume.ResumePayloadJSON)
	if perfilSolicitado == "" {
		perfilSolicitado = perfilPersistido
		perfilTarea = perfilPersistido
	}
	conectorRuntime := connectorConfigFromDBConector(conector)
	modeloPersistidoCompatible := runtimeagente.ModeloCompatibleConConector(conectorRuntime, modeloPersistido)
	if modeloSolicitado == "" && modeloPersistidoCompatible {
		modeloSolicitado = modeloPersistido
		modelo = modeloPersistido
	}
	if razonamientoSolicitado == "" && modeloPersistidoCompatible {
		razonamientoSolicitado = razonamientoPersistido
		razonamiento = razonamientoPersistido
	}
	var err error
	perfilTarea, modelo, razonamiento, err = ResolverPerfilEjecucionLanzamiento(
		&agenteRef,
		strings.TrimSpace(proyecto.Slug),
		perfilTarea,
		modelo,
		razonamiento,
	)
	if err != nil {
		return "", "", "", err
	}
	perfilTarea, modelo, razonamiento, err = runtimeagente.AplicarDefaultsConector(
		conectorRuntime,
		perfilSolicitado,
		modeloSolicitado,
		razonamientoSolicitado,
		perfilTarea,
		modelo,
		razonamiento,
	)
	if err != nil {
		return "", "", "", fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
	}
	if strings.TrimSpace(modeloSolicitado) == "" {
		if preferido := runtimeagente.ModeloPreferenteAgenteCompatible(strings.TrimSpace(agenteRef), conectorRuntime); preferido != "" {
			if strings.TrimSpace(modelo) == "" || !runtimeagente.ModeloAfinAgente(strings.TrimSpace(agenteRef), strings.TrimSpace(modelo)) {
				modelo = preferido
			}
		}
	}
	if !runtimeagente.ModeloCompatibleConConector(conectorRuntime, modelo) {
		modelo = ""
		if modeloSolicitado == "" {
			perfilTarea, modelo, razonamiento, err = runtimeagente.AplicarDefaultsConector(
				conectorRuntime,
				perfilSolicitado,
				"",
				razonamientoSolicitado,
				perfilTarea,
				"",
				razonamiento,
			)
			if err != nil {
				return "", "", "", fmt.Errorf("metadata_json inválido para '%s': %w", strings.TrimSpace(conector.Slug), err)
			}
		}
	}
	return perfilTarea, modelo, razonamiento, nil
}
