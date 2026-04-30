package db

import (
	"database/sql"
	"sort"
	"strings"
	"time"
)

func CuentaClaveAgente(agente *Agente) string {
	if agente == nil {
		return ""
	}
	if accountID := strings.ToLower(strings.TrimSpace(agente.CuentaID)); accountID != "" {
		return accountID
	}
	if email := strings.ToLower(strings.TrimSpace(agente.CuentaEmail)); email != "" {
		return email
	}
	if identidad := UltimaIdentidadCuentaObservadaAgente(strings.TrimSpace(agente.Nombre)); !identidadCuentaVacia(identidad) {
		if accountID := strings.ToLower(strings.TrimSpace(identidad.accountID)); accountID != "" {
			return accountID
		}
		if email := strings.ToLower(strings.TrimSpace(identidad.email)); email != "" {
			return email
		}
		if usuario := strings.ToLower(strings.TrimSpace(identidad.usuario)); usuario != "" {
			return usuario
		}
	}
	return strings.ToLower(strings.TrimSpace(agente.CuentaUsuario))
}

func CuentaCompartidaPermiteActivacionAgente(nombre string) (bool, string, error) {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return true, "", nil
	}
	if DB == nil {
		return true, "", nil
	}
	agente, err := GetAgente(nombre)
	if err == sql.ErrNoRows {
		return true, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return cuentaCompartidaPermiteActivacion(agente)
}

func cuentaCompartidaPermiteActivacion(agente *Agente) (bool, string, error) {
	if agente == nil {
		return true, "", nil
	}
	ceiling := cuentaCompartidaCeiling()
	if ceiling <= 0 {
		return true, "", nil
	}
	key := CuentaClaveAgente(agente)
	if key == "" {
		return true, "", nil
	}
	ocupantes, err := agentesOcupandoCuentaCompartida(key, strings.TrimSpace(agente.Nombre))
	if err != nil {
		return false, "", err
	}
	if len(ocupantes) >= ceiling {
		return false, ocupantes[0], nil
	}
	return true, "", nil
}

func cuentaCompartidaCeiling() int {
	return configIntOrDefault("runtime_shared_account_active_ceiling", 1)
}

func cuentaCompartidaOrderHold() time.Duration {
	seconds := configIntOrDefault("runtime_shared_account_order_hold_seconds", 300)
	if seconds <= 0 {
		seconds = 300
	}
	return time.Duration(seconds) * time.Second
}

func agentesOcupandoCuentaCompartida(cuentaKey string, exclude string) ([]string, error) {
	cuentaKey = strings.ToLower(strings.TrimSpace(cuentaKey))
	exclude = strings.ToLower(strings.TrimSpace(exclude))
	if cuentaKey == "" {
		return nil, nil
	}
	agentes, err := ListarAgentes()
	if err != nil {
		return nil, err
	}
	ocupantes := make([]string, 0, 2)
	for _, item := range agentes {
		if item == nil || !item.Habilitado {
			continue
		}
		nombre := strings.TrimSpace(item.Nombre)
		if nombre == "" || strings.ToLower(nombre) == exclude {
			continue
		}
		if CuentaClaveAgente(item) != cuentaKey {
			continue
		}
		ocupa, err := agenteOcupaCapacidadCuenta(nombre)
		if err != nil {
			return nil, err
		}
		if ocupa {
			ocupantes = append(ocupantes, nombre)
		}
	}
	sort.Strings(ocupantes)
	return ocupantes, nil
}

func agenteOcupaCapacidadCuenta(agente string) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, nil
	}
	infoAgente, err := GetAgente(agente)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	if infoAgente != nil {
		estadoCuota := strings.TrimSpace(infoAgente.EstadoCuota)
		if estadoCuota != "" && !strings.EqualFold(estadoCuota, "activo") {
			return false, nil
		}
	}
	// La capacidad compartida se reserva solo por evidencias canónicas del
	// runtime (handle activo/operativo o una orden viva reciente de start/
	// resume/handoff). Una sesión sola no basta: puede quedar viva por heartbeat
	// aunque el runtime real ya no exista, y bloquear capacidad de forma falsa.
	if handle, err := runtimeHandleOperativoRecienteConFallback(agente, nil); err != nil {
		return false, err
	} else if handle != nil {
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "pausado", "fallido", "cerrado":
		default:
			return true, nil
		}
	}
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Agente: &agente})
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	for _, order := range orders {
		if runtimeOrderOcupaCapacidadCuenta(order, now) {
			return true, nil
		}
	}
	return false, nil
}

func runtimeOrderOcupaCapacidadCuenta(order *RuntimeOrder, now time.Time) bool {
	if order == nil || !runtimeOrderEstadoVivo(order.Estado) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(order.Tipo)) {
	case "start", "resume", "handoff":
	default:
		return false
	}
	hold := cuentaCompartidaOrderHold()
	// updated_at no es una señal fiable de ocupación: puede moverse por
	// reconciliaciones o retoques administrativos y rejuvenecer órdenes
	// zombis. Para capacidad compartida solo valen los hitos reales de vida.
	baseline := order.CreatedAt.UTC()
	if order.StartedAt != nil && !order.StartedAt.IsZero() && order.StartedAt.UTC().After(baseline) {
		baseline = order.StartedAt.UTC()
	}
	if !order.AvailableAt.IsZero() && order.AvailableAt.UTC().After(baseline) {
		baseline = order.AvailableAt.UTC()
	}
	if baseline.IsZero() {
		return false
	}
	return !baseline.Before(now.Add(-hold))
}
