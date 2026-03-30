package db

import (
	"testing"
)

func TestHistorialVotacionesProyecto(t *testing.T) {
	prepararDBTemporal(t)

	var proyectoID int64
	if err := DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}

	prop := &Propuesta{
		Codigo:       "OP-900",
		Titulo:       "Decision de base",
		Descripcion:  "Descripcion",
		Tipo:         "arquitectura",
		PropuestoPor: "alberto",
		Distribuidor: "alberto",
		ProyectoID:   &proyectoID,
	}
	propID, err := CrearPropuesta(prop)
	if err != nil {
		t.Fatalf("CrearPropuesta: %v", err)
	}
	if _, err := Votar(propID, "codex1", VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("Votar codex1: %v", err)
	}
	if _, err := Votar(propID, "codex2", VotoDesacuerdo, "mejor interfaz"); err != nil {
		t.Fatalf("Votar codex2: %v", err)
	}

	items, err := ListarHistorialVotacionesProyecto(proyectoID)
	if err != nil {
		t.Fatalf("ListarHistorialVotacionesProyecto: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 item, tengo %d", len(items))
	}
	if items[0].Codigo != "OP-900" {
		t.Fatalf("codigo inesperado: %s", items[0].Codigo)
	}
	if items[0].Acuerdo != 1 || items[0].Desacuerdo != 1 {
		t.Fatalf("conteo inesperado: %+v", items[0])
	}
	if len(items[0].Votos) == 0 {
		t.Fatalf("sin votos cargados")
	}
}

func TestGuardarYListarDecisionesProyecto(t *testing.T) {
	prepararDBTemporal(t)
	var proyectoID int64
	if err := DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}

	id, err := GuardarDecisionProyecto(&DecisionProyecto{
		ProyectoID:   proyectoID,
		Categoria:    "arquitectura",
		Titulo:       "Puerto de almacenamiento",
		Solucion:     "Servicio y repositorio",
		Motivo:       "Desacoplar SQLite",
		Alternativas: "SQL directo en handlers",
		Impacto:      "Permite MySQL",
	})
	if err != nil {
		t.Fatalf("GuardarDecisionProyecto: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}

	items, err := ListarDecisionesProyecto(proyectoID)
	if err != nil {
		t.Fatalf("ListarDecisionesProyecto: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 decision, tengo %d", len(items))
	}
	if items[0].Titulo != "Puerto de almacenamiento" {
		t.Fatalf("titulo inesperado: %s", items[0].Titulo)
	}
}

func TestGuardarYListarDecisionesProyectoLegacySchema(t *testing.T) {
	prepararDBTemporal(t)
	if _, err := DB.Exec(`DROP TABLE decisiones_proyecto`); err != nil {
		t.Fatalf("drop decisiones_proyecto: %v", err)
	}
	if _, err := DB.Exec(`
		CREATE TABLE decisiones_proyecto (
			id                       INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto                 TEXT    NOT NULL,
			titulo                   TEXT    NOT NULL,
			solucion_elegida         TEXT    NOT NULL DEFAULT '',
			motivo                   TEXT    NOT NULL DEFAULT '',
			alternativas_descartadas TEXT    NOT NULL DEFAULT '',
			impacto                  TEXT    NOT NULL DEFAULT 'medio'
			                              CHECK (impacto IN ('alto','medio','bajo')),
			propuesta_codigo         TEXT    NOT NULL DEFAULT '',
			tarea_id                 INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
			registrado_por           TEXT    NOT NULL DEFAULT '',
			created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		t.Fatalf("create legacy decisiones_proyecto: %v", err)
	}

	var proyectoID int64
	if err := DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}

	id, err := GuardarDecisionProyecto(&DecisionProyecto{
		ProyectoID:   proyectoID,
		Categoria:    "arquitectura",
		Titulo:       "Puerto de almacenamiento",
		Solucion:     "Servicio y repositorio",
		Motivo:       "Desacoplar SQLite",
		Alternativas: "SQL directo en handlers",
		Impacto:      "Permite MySQL sin acoplamiento local",
	})
	if err != nil {
		t.Fatalf("GuardarDecisionProyecto legacy: %v", err)
	}
	if id == 0 {
		t.Fatalf("id legacy invalido")
	}

	items, err := ListarDecisionesProyecto(proyectoID)
	if err != nil {
		t.Fatalf("ListarDecisionesProyecto legacy: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 decision legacy, tengo %d", len(items))
	}
	if items[0].Titulo != "Puerto de almacenamiento" {
		t.Fatalf("titulo legacy inesperado: %s", items[0].Titulo)
	}
	if items[0].ProyectoID != proyectoID {
		t.Fatalf("proyecto_id legacy inesperado: %+v", items[0])
	}
	if items[0].Impacto != "medio" {
		t.Fatalf("impacto legacy inesperado: %+v", items[0])
	}
}

func TestGuardarYListarDocumentosExternosProyecto(t *testing.T) {
	prepararDBTemporal(t)
	var proyectoID int64
	if err := DB.QueryRow(
		`INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo) VALUES (?,?,?,?,1) RETURNING id`,
		"orquestador", "Orquestador", "/tmp/orquestador", "repo",
	).Scan(&proyectoID); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}

	id, err := GuardarDocumentoExterno(&DocumentoExterno{
		ProyectoID:    proyectoID,
		TipoDocumento: "markdown",
		Titulo:        "ADR-001",
		RutaRef:       "/tmp/adr-001.md",
		Resumen:       "Decision base",
	})
	if err != nil {
		t.Fatalf("GuardarDocumentoExterno: %v", err)
	}
	if id == 0 {
		t.Fatalf("id invalido")
	}

	items, err := ListarDocumentosExternosProyecto(proyectoID)
	if err != nil {
		t.Fatalf("ListarDocumentosExternosProyecto: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("esperaba 1 documento, tengo %d", len(items))
	}
	if items[0].RutaRef != "/tmp/adr-001.md" {
		t.Fatalf("ruta inesperada: %s", items[0].RutaRef)
	}
}
