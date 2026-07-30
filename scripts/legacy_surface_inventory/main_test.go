// Este fichero prueba el recorrido histórico y los casos de clasificación exigidos.
package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestInventoryIsDeterministicAndClassifiesRequiredSurfaces(t *testing.T) {
	repository := newTestRepository(t)
	files := map[string][]byte{
		"cmd/herramienta/main.go":            []byte("package main\n"),
		"internal/http/routes.go":            []byte("package http\n"),
		"internal/surface/api.go":            []byte("package surface // api\n"),
		"internal/surface/mcp.go":            []byte("package surface // mcp\n"),
		"internal/surface/panel_web.go":      []byte("package surface // web\n"),
		"config/app.toml":                    []byte("locale = \"es\"\n"),
		"migrations/001_init.sql":            []byte("CREATE TABLE sample(id TEXT);\n"),
		"scripts/tarea.py":                   []byte("#!/usr/bin/env python3\nprint('ok')\n"),
		"deploy/orquesta.service":            []byte("[Service]\nExecStart=/bin/true\n"),
		"tests/fixtures/caso.json":           []byte("{\"ok\":true}\n"),
		"docs/decisions/adr_001.md":          []byte("# Decisión\n"),
		"docs/incidencias/bug_001.md":        []byte("# Incidencia\n"),
		"docs/evidencias/receipt.txt":        []byte("evidencia\n"),
		"datos con espacios/formato.inusual": []byte("contenido\n"),
		"assets/imagen.desconocida":          {0xff, 0xfe, 0x00, 0x80},
		"vendor/tercero/omitido.js":          []byte("vendorizado\n"),
		"node_modules/paquete/omitido.js":    []byte("vendorizado\n"),
	}
	for relative, content := range files {
		writeTestBytes(t, repository, relative, content)
	}
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "superficies")
	gitTest(t, repository, "tag", "-a", "v1", "-m", "v1")

	firstJSONL := filepath.Join(t.TempDir(), "salida uno.jsonl")
	firstManifest := filepath.Join(t.TempDir(), "manifiesto uno.json")
	secondJSONL := filepath.Join(t.TempDir(), "salida dos.jsonl")
	secondManifest := filepath.Join(t.TempDir(), "manifiesto dos.json")
	if err := run(options{repository: repository, jsonl: firstJSONL, manifest: firstManifest}); err != nil {
		t.Fatal(err)
	}
	if err := run(options{repository: repository, jsonl: secondJSONL, manifest: secondManifest}); err != nil {
		t.Fatal(err)
	}
	assertSameFile(t, firstJSONL, secondJSONL)
	assertSameFile(t, firstManifest, secondManifest)
	assertMode(t, firstJSONL, 0o600)
	assertMode(t, firstManifest, 0o600)

	records := readRecords(t, firstJSONL)
	byPath := surfacesByUTF8Path(t, records)
	expectedFamilies := map[string][]string{
		"cmd/herramienta/main.go":            {"codigo_fuente", "ordenes"},
		"internal/http/routes.go":            {"codigo_fuente", "http_mcp_web"},
		"internal/surface/api.go":            {"codigo_fuente", "http_mcp_web"},
		"internal/surface/mcp.go":            {"codigo_fuente", "http_mcp_web"},
		"internal/surface/panel_web.go":      {"codigo_fuente", "http_mcp_web"},
		"config/app.toml":                    {"configuracion"},
		"migrations/001_init.sql":            {"sql_migraciones"},
		"scripts/tarea.py":                   {"shell_python"},
		"deploy/orquesta.service":            {"servicios_despliegue"},
		"tests/fixtures/caso.json":           {"configuracion", "pruebas_datos"},
		"docs/decisions/adr_001.md":          {"documentos_decisiones_incidencias_evidencias"},
		"datos con espacios/formato.inusual": {"desconocido"},
	}
	for filePath, expected := range expectedFamilies {
		items := byPath[filePath]
		if len(items) == 0 {
			t.Fatalf("no se inventarió %q", filePath)
		}
		item := items[0].facts
		if actual := familyNames(item); !equalStringSlices(actual, expected) {
			t.Fatalf("%s: familias=%q; esperadas=%q", filePath, actual, expected)
		}
		if len(item.Summaries) == 0 || len(item.DetectedTypes) == 0 ||
			item.BlobSHA == "" || item.BlobSize == 0 {
			t.Fatalf("%s: hechos incompletos: %#v", filePath, item)
		}
	}
	if byPath["assets/imagen.desconocida"][0].facts.Encoding != "binario" {
		t.Fatalf("el archivo no UTF-8 no fue detectado como binario: %#v", byPath["assets/imagen.desconocida"])
	}
	exclusions := map[string]record{}
	for filePath, surfaces := range byPath {
		for _, surface := range surfaces {
			if surface.entry.ExclusionCause != "" {
				exclusions[filePath] = surface.entry
			}
		}
	}
	expectedExclusions := map[string]string{
		"vendor":       "dependencia_vendorizada_vendor",
		"node_modules": "dependencia_vendorizada_node_modules",
	}
	for excludedPath, cause := range expectedExclusions {
		item, found := exclusions[excludedPath]
		if !found {
			t.Fatalf("la exclusión desapareció del inventario: %s", excludedPath)
		}
		if item.ExclusionCause != cause || item.TreeID == "" ||
			item.ChildObjectID == "" || item.FileMode == "" || item.ChildType != "tree" {
			t.Fatalf("exclusión sin causa o procedencia completa: %#v", item)
		}
	}

	var manifestValue manifest
	readJSONFile(t, firstManifest, &manifestValue)
	inventoryContent, err := os.ReadFile(firstJSONL)
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range []string{
		"ordenes", "http_mcp_web", "configuracion", "sql_migraciones",
		"shell_python", "servicios_despliegue", "pruebas_datos",
		"documentos_decisiones_incidencias_evidencias", "desconocido",
	} {
		if manifestValue.Counts["familia:"+family] == 0 {
			t.Errorf("el manifiesto no cuenta la familia %s", family)
		}
	}
	for _, cause := range expectedExclusions {
		if manifestValue.Counts["exclusion:"+cause] == 0 {
			t.Errorf("el manifiesto no cuenta la exclusión %s", cause)
		}
	}
	if manifestValue.SchemaVersion != schemaVersion ||
		manifestValue.Algorithm != inventoryAlgorithm ||
		manifestValue.InventoryHashDomain != inventoryHashDomain ||
		manifestValue.RecordSchemaSHA256 != recordSchemaDigest() ||
		manifestValue.ClassificationAlgorithm != classificationAlgorithm ||
		manifestValue.InventorySHA256 != inventoryDigest(inventoryContent) ||
		manifestValue.InventoryBytes != int64(len(inventoryContent)) ||
		manifestValue.RefSnapshotSHA256 == "" {
		t.Fatalf("manifiesto sin huellas: %#v", manifestValue)
	}
}

