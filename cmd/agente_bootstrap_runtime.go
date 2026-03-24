/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"orquesta/db"
	"orquesta/internal/bootstrapruntime"
	"orquesta/runtimeagente"
)

type agenteBootstrapRuntimeState = bootstrapruntime.State

func prepararBootstrapRuntimeAgente(agente string, proyecto *db.Proyecto, ultima *db.Sesion) (runtimeagente.ResumeContext, *agenteBootstrapRuntimeState, error) {
	return bootstrapruntime.Preparar(agente, proyecto, ultima)
}
