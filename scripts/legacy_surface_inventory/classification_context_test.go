// Estas pruebas demuestran el límite finito del contexto frente a un DAG de
// rutas exponenciales y su equivalencia con el recorrido ingenuo completo.
package main

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func TestFiniteClassificationContextBoundsDepthTwentyTwoDAG(t *testing.T) {
	const depth = 22
	cache, aggregates := sharedClassificationDAG(depth, false)
	visits := classifyObjectGraph([]string{"nivel-0"}, cache, aggregates)
	if visits != depth+1 {
		t.Fatalf("visitas=%d; esperadas=%d para %d rutas potenciales",
			visits, depth+1, 1<<depth)
	}
	if visits > len(cache)*classificationContextStates {
		t.Fatalf("visitas=%d superan el límite conceptual=%d", visits, len(cache)*classificationContextStates)
	}
	for blobID, aggregate := range aggregates {
		if len(aggregate.classifications) == 0 {
			t.Fatalf("el blob %s quedó sin clasificación", blobID)
		}
	}
}

func TestFiniteClassificationMatchesNaiveTraversal(t *testing.T) {
	const depth = 6
	cache, finite := sharedClassificationDAG(depth, true)
	_, naive := sharedClassificationDAG(depth, true)
	classifyObjectGraph([]string{"nivel-0"}, cache, finite)
	classifyTreeNaively("nivel-0", nil, cache, naive)
	for blobID := range finite {
		if left, right := aggregateViewOf(finite[blobID]), aggregateViewOf(naive[blobID]); !reflect.DeepEqual(left, right) {
			t.Fatalf("clasificación distinta para %s:\nfinita=%#v\ningenua=%#v", blobID, left, right)
		}
	}
}

func TestFiniteContextMatchesEveryCurrentPathSignal(t *testing.T) {
	paths := [][]byte{
		[]byte("tests/caso.json"),
		[]byte("raiz/test_caso/dato.bin"),
		[]byte("raiz/smoke_caso/dato.bin"),
		[]byte("raiz/caso_test.go"),
		[]byte("raiz/caso.golden.txt"),
		[]byte("cmd/herramienta/main.go"),
		[]byte(".github/workflows/prueba"),
		[]byte("raiz/foo.github/workflows/prueba"),
		[]byte("raiz/docker-compose.local"),
		[]byte("raiz/compose.prod"),
		[]byte("raiz/installador/dato"),
		[]byte("installador/dato"),
		[]byte("systemd/orquesta.service"),
		[]byte("migration/cambio"),
		[]byte("raiz/cambio.sql"),
		[]byte("config/app.toml"),
		[]byte("raiz/go.mod"),
		[]byte("internal/mcp.go"),
		[]byte("frontend/panel.ts"),
		[]byte("scripts/tarea.py"),
		[]byte("docs/decision_adr_1.md"),
		[]byte("docs/incidencia_bug_1.md"),
		[]byte("docs/evidencia_receipt_1.md"),
		[]byte("decision_servicio/guia.md"),
		[]byte("incidencia_servicio/guia.md"),
		[]byte("evidencia_servicio/guia.md"),
		[]byte("internal/dominio/modelo.go"),
		[]byte("datos/formato.desconocido"),
		{'g', 'u', 'i', 'a', '-', 0xff, '.', 'm', 'd'},
	}
	for _, filePath := range paths {
		naive := classifyPath(filePath, []byte("#!/usr/bin/env python3\n"))
		finite := classifyPathWithFiniteContext(filePath, []byte("#!/usr/bin/env python3\n"))
		if !reflect.DeepEqual(naive, finite) {
			t.Fatalf("ruta %q:\ningenua=%#v\nfinita=%#v", filePath, naive, finite)
		}
	}
}

func sharedClassificationDAG(
	depth int,
	semanticBranches bool,
) (map[string][]treeEntry, map[string]*blobAggregate) {
	cache := map[string][]treeEntry{}
	signals := []string{"decision", "cmd", "tests", "deploy", "config", "api"}
	for level := 0; level < depth; level++ {
		left := fmt.Sprintf("rama-a-%d", level)
		if semanticBranches && level < len(signals) {
			left = signals[level]
		}
		cache[fmt.Sprintf("nivel-%d", level)] = []treeEntry{
			{mode: "40000", oid: fmt.Sprintf("nivel-%d", level+1), name: []byte(left)},
			{mode: "40000", oid: fmt.Sprintf("nivel-%d", level+1), name: []byte(fmt.Sprintf("rama-b-%d", level))},
		}
	}
	cache[fmt.Sprintf("nivel-%d", depth)] = []treeEntry{
		{mode: "100644", oid: "blob-api", name: []byte("mcp.go")},
		{mode: "100644", oid: "blob-documento", name: []byte{'g', 'u', 'i', 'a', '-', 0xff, '.', 'm', 'd'}},
	}
	return cache, map[string]*blobAggregate{
		"blob-api": newBlobAggregate(blobFacts{
			size: 12, encoding: "utf-8", prefix: []byte("package api\n"), lineCount: 1,
		}),
		"blob-documento": newBlobAggregate(blobFacts{
			size: 12, encoding: "utf-8", prefix: []byte("# Documento\n"), lineCount: 1,
		}),
	}
}

func classifyTreeNaively(
	treeID string,
	prefix []byte,
	cache map[string][]treeEntry,
	aggregates map[string]*blobAggregate,
) {
	for _, entry := range cache[treeID] {
		childType := treeEntryType(entry.mode)
		if exclusionCause(entry.name, childType) != "" {
			continue
		}
		fullPath := joinGitPath(prefix, entry.name)
		if childType == "tree" {
			classifyTreeNaively(entry.oid, fullPath, cache, aggregates)
			continue
		}
		aggregate := aggregates[entry.oid]
		classified := classifyPath(fullPath, aggregate.facts.prefix)
		detectedType := detectType(fullPath, entry.mode, aggregate.facts)
		addClassification(aggregate, classified, detectedType, entry.mode)
	}
}

func classifyPathWithFiniteContext(filePath, prefix []byte) classification {
	segments := bytes.Split(filePath, []byte{'/'})
	var context classificationContext
	for index, segment := range segments {
		context = extendClassificationContext(context, segment, index < len(segments)-1)
	}
	return classifyWithContext(context, segments[len(segments)-1], prefix)
}

type aggregateView struct {
	families        []surfaceFamily
	detectedTypes   []string
	fileModes       []string
	summaries       []string
	classifications []string
}

func aggregateViewOf(aggregate *blobAggregate) aggregateView {
	return aggregateView{
		families:        sortedFamilies(aggregate.families),
		detectedTypes:   sortedSet(aggregate.detectedTypes),
		fileModes:       sortedSet(aggregate.fileModes),
		summaries:       sortedSet(aggregate.summaries),
		classifications: sortedSet(aggregate.classifications),
	}
}
