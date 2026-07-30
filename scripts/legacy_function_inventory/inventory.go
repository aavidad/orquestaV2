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
	repository, err = filepath.EvalSymlinks(filepath.Clean(repository))
	if err != nil {
		return fmt.Errorf("resolver físicamente el repositorio histórico: %w", err)
	}
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
	history, err := readHistory(repository, refsBefore, options.gitProcessStarted)
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
		if err := stream.emit(referenceRecord(ref)); err != nil {
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
		BlobDigestDomain:    goBlobDigestDomain,
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

func referenceRecord(ref refInfo) record {
	nameEncoding, nameText, nameBase64 := encodePathSegment(ref.Name)
	result := record{
		RecordKind:      "reference",
		RefNameEncoding: nameEncoding,
		RefName:         nameText,
		RefNameBase64:   nameBase64,
		RefMode:         ref.Mode,
		ObjectID:        ref.Object,
		CommitID:        ref.Commit,
	}
	if len(ref.Target) > 0 {
		result.RefTargetEncoding, result.RefTarget, result.RefTargetBase64 =
			encodePathSegment(ref.Target)
	}
	return result
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
	if info, lstatErr := os.Lstat(absolute); lstatErr == nil &&
		info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("la salida no puede reemplazar un enlace simbólico")
	} else if lstatErr != nil && !errors.Is(lstatErr, os.ErrNotExist) {
		return "", lstatErr
	}
	physical, err := resolvePhysicalDestination(absolute)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(repository, physical)
	if err == nil && relative != ".." &&
		(relative == "." || !startsOutside(relative)) {
		return "", fmt.Errorf("la salida no puede escribirse dentro del repositorio histórico: %s", physical)
	}
	return physical, nil
}

func resolvePhysicalDestination(path string) (string, error) {
	existing := path
	for {
		_, err := os.Lstat(existing)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", fmt.Errorf("no existe ningún padre resoluble para %s", path)
		}
		existing = parent
	}
	resolved, err := filepath.EvalSymlinks(existing)
	if err != nil {
		return "", err
	}
	suffix, err := filepath.Rel(existing, path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(resolved, suffix)), nil
}

func startsOutside(relative string) bool {
	return len(relative) > 3 &&
		relative[:3] == ".."+string(filepath.Separator)
}
