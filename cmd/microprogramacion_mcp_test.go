package cmd

import (
	"strconv"
	"strings"
	"testing"

	"orquesta/microprogramacionapp"
)

func TestMCPToolsMicroprogramacionOperanPorLaViaCanonica(t *testing.T) {
	withTempOrquestaDB(t, func() {
		insertTestProyecto(t, "orquestador", "orquestador", "/tmp/orquestador")

		createResult, err := callMCPTool("orquesta.microprogramacion.especificaciones.crear", map[string]any{
			"proyecto":                "orquestador",
			"titulo":                  "ResolverControlHandle",
			"archivo_objetivo":        "runtimesapp/service.go",
			"simbolo_objetivo":        "ResolveControlHandle",
			"descripcion":             "Resuelve el handle de control canónico sin depender de db directa",
			"precondiciones":          []any{"Existe runtime activo o se devuelve nil de forma explícita"},
			"postcondiciones":         []any{"No toca rutas fuera del write-set"},
			"dependencias_permitidas": []any{"runtimesapp", "sesionesapp"},
			"dependencias_prohibidas": []any{"db"},
			"tests_obligatorios":      []any{"go test ./runtimesapp -run TestResolveControlHandle -count=1"},
			"write_set":               []any{"runtimesapp/service.go", "runtimesapp/service_test.go"},
			"formato_salida":          "patch+evidencia",
			"creado_por":              "alberto",
		})
		if err != nil {
			t.Fatalf("crear especificacion via MCP: %v", err)
		}

		createPayload, ok := createResult["structuredContent"].(apiEspecificacionFuncionCreateResponse)
		if !ok {
			t.Fatalf("structuredContent inesperado en create: %#v", createResult["structuredContent"])
		}
		if !createPayload.OK || createPayload.ID <= 0 || createPayload.Especificacion == nil {
			t.Fatalf("payload de create inesperado: %#v", createPayload)
		}
		if createPayload.Especificacion.SimboloObjetivo != "ResolveControlHandle" {
			t.Fatalf("simbolo inesperado: %#v", createPayload.Especificacion)
		}

		listResult, err := callMCPTool("orquesta.microprogramacion.especificaciones.listar", map[string]any{
			"proyecto": "orquestador",
		})
		if err != nil {
			t.Fatalf("listar especificaciones via MCP: %v", err)
		}
		listPayload, ok := listResult["structuredContent"].(apiEspecificacionesFuncionResponse)
		if !ok {
			t.Fatalf("structuredContent inesperado en listar: %#v", listResult["structuredContent"])
		}
		if len(listPayload.Especificaciones) != 1 {
			t.Fatalf("listado inesperado: %#v", listPayload.Especificaciones)
		}
		if listPayload.Especificaciones[0].ID != createPayload.ID {
			t.Fatalf("id inesperado en list: got=%d want=%d", listPayload.Especificaciones[0].ID, createPayload.ID)
		}

		viewResult, err := callMCPTool("orquesta.microprogramacion.especificaciones.ver", map[string]any{
			"id": createPayload.ID,
		})
		if err != nil {
			t.Fatalf("ver especificacion via MCP: %v", err)
		}
		viewPayload, ok := viewResult["structuredContent"].(apiEspecificacionFuncionResponse)
		if !ok {
			t.Fatalf("structuredContent inesperado en ver: %#v", viewResult["structuredContent"])
		}
		if viewPayload.Especificacion == nil || viewPayload.Especificacion.ArchivoObjetivo != "runtimesapp/service.go" {
			t.Fatalf("detalle inesperado: %#v", viewPayload.Especificacion)
		}

		tools := listMCPTools()
		if !containsMCPTool(tools, "orquesta.microprogramacion.especificaciones.crear") {
			t.Fatalf("no aparece la tool de crear en listMCPTools")
		}
		emitResult, err := callMCPTool("orquesta.microprogramacion.especificaciones.emitir", map[string]any{
			"id":       createPayload.ID,
			"contexto": "No abras frentes laterales.",
		})
		if err != nil {
			t.Fatalf("emitir microtarea via MCP: %v", err)
		}
		emitPayload, ok := emitResult["structuredContent"].(apiMicrotareaEmitidaResponse)
		if !ok {
			t.Fatalf("structuredContent inesperado en emitir: %#v", emitResult["structuredContent"])
		}
		if emitPayload.Microtarea == nil || !strings.Contains(emitPayload.Microtarea.Mensaje, "No abras frentes laterales.") {
			t.Fatalf("microtarea emitida inesperada: %#v", emitPayload.Microtarea)
		}
		validateResult, err := callMCPTool("orquesta.microprogramacion.especificaciones.validar_entrega", map[string]any{
			"id":                  createPayload.ID,
			"simbolo_entregado":   "ResolveControlHandle",
			"write_set_entregado": []any{"runtimesapp/service.go", "runtimesapp/service_test.go"},
			"dependencias_usadas": []any{"orquesta/runtimesapp"},
			"tests_ejecutados":    []any{"go test ./runtimesapp -run TestResolveControlHandle -count=1"},
			"evidencia":           "go test ./runtimesapp -run TestResolveControlHandle -count=1 => ok",
		})
		if err != nil {
			t.Fatalf("validar entrega via MCP: %v", err)
		}
		validatePayload, ok := validateResult["structuredContent"].(apiValidacionEntregaResponse)
		if !ok {
			t.Fatalf("structuredContent inesperado en validar entrega: %#v", validateResult["structuredContent"])
		}
		if validatePayload.Resultado == nil || !validatePayload.Resultado.Valida {
			t.Fatalf("resultado de validacion inesperado: %#v", validatePayload.Resultado)
		}
	})
}

