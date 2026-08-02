package sqlite

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestPreservacionEntornoEsDurableIdempotenteYCausal(t *testing.T) {
	ctx := context.Background()
	sistema := newSQLiteV15System(t, 2)
	sistema.orchestrator = newSQLiteV16Orchestrator(t, sistema)
	creado, err := sistema.orchestrator.Submit(ctx, sistema.access, application.SubmitRequest{
		RequestRef: "request:a06-preservacion", Statement: "preservar un entorno aislado", Confirm: true,
		Plan: &application.PlanSpec{
			Phases:    []application.PhaseSpec{{Ref: "phase-instance:a06", Key: "phase:a06", TemplateRef: "phase-template:a06"}},
			WorkItems: []application.WorkItemSpec{{Key: "writer", Objective: "producir evidencia", Phase: "phase:a06", Role: "role:writer", WriteSet: []string{"internal/a06"}, RequiredTests: sqliteRequiredTestSpecs("required-test:a06"), CouncilPolicy: council.PolicyAuto, OutputContract: goal.OutputContractEvidenceBundle}},
		},
	})
	sqliteTestNoError(t, err)
	processSQLiteV16Actions(t, sistema, application.ActionPrepareWorkspace, application.ActionLaunchAgent)
	registro, err := sistema.repository.GetGoal(ctx, creado.Record.Goal.Ref())
	sqliteTestNoError(t, err)
	if len(registro.Executions) != 1 || len(registro.WorkspaceBindings) != 1 {
		t.Fatalf("frontera de preservación incompleta: %+v", registro)
	}
	ejecutada, binding := registro.Executions[0], registro.WorkspaceBindings[0]
	var cerca uint64
	for _, consumo := range registro.ConsumptionReceipts {
		if consumo.Kind == application.ActionLaunchAgent && consumo.ExecutionRef == ejecutada.Ref {
			cerca = consumo.Fence
		}
	}
	comprobante := comprobantePreservacionSQLite(t, registro, ejecutada, binding, cerca, sistema.clock.Now())
	persistido, creadoAhora, err := sistema.repository.RegistrarPreservacionEntornoAgente(ctx, comprobante)
	if err != nil || !creadoAhora || !reflect.DeepEqual(persistido, comprobante) {
		t.Fatalf("persistir comprobante: creado=%v got=%+v err=%v", creadoAhora, persistido, err)
	}
	if repetido, creadoOtraVez, err := sistema.repository.RegistrarPreservacionEntornoAgente(ctx, comprobante); err != nil || creadoOtraVez || !reflect.DeepEqual(repetido, comprobante) {
		t.Fatalf("replay exacto: creado=%v got=%+v err=%v", creadoOtraVez, repetido, err)
	}
	casos := []struct {
		nombre  string
		alterar func(*application.ComprobantePreservacionEntornoAgente)
	}{
		{"digest", func(c *application.ComprobantePreservacionEntornoAgente) {
			c.Resultado.RootFSDigest = strings.Repeat("9", 64)
		}},
		{"referencia", func(c *application.ComprobantePreservacionEntornoAgente) { c.Ref = "environment-receipt:otro" }},
		{"proyecto", func(c *application.ComprobantePreservacionEntornoAgente) {
			c.ProyectoRef = mustRef(t, "project:otro", goal.NewProjectRef)
		}},
		{"cerca", func(c *application.ComprobantePreservacionEntornoAgente) { c.Resultado.Cerca++ }},
	}
	for _, caso := range casos {
		alterado := comprobante
		caso.alterar(&alterado)
		alterado.Resultado.SelloDigest = ports.ResumenSelloPreservacionEntorno(alterado.Resultado)
		if _, _, err := sistema.repository.RegistrarPreservacionEntornoAgente(ctx, alterado); !application.IsStateError(err, application.StateConflict) {
			t.Fatalf("%s ajeno aceptado: %v", caso.nombre, err)
		}
	}
	if _, _, err := validateRecoveryDatabase(ctx, sistema.repository.db); err != nil {
		t.Fatalf("recovery previo al reinicio: %v", err)
	}
	sqliteTestNoError(t, sistema.repository.Close())
	reiniciado := openSQLiteV15Repository(t, sistema.path, sistema.clock.Now)
	recuperado, creadoTrasReinicio, err := reiniciado.RegistrarPreservacionEntornoAgente(ctx, comprobante)
	if err != nil || creadoTrasReinicio || !reflect.DeepEqual(recuperado, comprobante) {
		t.Fatalf("reinicio perdió el comprobante: creado=%v got=%+v err=%v", creadoTrasReinicio, recuperado, err)
	}
	if _, err := reiniciado.db.Exec(`UPDATE agent_environment_receipts SET fence=fence+1`); err == nil {
		t.Fatal("la evidencia mutable aceptó una cerca ajena")
	}
}

func comprobantePreservacionSQLite(t *testing.T, registro application.GoalRecord, ejecutada application.ExecutionRecord, binding application.WorkspaceBinding, cerca uint64, ahora time.Time) application.ComprobantePreservacionEntornoAgente {
	t.Helper()
	paquete, err := goal.NewArtifactRef("artifact:sha256:" + strings.Repeat("a", 64))
	sqliteTestNoError(t, err)
	inventario, err := goal.NewArtifactRef("artifact:sha256:" + strings.Repeat("b", 64))
	sqliteTestNoError(t, err)
	resultado := ports.ResultadoPreservacionEntornoAgente{
		Estado: ports.EntornoAgentePreservadoPendienteRevision, EjecucionRef: ejecutada.Ref,
		IntentoEjecucion: ejecutada.AttemptNo, IdentidadExterna: ejecutada.ExternalRef, Cerca: cerca,
		PaqueteRef: paquete, PaqueteDigest: strings.Repeat("a", 64), InventarioRef: inventario,
		InventarioDigest: strings.Repeat("b", 64), ConfiguracionDigest: strings.Repeat("c", 64),
		RootFSDigest: strings.Repeat("d", 64), ComprobanteRef: "provider-receipt:a06-preservacion",
		SelladoEn: ahora, PreservadoEn: ahora.Add(time.Second),
	}
	resultado.SelloDigest = ports.ResumenSelloPreservacionEntorno(resultado)
	return application.ComprobantePreservacionEntornoAgente{
		Ref: "environment-receipt:a06-preservacion", ClaveIdempotencia: "environment-preservation:a06-preservacion",
		ProyectoRef: registro.Goal.Project(), ObjetivoRef: registro.Goal.Ref(), ItemRef: ejecutada.WorkItemRef,
		EjecucionRef: ejecutada.Ref, EspacioTrabajoRef: binding.Ref, DigestBindingEspacio: binding.Digest(),
		BaseOID: binding.BaseOID, FormatoObjeto: binding.ObjectFormat, Resultado: resultado,
		RegistradoEn: resultado.PreservadoEn.Add(time.Second),
	}
}
