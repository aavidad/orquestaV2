package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestUpsertEntidadMemoriaInsertaYActualiza(t *testing.T) {
	tmp := prepararDBTemporal(t)

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

	id, err := UpsertEntidadMemoria(&EntidadMemoria{
		Nombre:        "Core_API",
		Tipo:          string(EntidadMemoriaAPI),
		ValorJSON:     `{"version":"v2","grpc":true}`,
		MetadataJSON:  `{"fuente":"manual"}`,
		VerificadoPor: "Codex1",
		ProyectoID:    &proyectoID,
	})
	if err != nil {
		t.Fatalf("insert entidad memoria: %v", err)
	}
	if id == 0 {
		t.Fatalf("id inesperado: %d", id)
	}

	entidad, err := GetEntidadMemoria("Core_API", &proyectoID)
	if err != nil {
		t.Fatalf("get entidad memoria: %v", err)
	}
	if entidad == nil || entidad.ID != id {
		t.Fatalf("entidad inesperada: %+v", entidad)
	}
	if entidad.VerificadoPor != "Codex1" {
		t.Fatalf("verificado_por inesperado: %+v", entidad)
	}

	actualizadoID, err := UpsertEntidadMemoria(&EntidadMemoria{
		Nombre:             "Core_API",
		Tipo:               string(EntidadMemoriaAPI),
		ValorJSON:          `{"version":"v3","grpc":true}`,
		MetadataJSON:       `{"fuente":"runtime"}`,
		VerificadoPor:      "Codex2",
		ProyectoID:         &proyectoID,
		UltimaVerificacion: time.Now().UTC().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("update entidad memoria: %v", err)
	}
	if actualizadoID != id {
		t.Fatalf("upsert deberia reutilizar id=%d, got=%d", id, actualizadoID)
	}

	entidad, err = GetEntidadMemoria("Core_API", &proyectoID)
	if err != nil {
		t.Fatalf("get entidad actualizada: %v", err)
	}
	if entidad == nil || entidad.ValorJSON != `{"version":"v3","grpc":true}` {
		t.Fatalf("valor actualizado inesperado: %+v", entidad)
	}
	if entidad.VerificadoPor != "Codex2" {
		t.Fatalf("verificado_por actualizado inesperado: %+v", entidad)
	}
}

func TestListarEntidadesMemoriaFiltraPorProyectoYTipo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "dashboard",
		Nombre:  "Dashboard",
		RutaAbs: filepath.Join(tmp, "dashboard"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}

	for _, entidad := range []*EntidadMemoria{
		{Nombre: "Core_API", Tipo: string(EntidadMemoriaAPI), ValorJSON: `{"version":"v2"}`, ProyectoID: &proyectoA},
		{Nombre: "DB_Cluster", Tipo: string(EntidadMemoriaInfra), ValorJSON: `{"readonly":true}`, ProyectoID: &proyectoA},
		{Nombre: "Widget_UI", Tipo: string(EntidadMemoriaAPI), ValorJSON: `{"framework":"htmx"}`, ProyectoID: &proyectoB},
	} {
		if _, err := UpsertEntidadMemoria(entidad); err != nil {
			t.Fatalf("upsert entidad %+v: %v", entidad, err)
		}
	}

	tipo := string(EntidadMemoriaAPI)
	lista, err := ListarEntidadesMemoria(FiltroEntidadesMemoria{
		ProyectoID: &proyectoA,
		Tipo:       &tipo,
	})
	if err != nil {
		t.Fatalf("listar entidades memoria: %v", err)
	}
	if len(lista) != 1 || lista[0].Nombre != "Core_API" {
		t.Fatalf("lista filtrada inesperada: %+v", lista)
	}
}
