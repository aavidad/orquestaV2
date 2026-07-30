// Este fichero coordina el censo desde referencias Git estables hacia el
// escritor progresivo. No acumula registros ni el JSONL completo en memoria.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

type options struct {
	repository string
	jsonl      string
	manifest   string

	gitProcessStarted func()
}

func run(options options) (runErr error) {
	repository, err := filepath.Abs(options.repository)
	if err != nil {
		return err
	}
	repository = filepath.Clean(repository)
	jsonlPath, err := normalizedOutput(repository, options.jsonl)
	if err != nil {
		return err
	}
	manifestPath, err := normalizedOutput(repository, options.manifest)
	if err != nil {
		return err
	}
	if jsonlPath == manifestPath {
		return errors.New("inventario y manifiesto necesitan rutas distintas")
	}

	refsBefore, err := readRefs(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	refDigest := digestJSON(referenceHashDomain, refsBefore)
	history, err := readHistory(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	if len(history) == 0 {
		return errors.New("el repositorio no tiene confirmaciones alcanzables")
	}
	commits := sortedCommits(history)
	objectFormat, err := gitText(
		repository,
		options.gitProcessStarted,
		"rev-parse",
		"--show-object-format",
	)
	if err != nil {
		return err
	}
	objectBytes, err := objectIDBytes(objectFormat)
	if err != nil {
		return err
	}

	stream, err := newInventoryStream(jsonlPath)
	if err != nil {
		return err
	}
	defer func() {
		if runErr != nil {
			stream.abort()
		}
	}()
	if err := stream.emit(record{
		RecordKind: "inventory_header", SchemaVersion: schemaVersion, Algorithm: inventoryAlgorithm,
	}); err != nil {
		return err
	}
	for _, ref := range refsBefore {
		if err := stream.emit(record{
			RecordKind: "reference", RefName: ref.Name,
			ObjectID: ref.Object, CommitID: ref.Commit,
		}); err != nil {
			return err
		}
	}
	for _, commit := range commits {
		header := history[commit]
		if err := stream.emit(record{
			RecordKind: "commit", CommitID: commit,
			TreeID: header.tree, Parents: header.parents,
		}); err != nil {
			return err
		}
	}

	batch, err := newGitBatch(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	if err := emitObjectGraph(batch, commits, history, objectBytes, stream); err != nil {
		_ = batch.close()
		return err
	}
	if err := batch.close(); err != nil {
		return err
	}
	refsAfter, err := readRefs(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	if !equalRefs(refsBefore, refsAfter) {
		return errors.New("las referencias Git cambiaron durante el censo")
	}
	inventorySHA, inventoryBytes, counts, err := stream.finish()
	if err != nil {
		return err
	}

	manifestBytes, err := json.MarshalIndent(manifest{
		SchemaVersion:       schemaVersion,
		Algorithm:           inventoryAlgorithm,
		InventoryHashDomain: inventoryHashDomain,
		RecordSchemaSHA256:  recordSchemaDigest(),
		Repository:          repository,
		GitObjectFormat:     objectFormat,
		GoVersion:           runtime.Version(),
		InventorySHA256:     inventorySHA,
		InventoryBytes:      inventoryBytes,
		Counts:              counts,
		RefSnapshotSHA256:   refDigest,
	}, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')
	manifestTemporary, err := writeTemporary(manifestPath, manifestBytes)
	if err != nil {
		return err
	}
	defer os.Remove(manifestTemporary)
	if err := publishPair(stream.temporaryPath, jsonlPath, manifestTemporary, manifestPath); err != nil {
		return err
	}
	stream.published = true
	return nil
}

func sortedCommits(history map[string]historyEntry) []string {
	commits := make([]string, 0, len(history))
	for commit := range history {
		commits = append(commits, commit)
	}
	sort.Strings(commits)
	return commits
}

func normalizedOutput(repository, output string) (string, error) {
	absolute, err := filepath.Abs(output)
	if err != nil {
		return "", err
	}
	absolute = filepath.Clean(absolute)
	relative, err := filepath.Rel(repository, absolute)
	if err == nil && relative != ".." &&
		(relative == "." || !startsOutside(relative)) {
		return "", fmt.Errorf("la salida no puede escribirse dentro del repositorio histórico: %s", absolute)
	}
	return absolute, nil
}

func startsOutside(relative string) bool {
	return len(relative) > 3 &&
		relative[:3] == ".."+string(filepath.Separator)
}
