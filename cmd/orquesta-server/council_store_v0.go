package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// El consejo tiene que sobrevivir a un reinicio: una decision que se pierde al
// reiniciar no es una decision, es una opinion. Se guarda un recibo por
// council_ref, con escritura atomica (temporal + rename) como el resto del
// estado de Orquesta.
const councilReceiptsDirNameV0 = "council-receipts"

type councilReceiptV0 struct {
	SchemaVersion string                             `json:"schema_version"`
	CouncilRef    string                             `json:"council_ref"`
	AuthorRef     string                             `json:"author_ref,omitempty"`
	DecidedAtUTC  string                             `json:"decided_at_utc"`
	Seats         []orquestamcp.MCPCouncilSeatV0     `json:"seats,omitempty"`
	Warnings      []string                           `json:"warnings,omitempty"`
	Outcome       string                             `json:"outcome,omitempty"`
	Approvals     int                                `json:"approvals,omitempty"`
	Reworks       int                                `json:"reworks,omitempty"`
	Blocks        int                                `json:"blocks,omitempty"`
	Total         int                                `json:"total,omitempty"`
	Rationale     string                             `json:"rationale,omitempty"`
	Overrides     []orquestamcp.MCPCouncilOverrideV0 `json:"overrides,omitempty"`
}

type councilReceiptStoreV0 struct {
	dir string
}

func newCouncilReceiptStoreV0(stateDir string) (councilReceiptStoreV0, error) {
	dir := filepath.Join(stateDir, councilReceiptsDirNameV0)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return councilReceiptStoreV0{}, fmt.Errorf("council receipts: %w", err)
	}
	return councilReceiptStoreV0{dir: dir}, nil
}

func (store councilReceiptStoreV0) pathV0(councilRef string) (string, error) {
	ref := strings.TrimSpace(councilRef)
	if ref == "" {
		return "", fmt.Errorf("council_ref_requerido")
	}
	// El council_ref viene de fuera: se sanea para que no escape del directorio.
	if strings.ContainsAny(ref, "/\\") || ref == "." || ref == ".." {
		return "", fmt.Errorf("council_ref_invalido")
	}
	return filepath.Join(store.dir, ref+".json"), nil
}

// LoadV0 devuelve el recibo previo si existe. Es lo que da IDEMPOTENCIA: convocar
// dos veces el mismo council_ref no vuelve a decidir, devuelve lo decidido.
func (store councilReceiptStoreV0) LoadV0(councilRef string) (councilReceiptV0, bool) {
	path, err := store.pathV0(councilRef)
	if err != nil {
		return councilReceiptV0{}, false
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		return councilReceiptV0{}, false
	}
	var receipt councilReceiptV0
	if err := json.Unmarshal(bytes, &receipt); err != nil {
		return councilReceiptV0{}, false
	}
	return receipt, true
}

func (store councilReceiptStoreV0) SaveV0(receipt councilReceiptV0) error {
	path, err := store.pathV0(receipt.CouncilRef)
	if err != nil {
		return err
	}
	receipt.SchemaVersion = "orquesta_council_receipt.v0"
	if receipt.DecidedAtUTC == "" {
		receipt.DecidedAtUTC = time.Now().UTC().Format(time.RFC3339)
	}
	bytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	temporal := path + ".tmp"
	if err := os.WriteFile(temporal, bytes, 0o600); err != nil {
		return err
	}
	return os.Rename(temporal, path)
}
