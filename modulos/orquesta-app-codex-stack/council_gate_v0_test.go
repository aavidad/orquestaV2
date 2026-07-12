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
	if decision.consultado != CouncilRefForAppRequestV0("req-1") {
		t.Fatalf("consulto el consejo equivocado: %q", decision.consultado)
	}
}

func TestCouncilGateDejaPasarConDecisionAceptadaV0(t *testing.T) {
	espia := &arranqueEspiaV0{}
	decision := &decisionFalsaV0{aceptadas: map[string]bool{
		CouncilRefForAppRequestV0("req-2"): true,
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
