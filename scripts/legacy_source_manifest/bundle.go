// Este fichero valida y sella paquetes Git sin materializar sus objetos.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

func inspectBundle(path string, timeout time.Duration) sourceRecord {
	file, err := os.Open(path)
	if err != nil {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error", ErrorCode: filesystemErrorCode(err),
		})
	}
	var header [16]byte
	readBytes, readErr := io.ReadFull(file, header[:])
	closeErr := file.Close()
	if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) && !errors.Is(readErr, io.EOF) {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error", ErrorCode: filesystemErrorCode(readErr),
		})
	}
	if closeErr != nil {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error", ErrorCode: filesystemErrorCode(closeErr),
		})
	}
	if !bytes.HasPrefix(header[:readBytes], []byte("# v2 git bundle\n")) &&
		!bytes.HasPrefix(header[:readBytes], []byte("# v3 git bundle\n")) {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "excluded", Reason: "invalid_git_bundle",
		})
	}
	before, err := os.Stat(path)
	if err != nil {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error", ErrorCode: filesystemErrorCode(err),
		})
	}
	output, err := runGit(timeout, "", "bundle", "list-heads", path)
	if err != nil {
		return gitInspectionError(path, "git_bundle", err)
	}
	refs, err := parseBundleReferences(output)
	if err != nil {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error", ErrorCode: "git_bundle_invalid_references",
		})
	}
	size, contentSHA, err := hashFile(path)
	if err != nil {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error", ErrorCode: filesystemErrorCode(err),
		})
	}
	finalOutput, err := runGit(timeout, "", "bundle", "list-heads", path)
	if err != nil {
		return gitInspectionError(path, "git_bundle", err)
	}
	finalRefs, err := parseBundleReferences(finalOutput)
	if err != nil {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error", ErrorCode: "git_bundle_invalid_references",
		})
	}
	after, err := os.Stat(path)
	if err != nil {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error", ErrorCode: filesystemErrorCode(err),
		})
	}
	if !equalReferences(refs, finalRefs) || size != before.Size() || size != after.Size() ||
		!before.ModTime().Equal(after.ModTime()) {
		return sealSource(sourceRecord{
			Path: path, Kind: "git_bundle", Status: "error",
			ErrorCode: "git_bundle_changed_during_scan",
		})
	}
	return sealSource(sourceRecord{
		Path:            path,
		Kind:            "git_bundle",
		Status:          "included",
		ObjectFormat:    referenceObjectFormat(refs),
		References:      refs,
		ReferenceSHA256: digestJSON("orquesta.git-references.v1", refs),
		SizeBytes:       size,
		ContentSHA256:   contentSHA,
	})
}

func parseBundleReferences(content []byte) ([]reference, error) {
	var refs []reference
	for _, line := range strings.Split(strings.TrimSuffix(string(content), "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] == "" || fields[1] == "" {
			return nil, errors.New("referencia de bundle inválida")
		}
		refs = append(refs, reference{Name: fields[1], Object: fields[0]})
	}
	sortReferences(refs)
	return refs, nil
}

func hashFile(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	hasher := sha256.New()
	size, copyErr := io.Copy(hasher, file)
	closeErr := file.Close()
	if copyErr != nil {
		return 0, "", copyErr
	}
	if closeErr != nil {
		return 0, "", closeErr
	}
	return size, hex.EncodeToString(hasher.Sum(nil)), nil
}
