/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
	"orquesta/gitgobernanza"
	"orquesta/panelapp"
	"orquesta/propuestasapp"
	"orquesta/runtimesapp"
	"orquesta/sesionesapp"
	"orquesta/tareasapp"
)

const (
	mcpProtocolLatest = "2025-11-25"
	mcpServerName     = "orquesta"
	mcpServerVersion  = "0.1.0"
)

var mcpSupportedProtocols = []string{
	"2025-11-25",
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

var panelService = panelapp.NewService(panelapp.Repository{})
var propuestasService = propuestasapp.NewService(propuestasapp.Repository{})
var runtimesService = runtimesapp.NewService(runtimesapp.Repository{})
var sesionesAPIService = sesionesapp.NewService(sesionesapp.Repository{})
var tareasService = tareasapp.NewService(tareasapp.Repository{})

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type mcpServer struct {
	reader      *bufio.Reader
	writer      *bufio.Writer
	initialized bool
}

type mcpResource struct {
	URI         string         `json:"uri"`
	Name        string         `json:"name"`
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	MIMEType    string         `json:"mimeType,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

type mcpResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MIMEType    string `json:"mimeType,omitempty"`
}

type mcpPrompt struct {
	Name        string              `json:"name"`
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Arguments   []mcpPromptArgument `json:"arguments,omitempty"`
}

type mcpPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type mcpPromptMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type mcpTool struct {
	Name         string         `json:"name"`
	Title        string         `json:"title,omitempty"`
	Description  string         `json:"description,omitempty"`
	InputSchema  map[string]any `json:"inputSchema"`
	OutputSchema map[string]any `json:"outputSchema,omitempty"`
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Servidor MCP y utilidades de integracion",
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Arranca un servidor MCP por stdio",
	Long: `Expone Orquesta como servidor MCP por stdio usando JSON-RPC 2.0.
Incluye resources, prompts y tools alineados con la arquitectura objetivo:
- resources: estado vivo del orquestador y elementos clave de trabajo
- prompts: briefing operativo y revision de propuestas
- tools: consultas y acciones controladas sobre tareas/propuestas`,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := &mcpServer{
			reader: bufio.NewReader(os.Stdin),
			writer: bufio.NewWriter(os.Stdout),
		}
		return srv.serve()
	},
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
	rootCmd.AddCommand(mcpCmd)
}

func (s *mcpServer) serve() error {
	for {
		payload, err := readMCPFrame(s.reader)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		var req mcpRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			if err := s.writeError(nil, -32700, "JSON inválido", err.Error()); err != nil {
				return err
			}
			continue
		}

		if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
			if err := s.writeError(req.ID, -32600, "Request inválida", nil); err != nil {
				return err
			}
			continue
		}

		result, rpcErr := s.handleRequest(req)
		if !req.hasID() {
			continue
		}
		if rpcErr != nil {
			if err := s.writeError(req.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data); err != nil {
				return err
			}
			continue
		}
		if err := s.writeResult(req.ID, result); err != nil {
			return err
		}
	}
}

func (s *mcpServer) handleRequest(req mcpRequest) (any, *mcpError) {
	if req.Method != "initialize" && req.Method != "notifications/initialized" && req.Method != "ping" && !s.initialized {
		return nil, &mcpError{Code: -32002, Message: "Servidor MCP no inicializado"}
	}

	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string         `json:"protocolVersion"`
			Capabilities    map[string]any `json:"capabilities"`
			ClientInfo      map[string]any `json:"clientInfo"`
		}
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		s.initialized = true
		return map[string]any{
			"protocolVersion": negotiateProtocolVersion(params.ProtocolVersion),
			"capabilities": map[string]any{
				"prompts":   map[string]any{},
				"resources": map[string]any{},
				"tools":     map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    mcpServerName,
				"title":   "Orquesta MCP",
				"version": mcpServerVersion,
			},
			"instructions": "Usa resources para contexto vivo, prompts para briefing/gobierno y tools para acciones controladas. Las mutaciones deben mantenerse con confirmacion humana.",
		}, nil

	case "notifications/initialized":
		return nil, nil

	case "ping":
		return map[string]any{}, nil

	case "resources/list":
		resources, err := listMCPResources()
		if err != nil {
			return nil, internalRPCError(err)
		}
		return map[string]any{"resources": resources}, nil

	case "resources/templates/list":
		return map[string]any{"resourceTemplates": mcpResourceTemplates()}, nil

	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		contents, err := readMCPResource(strings.TrimSpace(params.URI))
		if err != nil {
			return nil, resourceNotFound(params.URI, err)
		}
		return map[string]any{"contents": contents}, nil

	case "prompts/list":
		return map[string]any{"prompts": listMCPPrompts()}, nil

	case "prompts/get":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		result, err := getMCPPrompt(params.Name, params.Arguments)
		if err != nil {
			return nil, invalidParams(err)
		}
		return result, nil

	case "tools/list":
		return map[string]any{"tools": listMCPTools()}, nil

	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := decodeParams(req.Params, &params); err != nil {
			return nil, invalidParams(err)
		}
		result, err := callMCPTool(params.Name, params.Arguments)
		if err != nil {
			return nil, toolCallError(err)
		}
		return result, nil
	}

	return nil, &mcpError{Code: -32601, Message: "Metodo no soportado", Data: map[string]any{"method": req.Method}}
}

func (s *mcpServer) writeResult(id json.RawMessage, result any) error {
	return writeMCPFrame(s.writer, mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *mcpServer) writeError(id json.RawMessage, code int, message string, data any) error {
	return writeMCPFrame(s.writer, mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcpError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

func (r mcpRequest) hasID() bool {
	if len(r.ID) == 0 {
		return false
	}
	return strings.TrimSpace(string(r.ID)) != "null"
}

func readMCPFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && contentLength == -1 && len(line) == 0 {
				return nil, io.EOF
			}
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("cabecera MCP inválida: %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 {
				return nil, fmt.Errorf("Content-Length inválido: %q", value)
			}
			contentLength = n
		}
	}

	if contentLength < 0 {
		return nil, fmt.Errorf("mensaje MCP sin Content-Length")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeMCPFrame(writer *bufio.Writer, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		return err
	}
	return writer.Flush()
}

func decodeParams(raw json.RawMessage, dst any) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dst)
}

func negotiateProtocolVersion(requested string) string {
	requested = strings.TrimSpace(requested)
	for _, supported := range mcpSupportedProtocols {
		if requested == supported {
			return requested
		}
	}
	return mcpProtocolLatest
}

func listMCPResources() ([]mcpResource, error) {
	status, err := buildEstadoResumen()
	if err != nil {
		return nil, err
	}

	resources := []mcpResource{
		{
			URI:         "orquesta://arquitectura/mcp",
			Name:        "arquitectura-mcp",
			Title:       "Arquitectura MCP de Orquesta",
			Description: "Criterios arquitectonicos para resources, prompts y tools en Orquesta",
			MIMEType:    "text/markdown",
			Annotations: audienceAssistant(1),
		},
		{
			URI:         "orquesta://estado/resumen",
			Name:        "estado-resumen",
			Title:       "Resumen de estado",
			Description: "Resumen operativo de tareas, propuestas, agentes, proyectos y sesiones",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(1),
		},
		{
			URI:         "orquesta://proyectos",
			Name:        "proyectos",
			Title:       "Proyectos registrados",
			Description: "Listado de proyectos conocidos por Orquesta",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
		},
		{
			URI:         "orquesta://asignaciones/activas",
			Name:        "asignaciones-activas",
			Title:       "Asignaciones activas",
			Description: "Asignaciones de agentes a proyectos activas",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		},
		{
			URI:         "orquesta://agentes/activos",
			Name:        "agentes-activos",
			Title:       "Agentes activos",
			Description: "Listado de agentes con sesion activa",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		},
		{
			URI:         "orquesta://propuestas/abiertas",
			Name:        "propuestas-abiertas",
			Title:       "Propuestas abiertas",
			Description: "Propuestas pendientes de consenso o arbitraje",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
		},
		{
			URI:         "orquesta://tareas/en_progreso",
			Name:        "tareas-en-progreso",
			Title:       "Tareas en progreso",
			Description: "Trabajo activo actualmente en curso",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		},
		{
			URI:         "orquesta://conectores",
			Name:        "conectores",
			Title:       "Conectores registrados",
			Description: "Conectores y transportes conocidos por Orquesta",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.6),
		},
		{
			URI:         "orquesta://pools",
			Name:        "pools",
			Title:       "Pools de capacidad",
			Description: "Pools de capacidad configurados en Orquesta",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
		},
		{
			URI:         "orquesta://politicas-modelo",
			Name:        "politicas-modelo",
			Title:       "Politicas de modelo",
			Description: "Politicas activas para resolver pool, modelo y reasoning",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		},
		{
			URI:         "orquesta://worktrees/activas",
			Name:        "worktrees-activas",
			Title:       "Worktrees activas",
			Description: "Worktrees activas registradas para trabajo paralelo",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		},
		{
			URI:         "orquesta://locks/activos",
			Name:        "locks-activos",
			Title:       "Locks activos",
			Description: "Locks activos para coordinacion segura",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		},
		{
			URI:         "orquesta://sesiones/activas",
			Name:        "sesiones-activas",
			Title:       "Sesiones activas",
			Description: "Sesiones activas y reanudables conocidas por Orquesta",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		},
	}

	for _, proyecto := range status.Proyectos {
		resources = append(resources, mcpResource{
			URI:         "orquesta://proyectos/" + proyecto["slug"].(string),
			Name:        "proyecto-" + proyecto["slug"].(string),
			Title:       fmt.Sprintf("%v", proyecto["nombre"]),
			Description: "Detalle del proyecto " + proyecto["slug"].(string),
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		})
	}

	for _, pool := range status.Pools {
		resources = append(resources, mcpResource{
			URI:         "orquesta://pools/" + pool["slug"].(string),
			Name:        "pool-" + pool["slug"].(string),
			Title:       fmt.Sprintf("%v", pool["slug"]),
			Description: "Detalle del pool " + pool["slug"].(string),
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		})
	}

	for _, p := range status.PropuestasAbiertas {
		resources = append(resources, mcpResource{
			URI:         "orquesta://propuestas/" + p.Codigo,
			Name:        p.Codigo,
			Title:       p.Titulo,
			Description: "Detalle de propuesta " + p.Codigo,
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
		})
	}

	for _, t := range status.TareasActivas {
		resources = append(resources, mcpResource{
			URI:         fmt.Sprintf("orquesta://tareas/%d", t.ID),
			Name:        fmt.Sprintf("tarea-%d", t.ID),
			Title:       t.Titulo,
			Description: fmt.Sprintf("Detalle de la tarea #%d", t.ID),
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		})
	}

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].URI < resources[j].URI
	})
	return resources, nil
}

func mcpResourceTemplates() []mcpResourceTemplate {
	return []mcpResourceTemplate{
		{
			URITemplate: "orquesta://propuestas/{codigo}",
			Name:        "propuesta-por-codigo",
			Title:       "Detalle de propuesta",
			Description: "Acceso directo al detalle de una propuesta por su codigo OP-XXX",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://tareas/{id}",
			Name:        "tarea-por-id",
			Title:       "Detalle de tarea",
			Description: "Acceso directo al detalle de una tarea por su identificador",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://agentes/{agente}/briefing",
			Name:        "briefing-agente",
			Title:       "Briefing de agente",
			Description: "Resumen operativo del agente: reglas, skills, workflow y propuestas pendientes",
			MIMEType:    "text/markdown",
		},
		{
			URITemplate: "orquesta://proyectos/{slug}",
			Name:        "proyecto-por-slug",
			Title:       "Detalle de proyecto",
			Description: "Acceso directo al detalle de un proyecto por su slug",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://pools/{slug}",
			Name:        "pool-por-slug",
			Title:       "Detalle de pool",
			Description: "Acceso directo al detalle de un pool y sus modelos por slug",
			MIMEType:    "application/json",
		},
	}
}

func readMCPResource(uri string) ([]map[string]any, error) {
	switch {
	case uri == "orquesta://arquitectura/mcp":
		return resourceText(uri, "text/markdown", arquitecturaMCPText()), nil
	case uri == "orquesta://estado/resumen":
		s, err := buildEstadoResumen()
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(s)), nil
	case uri == "orquesta://proyectos":
		proyectos, err := listarProyectos()
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(proyectos)), nil
	case uri == "orquesta://asignaciones/activas":
		asignaciones, err := listarAsignaciones("activa")
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(asignaciones)), nil
	case uri == "orquesta://agentes/activos":
		agentes, err := propuestasService.ListAgents()
		if err != nil {
			return nil, err
		}
		var activos []*db.Agente
		for _, a := range agentes {
			if a.Activo {
				activos = append(activos, a)
			}
		}
		return resourceText(uri, "application/json", prettyJSON(activos)), nil
	case uri == "orquesta://propuestas/abiertas":
		propuestas, err := propuestasService.List(ptrPropuestaEstado(db.PropuestaAbierta))
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(propuestas)), nil
	case uri == "orquesta://tareas/en_progreso":
		tareas, err := tareasService.List(db.FiltroTareas{Estado: ptrEstadoTarea(db.TareaEnProgreso)})
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(tareas)), nil
	case uri == "orquesta://conectores":
		conectores, err := listarConectores()
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(conectores)), nil
	case uri == "orquesta://pools":
		pools, err := listarPoolsResumenMCP(nil)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(pools)), nil
	case uri == "orquesta://politicas-modelo":
		activa := true
		politicas, err := listarPoliticasModeloMCP("", "", &activa)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(politicas)), nil
	case uri == "orquesta://worktrees/activas":
		worktrees, err := listarWorktrees("activa")
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(worktrees)), nil
	case uri == "orquesta://locks/activos":
		locks, err := listarLocks("activa")
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(locks)), nil
	case uri == "orquesta://sesiones/activas":
		sesiones, err := listarSesionesActivas()
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(sesiones)), nil
	case strings.HasPrefix(uri, "orquesta://proyectos/"):
		slug := strings.TrimPrefix(uri, "orquesta://proyectos/")
		proyecto, err := detalleProyecto(slug)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(proyecto)), nil
	case strings.HasPrefix(uri, "orquesta://pools/"):
		slug := strings.TrimPrefix(uri, "orquesta://pools/")
		pool, err := detallePoolMCP(slug)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(pool)), nil
	case strings.HasPrefix(uri, "orquesta://propuestas/"):
		codigo := strings.TrimPrefix(uri, "orquesta://propuestas/")
		detail, err := propuestasService.GetDetail(codigo)
		if err != nil {
			return nil, err
		}
		payload := map[string]any{
			"propuesta": detail.Proposal,
			"votos":     detail.Votes,
		}
		return resourceText(uri, "application/json", prettyJSON(payload)), nil
	case strings.HasPrefix(uri, "orquesta://tareas/"):
		idStr := strings.TrimPrefix(uri, "orquesta://tareas/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("id de tarea inválido")
		}
		t, err := tareasService.Get(id)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(t)), nil
	case strings.HasPrefix(uri, "orquesta://agentes/") && strings.HasSuffix(uri, "/briefing"):
		agente := strings.TrimSuffix(strings.TrimPrefix(uri, "orquesta://agentes/"), "/briefing")
		texto, err := buildAgentBriefing(agente)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "text/markdown", texto), nil
	default:
		return nil, fmt.Errorf("resource not found")
	}
}

func listMCPPrompts() []mcpPrompt {
	return []mcpPrompt{
		{
			Name:        "orquesta.briefing.agente",
			Title:       "Briefing de agente",
			Description: "Devuelve el briefing operativo del agente usando reglas, skills, workflows y propuestas pendientes",
			Arguments: []mcpPromptArgument{
				{Name: "agente", Description: "Nombre del agente registrado en Orquesta", Required: true},
			},
		},
		{
			Name:        "orquesta.revision.propuesta",
			Title:       "Revision tecnica de propuesta",
			Description: "Genera una guia de revision tecnica y de voto para una propuesta de Orquesta",
			Arguments: []mcpPromptArgument{
				{Name: "codigo", Description: "Codigo de la propuesta, por ejemplo OP-049", Required: true},
				{Name: "agente", Description: "Agente que revisa la propuesta", Required: false},
			},
		},
		{
			Name:        "orquesta.plan.tarea",
			Title:       "Plan de tarea",
			Description: "Resume una tarea y pide un plan corto de ejecucion alineado con Orquesta",
			Arguments: []mcpPromptArgument{
				{Name: "id", Description: "Identificador numerico de la tarea", Required: true},
				{Name: "agente", Description: "Agente que asume la tarea", Required: false},
			},
		},
		{
			Name:        "orquesta.contexto.proyecto",
			Title:       "Contexto de proyecto",
			Description: "Resume el contexto operativo de un proyecto: asignaciones, worktrees, tareas y propuestas abiertas",
			Arguments: []mcpPromptArgument{
				{Name: "slug", Description: "Slug del proyecto registrado en Orquesta", Required: true},
			},
		},
	}
}

func getMCPPrompt(name string, args map[string]any) (map[string]any, error) {
	switch strings.TrimSpace(name) {
	case "orquesta.briefing.agente":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		briefing, err := buildAgentBriefing(agente)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Briefing operativo generado desde la BD de Orquesta",
			"messages": []mcpPromptMessage{
				{
					Role: "user",
					Content: map[string]any{
						"type": "text",
						"text": briefing,
					},
				},
			},
		}, nil

	case "orquesta.revision.propuesta":
		codigo, err := requiredStringArg(args, "codigo")
		if err != nil {
			return nil, err
		}
		agente := optionalStringArg(args, "agente")
		texto, err := buildProposalReviewPrompt(codigo, agente)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Guia de revision tecnica y voto",
			"messages": []mcpPromptMessage{
				{
					Role: "user",
					Content: map[string]any{
						"type": "text",
						"text": texto,
					},
				},
			},
		}, nil

	case "orquesta.plan.tarea":
		idStr, err := requiredStringArg(args, "id")
		if err != nil {
			return nil, err
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("id inválido")
		}
		agente := optionalStringArg(args, "agente")
		texto, err := buildTaskPlanPrompt(id, agente)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Plan corto de ejecucion para una tarea registrada en Orquesta",
			"messages": []mcpPromptMessage{
				{
					Role: "user",
					Content: map[string]any{
						"type": "text",
						"text": texto,
					},
				},
			},
		}, nil

	case "orquesta.contexto.proyecto":
		slug, err := requiredStringArg(args, "slug")
		if err != nil {
			return nil, err
		}
		texto, err := buildProjectContextPrompt(slug)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Contexto operativo de un proyecto registrado en Orquesta",
			"messages": []mcpPromptMessage{
				{
					Role: "user",
					Content: map[string]any{
						"type": "text",
						"text": texto,
					},
				},
			},
		}, nil
	}

	return nil, fmt.Errorf("prompt no soportado: %s", name)
}

func listMCPTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "orquesta.estado.resumen",
			Title:       "Resumen del estado",
			Description: "Devuelve un resumen operativo del estado actual de Orquesta",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.tareas.listar",
			Title:       "Listar tareas",
			Description: "Lista tareas filtradas por estado, agente, modulo o propuesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"estado":    map[string]any{"type": "string"},
					"agente":    map[string]any{"type": "string"},
					"modulo":    map[string]any{"type": "string"},
					"propuesta": map[string]any{"type": "string"},
					"libre":     map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.tareas.iniciar",
			Title:       "Iniciar tarea",
			Description: "Marca una tarea asignada como en progreso",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "integer"},
					"agente": map[string]any{"type": "string"},
				},
				"required":             []string{"id", "agente"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.propuestas.listar",
			Title:       "Listar propuestas",
			Description: "Lista propuestas, opcionalmente filtradas por estado",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"estado": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.proyectos.listar",
			Title:       "Listar proyectos",
			Description: "Lista proyectos registrados en Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tipo":   map[string]any{"type": "string"},
					"activo": map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.pools.listar",
			Title:       "Listar pools",
			Description: "Lista pools de capacidad con capacidad disponible",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"activo": map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.pools.modelos",
			Title:       "Listar modelos de pool",
			Description: "Lista modelos habilitados dentro de un pool",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pool_slug": map[string]any{"type": "string"},
				},
				"required":             []string{"pool_slug"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.politicas-modelo.listar",
			Title:       "Listar politicas de modelo",
			Description: "Lista politicas activas o filtradas por scope",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope_tipo": map[string]any{"type": "string"},
					"scope_ref":  map[string]any{"type": "string"},
					"activa":     map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.skills.detectar-carencia",
			Title:       "Detectar skill faltante",
			Description: "Consulta el catalogo de skills, detecta carencias y prepara un borrador revisable con invocacion controlada a $skill-creator",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":            map[string]any{"type": "string"},
					"tipo_agente":       map[string]any{"type": "string"},
					"nombre":            map[string]any{"type": "string"},
					"descripcion":       map[string]any{"type": "string"},
					"cuando_usar":       map[string]any{"type": "string"},
					"escenario":         map[string]any{"type": "string"},
					"aliases_json":      map[string]any{"type": "string"},
					"herramientas_json": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.modelos.resolver",
			Title:       "Resolver modelo",
			Description: "Resuelve pool, modelo y reasoning segun politicas y fallback",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tarea_id": map[string]any{"type": "integer"},
					"proyecto": map[string]any{"type": "string"},
					"fase":     map[string]any{"type": "string"},
					"perfil":   map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.asignaciones.listar",
			Title:       "Listar asignaciones",
			Description: "Lista asignaciones de agentes a proyectos",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"estado": map[string]any{"type": "string"},
					"agente": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.worktrees.listar",
			Title:       "Listar worktrees",
			Description: "Lista worktrees registradas en Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"estado": map[string]any{"type": "string"},
					"agente": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.locks.listar",
			Title:       "Listar locks",
			Description: "Lista locks registrados en Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"estado": map[string]any{"type": "string"},
					"agente": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.sesiones.activas",
			Title:       "Sesiones activas",
			Description: "Lista sesiones activas con su contexto de continuidad",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.propuestas.votar",
			Title:       "Votar propuesta",
			Description: "Registra el voto de un agente sobre una propuesta abierta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"codigo":     map[string]any{"type": "string"},
					"agente":     map[string]any{"type": "string"},
					"posicion":   map[string]any{"type": "string", "enum": []string{"acuerdo", "desacuerdo", "abstencion"}},
					"comentario": map[string]any{"type": "string"},
				},
				"required":             []string{"codigo", "agente", "posicion"},
				"additionalProperties": false,
			},
		},
	}
}

func callMCPTool(name string, args map[string]any) (map[string]any, error) {
	switch strings.TrimSpace(name) {
	case "orquesta.estado.resumen":
		resumen, err := buildEstadoResumen()
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(resumen), resumen, false), nil

	case "orquesta.tareas.listar":
		filtro, err := buildFiltroTareas(args)
		if err != nil {
			return nil, err
		}
		tareas, err := tareasService.List(filtro)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(tareas), tareas, false), nil

	case "orquesta.tareas.iniciar":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, err
		}
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		if err := tareasService.Start(id, agente); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		t, err := tareasService.Get(id)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(t), t, false), nil

	case "orquesta.propuestas.listar":
		var estadoPtr *db.EstadoPropuesta
		if estado := optionalStringArg(args, "estado"); estado != "" {
			e := db.EstadoPropuesta(estado)
			estadoPtr = &e
		}
		propuestas, err := propuestasService.List(estadoPtr)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(propuestas), propuestas, false), nil

	case "orquesta.proyectos.listar":
		proyectos, err := listarProyectosFiltrados(optionalStringArg(args, "tipo"), boolPtrArg(args, "activo"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(proyectos), proyectos, false), nil

	case "orquesta.pools.listar":
		pools, err := listarPoolsResumenMCP(boolPtrArg(args, "activo"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(pools), pools, false), nil

	case "orquesta.pools.modelos":
		poolSlug, err := requiredStringArg(args, "pool_slug")
		if err != nil {
			return nil, err
		}
		modelos, err := capacidadService.ListPoolModels(poolSlug)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(modelos), modelos, false), nil

	case "orquesta.politicas-modelo.listar":
		politicas, err := listarPoliticasModeloMCP(optionalStringArg(args, "scope_tipo"), optionalStringArg(args, "scope_ref"), boolPtrArg(args, "activa"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(politicas), politicas, false), nil

	case "orquesta.skills.detectar-carencia":
		req, err := buildSolicitudDeteccionSkillMCP(args)
		if err != nil {
			return nil, err
		}
		resultado, err := db.DetectarCarenciaSkill(req)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(resultado), resultado, false), nil

	case "orquesta.modelos.resolver":
		var tareaIDPtr *int64
		if raw, ok := args["tarea_id"]; ok && raw != nil {
			value, err := requiredInt64Arg(args, "tarea_id")
			if err != nil {
				return nil, err
			}
			tareaIDPtr = &value
		}
		res, err := capacidadService.ResolveModelPolicy(db.ResolverPoliticaInput{
			TareaID:      tareaIDPtr,
			ProyectoSlug: optionalStringArg(args, "proyecto"),
			Fase:         optionalStringArg(args, "fase"),
			PerfilTarea:  optionalStringArg(args, "perfil"),
		})
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(res), res, false), nil

	case "orquesta.asignaciones.listar":
		asignaciones, err := listarAsignacionesFiltradas(optionalStringArg(args, "estado"), optionalStringArg(args, "agente"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(asignaciones), asignaciones, false), nil

	case "orquesta.worktrees.listar":
		worktrees, err := listarWorktreesFiltradas(optionalStringArg(args, "estado"), optionalStringArg(args, "agente"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(worktrees), worktrees, false), nil

	case "orquesta.locks.listar":
		locks, err := listarLocksFiltrados(optionalStringArg(args, "estado"), optionalStringArg(args, "agente"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(locks), locks, false), nil

	case "orquesta.sesiones.activas":
		sesiones, err := listarSesionesActivas()
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(sesiones), sesiones, false), nil

	case "orquesta.propuestas.votar":
		codigo, err := requiredStringArg(args, "codigo")
		if err != nil {
			return nil, err
		}
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		posicionStr, err := requiredStringArg(args, "posicion")
		if err != nil {
			return nil, err
		}
		posicion := db.PosicionVoto(posicionStr)
		switch posicion {
		case db.VotoAcuerdo, db.VotoDesacuerdo, db.VotoAbstencion:
		default:
			return nil, fmt.Errorf("posicion inválida: %s", posicionStr)
		}
		comentario := optionalStringArg(args, "comentario")
		result, err := propuestasService.VoteDetail(codigo, agente, posicion, comentario)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		resumen := map[string]any{
			"codigo":            codigo,
			"agente":            agente,
			"posicion":          posicion,
			"comentario":        comentario,
			"consensoAlcanzado": result.Consenso,
			"conteo": map[string]int{
				"acuerdo":    result.Acuerdo,
				"desacuerdo": result.Desacuerdo,
				"abstencion": result.Abstencion,
				"pendiente":  result.Pendiente,
			},
		}
		return toolResult(prettyJSON(resumen), resumen, false), nil
	}

	return nil, fmt.Errorf("tool no soportada: %s", name)
}

type estadoResumen struct {
	Generado           string           `json:"generado"`
	TareasPorEstado    map[string]int   `json:"tareasPorEstado"`
	AgentesActivos     []*db.Agente     `json:"agentesActivos"`
	Proyectos          []map[string]any `json:"proyectos"`
	Pools              []map[string]any `json:"pools"`
	Asignaciones       []map[string]any `json:"asignaciones"`
	PropuestasAbiertas []propuestaLite  `json:"propuestasAbiertas"`
	TareasActivas      []tareaLite      `json:"tareasActivas"`
	WorktreesActivas   []map[string]any `json:"worktreesActivas"`
	LocksActivos       []map[string]any `json:"locksActivos"`
	SesionesActivas    []map[string]any `json:"sesionesActivas"`
	Conectores         []map[string]any `json:"conectores"`
}

type propuestaLite struct {
	ID           int64              `json:"id"`
	Codigo       string             `json:"codigo"`
	Titulo       string             `json:"titulo"`
	Estado       db.EstadoPropuesta `json:"estado"`
	PropuestoPor string             `json:"propuestoPor"`
	Acuerdo      int                `json:"acuerdo"`
	Desacuerdo   int                `json:"desacuerdo"`
	Abstencion   int                `json:"abstencion"`
	Pendiente    int                `json:"pendiente"`
}

type tareaLite struct {
	ID        int64             `json:"id"`
	Titulo    string            `json:"titulo"`
	Estado    db.EstadoTarea    `json:"estado"`
	Modulo    string            `json:"modulo"`
	Agente    string            `json:"agente,omitempty"`
	Prioridad db.PrioridadTarea `json:"prioridad"`
}

func buildEstadoResumen() (*estadoResumen, error) {
	summary, err := panelService.BuildSummary()
	if err != nil {
		return nil, err
	}
	var activos []*db.Agente
	for _, a := range summary.Agents {
		if a.Activo {
			activos = append(activos, a)
		}
	}
	var abiertas []propuestaLite
	for _, p := range summary.OpenProps {
		abiertas = append(abiertas, propuestaLite{
			Codigo:     p.Codigo,
			Titulo:     p.Titulo,
			Estado:     db.PropuestaAbierta,
			Acuerdo:    p.Acuerdo,
			Desacuerdo: p.Desacuerdo,
			Pendiente:  p.Pendiente,
		})
	}

	tareas, err := tareasService.List(db.FiltroTareas{})
	if err != nil {
		return nil, err
	}
	var activas []tareaLite
	for _, t := range tareas {
		if t.Estado != db.TareaAsignada && t.Estado != db.TareaEnProgreso && t.Estado != db.TareaBloqueada {
			continue
		}
		row := tareaLite{
			ID:        t.ID,
			Titulo:    t.Titulo,
			Estado:    t.Estado,
			Modulo:    t.Modulo,
			Prioridad: t.Prioridad,
		}
		if t.Agente != nil {
			row.Agente = *t.Agente
		}
		activas = append(activas, row)
	}

	conectores, err := listarConectores()
	if err != nil {
		return nil, err
	}
	proyectos, err := listarProyectos()
	if err != nil {
		return nil, err
	}
	pools, err := listarPoolsResumenDesdeServicio(nil)
	if err != nil {
		return nil, err
	}
	asignaciones, err := listarAsignaciones("activa")
	if err != nil {
		return nil, err
	}
	worktrees, err := listarWorktrees("activa")
	if err != nil {
		return nil, err
	}
	locks, err := listarLocks("activa")
	if err != nil {
		return nil, err
	}
	sesiones, err := listarSesionesActivas()
	if err != nil {
		return nil, err
	}

	return &estadoResumen{
		Generado:           time.Now().UTC().Format(time.RFC3339),
		TareasPorEstado:    summary.TaskCounts,
		AgentesActivos:     activos,
		Proyectos:          proyectos,
		Pools:              pools,
		Asignaciones:       asignaciones,
		PropuestasAbiertas: abiertas,
		TareasActivas:      activas,
		WorktreesActivas:   worktrees,
		LocksActivos:       locks,
		SesionesActivas:    sesiones,
		Conectores:         conectores,
	}, nil
}

func listarPoolsResumenDesdeServicio(activo *bool) ([]map[string]any, error) {
	rows, err := capacidadService.ListPoolsSummary(activo)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, item := range rows {
		out = append(out, map[string]any{
			"slug":                 item.Pool.Slug,
			"proveedor":            item.Pool.Proveedor,
			"runtime":              item.Pool.Runtime,
			"plan":                 item.Pool.Plan,
			"capacidad_total":      item.Pool.CapacidadTotal,
			"capacidad_reservada":  item.Pool.CapacidadReservada,
			"sesiones_activas":     item.SesionesActivas,
			"capacidad_disponible": item.CapacidadDisponible,
			"politica_handoff":     item.Pool.PoliticaHandoff,
			"fuente_telemetria":    item.Pool.FuenteTelemetria,
			"activo":               item.Pool.Activo,
		})
	}
	return out, nil
}

func buildAgentBriefing(agente string) (string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return "", fmt.Errorf("agente obligatorio")
	}
	briefing, err := sesionesAPIService.BuildBriefing(agente)
	if err != nil {
		return "", fmt.Errorf("agente no encontrado: %s", agente)
	}
	actual := briefing.Agent

	var b strings.Builder
	fmt.Fprintf(&b, "# Briefing de %s\n\n", actual.Nombre)
	fmt.Fprintf(&b, "- Rol: %s\n", actual.Rol)
	fmt.Fprintf(&b, "- Activo: %t\n", actual.Activo)
	if strings.TrimSpace(actual.EstadoSesion) != "" {
		fmt.Fprintf(&b, "- Estado de sesion: %s\n", actual.EstadoSesion)
	}
	b.WriteString("\n")

	if actual.Rol == "admin" {
		b.WriteString("Agente admin: no se genera briefing operativo adicional.\n")
		return b.String(), nil
	}

	if len(briefing.PropuestasPendientes) > 0 {
		b.WriteString("## Propuestas pendientes de voto\n")
		for _, p := range briefing.PropuestasPendientes {
			fmt.Fprintf(&b, "- %s: %s\n", p.Codigo, p.Titulo)
		}
		b.WriteString("\n")
	}

	if len(briefing.Reglas) > 0 {
		b.WriteString("## Reglas activas\n")
		for _, r := range briefing.Reglas {
			fmt.Fprintf(&b, "- [%s] %s: %s\n", r.Categoria, r.Titulo, r.Descripcion)
		}
		b.WriteString("\n")
	}

	if len(briefing.Skills) > 0 {
		b.WriteString("## Skills\n")
		for _, s := range briefing.Skills {
			fmt.Fprintf(&b, "- %s: %s\n", s.Nombre, s.CuandoUsar)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Preflight de skills\n")
	b.WriteString("- Antes de crear una skill nueva, consulta el catalogo existente.\n")
	b.WriteString("- Si ninguna skill cubre la necesidad actual, usa la tool MCP `orquesta.skills.detectar-carencia` para preparar un borrador revisable.\n\n")

	if len(briefing.Workflows) > 0 {
		b.WriteString("## Workflows\n")
		for _, wf := range briefing.Workflows {
			fmt.Fprintf(&b, "- %s: %s\n", wf.Nombre, wf.Descripcion)
		}
		b.WriteString("\n")
	}

	tareas, err := tareasService.List(db.FiltroTareas{Agente: &actual.Nombre})
	if err != nil {
		return "", err
	}
	if len(tareas) > 0 {
		b.WriteString("## Tareas asignadas\n")
		for _, t := range tareas {
			fmt.Fprintf(&b, "- #%d [%s] %s\n", t.ID, t.Estado, t.Titulo)
		}
	}

	return strings.TrimSpace(b.String()) + "\n", nil
}

func buildProposalReviewPrompt(codigo, agente string) (string, error) {
	detail, err := propuestasService.GetDetail(codigo)
	if err != nil {
		return "", err
	}
	p := detail.Proposal

	var b strings.Builder
	fmt.Fprintf(&b, "Revisa la propuesta %s de Orquesta.\n\n", p.Codigo)
	if strings.TrimSpace(agente) != "" {
		fmt.Fprintf(&b, "Agente revisor: %s.\n\n", strings.TrimSpace(agente))
	}
	fmt.Fprintf(&b, "Titulo: %s\n", p.Titulo)
	fmt.Fprintf(&b, "Estado: %s\n", p.Estado)
	fmt.Fprintf(&b, "Tipo: %s\n", p.Tipo)
	fmt.Fprintf(&b, "Propuesto por: %s\n\n", p.PropuestoPor)
	fmt.Fprintf(&b, "Descripcion:\n%s\n\n", p.Descripcion)
	b.WriteString("Evalua con estos criterios:\n")
	b.WriteString("- coherencia tecnica\n")
	b.WriteString("- compatibilidad con lo ya implementado\n")
	b.WriteString("- coste de evolucion razonable\n")
	b.WriteString("- seguridad\n")
	b.WriteString("- mantenibilidad\n")
	b.WriteString("- utilidad real para el flujo diario\n\n")
	b.WriteString("Responde con este formato:\n")
	b.WriteString("- posicion global: acuerdo, desacuerdo o abstencion\n")
	b.WriteString("- puntos correctos\n")
	b.WriteString("- riesgos o ajustes\n")
	b.WriteString("- recomendacion de orden de implementacion\n")
	return b.String(), nil
}

func buildTaskPlanPrompt(id int64, agente string) (string, error) {
	t, err := tareasService.Get(id)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Planifica la tarea #%d de Orquesta.\n\n", t.ID)
	fmt.Fprintf(&b, "Titulo: %s\n", t.Titulo)
	fmt.Fprintf(&b, "Estado: %s\n", t.Estado)
	fmt.Fprintf(&b, "Prioridad: %s\n", t.Prioridad)
	fmt.Fprintf(&b, "Modulo: %s\n", t.Modulo)
	if agente != "" {
		fmt.Fprintf(&b, "Agente responsable: %s\n", agente)
	} else if t.Agente != nil {
		fmt.Fprintf(&b, "Agente responsable: %s\n", *t.Agente)
	}
	if strings.TrimSpace(t.Descripcion) != "" {
		fmt.Fprintf(&b, "\nDescripcion:\n%s\n", t.Descripcion)
	}
	b.WriteString("\nGenera:\n")
	b.WriteString("1. un plan corto de 3 a 6 pasos\n")
	b.WriteString("2. riesgos o dependencias inmediatas\n")
	b.WriteString("3. validaciones minimas antes de cerrar la tarea\n")
	b.WriteString("4. recordatorio: no modificar datos existentes de forma destructiva\n")
	return b.String(), nil
}

func buildProjectContextPrompt(slug string) (string, error) {
	proyecto, err := detalleProyecto(slug)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Contexto del proyecto %s.\n\n", slug)
	fmt.Fprintf(&b, "Proyecto:\n%s\n\n", prettyJSON(proyecto["proyecto"]))
	fmt.Fprintf(&b, "Asignaciones activas:\n%s\n\n", prettyJSON(proyecto["asignaciones"]))
	fmt.Fprintf(&b, "Worktrees activas:\n%s\n\n", prettyJSON(proyecto["worktrees"]))
	fmt.Fprintf(&b, "Tareas activas:\n%s\n\n", prettyJSON(proyecto["tareas"]))
	fmt.Fprintf(&b, "Propuestas abiertas:\n%s\n\n", prettyJSON(proyecto["propuestas"]))
	b.WriteString("Genera:\n")
	b.WriteString("1. resumen operativo del proyecto\n")
	b.WriteString("2. riesgos de coordinacion entre agentes\n")
	b.WriteString("3. siguiente bloque tecnico con mejor retorno\n")
	b.WriteString("4. precauciones para no romper la arquitectura hexagonal ni tocar datos de forma destructiva\n")
	return b.String(), nil
}

func buildFiltroTareas(args map[string]any) (db.FiltroTareas, error) {
	var filtro db.FiltroTareas
	if estado := optionalStringArg(args, "estado"); estado != "" {
		e := db.EstadoTarea(estado)
		filtro.Estado = &e
	}
	if agente := optionalStringArg(args, "agente"); agente != "" {
		filtro.Agente = &agente
	}
	if modulo := optionalStringArg(args, "modulo"); modulo != "" {
		filtro.Modulo = &modulo
	}
	if propuesta := optionalStringArg(args, "propuesta"); propuesta != "" {
		pid, err := tareasService.ResolveProposalID(propuesta)
		if err != nil {
			return filtro, err
		}
		filtro.PropuestaID = pid
	}
	if libre, ok := boolArg(args, "libre"); ok {
		filtro.Libre = libre
	}
	return filtro, nil
}

func buildSolicitudDeteccionSkillMCP(args map[string]any) (*db.SolicitudDeteccionSkill, error) {
	tipoAgente := optionalStringArg(args, "tipo_agente")
	if strings.TrimSpace(tipoAgente) == "" {
		if agente := optionalStringArg(args, "agente"); strings.TrimSpace(agente) != "" {
			item, err := db.GetAgente(strings.TrimSpace(agente))
			if err != nil {
				return nil, err
			}
			tipoAgente = item.Rol
		}
	}
	if strings.TrimSpace(tipoAgente) == "" {
		return nil, fmt.Errorf("tipo_agente o agente obligatorio")
	}
	return &db.SolicitudDeteccionSkill{
		TipoAgente:       tipoAgente,
		Nombre:           optionalStringArg(args, "nombre"),
		Descripcion:      optionalStringArg(args, "descripcion"),
		CuandoUsar:       optionalStringArg(args, "cuando_usar"),
		Escenario:        optionalStringArg(args, "escenario"),
		AliasesJSON:      optionalStringArg(args, "aliases_json"),
		HerramientasJSON: optionalStringArg(args, "herramientas_json"),
	}, nil
}

func listarConectores() ([]map[string]any, error) {
	rows, err := operacionesService.ListConnectors()
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	for _, row := range rows {
		list = append(list, map[string]any{
			"slug":       row.Slug,
			"nombre":     row.Nombre,
			"transporte": row.Transporte,
			"comando":    row.Comando,
			"args_json":  row.ArgsJSON,
			"env_json":   row.EnvJSON,
			"metadata":   row.MetadataJSON,
			"activo":     row.Activo,
		})
	}
	return list, nil
}

func listarProyectos() ([]map[string]any, error) {
	return listarProyectosFiltrados("", nil)
}

func listarProyectosFiltrados(tipo string, activo *bool) ([]map[string]any, error) {
	rows, err := memoriaProyectoService.ListProjects(activo)
	if err != nil {
		return nil, err
	}

	var list []map[string]any
	for _, row := range rows {
		if strings.TrimSpace(tipo) != "" && string(row.Tipo) != strings.TrimSpace(tipo) {
			continue
		}
		var parentID any
		if row.ParentID != nil {
			parentID = *row.ParentID
		}
		list = append(list, map[string]any{
			"id":         row.ID,
			"slug":       row.Slug,
			"nombre":     row.Nombre,
			"ruta_abs":   row.RutaAbs,
			"tipo":       row.Tipo,
			"parent_id":  parentID,
			"activo":     row.Activo,
			"created_at": row.CreatedAt.Format(time.RFC3339),
			"updated_at": row.UpdatedAt.Format(time.RFC3339),
		})
	}
	return list, nil
}

func listarAsignaciones(estado string) ([]map[string]any, error) {
	return listarAsignacionesFiltradas(estado, "")
}

func listarAsignacionesFiltradas(estado, agente string) ([]map[string]any, error) {
	rows, err := operacionesService.ListAssignments(estado, agente)
	if err != nil {
		return nil, err
	}

	var list []map[string]any
	for _, row := range rows {
		var cerradaAt any
		if row.CerradaAt != nil {
			cerradaAt = row.CerradaAt.Format(time.RFC3339)
		}
		list = append(list, map[string]any{
			"id":            row.ID,
			"agente":        row.Agente,
			"proyecto_id":   row.ProyectoID,
			"proyecto_slug": row.ProyectoSlug,
			"estado":        row.Estado,
			"nota":          row.Nota,
			"created_at":    row.CreatedAt.Format(time.RFC3339),
			"updated_at":    row.UpdatedAt.Format(time.RFC3339),
			"cerrada_at":    cerradaAt,
		})
	}
	return list, nil
}

func listarWorktrees(estado string) ([]map[string]any, error) {
	return listarWorktreesFiltradas(estado, "")
}

func listarWorktreesFiltradas(estado, agente string) ([]map[string]any, error) {
	rows, err := gitgobernanza.NewService(gitgobernanza.Repository{}).ListWorktrees(estado, agente)
	if err != nil {
		return nil, err
	}

	var list []map[string]any
	for _, row := range rows {
		var tareaID, lockID, cerradaAt any
		if row.TareaID != nil {
			tareaID = *row.TareaID
		}
		if row.LockID != nil {
			lockID = *row.LockID
		}
		if row.CerradaAt != nil {
			cerradaAt = row.CerradaAt.Format(time.RFC3339)
		}
		list = append(list, map[string]any{
			"id":            row.ID,
			"proyecto_id":   row.ProyectoID,
			"proyecto_slug": row.ProyectoSlug,
			"tarea_id":      tareaID,
			"lock_id":       lockID,
			"agente":        row.Agente,
			"nombre":        row.Nombre,
			"ruta_abs":      row.RutaAbs,
			"branch":        row.Branch,
			"base_ref":      row.BaseRef,
			"estado":        row.Estado,
			"motivo":        row.Motivo,
			"created_at":    row.CreatedAt.Format(time.RFC3339),
			"updated_at":    row.UpdatedAt.Format(time.RFC3339),
			"cerrada_at":    cerradaAt,
		})
	}
	return list, nil
}

func listarLocks(estado string) ([]map[string]any, error) {
	return listarLocksFiltrados(estado, "")
}

func listarLocksFiltrados(estado, agente string) ([]map[string]any, error) {
	rows, err := gitgobernanza.NewService(gitgobernanza.Repository{}).ListLocks(estado, agente)
	if err != nil {
		return nil, err
	}

	var list []map[string]any
	for _, row := range rows {
		var proyectoID, tareaID, sesionID, heartbeatAt, liberadaAt any
		if row.ProyectoID != nil {
			proyectoID = *row.ProyectoID
		}
		if row.TareaID != nil {
			tareaID = *row.TareaID
		}
		if row.SesionID != nil {
			sesionID = *row.SesionID
		}
		if row.HeartbeatAt != nil {
			heartbeatAt = row.HeartbeatAt.Format(time.RFC3339)
		}
		if row.LiberadaAt != nil {
			liberadaAt = row.LiberadaAt.Format(time.RFC3339)
		}
		list = append(list, map[string]any{
			"id":           row.ID,
			"proyecto_id":  proyectoID,
			"tarea_id":     tareaID,
			"sesion_id":    sesionID,
			"agente":       row.Agente,
			"scope_type":   row.ScopeType,
			"scope_key":    row.ScopeKey,
			"ruta_abs":     row.RutaAbs,
			"branch":       row.Branch,
			"motivo":       row.Motivo,
			"token_lease":  row.TokenLease,
			"estado":       row.Estado,
			"heartbeat_at": heartbeatAt,
			"expires_at":   row.ExpiresAt.Format(time.RFC3339),
			"created_at":   row.CreatedAt.Format(time.RFC3339),
			"updated_at":   row.UpdatedAt.Format(time.RFC3339),
			"liberada_at":  liberadaAt,
		})
	}
	return list, nil
}

func listarSesionesActivas() ([]map[string]any, error) {
	rows, err := operacionesService.ListActiveSessions()
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	for _, row := range rows {
		var fin, conectorID, proyectoID, heartbeatAt, pid any
		if row.Fin != nil {
			fin = row.Fin.Format(time.RFC3339)
		}
		if row.ConectorID != nil {
			conectorID = *row.ConectorID
		}
		if row.ProyectoID != nil {
			proyectoID = *row.ProyectoID
		}
		if row.HeartbeatAt != nil {
			heartbeatAt = row.HeartbeatAt.Format(time.RFC3339)
		}
		if row.PID != nil {
			pid = *row.PID
		}
		list = append(list, map[string]any{
			"id":                  row.ID,
			"agente":              row.Agente,
			"inicio":              row.Inicio.Format(time.RFC3339),
			"fin":                 fin,
			"activa":              row.Activa,
			"conector_id":         conectorID,
			"conector_slug":       row.ConectorSlug,
			"proyecto_id":         proyectoID,
			"proyecto_slug":       row.ProyectoSlug,
			"estado":              row.Estado,
			"cwd":                 row.Cwd,
			"herramienta":         row.Herramienta,
			"external_session_id": row.ExternalSessionID,
			"resume_payload_json": row.ResumePayloadJSON,
			"resumen_continuidad": row.ResumenContinuidad,
			"branch":              row.Branch,
			"heartbeat_at":        heartbeatAt,
			"host":                row.Host,
			"pid":                 pid,
		})
	}
	return list, nil
}

func detalleProyecto(slug string) (map[string]any, error) {
	rows, err := listarProyectosFiltrados("", nil)
	if err != nil {
		return nil, err
	}
	var proyecto map[string]any
	for _, item := range rows {
		if item["slug"] == slug {
			proyecto = item
			break
		}
	}
	if proyecto == nil {
		return nil, fmt.Errorf("proyecto no encontrado: %s", slug)
	}

	asignaciones, err := listarAsignaciones("")
	if err != nil {
		return nil, err
	}
	worktrees, err := listarWorktrees("")
	if err != nil {
		return nil, err
	}
	tareas, err := tareasService.List(db.FiltroTareas{Modulo: strPtr(slug)})
	if err != nil {
		return nil, err
	}
	propuestas, err := propuestasService.List(ptrPropuestaEstado(db.PropuestaAbierta))
	if err != nil {
		return nil, err
	}

	var proyectoAsignaciones []map[string]any
	for _, a := range asignaciones {
		if a["proyecto_slug"] == slug {
			proyectoAsignaciones = append(proyectoAsignaciones, a)
		}
	}
	var proyectoWorktrees []map[string]any
	for _, w := range worktrees {
		if w["proyecto_slug"] == slug {
			proyectoWorktrees = append(proyectoWorktrees, w)
		}
	}
	var propuestasLite []propuestaLite
	for _, p := range propuestas {
		propuestasLite = append(propuestasLite, propuestaLite{
			ID:           p.ID,
			Codigo:       p.Codigo,
			Titulo:       p.Titulo,
			Estado:       p.Estado,
			PropuestoPor: p.PropuestoPor,
		})
	}

	return map[string]any{
		"proyecto":     proyecto,
		"asignaciones": proyectoAsignaciones,
		"worktrees":    proyectoWorktrees,
		"tareas":       tareas,
		"propuestas":   propuestasLite,
	}, nil
}

func listarPoolsResumenMCP(activo *bool) ([]map[string]any, error) {
	pools, err := capacidadService.ListPoolsSummary(activo)
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	for _, item := range pools {
		list = append(list, map[string]any{
			"id":                    item.Pool.ID,
			"slug":                  item.Pool.Slug,
			"proveedor":             item.Pool.Proveedor,
			"runtime":               item.Pool.Runtime,
			"plan":                  item.Pool.Plan,
			"es_de_pago":            item.Pool.EsDePago,
			"capacidad_total":       item.Pool.CapacidadTotal,
			"capacidad_reservada":   item.Pool.CapacidadReservada,
			"sesiones_activas":      item.SesionesActivas,
			"capacidad_disponible":  item.CapacidadDisponible,
			"permite_hijos":         item.Pool.PermiteHijos,
			"permite_modelos_multi": item.Pool.PermiteModelosMulti,
			"permite_sobrecoste":    item.Pool.PermiteSobrecoste,
			"politica_handoff":      item.Pool.PoliticaHandoff,
			"fuente_telemetria":     item.Pool.FuenteTelemetria,
			"metadata_json":         item.Pool.MetadataJSON,
			"activo":                item.Pool.Activo,
		})
	}
	return list, nil
}

func listarPoliticasModeloMCP(scopeTipo, scopeRef string, activa *bool) ([]map[string]any, error) {
	items, err := capacidadService.ListModelPolicies(scopeTipo, scopeRef, activa)
	if err != nil {
		return nil, err
	}
	var list []map[string]any
	for _, item := range items {
		list = append(list, map[string]any{
			"id":               item.ID,
			"scope_tipo":       item.ScopeTipo,
			"scope_ref":        item.ScopeRef,
			"perfil_tarea":     item.PerfilTarea,
			"pool_slug":        item.PoolSlug,
			"model_slug":       item.ModelSlug,
			"reasoning_effort": item.ReasoningEffort,
			"prioridad":        item.Prioridad,
			"activa":           item.Activa,
			"metadata_json":    item.MetadataJSON,
			"created_at":       item.CreatedAt,
			"updated_at":       item.UpdatedAt,
		})
	}
	return list, nil
}

func detallePoolMCP(slug string) (map[string]any, error) {
	detail, err := capacidadService.GetPoolDetail(slug)
	if err != nil {
		return nil, err
	}
	poolResumen := map[string]any{
		"id":                    detail.Pool.ID,
		"slug":                  detail.Pool.Slug,
		"proveedor":             detail.Pool.Proveedor,
		"runtime":               detail.Pool.Runtime,
		"plan":                  detail.Pool.Plan,
		"es_de_pago":            detail.Pool.EsDePago,
		"capacidad_total":       detail.Pool.CapacidadTotal,
		"capacidad_reservada":   detail.Pool.CapacidadReservada,
		"sesiones_activas":      detail.SesionesActivas,
		"capacidad_disponible":  detail.CapacidadDisponible,
		"permite_hijos":         detail.Pool.PermiteHijos,
		"permite_modelos_multi": detail.Pool.PermiteModelosMulti,
		"permite_sobrecoste":    detail.Pool.PermiteSobrecoste,
		"politica_handoff":      detail.Pool.PoliticaHandoff,
		"fuente_telemetria":     detail.Pool.FuenteTelemetria,
		"metadata_json":         detail.Pool.MetadataJSON,
		"activo":                detail.Pool.Activo,
	}
	return map[string]any{
		"pool":    detail.Pool,
		"resumen": poolResumen,
		"modelos": detail.Modelos,
	}, nil
}

func resourceText(uri, mimeType, text string) []map[string]any {
	return []map[string]any{{
		"uri":      uri,
		"mimeType": mimeType,
		"text":     text,
	}}
}

func toolResult(text string, structured any, isError bool) map[string]any {
	result := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": text,
		}},
		"isError": isError,
	}
	if structured != nil {
		result["structuredContent"] = structured
	}
	return result
}

func prettyJSON(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}

func arquitecturaMCPText() string {
	return `# MCP en Orquesta

MCP se implementa como adaptador de entrada, no como nucleo.

## Principios
- El nucleo de Orquesta no debe depender de MCP, SQLite ni de un proveedor concreto.
- MCP sirve para exponer contexto y acciones controladas al agente.
- La base de datos sigue siendo un adaptador; las mutaciones deben apoyarse en reglas ya existentes.
- La evolucion debe ser aditiva e idempotente.

## Primer bloque expuesto
- resources: estado del orquestador, propuestas, tareas, conectores y briefing por agente
- prompts: briefing operativo, revision de propuesta y plan de tarea
- tools: listar tareas/propuestas, iniciar tarea y votar propuesta

## Criterio de implantacion
- Priorizar lectura de contexto y acciones seguras.
- Mantener confirmacion humana para mutaciones.
- No duplicar logica de negocio en la capa MCP.`
}

func audienceAssistant(priority float64) map[string]any {
	return map[string]any{
		"audience":     []string{"assistant", "user"},
		"priority":     priority,
		"lastModified": time.Now().UTC().Format(time.RFC3339),
	}
}

func ptrEstadoTarea(v db.EstadoTarea) *db.EstadoTarea {
	return &v
}

func ptrPropuestaEstado(v db.EstadoPropuesta) *db.EstadoPropuesta {
	return &v
}

func strPtr(v string) *string {
	return &v
}

func optionalStringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	v, ok := args[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func requiredStringArg(args map[string]any, key string) (string, error) {
	value := optionalStringArg(args, key)
	if value == "" {
		return "", fmt.Errorf("%s es obligatorio", key)
	}
	return value, nil
}

func requiredInt64Arg(args map[string]any, key string) (int64, error) {
	if args == nil {
		return 0, fmt.Errorf("%s es obligatorio", key)
	}
	v, ok := args[key]
	if !ok || v == nil {
		return 0, fmt.Errorf("%s es obligatorio", key)
	}
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case json.Number:
		return n.Int64()
	case string:
		return strconv.ParseInt(strings.TrimSpace(n), 10, 64)
	default:
		return 0, fmt.Errorf("%s debe ser entero", key)
	}
}

func boolArg(args map[string]any, key string) (bool, bool) {
	if args == nil {
		return false, false
	}
	v, ok := args[key]
	if !ok || v == nil {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

func boolPtrArg(args map[string]any, key string) *bool {
	if v, ok := boolArg(args, key); ok {
		return &v
	}
	return nil
}

func invalidParams(err error) *mcpError {
	return &mcpError{Code: -32602, Message: "Parametros inválidos", Data: err.Error()}
}

func internalRPCError(err error) *mcpError {
	return &mcpError{Code: -32603, Message: "Error interno", Data: err.Error()}
}

func resourceNotFound(uri string, err error) *mcpError {
	return &mcpError{
		Code:    -32002,
		Message: "Resource not found",
		Data: map[string]any{
			"uri":     uri,
			"detalle": err.Error(),
		},
	}
}

func toolCallError(err error) *mcpError {
	return &mcpError{
		Code:    -32602,
		Message: "Error en tool",
		Data:    err.Error(),
	}
}
