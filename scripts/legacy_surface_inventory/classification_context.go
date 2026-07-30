// Este fichero reduce cualquier prefijo de ruta a las señales finitas que
// consumen las reglas de clasificación. No conserva rutas ni texto creciente.
package main

import (
	"bytes"
	"path"
	"sort"
	"strings"
)

type classificationContext uint16

const (
	contextTest classificationContext = 1 << iota
	contextCommand
	contextDeploy
	contextMigration
	contextConfig
	contextPublic
	contextDocs
	contextDecision
	contextIncident
	contextEvidence
	contextLastDotGitHub
	contextHasPriorSegment

	classificationContextStates = 1 << 12
)

func extendClassificationContext(
	parent classificationContext,
	segment []byte,
	hasDescendant bool,
) classificationContext {
	lower := string(asciiLower(segment))
	result := parent
	hasPrior := parent&contextHasPriorSegment != 0

	if stringIn(lower, "test", "tests", "testdata", "fixtures", "fixture", "golden", "snapshots") ||
		hasPrior && (strings.HasPrefix(lower, "test_") || strings.HasPrefix(lower, "smoke_")) ||
		strings.Contains(lower, "_test.") || strings.Contains(lower, ".golden.") {
		result |= contextTest
	}
	if stringIn(lower, "cmd", "commands", "command", "comandos", "cli") {
		result |= contextCommand
	}
	if stringIn(lower, "deploy", "deployment", "deployments", "systemd", "k8s", "kubernetes", "helm", "containers") ||
		strings.Contains(lower, "docker-compose") || strings.Contains(lower, "compose.") ||
		hasPrior && (strings.HasPrefix(lower, "install") || strings.HasPrefix(lower, "uninstall")) ||
		parent&contextLastDotGitHub != 0 && lower == "workflows" && hasDescendant {
		result |= contextDeploy
	}
	if stringIn(lower, "migration", "migrations", "migracion", "migraciones") {
		result |= contextMigration
	}
	if stringIn(lower, "config", "configs", "configuration", "conf") {
		result |= contextConfig
	}
	if stringIn(lower, "http", "mcp", "web", "frontend", "ui", "public", "static", "templates", "api") {
		result |= contextPublic
	}
	if stringIn(lower, "docs", "doc", "documentation", "decisions", "decision", "adr",
		"incidents", "incidencias", "evidence", "evidencias") {
		result |= contextDocs
	}
	if stringIn(lower, "decisions", "decision", "adr") ||
		strings.Contains(lower, "decision") || strings.Contains(lower, "adr_") {
		result |= contextDecision
	}
	if stringIn(lower, "incidents", "incidencias") ||
		strings.Contains(lower, "incidencia") || strings.Contains(lower, "bug") {
		result |= contextIncident
	}
	if stringIn(lower, "evidence", "evidencias") ||
		strings.Contains(lower, "evidencia") || strings.Contains(lower, "receipt") {
		result |= contextEvidence
	}

	result &^= contextLastDotGitHub
	if hasDescendant && strings.HasSuffix(lower, ".github") {
		result |= contextLastDotGitHub
	}
	result |= contextHasPriorSegment
	return result
}

func classifyWithContext(
	context classificationContext,
	segment []byte,
	prefix []byte,
) classification {
	lower := string(asciiLower(segment))
	base := path.Base(lower)
	extension := path.Ext(base)
	var families []surfaceFamily
	seen := map[surfaceFamily]struct{}{}
	add := func(family, subtype string) {
		item := surfaceFamily{Family: family, Subtype: subtype}
		if _, found := seen[item]; found {
			return
		}
		seen[item] = struct{}{}
		families = append(families, item)
	}

	if context&contextTest != 0 ||
		strings.HasSuffix(base, "_test.go") ||
		strings.HasPrefix(base, "test_") || strings.HasPrefix(base, "smoke_") {
		add("pruebas_datos", "prueba_o_dato")
	}
	if context&contextCommand != 0 {
		add("ordenes", "entrada_ejecutable")
	}
	if context&contextDeploy != 0 ||
		base == "dockerfile" || base == "containerfile" ||
		extension == ".service" || extension == ".socket" || extension == ".timer" {
		add("servicios_despliegue", "servicio_o_despliegue")
	}
	if context&contextMigration != 0 || extension == ".sql" {
		add("sql_migraciones", "sql_o_migracion")
	}
	if context&contextConfig != 0 ||
		base == "go.mod" || base == "go.sum" || base == "makefile" ||
		extensionIn(extension, ".json", ".toml", ".yaml", ".yml", ".ini", ".conf", ".env", ".properties") {
		add("configuracion", "configuracion")
	}
	if context&contextPublic != 0 ||
		baseHasSurfaceToken(base, extension) ||
		extensionIn(extension, ".html", ".htm", ".css", ".js", ".jsx", ".ts", ".tsx", ".vue", ".svelte") {
		add("http_mcp_web", "superficie_publica")
	}
	if extensionIn(extension, ".sh", ".bash", ".zsh", ".fish", ".py", ".pyw") ||
		bytes.HasPrefix(prefix, []byte("#!/bin/sh")) ||
		bytes.HasPrefix(prefix, []byte("#!/usr/bin/env bash")) ||
		bytes.HasPrefix(prefix, []byte("#!/usr/bin/env python")) {
		add("shell_python", "guion")
	}
	if context&contextDocs != 0 ||
		extensionIn(extension, ".md", ".markdown", ".rst", ".adoc", ".txt", ".pdf", ".doc", ".docx") {
		subtype := "documentacion"
		switch {
		case context&contextDecision != 0:
			subtype = "decision"
		case context&contextIncident != 0:
			subtype = "incidencia"
		case context&contextEvidence != 0:
			subtype = "evidencia"
		}
		add("documentos_decisiones_incidencias_evidencias", subtype)
	}
	if extensionIn(extension, ".go", ".c", ".h", ".cc", ".cpp", ".rs", ".java", ".kt", ".rb", ".php", ".cs") {
		add("codigo_fuente", "codigo_no_clasificado")
	}
	if len(families) == 0 {
		add("desconocido", "formato_no_clasificado")
	}
	sort.Slice(families, func(left, right int) bool {
		if families[left].Family != families[right].Family {
			return families[left].Family < families[right].Family
		}
		return families[left].Subtype < families[right].Subtype
	})
	return classification{families: families}
}

func baseHasSurfaceToken(base, extension string) bool {
	stem := strings.TrimSuffix(base, extension)
	for _, token := range strings.FieldsFunc(stem, func(value rune) bool {
		return value < '0' || value > '9' && value < 'a' || value > 'z'
	}) {
		if stringIn(token, "http", "mcp", "web", "frontend", "ui", "api") {
			return true
		}
	}
	return false
}

func stringIn(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}
