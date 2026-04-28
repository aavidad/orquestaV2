package db

import (
	"strings"
	"time"

	"orquesta/planificadorpolicy"
)

func PoolLocalCompartidoPermiteActivacionAgenteProyecto(agente, proyectoSlug string) (bool, string, error) {
	agente = strings.TrimSpace(agente)
	proyectoSlug = strings.TrimSpace(proyectoSlug)
	if agente == "" {
		return true, "", nil
	}
	resolucion, err := ResolverPoliticaModelo(ResolverPoliticaInput{
		AgenteNombre: &agente,
		ProyectoSlug: proyectoSlug,
	})
	if err != nil {
		return false, "", err
	}
	if resolucion == nil || strings.TrimSpace(resolucion.PoolSlug) == "" {
		return true, "", nil
	}
	pool, err := GetPool(strings.TrimSpace(resolucion.PoolSlug))
	if err != nil {
		return false, "", err
	}
	if !poolUsaConectorPoolLocalCompartido(pool) {
		return true, "", nil
	}
	resumen, err := ListarPoolsResumen(boolPtr(true))
	if err != nil {
		return false, "", err
	}
	for _, item := range resumen {
		if item == nil || item.Pool == nil || item.Pool.ID != pool.ID {
			continue
		}
		reservasPendientes, err := reservasPendientesPoolLocalCompartido(pool.ID, strings.TrimSpace(pool.Slug))
		if err != nil {
			return false, "", err
		}
		return item.CapacidadDisponible-reservasPendientes > 0, strings.TrimSpace(pool.Slug), nil
	}
	return true, strings.TrimSpace(pool.Slug), nil
}

func reservasPendientesPoolLocalCompartido(poolID int64, poolSlug string) (int, error) {
	if poolID <= 0 || strings.TrimSpace(poolSlug) == "" {
		return 0, nil
	}
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{})
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	reservas := 0
	for _, order := range orders {
		if !runtimeOrderOcupaPoolLocalCompartido(order, poolID, strings.TrimSpace(poolSlug), now) {
			continue
		}
		reservas++
	}
	return reservas, nil
}

func runtimeOrderOcupaPoolLocalCompartido(order *RuntimeOrder, poolID int64, poolSlug string, now time.Time) bool {
	if order == nil || poolID <= 0 || strings.TrimSpace(poolSlug) == "" {
		return false
	}
	if !runtimeOrderOcupaCapacidadCuenta(order, now) {
		return false
	}
	if agenteOcupa, err := agenteTieneSesionActivaEnPoolLocalCompartido(order.Agente, poolID); err == nil && agenteOcupa {
		return false
	}
	proyectoSlug := strings.TrimSpace(runtimeOrderProyectoSlug(order))
	if proyectoSlug == "" {
		return false
	}
	agente := strings.TrimSpace(order.Agente)
	resolucion, err := ResolverPoliticaModelo(ResolverPoliticaInput{
		AgenteNombre: &agente,
		ProyectoSlug: proyectoSlug,
	})
	if err != nil || resolucion == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(resolucion.PoolSlug), strings.TrimSpace(poolSlug))
}

func agenteTieneSesionActivaEnPoolLocalCompartido(agente string, poolID int64) (bool, error) {
	var err error
	agente, err = CanonicalizeAgentName(agente)
	if err != nil {
		return false, err
	}
	agente = strings.TrimSpace(agente)
	if agente == "" || poolID <= 0 {
		return false, nil
	}
	var n int
	if err := DB.QueryRow(`
		SELECT COUNT(*)
		FROM sesiones
		WHERE agente = ?
		  AND activa = 1
		  AND pool_id = ?`,
		agente, poolID,
	).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func runtimeOrderProyectoSlug(order *RuntimeOrder) string {
	if order == nil {
		return ""
	}
	if order.ProyectoID != nil && *order.ProyectoID > 0 {
		if proyecto, err := GetProyecto(jsonNumber(*order.ProyectoID)); err == nil && proyecto != nil {
			return strings.TrimSpace(proyecto.Slug)
		}
	}
	return strings.TrimSpace(stringFromMap(mapFromJSON(order.PayloadJSON), "proyecto", ""))
}

func poolUsaConectorPoolLocalCompartido(pool *PoolCapacidad) bool {
	if pool == nil {
		return false
	}
	return planificadorpolicy.PoolUsesSharedLocalConnector(pool.Runtime, pool.MetadataJSON)
}

func existeRuntimeOrderAbierta(agente string, proyectoID *int64, tipos ...string) (bool, error) {
	if len(tipos) == 0 {
		return false, nil
	}
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{
		Agente:     stringsPtrTrimmed(agente),
		ProyectoID: proyectoID,
		Estado:     &estado,
		Tipos:      append([]string(nil), tipos...),
		Limit:      1,
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if order == nil {
			continue
		}
		return true, nil
	}
	return false, nil
}