func TestMCPResourceMicroprogramacionDevuelveJSON(t *testing.T) {
	withTempOrquestaDB(t, func() {
		id, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
			Titulo:            "EmitirMicrotarea",
			ArchivoObjetivo:   "microprogramacionapp/service.go",
			SimboloObjetivo:   "Emitir",
			Descripcion:       "Construye la microtarea cerrada para el agente",
			TestsObligatorios: []string{"go test ./microprogramacionapp -run TestServicioEmitir -count=1"},
			WriteSet:          []string{"microprogramacionapp/service.go", "microprogramacionapp/service_test.go"},
			CreadoPor:         "alberto",
		})
		if err != nil {
			t.Fatalf("crear especificacion base: %v", err)
		}

		resources, err := listMCPResources()
		if err != nil {
			t.Fatalf("listMCPResources: %v", err)
		}
		if !containsMCPResource(resources, microprogramacionMCPResourceEspecificaciones) {
			t.Fatalf("no aparece el resource de microprogramacion en listMCPResources")
		}

		templates := mcpResourceTemplates()
		if !containsMCPResourceTemplate(templates, "orquesta://microprogramacion/especificaciones/{id}") {
			t.Fatalf("no aparece el template de microprogramacion")
		}

		listContents, err := readMCPResource(microprogramacionMCPResourceEspecificaciones)
		if err != nil {
			t.Fatalf("readMCPResource list: %v", err)
		}
		if len(listContents) != 1 {
			t.Fatalf("contents list inesperado: %#v", listContents)
		}
		if text, _ := listContents[0]["text"].(string); !strings.Contains(text, "EmitirMicrotarea") {
			t.Fatalf("el listado no contiene la especificacion creada: %s", text)
		}

		detailContents, err := readMCPResource("orquesta://microprogramacion/especificaciones/" + itoa64(id))
		if err != nil {
			t.Fatalf("readMCPResource detail: %v", err)
		}
		if len(detailContents) != 1 {
			t.Fatalf("contents detail inesperado: %#v", detailContents)
		}
		text, _ := detailContents[0]["text"].(string)
		if !strings.Contains(text, "\"SimboloObjetivo\": \"Emitir\"") {
			t.Fatalf("el detalle no contiene el simbolo esperado: %s", text)
		}
	})
}

func containsMCPTool(items []mcpTool, name string) bool {
	for _, item := range items {
		if item.Name == name {
			return true
		}
	}
	return false
}

func containsMCPResource(items []mcpResource, uri string) bool {
	for _, item := range items {
		if item.URI == uri {
			return true
		}
	}
	return false
}

func containsMCPResourceTemplate(items []mcpResourceTemplate, uri string) bool {
	for _, item := range items {
		if item.URITemplate == uri {
			return true
		}
	}
	return false
}

func itoa64(value int64) string {
	return strconv.FormatInt(value, 10)
}
