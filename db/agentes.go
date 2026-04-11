package db

import (
	"database/sql"
	"fmt"
	"time"
)

// AgenteInfo contiene toda la información de estado y cuota de un agente.
type AgenteInfo struct {
	Nombre           string
	Rol              string
	Activo           bool
	Habilitado       bool
	UltimaSesion     *time.Time
	ConsumoDia       int
	ConsumoSemanal   int
	LimiteDia        int
	LimiteSemanal    int
	EstadoCuota      string
	ReanimarAt       *time.Time
	MotivoPausa      string
}

// ListarAgenteInfo devuelve todos los agentes registrados con su estado actual de salud/cuota.
func ListarAgenteInfo() ([]*AgenteInfo, error) {
	rows, err := DB.Query(`
		SELECT nombre, rol, activo, habilitado, ultima_sesion,
		       consumo_dia_segundos, consumo_semanal_segundos,
		       limite_dia_segundos, limite_semanal_segundos,
		       estado_cuota, reanimar_at, motivo_pausa
		FROM agentes
		ORDER BY rol, nombre
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*AgenteInfo
	for rows.Next() {
		a := &AgenteInfo{}
		var activo, habilitado int
		var ultimaSesion, reanimarAt sql.NullTime
		var motivoPausa sql.NullString

		err := rows.Scan(
			&a.Nombre, &a.Rol, &activo, &habilitado, &ultimaSesion,
			&a.ConsumoDia, &a.ConsumoSemanal,
			&a.LimiteDia, &a.LimiteSemanal,
			&a.EstadoCuota, &reanimarAt, &motivoPausa,
		)
		if err != nil {
			return nil, err
		}

		a.Activo = activo == 1
		a.Habilitado = habilitado == 1
		if ultimaSesion.Valid { a.UltimaSesion = &ultimaSesion.Time }
		if reanimarAt.Valid { a.ReanimarAt = &reanimarAt.Time }
		if motivoPausa.Valid { a.MotivoPausa = motivoPausa.String }

		res = append(res, a)
	}
	return res, nil
}
// AgenteSetHabilitado activa o desactiva la capacidad operativa de un agente.
func AgenteSetHabilitado(nombre string, habilitado bool) error {
	val := 0
	if habilitado { val = 1 }
	_, err := DB.Exec(`UPDATE agentes SET habilitado = ? WHERE nombre = ?`, val, nombre)
	if err == nil {
		Audit("telegram_admin", "set_habilitado", "agente", 0, fmt.Sprintf("%s: %t", nombre, habilitado))
	}
	return err
}

// ResetEstadoAgentes pone a todos los agentes en modo 'inactivo' (0). Útil para limpiar fantasmas después de un crash.
func ResetEstadoAgentes() error {
	_, err := DB.Exec(`UPDATE agentes SET activo = 0`)
	if err == nil {
		Audit("system", "reset_agentes", "agente", 0, "Hard reset de estados activos")
	}
	return err
}

// GarantizarSaludAgentes realiza el mantenimiento automático de los estados y cuotas de los agentes.
// Se encarga de resetear cuotas a las 2 AM, desactivar agentes agotados y limpiar estados fantasma.
func GarantizarSaludAgentes() error {
	ahora := time.Now()
	
	// 1. Resetear cuotas si ha pasado de las 2 AM y no se ha reseteado hoy.
	// Nota: Si last_usage_reset_at es NULL o no es hoy y ya son las 02:XX AM o más.
	if _, err := DB.Exec(`
			UPDATE agentes 
			SET consumo_dia_segundos = 0, 
			    estado_cuota = 'activo', 
			    last_usage_reset_at = CURRENT_TIMESTAMP
			WHERE last_usage_reset_at IS NULL 
			   OR (
				 strftime('%Y-%m-%d', last_usage_reset_at) != strftime('%Y-%m-%d', 'now') 
				 AND strftime('%H', 'now') >= '02'
			   )
		`); err != nil {
		return err
	}

	// 2. Mandar a dormir (agotado) a los que se pasen del límite diario.
	if _, err := DB.Exec(`
			UPDATE agentes 
			SET estado_cuota = 'agotado', 
			    motivo_pausa = 'Cuota diaria agotada' 
			WHERE estado_cuota = 'activo' 
			  AND consumo_dia_segundos >= limite_dia_segundos
		`); err != nil {
		return err
	}

	// 3. Limpiar fantasmas (inactivos): si un agente figura como activo (1) 
	// pero su última sesión fue hace más de 10 minutos, lo ponemos a 0.
	limiteInactividad := ahora.Add(-10 * time.Minute)
	if _, err := DB.Exec(`
			UPDATE agentes 
			SET activo = 0 
			WHERE activo = 1 
			  AND (ultima_sesion < ? OR ultima_sesion IS NULL)
		`, limiteInactividad); err != nil {
		return err
	}

	return nil
}
