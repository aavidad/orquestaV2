package orquestastatefile

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"

	orquestaautonomyprogram "orquesta/modulos/orquesta-autonomy-program"
)

type autonomyProgramDocumentV0 struct {
	SchemaVersion string                                    `json:"schema_version"`
	ProgramRef    string                                    `json:"program_ref"`
	ProjectRef    string                                    `json:"project_ref"`
	RootRef       string                                    `json:"root_ref"`
	Program       orquestaautonomyprogram.AutonomyProgramV0 `json:"program"`
}

func (store *StoreV0) SaveAutonomyProgramV0(ctx context.Context, program orquestaautonomyprogram.AutonomyProgramV0) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := orquestaautonomyprogram.NewAutonomyProgramV0(program)
	if err != nil {
		return invalidErrorV0("autonomy_program", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.autonomyProgramPathV0(normalized.ProjectRef, normalized.RootRef, normalized.ProgramRef)
	return store.withAutonomyProgramFileLockV0(ctx, normalized.ProjectRef, normalized.RootRef, normalized.ProgramRef, func() error {
		existing, ok, err := readJSONFileV0[autonomyProgramDocumentV0](path)
		if err != nil {
			return err
		}
		if ok {
			loaded, err := validateAutonomyProgramDocumentV0(existing, normalized.ProjectRef, normalized.RootRef, normalized.ProgramRef)
			if err != nil {
				return err
			}
			if autonomyProgramStateEqualV0(loaded, normalized) {
				return nil
			}
			if !orquestaautonomyprogram.SameAutonomyProgramTopologyV0(loaded, normalized) {
				return storeErrorV0("autonomy_program", "programa existente con DAG distinto")
			}
			return storeErrorV0("autonomy_program", "estado existente requiere compare_and_swap")
		}
		return writeJSONAtomicV0(path, autonomyProgramDocumentV0{SchemaVersion: autonomyProgramDocumentSchemaV0, ProgramRef: normalized.ProgramRef, ProjectRef: normalized.ProjectRef, RootRef: normalized.RootRef, Program: normalized})
	})
}

func (store *StoreV0) CompareAndSwapAutonomyProgramV0(ctx context.Context, expected, next orquestaautonomyprogram.AutonomyProgramV0) (bool, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return false, err
	}
	expected, err := orquestaautonomyprogram.NewAutonomyProgramV0(expected)
	if err != nil {
		return false, invalidErrorV0("autonomy_program.expected", err.Error())
	}
	next, err = orquestaautonomyprogram.NewAutonomyProgramV0(next)
	if err != nil {
		return false, invalidErrorV0("autonomy_program.next", err.Error())
	}
	if err := orquestaautonomyprogram.ValidateAutonomyProgramSuccessorV0(expected, next); err != nil {
		return false, invalidErrorV0("autonomy_program.successor", err.Error())
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	path := store.autonomyProgramPathV0(expected.ProjectRef, expected.RootRef, expected.ProgramRef)
	swapped := false
	err = store.withAutonomyProgramFileLockV0(ctx, expected.ProjectRef, expected.RootRef, expected.ProgramRef, func() error {
		document, ok, readErr := readJSONFileV0[autonomyProgramDocumentV0](path)
		if readErr != nil {
			return readErr
		}
		if !ok {
			return nil
		}
		loaded, validateErr := validateAutonomyProgramDocumentV0(document, expected.ProjectRef, expected.RootRef, expected.ProgramRef)
		if validateErr != nil {
			return validateErr
		}
		if !autonomyProgramStateEqualV0(loaded, expected) {
			return nil
		}
		if writeErr := writeJSONAtomicV0(path, autonomyProgramDocumentV0{SchemaVersion: autonomyProgramDocumentSchemaV0, ProgramRef: next.ProgramRef, ProjectRef: next.ProjectRef, RootRef: next.RootRef, Program: next}); writeErr != nil {
			return writeErr
		}
		swapped = true
		return nil
	})
	return swapped, err
}

func autonomyProgramStateEqualV0(left, right orquestaautonomyprogram.AutonomyProgramV0) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func (store *StoreV0) withAutonomyProgramFileLockV0(ctx context.Context, projectRef, rootRef, programRef string, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	lockPath := store.autonomyProgramLockPathV0(projectRef, rootRef, programRef)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN) //nolint:errcheck -- best effort after the protected operation
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}

func (store *StoreV0) LoadAutonomyProgramV0(ctx context.Context, projectRef string, rootRef string, programRef string) (orquestaautonomyprogram.AutonomyProgramV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestaautonomyprogram.AutonomyProgramV0{}, err
	}
	projectRef, rootRef, programRef = normalizeRefV0(projectRef), normalizeRefV0(rootRef), normalizeRefV0(programRef)
	if projectRef == "" || rootRef == "" || programRef == "" {
		return orquestaautonomyprogram.AutonomyProgramV0{}, invalidErrorV0("autonomy_program.scope", "project_ref, root_ref y program_ref requeridos")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	document, ok, err := readJSONFileV0[autonomyProgramDocumentV0](store.autonomyProgramPathV0(projectRef, rootRef, programRef))
	if err != nil {
		return orquestaautonomyprogram.AutonomyProgramV0{}, err
	}
	if !ok {
		return orquestaautonomyprogram.AutonomyProgramV0{}, storeErrorV0("autonomy_program", "programa no encontrado")
	}
	return validateAutonomyProgramDocumentV0(document, projectRef, rootRef, programRef)
}

func validateAutonomyProgramDocumentV0(document autonomyProgramDocumentV0, projectRef string, rootRef string, programRef string) (orquestaautonomyprogram.AutonomyProgramV0, error) {
	if document.SchemaVersion != autonomyProgramDocumentSchemaV0 {
		return orquestaautonomyprogram.AutonomyProgramV0{}, storeErrorV0("autonomy_program.schema_version", "schema_version invalida")
	}
	if document.ProjectRef != projectRef || document.RootRef != rootRef || document.ProgramRef != programRef {
		return orquestaautonomyprogram.AutonomyProgramV0{}, storeErrorV0("autonomy_program.ref", "ref inconsistente")
	}
	program, err := orquestaautonomyprogram.NewAutonomyProgramV0(document.Program)
	if err != nil {
		return orquestaautonomyprogram.AutonomyProgramV0{}, err
	}
	if program.ProjectRef != projectRef || program.RootRef != rootRef || program.ProgramRef != programRef {
		return orquestaautonomyprogram.AutonomyProgramV0{}, storeErrorV0("autonomy_program.state_ref", "ref interna inconsistente")
	}
	return program, nil
}
