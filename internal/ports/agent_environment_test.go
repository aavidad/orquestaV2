package ports

import (
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestPreservacionEntornoAgenteLigaSelloEInventarioSinRutas(t *testing.T) {
	digest := strings.Repeat("a", 64)
	ejecutada, _ := goal.NewExecutionRef("execution:environment")
	paquete, _ := goal.NewArtifactRef("artifact:sha256:" + digest)
	inventario, _ := goal.NewArtifactRef("artifact:sha256:" + digest)
	resultado := ResultadoPreservacionEntornoAgente{Estado: EntornoAgentePreservadoPendienteRevision,
		EjecucionRef: ejecutada, IntentoEjecucion: 1, IdentidadExterna: "external:environment", Cerca: 2,
		PaqueteRef: paquete, PaqueteDigest: digest, InventarioRef: inventario, InventarioDigest: digest,
		ConfiguracionDigest: digest, RootFSDigest: digest, ComprobanteRef: "receipt:environment",
		SelladoEn: time.Unix(10, 0).UTC(), PreservadoEn: time.Unix(11, 0).UTC()}
	resultado.SelloDigest = ResumenSelloPreservacionEntorno(resultado)
	if err := ValidarResultadoPreservacionEntornoAgente(resultado); err != nil {
		t.Fatal(err)
	}
	for nombre, mutar := range map[string]func(*ResultadoPreservacionEntornoAgente){
		"estado":     func(valor *ResultadoPreservacionEntornoAgente) { valor.Estado = "deleted" },
		"cerca":      func(valor *ResultadoPreservacionEntornoAgente) { valor.Cerca++ },
		"inventario": func(valor *ResultadoPreservacionEntornoAgente) { valor.InventarioDigest = strings.Repeat("b", 64) },
		"sello":      func(valor *ResultadoPreservacionEntornoAgente) { valor.SelloDigest = strings.Repeat("c", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := resultado
			mutar(&alterado)
			if ValidarResultadoPreservacionEntornoAgente(alterado) == nil {
				t.Fatal("preservación alterada aceptada")
			}
		})
	}
}
