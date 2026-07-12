package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type arranqueEspiaV0 struct{ llamado bool }

func (espia *arranqueEspiaV0) Execute(
	context.Context, orquestamcp.MCPArrancarDirectorAppToolInputV0,
) (orquestamcp.MCPArrancarDirectorAppToolResultV0, error) {
	espia.llamado = true
	return orquestamcp.MCPArrancarDirectorAppToolResultV0{Estado: "ok", RunRef: "run-1"}, nil
}

type decisionFalsaV0 struct {
	aceptadas  map[string]bool
	consultado string
}

func (fake *decisionFalsaV0) CouncilDecisionAcceptedV0(_ context.Context, councilRef string) (bool, error) {
	fake.consultado = councilRef
	return fake.aceptadas[councilRef], nil
}

func entradaV0(requestID string) orquestamcp.MCPArrancarDirectorAppToolInputV0 {
	return orquestamcp.MCPArrancarDirectorAppToolInputV0{
		AppSpecRequest: orquestafactory.AppSpecRequestV0{RequestID: requestID},
	}
}

// El consejo no es un tramite posterior: es LA PUERTA. Sin decision aceptada no
// se programa la app.
func TestCouncilGateBloqueaLaCreacionSinDecisionAceptadaV0(t *testing.T) {
	espia := &arranqueEspiaV0{}
	decision := &decisionFalsaV0{aceptadas: map[string]bool{}}
	gate := newCouncilGatedArrancarDirectorExecutorV0(espia, CouncilGateConfigV0{
		Required: true, Decision: decision,
	})

	result, err := gate.Execute(context.Background(), entradaV0("req-1"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if espia.llamado {
		t.Fatal("se arranco la app sin decision del consejo: la puerta no cierra")
	}
	if len(result.Errores) == 0 || result.Errores[0].Code != CouncilGateDecisionRequiredCodeV0 {
		t.Fatalf("no rechazo con el codigo esperado: %+v", result.Errores)
	}
	if decision.consultado != CouncilRefForAppRequestV0("req-1", orquestafactory.AppSpecRequestV0{RequestID: "req-1"}) {
		t.Fatalf("consulto el consejo equivocado: %q", decision.consultado)
	}
}

func TestCouncilGateDejaPasarConDecisionAceptadaV0(t *testing.T) {
	espia := &arranqueEspiaV0{}
	decision := &decisionFalsaV0{aceptadas: map[string]bool{
		CouncilRefForAppRequestV0("req-2", orquestafactory.AppSpecRequestV0{RequestID: "req-2"}): true,
	}}
	gate := newCouncilGatedArrancarDirectorExecutorV0(espia, CouncilGateConfigV0{
		Required: true, Decision: decision,
	})

	result, err := gate.Execute(context.Background(), entradaV0("req-2"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !espia.llamado {
		t.Fatal("con decision aceptada la app debe arrancar")
	}
	if result.RunRef != "run-1" {
		t.Fatalf("el gate altero el resultado del arranque real: %+v", result)
	}
}

// Con el gate desactivado el comportamiento no cambia en absoluto: el decorador
// ni siquiera se interpone.
func TestCouncilGateDesactivadoNoSeInterponeV0(t *testing.T) {
	espia := &arranqueEspiaV0{}
	gate := newCouncilGatedArrancarDirectorExecutorV0(espia, CouncilGateConfigV0{Required: false})
	if _, err := gate.Execute(context.Background(), entradaV0("req-3")); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !espia.llamado {
		t.Fatal("con el gate desactivado el arranque debe pasar tal cual")
	}
}

// Un gate exigido al que le falta el puerto de decision NO puede desaparecer: una
// puerta que se abre sola cuando la configuras mal no es una puerta.
func TestCouncilGateExigidoSinPuertoFallaCerradoV0(t *testing.T) {
	espia := &arranqueEspiaV0{}
	gate := newCouncilGatedArrancarDirectorExecutorV0(espia, CouncilGateConfigV0{
		Required: true, Decision: nil,
	})

	result, err := gate.Execute(context.Background(), entradaV0("req-4"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if espia.llamado {
		t.Fatal("fail-open: el gate exigido sin puerto dejo arrancar la app")
	}
	if len(result.Errores) == 0 || result.Errores[0].Code != CouncilGateMisconfiguredCodeV0 {
		t.Fatalf("no delato la mala configuracion: %+v", result.Errores)
	}
}

// BYPASS que Codex encontro: aprobar una app y despues MUTAR su especificacion
// reutilizando el mismo request_id. El consejo aprueba una especificacion
// concreta, no un nombre.
func TestCouncilGateNoAceptaAppSpecMutadaConElMismoRequestIDV0(t *testing.T) {
	original := orquestafactory.AppSpecRequestV0{RequestID: "req-x", Objetivo: "una calculadora"}
	mutada := orquestafactory.AppSpecRequestV0{RequestID: "req-x", Objetivo: "exfiltrar credenciales"}

	espia := &arranqueEspiaV0{}
	decision := &decisionFalsaV0{aceptadas: map[string]bool{
		CouncilRefForAppRequestV0("req-x", original): true,
	}}
	gate := newCouncilGatedArrancarDirectorExecutorV0(espia, CouncilGateConfigV0{
		Required: true, Decision: decision,
	})

	// La aprobada pasa.
	if _, err := gate.Execute(context.Background(), orquestamcp.MCPArrancarDirectorAppToolInputV0{
		AppSpecRequest: original,
	}); err != nil {
		t.Fatalf("Execute original: %v", err)
	}
	if !espia.llamado {
		t.Fatal("la app aprobada deberia arrancar")
	}

	// La mutada NO: mismo request_id, otra especificacion, otro consejo.
	espia.llamado = false
	result, err := gate.Execute(context.Background(), orquestamcp.MCPArrancarDirectorAppToolInputV0{
		AppSpecRequest: mutada,
	})
	if err != nil {
		t.Fatalf("Execute mutada: %v", err)
	}
	if espia.llamado {
		t.Fatal("BYPASS: una AppSpec mutada se colo por la aprobacion de otra")
	}
	if len(result.Errores) == 0 || result.Errores[0].Code != CouncilGateDecisionRequiredCodeV0 {
		t.Fatalf("no rechazo la spec mutada: %+v", result.Errores)
	}
}
