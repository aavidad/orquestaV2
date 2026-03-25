/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strings"
	"time"
)

func buildExportStateMarkdown() (string, error) {
	var sb strings.Builder
	sb.WriteString("# Orquesta — Estado exportado\n\n")
	sb.WriteString(fmt.Sprintf("_Generado: %s_\n\n", time.Now().Format("2006-01-02 15:04:05")))

	resumen, err := buildEstadoResumen()
	if err != nil {
		return "", err
	}

	sb.WriteString("## Agentes\n\n")
	sb.WriteString("| Agente | Rol | Activo |\n")
	sb.WriteString("|--------|-----|--------|\n")
	for _, a := range resumen.AgentesActivos {
		sb.WriteString(fmt.Sprintf("| %s | %s | sí |\n", a.Nombre, a.Rol))
	}
	sb.WriteString("\n## Propuestas abiertas\n\n")
	for _, p := range resumen.PropuestasAbiertas {
		sb.WriteString(fmt.Sprintf("- %s — %s\n", p.Codigo, p.Titulo))
	}
	sb.WriteString("\n## Tareas activas\n\n")
	for _, t := range resumen.TareasActivas {
		sb.WriteString(fmt.Sprintf("- #%d %s [%s]\n", t.ID, t.Titulo, t.Estado))
	}
	return sb.String(), nil
}

func buildExportAuditMarkdown(limit int) (string, error) {
	entries, err := operacionesService.AuditLog(limit)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Audit Log (últimas %d entradas)\n\n", limit))
	sb.WriteString("| Fecha | Agente | Acción | Entidad | ID | Detalle |\n")
	sb.WriteString("|-------|--------|--------|---------|----|---------|\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d | %s |\n",
			e.CreatedAt.Format("2006-01-02 15:04"),
			e.Agente, e.Accion, e.Entidad, e.EntidadID, e.Detalle))
	}
	return sb.String(), nil
}
