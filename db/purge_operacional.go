/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "time"

// PurgarDatosOperacionales elimina datos transitorios que se acumulan sin límite
// y que no son necesarios para orquestar el trabajo de los agentes:
//
//   - runtime_mailbox mensajes consumidos hace más de 24h
//   - presupuestos_sesion, conservando solo el último por sesión
//   - runtime_telemetry_samples con más de 48h
//   - audit_log con más de 30 días
//   - runtime_transcript señales con más de 7 días
//
// Es seguro llamarlo con el servidor en marcha.
func PurgarDatosOperacionales() (int, error) {
	if DB == nil {
		return 0, nil
	}
	total := 0

	// Mailbox consumido >24h
	res, err := DB.Exec(`DELETE FROM runtime_mailbox WHERE estado='consumido' AND updated_at < ?`,
		time.Now().UTC().Add(-24*time.Hour))
	if err == nil {
		n, _ := res.RowsAffected()
		total += int(n)
	}

	// Presupuestos: conservar solo el último por sesión
	res, err = DB.Exec(`DELETE FROM presupuestos_sesion
		WHERE id NOT IN (
			SELECT MAX(id) FROM presupuestos_sesion GROUP BY sesion_id
		)`)
	if err == nil {
		n, _ := res.RowsAffected()
		total += int(n)
	}

	// Telemetría >48h
	res, err = DB.Exec(`DELETE FROM runtime_telemetry_samples WHERE sampled_at < ?`,
		time.Now().UTC().Add(-48*time.Hour))
	if err == nil {
		n, _ := res.RowsAffected()
		total += int(n)
	}

	// Audit log >30 días
	res, err = DB.Exec(`DELETE FROM audit_log WHERE created_at < ?`,
		time.Now().UTC().Add(-30*24*time.Hour))
	if err == nil {
		n, _ := res.RowsAffected()
		total += int(n)
	}

	// Señales de transcript >7 días
	res, err = DB.Exec(`DELETE FROM runtime_transcript WHERE created_at < ?`,
		time.Now().UTC().Add(-7*24*time.Hour))
	if err == nil {
		n, _ := res.RowsAffected()
		total += int(n)
	}

	return total, nil
}
