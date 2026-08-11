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

const consultaPreservacionEntorno = `SELECT ref,idempotency_key,project_ref,goal_ref,work_item_ref,execution_ref,workspace_scope,COALESCE(workspace_ref,''),workspace_binding_digest,base_oid,object_format,change_set_ref,change_digest,state,execution_attempt,external_ref,fence,bundle_ref,bundle_digest,inventory_ref,inventory_digest,configuration_digest,rootfs_digest,seal_digest,provider_receipt_ref,sealed_at,preserved_at,recorded_at,physical_manifest_ref,physical_manifest_digest FROM agent_environment_receipts`
const consultaPreservacionEntornoV34 = `SELECT ref,idempotency_key,project_ref,goal_ref,work_item_ref,execution_ref,'',workspace_ref,workspace_binding_digest,base_oid,object_format,change_set_ref,change_digest,state,execution_attempt,external_ref,fence,bundle_ref,bundle_digest,inventory_ref,inventory_digest,configuration_digest,rootfs_digest,seal_digest,provider_receipt_ref,sealed_at,preserved_at,recorded_at,physical_manifest_ref,physical_manifest_digest FROM agent_environment_receipts`
const consultaPreservacionEntornoV25 = `SELECT ref,idempotency_key,project_ref,goal_ref,work_item_ref,execution_ref,'',workspace_ref,workspace_binding_digest,base_oid,object_format,change_set_ref,change_digest,state,execution_attempt,external_ref,fence,bundle_ref,bundle_digest,inventory_ref,inventory_digest,configuration_digest,rootfs_digest,seal_digest,provider_receipt_ref,sealed_at,preserved_at,recorded_at,NULL,NULL FROM agent_environment_receipts`

func (r *Repository) RegistrarPreservacionEntornoAgente(ctx context.Context, comprobante application.ComprobantePreservacionEntornoAgente) (application.ComprobantePreservacionEntornoAgente, bool, error) {
	comprobante = normalizarPreservacionEntornoAgente(comprobante)
	tx, err := beginTransaction(ctx, r)
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	defer tx.Rollback()
	persistido, creado, err := insertAgentEnvironmentPreservation(ctx, tx, comprobante)
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	if err = commit(tx); err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	return persistido, creado, nil
}

func normalizarPreservacionEntornoAgente(
	comprobante application.ComprobantePreservacionEntornoAgente,
) application.ComprobantePreservacionEntornoAgente {
	comprobante.Resultado.SelladoEn = comprobante.Resultado.SelladoEn.Round(0).UTC()
	comprobante.Resultado.PreservadoEn = comprobante.Resultado.PreservadoEn.Round(0).UTC()
	comprobante.RegistradoEn = comprobante.RegistradoEn.Round(0).UTC()
	return comprobante
}

