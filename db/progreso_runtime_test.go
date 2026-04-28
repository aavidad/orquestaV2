/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestProgresoEstimadoPorSenales(t *testing.T) {
	prepararDBTemporal(t)

	// Tarea inexistente → 0, nil
	pct, err := ProgresoEstimadoPorSenales(9999)
	if err != nil {
		t.Fatalf("tarea inexistente: error inesperado: %v", err)
	}
	if pct != 0 {
		t.Fatalf("tarea inexistente: esperaba 0, got %d", pct)
	}

	// Registrar agente y crear tarea asignada
	if err := RegistrarAgente("TestAgenteProgreso", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Test tarea progreso",
		CreadoPor: "test",
		Prioridad: "media",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "TestAgenteProgreso"); err != nil {
		t.Fatalf("asignar tarea: %v", err)
	}

	// Sin señales → 0
	pct, err = ProgresoEstimadoPorSenales(tareaID)
	if err != nil {
		t.Fatalf("sin señales: error inesperado: %v", err)
	}
	if pct != 0 {
		t.Fatalf("sin señales: esperaba 0, got %d", pct)
	}

	// Insertar 2 checkpoints → 40% (2×20)
	for range 2 {
		if _, err := DB.Exec(`INSERT INTO runtime_checkpoints (agente, checkpoint_kind, resumen) VALUES (?,?,?)`,
			"TestAgenteProgreso", "manual", "test"); err != nil {
			t.Fatalf("insertar checkpoint: %v", err)
		}
	}
	pct, err = ProgresoEstimadoPorSenales(tareaID)
	if err != nil {
		t.Fatalf("2 checkpoints: error inesperado: %v", err)
	}
	if pct != 40 {
		t.Fatalf("2 checkpoints: esperaba 40, got %d", pct)
	}

	// 4 checkpoints → 60% (tope), sin mailbox
	for range 2 {
		if _, err := DB.Exec(`INSERT INTO runtime_checkpoints (agente, checkpoint_kind, resumen) VALUES (?,?,?)`,
			"TestAgenteProgreso", "manual", "test"); err != nil {
			t.Fatalf("insertar checkpoint: %v", err)
		}
	}
	pct, err = ProgresoEstimadoPorSenales(tareaID)
	if err != nil {
		t.Fatalf("4 checkpoints: error inesperado: %v", err)
	}
	if pct != 60 {
		t.Fatalf("4 checkpoints (tope): esperaba 60, got %d", pct)
	}

	// Añadir 6 mensajes mailbox consumidos → +20% mailbox (tope 40% daría 60+20=80 con tope)
	for range 6 {
		if _, err := DB.Exec(`INSERT INTO runtime_mailbox (from_agente, to_agente, kind, estado) VALUES (?,?,?,?)`,
			"server", "TestAgenteProgreso", "nudge", "consumido"); err != nil {
			t.Fatalf("insertar mailbox: %v", err)
		}
	}
	pct, err = ProgresoEstimadoPorSenales(tareaID)
	if err != nil {
		t.Fatalf("4 ckpt + 6 mailbox: error inesperado: %v", err)
	}
	// checkpointPct=60 + mailboxPct=(6/3)*10=20 = 80
	if pct != 80 {
		t.Fatalf("4 ckpt + 6 mailbox: esperaba 80, got %d", pct)
	}
}

func TestProgresoEstimadoPorSenalesCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex91", "codex91"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Test tarea progreso canon",
		CreadoPor: "test",
		Prioridad: "media",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "codex91"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO runtime_checkpoints (agente, checkpoint_kind, resumen) VALUES (?,?,?)`,
		"Codex91", "manual", "test"); err != nil {
		t.Fatalf("insertar checkpoint: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO runtime_mailbox (from_agente, to_agente, kind, estado) VALUES (?,?,?,?)`,
		"server", "Codex91", "nudge", "consumido"); err != nil {
		t.Fatalf("insertar mailbox: %v", err)
	}

	pct, err := ProgresoEstimadoPorSenales(tareaID)
	if err != nil {
		t.Fatalf("ProgresoEstimadoPorSenales: %v", err)
	}
	if pct != 20 {
		t.Fatalf("progreso inesperado: got=%d want=20", pct)
	}
}
