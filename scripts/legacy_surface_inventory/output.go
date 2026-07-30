// Este fichero publica cada salida de forma atómica y la pareja como verificable.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode/utf8"
)

func publishInventory(
	selected options,
	repository, objectFormat string,
	refs []refInfo,
	inventoryDigest string,
	inventoryBytes int64,
	counts map[string]int,
	jsonlTemporary string,
) error {
	value := manifest{
		SchemaVersion: schemaVersion, Algorithm: inventoryAlgorithm,
		InventoryHashDomain: inventoryHashDomain, RecordSchemaSHA256: recordSchemaDigest(),
		ClassificationAlgorithm: classificationAlgorithm,
		Repository:              repository, GitObjectFormat: objectFormat,
		InventorySHA256: inventoryDigest, InventoryBytes: inventoryBytes,
		RefSnapshotSHA256: domainDigest(referenceHashDomain, mustJSON(refs)),
		Counts:            counts,
	}
	manifestBytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')

	previousManifest, previousManifestExists, err := readPreviousManifest(selected.manifest)
	if err != nil {
		return err
	}
	manifestTemporary, err := prepareAtomicFile(selected.manifest, manifestBytes)
	if err != nil {
		return err
	}
	defer os.Remove(manifestTemporary)
	rename := selected.renameFile
	if rename == nil {
		rename = os.Rename
	}

	// El manifiesto se publica primero: incluso una caída entre renombrados
	// nunca deja un manifiesto antiguo validando un JSONL nuevo.
	if err := rename(manifestTemporary, selected.manifest); err != nil {
		return fmt.Errorf("publicar manifiesto: %w", err)
	}
	// Este punto inyectable representa una caída abrupta: no revierte porque un
	// proceso terminado tampoco podría hacerlo. La repetición repara la pareja.
	if selected.interruptAfterManifest != nil {
		if err := selected.interruptAfterManifest(); err != nil {
			return fmt.Errorf("interrupción tras publicar manifiesto: %w", err)
		}
	}
	if err := rename(jsonlTemporary, selected.jsonl); err != nil {
		rollbackErr := restorePreviousManifest(
			selected.manifest,
			previousManifest,
			previousManifestExists,
		)
		if rollbackErr != nil {
			return fmt.Errorf("publicar JSONL: %w; invalidar o restaurar manifiesto: %v", err, rollbackErr)
		}
		return fmt.Errorf("publicar JSONL: %w; se restauró el manifiesto anterior", err)
	}
	if err := syncParentDirectories(selected.manifest, selected.jsonl); err != nil {
		return fmt.Errorf("sincronizar publicación: %w", err)
	}
	return nil
}

func readPreviousManifest(path string) ([]byte, bool, error) {
	if err := rejectSymbolicOutput(path); err != nil {
		return nil, false, err
	}
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("leer manifiesto anterior: %w", err)
	}
	return content, true, nil
}

func restorePreviousManifest(path string, content []byte, existed bool) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !existed {
		return syncParentDirectories(path)
	}
	temporary, err := prepareAtomicFile(path, content)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return syncParentDirectories(path)
}

func newInventoryWriter(destination string) (*inventoryWriter, error) {
	if err := rejectSymbolicOutput(destination); err != nil {
		return nil, err
	}
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(directory, "."+filepath.Base(destination)+".tmp-*")
	if err != nil {
		return nil, err
	}
	writer := &inventoryWriter{
		file:   file,
		name:   file.Name(),
		hasher: newInventoryHash(),
		counts: map[string]int{},
	}
	if err := file.Chmod(0o600); err != nil {
		writer.abort()
		return nil, err
	}
	encoder := json.NewEncoder(io.MultiWriter(file, writer.hasher))
	encoder.SetEscapeHTML(false)
	writer.encoder = encoder
	return writer, nil
}

func (writer *inventoryWriter) write(item record) error {
	if writer.sealed {
		return fmt.Errorf("la salida temporal ya está sellada")
	}
	if err := writer.encoder.Encode(item); err != nil {
		return err
	}
	writer.counts[item.RecordKind]++
	if item.RecordKind == "hechos_blob" {
		for _, family := range item.Families {
			writer.counts["familia:"+family.Family]++
		}
	}
	if item.RecordKind == "entrada_arbol" && item.ExclusionCause != "" {
		writer.counts["exclusion:"+item.ExclusionCause]++
		if item.ReviewObligation != "" {
			writer.counts["obligacion:"+item.ReviewObligation]++
		}
	}
	return nil
}

