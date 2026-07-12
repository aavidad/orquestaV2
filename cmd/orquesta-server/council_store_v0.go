package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// El consejo tiene que sobrevivir a un reinicio: una decision que se pierde al
// reiniciar no es una decision, es una opinion. Se guarda un recibo por
// council_ref con semantica CREATE-IF-ABSENT: el primero que decide gana y el
// segundo choca. Sobrescribir seria dejar que dos veredictos distintos se pisen
// en silencio.
const councilReceiptsDirNameV0 = "council-receipts"

const councilOutcomeAcceptedV0 = "council_decision_accepted"

const councilReceiptSchemaVersionV0 = "orquesta_council_receipt.v0"

var (
	// ErrCouncilReceiptConflictV0 se devuelve cuando el mismo council_ref se
	// reutiliza con una convocatoria DISTINTA. Sin esto, cambiar de autor, de
	// miembros o de votos devolveria el veredicto viejo como si nada.
	ErrCouncilReceiptConflictV0 = errors.New("council_receipt_conflict")
	// ErrCouncilReceiptCorruptV0 se devuelve si el recibo esta ilegible. NO se
	// trata como "no existe": eso seria fail-open, y permitiria sobrescribir una
	// decision tomada rompiendo el fichero.
	ErrCouncilReceiptCorruptV0 = errors.New("council_receipt_corrupt")
)

type councilReceiptV0 struct {
	SchemaVersion string `json:"schema_version"`
	CouncilRef    string `json:"council_ref"`
	AuthorRef     string `json:"author_ref,omitempty"`
	// InputFingerprint identifica la convocatoria (autor, miembros, votos). Es lo
	// que distingue "reintento del mismo consejo" de "consejo distinto con el
	// mismo nombre".
	InputFingerprint string                             `json:"input_fingerprint"`
	DecidedAtUTC     string                             `json:"decided_at_utc"`
	Seats            []orquestamcp.MCPCouncilSeatV0     `json:"seats,omitempty"`
	Warnings         []string                           `json:"warnings,omitempty"`
	Outcome          string                             `json:"outcome,omitempty"`
	Approvals        int                                `json:"approvals,omitempty"`
	Reworks          int                                `json:"reworks,omitempty"`
	Blocks           int                                `json:"blocks,omitempty"`
	Total            int                                `json:"total,omitempty"`
	Rationale        string                             `json:"rationale,omitempty"`
	Overrides        []orquestamcp.MCPCouncilOverrideV0 `json:"overrides,omitempty"`
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

// LoadV0 distingue las tres situaciones que importan: no existe, existe y es
// legible, o existe y esta corrupto. Confundir la tercera con la primera es
// fail-open.
func (store councilReceiptStoreV0) LoadV0(councilRef string) (councilReceiptV0, bool, error) {
	path, err := store.pathV0(councilRef)
	if err != nil {
		return councilReceiptV0{}, false, err
	}
	bytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return councilReceiptV0{}, false, nil
	}
	if err != nil {
		return councilReceiptV0{}, false, fmt.Errorf("%w: %v", ErrCouncilReceiptCorruptV0, err)
	}
	var receipt councilReceiptV0
	if err := json.Unmarshal(bytes, &receipt); err != nil {
		return councilReceiptV0{}, false, fmt.Errorf("%w: %s", ErrCouncilReceiptCorruptV0, councilRef)
	}
	// Un JSON que parsea no es un recibo valido. Sin esta validacion, dejar caer
	// {"outcome":"council_decision_accepted"} en el directorio de estado abria el
	// gate: dos lineas de fichero valian por una decision del consejo.
	if err := validarReciboV0(receipt, councilRef); err != nil {
		return councilReceiptV0{}, false, err
	}
	return receipt, true, nil
}

func validarReciboV0(receipt councilReceiptV0, councilRef string) error {
	if receipt.SchemaVersion != councilReceiptSchemaVersionV0 {
		return fmt.Errorf("%w: schema_version=%q", ErrCouncilReceiptCorruptV0, receipt.SchemaVersion)
	}
	if strings.TrimSpace(receipt.CouncilRef) != strings.TrimSpace(councilRef) {
		return fmt.Errorf("%w: council_ref no coincide con el fichero", ErrCouncilReceiptCorruptV0)
	}
	if strings.TrimSpace(receipt.InputFingerprint) == "" {
		return fmt.Errorf("%w: sin huella de convocatoria", ErrCouncilReceiptCorruptV0)
	}
	switch receipt.Outcome {
	case councilOutcomeAcceptedV0, "council_decision_rework", "council_decision_blocked":
	default:
		return fmt.Errorf("%w: outcome=%q", ErrCouncilReceiptCorruptV0, receipt.Outcome)
	}
	// Una decision sin votantes no es una decision.
	if receipt.Total <= 0 || len(receipt.Seats) == 0 {
		return fmt.Errorf("%w: decision sin consejo (total=%d)", ErrCouncilReceiptCorruptV0, receipt.Total)
	}
	return nil
}

