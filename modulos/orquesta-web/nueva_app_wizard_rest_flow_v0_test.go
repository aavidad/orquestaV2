package orquestaweb

import (
	"context"
	"testing"
	"time"

	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestafactoryhttp "orquesta/modulos/orquesta-factory-http"
)

func TestWizardNuevoListoCierraFactoryRealPorPuertoV0(t *testing.T) {
	session := wizardSessionWithObjectiveV0(
		"req-wizard-factory-real-001",
		"quiero una agenda compartida para clientes con calendario, avisos y disponibilidad",
	)
	session, turns := AcceptWebNuevaAppWizardRecommendationsV0(session, 8)
	if len(turns) == 0 {
		t.Fatalf("wizard sin turnos")
	}
	final := NewWebNuevaAppWizardTurnResultV0(session)
	if !final.LaunchReady || !final.SpecComplete || final.SpecPreview == nil {
		t.Fatalf("wizard no deja contrato listo: %+v", final)
	}
	if issues := orquestafactory.ValidateAppSpecRequestV0(*final.SpecPreview); len(issues) != 0 {
		t.Fatalf("preview invalida antes de factory: %+v\n%+v", issues, *final.SpecPreview)
	}
	if len(final.SpecPreview.Integraciones) == 0 ||
		final.SpecPreview.Integraciones[0].Tipo != "calendar" ||
		final.SpecPreview.Integraciones[0].Nombre == "" ||
		final.SpecPreview.Integraciones[0].Proposito == "" ||
		!final.SpecPreview.Integraciones[0].Requerido {
		t.Fatalf("wizard no preserva conector punteado: %+v", final.SpecPreview.Integraciones)
	}

	server := newWebHTTPTestServerV0(t, orquestafactoryhttp.NewAppSpecHTTPHandlerV0(fixedRESTFlowClockV0))
	defer server.Close()
	client := NewRESTSolicitarNuevaAppClientV0(server.URL, time.Second)
	vm, err := client.SolicitarNuevaApp(context.Background(), session.Form)
	if err != nil {
		t.Fatalf("SolicitarNuevaApp: %v", err)
	}

	if vm.Estado != WebNuevaAppEstadoValida ||
		vm.RequestID != "req-wizard-factory-real-001" ||
		vm.SpecID == "" ||
		vm.AppSpecSchemaVersion != orquestafactory.AppSpecSchemaV0 ||
		vm.BacklogSchemaVersion != orquestafactory.BacklogInicialPropuestoSchemaV0 ||
		vm.ValidationEstadoFuente != "valida" {
		t.Fatalf("factory no cerro spec real: %+v", vm)
	}
	if vm.ResumenApp.Nombre == "" ||
		vm.ResumenApp.TipoApp == "" ||
		len(vm.Fases) == 0 ||
		len(vm.Microtareas) == 0 ||
		!containsRESTFlowValueV0(vm.ContratosRequeridos, "ConnectorContract v0") {
		t.Fatalf("backlog real incompleto: %+v", vm)
	}
}
