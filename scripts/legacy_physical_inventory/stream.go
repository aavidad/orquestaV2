// Este fichero limita el flujo JSONL y define el dominio exacto de sus sellos.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
)

const terminalOutputReserve int64 = 4 << 10

type recordWriter struct {
	file     *os.File
	maxBytes int64
	written  int64
}

func (writer *recordWriter) write(value record) error {
	return writer.writeWithin(value, writer.maxBytes-terminalOutputReserve)
}
func (writer *recordWriter) writeTerminal(value record) error {
	return writer.writeWithin(value, writer.maxBytes)
}
func (writer *recordWriter) writeWithin(value record, limit int64) error {
	content, err := json.Marshal(value)
	if err != nil {
		return err
	}
	required := int64(len(content) + 1)
	if writer.written+required > limit {
		return errOutputBudget
	}
	content = append(content, '\n')
	count, err := writer.file.Write(content)
	writer.written += int64(count)
	if err != nil {
		return err
	}
	if count != len(content) {
		return io.ErrShortWrite
	}
	return nil
}
func buildManifest(opts options, state *scanState, publication *publication) (manifest, error) {
	stat, err := publication.jsonl.Stat()
	if err != nil {
		return manifest{}, err
	}
	digest, err := digestOpenFile(publication.jsonl)
	if err != nil {
		return manifest{}, err
	}
	deniedValues := make([]deniedManifest, 0)
	for _, root := range opts.roots {
		for _, parts := range root.denied {
			deniedValues = append(deniedValues, deniedManifest{Root: root.alias, Path: encodePath(parts)})
		}
	}
	sort.Slice(deniedValues, func(i, j int) bool {
		left, _ := json.Marshal(deniedValues[i])
		right, _ := json.Marshal(deniedValues[j])
		return string(left) < string(right)
	})
	return manifest{
		Schema: schemaVersion, Algorithm: "sha256", Complete: state.complete,
		JSONLSHA256: digest, JSONLBytes: stat.Size(), JSONLName: publication.jsonlFinal,
		Entries: state.entries, HashedBytes: state.hashedBytes,
		MaxEntries: opts.budget.maxEntries, MaxDirectoryEntries: opts.budget.maxDirectoryEntries,
		MaxDepth: opts.budget.maxDepth, MaxPathBytes: opts.budget.maxPathBytes,
		MaxOutputBytes: opts.budget.maxOutputBytes,
		MaxHashBytes:   opts.budget.maxHashBytes, MaxFileBytes: opts.budget.maxFileBytes,
		TimeoutNanoseconds: opts.budget.timeout.Nanoseconds(), Roots: state.rootSummaries,
		Denied: deniedValues, LocalIdentityScope: "local_nonportable_device_inode_digest",
		ManifestDigestDomain: manifestDigestDomain,
	}, nil
}
func digestOpenFile(file *os.File) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func sealManifest(value manifest) (manifest, error) {
	value.ManifestSHA256 = ""
	content, err := json.Marshal(value)
	if err != nil {
		return manifest{}, err
	}
	sum := sha256.Sum256(content)
	value.ManifestSHA256 = hex.EncodeToString(sum[:])
	return value, nil
}
func verifyPublishedPair(content []byte, jsonl *os.File, expectedName string) error {
	var value manifest
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return err
	}
	if value.Schema != schemaVersion || value.Algorithm != "sha256" ||
		value.ManifestDigestDomain != manifestDigestDomain {
		return errInvalidManifestSeal
	}
	seal := value.ManifestSHA256
	sealed, err := sealManifest(value)
	if err != nil || sealed.ManifestSHA256 != seal {
		return errInvalidManifestSeal
	}
	if value.JSONLName != filepath.Base(expectedName) {
		return errInvalidManifestSeal
	}
	digest, err := digestOpenFile(jsonl)
	if err != nil {
		return err
	}
	stat, err := jsonl.Stat()
	if err != nil {
		return err
	}
	if digest != value.JSONLSHA256 || stat.Size() != value.JSONLBytes {
		return errInvalidManifestSeal
	}
	return nil
}
func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("unexpected_json_value")
	}
	return err
}
func readLimited(file *os.File, limit int64) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, errors.New("artifact_too_large")
	}
	return content, nil
}
