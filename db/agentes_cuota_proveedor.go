package db

import (
	"encoding/json"
	"strings"
)

func enriquecerAgentesSinCuotaProveedor(list []*Agente) {
	if len(list) == 0 {
		return
	}
	nombres := make([]string, 0, len(list))
	for _, agente := range list {
		if agente == nil {
			continue
		}
		nombres = append(nombres, agente.Nombre)
	}
	mapa, err := mapaAgentesSinCuotaProveedor(nombres)
	if err != nil {
		mapa = map[string]bool{}
	}
	for _, agente := range list {
		if agente == nil {
			continue
		}
		if mapa[strings.ToLower(strings.TrimSpace(agente.Nombre))] {
			normalizarAgenteSinCuotaProveedor(agente)
		}
	}
}

func enriquecerAgenteSinCuotaProveedor(agente *Agente) {
	if agente == nil {
		return
	}
	if nombrePareceAgenteSinCuotaProveedor(agente.Nombre) {
		normalizarAgenteSinCuotaProveedor(agente)
		return
	}
	mapa, err := mapaAgentesSinCuotaProveedor([]string{agente.Nombre})
	if err != nil {
		return
	}
	if mapa[strings.ToLower(strings.TrimSpace(agente.Nombre))] {
		normalizarAgenteSinCuotaProveedor(agente)
	}
}

func mapaAgentesSinCuotaProveedor(nombres []string) (map[string]bool, error) {
	refs, err := mapaConectorRecienteAgente(nombres)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(nombres))
	for _, nombre := range nombres {
		key := strings.ToLower(strings.TrimSpace(nombre))
		if key == "" {
			continue
		}
		if conectorSinCuotaProveedorPorRef(refs[key]) || nombrePareceAgenteSinCuotaProveedor(nombre) {
			out[key] = true
		}
	}
	return out, nil
}

func mapaConectorRecienteAgente(nombres []string) (map[string]string, error) {
	args := make([]any, 0, len(nombres))
	seen := make(map[string]struct{}, len(nombres))
	for _, nombre := range nombres {
		key := strings.ToLower(strings.TrimSpace(nombre))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		args = append(args, strings.TrimSpace(nombre))
	}
	out := make(map[string]string, len(args))
	if len(args) == 0 {
		return out, nil
	}

	querySesiones := `
		SELECT s.agente, COALESCE(NULLIF(TRIM(c.slug), ''), NULLIF(TRIM(s.herramienta), ''))
		FROM sesiones s
		LEFT JOIN conectores c ON c.id = s.conector_id
		JOIN (
			SELECT agente, MAX(id) AS max_id
			FROM sesiones
			WHERE agente IN (` + runtimeSQLPlaceholders(len(args)) + `)
			GROUP BY agente
		) ult ON ult.max_id = s.id`
	rows, err := DB.Query(querySesiones, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var agente, ref string
		if err := rows.Scan(&agente, &ref); err != nil {
			return nil, err
		}
		key := strings.ToLower(strings.TrimSpace(agente))
		if key != "" && strings.TrimSpace(ref) != "" {
			out[key] = strings.TrimSpace(ref)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	faltan := make([]any, 0, len(args))
	for _, arg := range args {
		nombre := strings.ToLower(strings.TrimSpace(arg.(string)))
		if nombre == "" {
			continue
		}
		if _, ok := out[nombre]; ok {
			continue
		}
		faltan = append(faltan, arg)
	}
	if len(faltan) == 0 {
		return out, nil
	}

	queryRuntimes := `
		SELECT r.agente, TRIM(r.connector)
		FROM runtime_instances r
		JOIN (
			SELECT agente, MAX(id) AS max_id
			FROM runtime_instances
			WHERE agente IN (` + runtimeSQLPlaceholders(len(faltan)) + `)
			GROUP BY agente
		) ult ON ult.max_id = r.id`
	rowsRt, err := DB.Query(queryRuntimes, faltan...)
	if err != nil {
		return nil, err
	}
	defer rowsRt.Close()
	for rowsRt.Next() {
		var agente, ref string
		if err := rowsRt.Scan(&agente, &ref); err != nil {
			return nil, err
		}
		key := strings.ToLower(strings.TrimSpace(agente))
		if key != "" && strings.TrimSpace(ref) != "" {
			out[key] = strings.TrimSpace(ref)
		}
	}
	if err := rowsRt.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func nombrePareceAgenteSinCuotaProveedor(nombre string) bool {
	nombre = strings.ToLower(strings.TrimSpace(nombre))
	if nombre == "" {
		return false
	}
	for _, prefijo := range []string{"ollama", "gemma", "qwen", "llama", "starcoder", "deepseek"} {
		if strings.HasPrefix(nombre, prefijo) {
			return true
		}
	}
	return false
}

func AgenteSinCuotaProveedorEfectivo(a *Agente) bool {
	if a == nil {
		return false
	}
	if a.SinCuotaProveedor {
		return true
	}
	return nombrePareceAgenteSinCuotaProveedor(a.Nombre)
}

func conectorSinCuotaProveedorPorRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	lower := strings.ToLower(ref)
	if strings.Contains(lower, "ollama") {
		return true
	}
	conector, err := GetConector(ref)
	if err != nil || conector == nil {
		return false
	}
	return conectorSinCuotaProveedor(conector)
}

func conectorSinCuotaProveedor(conector *Conector) bool {
	if conector == nil {
		return false
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(conector.Slug)), "ollama") {
		return true
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(conector.Comando)), "ollama") {
		return true
	}
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(conector.MetadataJSON)), &meta); err == nil && len(meta) > 0 {
		if value, ok := meta["sin_cuota_proveedor"].(bool); ok && value {
			return true
		}
		if familia, ok := meta["familia"].(string); ok && strings.EqualFold(strings.TrimSpace(familia), "ollama") {
			return true
		}
	}
	return false
}

func normalizarAgenteSinCuotaProveedor(a *Agente) {
	if a == nil {
		return
	}
	a.SinCuotaProveedor = true
	a.EstadoCuota = "activo"
	a.ReanimarAt = nil
	if !agenteMotivoPausaOperativaVisible(a.MotivoPausa) {
		a.MotivoPausa = ""
	}
	a.CuotaRestantePct = nil
	a.PresupuestoEstado = ""
	a.PresupuestoFuente = "local"
	a.PresupuestoCheckedAt = nil
	a.PresupuestoStale = false
	a.PresupuestoVentana = "indefinida"
	a.PresupuestoResetAt = nil
	a.PresupuestoSesionPct = nil
	a.PresupuestoSesionResetAt = nil
	a.PresupuestoDiarioPct = nil
	a.PresupuestoDiarioResetAt = nil
	a.PresupuestoSemanalPct = nil
	a.PresupuestoSemanalResetAt = nil
	a.RemainingSeconds = nil
	a.RemainingMessages = nil
	a.RemainingTokens = nil
	a.RemainingCredits = nil
}
