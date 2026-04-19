/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"path/filepath"
	"testing"
)

func TestRuntimeOrdersHotIndexSincronizaMutacionesCanonicas(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}

	estadoPendiente := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar runtime orders vivas: %v", err)
	}
	if len(orders) != 1 || orders[0].ID != orderID {
		t.Fatalf("runtime orders vivas inesperadas: %+v", orders)
	}
	assertRuntimeOrdersHotIndexState(t, false, orderID, "pendiente")

	claimed, err := ClaimNextRuntimeOrder("Codex1")
	if err != nil {
		t.Fatalf("claim runtime order: %v", err)
	}
	if claimed == nil || claimed.ID != orderID || claimed.Estado != "tomada" {
		t.Fatalf("claim inesperado: %+v", claimed)
	}
	assertRuntimeOrdersHotIndexState(t, false, orderID, "tomada")

	if err := MarcarRuntimeOrderEstado(orderID, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("completar runtime order: %v", err)
	}
	runtimeOrdersHotIndex.mu.RLock()
	defer runtimeOrdersHotIndex.mu.RUnlock()
	if runtimeOrdersHotIndex.dirty {
		t.Fatalf("el indice no deberia quedar dirty tras la mutacion canonica")
	}
	if got := runtimeOrdersHotIndex.byID[orderID]; got != nil {
		t.Fatalf("la orden completada no deberia seguir viva en el indice: %+v", got)
	}
	if ids := runtimeOrdersHotIndex.byState["completada"]; len(ids) != 0 {
		t.Fatalf("la orden terminal no deberia indexarse como viva: %+v", ids)
	}
}

func TestRuntimeOrdersHotIndexRecargaTrasTxDirecta(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar runtime order: %v", err)
	}

	estadoPendiente := "pendiente"
	if _, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estadoPendiente}); err != nil {
		t.Fatalf("precargar indice: %v", err)
	}

	tx, err := DB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if _, err := tx.Exec(`UPDATE runtime_orders SET estado='ejecutando' WHERE id=?`, orderID); err != nil {
		_ = tx.Rollback()
		t.Fatalf("update tx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	runtimeOrdersHotIndex.mu.RLock()
	dirty := runtimeOrdersHotIndex.dirty
	runtimeOrdersHotIndex.mu.RUnlock()
	if !dirty {
		t.Fatalf("una mutacion SQL directa deberia ensuciar el indice")
	}

	estadoEjecutando := "ejecutando"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estadoEjecutando})
	if err != nil {
		t.Fatalf("listar runtime orders recargadas: %v", err)
	}
	if len(orders) != 1 || orders[0].ID != orderID || orders[0].Estado != "ejecutando" {
		t.Fatalf("la recarga del indice no reflejo la tx directa: %+v", orders)
	}
	assertRuntimeOrdersHotIndexState(t, false, orderID, "ejecutando")
}

func TestRuntimeOrdersHotIndexRecargaCompletaSiHabiaDirtyPrevio(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	orderA, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar orderA: %v", err)
	}
	orderB, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar orderB: %v", err)
	}

	estadoPendiente := "pendiente"
	if _, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estadoPendiente}); err != nil {
		t.Fatalf("precargar indice: %v", err)
	}

	tx, err := DB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if _, err := tx.Exec(`UPDATE runtime_orders SET estado='ejecutando' WHERE id=?`, orderB); err != nil {
		_ = tx.Rollback()
		t.Fatalf("update tx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	if err := MarcarRuntimeOrderEstado(orderA, "completada", `{"ok":true}`, ""); err != nil {
		t.Fatalf("completar orderA: %v", err)
	}

	estadoEjecutando := "ejecutando"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Estado: &estadoEjecutando})
	if err != nil {
		t.Fatalf("listar runtime orders vivas: %v", err)
	}
	if len(orders) != 1 || orders[0].ID != orderB || orders[0].Estado != "ejecutando" {
		t.Fatalf("la recarga completa no reflejo el dirty previo: %+v", orders)
	}

	runtimeOrdersHotIndex.mu.RLock()
	defer runtimeOrdersHotIndex.mu.RUnlock()
	if runtimeOrdersHotIndex.dirty {
		t.Fatalf("el indice deberia quedar limpio tras la recarga completa")
	}
	if got := runtimeOrdersHotIndex.byID[orderA]; got != nil {
		t.Fatalf("orderA no deberia seguir viva en el indice: %+v", got)
	}
	if got := runtimeOrdersHotIndex.byID[orderB]; got == nil || got.Estado != "ejecutando" {
		t.Fatalf("orderB deberia reflejarse como ejecutando tras recarga completa: %+v", got)
	}
}

func TestRuntimeOrdersHotIndexRespetaFiltroPorTipo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "checkpoint",
		PayloadJSON: `{}`,
	}); err != nil {
		t.Fatalf("encolar checkpoint: %v", err)
	}
	nudgeID, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "nudge",
		PayloadJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("encolar nudge: %v", err)
	}

	estadoPendiente := "pendiente"
	agente := "Codex1"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{
		Agente:     &agente,
		ProyectoID: &proyectoID,
		Estado:     &estadoPendiente,
		Tipos:      []string{"nudge"},
	})
	if err != nil {
		t.Fatalf("listar runtime orders vivas por tipo: %v", err)
	}
	if len(orders) != 1 || orders[0].ID != nudgeID || orders[0].Tipo != "nudge" {
		t.Fatalf("filtro por tipo inesperado: %+v", orders)
	}
}

func assertRuntimeOrdersHotIndexState(t *testing.T, wantDirty bool, id int64, estado string) {
	t.Helper()
	runtimeOrdersHotIndex.mu.RLock()
	defer runtimeOrdersHotIndex.mu.RUnlock()
	if runtimeOrdersHotIndex.dirty != wantDirty {
		t.Fatalf("dirty inesperado: got=%t want=%t", runtimeOrdersHotIndex.dirty, wantDirty)
	}
	got := runtimeOrdersHotIndex.byID[id]
	if wantDirty {
		return
	}
	if got == nil {
		t.Fatalf("orden %d no indexada", id)
	}
	if got.Estado != estado {
		t.Fatalf("estado inesperado en indice: got=%s want=%s", got.Estado, estado)
	}
}
