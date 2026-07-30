// Este fichero valida la identidad física de repositorio y salidas.
// Rechaza enlaces en cualquier componente antes de crear o publicar ficheros.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func validateOutputDestinations(repository, jsonl, manifestPath string) error {
	if sameFileName(jsonl, manifestPath) {
		return errors.New("--jsonl y --manifest deben señalar ficheros distintos")
	}
	for _, destination := range []string{jsonl, manifestPath} {
		if err := preparePhysicalDestination(repository, destination); err != nil {
			return err
		}
	}
	for _, destination := range []string{jsonl, manifestPath} {
		if err := validateOutputLeaf(destination); err != nil {
			return err
		}
	}
	if err := rejectPhysicalAlias(jsonl, manifestPath); err != nil {
		return err
	}
	return nil
}

func preparePhysicalDestination(repository, destination string) error {
	if err := rejectSymbolicOutput(destination); err != nil {
		return err
	}
	directory := filepath.Dir(destination)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	if err := rejectSymbolicOutput(destination); err != nil {
		return err
	}
	repositoryReal, err := filepath.EvalSymlinks(repository)
	if err != nil {
		return fmt.Errorf("resolver repositorio observado: %w", err)
	}
	directoryReal, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return fmt.Errorf("resolver directorio de salida: %w", err)
	}
	destinationReal := filepath.Join(directoryReal, filepath.Base(destination))
	if pathInside(repositoryReal, destinationReal) {
		return fmt.Errorf("la salida física no puede quedar dentro del repositorio histórico: %s", destination)
	}
	return validateOutputLeaf(destination)
}

func validateOutputLeaf(destination string) error {
	if info, err := os.Lstat(destination); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("la salida existente no es un fichero regular: %s", destination)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return rejectSymbolicOutput(destination)
}

func rejectPhysicalAlias(left, right string) error {
	leftDirectory, err := os.Stat(filepath.Dir(left))
	if err != nil {
		return err
	}
	rightDirectory, err := os.Stat(filepath.Dir(right))
	if err != nil {
		return err
	}
	if os.SameFile(leftDirectory, rightDirectory) && filepath.Base(left) == filepath.Base(right) {
		return errors.New("--jsonl y --manifest resuelven al mismo fichero físico")
	}
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo) {
		return errors.New("--jsonl y --manifest son alias físicos del mismo fichero")
	}
	if leftErr != nil && !errors.Is(leftErr, fs.ErrNotExist) {
		return leftErr
	}
	if rightErr != nil && !errors.Is(rightErr, fs.ErrNotExist) {
		return rightErr
	}
	return nil
}

func rejectSymbolicOutput(path string) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	current := filepath.VolumeName(absolute) + string(filepath.Separator)
	relative := strings.TrimPrefix(absolute, current)
	for _, segment := range strings.Split(relative, string(filepath.Separator)) {
		if segment == "" {
			continue
		}
		current = filepath.Join(current, segment)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, fs.ErrNotExist) {
			return nil
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("la salida no puede atravesar un enlace simbólico: %s", current)
		}
	}
	return nil
}

func pathInside(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	return err == nil && (relative == "." ||
		(relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))))
}

func syncParentDirectories(paths ...string) error {
	seen := map[string]struct{}{}
	for _, path := range paths {
		directory := filepath.Dir(path)
		if _, exists := seen[directory]; exists {
			continue
		}
		seen[directory] = struct{}{}
		handle, err := os.Open(directory)
		if err != nil {
			return err
		}
		syncErr := handle.Sync()
		closeErr := handle.Close()
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
