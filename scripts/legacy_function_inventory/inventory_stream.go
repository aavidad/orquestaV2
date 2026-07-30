// Este fichero transmite registros a un temporal privado y publica el par
// inventario/manifiesto sin mantener la salida completa en memoria.
package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
)

type inventoryStream struct {
	file          *os.File
	writer        *bufio.Writer
	hasher        hash.Hash
	counts        map[string]int
	bytes         int64
	temporaryPath string
	closed        bool
	published     bool
}

type publicationOps struct {
	rename func(string, string) error
	link   func(string, string) error
	remove func(string) error
}

var operatingSystemPublication = publicationOps{
	rename: os.Rename,
	link:   os.Link,
	remove: os.Remove,
}

func newInventoryStream(finalPath string) (*inventoryStream, error) {
	temporary, err := createTemporary(finalPath)
	if err != nil {
		return nil, err
	}
	return &inventoryStream{
		file: temporary, writer: bufio.NewWriterSize(temporary, 256*1024),
		hasher: newInventoryHash(), counts: make(map[string]int),
		temporaryPath: temporary.Name(),
	}, nil
}

func (stream *inventoryStream) emit(item record) error {
	if stream.closed {
		return errors.New("inventario progresivo cerrado")
	}
	var line bytes.Buffer
	encoder := json.NewEncoder(&line)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(item); err != nil {
		return err
	}
	content := line.Bytes()
	if _, err := stream.writer.Write(content); err != nil {
		return err
	}
	if _, err := stream.hasher.Write(content); err != nil {
		return err
	}
	stream.bytes += int64(len(content))
	stream.counts[item.RecordKind]++
	return nil
}

func (stream *inventoryStream) finish() (string, int64, map[string]int, error) {
	if stream.closed {
		return "", 0, nil, errors.New("inventario progresivo ya cerrado")
	}
	stream.closed = true
	if err := stream.writer.Flush(); err != nil {
		_ = stream.file.Close()
		return "", 0, nil, err
	}
	if err := stream.file.Sync(); err != nil {
		_ = stream.file.Close()
		return "", 0, nil, err
	}
	if err := stream.file.Close(); err != nil {
		return "", 0, nil, err
	}
	counts := make(map[string]int, len(stream.counts))
	for kind, count := range stream.counts {
		counts[kind] = count
	}
	return "sha256:" + hex.EncodeToString(stream.hasher.Sum(nil)), stream.bytes, counts, nil
}

func (stream *inventoryStream) abort() {
	if stream == nil || stream.published {
		return
	}
	if !stream.closed {
		_ = stream.file.Close()
		stream.closed = true
	}
	_ = os.Remove(stream.temporaryPath)
}

func writeTemporary(finalPath string, content []byte) (string, error) {
	temporary, err := createTemporary(finalPath)
	if err != nil {
		return "", err
	}
	path := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(path)
	}
	if _, err := temporary.Write(content); err != nil {
		cleanup()
		return "", err
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func createTemporary(finalPath string) (*os.File, error) {
	if info, err := os.Lstat(finalPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("la salida no puede reemplazar un enlace simbólico")
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	directory := filepath.Dir(finalPath)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	temporary, err := os.CreateTemp(directory, ".legacy-function-inventory-*")
	if err != nil {
		return nil, err
	}
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		_ = os.Remove(temporary.Name())
		return nil, err
	}
	return temporary, nil
}

func publishPair(inventoryTemporary, inventoryPath, manifestTemporary, manifestPath string) error {
	return publishPairWithOps(
		inventoryTemporary,
		inventoryPath,
		manifestTemporary,
		manifestPath,
		operatingSystemPublication,
	)
}

func publishPairWithOps(
	inventoryTemporary string,
	inventoryPath string,
	manifestTemporary string,
	manifestPath string,
	operations publicationOps,
) error {
	backupPath := manifestTemporary + ".previous"
	hadManifest, err := backupManifest(manifestPath, backupPath, operations)
	if err != nil {
		return err
	}
	if err := operations.rename(manifestTemporary, manifestPath); err != nil {
		cleanupErr := removeIfPresent(backupPath, operations)
		return errors.Join(err, cleanupErr)
	}
	if err := operations.rename(inventoryTemporary, inventoryPath); err != nil {
		rollbackErr := rollbackManifest(manifestPath, backupPath, hadManifest, operations)
		if rollbackErr != nil {
			return fmt.Errorf("publicar inventario: %w; restaurar manifiesto: %v; respaldo: %s",
				err, rollbackErr, backupPath)
		}
		return err
	}
	if err := removeIfPresent(backupPath, operations); err != nil {
		return err
	}
	return nil
}

func backupManifest(path, backupPath string, operations publicationOps) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, errors.New("el manifiesto final existente no es un fichero regular")
	}
	if err := operations.link(path, backupPath); err != nil {
		return false, err
	}
	return true, nil
}

func rollbackManifest(
	manifestPath string,
	backupPath string,
	hadManifest bool,
	operations publicationOps,
) error {
	if hadManifest {
		return operations.rename(backupPath, manifestPath)
	}
	return removeIfPresent(manifestPath, operations)
}

func removeIfPresent(path string, operations publicationOps) error {
	err := operations.remove(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
