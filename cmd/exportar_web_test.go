/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"strings"
	"testing"
	"time"

	"orquesta/db"
)

func TestBuildExportAuditMarkdownUsaAPI(t *testing.T) {
	prepararDBTemporalCmd(t)

	db.Audit("Codex1", "crear_tarea", "tarea", 7, "detalle de prueba")
	db.Audit("Codex2", "votar", "propuesta", 9, "acuerdo")

	out, err := buildExportAuditMarkdown(10)
	if err != nil {
		t.Fatalf("buildExportAuditMarkdown: %v", err)
	}
	for _, token := range []string{
		"# Audit Log",
		"Codex1",
		"crear_tarea",
		"detalle de prueba",
		"Codex2",
		"propuesta",
		time.Now().Format("2006-01-02"),
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("salida audit markdown sin %q:\n%s", token, out)
		}
	}
}
