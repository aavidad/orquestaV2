package application

import (
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestComprobantePreservacionEntornoAgenteConservaCausalidadOrquesta(t *testing.T) {
	digest := strings.Repeat("a", 64)
	proyecto, _ := goal.NewProjectRef("project:environment")
	objetivo, _ := goal.NewGoalRef("goal:environment")
	item, _ := goal.NewWorkItemRef("work-item:environment")
	ejecutada, _ := goal.NewExecutionRef("execution:environment")
	espacio, _ := ports.NewExecutionWorkspaceRef("execution-workspace:environment")
	paquete, _ := goal.NewArtifactRef("artifact:sha256:" + digest)
	inventario, _ := goal.NewArtifactRef("artifact:sha256:" + digest)
	resultado := ports.ResultadoPreservacionEntornoAgente{Estado: ports.EntornoAgentePreservadoPendienteRevision,
		EjecucionRef: ejecutada, IntentoEjecucion: 1, IdentidadExterna: "external:environment", Cerca: 2,
		PaqueteRef: paquete, PaqueteDigest: digest, InventarioRef: inventario, InventarioDigest: digest,
		ConfiguracionDigest: digest, RootFSDigest: digest, ComprobanteRef: "receipt:environment",
		SelladoEn: time.Unix(10, 0).UTC(), PreservadoEn: time.Unix(11, 0).UTC()}
	resultado.SelloDigest = ports.ResumenSelloPreservacionEntorno(resultado)
	comprobante := ComprobantePreservacionEntornoAgente{Ref: "environment-receipt:one", ClaveIdempotencia: "environment-idempotency:one",
		ProyectoRef: proyecto, ObjetivoRef: objetivo, ItemRef: item, EjecucionRef: ejecutada, EspacioTrabajoRef: espacio,
		DigestBindingEspacio: digest, BaseOID: strings.Repeat("b", 40), FormatoObjeto: ports.GitObjectFormatSHA1,
		Resultado: resultado, RegistradoEn: time.Unix(12, 0).UTC()}
	if err := ValidarComprobantePreservacionEntornoAgente(comprobante); err != nil {
		t.Fatal(err)
	}
	comprobante.Resultado.EjecucionRef, _ = goal.NewExecutionRef("execution:other")
	if ValidarComprobantePreservacionEntornoAgente(comprobante) == nil {
		t.Fatal("comprobante cruzado aceptado")
	}
}
