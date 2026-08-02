package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const consultaPreservacionEntorno = `SELECT ref,idempotency_key,project_ref,goal_ref,work_item_ref,execution_ref,workspace_ref,workspace_binding_digest,base_oid,object_format,change_set_ref,change_digest,state,execution_attempt,external_ref,fence,bundle_ref,bundle_digest,inventory_ref,inventory_digest,configuration_digest,rootfs_digest,seal_digest,provider_receipt_ref,sealed_at,preserved_at,recorded_at FROM agent_environment_receipts`

func (r *Repository) RegistrarPreservacionEntornoAgente(ctx context.Context, comprobante application.ComprobantePreservacionEntornoAgente) (application.ComprobantePreservacionEntornoAgente, bool, error) {
	comprobante.Resultado.SelladoEn, comprobante.Resultado.PreservadoEn = comprobante.Resultado.SelladoEn.Round(0).UTC(), comprobante.Resultado.PreservadoEn.Round(0).UTC()
	comprobante.RegistradoEn = comprobante.RegistradoEn.Round(0).UTC()
	if application.ValidarComprobantePreservacionEntornoAgente(comprobante) != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, invalid(errors.New("sqlite.agent_environment_receipt_invalid"))
	}
	tx, err := beginTransaction(ctx, r)
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	defer tx.Rollback()
	previo, existe, err := leerPreservacionEntorno(ctx, tx, consultaPreservacionEntorno+` WHERE ref=? OR idempotency_key=? OR execution_ref=? LIMIT 1`, comprobante.Ref, comprobante.ClaveIdempotencia, comprobante.EjecucionRef.String())
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	if existe {
		if !reflect.DeepEqual(previo, comprobante) {
			return application.ComprobantePreservacionEntornoAgente{}, false, conflict(errors.New("sqlite.agent_environment_receipt_conflict"))
		}
		return previo, false, commit(tx)
	}
	registro, err := readGoalRecord(ctx, tx, comprobante.ObjetivoRef.String())
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	if application.ValidarCausalidadPreservacionEntornoAgente(comprobante, registro) != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, conflict(errors.New("sqlite.agent_environment_receipt_causality_invalid"))
	}
	resultado := comprobante.Resultado
	cambio := sql.NullString{String: comprobante.CambioRef.String(), Valid: comprobante.CambioRef.String() != ""}
	digestCambio := sql.NullString{String: comprobante.DigestCambio, Valid: comprobante.DigestCambio != ""}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_environment_receipts VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		comprobante.Ref, comprobante.ClaveIdempotencia, comprobante.ProyectoRef.String(), comprobante.ObjetivoRef.String(), comprobante.ItemRef.String(), comprobante.EjecucionRef.String(), comprobante.EspacioTrabajoRef.String(), comprobante.DigestBindingEspacio, comprobante.BaseOID, string(comprobante.FormatoObjeto), cambio, digestCambio, string(resultado.Estado), resultado.IntentoEjecucion, resultado.IdentidadExterna, resultado.Cerca, resultado.PaqueteRef.String(), resultado.PaqueteDigest, resultado.InventarioRef.String(), resultado.InventarioDigest, resultado.ConfiguracionDigest, resultado.RootFSDigest, resultado.SelloDigest, resultado.ComprobanteRef, requiredTime(resultado.SelladoEn), requiredTime(resultado.PreservadoEn), requiredTime(comprobante.RegistradoEn))
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, mapDatabaseError(err)
	}
	if err = commit(tx); err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	return comprobante, true, nil
}

func leerPreservacionEntorno(ctx context.Context, source queryer, consulta string, argumentos ...any) (application.ComprobantePreservacionEntornoAgente, bool, error) {
	comprobante, err := escanearPreservacionEntorno(source.QueryRowContext(ctx, consulta, argumentos...))
	if errors.Is(err, sql.ErrNoRows) {
		return comprobante, false, nil
	}
	return comprobante, err == nil, err
}

