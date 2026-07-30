// Este fichero coordina un censo completo e inmutable desde referencias Git.
// No interpreta la utilidad del legado: produce registros deterministas para
// que el inventario semántico y sus revisiones trabajen sobre hechos.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const schemaVersion = 1

type options struct {
	repository string
	jsonl      string
	manifest   string

	gitProcessStarted func()
}

func run(options options) error {
	repository, err := filepath.Abs(options.repository)
	if err != nil {
		return err
	}
	refsBefore, err := readRefs(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	refDigest := digestJSON("orquesta.legacy-function-inventory.refs.v1", refsBefore)
	history, err := readHistory(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	if len(history) == 0 {
		return errors.New("el repositorio no tiene confirmaciones alcanzables")
	}
	commits := make([]string, 0, len(history))
	for commit := range history {
		commits = append(commits, commit)
	}
	sort.Strings(commits)

	records := make([]record, 0, len(commits)*8)
	counts := map[string]int{}
	for _, ref := range refsBefore {
		records = append(records, record{
			RecordKind: "ref", RefName: ref.Name, ObjectID: ref.Object, CommitID: ref.Commit,
		})
		reachable, err := reachableCommits(ref.Commit, history)
		if err != nil {
			return err
		}
		for _, commit := range reachable {
			records = append(records, record{
				RecordKind: "commit_ref_reachability", RefName: ref.Name, CommitID: commit,
			})
		}
	}

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
	batch, err := newGitBatch(repository, options.gitProcessStarted)
	if err != nil {
		return err
	}
	defer batch.close()

	seenTrees := map[string]struct{}{}
	treeCache := map[string][]treeEntry{}
	seenBlobs := map[string]blobResult{}
	variants := map[string]record{}
	for _, commit := range commits {
		header := history[commit]
		records = append(records, record{
			RecordKind: "commit", CommitID: commit, TreeID: header.tree, Parents: header.parents,
		})
		if _, found := seenTrees[header.tree]; !found {
			seenTrees[header.tree] = struct{}{}
			records = append(records, record{RecordKind: "tree", TreeID: header.tree})
		}
		entries, err := readTree(batch, header.tree, objectBytes, treeCache)
		if err != nil {
			return fmt.Errorf("árbol %s: %w", header.tree, err)
		}
		for _, entry := range entries {
			if entry.Type == "tree" {
				if _, found := seenTrees[entry.OID]; !found {
					seenTrees[entry.OID] = struct{}{}
					records = append(records, record{RecordKind: "tree", TreeID: entry.OID})
				}
				continue
			}
			if entry.Type != "blob" || !strings.HasSuffix(entry.Path, ".go") {
				continue
			}
			result, found := seenBlobs[entry.OID]
			if !found {
				object, err := batch.get(entry.OID)
				if err != nil {
					return err
				}
				if object.kind != "blob" {
					return fmt.Errorf("objeto %s de %s no es un blob", entry.OID, entry.Path)
				}
				result, err = parseBlob(object.content, entry.Path)
				if err != nil {
					return err
				}
				seenBlobs[entry.OID] = result
				blobRecord := record{
					RecordKind: "go_blob", BlobID: entry.OID, BlobSize: result.size,
					BlobSHA: result.sha,
				}
				records = append(records, blobRecord)
				if result.failure != nil {
					failure := *result.failure
					failure.BlobID = entry.OID
					records = append(records, failure)
				}
				for _, symbol := range result.records {
					if previous, exists := variants[symbol.VariantRef]; exists &&
						previous.CanonicalSource != symbol.CanonicalSource {
						return fmt.Errorf("colisión de variante %s", symbol.VariantRef)
					}
					if _, exists := variants[symbol.VariantRef]; !exists {
						variant := symbol
						variant.RecordKind = "function_variant"
						variant.Path = ""
						variant.TestFile = false
						variant.StartOffset, variant.EndOffset = 0, 0
						variant.StartLine, variant.EndLine = 0, 0
						variants[symbol.VariantRef] = variant
					}
				}
			}
			if result.failure != nil {
				records = append(records, record{
					RecordKind: "parse_failure_occurrence", CommitID: commit, TreeID: header.tree,
					Path: entry.Path, FileMode: entry.Mode, BlobID: entry.OID,
					ErrorCode: result.failure.ErrorCode,
				})
				continue
			}
			for _, symbol := range result.records {
				occurrence := symbol
				occurrence.RecordKind = "function_occurrence"
				occurrence.CommitID = commit
				occurrence.TreeID = header.tree
				occurrence.Path = entry.Path
				occurrence.TestFile = strings.HasSuffix(entry.Path, "_test.go")
				occurrence.FileMode = entry.Mode
				occurrence.BlobID = entry.OID
				occurrence.BlobSize = result.size
				occurrence.BlobSHA = result.sha
				occurrence.OccurrenceRef = digestStrings(
					"orquesta.legacy-go-function-occurrence.v1",
					commit, entry.Path, strconv.Itoa(symbol.StartOffset), symbol.VariantRef,
				)
				occurrence.CanonicalSource = ""
				records = append(records, occurrence)
			}
		}
	}
	var variantKeys []string
	for key := range variants {
		variantKeys = append(variantKeys, key)
	}
	sort.Strings(variantKeys)
	for _, key := range variantKeys {
		records = append(records, variants[key])
	}
	sortRecords(records)

	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	for _, item := range records {
		if err := encoder.Encode(item); err != nil {
			return err
		}
		counts[item.RecordKind]++
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
	inventorySHA := digest("orquesta.legacy-function-inventory.jsonl.v1", output.Bytes())
	manifestBytes, err := json.MarshalIndent(manifest{
		SchemaVersion:     schemaVersion,
		Algorithm:         "git-objects-go-ast-function-declarations.v1",
		Repository:        repository,
		GitObjectFormat:   objectFormat,
		GoVersion:         strings.TrimSpace(mustCommand("go", "version")),
		InventorySHA256:   inventorySHA,
		Counts:            counts,
		RefSnapshotSHA256: refDigest,
	}, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := writeAtomic(options.jsonl, output.Bytes()); err != nil {
		return err
	}
	return writeAtomic(options.manifest, manifestBytes)
}

func writeAtomic(path string, content []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".legacy-function-inventory-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func mustCommand(name string, arguments ...string) string {
	content, err := exec.Command(name, arguments...).Output()
	if err != nil {
		panic(err)
	}
	return string(content)
}
