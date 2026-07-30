// Este fichero captura referencias Git y HEAD como hechos binarios estables.
// Conserva estado simbólico, destino y OID sin volver a resolver nombres vivos.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type refInfo struct {
	Name   []byte `json:"name_bytes"`
	Mode   string `json:"mode"`
	Target []byte `json:"target_bytes,omitempty"`
	Object string `json:"object_id,omitempty"`
	Commit string `json:"commit_id,omitempty"`
}

type rawRef struct {
	name   []byte
	mode   string
	target []byte
	object []byte
}

func readRefs(repository string, processStarted func()) ([]refInfo, error) {
	content, err := gitBytes(
		repository,
		processStarted,
		"for-each-ref",
		"--format=%(refname)%00%(objectname)%00%(symref)",
		"refs",
	)
	if err != nil {
		return nil, err
	}
	rawRefs, err := parseRawRefs(content)
	if err != nil {
		return nil, err
	}
	head, err := readHead(repository, processStarted)
	if err != nil {
		return nil, err
	}
	batch, err := newGitBatch(repository, processStarted)
	if err != nil {
		return nil, err
	}
	defer batch.close()
	result := make([]refInfo, 0, len(rawRefs)+1)
	for _, raw := range rawRefs {
		ref, resolveErr := resolveRef(raw, batch)
		if resolveErr != nil {
			return nil, resolveErr
		}
		result = append(result, ref)
	}
	headRef, err := resolveHead(head, result, batch)
	if err != nil {
		return nil, err
	}
	result = append(result, headRef)
	sort.Slice(result, func(i, j int) bool {
		return bytes.Compare(result[i].Name, result[j].Name) < 0
	})
	if err := batch.close(); err != nil {
		return nil, err
	}
	return result, nil
}

func parseRawRefs(content []byte) ([]rawRef, error) {
	content = bytes.TrimSuffix(content, []byte{'\n'})
	if len(content) == 0 {
		return nil, nil
	}
	lines := bytes.Split(content, []byte{'\n'})
	result := make([]rawRef, 0, len(lines))
	for _, line := range lines {
		fields := bytes.Split(line, []byte{0})
		if len(fields) != 3 || len(fields[0]) == 0 {
			return nil, fmt.Errorf("referencia Git binaria inválida: %q", line)
		}
		mode := "direct"
		if len(fields[2]) > 0 {
			mode = "symbolic"
		}
		result = append(result, rawRef{
			name: bytes.Clone(fields[0]), mode: mode,
			target: bytes.Clone(fields[2]), object: bytes.Clone(fields[1]),
		})
	}
	return result, nil
}

func readHead(repository string, processStarted func()) (rawRef, error) {
	gitDirectory, err := gitBytes(repository, processStarted, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return rawRef{}, err
	}
	gitDirectory, err = singleGitLine(gitDirectory)
	if err != nil {
		return rawRef{}, err
	}
	content, err := os.ReadFile(filepath.Join(string(gitDirectory), "HEAD"))
	if err != nil {
		return rawRef{}, err
	}
	content = bytes.TrimSuffix(content, []byte{'\n'})
	content = bytes.TrimSuffix(content, []byte{'\r'})
	head := rawRef{name: []byte("HEAD"), mode: "direct", object: bytes.Clone(content)}
	if bytes.HasPrefix(content, []byte("ref: ")) {
		head.mode = "symbolic"
		head.target = bytes.Clone(bytes.TrimPrefix(content, []byte("ref: ")))
		head.object = nil
		if len(head.target) == 0 {
			return rawRef{}, errors.New("HEAD simbólico sin destino")
		}
	}
	return head, nil
}

func singleGitLine(content []byte) ([]byte, error) {
	content = bytes.TrimSuffix(content, []byte{'\n'})
	if len(content) == 0 || bytes.ContainsAny(content, "\r\n\x00") {
		return nil, errors.New("Git devolvió una ruta interna inválida")
	}
	return content, nil
}

func resolveHead(head rawRef, refs []refInfo, batch *gitBatch) (refInfo, error) {
	if head.mode == "direct" {
		return resolveRef(head, batch)
	}
	result := refInfo{
		Name: bytes.Clone(head.name), Mode: head.mode, Target: bytes.Clone(head.target),
	}
	for _, ref := range refs {
		if bytes.Equal(ref.Name, head.target) {
			result.Object = ref.Object
			result.Commit = ref.Commit
			break
		}
	}
	return result, nil
}

func resolveRef(raw rawRef, batch *gitBatch) (refInfo, error) {
	result := refInfo{
		Name: bytes.Clone(raw.name), Mode: raw.mode, Target: bytes.Clone(raw.target),
	}
	if len(raw.object) == 0 {
		if raw.mode == "symbolic" {
			return result, nil
		}
		return refInfo{}, fmt.Errorf("referencia %q sin objeto", raw.name)
	}
	objectID := string(raw.object)
	if !validObjectID(objectID) {
		return refInfo{}, fmt.Errorf("referencia %q con OID inválido", raw.name)
	}
	object, err := batch.get(objectID)
	if errors.Is(err, errGitObjectMissing) {
		return refInfo{}, fmt.Errorf("la referencia %q apunta al objeto ausente %s", raw.name, objectID)
	}
	if err != nil {
		return refInfo{}, err
	}
	result.Object = object.oid
	result.Commit, err = peelCommit(raw.name, object, batch, make(map[string]struct{}))
	if err != nil {
		return refInfo{}, err
	}
	return result, nil
}

func validObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return false
		}
	}
	return true
}

func peelCommit(
	refName []byte,
	object gitObject,
	batch *gitBatch,
	seen map[string]struct{},
) (string, error) {
	if _, found := seen[object.oid]; found {
		return "", fmt.Errorf("ciclo de etiquetas en %q", refName)
	}
	seen[object.oid] = struct{}{}
	switch object.kind {
	case "commit":
		return object.oid, nil
	case "tag":
		targetOID, targetKind, err := tagTarget(object.content)
		if err != nil {
			return "", fmt.Errorf("etiqueta inválida en %q: %w", refName, err)
		}
		target, err := batch.get(targetOID)
		if errors.Is(err, errGitObjectMissing) {
			return "", fmt.Errorf("la referencia %q alcanza el objeto ausente %s", refName, targetOID)
		}
		if err != nil {
			return "", err
		}
		if target.kind != targetKind {
			return "", fmt.Errorf(
				"la etiqueta de %q declara %s para un objeto %s",
				refName, targetKind, target.kind,
			)
		}
		return peelCommit(refName, target, batch, seen)
	default:
		return "", nil
	}
}

func tagTarget(content []byte) (string, string, error) {
	lines := bytes.Split(content, []byte{'\n'})
	if len(lines) < 2 ||
		!bytes.HasPrefix(lines[0], []byte("object ")) ||
		!bytes.HasPrefix(lines[1], []byte("type ")) {
		return "", "", errors.New("cabecera object/type ausente")
	}
	objectID := string(bytes.TrimPrefix(lines[0], []byte("object ")))
	kind := string(bytes.TrimPrefix(lines[1], []byte("type ")))
	if !validObjectID(objectID) || kind == "" || strings.ContainsAny(kind, " \t\r\n") {
		return "", "", errors.New("destino object/type inválido")
	}
	return objectID, kind, nil
}

func equalRefs(left, right []refInfo) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !bytes.Equal(left[index].Name, right[index].Name) ||
			left[index].Mode != right[index].Mode ||
			!bytes.Equal(left[index].Target, right[index].Target) ||
			left[index].Object != right[index].Object ||
			left[index].Commit != right[index].Commit {
			return false
		}
	}
	return true
}
