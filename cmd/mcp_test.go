/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestMCPHandleInitializeNegociaVersion(t *testing.T) {
	t.Helper()

	srv := &mcpServer{}
	params := mustJSON(t, map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
	})
	result, rpcErr := srv.handleRequest(mcpRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params:  params,
	})
	if rpcErr != nil {
		t.Fatalf("initialize devolvio error: %+v", rpcErr)
	}

	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("resultado inesperado: %#v", result)
	}
	if payload["protocolVersion"] != "2025-06-18" {
		t.Fatalf("version inesperada: %#v", payload["protocolVersion"])
	}
	if !srv.initialized {
		t.Fatalf("el servidor no quedo inicializado")
	}
}

func TestMCPResourceReadDetallePropuesta(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		insertTestPropuesta(t, "OP-049", "Arquitectura objetivo", "MCP como adaptador de entrada")

		contents, err := readMCPResource("orquesta://propuestas/OP-049")
		if err != nil {
			t.Fatalf("readMCPResource: %v", err)
		}
		if len(contents) != 1 {
			t.Fatalf("contents inesperado: %#v", contents)
		}
		text, _ := contents[0]["text"].(string)
		if !strings.Contains(text, "OP-049") {
			t.Fatalf("el recurso no contiene la propuesta: %s", text)
		}
		if !strings.Contains(text, "MCP como adaptador de entrada") {
			t.Fatalf("el recurso no contiene la descripcion: %s", text)
		}
	})
}

func TestMCPPromptBriefingIncluyeReglasYPropuestasPendientes(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		propuestaID := insertTestPropuesta(t, "OP-049", "Arquitectura objetivo", "Descripcion")
		insertTestVotoPendiente(t, propuestaID, "Codex2")

		result, err := getMCPPrompt("orquesta.briefing.agente", map[string]any{"agente": "Codex2"})
		if err != nil {
			t.Fatalf("getMCPPrompt: %v", err)
		}
		messages, ok := result["messages"].([]mcpPromptMessage)
		if !ok || len(messages) != 1 {
			t.Fatalf("mensajes inesperados: %#v", result["messages"])
		}
		content, _ := messages[0].Content.(map[string]any)
		text, _ := content["text"].(string)
		if !strings.Contains(text, "Propuestas pendientes de voto") {
			t.Fatalf("faltan propuestas pendientes en el briefing: %s", text)
		}
		if !strings.Contains(text, "Reglas activas") {
			t.Fatalf("faltan reglas activas en el briefing: %s", text)
		}
	})
}

func TestMCPToolVotarPropuestaActualizaEstado(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		if err := db.ConfigSet("propuesta_min_votes", "1"); err != nil {
			t.Fatalf("config propuesta_min_votes: %v", err)
		}
		if err := db.ConfigSet("propuesta_min_non_author_votes", "1"); err != nil {
			t.Fatalf("config propuesta_min_non_author_votes: %v", err)
		}
		propuestaID := insertTestPropuesta(t, "OP-049", "Arquitectura objetivo", "Descripcion")
		insertTestVotoPendiente(t, propuestaID, "Codex2")

		result, err := callMCPTool("orquesta.propuestas.votar", map[string]any{
			"codigo":     "OP-049",
			"agente":     "Codex2",
			"posicion":   "acuerdo",
			"comentario": "Coherente y evolutiva",
		})
		if err != nil {
			t.Fatalf("callMCPTool: %v", err)
		}
		if result["isError"] != false {
			t.Fatalf("resultado marcado como error: %#v", result)
		}

		p, err := db.GetPropuesta("OP-049")
		if err != nil {
			t.Fatalf("leyendo propuesta: %v", err)
		}
		if p.Estado != db.PropuestaConsenso {
			t.Fatalf("estado inesperado tras votar: %s", p.Estado)
		}
	})
}

func TestMCPResourceReadProyectoIncluyeWorktreesYAsignaciones(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		insertTestAsignacion(t, "Codex2", proyectoID, "mcp server y recursos")
		insertTestWorktree(t, proyectoID, "Codex2", "orquestador-codex2", "/tmp/orquestador/.orquesta-worktrees/orquestador-codex2", "orq-orquestador-codex2")

		contents, err := readMCPResource("orquesta://proyectos/orquestador")
		if err != nil {
			t.Fatalf("readMCPResource proyecto: %v", err)
		}
		text, _ := contents[0]["text"].(string)
		if !strings.Contains(text, "orquestador-codex2") {
			t.Fatalf("faltan worktrees en el recurso: %s", text)
		}
		if !strings.Contains(text, "mcp server y recursos") {
			t.Fatalf("faltan asignaciones en el recurso: %s", text)
		}
	})
}

