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

func TestPreservacionEntornoConManifestFisicoEsDurableEIdempotente(t *testing.T) {
	ctx := context.Background()
	sistema := newSQLiteV15System(t, 2)
	sistema.external.requierePreservacion = true
	sistema.orchestrator = newSQLiteV16Orchestrator(t, sistema)
	creado, err := sistema.orchestrator.Submit(ctx, sistema.access, application.SubmitRequest{
		RequestRef: "request:a06-preservacion-manifest", Statement: "preservar con manifiesto físico", Confirm: true,
		Plan: &application.PlanSpec{
			Phases: []application.PhaseSpec{{
				Ref: "phase-instance:a06-manifest", Key: "phase:a06-manifest",
				TemplateRef: "phase-template:a06-manifest",
			}},
			WorkItems: []application.WorkItemSpec{{
				Key: "writer", Objective: "producir evidencia física", Phase: "phase:a06-manifest",
				Role: "role:writer", WriteSet: []string{"internal/a06-manifest"},
				RequiredTests: sqliteRequiredTestSpecs("required-test:a06-manifest"),
				CouncilPolicy: council.PolicyAuto, OutputContract: goal.OutputContractEvidenceBundle,
			}},
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
	comprobante.ManifiestoFisicoRef = "physical-manifest:a06"
	comprobante.ManifiestoFisicoDigest = strings.Repeat("e", 64)

	persistido, creadoAhora, err := sistema.repository.RegistrarPreservacionEntornoAgente(ctx, comprobante)
	if err != nil || !creadoAhora || !reflect.DeepEqual(persistido, comprobante) {
		t.Fatalf("manifest receipt persistido creado=%v got=%+v err=%v", creadoAhora, persistido, err)
	}
	repetido, creadoAhora, err := sistema.repository.RegistrarPreservacionEntornoAgente(ctx, comprobante)
	if err != nil || creadoAhora || !reflect.DeepEqual(repetido, comprobante) {
		t.Fatalf("manifest receipt repetido creado=%v got=%+v err=%v", creadoAhora, repetido, err)
	}
	sqliteTestNoError(t, sistema.repository.Close())
	reiniciado := openSQLiteV15Repository(t, sistema.path, sistema.clock.Now)
	recuperado, creadoAhora, err := reiniciado.RegistrarPreservacionEntornoAgente(ctx, comprobante)
	if err != nil || creadoAhora || !reflect.DeepEqual(recuperado, comprobante) {
		t.Fatalf("manifest receipt tras reinicio creado=%v got=%+v err=%v", creadoAhora, recuperado, err)
	}
	connection, err := reiniciado.db.Conn(ctx)
	sqliteTestNoError(t, err)
	defer connection.Close()
	_, err = connection.ExecContext(ctx, `DROP TRIGGER agent_environment_receipts_immutable_update`)
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(ctx, `PRAGMA ignore_check_constraints=ON`)
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(ctx, `
UPDATE agent_environment_receipts SET physical_manifest_digest=NULL WHERE ref=?`, comprobante.Ref)
	sqliteTestNoError(t, err)
	_, err = connection.ExecContext(ctx, `PRAGMA ignore_check_constraints=OFF`)
	sqliteTestNoError(t, err)
	if _, _, err := leerPreservacionEntorno(
		ctx, reiniciado.db, consultaPreservacionEntorno+` WHERE ref=?`, comprobante.Ref,
	); err == nil || !strings.Contains(sqliteTestErrorChain(err), "sqlite.agent_environment_receipt_corrupt") {
		t.Fatalf("manifest físico parcial se degradó a legacy: %s", sqliteTestErrorChain(err))
	}
}

func TestPreservacionEntornoEsDurableIdempotenteYCausal(t *testing.T) {
	ctx := context.Background()
	sistema := newSQLiteV15System(t, 2)
	sistema.external.requierePreservacion = true
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
	if !ejecutada.RequierePreservacionEntorno {
		t.Fatal("la aceptación perdió el requisito de preservación")
	}
	var cerca uint64
	for _, consumo := range registro.ConsumptionReceipts {
		if consumo.Kind == application.ActionLaunchAgent && consumo.ExecutionRef == ejecutada.Ref {
			cerca = consumo.Fence
		}
	}
	comprobante := comprobantePreservacionSQLite(t, registro, ejecutada, binding, cerca, sistema.clock.Now())
	terminal := ejecutada
	terminal.State, terminal.FinishedAt = application.ExecutionSucceeded, sistema.clock.Now().Add(time.Second)
	tx, err := beginTransaction(ctx, sistema.repository)
	sqliteTestNoError(t, err)
	if err = updateExecutionCAS(ctx, tx, terminal, application.ExecutionRunning); !application.IsStateError(err, application.StateConflict) {
		t.Fatalf("terminalización sin preservar aceptada: %v", err)
	}
	sqliteTestNoError(t, tx.Rollback())
	_, err = sistema.repository.db.Exec(`UPDATE executions SET state='succeeded',finished_at=? WHERE ref=?`, requiredTime(terminal.FinishedAt), ejecutada.Ref.String())
	sqliteTestNoError(t, err)
	tx, err = beginReadTransaction(ctx, sistema.repository)
	sqliteTestNoError(t, err)
	if err = validarRecuperacionPreservacionEntorno(ctx, tx); err == nil || !recoveryErrorContains(err, "sqlite.recovery_agent_environment_preservation_required") {
		t.Fatalf("recovery específico aceptó terminalización sin preservar: %v", err)
	}
	sqliteTestNoError(t, tx.Rollback())
	_, err = sistema.repository.db.Exec(`UPDATE executions SET state='running',finished_at=NULL WHERE ref=?`, ejecutada.Ref.String())
	sqliteTestNoError(t, err)
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
	tx, err = beginTransaction(ctx, sistema.repository)
	sqliteTestNoError(t, err)
	if err = updateExecutionCAS(ctx, tx, terminal, application.ExecutionRunning); err != nil {
		t.Fatalf("comprobante exacto no abrió la compuerta terminal: %v", err)
	}
	sqliteTestNoError(t, tx.Rollback())
	sqliteTestNoError(t, sistema.repository.Close())
	reiniciado := openSQLiteV15Repository(t, sistema.path, sistema.clock.Now)
	recuperado, creadoTrasReinicio, err := reiniciado.RegistrarPreservacionEntornoAgente(ctx, comprobante)
	if err != nil || creadoTrasReinicio || !reflect.DeepEqual(recuperado, comprobante) {
		t.Fatalf("reinicio perdió el comprobante: creado=%v got=%+v err=%v", creadoTrasReinicio, recuperado, err)
	}
	registroReiniciado, err := reiniciado.GetGoal(ctx, registro.Goal.Ref())
	if err != nil || !registroReiniciado.Executions[0].RequierePreservacionEntorno {
		t.Fatalf("reinicio perdió la compuerta: err=%v", err)
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
