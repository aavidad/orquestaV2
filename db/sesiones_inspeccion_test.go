package db

import (
	"path/filepath"
	"testing"
)

func prepararDBTemporalInspeccionSesiones(t *testing.T) string {
	t.Helper()
	return prepararDBTemporalConNombre(t, "orquesta-inspeccion-sesiones-test.db")
}

func TestListarSesionesInspeccionFiltraPorAgenteProyectoActivaYEstado(t *testing.T) {
	tmp := prepararDBTemporalInspeccionSesiones(t)

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

	sesion1, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-001",
		ResumenContinuidad: "primera sesion",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 1: %v", err)
	}
	if err := GuardarSesionActiva("Codex1", &proyectoID, SesionUpdate{Estado: stringPtr("pausada"), Heartbeat: true}); err != nil {
		t.Fatalf("guardar estado pausada: %v", err)
	}
	if err := FinSesion("Codex1"); err != nil {
		t.Fatalf("fin sesion 1: %v", err)
	}

	sesion2, err := IniciarSesionContexto(SesionInicio{
		Agente:             "Codex1",
		ProyectoID:         &proyectoID,
		CWD:                filepath.Join(tmp, "orquestador"),
		Herramienta:        "codex-cli",
		ExternalSessionID:  "sess-002",
		ResumenContinuidad: "segunda sesion",
	})
	if err != nil {
		t.Fatalf("iniciar sesion 2: %v", err)
	}

	t.Run("listado global", func(t *testing.T) {
		lista, err := ListarSesionesInspeccion(FiltroSesionesInspeccion{})
		if err != nil {
			t.Fatalf("listar sesiones: %v", err)
		}
		if len(lista) != 2 {
			t.Fatalf("esperaba 2 sesiones, got=%d", len(lista))
		}
	})

	t.Run("filtro por agente", func(t *testing.T) {
		agente := "Codex1"
		lista, err := ListarSesionesInspeccion(FiltroSesionesInspeccion{Agente: &agente})
		if err != nil {
			t.Fatalf("listar por agente: %v", err)
		}
		if len(lista) != 2 {
			t.Fatalf("esperaba 2 sesiones por agente, got=%d", len(lista))
		}
	})

	t.Run("filtro por proyecto", func(t *testing.T) {
		lista, err := ListarSesionesInspeccion(FiltroSesionesInspeccion{ProyectoID: &proyectoID})
		if err != nil {
			t.Fatalf("listar por proyecto: %v", err)
		}
		if len(lista) != 2 {
			t.Fatalf("esperaba 2 sesiones por proyecto, got=%d", len(lista))
		}
	})

	t.Run("filtro activas", func(t *testing.T) {
		activa := true
		lista, err := ListarSesionesInspeccion(FiltroSesionesInspeccion{Activa: &activa})
		if err != nil {
			t.Fatalf("listar activas: %v", err)
		}
		if len(lista) != 1 {
			t.Fatalf("esperaba 1 sesión activa, got=%d", len(lista))
		}
		if lista[0].ID != sesion2.ID {
			t.Fatalf("sesion activa inesperada: got=%d want=%d", lista[0].ID, sesion2.ID)
		}
	})

	t.Run("filtro cerradas", func(t *testing.T) {
		activa := false
		estado := "cerrada"
		lista, err := ListarSesionesInspeccion(FiltroSesionesInspeccion{Activa: &activa, Estado: &estado})
		if err != nil {
			t.Fatalf("listar cerradas: %v", err)
		}
		if len(lista) != 1 {
			t.Fatalf("esperaba 1 sesión cerrada, got=%d", len(lista))
		}
		if lista[0].ID != sesion1.ID {
			t.Fatalf("sesion cerrada inesperada: got=%d want=%d", lista[0].ID, sesion1.ID)
		}
	})

	t.Run("detalle por id", func(t *testing.T) {
		sesion, err := GetSesionInspeccionByID(sesion2.ID)
		if err != nil {
			t.Fatalf("get detalle: %v", err)
		}
		if sesion.ID != sesion2.ID || sesion.ExternalSessionID != "sess-002" {
			t.Fatalf("sesion detalle inesperada: %+v", sesion)
		}
	})
}

func TestListarSesionesInspeccionCanonicalizaAliasCodex(t *testing.T) {
	tmp := prepararDBTemporalInspeccionSesiones(t)

	for _, nombre := range []string{"Codex81", "codex81"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "sesiones-canon",
		Nombre:  "Sesiones Canon",
		RutaAbs: filepath.Join(tmp, "sesiones-canon"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex81",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "sesiones-canon"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}

	agente := "codex81"
	lista, err := ListarSesionesInspeccion(FiltroSesionesInspeccion{Agente: &agente})
	if err != nil {
		t.Fatalf("ListarSesionesInspeccion: %v", err)
	}
	if len(lista) != 1 || lista[0] == nil || lista[0].Agente != "Codex81" {
		t.Fatalf("sesiones inesperadas: %+v", lista)
	}
}

func stringPtr(v string) *string {
	return &v
}