func insertAgentEnvironmentPreservation(
	ctx context.Context,
	tx *sql.Tx,
	comprobante application.ComprobantePreservacionEntornoAgente,
) (application.ComprobantePreservacionEntornoAgente, bool, error) {
	comprobante = normalizarPreservacionEntornoAgente(comprobante)
	if application.ValidarComprobantePreservacionEntornoAgente(comprobante) != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false,
			invalid(errors.New("sqlite.agent_environment_receipt_invalid"))
	}
	previo, existe, err := leerPreservacionEntorno(ctx, tx,
		consultaPreservacionEntorno+` WHERE ref=? OR idempotency_key=? OR execution_ref=? LIMIT 1`,
		comprobante.Ref, comprobante.ClaveIdempotencia, comprobante.EjecucionRef.String())
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	if existe {
		if !reflect.DeepEqual(previo, comprobante) {
			return application.ComprobantePreservacionEntornoAgente{}, false,
				conflict(errors.New("sqlite.agent_environment_receipt_conflict"))
		}
		return previo, false, nil
	}
	registro, err := readGoalRecord(ctx, tx, comprobante.ObjetivoRef.String())
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, err
	}
	if application.ValidarCausalidadPreservacionEntornoAgente(comprobante, registro) != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false,
			conflict(errors.New("sqlite.agent_environment_receipt_causality_invalid"))
	}
	resultado := comprobante.Resultado
	cambio := sql.NullString{String: comprobante.CambioRef.String(), Valid: comprobante.CambioRef.String() != ""}
	digestCambio := sql.NullString{String: comprobante.DigestCambio, Valid: comprobante.DigestCambio != ""}
	manifiestoRef := sql.NullString{String: comprobante.ManifiestoFisicoRef, Valid: comprobante.ManifiestoFisicoRef != ""}
	manifiestoDigest := sql.NullString{String: comprobante.ManifiestoFisicoDigest, Valid: comprobante.ManifiestoFisicoDigest != ""}
	_, err = tx.ExecContext(ctx, `INSERT INTO agent_environment_receipts(ref,idempotency_key,project_ref,goal_ref,work_item_ref,execution_ref,workspace_scope,workspace_ref,workspace_binding_digest,base_oid,object_format,change_set_ref,change_digest,state,execution_attempt,external_ref,fence,bundle_ref,bundle_digest,inventory_ref,inventory_digest,configuration_digest,rootfs_digest,seal_digest,provider_receipt_ref,sealed_at,preserved_at,recorded_at,physical_manifest_ref,physical_manifest_digest) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		comprobante.Ref, comprobante.ClaveIdempotencia, comprobante.ProyectoRef.String(), comprobante.ObjetivoRef.String(), comprobante.ItemRef.String(), comprobante.EjecucionRef.String(), string(comprobante.AlcanceEspacio), nullableString(comprobante.EspacioTrabajoRef.String()), comprobante.DigestBindingEspacio, comprobante.BaseOID, string(comprobante.FormatoObjeto), cambio, digestCambio, string(resultado.Estado), resultado.IntentoEjecucion, resultado.IdentidadExterna, resultado.Cerca, resultado.PaqueteRef.String(), resultado.PaqueteDigest, resultado.InventarioRef.String(), resultado.InventarioDigest, resultado.ConfiguracionDigest, resultado.RootFSDigest, resultado.SelloDigest, resultado.ComprobanteRef, requiredTime(resultado.SelladoEn), requiredTime(resultado.PreservadoEn), requiredTime(comprobante.RegistradoEn), manifiestoRef, manifiestoDigest)
	if err != nil {
		return application.ComprobantePreservacionEntornoAgente{}, false, mapDatabaseError(err)
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
	consulta := consultaPreservacionEntorno
	conAlcance, err := sqliteTableHasColumn(ctx, tx, "agent_environment_receipts", "workspace_scope")
	if err != nil {
		return err
	}
	conManifiesto, err := sqliteTableHasColumn(ctx, tx, "agent_environment_receipts", "physical_manifest_ref")
	if err != nil {
		return err
	}
	if !conAlcance && conManifiesto {
		consulta = consultaPreservacionEntornoV34
	} else if !conManifiesto {
		consulta = consultaPreservacionEntornoV25
	}
	refs, err := readSingleColumn(ctx, tx, `SELECT ref FROM agent_environment_receipts ORDER BY ref`)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		comprobante, encontrado, readErr := leerPreservacionEntorno(ctx, tx, consulta+` WHERE ref=?`, ref)
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
	var cambio, digestCambio, manifiestoRef, manifiestoDigest sql.NullString
	var intento, cerca, sellado, preservado, registrado int64
	r := &c.Resultado
	err := fila.Scan(&c.Ref, &c.ClaveIdempotencia, &proyecto, &objetivo, &item, &ejecucion, &c.AlcanceEspacio, &espacio, &c.DigestBindingEspacio, &c.BaseOID, &c.FormatoObjeto, &cambio, &digestCambio, &r.Estado, &intento, &r.IdentidadExterna, &cerca, &paquete, &r.PaqueteDigest, &inventario, &r.InventarioDigest, &r.ConfiguracionDigest, &r.RootFSDigest, &r.SelloDigest, &r.ComprobanteRef, &sellado, &preservado, &registrado, &manifiestoRef, &manifiestoDigest)
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
	if refErr == nil && espacio != "" {
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
	if refErr == nil {
		if manifiestoRef.Valid != manifiestoDigest.Valid {
			refErr = errors.New("sqlite.agent_environment_physical_manifest_partial")
		} else if manifiestoRef.Valid {
			c.ManifiestoFisicoRef, c.ManifiestoFisicoDigest = manifiestoRef.String, manifiestoDigest.String
		}
	}
	r.IntentoEjecucion, r.Cerca = uint64(intento), uint64(cerca)
	r.SelladoEn, r.PreservadoEn, c.RegistradoEn = time.Unix(0, sellado).UTC(), time.Unix(0, preservado).UTC(), time.Unix(0, registrado).UTC()
	if refErr != nil || application.ValidarComprobantePreservacionEntornoAgente(c) != nil {
		return c, invalid(errors.New("sqlite.agent_environment_receipt_corrupt"))
	}
	return c, nil
}
