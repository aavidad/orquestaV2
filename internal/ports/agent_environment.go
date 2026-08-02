package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/goal"
)

type EstadoPreservacionEntornoAgente string

const EntornoAgentePreservadoPendienteRevision EstadoPreservacionEntornoAgente = "preserved_pending_review"

type ResultadoPreservacionEntornoAgente struct {
	Estado                                            EstadoPreservacionEntornoAgente
	EjecucionRef                                      goal.ExecutionRef
	IntentoEjecucion, Cerca                           uint64
	PaqueteRef, InventarioRef                         goal.ArtifactRef
	IdentidadExterna, PaqueteDigest, InventarioDigest string
	ConfiguracionDigest, RootFSDigest, SelloDigest    string
	ComprobanteRef                                    string
	SelladoEn, PreservadoEn                           time.Time
}

func ResumenSelloPreservacionEntorno(resultado ResultadoPreservacionEntornoAgente) string {
	hash := sha256.New()
	for _, valor := range []string{resultado.EjecucionRef.String(), strconv.FormatUint(resultado.IntentoEjecucion, 10),
		resultado.IdentidadExterna, strconv.FormatUint(resultado.Cerca, 10), resultado.PaqueteDigest,
		resultado.InventarioDigest, resultado.ConfiguracionDigest, resultado.RootFSDigest,
		resultado.SelladoEn.UTC().Format(time.RFC3339Nano)} {
		hash.Write([]byte(valor))
		hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func ValidarResultadoPreservacionEntornoAgente(resultado ResultadoPreservacionEntornoAgente) error {
	if resultado.Estado != EntornoAgentePreservadoPendienteRevision || resultado.EjecucionRef.String() == "" ||
		resultado.IntentoEjecucion == 0 || !validAgentIdentityRef(resultado.IdentidadExterna) || resultado.Cerca == 0 ||
		!artefactoEntornoValido(resultado.PaqueteRef, resultado.PaqueteDigest) ||
		!artefactoEntornoValido(resultado.InventarioRef, resultado.InventarioDigest) ||
		!resumenEntornoValido(resultado.ConfiguracionDigest) || !resumenEntornoValido(resultado.RootFSDigest) ||
		resultado.SelloDigest != ResumenSelloPreservacionEntorno(resultado) || !validAgentReceiptRef(resultado.ComprobanteRef) ||
		resultado.SelladoEn.IsZero() || resultado.PreservadoEn.Before(resultado.SelladoEn) {
		return &AgentContractError{Code: "agent.environment_preservation_invalid"}
	}
	return nil
}

func artefactoEntornoValido(referencia goal.ArtifactRef, resumen string) bool {
	return resumenEntornoValido(resumen) && referencia.String() == "artifact:sha256:"+resumen
}

func resumenEntornoValido(valor string) bool {
	if len(valor) != sha256.Size*2 || strings.ToLower(valor) != valor {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}
