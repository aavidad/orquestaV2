package db

import (
	"database/sql"
	"fmt"
	"orquesta/runtimeagente"
	"strings"
)

func cargarContextoStartRuntime(agenteRef, proyectoRef, conectorRef string) (*Agente, *Proyecto, *Conector, *Sesion, runtimeagente.ResumeContext, error) {
	agente, err := GetAgente(strings.TrimSpace(agenteRef))
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, err
	}
	if agente == nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, fmt.Errorf("agente no encontrado")
	}
	if !agente.Habilitado {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, fmt.Errorf("agente retirado o deshabilitado")
	}
	proyecto, err := GetProyecto(strings.TrimSpace(proyectoRef))
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, err
	}
	proyecto = ProyectoPrepareLiteConRutaEfectiva(proyecto, strings.TrimSpace(agenteRef))
	ultima, err := ObtenerUltimaSesion(strings.TrimSpace(agenteRef), &proyecto.ID)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, err
	}
	conector, err := resolverConectorRuntimeOrder(strings.TrimSpace(agenteRef), strings.TrimSpace(proyecto.Slug), conectorRef, ultima)
	if err != nil {
		return nil, nil, nil, nil, runtimeagente.ResumeContext{}, err
	}
	resume := runtimeagente.ResumeContext{}
	if ultima != nil {
		resume = runtimeagente.ResumeContext{
			ExternalSessionID:  strings.TrimSpace(ultima.ExternalSessionID),
			ResumePayloadJSON:  strings.TrimSpace(ultima.ResumePayloadJSON),
			ResumenContinuidad: strings.TrimSpace(ultima.ResumenContinuidad),
			Branch:             strings.TrimSpace(ultima.Branch),
			CWD:                strings.TrimSpace(ultima.CWD),
		}
	}
	return agente, proyecto, conector, ultima, SanitizeResumeContextForProject(resume, proyecto), nil
}
