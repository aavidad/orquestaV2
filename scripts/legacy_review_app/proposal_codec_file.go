// Este fichero materializa el historial privado y durable por orden del almacén.
// No decide propuestas: limita líneas, verifica identidad y sincroniza publicación.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type proposalDirectorySync func(string) error

func readProposals(filePath string) ([]proposal, error) {
	pathInfo, err := os.Lstat(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !pathInfo.Mode().IsRegular() || pathInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("la ruta no es un fichero regular")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(pathInfo, openedInfo) {
		return nil, errors.New("el historial de propuestas cambió al abrirlo")
	}
	if openedInfo.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("el historial de propuestas no es privado")
	}
	reader := bufio.NewReaderSize(file, 64*1024)
	var result []proposal
	for lineNumber := 1; ; lineNumber++ {
		raw, end, err := readProposalLine(reader, maxProposalLineBytes)
		if err != nil {
			return nil, fmt.Errorf("propuestas línea %d: %w", lineNumber, err)
		}
		if end {
			break
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			return nil, fmt.Errorf("propuestas línea %d: registro vacío no canónico", lineNumber)
		}
		item, err := decodeCanonicalProposal(raw)
		if err != nil {
			return nil, fmt.Errorf("propuestas línea %d: %w", lineNumber, err)
		}
		result = append(result, item)
	}
	if err := verifyProposalHistoryIdentity(filePath, file, openedInfo); err != nil {
		return nil, err
	}
	return result, validateStoredProposals(result)
}

func readProposalLine(reader *bufio.Reader, limit int) ([]byte, bool, error) {
	var line []byte
	for {
		fragment, err := reader.ReadSlice('\n')
		hasLF := len(fragment) > 0 && fragment[len(fragment)-1] == '\n'
		if hasLF {
			fragment = fragment[:len(fragment)-1]
		}
		if len(line)+len(fragment) > limit {
			return nil, false, fmt.Errorf("registro supera %d bytes", limit)
		}
		line = append(line, fragment...)
		if hasLF {
			return line, false, nil
		}
		switch {
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF) && len(line) == 0:
			return nil, true, nil
		case errors.Is(err, io.EOF):
			return nil, false, errors.New("falta LF final")
		case err != nil:
			return nil, false, err
		default:
			return nil, false, errors.New("lectura de línea incompleta")
		}
	}
}

func verifyProposalHistoryIdentity(filePath string, file *os.File, openedInfo os.FileInfo) error {
	finalInfo, finalErr := file.Stat()
	pathInfo, pathErr := os.Lstat(filePath)
	if finalErr != nil || pathErr != nil || !finalInfo.Mode().IsRegular() ||
		!pathInfo.Mode().IsRegular() || pathInfo.Mode()&os.ModeSymlink != 0 ||
		finalInfo.Mode().Perm()&0o077 != 0 || pathInfo.Mode().Perm()&0o077 != 0 ||
		!os.SameFile(openedInfo, finalInfo) || !os.SameFile(openedInfo, pathInfo) ||
		finalInfo.Size() != openedInfo.Size() {
		return errors.New("el historial de propuestas cambió durante la lectura")
	}
	return nil
}

func writeProposals(filePath string, proposals []proposal) error {
	return writeProposalsWithDirectorySync(filePath, proposals, syncProposalDirectory)
}

func writeProposalsWithDirectorySync(
	filePath string,
	proposals []proposal,
	syncDirectory proposalDirectorySync,
) error {
	if err := rejectSymlink(filePath); err != nil {
		return err
	}
	directory := filepath.Dir(filePath)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(filePath)+".tmp-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	for _, item := range proposals {
		line, err := encodeCanonicalProposal(item)
		if err != nil {
			temporary.Close()
			return err
		}
		if len(line) > maxProposalLineBytes {
			temporary.Close()
			return fmt.Errorf("registro supera %d bytes", maxProposalLineBytes)
		}
		if _, err := temporary.Write(append(line, '\n')); err != nil {
			temporary.Close()
			return err
		}
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, filePath); err != nil {
		return err
	}
	if err := syncDirectory(directory); err != nil {
		return fmt.Errorf("sincronizar directorio de propuestas: %w", err)
	}
	return nil
}

func syncProposalDirectory(directory string) error {
	file, err := os.Open(directory)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.IsDir() {
		return errors.New("directorio de propuestas inválido")
	}
	return file.Sync()
}
