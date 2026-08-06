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

func TestComprobantePreservacionEntornoAgenteSinEspacioNoInventaGit(t *testing.T) {
	digest := strings.Repeat("a", 64)
	proyecto, _ := goal.NewProjectRef("project:environment-without-workspace")
	objetivo, _ := goal.NewGoalRef("goal:environment-without-workspace")
	item, _ := goal.NewWorkItemRef("work-item:environment-without-workspace")
	ejecutada, _ := goal.NewExecutionRef("execution:environment-without-workspace")
	paquete, _ := goal.NewArtifactRef("artifact:sha256:" + digest)
	inventario, _ := goal.NewArtifactRef("artifact:sha256:" + digest)
	resultado := ports.ResultadoPreservacionEntornoAgente{Estado: ports.EntornoAgentePreservadoPendienteRevision,
		EjecucionRef: ejecutada, IntentoEjecucion: 1, IdentidadExterna: "external:environment-without-workspace", Cerca: 2,
		PaqueteRef: paquete, PaqueteDigest: digest, InventarioRef: inventario, InventarioDigest: digest,
		ConfiguracionDigest: digest, RootFSDigest: digest, ComprobanteRef: "receipt:environment-without-workspace",
		SelladoEn: time.Unix(10, 0).UTC(), PreservadoEn: time.Unix(11, 0).UTC()}
	resultado.SelloDigest = ports.ResumenSelloPreservacionEntorno(resultado)
	comprobante := ComprobantePreservacionEntornoAgente{Ref: "environment-receipt:without-workspace",
		ClaveIdempotencia: "environment-idempotency:without-workspace", ProyectoRef: proyecto, ObjetivoRef: objetivo,
		ItemRef: item, EjecucionRef: ejecutada, AlcanceEspacio: PreservacionEntornoSinEspacioTrabajo,
		Resultado: resultado, RegistradoEn: time.Unix(12, 0).UTC()}
	if err := ValidarComprobantePreservacionEntornoAgente(comprobante); err != nil {
		t.Fatalf("preservation without workspace rejected: %v", err)
	}

	for name, mutate := range map[string]func(*ComprobantePreservacionEntornoAgente){
		"workspace ref": func(value *ComprobantePreservacionEntornoAgente) {
			value.EspacioTrabajoRef, _ = ports.NewExecutionWorkspaceRef("execution-workspace:invented")
		},
		"binding digest": func(value *ComprobantePreservacionEntornoAgente) { value.DigestBindingEspacio = digest },
		"git oid":        func(value *ComprobantePreservacionEntornoAgente) { value.BaseOID = strings.Repeat("b", 40) },
		"git format":     func(value *ComprobantePreservacionEntornoAgente) { value.FormatoObjeto = ports.GitObjectFormatSHA1 },
		"change ref": func(value *ComprobantePreservacionEntornoAgente) {
			value.CambioRef, _ = ports.NewChangeSetRef("change-set:invented")
		},
		"change digest": func(value *ComprobantePreservacionEntornoAgente) { value.DigestCambio = digest },
	} {
		t.Run(name, func(t *testing.T) {
			changed := comprobante
			mutate(&changed)
			if ValidarComprobantePreservacionEntornoAgente(changed) == nil {
				t.Fatalf("workspace/Git fact invented for absent workspace: %+v", changed)
			}
		})
	}
}

func TestCausalidadPreservacionSinEspacioExigeAusenciaRealDeBinding(t *testing.T) {
	fixture := newAgentLaunchRecoveryFixtureWithPreservation(t, true)
	record := fixture.record
	var execution ExecutionRecord
	for index := range record.Executions {
		if record.Executions[index].Ref != fixture.request.ExecutionRef {
			continue
		}
		record.Executions[index].ExternalRef = "external:environment-without-workspace"
		record.Executions[index].LaunchReceiptRef = "receipt:launch:environment-without-workspace"
		execution = record.Executions[index]
	}
	if execution.Ref.String() == "" {
		t.Fatal("execution fixture missing")
	}
	digest := strings.Repeat("a", 64)
	paquete, _ := goal.NewArtifactRef("artifact:sha256:" + digest)
	inventario, _ := goal.NewArtifactRef("artifact:sha256:" + digest)
	resultado := ports.ResultadoPreservacionEntornoAgente{Estado: ports.EntornoAgentePreservadoPendienteRevision,
		EjecucionRef: execution.Ref, IntentoEjecucion: execution.AttemptNo,
		IdentidadExterna: execution.ExternalRef, Cerca: 7, PaqueteRef: paquete, PaqueteDigest: digest,
		InventarioRef: inventario, InventarioDigest: digest, ConfiguracionDigest: digest, RootFSDigest: digest,
		ComprobanteRef: "receipt:environment-without-workspace", SelladoEn: time.Unix(10, 0).UTC(),
		PreservadoEn: time.Unix(11, 0).UTC()}
	resultado.SelloDigest = ports.ResumenSelloPreservacionEntorno(resultado)
	comprobante := ComprobantePreservacionEntornoAgente{Ref: "environment-receipt:without-workspace",
		ClaveIdempotencia: "environment-idempotency:without-workspace", ProyectoRef: record.Goal.Project(),
		ObjetivoRef: record.Goal.Ref(), ItemRef: execution.WorkItemRef, EjecucionRef: execution.Ref,
		AlcanceEspacio: PreservacionEntornoSinEspacioTrabajo, Resultado: resultado,
		RegistradoEn: time.Unix(12, 0).UTC()}
	record.ConsumptionReceipts = append(record.ConsumptionReceipts, ActionConsumptionReceipt{
		Kind: ActionLaunchAgent, ExecutionRef: execution.Ref, Fence: resultado.Cerca, Outcome: ActionConsumedCompleted,
	})
	if err := ValidarCausalidadPreservacionEntornoAgente(comprobante, record); err != nil {
		t.Fatalf("real absence rejected: %v", err)
	}

	workspace, _ := ports.NewExecutionWorkspaceRef("execution-workspace:unexpected")
	record.WorkspaceBindings = append(record.WorkspaceBindings, WorkspaceBinding{Ref: workspace, ExecutionRef: execution.Ref})
	if err := ValidarCausalidadPreservacionEntornoAgente(comprobante, record); err == nil {
		t.Fatal("declared workspace absence accepted with binding for exact execution")
	}
	record.WorkspaceBindings = nil
	record.ChangeSets = append(record.ChangeSets, ChangeSet{ExecutionRef: execution.Ref})
	if err := ValidarCausalidadPreservacionEntornoAgente(comprobante, record); err == nil {
		t.Fatal("declared workspace absence accepted with change set for exact execution")
	}
}
