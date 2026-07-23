package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type prepareGeneratedFile func(path string, body []byte) (string, error)

func writeGenerated(root string, paths []string, outputs map[string][]byte) error {
	return writeGeneratedWith(root, paths, outputs, prepareGeneratedTemp)
}

// writeGeneratedWith separates the fallible preparation phase from commit.
// No destination is renamed until every target has passed preflight and every
// complete, synced temporary file exists beside its destination.
func writeGeneratedWith(root string, paths []string, outputs map[string][]byte, prepare prepareGeneratedFile) (err error) {
	if prepare == nil {
		return errors.New("commandgen.prepare_required")
	}
	root, err = preflightGeneratedRoot(root)
	if err != nil {
		return err
	}
	targets := make(map[string]string, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, relative := range paths {
		clean, cleanErr := cleanGeneratedPath(relative)
		if cleanErr != nil {
			return cleanErr
		}
		if _, duplicate := seen[clean]; duplicate {
			return fmt.Errorf("commandgen.output_duplicate:%s", relative)
		}
		seen[clean] = struct{}{}
		body, exists := outputs[relative]
		if !exists || body == nil {
			return fmt.Errorf("commandgen.output_missing:%s", relative)
		}
		target := filepath.Join(root, clean)
		if err := preflightGeneratedTarget(root, target); err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
		targets[relative] = target
	}
	temporaries := make(map[string]string, len(paths))
	defer func() {
		for _, temporary := range temporaries {
			_ = os.Remove(temporary)
		}
	}()
	for _, relative := range paths {
		temporary, prepareErr := prepare(targets[relative], outputs[relative])
		if prepareErr != nil {
			return fmt.Errorf("%s: %w", relative, prepareErr)
		}
		temporaries[relative] = temporary
	}
	for _, relative := range paths {
		if err := os.Rename(temporaries[relative], targets[relative]); err != nil {
			// Multi-file rename is not crash-atomic. A concurrent filesystem
			// failure can leave generated drift; commandgen --check detects it.
			return fmt.Errorf("%s: %w", relative, err)
		}
		delete(temporaries, relative)
	}
	return nil
}

func preflightGeneratedRoot(root string) (string, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errors.New("commandgen.root_invalid")
	}
	return filepath.Clean(absolute), nil
}

func cleanGeneratedPath(relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", errors.New("commandgen.output_path_invalid")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("commandgen.output_path_invalid")
	}
	return clean, nil
}

func preflightGeneratedTarget(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("commandgen.output_path_invalid")
	}
	current := root
	for _, component := range strings.Split(filepath.Dir(relative), string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		current = filepath.Join(current, component)
		info, statErr := os.Lstat(current)
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("commandgen.parent_invalid")
		}
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("commandgen.target_invalid")
	}
	return nil
}

func prepareGeneratedTemp(path string, body []byte) (temporaryPath string, err error) {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".commandgen-*")
	if err != nil {
		return "", err
	}
	temporaryPath = temporary.Name()
	defer func() {
		_ = temporary.Close()
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err = temporary.Chmod(0o644); err == nil {
		_, err = temporary.Write(body)
	}
	if err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	return temporaryPath, err
}