func TestMCPToolListaProyectosFiltrados(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		insertTestProyectoConTipoActivo(t, "grupo-pm", "PlataformaMunicipal", "/tmp/PlataformaMunicipal", "grupo", true)
		insertTestProyectoConTipoActivo(t, "repo-inactivo", "repo-inactivo", "/tmp/repo-inactivo", "repo", false)

		result, err := callMCPTool("orquesta.proyectos.listar", map[string]any{
			"tipo":   "repo",
			"activo": true,
		})
		if err != nil {
			t.Fatalf("callMCPTool proyectos.listar: %v", err)
		}
		text := result["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(text, `"slug": "orquestador"`) {
			t.Fatalf("no aparece el repo esperado: %s", text)
		}
		if strings.Contains(text, "repo-inactivo") {
			t.Fatalf("aparece un proyecto inactivo filtrado: %s", text)
		}
		if strings.Contains(text, "grupo-pm") {
			t.Fatalf("aparece un proyecto de otro tipo: %s", text)
		}
	})
}

func TestMCPPromptContextoProyectoIncluyeBloquesOperativos(t *testing.T) {
	withTempOrquestaDB(t, func() {
		if err := db.RegistrarAgente("Codex2", "programador"); err != nil {
			t.Fatalf("registrando agente: %v", err)
		}
		proyectoID := insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")
		insertTestAsignacion(t, "Codex2", proyectoID, "mcp server y recursos")

		result, err := getMCPPrompt("orquesta.contexto.proyecto", map[string]any{"slug": "orquestador"})
		if err != nil {
			t.Fatalf("getMCPPrompt contexto.proyecto: %v", err)
		}
		messages := result["messages"].([]mcpPromptMessage)
		content := messages[0].Content.(map[string]any)
		text := content["text"].(string)
		if !strings.Contains(text, "Asignaciones activas") {
			t.Fatalf("faltan asignaciones en prompt: %s", text)
		}
		if !strings.Contains(text, "siguiente bloque tecnico con mejor retorno") {
			t.Fatalf("falta la guia de salida esperada: %s", text)
		}
	})
}

func TestMCPResourceReadPoolIncluyeModelosYCapacidad(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestPool(t, "codex", "OpenAI", "codex", 4, 1)
		insertTestPoolModelo(t, "codex", "gpt-5.4", 10, 1.5)

		contents, err := readMCPResource("orquesta://pools/codex")
		if err != nil {
			t.Fatalf("readMCPResource pool: %v", err)
		}
		text, _ := contents[0]["text"].(string)
		if !strings.Contains(text, "gpt-5.4") {
			t.Fatalf("faltan modelos en el recurso: %s", text)
		}
		if !strings.Contains(text, "capacidad_disponible") {
			t.Fatalf("falta resumen de capacidad en el recurso: %s", text)
		}
	})
}

func TestMCPToolListaPoolsYModelos(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestPool(t, "codex", "OpenAI", "codex", 4, 1)
		insertTestPoolModelo(t, "codex", "gpt-5.4", 10, 1.5)

		poolsResult, err := callMCPTool("orquesta.pools.listar", map[string]any{})
		if err != nil {
			t.Fatalf("callMCPTool pools.listar: %v", err)
		}
		poolsText := poolsResult["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(poolsText, `"slug": "codex"`) {
			t.Fatalf("no aparece el pool esperado: %s", poolsText)
		}

		modelosResult, err := callMCPTool("orquesta.pools.modelos", map[string]any{"pool_slug": "codex"})
		if err != nil {
			t.Fatalf("callMCPTool pools.modelos: %v", err)
		}
		modelosText := modelosResult["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(modelosText, "gpt-5.4") {
			t.Fatalf("no aparece el modelo esperado: %s", modelosText)
		}
	})
}

func TestMCPToolResuelveModeloPorPolitica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestPool(t, "codex", "OpenAI", "codex", 4, 0)
		insertTestPoolModelo(t, "codex", "gpt-5.4", 10, 1.5)
		insertTestPoolModelo(t, "codex", "gpt-5.4-mini", 20, 0.5)
		insertTestPoliticaModelo(t, db.PoliticaModelo{
			ScopeTipo:       "perfil",
			ScopeRef:        "script",
			PerfilTarea:     "script",
			PoolSlug:        "codex",
			ReasoningEffort: "medium",
			Prioridad:       10,
			Activa:          true,
		})

		result, err := callMCPTool("orquesta.modelos.resolver", map[string]any{
			"perfil": "script",
		})
		if err != nil {
			t.Fatalf("callMCPTool modelos.resolver: %v", err)
		}
		text := result["content"].([]map[string]any)[0]["text"].(string)
		if !strings.Contains(text, `"pool_slug": "codex"`) {
			t.Fatalf("pool no resuelto correctamente: %s", text)
		}
		if !strings.Contains(text, `"model_slug": "gpt-5.4-mini"`) {
			t.Fatalf("modelo economico no resuelto: %s", text)
		}
		if !strings.Contains(text, `"reasoning_effort": "medium"`) {
			t.Fatalf("reasoning no resuelto: %s", text)
		}
	})
}

