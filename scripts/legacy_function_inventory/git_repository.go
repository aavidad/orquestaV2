// Este fichero reconstruye referencias, grafo y árboles desde objetos Git.
// Usa el lector acotado del paquete, pero no decide cómo serializar ni admitir
// las conductas históricas encontradas.
package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type treeEntry struct {
	Mode string
	Type string
	OID  string
	Path []byte
}

type historyEntry struct {
	tree    string
	parents []string
}

func readHistory(
	repository string,
	refs []refInfo,
	processStarted func(),
) (map[string]historyEntry, error) {
	roots := make([]string, 0, len(refs))
	seen := make(map[string]struct{})
	for _, ref := range refs {
		if ref.Commit == "" {
			continue
		}
		if _, found := seen[ref.Commit]; found {
			continue
		}
		seen[ref.Commit] = struct{}{}
		roots = append(roots, ref.Commit)
	}
	sort.Strings(roots)
	var input bytes.Buffer
	for _, root := range roots {
		input.WriteString(root)
		input.WriteByte('\n')
	}
	content, err := gitBytesInput(
		repository,
		processStarted,
		input.Bytes(),
		"rev-list",
		"--format=%H%x00%T%x00%P",
		"--no-commit-header",
		"--stdin",
	)
	if err != nil {
		return nil, err
	}
	result := make(map[string]historyEntry)
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return result, nil
	}
	for _, line := range bytes.Split(trimmed, []byte{'\n'}) {
		fields := bytes.Split(line, []byte{0})
		if len(fields) != 3 {
			return nil, errors.New("entrada inválida en el grafo histórico")
		}
		commit := string(fields[0])
		tree := string(fields[1])
		if commit == "" || tree == "" {
			return nil, errors.New("confirmación histórica sin identificador o árbol")
		}
		var parents []string
		if len(fields[2]) > 0 {
			parents = strings.Fields(string(fields[2]))
		}
		if previous, found := result[commit]; found &&
			(previous.tree != tree || !equalStrings(previous.parents, parents)) {
			return nil, fmt.Errorf("datos contradictorios para la confirmación %s", commit)
		}
		result[commit] = historyEntry{tree: tree, parents: parents}
	}
	return result, nil
}

func reachableCommits(root string, history map[string]historyEntry) ([]string, error) {
	if _, found := history[root]; !found {
		return nil, fmt.Errorf("la referencia apunta a una confirmación no censada: %s", root)
	}
	seen := make(map[string]struct{})
	pending := []string{root}
	for len(pending) > 0 {
		last := len(pending) - 1
		commit := pending[last]
		pending = pending[:last]
		if _, found := seen[commit]; found {
			continue
		}
		entry, found := history[commit]
		if !found {
			return nil, fmt.Errorf("falta el ancestro %s de la confirmación %s", commit, root)
		}
		seen[commit] = struct{}{}
		pending = append(pending, entry.parents...)
	}
	result := make([]string, 0, len(seen))
	for commit := range seen {
		result = append(result, commit)
	}
	sort.Strings(result)
	return result, nil
}

func readTreeObject(batch *gitBatch, treeID string, objectBytes int) ([]treeEntry, error) {
	object, err := batch.get(treeID)
	if err != nil {
		return nil, err
	}
	if object.kind != "tree" {
		return nil, fmt.Errorf("objeto %s no es un árbol", treeID)
	}
	result, err := parseTree(object.content, objectBytes)
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		if comparison := bytes.Compare(result[i].Path, result[j].Path); comparison != 0 {
			return comparison < 0
		}
		return result[i].OID < result[j].OID
	})
	return result, nil
}

func parseTree(content []byte, objectBytes int) ([]treeEntry, error) {
	var result []treeEntry
	for len(content) > 0 {
		modeEnd := bytes.IndexByte(content, ' ')
		if modeEnd <= 0 {
			return nil, errors.New("modo ausente en entrada de árbol")
		}
		nameEnd := bytes.IndexByte(content[modeEnd+1:], 0)
		if nameEnd < 0 {
			return nil, errors.New("nombre sin terminador en entrada de árbol")
		}
		nameEnd += modeEnd + 1
		objectStart := nameEnd + 1
		objectEnd := objectStart + objectBytes
		if objectEnd > len(content) {
			return nil, errors.New("identificador truncado en entrada de árbol")
		}
		mode := string(content[:modeEnd])
		kind := "blob"
		switch mode {
		case "40000":
			mode = "040000"
			kind = "tree"
		case "160000":
			kind = "commit"
		}
		result = append(result, treeEntry{
			Mode: mode,
			Type: kind,
			OID:  hex.EncodeToString(content[objectStart:objectEnd]),
			Path: bytes.Clone(content[modeEnd+1 : nameEnd]),
		})
		content = content[objectEnd:]
	}
	return result, nil
}

func objectIDBytes(objectFormat string) (int, error) {
	switch strings.TrimSpace(objectFormat) {
	case "sha1":
		return sha1.Size, nil
	case "sha256":
		return sha256.Size, nil
	default:
		return 0, fmt.Errorf("formato de objetos Git no admitido: %q", objectFormat)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
