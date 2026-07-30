// Este fichero contiene la entrada y coordina el grafo histórico normalizado.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	var selected options
	flag.StringVar(&selected.repository, "repo", ".", "repositorio Git que se censará")
	flag.StringVar(&selected.jsonl, "jsonl", "", "salida JSONL obligatoria")
	flag.StringVar(&selected.manifest, "manifest", "", "manifiesto JSON obligatorio")
	flag.Parse()
	if selected.jsonl == "" || selected.manifest == "" {
		fatal(errors.New("debes indicar --jsonl y --manifest"))
	}
	if err := run(selected); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "legacy_surface_inventory:", err)
	os.Exit(1)
}

func run(selected options) error {
	repository, err := filepath.Abs(selected.repository)
	if err != nil {
		return err
	}
	repository = filepath.Clean(repository)
	selected.jsonl, err = normalizedOutput(repository, selected.jsonl)
	if err != nil {
		return err
	}
	selected.manifest, err = normalizedOutput(repository, selected.manifest)
	if err != nil {
		return err
	}
	if sameFileName(selected.jsonl, selected.manifest) {
		return errors.New("--jsonl y --manifest deben señalar ficheros distintos")
	}
	refsBefore, err := readRefs(repository, selected.gitProcessStarted)
	if err != nil {
		return err
	}
	if len(refsBefore) == 0 {
		return errors.New("el repositorio no tiene referencias")
	}
	if err := validateOutputDestinations(repository, selected.jsonl, selected.manifest); err != nil {
		return err
	}
	history, err := readHistory(repository, refsBefore, selected.gitProcessStarted)
	if err != nil {
		return err
	}
	objectFormat, err := gitText(repository, selected.gitProcessStarted, "rev-parse", "--show-object-format")
	if err != nil {
		return err
	}
	objectBytes, err := objectIDBytes(objectFormat)
	if err != nil {
		return err
	}

	output, err := newInventoryWriter(selected.jsonl)
	if err != nil {
		return err
	}
	defer output.abort()
	if err := output.write(record{
		RecordKind: "cabecera_inventario", SchemaVersion: schemaVersion, Algorithm: inventoryAlgorithm,
	}); err != nil {
		return err
	}
	for _, ref := range refsBefore {
		if err := output.write(record{
			RecordKind: "referencia", RefName: ref.Name, ObjectID: ref.Object,
			ObjectType: ref.ObjectType, TargetID: ref.Target, TargetType: ref.TargetType,
		}); err != nil {
			return err
		}
	}
	commits := make([]string, 0, len(history))
	rootSet := map[string]struct{}{}
	for commit := range history {
		commits = append(commits, commit)
		rootSet[history[commit].tree] = struct{}{}
	}
	directBlobSet := map[string]struct{}{}
	for _, ref := range refsBefore {
		switch ref.TargetType {
		case "tree":
			rootSet[ref.Target] = struct{}{}
		case "blob":
			directBlobSet[ref.Target] = struct{}{}
		}
	}
	if len(rootSet) == 0 && len(directBlobSet) == 0 {
		return errors.New("las referencias no alcanzan confirmaciones, árboles ni blobs")
	}
	sort.Strings(commits)
	for _, commit := range commits {
		header := history[commit]
		if err := output.write(record{
			RecordKind: "confirmacion", CommitID: commit,
			TreeID: header.tree, Parents: header.parents,
		}); err != nil {
			return err
		}
	}
	rootTrees := make([]string, 0, len(rootSet))
	for root := range rootSet {
		rootTrees = append(rootTrees, root)
	}
	sort.Strings(rootTrees)
	directBlobs := make([]string, 0, len(directBlobSet))
	for blob := range directBlobSet {
		directBlobs = append(directBlobs, blob)
	}
	sort.Strings(directBlobs)

	batch, err := newGitBatch(repository, selected.gitProcessStarted)
	if err != nil {
		return err
	}
	cache, err := emitObjectGraph(batch, rootTrees, objectBytes, output)
	if err != nil {
		_ = batch.close()
		return err
	}
	readableBlobs := collectReadableBlobs(rootTrees, directBlobs, cache)
	aggregates, err := readBlobAggregates(batch, readableBlobs, output)
	if err != nil {
		_ = batch.close()
		return err
	}
	classificationVisits := classifyObjectGraph(rootTrees, cache, aggregates)
	classifyDirectBlobs(directBlobs, aggregates)
	if err := emitBlobAggregates(output, aggregates); err != nil {
		_ = batch.close()
		return err
	}
	if err := batch.close(); err != nil {
		return err
	}
	inventoryDigest, inventoryBytes, counts, jsonlTemporary, err := output.seal()
	if err != nil {
		return err
	}
	counts["contextos_clasificacion_visitados"] = classificationVisits
	counts["estados_contexto_maximos_por_arbol"] = classificationContextStates

	refsAfter, err := readRefs(repository, selected.gitProcessStarted)
	if err != nil {
		return err
	}
	if !equalRefs(refsBefore, refsAfter) {
		return errors.New("las referencias cambiaron durante el censo; no se publicaron salidas")
	}
	if err := validateOutputDestinations(repository, selected.jsonl, selected.manifest); err != nil {
		return err
	}
	return publishInventory(
		selected, repository, objectFormat, refsBefore,
		inventoryDigest, inventoryBytes, counts, jsonlTemporary,
	)
}

func normalizedOutput(repository, destination string) (string, error) {
	absolute, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	relative, err := filepath.Rel(repository, absolute)
	if err == nil && (relative == "." ||
		(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))) {
		return "", fmt.Errorf("la salida no puede escribirse dentro del repositorio histórico: %s", absolute)
	}
	return absolute, nil
}