func validarRecuperacionPreservacionEntorno(ctx context.Context, tx *sql.Tx) error {
	refs, err := readSingleColumn(ctx, tx, `SELECT ref FROM agent_environment_receipts ORDER BY ref`)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		comprobante, encontrado, readErr := leerPreservacionEntorno(ctx, tx, consultaPreservacionEntorno+` WHERE ref=?`, ref)
		if readErr != nil || !encontrado {
			return errors.New("sqlite.recovery_agent_environment_receipt_invalid")
		}
		registro, readErr := readGoalRecord(ctx, tx, comprobante.ObjetivoRef.String())
		if readErr != nil || application.ValidarCausalidadPreservacionEntornoAgente(comprobante, registro) != nil {
			return errors.New("sqlite.recovery_agent_environment_receipt_invalid")
		}
	}
	conCompuerta, err := sqliteTableHasColumn(ctx, tx, "executions", "environment_preservation_required")
	if err != nil || !conCompuerta {
		return mapDatabaseError(err)
	}
	var ausentes int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM executions e LEFT JOIN agent_environment_receipts r ON r.execution_ref=e.ref WHERE e.environment_preservation_required=1 AND e.state IN ('succeeded','failed','canceled','stopped') AND r.ref IS NULL`).Scan(&ausentes)
	if err != nil || ausentes != 0 {
		return errors.New("sqlite.recovery_agent_environment_preservation_required")
	}
	return nil
}

type escanerFila interface{ Scan(...any) error }

func escanearPreservacionEntorno(fila escanerFila) (application.ComprobantePreservacionEntornoAgente, error) {
	var c application.ComprobantePreservacionEntornoAgente
	var proyecto, objetivo, item, ejecucion, espacio, paquete, inventario string
	var cambio, digestCambio sql.NullString
	var intento, cerca, sellado, preservado, registrado int64
	r := &c.Resultado
	err := fila.Scan(&c.Ref, &c.ClaveIdempotencia, &proyecto, &objetivo, &item, &ejecucion, &espacio, &c.DigestBindingEspacio, &c.BaseOID, &c.FormatoObjeto, &cambio, &digestCambio, &r.Estado, &intento, &r.IdentidadExterna, &cerca, &paquete, &r.PaqueteDigest, &inventario, &r.InventarioDigest, &r.ConfiguracionDigest, &r.RootFSDigest, &r.SelloDigest, &r.ComprobanteRef, &sellado, &preservado, &registrado)
	if err != nil {
		return c, mapDatabaseError(err)
	}
	if intento <= 0 || cerca <= 0 {
		return c, invalid(errors.New("sqlite.agent_environment_receipt_corrupt"))
	}
	var refErr error
	if c.ProyectoRef, refErr = goal.NewProjectRef(proyecto); refErr == nil {
		c.ObjetivoRef, refErr = goal.NewGoalRef(objetivo)
	}
	if refErr == nil {
		c.ItemRef, refErr = goal.NewWorkItemRef(item)
	}
	if refErr == nil {
		c.EjecucionRef, refErr = goal.NewExecutionRef(ejecucion)
		r.EjecucionRef = c.EjecucionRef
	}
	if refErr == nil {
		c.EspacioTrabajoRef, refErr = ports.NewExecutionWorkspaceRef(espacio)
	}
	if refErr == nil && cambio.Valid {
		c.CambioRef, refErr = ports.NewChangeSetRef(cambio.String)
		c.DigestCambio = digestCambio.String
	}
	if refErr == nil {
		r.PaqueteRef, refErr = goal.NewArtifactRef(paquete)
	}
	if refErr == nil {
		r.InventarioRef, refErr = goal.NewArtifactRef(inventario)
	}
	r.IntentoEjecucion, r.Cerca = uint64(intento), uint64(cerca)
	r.SelladoEn, r.PreservadoEn, c.RegistradoEn = time.Unix(0, sellado).UTC(), time.Unix(0, preservado).UTC(), time.Unix(0, registrado).UTC()
	if refErr != nil || application.ValidarComprobantePreservacionEntornoAgente(c) != nil {
		return c, invalid(errors.New("sqlite.agent_environment_receipt_corrupt"))
	}
	return c, nil
}
