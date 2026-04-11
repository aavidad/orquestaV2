package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"orquesta/microprogramacionapp"
)

const microprogramacionMCPResourceEspecificaciones = "orquesta://microprogramacion/especificaciones"

func microprogramacionMCPResources() []mcpResource {
	return []mcpResource{
		{
			URI:         microprogramacionMCPResourceEspecificaciones,
			Name:        "microprogramacion-especificaciones",
			Title:       "Especificaciones de microprogramacion",
			Description: "Especificaciones de funcion dirigidas por el orquestador",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
		},
	}
}

func microprogramacionMCPResourceTemplates() []mcpResourceTemplate {
	return []mcpResourceTemplate{
		{
			URITemplate: "orquesta://microprogramacion/especificaciones/{id}",
			Name:        "microprogramacion-especificacion-por-id",
			Title:       "Detalle de especificacion de funcion",
			Description: "Acceso directo a una especificacion de funcion dirigida por Orquesta",
			MIMEType:    "application/json",
		},
	}
}

func readMicroprogramacionMCPResource(uri string) ([]map[string]any, bool, error) {
	switch {
	case uri == microprogramacionMCPResourceEspecificaciones:
		items, err := microprogramacionService.Listar(microprogramacionapp.FiltroEspecificaciones{})
		if err != nil {
			return nil, true, err
		}
		return resourceText(uri, "application/json", prettyJSON(apiEspecificacionesFuncionResponse{
			Especificaciones: items,
		})), true, nil
	case strings.HasPrefix(uri, microprogramacionMCPResourceEspecificaciones+"/"):
		idRef := strings.Trim(strings.TrimPrefix(uri, microprogramacionMCPResourceEspecificaciones+"/"), "/")
		id, err := strconv.ParseInt(idRef, 10, 64)
		if err != nil || id <= 0 {
			return nil, true, errParametroInvalido("id")
		}
		item, err := microprogramacionService.Obtener(id)
		if err != nil {
			return nil, true, err
		}
		if item == nil {
			return nil, true, errNoEncontrado("especificacion_funcion")
		}
		return resourceText(uri, "application/json", prettyJSON(apiEspecificacionFuncionResponse{
			Especificacion: item,
		})), true, nil
	default:
		return nil, false, nil
	}
}

func microprogramacionMCPTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "orquesta.microprogramacion.especificaciones.listar",
			Title:       "Listar especificaciones de funcion",
			Description: "Lista especificaciones activas de microprogramacion dirigidas por Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tarea_id":    map[string]any{"type": "integer"},
					"proyecto":    map[string]any{"type": "string"},
					"proyecto_id": map[string]any{"type": "integer"},
					"estado":      map[string]any{"type": "string"},
					"limit":       map[string]any{"type": "integer"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.microprogramacion.especificaciones.ver",
			Title:       "Ver especificacion de funcion",
			Description: "Devuelve el detalle de una especificacion de funcion dirigida por Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "integer"},
				},
				"required":             []string{"id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.microprogramacion.especificaciones.crear",
			Title:       "Crear especificacion de funcion",
			Description: "Crea una especificacion de funcion por la via canónica de microprogramacion",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tarea_id":                map[string]any{"type": "integer"},
					"proyecto":                map[string]any{"type": "string"},
					"proyecto_id":             map[string]any{"type": "integer"},
					"titulo":                  map[string]any{"type": "string"},
					"archivo_objetivo":        map[string]any{"type": "string"},
					"simbolo_objetivo":        map[string]any{"type": "string"},
					"descripcion":             map[string]any{"type": "string"},
					"precondiciones":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"postcondiciones":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"dependencias_permitidas": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"dependencias_prohibidas": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"tests_obligatorios":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"write_set":               map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"formato_salida":          map[string]any{"type": "string"},
					"creado_por":              map[string]any{"type": "string"},
				},
				"required":             []string{"archivo_objetivo", "simbolo_objetivo", "descripcion", "tests_obligatorios", "write_set"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.microprogramacion.especificaciones.emitir",
			Title:       "Emitir microtarea",
			Description: "Construye la microtarea cerrada a partir de una especificacion de funcion",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":       map[string]any{"type": "integer"},
					"contexto": map[string]any{"type": "string"},
				},
				"required":             []string{"id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.microprogramacion.especificaciones.validar_entrega",
			Title:       "Validar entrega",
			Description: "Valida una entrega de agente contra firma, write-set, dependencias y tests obligatorios",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":                  map[string]any{"type": "integer"},
					"simbolo_entregado":   map[string]any{"type": "string"},
					"write_set_entregado": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"dependencias_usadas": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"tests_ejecutados":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"tests_fallidos":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"evidencia":           map[string]any{"type": "string"},
				},
				"required":             []string{"id"},
				"additionalProperties": false,
			},
		},
	}
}