func (writer *inventoryWriter) seal() (string, int64, map[string]int, string, error) {
	if writer.sealed {
		return "", 0, nil, "", fmt.Errorf("la salida temporal ya está sellada")
	}
	writer.sealed = true
	if err := writer.file.Sync(); err != nil {
		return "", 0, nil, "", err
	}
	info, err := writer.file.Stat()
	if err != nil {
		return "", 0, nil, "", err
	}
	if err := writer.file.Close(); err != nil {
		return "", 0, nil, "", err
	}
	return hex.EncodeToString(writer.hasher.Sum(nil)), info.Size(), writer.counts, writer.name, nil
}

func (writer *inventoryWriter) abort() {
	if writer == nil {
		return
	}
	if writer.file != nil {
		_ = writer.file.Close()
	}
	_ = os.Remove(writer.name)
}

func exclusionCause(filePath []byte, childType string) string {
	if childType != "tree" {
		return ""
	}
	for _, segment := range bytes.Split(filePath, []byte{'/'}) {
		switch {
		case bytes.EqualFold(segment, []byte("vendor")):
			return "dependencia_vendorizada_vendor"
		case bytes.EqualFold(segment, []byte("node_modules")):
			return "dependencia_vendorizada_node_modules"
		}
	}
	return ""
}

func displayGitPath(raw []byte) (string, string, string) {
	if utf8.Valid(raw) {
		return "utf8", string(raw), ""
	}
	return "base64", "", base64.StdEncoding.EncodeToString(raw)
}

func joinGitPath(prefix, name []byte) []byte {
	if len(prefix) == 0 {
		return append([]byte(nil), name...)
	}
	result := make([]byte, 0, len(prefix)+1+len(name))
	result = append(result, prefix...)
	result = append(result, '/')
	result = append(result, name...)
	return result
}

func stableRef(namespace, value string) string {
	digest := sha256.Sum256([]byte(namespace + "\x00" + value))
	return namespace + ":sha256:" + hex.EncodeToString(digest[:])
}

func mustJSON(value any) []byte {
	content, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return content
}

func equalRefs(left, right []refInfo) bool {
	return bytes.Equal(mustJSON(left), mustJSON(right))
}

func sameFileName(left, right string) bool {
	leftAbsolute, leftErr := filepath.Abs(left)
	rightAbsolute, rightErr := filepath.Abs(right)
	return leftErr == nil && rightErr == nil && leftAbsolute == rightAbsolute
}

func prepareAtomicFile(destination string, content []byte) (string, error) {
	if err := rejectSymbolicOutput(destination); err != nil {
		return "", err
	}
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(destination)+".tmp-*")
	if err != nil {
		return "", err
	}
	name := temporary.Name()
	ok := false
	defer func() {
		if !ok {
			temporary.Close()
			os.Remove(name)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return "", err
	}
	if _, err := temporary.Write(content); err != nil {
		return "", err
	}
	if err := temporary.Sync(); err != nil {
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	ok = true
	return name, nil
}

func newInventoryHash() hash.Hash {
	hashValue := sha256.New()
	writeHashField(hashValue, []byte(inventoryHashDomain))
	return hashValue
}

func inventoryDigest(content []byte) string {
	hashValue := newInventoryHash()
	_, _ = hashValue.Write(content)
	return hex.EncodeToString(hashValue.Sum(nil))
}

func domainDigest(domain string, content []byte) string {
	hashValue := sha256.New()
	writeHashField(hashValue, []byte(domain))
	writeHashField(hashValue, content)
	return hex.EncodeToString(hashValue.Sum(nil))
}

func writeHashField(hashValue hash.Hash, content []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(content)))
	_, _ = hashValue.Write(length[:])
	_, _ = hashValue.Write(content)
}

func recordSchemaDigest() string {
	recordType := reflect.TypeOf(record{})
	fields := make([]string, 0, recordType.NumField())
	for index := 0; index < recordType.NumField(); index++ {
		field := recordType.Field(index)
		fields = append(fields, field.Name+"\x00"+string(field.Tag)+"\x00"+field.Type.String())
	}
	return domainDigest(recordSchemaDomain, mustJSON(struct {
		Kinds  []string `json:"kinds"`
		Fields []string `json:"fields"`
	}{Kinds: recordKinds, Fields: fields}))
}

func boundedDetail(value string) string {
	value = strings.TrimSpace(value)
	const limit = 512
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}
