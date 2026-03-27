package db

import (
	"testing"
)

func TestGuardarPoolYListarResumen(t *testing.T) {
	withTempDBPools(t, func() {
		if _, err := GuardarPool(&PoolCapacidad{
			Slug:                "codex",
			Proveedor:           "OpenAI",
			Runtime:             "codex",
			Plan:                "default",
			EsDePago:            true,
			CapacidadTotal:      4,
			CapacidadReservada:  1,
			PermiteHijos:        true,
			PermiteModelosMulti: true,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		}); err != nil {
			t.Fatalf("GuardarPool: %v", err)
		}

		pool, err := GetPool("codex")
		if err != nil {
			t.Fatalf("GetPool: %v", err)
		}
		if pool.CapacidadTotal != 4 || pool.CapacidadReservada != 1 {
			t.Fatalf("pool inesperado: %+v", pool)
		}

		if _, err := DB.Exec(`UPDATE sesiones SET pool_id = ? WHERE agente = 'codex1' AND activa = 1`, pool.ID); err != nil {
			t.Fatalf("update sesiones.pool_id: %v", err)
		}

		resumen, err := ListarPoolsResumen(nil)
		if err != nil {
			t.Fatalf("ListarPoolsResumen: %v", err)
		}
		if len(resumen) != 1 {
			t.Fatalf("resumen inesperado: %+v", resumen)
		}
		if resumen[0].SesionesActivas != 1 {
			t.Fatalf("sesiones activas inesperadas: %+v", resumen[0])
		}
		if resumen[0].CapacidadDisponible != 2 {
			t.Fatalf("capacidad disponible inesperada: %+v", resumen[0])
		}
	})
}

func TestGuardarPoolModeloYListar(t *testing.T) {
	withTempDBPools(t, func() {
		if _, err := GuardarPool(&PoolCapacidad{
			Slug:                "codex",
			Proveedor:           "OpenAI",
			Runtime:             "codex",
			Plan:                "default",
			EsDePago:            true,
			CapacidadTotal:      4,
			CapacidadReservada:  0,
			PermiteHijos:        true,
			PermiteModelosMulti: true,
			PermiteSobrecoste:   false,
			PoliticaHandoff:     "preventivo",
			FuenteTelemetria:    "manual",
			MetadataJSON:        "{}",
			Activo:              true,
		}); err != nil {
			t.Fatalf("GuardarPool: %v", err)
		}

		if _, err := GuardarPoolModelo("codex", &PoolModelo{
			ModelSlug:          "gpt-5.4",
			Activo:             true,
			Prioridad:          10,
			CosteRelativo:      1.5,
			LimiteConocidoJSON: `{"window":"5h"}`,
		}); err != nil {
			t.Fatalf("GuardarPoolModelo 1: %v", err)
		}
		if _, err := GuardarPoolModelo("codex", &PoolModelo{
			ModelSlug:          "gpt-5.4-mini",
			Activo:             true,
			Prioridad:          20,
			CosteRelativo:      0.5,
			LimiteConocidoJSON: `{}`,
		}); err != nil {
			t.Fatalf("GuardarPoolModelo 2: %v", err)
		}

		modelos, err := ListarModelosPool("codex")
		if err != nil {
			t.Fatalf("ListarModelosPool: %v", err)
		}
		if len(modelos) != 2 {
			t.Fatalf("número de modelos inesperado: %+v", modelos)
		}
		if modelos[0].ModelSlug != "gpt-5.4" || modelos[1].ModelSlug != "gpt-5.4-mini" {
			t.Fatalf("orden/prioridad inesperada: %+v", modelos)
		}
	})
}

func withTempDBPools(t *testing.T, fn func()) {
	t.Helper()
	prepararDBTemporal(t)
	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO sesiones (agente, activa) VALUES ('codex1', 1)`); err != nil {
		t.Fatalf("insert sesion: %v", err)
	}
	fn()
}