func TestInventoryPreservesHistoricalVariantsAndGitSymlink(t *testing.T) {
	repository := newTestRepository(t)
	writeTestBytes(t, repository, "docs/guia.md", []byte("primera\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "primera")

	writeTestBytes(t, repository, "docs/guia.md", []byte("segunda\n"))
	writeTestBytes(t, repository, "destino con espacios", []byte("destino\n"))
	gitTest(t, repository, "add", ".")
	blob := strings.TrimSpace(gitTestInput(t, repository, []byte("destino con espacios"), "hash-object", "-w", "--stdin"))
	gitTest(t, repository, "update-index", "--add", "--cacheinfo", "120000,"+blob+",enlace con espacios")
	gitTest(t, repository, "commit", "-qm", "segunda")

	jsonl := filepath.Join(t.TempDir(), "inventario.jsonl")
	manifestPath := filepath.Join(t.TempDir(), "manifiesto.json")
	if err := run(options{repository: repository, jsonl: jsonl, manifest: manifestPath}); err != nil {
		t.Fatal(err)
	}
	records := readRecords(t, jsonl)
	byPath := surfacesByUTF8Path(t, records)
	var variantBlobs = map[string]struct{}{}
	for _, surface := range byPath["docs/guia.md"] {
		variantBlobs[surface.facts.BlobID] = struct{}{}
	}
	if len(variantBlobs) != 2 {
		t.Fatalf("historia incompleta: blobs=%v", variantBlobs)
	}
	symlinks := byPath["enlace con espacios"]
	if len(symlinks) == 0 ||
		!containsString(symlinks[0].facts.DetectedTypes, "enlace_simbolico_git") ||
		symlinks[0].entry.FileMode != "120000" {
		var symlink reconstructedSurface
		if len(symlinks) > 0 {
			symlink = symlinks[0]
		}
		t.Fatalf("enlace Git no conservado: %#v", symlink)
	}
}

func TestInventorySupportsInvalidUTF8GitPath(t *testing.T) {
	repository := newTestRepository(t)
	blob := strings.TrimSpace(gitTestInput(t, repository, []byte("contenido\n"), "hash-object", "-w", "--stdin"))
	rawPath := []byte{'g', 'u', 'i', 'a', '-', 0xff, '.', 'm', 'd'}
	index := []byte("100644 " + blob + "\t")
	index = append(index, rawPath...)
	index = append(index, 0)
	gitTestInput(t, repository, index, "update-index", "-z", "--index-info")
	gitTest(t, repository, "commit", "-qm", "ruta no utf8")

	jsonl := filepath.Join(t.TempDir(), "inventario.jsonl")
	if err := run(options{
		repository: repository,
		jsonl:      jsonl,
		manifest:   filepath.Join(t.TempDir(), "manifiesto.json"),
	}); err != nil {
		t.Fatal(err)
	}
	var encoded, classified bool
	for _, surface := range reconstructSurfaces(t, readRecords(t, jsonl)) {
		if string(surface.path) == string(rawPath) &&
			surface.entry.PathEncoding == "base64" &&
			surface.entry.PathSegmentBase64 != "" {
			encoded = true
			classified = containsString(
				familyNames(surface.facts),
				"documentos_decisiones_incidencias_evidencias",
			) && containsString(surface.facts.DetectedTypes, "markdown")
		}
	}
	if !encoded {
		t.Fatal("no se preservó la ruta Git no UTF-8 mediante base64")
	}
	if !classified {
		t.Fatal("la extensión de la ruta no UTF-8 no conservó su clasificación")
	}
}

func TestInventoryUsesBoundedGitProcesses(t *testing.T) {
	repository := newTestRepository(t)
	for index := 0; index < 15; index++ {
		writeTestBytes(t, repository, "scripts/tarea "+strconv.Itoa(index)+".sh", []byte("#!/bin/sh\nexit 0\n"))
		gitTest(t, repository, "add", ".")
		gitTest(t, repository, "commit", "-qm", "cambio")
		if index%3 == 0 {
			gitTest(t, repository, "branch", "rama-"+strconv.Itoa(index))
		}
	}
	processes := 0
	err := run(options{
		repository: repository,
		jsonl:      filepath.Join(t.TempDir(), "inventario.jsonl"),
		manifest:   filepath.Join(t.TempDir(), "manifiesto.json"),
		gitProcessStarted: func() {
			processes++
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if processes > 6 {
		t.Fatalf("se abrieron %d procesos Git; se esperaban como máximo 6", processes)
	}
}

func TestCollectTreePreservesUnreadableObjectAsFailure(t *testing.T) {
	repository := newTestRepository(t)
	writeTestBytes(t, repository, "README.md", []byte("# muestra\n"))
	gitTest(t, repository, "add", ".")
	gitTest(t, repository, "commit", "-qm", "muestra")
	batch, err := newGitBatch(repository, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer batch.close()
	destination := filepath.Join(t.TempDir(), "fallos.jsonl")
	writer, err := newInventoryWriter(destination)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emitObjectGraph(batch, []string{strings.Repeat("0", 40)}, 20, writer)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, temporary, err := writer.seal()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(temporary)
	failures := readRecords(t, temporary)
	if len(failures) != 1 || failures[0].RecordKind != "fallo" ||
		failures[0].ErrorCode != "arbol_no_legible" {
		t.Fatalf("fallo no preservado de forma estructurada: %#v", failures)
	}
}

func TestInventoryRejectsSameOutputAndRepositoryWithoutRefs(t *testing.T) {
	repository := t.TempDir()
	gitTest(t, repository, "init", "-q")
	output := filepath.Join(t.TempDir(), "salida")
	if err := run(options{repository: repository, jsonl: output, manifest: output}); err == nil {
		t.Fatal("se esperaba rechazo de salidas coincidentes")
	}
	if err := run(options{
		repository: repository,
		jsonl:      filepath.Join(t.TempDir(), "inventario.jsonl"),
		manifest:   filepath.Join(t.TempDir(), "manifiesto.json"),
	}); err == nil {
		t.Fatal("se esperaba rechazo de repositorio sin referencias")
	}
}

func TestUTF8ValidatorHandlesSplitRunesAndTruncation(t *testing.T) {
	valid := []byte("ámbito")
	for split := 1; split < len(valid); split++ {
		validator := utf8Validator{valid: true}
		validator.write(valid[:split])
		validator.write(valid[split:])
		if !validator.complete() {
			t.Fatalf("runa válida rechazada al dividir en %d", split)
		}
	}
	validator := utf8Validator{valid: true}
	validator.write([]byte{0xe2, 0x82})
	if validator.complete() {
		t.Fatal("se aceptó una runa UTF-8 truncada")
	}
}
