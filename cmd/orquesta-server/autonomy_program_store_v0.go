package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	autonomy "orquesta/modulos/orquesta-autonomy-program"
)

// El programa de autonomia es un grafo con estado: si dos planificadores lo
// escribieran a la vez sin CAS, uno pisaria el avance del otro y se relanzarian
// nodos ya lanzados. Por eso el dominio EXIGE CompareAndSwap, y por eso este
// almacen lo implementa de verdad y no con un Save que sobrescribe.
const autonomyProgramsDirNameV0 = "autonomy-programs"

var (
	ErrAutonomyProgramNotFoundV0 = errors.New("autonomy_program_no_encontrado")
	ErrAutonomyProgramCorruptV0  = errors.New("autonomy_program_corrupto")
	ErrAutonomyProgramConflictV0 = errors.New("autonomy_program_conflicto")
)

type autonomyProgramStoreV0 struct {
	dir string
}

var _ autonomy.AutonomyProgramStorePortV0 = autonomyProgramStoreV0{}

func newAutonomyProgramStoreV0(stateDir string) (autonomyProgramStoreV0, error) {
	dir := filepath.Join(stateDir, autonomyProgramsDirNameV0)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return autonomyProgramStoreV0{}, fmt.Errorf("autonomy programs: %w", err)
	}
	return autonomyProgramStoreV0{dir: dir}, nil
}

// pathV0 identifica el programa por proyecto+raiz+programa. Los tres vienen de
// fuera, asi que los tres se sanean.
func (store autonomyProgramStoreV0) pathV0(projectRef, rootRef, programRef string) (string, error) {
	partes := []string{projectRef, rootRef, programRef}
	for _, parte := range partes {
		ref := strings.TrimSpace(parte)
		if ref == "" {
			return "", fmt.Errorf("autonomy_program_ref_requerido")
		}
		if strings.ContainsAny(ref, "/\\") || ref == "." || ref == ".." {
			return "", fmt.Errorf("autonomy_program_ref_invalido: %s", ref)
		}
	}
	nombre := strings.TrimSpace(projectRef) + "__" + strings.TrimSpace(rootRef) + "__" + strings.TrimSpace(programRef)
	return filepath.Join(store.dir, nombre+".json"), nil
}

func (store autonomyProgramStoreV0) loadV0(path string) (autonomy.AutonomyProgramV0, bool, error) {
	bytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return autonomy.AutonomyProgramV0{}, false, nil
	}
	if err != nil {
		return autonomy.AutonomyProgramV0{}, false, fmt.Errorf("%w: %v", ErrAutonomyProgramCorruptV0, err)
	}
	var program autonomy.AutonomyProgramV0
	if err := json.Unmarshal(bytes, &program); err != nil {
		return autonomy.AutonomyProgramV0{}, false, fmt.Errorf("%w: %s", ErrAutonomyProgramCorruptV0, path)
	}
	return program, true, nil
}

func (store autonomyProgramStoreV0) writeV0(path string, program autonomy.AutonomyProgramV0) error {
	bytes, err := json.MarshalIndent(program, "", "  ")
	if err != nil {
		return err
	}
	temporal, err := os.CreateTemp(store.dir, "program-*.tmp")
	if err != nil {
		return err
	}
	temporalPath := temporal.Name()
	defer os.Remove(temporalPath)
	if _, err := temporal.Write(bytes); err != nil {
		temporal.Close()
		return err
	}
	if err := temporal.Chmod(0o600); err != nil {
		temporal.Close()
		return err
	}
	if err := temporal.Close(); err != nil {
		return err
	}
	return os.Rename(temporalPath, path)
}

// SaveAutonomyProgramV0 crea el agregado o acepta un reintento IDENTICO. Si ya
// existe con otra topologia, choca: guardar encima seria perder el avance.
func (store autonomyProgramStoreV0) SaveAutonomyProgramV0(
	_ context.Context,
	program autonomy.AutonomyProgramV0,
) error {
	path, err := store.pathV0(program.ProjectRef, program.RootRef, program.ProgramRef)
	if err != nil {
		return err
	}
	existente, existe, err := store.loadV0(path)
	if err != nil {
		return err
	}
	if existe {
		if autonomy.SameAutonomyProgramTopologyV0(existente, program) {
			return nil
		}
		return fmt.Errorf("%w: %s", ErrAutonomyProgramConflictV0, program.ProgramRef)
	}
	return store.writeV0(path, program)
}

func (store autonomyProgramStoreV0) LoadAutonomyProgramV0(
	_ context.Context,
	projectRef string,
	rootRef string,
	programRef string,
) (autonomy.AutonomyProgramV0, error) {
	path, err := store.pathV0(projectRef, rootRef, programRef)
	if err != nil {
		return autonomy.AutonomyProgramV0{}, err
	}
	program, existe, err := store.loadV0(path)
	if err != nil {
		return autonomy.AutonomyProgramV0{}, err
	}
	if !existe {
		return autonomy.AutonomyProgramV0{}, fmt.Errorf("%w: %s", ErrAutonomyProgramNotFoundV0, programRef)
	}
	return program, nil
}

// CompareAndSwapAutonomyProgramV0 solo escribe si lo que hay en disco es EXACTAMENTE
// lo que el llamante creia. Sin esto, dos planificadores concurrentes relanzarian
// nodos ya lanzados.
func (store autonomyProgramStoreV0) CompareAndSwapAutonomyProgramV0(
	_ context.Context,
	expected autonomy.AutonomyProgramV0,
	next autonomy.AutonomyProgramV0,
) (bool, error) {
	path, err := store.pathV0(next.ProjectRef, next.RootRef, next.ProgramRef)
	if err != nil {
		return false, err
	}
	actual, existe, err := store.loadV0(path)
	if err != nil {
		return false, err
	}
	if !existe {
		return false, fmt.Errorf("%w: %s", ErrAutonomyProgramNotFoundV0, next.ProgramRef)
	}
	iguales, err := mismoProgramaV0(actual, expected)
	if err != nil {
		return false, err
	}
	if !iguales {
		// No es un error: es que otro planificador avanzo primero. El llamante
		// recarga y reintenta.
		return false, nil
	}
	if err := store.writeV0(path, next); err != nil {
		return false, err
	}
	return true, nil
}

func mismoProgramaV0(izquierda, derecha autonomy.AutonomyProgramV0) (bool, error) {
	unaBytes, err := json.Marshal(izquierda)
	if err != nil {
		return false, err
	}
	otraBytes, err := json.Marshal(derecha)
	if err != nil {
		return false, err
	}
	return string(unaBytes) == string(otraBytes), nil
}
