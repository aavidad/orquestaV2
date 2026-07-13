package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// Cuando el gate encuentra una app sin decision, CONVOCA al consejo y deja la
// convocatoria pendiente en disco: quien tenga que votar puede ver que se le
// espera. Sin esto, el gate solo sabria decir "no" y el trabajo se quedaria
// esperando a que alguien se acordara de convocar a mano.
const councilPendingDirNameV0 = "council-pending"

const councilPendingSchemaVersionV0 = "orquesta_council_pending.v0"

type councilPendingConvocationV0 struct {
	SchemaVersion string                         `json:"schema_version"`
	CouncilRef    string                         `json:"council_ref"`
	AuthorRef     string                         `json:"author_ref,omitempty"`
	ConvenedAtUTC string                         `json:"convened_at_utc"`
	Seats         []orquestamcp.MCPCouncilSeatV0 `json:"seats,omitempty"`
	Warnings      []string                       `json:"warnings,omitempty"`
}

type councilConvenerV0 struct {
	dir      string
	executor councilExecutorV0
}

func newCouncilConvenerV0(stateDir string, executor councilExecutorV0) (councilConvenerV0, error) {
	dir := filepath.Join(stateDir, councilPendingDirNameV0)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return councilConvenerV0{}, fmt.Errorf("council pending: %w", err)
	}
	return councilConvenerV0{dir: dir, executor: executor}, nil
}

// ConveneCouncilForRefV0 es idempotente: si el consejo ya esta convocado, no se
// vuelve a convocar. Reconvocar cambiaria el reparto de roles a mitad de partida.
func (convener councilConvenerV0) ConveneCouncilForRefV0(
	ctx context.Context,
	councilRef string,
	authorRef string,
) error {
	path, err := convener.pathV0(councilRef)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	result, err := convener.executor.ConveneCouncilV0(ctx, orquestamcp.MCPCouncilToolInputV0{
		Action:     orquestamcp.MCPCouncilActionAssignV0,
		CouncilRef: councilRef,
		AuthorRef:  strings.TrimSpace(authorRef),
	})
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(councilPendingConvocationV0{
		SchemaVersion: councilPendingSchemaVersionV0,
		CouncilRef:    councilRef,
		AuthorRef:     result.AuthorRef,
		ConvenedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Seats:         result.Seats,
		Warnings:      result.Warnings,
	}, "", "  ")
	if err != nil {
		return err
	}
	temporal, err := os.CreateTemp(convener.dir, "pending-*.tmp")
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
	// Link, no Rename: si otro escritor convoco primero, gana el suyo. Un consejo
	// se convoca UNA vez.
	if err := os.Link(temporalPath, path); err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func (convener councilConvenerV0) pathV0(councilRef string) (string, error) {
	ref := strings.TrimSpace(councilRef)
	if ref == "" {
		return "", fmt.Errorf("council_ref_requerido")
	}
	if strings.ContainsAny(ref, "/\\") || ref == "." || ref == ".." {
		return "", fmt.Errorf("council_ref_invalido")
	}
	return filepath.Join(convener.dir, ref+".json"), nil
}
