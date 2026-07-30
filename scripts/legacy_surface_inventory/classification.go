// Este fichero clasifica formas observables sin decidir su semántica ni adopción.
package main

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"
)

func classifyPath(filePath []byte, prefix []byte) classification {
	lower := string(asciiLower(filePath))
	base := path.Base(lower)
	extension := path.Ext(base)
	segments := strings.Split(lower, "/")
	has := func(names ...string) bool {
		for _, segment := range segments {
			for _, name := range names {
				if segment == name {
					return true
				}
			}
		}
		return false
	}
	contains := func(parts ...string) bool {
		for _, part := range parts {
			if strings.Contains(lower, part) {
				return true
			}
		}
		return false
	}

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

	if strings.HasSuffix(base, "_test.go") ||
		strings.HasPrefix(base, "test_") || strings.HasPrefix(base, "smoke_") ||
		has("test", "tests", "testdata", "fixtures", "fixture", "golden", "snapshots") ||
		contains("/test_", "/smoke_", "_test.", ".golden.") {
		add("pruebas_datos", "prueba_o_dato")
	}
	if has("cmd", "commands", "command", "comandos", "cli") {
		add("ordenes", "entrada_ejecutable")
	}
	if has("deploy", "deployment", "deployments", "systemd", "k8s", "kubernetes", "helm", "containers") ||
		contains(".github/workflows/", "docker-compose", "compose.", "/install", "/uninstall") ||
		base == "dockerfile" || base == "containerfile" ||
		extension == ".service" || extension == ".socket" || extension == ".timer" {
		add("servicios_despliegue", "servicio_o_despliegue")
	}
	if extension == ".sql" || has("migration", "migrations", "migracion", "migraciones") {
		add("sql_migraciones", "sql_o_migracion")
	}
	if has("config", "configs", "configuration", "conf") ||
		base == "go.mod" || base == "go.sum" || base == "makefile" ||
		extensionIn(extension, ".json", ".toml", ".yaml", ".yml", ".ini", ".conf", ".env", ".properties") {
		add("configuracion", "configuracion")
	}
	if has("http", "mcp", "web", "frontend", "ui", "public", "static", "templates", "api") ||
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
	if has("docs", "doc", "documentation", "decisions", "decision", "adr", "incidents", "incidencias", "evidence", "evidencias") ||
		extensionIn(extension, ".md", ".markdown", ".rst", ".adoc", ".txt", ".pdf", ".doc", ".docx") {
		subtype := "documentacion"
		switch {
		case has("decisions", "decision", "adr") || contains("decision", "adr_"):
			subtype = "decision"
		case has("incidents", "incidencias") || contains("incidencia", "bug"):
			subtype = "incidencia"
		case has("evidence", "evidencias") || contains("evidencia", "receipt"):
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
	sort.Slice(families, func(i, j int) bool {
		if families[i].Family != families[j].Family {
			return families[i].Family < families[j].Family
		}
		return families[i].Subtype < families[j].Subtype
	})
	return classification{families: families}
}

func detectType(filePath []byte, mode string, facts blobFacts) string {
	if mode == "120000" {
		return "enlace_simbolico_git"
	}
	extension := path.Ext(string(asciiLower(filePath)))
	known := map[string]string{
		".go": "go", ".sh": "shell", ".bash": "shell", ".zsh": "shell",
		".py": "python", ".sql": "sql", ".json": "json", ".toml": "toml",
		".yaml": "yaml", ".yml": "yaml", ".md": "markdown", ".rst": "restructuredtext",
		".html": "html", ".htm": "html", ".css": "css", ".js": "javascript",
		".ts": "typescript", ".tsx": "typescript", ".jsx": "javascript",
		".service": "unidad_systemd", ".socket": "unidad_systemd", ".timer": "unidad_systemd",
		".png": "imagen_png", ".jpg": "imagen_jpeg", ".jpeg": "imagen_jpeg",
		".gif": "imagen_gif", ".pdf": "documento_pdf", ".zip": "archivo_zip",
	}
	if detected := known[extension]; detected != "" {
		return detected
	}
	if facts.encoding == "binario" {
		return "binario_desconocido"
	}
	if extension == "" {
		return "texto_sin_extension"
	}
	return "texto_extension_desconocida"
}

func asciiLower(value []byte) []byte {
	result := make([]byte, len(value))
	for index, current := range value {
		if current >= 'A' && current <= 'Z' {
			current += 'a' - 'A'
		}
		result[index] = current
	}
	return result
}

func structuralSummary(mode string, facts blobFacts, detectedType string) string {
	if mode == "120000" {
		return fmt.Sprintf("enlace simbólico Git de %d bytes; destino no interpretado", facts.size)
	}
	if facts.encoding == "binario" {
		return fmt.Sprintf("contenido binario de %d bytes; tipo %s", facts.size, detectedType)
	}
	return fmt.Sprintf("texto UTF-8 de %d bytes y %d líneas; tipo %s", facts.size, facts.lineCount, detectedType)
}

func extensionIn(extension string, values ...string) bool {
	for _, value := range values {
		if extension == value {
			return true
		}
	}
	return false
}