// SaveV0 crea el recibo con O_EXCL: si ya existe, NO lo sobrescribe. Dos decide
// concurrentes no pueden pisarse; el segundo o reconoce el mismo veredicto (misma
// huella) o choca con conflicto tipado.
func (store councilReceiptStoreV0) SaveV0(receipt councilReceiptV0) (councilReceiptV0, error) {
	path, err := store.pathV0(receipt.CouncilRef)
	if err != nil {
		return councilReceiptV0{}, err
	}
	receipt.SchemaVersion = councilReceiptSchemaVersionV0
	if receipt.DecidedAtUTC == "" {
		receipt.DecidedAtUTC = time.Now().UTC().Format(time.RFC3339)
	}
	bytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return councilReceiptV0{}, err
	}

	// Temporal UNICO por escritor: un ".tmp" compartido es una carrera entre dos
	// procesos que escriben a la vez.
	temporal, err := os.CreateTemp(store.dir, "receipt-*.tmp")
	if err != nil {
		return councilReceiptV0{}, err
	}
	temporalPath := temporal.Name()
	defer os.Remove(temporalPath)
	if _, err := temporal.Write(bytes); err != nil {
		temporal.Close()
		return councilReceiptV0{}, err
	}
	if err := temporal.Chmod(0o600); err != nil {
		temporal.Close()
		return councilReceiptV0{}, err
	}
	if err := temporal.Close(); err != nil {
		return councilReceiptV0{}, err
	}

	// Link falla si el destino existe: es el create-if-absent que Rename no da.
	if err := os.Link(temporalPath, path); err != nil {
		if !os.IsExist(err) {
			return councilReceiptV0{}, err
		}
		existente, ok, loadErr := store.LoadV0(receipt.CouncilRef)
		if loadErr != nil {
			return councilReceiptV0{}, loadErr
		}
		if !ok {
			return councilReceiptV0{}, fmt.Errorf("%w: %s", ErrCouncilReceiptConflictV0, receipt.CouncilRef)
		}
		// Mismo consejo, misma convocatoria: es un reintento, devuelve lo decidido.
		if existente.InputFingerprint == receipt.InputFingerprint {
			return existente, nil
		}
		// Mismo nombre, convocatoria distinta: eso NO es un reintento.
		return councilReceiptV0{}, fmt.Errorf("%w: %s", ErrCouncilReceiptConflictV0, receipt.CouncilRef)
	}
	return receipt, nil
}

// councilInputFingerprintV0 resume la convocatoria: autor, criticidad, miembros,
// overrides y votos. Reutilizar un council_ref con cualquiera de esos datos
// cambiados produce una huella distinta y, por tanto, un conflicto.
func councilInputFingerprintV0(input orquestamcp.MCPCouncilToolInputV0) string {
	// CANONICALIZACION: los overrides salen de un map y los miembros/votos pueden
	// llegar en cualquier orden. Sin ordenarlos, la misma convocatoria produce
	// huellas distintas segun el barrido del map y un reintento legitimo choca.
	members := append([]orquestamcp.MCPCouncilMemberV0(nil), input.Members...)
	sort.Slice(members, func(i, j int) bool { return members[i].MemberRef < members[j].MemberRef })
	overrides := append([]orquestamcp.MCPCouncilOverrideV0(nil), input.Overrides...)
	sort.Slice(overrides, func(i, j int) bool {
		if overrides[i].Role != overrides[j].Role {
			return overrides[i].Role < overrides[j].Role
		}
		return overrides[i].MemberRef < overrides[j].MemberRef
	})
	ballots := append([]orquestamcp.MCPCouncilBallotV0(nil), input.Ballots...)
	sort.Slice(ballots, func(i, j int) bool { return ballots[i].MemberRef < ballots[j].MemberRef })

	material := struct {
		AuthorRef        string                             `json:"author_ref"`
		SecurityCritical bool                               `json:"security_critical"`
		Members          []orquestamcp.MCPCouncilMemberV0   `json:"members"`
		Overrides        []orquestamcp.MCPCouncilOverrideV0 `json:"overrides"`
		Ballots          []orquestamcp.MCPCouncilBallotV0   `json:"ballots"`
	}{
		AuthorRef:        input.AuthorRef,
		SecurityCritical: input.SecurityCritical,
		Members:          members,
		Overrides:        overrides,
		Ballots:          ballots,
	}
	bytes, err := json.Marshal(material)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(bytes)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// CouncilDecisionAcceptedV0 es lo que consulta el gate de creacion. Un recibo
// ilegible NO abre la puerta: el gate falla cerrado.
func (store councilReceiptStoreV0) CouncilDecisionAcceptedV0(
	_ context.Context,
	councilRef string,
) (bool, error) {
	receipt, ok, err := store.LoadV0(councilRef)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	return receipt.Outcome == councilOutcomeAcceptedV0, nil
}