func TestMCPResourceReadPoliticasModelo(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestPool(t, "codex", "OpenAI", "codex", 4, 0)
		insertTestPoliticaModelo(t, db.PoliticaModelo{
			ScopeTipo:       "perfil",
			ScopeRef:        "orquestacion",
			PerfilTarea:     "orquestacion",
			PoolSlug:        "codex",
			ReasoningEffort: "xhigh",
			Prioridad:       10,
			Activa:          true,
		})

		contents, err := readMCPResource("orquesta://politicas-modelo")
		if err != nil {
			t.Fatalf("readMCPResource politicas-modelo: %v", err)
		}
		text := contents[0]["text"].(string)
		if !strings.Contains(text, `"scope_tipo": "perfil"`) {
			t.Fatalf("falta scope_tipo en recurso: %s", text)
		}
		if !strings.Contains(text, `"reasoning_effort": "xhigh"`) {
			t.Fatalf("falta reasoning en recurso: %s", text)
		}
	})
}

func withTempOrquestaDB(t *testing.T, fn func()) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	previous := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("ORQUESTA_DB", previous)
		db.Close()
	})

	if err := db.Open(); err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	fn()
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

func insertTestPropuesta(t *testing.T, codigo, titulo, descripcion string) int64 {
	t.Helper()
	var id int64
	if err := db.DB.QueryRow(`
		INSERT INTO propuestas (codigo, titulo, descripcion, tipo, propuesto_por, distribuidor)
		VALUES (?,?,?,?,?,?)
		RETURNING id`,
		codigo, titulo, descripcion, "arquitectura", "Codex1", "Codex1",
	).Scan(&id); err != nil {
		t.Fatalf("insert propuesta: %v", err)
	}
	return id
}

func insertTestVotoPendiente(t *testing.T, propuestaID int64, agente string) {
	t.Helper()
	if _, err := db.DB.Exec(
		`INSERT INTO votos (propuesta_id, agente, posicion, comentario) VALUES (?,?,?,?)`,
		propuestaID, agente, db.VotoPendiente, "",
	); err != nil {
		t.Fatalf("insert voto pendiente: %v", err)
	}
}

func insertTestProyecto(t *testing.T, slug, nombre, rutaAbs string) int64 {
	t.Helper()
	return insertTestProyectoConTipoActivo(t, slug, nombre, rutaAbs, "repo", true)
}

func insertTestProyectoConTipoActivo(t *testing.T, slug, nombre, rutaAbs, tipo string, activo bool) int64 {
	t.Helper()
	var id int64
	if err := db.DB.QueryRow(`
		INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo)
		VALUES (?,?,?,?,?)
		RETURNING id`,
		slug, nombre, rutaAbs, tipo, activo,
	).Scan(&id); err != nil {
		t.Fatalf("insert proyecto: %v", err)
	}
	return id
}

func insertTestAsignacion(t *testing.T, agente string, proyectoID int64, nota string) {
	t.Helper()
	if _, err := db.DB.Exec(`
		INSERT INTO asignaciones (agente, proyecto_id, estado, nota)
		VALUES (?,?,?,?)`,
		agente, proyectoID, "activa", nota,
	); err != nil {
		t.Fatalf("insert asignacion: %v", err)
	}
}

func insertTestWorktree(t *testing.T, proyectoID int64, agente, nombre, rutaAbs, branch string) {
	t.Helper()
	if _, err := db.DB.Exec(`
		INSERT INTO worktrees (proyecto_id, agente, nombre, ruta_abs, branch, estado)
		VALUES (?,?,?,?,?,?)`,
		proyectoID, agente, nombre, rutaAbs, branch, "activa",
	); err != nil {
		t.Fatalf("insert worktree: %v", err)
	}
}

func insertTestPool(t *testing.T, slug, proveedor, runtime string, capacidadTotal, capacidadReservada int) {
	t.Helper()
	if _, err := db.GuardarPool(&db.PoolCapacidad{
		Slug:                slug,
		Proveedor:           proveedor,
		Runtime:             runtime,
		Plan:                "default",
		EsDePago:            true,
		CapacidadTotal:      capacidadTotal,
		CapacidadReservada:  capacidadReservada,
		PermiteHijos:        true,
		PermiteModelosMulti: true,
		PermiteSobrecoste:   false,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        "{}",
		Activo:              true,
	}); err != nil {
		t.Fatalf("insert pool: %v", err)
	}
}

func insertTestPoolModelo(t *testing.T, poolSlug, modelSlug string, prioridad int, coste float64) {
	t.Helper()
	if _, err := db.GuardarPoolModelo(poolSlug, &db.PoolModelo{
		ModelSlug:          modelSlug,
		Activo:             true,
		Prioridad:          prioridad,
		CosteRelativo:      coste,
		LimiteConocidoJSON: "{}",
	}); err != nil {
		t.Fatalf("insert pool modelo: %v", err)
	}
}

func insertTestPoliticaModelo(t *testing.T, politica db.PoliticaModelo) {
	t.Helper()
	if _, err := db.GuardarPoliticaModelo(&politica); err != nil {
		t.Fatalf("insert politica modelo: %v", err)
	}
}