func callMicroprogramacionMCPTool(name string, args map[string]any) (map[string]any, bool, error) {
	switch strings.TrimSpace(name) {
	case "orquesta.microprogramacion.especificaciones.listar":
		filtro, err := filtroEspecificacionesFuncionDesdeMCPArgs(args)
		if err != nil {
			return nil, true, err
		}
		items, err := microprogramacionService.Listar(filtro)
		if err != nil {
			return nil, true, err
		}
		result := apiEspecificacionesFuncionResponse{Especificaciones: items}
		return toolResult(prettyJSON(result), result, false), true, nil
	case "orquesta.microprogramacion.especificaciones.ver":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, true, err
		}
		item, err := microprogramacionService.Obtener(id)
		if err != nil {
			return nil, true, err
		}
		if item == nil {
			return toolResult("especificacion_funcion no encontrada", nil, true), true, nil
		}
		result := apiEspecificacionFuncionResponse{Especificacion: item}
		return toolResult(prettyJSON(result), result, false), true, nil
	case "orquesta.microprogramacion.especificaciones.crear":
		proyectoID, err := resolverProyectoEspecificacionFuncion(optionalInt64PtrArg(args, "proyecto_id"), optionalStringArg(args, "proyecto"))
		if err != nil {
			return nil, true, err
		}
		id, err := microprogramacionService.Crear(microprogramacionapp.EntradaCrearEspecificacion{
			TareaID:                optionalInt64PtrArg(args, "tarea_id"),
			ProyectoID:             proyectoID,
			Titulo:                 optionalStringArg(args, "titulo"),
			ArchivoObjetivo:        optionalStringArg(args, "archivo_objetivo"),
			SimboloObjetivo:        optionalStringArg(args, "simbolo_objetivo"),
			Descripcion:            optionalStringArg(args, "descripcion"),
			Precondiciones:         optionalStringSliceArg(args, "precondiciones"),
			Postcondiciones:        optionalStringSliceArg(args, "postcondiciones"),
			DependenciasPermitidas: optionalStringSliceArg(args, "dependencias_permitidas"),
			DependenciasProhibidas: optionalStringSliceArg(args, "dependencias_prohibidas"),
			TestsObligatorios:      optionalStringSliceArg(args, "tests_obligatorios"),
			WriteSet:               optionalStringSliceArg(args, "write_set"),
			FormatoSalida:          optionalStringArg(args, "formato_salida"),
			CreadoPor:              optionalStringArg(args, "creado_por"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), true, nil
		}
		item, err := microprogramacionService.Obtener(id)
		if err != nil {
			return nil, true, err
		}
		result := apiEspecificacionFuncionCreateResponse{
			OK:             true,
			ID:             id,
			Especificacion: item,
		}
		return toolResult(prettyJSON(result), result, false), true, nil
	case "orquesta.microprogramacion.especificaciones.emitir":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, true, err
		}
		item, err := microprogramacionService.Emitir(id, microprogramacionapp.EntradaEmitirMicrotarea{
			Contexto: optionalStringArg(args, "contexto"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), true, nil
		}
		result := apiMicrotareaEmitidaResponse{Microtarea: item}
		return toolResult(prettyJSON(result), result, false), true, nil
	case "orquesta.microprogramacion.especificaciones.validar_entrega":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, true, err
		}
		resultado, err := microprogramacionService.ValidarEntrega(id, microprogramacionapp.EntradaValidarEntrega{
			SimboloEntregado:   optionalStringArg(args, "simbolo_entregado"),
			WriteSetEntregado:  optionalStringSliceArg(args, "write_set_entregado"),
			DependenciasUsadas: optionalStringSliceArg(args, "dependencias_usadas"),
			TestsEjecutados:    optionalStringSliceArg(args, "tests_ejecutados"),
			TestsFallidos:      optionalStringSliceArg(args, "tests_fallidos"),
			Evidencia:          optionalStringArg(args, "evidencia"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), true, nil
		}
		result := apiValidacionEntregaResponse{Resultado: resultado}
		return toolResult(prettyJSON(result), result, false), true, nil
	default:
		return nil, false, nil
	}
}

func filtroEspecificacionesFuncionDesdeMCPArgs(args map[string]any) (microprogramacionapp.FiltroEspecificaciones, error) {
	var filtro microprogramacionapp.FiltroEspecificaciones
	if tareaID := optionalInt64PtrArg(args, "tarea_id"); tareaID != nil {
		filtro.TareaID = tareaID
	}
	proyectoID, err := resolverProyectoEspecificacionFuncion(optionalInt64PtrArg(args, "proyecto_id"), optionalStringArg(args, "proyecto"))
	if err != nil {
		return filtro, err
	}
	filtro.ProyectoID = proyectoID
	if estado := strings.TrimSpace(optionalStringArg(args, "estado")); estado != "" {
		value := microprogramacionapp.EstadoEspecificacion(estado)
		filtro.Estado = &value
	}
	if limit := optionalIntArg(args, "limit"); limit > 0 {
		filtro.Limit = limit
	}
	return filtro, nil
}

func optionalStringSliceArg(args map[string]any, key string) []string {
	if args == nil {
		return nil
	}
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return normalizarListaTextoMCP(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" {
				items = append(items, text)
			}
		}
		return normalizarListaTextoMCP(items)
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		return []string{text}
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func normalizarListaTextoMCP(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		repetido := false
		for _, actual := range out {
			if actual == item {
				repetido = true
				break
			}
		}
		if !repetido {
			out = append(out, item)
		}
	}
	return out
}
