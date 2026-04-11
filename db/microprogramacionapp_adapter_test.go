package db

import (
	"testing"

	"orquesta/microprogramacionapp"
)

func TestSqliteMicroprogramacionRepoCreaYListaEspecificaciones(t *testing.T) {
	prepararDBTemporal(t)
	repo := SqliteMicroprogramacionRepo{}
	taskID, err := CrearTarea(&Tarea{
		Titulo:      "micro tarea",
		Descripcion: "definir contrato",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	specID, err := repo.CrearEspecificacionFuncion(&microprogramacionapp.EspecificacionFuncion{
		TareaID:                &taskID,
		Titulo:                 "cmd/api.go::apiHandler",
		ArchivoObjetivo:        "cmd/api.go",
		SimboloObjetivo:        "apiHandler",
		Descripcion:            "Implementar handler",
		Precondiciones:         []string{"existe runtime"},
		Postcondiciones:        []string{"devuelve 200"},
		DependenciasPermitidas: []string{"orquesta/runtimesapp"},
		DependenciasProhibidas: []string{"orquesta/db"},
		TestsObligatorios:      []string{"go test ./cmd -run TestAPI"},
		WriteSet:               []string{"cmd/api.go"},
		FormatoSalida:          "patch+evidencia",
		Estado:                 microprogramacionapp.EstadoEspecificacionActiva,
		Version:                1,
		CreadoPor:              "alberto",
	})
	if err != nil {
		t.Fatalf("crear especificacion: %v", err)
	}
	got, err := repo.ObtenerEspecificacionFuncion(specID)
	if err != nil {
		t.Fatalf("obtener especificacion: %v", err)
	}
	if got.ArchivoObjetivo != "cmd/api.go" || got.SimboloObjetivo != "apiHandler" {
		t.Fatalf("spec inesperada: %+v", got)
	}
	lista, err := repo.ListarEspecificacionesFuncion(microprogramacionapp.FiltroEspecificaciones{TareaID: &taskID})
	if err != nil {
		t.Fatalf("listar especificaciones: %v", err)
	}
	if len(lista) != 1 || lista[0].ID != specID {
		t.Fatalf("lista inesperada: %+v", lista)
	}
}
