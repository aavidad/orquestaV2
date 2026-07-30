// Este fichero transmite registros a un temporal privado y publica el par
// inventario/manifiesto sin mantener la salida completa en memoria.
package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	if err := os.Rename(inventoryTemporary, inventoryPath); err != nil {
		return err
	}
	if err := os.Rename(manifestTemporary, manifestPath); err != nil {
		return err
	}
	return nil
}
