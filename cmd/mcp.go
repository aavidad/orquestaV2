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
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitgobernanza"
	"orquesta/lenguajeapp"
	"orquesta/notificaciones"
	"orquesta/orquestacionagentesapp"
	"orquesta/panelapp"
	"orquesta/propuestasapp"
	"orquesta/reviewapp"
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
var sesionesAPIService = sesionesapp.NewService(db.SqliteSesionesRepo{})
var tareasService = tareasapp.NewService(tareasapp.Repository{})
var mcpRecogerResultadoGitSubagenteFn = recogerResultadoGitSubagente
var mcpPipelineDispatchFn = capacidadService.EjecutarYDespacharSiguientePasoPipelineLocalDeterminista

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

type mcpHandleOptions struct {
	RequireInitialize bool
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
	result, rpcErr := handleMCPRequest(req, s.initialized, mcpHandleOptions{RequireInitialize: true})
	if rpcErr == nil && req.Method == "initialize" {
		s.initialized = true
	}
	return result, rpcErr
}

func handleMCPRequest(req mcpRequest, initialized bool, opts mcpHandleOptions) (any, *mcpError) {
	if opts.RequireInitialize && req.Method != "initialize" && req.Method != "notifications/initialized" && req.Method != "ping" && !initialized {
		return nil, &mcpError{Code: -32002, Message: "Servidor MCP no inicializado"}
	}
	switch req.Method {
	case "initialize":
		return handleMCPInitialize(req.Params)

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

func handleMCPInitialize(raw json.RawMessage) (map[string]any, *mcpError) {
	var params struct {
		ProtocolVersion string         `json:"protocolVersion"`
		Capabilities    map[string]any `json:"capabilities"`
		ClientInfo      map[string]any `json:"clientInfo"`
	}
	if err := decodeParams(raw, &params); err != nil {
		return nil, invalidParams(err)
	}
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
			URI:         "orquesta://server/operational",
			Name:        "server-operational",
			Title:       "Estado operativo del servidor",
			Description: "Salud operativa canónica del control plane con degradación rápida",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(1),
		},
		{
			URI:         "orquesta://workspace/control",
			Name:        "workspace-control",
			Title:       "Control global del workspace",
			Description: "Vista global compacta del workspace con proyectos activos, flota y autonomia reciente",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(1),
		},
		{
			URI:         "orquesta://openclaw/operator",
			Name:        "openclaw-operator",
			Title:       "Snapshot operador OpenClaw",
			Description: "Snapshot canónico del operador OpenClaw para decisión remota por MCP",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(1),
		},
		{
			URI:         "orquesta://runtimes/tree",
			Name:        "runtimes-tree",
			Title:       "Árbol de runtimes",
			Description: "Vista canónica del árbol de runtimes; admite ?agente=...&proyecto=...&activos=true|false",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
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
			Description: "Listado de agentes con sesión activa; acepta schema=canonical para la vista canónica",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.8),
		},
		{
			URI:         "orquesta://agentes/panel",
			Name:        "agentes-panel",
			Title:       "Panel canónico de agentes",
			Description: "Vista canónica de flota por agente para operadores y OpenClaw",
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
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
		resources = append(resources, mcpResource{
			URI:         "orquesta://proyectos/" + proyecto["slug"].(string) + "/control",
			Name:        "proyecto-control-" + proyecto["slug"].(string),
			Title:       fmt.Sprintf("Control %v", proyecto["nombre"]),
			Description: "Control operativo y Git del proyecto " + proyecto["slug"].(string),
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
		})
		resources = append(resources, mcpResource{
			URI:         "orquesta://git/stats?proyecto=" + url.QueryEscape(proyecto["slug"].(string)),
			Name:        "git-stats-" + proyecto["slug"].(string),
			Title:       fmt.Sprintf("Git stats %v", proyecto["nombre"]),
			Description: "Métricas Git canónicas del proyecto " + proyecto["slug"].(string),
			MIMEType:    "application/json",
			Annotations: audienceAssistant(0.9),
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
	resources = append(resources, microprogramacionMCPResources()...)

	sort.Slice(resources, func(i, j int) bool {
		return resources[i].URI < resources[j].URI
	})
	return resources, nil
}

func mcpResourceTemplates() []mcpResourceTemplate {
	templates := []mcpResourceTemplate{
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
			URITemplate: "orquesta://agentes/{agente}/guidance",
			Name:        "guidance-agente",
			Title:       "Guidance canónica de agente",
			Description: "Contrato estructurado de guidance para el agente: intención, principios, protocolo, seguridad, verificación y recuperación",
			MIMEType:    "text/markdown",
		},
		{
			URITemplate: "orquesta://agentes/{agente}/overview",
			Name:        "overview-agente",
			Title:       "Overview de agente",
			Description: "Detalle operativo canónico del agente: leases, foco de tarea y orden dominante",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://agentes/{agente}/actividad",
			Name:        "actividad-agente",
			Title:       "Actividad de agente",
			Description: "Resumen operativo y Git del agente; admite ?desde=1h o RFC3339",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://runtime/transcript/{agente}",
			Name:        "transcript-runtime-agente",
			Title:       "Transcript de runtime por agente",
			Description: "Transcript canónico del runtime; admite ?proyecto=slug&classification=...&stream=...&desde=RFC3339&limit=50",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://runtimes/tree{?agente,proyecto,activos}",
			Name:        "runtimes-tree",
			Title:       "Árbol de runtimes",
			Description: "Vista canónica del árbol de runtimes por agente/proyecto y filtro activos",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://runtime/trace{?agente,proyecto,handle_id,bytes}",
			Name:        "trace-runtime",
			Title:       "Trace raw de runtime",
			Description: "Traza raw PTY canónica por agente/proyecto o handle; admite ?bytes=8192",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://runtime/checkpoints/{agente}",
			Name:        "checkpoints-runtime-agente",
			Title:       "Checkpoints de runtime por agente",
			Description: "Lista checkpoints de runtime; admite ?proyecto=slug&kind=...&source=...&limit=20",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://runtime/checkpoints/{agente}/latest",
			Name:        "checkpoint-runtime-agente-latest",
			Title:       "Último checkpoint de runtime por agente",
			Description: "Devuelve el último checkpoint de runtime; admite ?proyecto=slug",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://openclaw/operator{?supervisor}",
			Name:        "openclaw-operator",
			Title:       "Snapshot operador OpenClaw",
			Description: "Snapshot canónico del operador OpenClaw; admite ?supervisor=OpenClaw",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://supervision/{supervisor}/briefing",
			Name:        "briefing-supervisor",
			Title:       "Briefing de supervisor",
			Description: "Resumen operativo integral para el agente jefe: flota, cuota, tareas retenidas y frentes activos",
			MIMEType:    "text/markdown",
		},
		{
			URITemplate: "orquesta://supervision/{supervisor}/guidance",
			Name:        "guidance-supervisor",
			Title:       "Guidance canónica de supervisor",
			Description: "Contrato estructurado del supervisor OpenClaw: intención, principios, protocolo, seguridad, verificación y recuperación",
			MIMEType:    "text/markdown",
		},
		{
			URITemplate: "orquesta://supervision/{supervisor}/revision",
			Name:        "revision-supervisor",
			Title:       "Cola de revisión del supervisor",
			Description: "Resumen estrecho de gates y señales recientes para decidir integración, cambios o relevo",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://supervision/{supervisor}/eventos",
			Name:        "eventos-supervisor",
			Title:       "Eventos normalizados del supervisor",
			Description: "Cola compacta de eventos normalizados para el supervisor/OpenClaw",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://supervision/{supervisor}/threads",
			Name:        "threads-supervisor",
			Title:       "Threads del supervisor",
			Description: "Tracking ligero de threads y subagentes del supervisor para handoff, review e integración",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://supervision/{supervisor}/pipeline",
			Name:        "pipeline-supervisor",
			Title:       "Pipeline del supervisor",
			Description: "Estado explícito de fases y pipeline operativa del supervisor/OpenClaw",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://supervision/{supervisor}/subagents",
			Name:        "subagents-supervisor",
			Title:       "Subagentes del supervisor",
			Description: "Modelo persistente de subagentes explícitos y perfiles canónicos del supervisor/OpenClaw",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://proyectos/{slug}",
			Name:        "proyecto-por-slug",
			Title:       "Detalle de proyecto",
			Description: "Acceso directo al detalle de un proyecto por su slug",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://proyectos/{slug}/control",
			Name:        "proyecto-control-por-slug",
			Title:       "Control de proyecto",
			Description: "Resumen canónico de control del proyecto con métricas operativas y Git; admite ?desde=1h o RFC3339",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "orquesta://git/stats{?proyecto,path,cwd,desde}",
			Name:        "git-stats",
			Title:       "Git stats canónicos",
			Description: "Métricas Git canónicas por proyecto o path/cwd; admite ?desde=1h o RFC3339",
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
	return append(templates, microprogramacionMCPResourceTemplates()...)
}

func mcpProjectControlReport(slug string, since time.Time) (*projectControlReport, error) {
	if workspaceControlProjectBuilder != nil {
		return workspaceControlProjectBuilder(slug, since)
	}
	return buildProjectControlReport(slug, since)
}

func parseMCPProjectControlSince(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return parseStatsSince("24h")
	}
	return parseStatsSince(raw)
}

func mcpResolveGitStatsScope(projectRef, path, cwd string) (string, string, string, error) {
	projectRef = strings.TrimSpace(projectRef)
	path = normalizeGitStatsPath(path)
	cwd = normalizeGitStatsPath(cwd)
	if path == "" {
		path = cwd
	}
	if projectRef != "" && path != "" {
		return "", "", "", fmt.Errorf("usa proyecto o path/cwd, pero no ambos")
	}
	if projectRef != "" {
		project, err := apiGitStatsProjectLookupFn(projectRef)
		if err != nil {
			return "", "", "", err
		}
		if project == nil || strings.TrimSpace(project.RutaAbs) == "" {
			return "", "", "", fmt.Errorf("proyecto sin ruta git: %s", projectRef)
		}
		return "proyecto", strings.TrimSpace(project.Slug), normalizeGitStatsPath(project.RutaAbs), nil
	}
	if path != "" {
		return "path", "", path, nil
	}
	return "", "", "", fmt.Errorf("path/cwd o proyecto es obligatorio")
}

func mcpGitStatsResponse(projectRef, path, cwd string, since time.Time) (apiGitStatsResponse, error) {
	scope, project, resolvedPath, err := mcpResolveGitStatsScope(projectRef, path, cwd)
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	stats, err := apiGitStatsService.Collect(resolvedPath, since)
	if err != nil {
		return apiGitStatsResponse{}, err
	}
	return apiGitStatsResponse{
		OK:        true,
		Scope:     scope,
		Project:   project,
		Path:      resolvedPath,
		Since:     since.UTC(),
		Generated: time.Now().UTC(),
		Stats:     stats,
	}, nil
}

func mcpRuntimeTraceResponse(handleID int64, agente, proyecto string, maxBytes int) (apiRuntimeTraceResponse, error) {
	req := runtimesapp.RuntimeTraceRequest{
		HandleID: handleID,
		Agente:   strings.TrimSpace(agente),
		Proyecto: strings.TrimSpace(proyecto),
		MaxBytes: maxBytes,
	}
	if req.HandleID <= 0 && req.Agente == "" {
		return apiRuntimeTraceResponse{}, fmt.Errorf("handle_id o agente es obligatorio")
	}
	trace, err := runtimesService.ReadRuntimeTrace(req)
	if err != nil {
		return apiRuntimeTraceResponse{}, err
	}
	return apiRuntimeTraceResponse{Trace: trace}, nil
}

func mcpRuntimeTreeResponse(agente, proyecto string, activos *bool) (apiRuntimeTreeResponse, error) {
	filter := db.FiltroRuntimes{
		Agente:  stringPtrOrNil(strings.TrimSpace(agente)),
		Activos: activos,
	}
	if proyecto := strings.TrimSpace(proyecto); proyecto != "" {
		item, err := runtimesService.GetProject(proyecto)
		if err != nil {
			return apiRuntimeTreeResponse{}, err
		}
		filter.ProyectoID = &item.ID
	}
	tree, err := runtimesService.BuildRuntimeTree(filter)
	if err != nil {
		return apiRuntimeTreeResponse{}, err
	}
	return apiRuntimeTreeResponse{Runtimes: runtimeTreeNodesToAPI(tree)}, nil
}

func runtimeTreeNodesToAPI(nodes []*db.RuntimeTreeNode) []*apiRuntimeTreeNode {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]*apiRuntimeTreeNode, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		out = append(out, &apiRuntimeTreeNode{
			Runtime: node.Runtime,
			Hijos:   runtimeTreeNodesToAPI(node.Hijos),
		})
	}
	return out
}

func readMCPResource(uri string) ([]map[string]any, error) {
	if contents, handled, err := readMicroprogramacionMCPResource(uri); handled {
		return contents, err
	}
	switch {
	case uri == "orquesta://arquitectura/mcp":
		return resourceText(uri, "text/markdown", arquitecturaMCPText()), nil
	case uri == "orquesta://estado/resumen":
		s, err := buildEstadoResumen()
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(s)), nil
	case uri == "orquesta://server/operational":
		info := buildMCPServerOperationalInfo()
		return resourceText(uri, "application/json", prettyJSON(info)), nil
	case strings.HasPrefix(uri, "orquesta://openclaw/operator"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		rich := strings.EqualFold(strings.TrimSpace(parsed.Query().Get("rich")), "true") || strings.TrimSpace(parsed.Query().Get("rich")) == "1"
		payload, err := buildMCPOpenClawOperatorSnapshot(parsed.Query().Get("supervisor"), rich)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(payload)), nil
	case strings.HasPrefix(uri, "orquesta://workspace/control"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		since, err := parseWorkspaceControlSince(parsed.Query().Get("desde"))
		if err != nil {
			return nil, err
		}
		report, err := buildWorkspaceControlReportSince(since)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(apiWorkspaceControlResponse{Control: report})), nil
	case strings.HasPrefix(uri, "orquesta://runtimes/tree"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(parsed.Host) != "runtimes" || strings.TrimSpace(parsed.Path) != "/tree" {
			return nil, fmt.Errorf("runtimes tree inválido")
		}
		var activos *bool
		if raw := strings.TrimSpace(parsed.Query().Get("activos")); raw != "" {
			switch raw {
			case "true":
				v := true
				activos = &v
			case "false":
				v := false
				activos = &v
			default:
				return nil, fmt.Errorf("activos inválido")
			}
		}
		resp, err := mcpRuntimeTreeResponse(parsed.Query().Get("agente"), parsed.Query().Get("proyecto"), activos)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(resp)), nil
	case uri == "orquesta://proyectos":
		proyectos, err := listarProyectos()
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(proyectos)), nil
	case strings.HasPrefix(uri, "orquesta://git/stats"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		since, err := parseStatsSince(parsed.Query().Get("desde"))
		if err != nil {
			return nil, err
		}
		resp, err := mcpGitStatsResponse(parsed.Query().Get("proyecto"), parsed.Query().Get("path"), parsed.Query().Get("cwd"), since)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(resp)), nil
	case strings.HasPrefix(uri, "orquesta://runtime/trace"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(parsed.Host) != "runtime" || strings.TrimSpace(parsed.Path) != "/trace" {
			return nil, fmt.Errorf("runtime trace inválido")
		}
		var handleID int64
		if raw := strings.TrimSpace(parsed.Query().Get("handle_id")); raw != "" {
			value, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || value <= 0 {
				return nil, fmt.Errorf("handle_id inválido")
			}
			handleID = value
		}
		maxBytes := 0
		if raw := strings.TrimSpace(parsed.Query().Get("bytes")); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value <= 0 {
				return nil, fmt.Errorf("bytes inválido")
			}
			maxBytes = value
		}
		resp, err := mcpRuntimeTraceResponse(handleID, parsed.Query().Get("agente"), parsed.Query().Get("proyecto"), maxBytes)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(resp)), nil
	case uri == "orquesta://asignaciones/activas":
		asignaciones, err := listarAsignaciones("activa")
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(asignaciones)), nil
	case strings.HasPrefix(uri, "orquesta://agentes/activos"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(parsed.Host) != "agentes" || parsed.Path != "/activos" {
			return nil, fmt.Errorf("recurso de agentes activos inválido")
		}
		if strings.EqualFold(strings.TrimSpace(parsed.Query().Get("schema")), "canonical") {
			out, err := buildCanonicalActiveAgentsResponse()
			if err != nil {
				return nil, err
			}
			return resourceText(uri, "application/json", prettyJSON(out)), nil
		}
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
	case uri == "orquesta://agentes/panel":
		rows, err := fetchAgentPanelRowsCached(apiAgentsPanelTimeout)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		return resourceText(uri, "application/json", prettyJSON(apiAgentesPanelCanonicalResponse{
			Agents: agentesService.BuildPanelEntities(normalizeAgentPanelRowsForAPI(rows, now), now),
		})), nil
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
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		path := strings.TrimPrefix(parsed.Path, "/")
		if strings.HasSuffix(path, "/control") {
			slug := strings.TrimSuffix(path, "/control")
			since, err := parseMCPProjectControlSince(parsed.Query().Get("desde"))
			if err != nil {
				return nil, err
			}
			report, err := mcpProjectControlReport(slug, since)
			if err != nil {
				return nil, err
			}
			return resourceText(uri, "application/json", prettyJSON(apiProyectoControlResponse{Control: report})), nil
		}
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
	case strings.HasPrefix(uri, "orquesta://agentes/") && strings.HasSuffix(uri, "/guidance"):
		agente := strings.TrimSuffix(strings.TrimPrefix(uri, "orquesta://agentes/"), "/guidance")
		texto, err := buildAgentGuidance(agente)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "text/markdown", texto), nil
	case strings.HasPrefix(uri, "orquesta://agentes/") && strings.Contains(uri, "/overview"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		agente := strings.TrimSuffix(strings.TrimPrefix(parsed.Path, "/"), "/overview")
		if agente == "" || strings.TrimSpace(parsed.Host) != "agentes" {
			return nil, fmt.Errorf("agente inválido")
		}
		detail, err := agentesService.BuildDetailCompact(agente)
		if err != nil {
			return nil, err
		}
		if strings.EqualFold(strings.TrimSpace(parsed.Query().Get("schema")), "canonical") {
			return resourceText(uri, "application/json", prettyJSON(apiAgenteOverviewResponse{
				Agent:  buildCanonicalAgentOverview(detail),
				Detail: detail,
			})), nil
		}
		return resourceText(uri, "application/json", prettyJSON(detail)), nil
	case strings.HasPrefix(uri, "orquesta://agentes/") && strings.Contains(uri, "/actividad"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		agente := strings.TrimSuffix(strings.TrimPrefix(parsed.Path, "/"), "/actividad")
		if agente == "" || strings.TrimSpace(parsed.Host) != "agentes" {
			return nil, fmt.Errorf("agente inválido")
		}
		since, err := parseStatsSince(parsed.Query().Get("desde"))
		if err != nil {
			return nil, err
		}
		report, err := buildAgentActivityReportLocal(agente, strings.TrimSpace(parsed.Query().Get("proyecto")), since, 200, 400)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(report)), nil
	case strings.HasPrefix(uri, "orquesta://runtime/transcript/"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		agente := strings.TrimPrefix(strings.TrimPrefix(parsed.Path, "/"), "transcript/")
		if agente == "" || strings.TrimSpace(parsed.Host) != "runtime" {
			return nil, fmt.Errorf("agente inválido")
		}
		filter := db.FiltroRuntimeTranscript{
			Agente:         stringPtrOrNil(agente),
			Stream:         stringPtrOrNil(strings.TrimSpace(parsed.Query().Get("stream"))),
			Classification: stringPtrOrNil(strings.TrimSpace(parsed.Query().Get("classification"))),
			Query:          stringPtrOrNil(strings.TrimSpace(parsed.Query().Get("q"))),
			Limit:          50,
		}
		if raw := strings.TrimSpace(parsed.Query().Get("desde")); raw != "" {
			since, err := apiParseRFC3339QueryTime(raw)
			if err != nil {
				return nil, err
			}
			filter.Desde = since
		}
		if raw := strings.TrimSpace(parsed.Query().Get("signals_pending")); raw != "" {
			switch strings.ToLower(raw) {
			case "1", "true", "yes", "si", "sí", "on":
				filter.SoloSenalesPend = true
			}
		}
		if raw := strings.TrimSpace(parsed.Query().Get("limit")); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("limit inválido")
			}
			filter.Limit = n
		}
		if proyecto := strings.TrimSpace(parsed.Query().Get("proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return nil, err
			}
			filter.ProyectoID = &p.ID
		}
		items, err := apiListRuntimeTranscriptFn(filter)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(items)), nil
	case strings.HasPrefix(uri, "orquesta://runtime/checkpoints/"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(parsed.Host) != "runtime" {
			return nil, fmt.Errorf("agente inválido")
		}
		if strings.HasSuffix(parsed.Path, "/latest") {
			agente := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(parsed.Path, "/"), "checkpoints/"), "/latest")
			if agente == "" {
				return nil, fmt.Errorf("agente inválido")
			}
			var proyectoID *int64
			if proyecto := strings.TrimSpace(parsed.Query().Get("proyecto")); proyecto != "" {
				p, err := runtimesService.GetProject(proyecto)
				if err != nil {
					return nil, err
				}
				proyectoID = &p.ID
			}
			checkpoint, err := runtimesService.LatestRuntimeCheckpoint(agente, proyectoID)
			if err != nil {
				return nil, err
			}
			return resourceText(uri, "application/json", prettyJSON(checkpoint)), nil
		}
		agente := strings.TrimPrefix(strings.TrimPrefix(parsed.Path, "/"), "checkpoints/")
		if agente == "" {
			return nil, fmt.Errorf("agente inválido")
		}
		filter := db.FiltroRuntimeCheckpoints{Agente: &agente}
		if raw := strings.TrimSpace(parsed.Query().Get("limit")); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("limit inválido")
			}
			filter.Limit = n
		}
		if kind := strings.TrimSpace(parsed.Query().Get("kind")); kind != "" {
			filter.CheckpointKind = &kind
		}
		if source := strings.TrimSpace(parsed.Query().Get("source")); source != "" {
			filter.Source = &source
		}
		if proyecto := strings.TrimSpace(parsed.Query().Get("proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return nil, err
			}
			filter.ProyectoID = &p.ID
		}
		checkpoints, err := runtimesService.ListRuntimeCheckpoints(filter)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(checkpoints)), nil
	case strings.HasPrefix(uri, "orquesta://supervision/") && strings.HasSuffix(uri, "/briefing"):
		supervisor := strings.TrimSuffix(strings.TrimPrefix(uri, "orquesta://supervision/"), "/briefing")
		texto, err := buildSupervisorBriefing(supervisor)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "text/markdown", texto), nil
	case strings.HasPrefix(uri, "orquesta://supervision/") && strings.HasSuffix(uri, "/guidance"):
		supervisor := strings.TrimSuffix(strings.TrimPrefix(uri, "orquesta://supervision/"), "/guidance")
		texto, err := buildSupervisorGuidance(supervisor)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "text/markdown", texto), nil
	case strings.HasPrefix(uri, "orquesta://supervision/") && strings.HasSuffix(uri, "/revision"):
		supervisor := strings.TrimSuffix(strings.TrimPrefix(uri, "orquesta://supervision/"), "/revision")
		snapshot, err := buildSupervisorRevisionSnapshot(supervisor)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(snapshot)), nil
	case strings.HasPrefix(uri, "orquesta://supervision/") && strings.Contains(uri, "/eventos"):
		parsed, err := url.Parse(uri)
		if err != nil {
			return nil, err
		}
		supervisor := strings.TrimSuffix(strings.TrimPrefix(parsed.Path, "/"), "/eventos")
		if supervisor == "" || strings.TrimSpace(parsed.Host) != "supervision" {
			return nil, fmt.Errorf("supervisor inválido")
		}
		limit := mcpSupervisorEventsDefaultLimit
		if raw := strings.TrimSpace(parsed.Query().Get("limit")); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("limit inválido")
			}
			limit = n
		}
		items, err := buildOpenClawNormalizedEvents(limit)
		if err != nil {
			return nil, err
		}
		payload := map[string]any{
			"supervisor":        resolveSupervisorName(supervisor),
			"limit":             limit,
			"normalized_events": items,
		}
		return resourceText(uri, "application/json", prettyJSON(payload)), nil
	case strings.HasPrefix(uri, "orquesta://supervision/") && strings.HasSuffix(uri, "/threads"):
		supervisor := strings.TrimSuffix(strings.TrimPrefix(uri, "orquesta://supervision/"), "/threads")
		snapshot, err := buildSupervisorThreadsSnapshot(supervisor, "", 100)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(snapshot)), nil
	case strings.HasPrefix(uri, "orquesta://supervision/") && strings.HasSuffix(uri, "/pipeline"):
		supervisor := strings.TrimSuffix(strings.TrimPrefix(uri, "orquesta://supervision/"), "/pipeline")
		snapshot, err := buildSupervisorPipelineSnapshot(supervisor, "", 20)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(snapshot)), nil
	case strings.HasPrefix(uri, "orquesta://supervision/") && strings.HasSuffix(uri, "/subagents"):
		supervisor := strings.TrimSuffix(strings.TrimPrefix(uri, "orquesta://supervision/"), "/subagents")
		snapshot, err := buildSupervisorSubagentsSnapshot(supervisor, "", "", 100)
		if err != nil {
			return nil, err
		}
		return resourceText(uri, "application/json", prettyJSON(snapshot)), nil
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
			Name:        "orquesta.briefing.supervisor",
			Title:       "Briefing de supervisor",
			Description: "Devuelve el briefing operativo integral para el agente jefe que coordina la flota",
			Arguments: []mcpPromptArgument{
				{Name: "supervisor", Description: "Nombre del supervisor operativo, por ejemplo OpenClaw", Required: false},
			},
		},
		{
			Name:        "orquesta.guidance.agente",
			Title:       "Guidance canónica de agente",
			Description: "Devuelve la guidance estructurada del agente con secciones canónicas de ejecución, seguridad y recuperación",
			Arguments: []mcpPromptArgument{
				{Name: "agente", Description: "Nombre del agente registrado en Orquesta", Required: true},
			},
		},
		{
			Name:        "orquesta.guidance.supervisor",
			Title:       "Guidance canónica de supervisor",
			Description: "Devuelve la guidance estructurada del supervisor OpenClaw sobre el estado vivo de Orquesta",
			Arguments: []mcpPromptArgument{
				{Name: "supervisor", Description: "Nombre del supervisor operativo, por ejemplo OpenClaw", Required: false},
			},
		},
		{
			Name:        "orquesta.openclaw.operator",
			Title:       "Snapshot operador OpenClaw",
			Description: "Resume el snapshot canónico del operador OpenClaw listo para decidir por MCP",
			Arguments: []mcpPromptArgument{
				{Name: "supervisor", Description: "Nombre del supervisor operativo, por ejemplo OpenClaw", Required: false},
			},
		},
		{
			Name:        "orquesta.supervision.revision",
			Title:       "Cola de revisión del supervisor",
			Description: "Devuelve la vista estrecha de integración: review gates y señales recientes para OpenClaw",
			Arguments: []mcpPromptArgument{
				{Name: "supervisor", Description: "Nombre del supervisor operativo, por ejemplo OpenClaw", Required: false},
			},
		},
		{
			Name:        "orquesta.supervision.eventos",
			Title:       "Eventos del supervisor",
			Description: "Resume la cola compacta de eventos normalizados del supervisor/OpenClaw",
			Arguments: []mcpPromptArgument{
				{Name: "supervisor", Description: "Nombre del supervisor operativo, por ejemplo OpenClaw", Required: false},
				{Name: "limit", Description: "Número máximo de eventos a incluir", Required: false},
			},
		},
		{
			Name:        "orquesta.supervision.threads",
			Title:       "Threads del supervisor",
			Description: "Resume threads y subagentes activos del supervisor/OpenClaw",
			Arguments: []mcpPromptArgument{
				{Name: "supervisor", Description: "Nombre del supervisor operativo, por ejemplo OpenClaw", Required: false},
			},
		},
		{
			Name:        "orquesta.supervision.pipeline",
			Title:       "Pipeline del supervisor",
			Description: "Resume la fase actual y el estado explícito del pipeline del supervisor/OpenClaw",
			Arguments: []mcpPromptArgument{
				{Name: "supervisor", Description: "Nombre del supervisor operativo, por ejemplo OpenClaw", Required: false},
			},
		},
		{
			Name:        "orquesta.supervision.subagentes",
			Title:       "Subagentes del supervisor",
			Description: "Resume los subagentes explícitos y perfiles canónicos del supervisor/OpenClaw",
			Arguments: []mcpPromptArgument{
				{Name: "supervisor", Description: "Nombre del supervisor operativo, por ejemplo OpenClaw", Required: false},
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

	case "orquesta.briefing.supervisor":
		supervisor := optionalStringArg(args, "supervisor")
		briefing, err := buildSupervisorBriefing(supervisor)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Briefing operativo integral para el supervisor de Orquesta",
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

	case "orquesta.guidance.agente":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		guidance, err := buildAgentGuidance(agente)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Guidance canónica de ejecución para un agente de Orquesta",
			"messages": []mcpPromptMessage{
				{
					Role: "user",
					Content: map[string]any{
						"type": "text",
						"text": guidance,
					},
				},
			},
		}, nil

	case "orquesta.guidance.supervisor":
		supervisor := optionalStringArg(args, "supervisor")
		guidance, err := buildSupervisorGuidance(supervisor)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Guidance canónica del supervisor OpenClaw sobre el estado vivo de Orquesta",
			"messages": []mcpPromptMessage{
				{
					Role: "user",
					Content: map[string]any{
						"type": "text",
						"text": guidance,
					},
				},
			},
		}, nil

	case "orquesta.openclaw.operator":
		supervisor := optionalStringArg(args, "supervisor")
		texto, err := buildMCPOpenClawOperatorOverview(supervisor)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Resumen operativo del snapshot canónico de OpenClaw",
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

	case "orquesta.supervision.revision":
		supervisor := optionalStringArg(args, "supervisor")
		briefing, err := buildSupervisorReviewBriefing(supervisor)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Cola estrecha de revisión e integración para el supervisor",
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

	case "orquesta.supervision.eventos":
		supervisor := optionalStringArg(args, "supervisor")
		texto, err := buildSupervisorEventsOverview(supervisor, intArgOrDefault(args, "limit", mcpSupervisorEventsDefaultLimit))
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Resumen de eventos normalizados del supervisor",
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

	case "orquesta.supervision.threads":
		supervisor := optionalStringArg(args, "supervisor")
		texto, err := buildSupervisorThreadsOverview(supervisor)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Resumen de threads y subagentes del supervisor",
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

	case "orquesta.supervision.pipeline":
		supervisor := optionalStringArg(args, "supervisor")
		texto, err := buildSupervisorPipelineOverview(supervisor)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Resumen de pipeline y fases del supervisor",
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

	case "orquesta.supervision.subagentes":
		supervisor := optionalStringArg(args, "supervisor")
		texto, err := buildSupervisorSubagentsOverview(supervisor)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"description": "Resumen de subagentes explícitos y perfiles del supervisor",
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
	tools := []mcpTool{
		{
			Name:        "orquesta.estado.resumen",
			Title:       "Resumen del estado",
			Description: "Devuelve un resumen operativo del estado actual de Orquesta",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
			},
		},
		mcpServerOperationalTool(),
		mcpServerSelfHealTool(),
		{
			Name:        "orquesta.server.rearm",
			Title:       "Rearmar supervisor",
			Description: "Aplica la siguiente acción segura del supervisor para recuperación operativa sin CLI",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		mcpWorkspaceControlTool(),
		mcpOpenClawOperatorTool(),
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
			Name:        "orquesta.tareas.crear",
			Title:       "Crear tarea",
			Description: "Crea una tarea por la vía canónica de Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"titulo":      map[string]any{"type": "string"},
					"descripcion": map[string]any{"type": "string"},
					"proyecto":    map[string]any{"type": "string"},
					"modulo":      map[string]any{"type": "string"},
					"prioridad":   map[string]any{"type": "string"},
					"creado_por":  map[string]any{"type": "string"},
					"notas":       map[string]any{"type": "string"},
					"propuesta":   map[string]any{"type": "string"},
					"agente":      map[string]any{"type": "string"},
				},
				"required":             []string{"titulo"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.tareas.accion",
			Title:       "Acción sobre tarea",
			Description: "Ejecuta una acción canónica sobre una tarea existente",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":           map[string]any{"type": "integer"},
					"accion":       map[string]any{"type": "string", "enum": []string{"tomar", "iniciar", "completar", "bloquear", "desbloquear", "nota", "backlog", "cancelar", "reasignar"}},
					"agente":       map[string]any{"type": "string"},
					"commit":       map[string]any{"type": "string"},
					"motivo":       map[string]any{"type": "string"},
					"resolucion":   map[string]any{"type": "string"},
					"nota":         map[string]any{"type": "string"},
					"nuevo_agente": map[string]any{"type": "string"},
				},
				"required":             []string{"id", "accion"},
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
			Name:        "orquesta.propuestas.crear",
			Title:       "Crear propuesta",
			Description: "Crea una propuesta por la vía canónica de Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"codigo":        map[string]any{"type": "string"},
					"titulo":        map[string]any{"type": "string"},
					"descripcion":   map[string]any{"type": "string"},
					"tipo":          map[string]any{"type": "string"},
					"proyecto":      map[string]any{"type": "string"},
					"propuesto_por": map[string]any{"type": "string"},
					"distribuidor":  map[string]any{"type": "string"},
				},
				"required":             []string{"codigo", "titulo"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.propuestas.accion",
			Title:       "Acción sobre propuesta",
			Description: "Ejecuta una acción canónica sobre una propuesta existente",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"codigo":        map[string]any{"type": "string"},
					"accion":        map[string]any{"type": "string", "enum": []string{"cerrar", "reabrir", "reparar_votos"}},
					"estado_cierre": map[string]any{"type": "string"},
					"agente":        map[string]any{"type": "string"},
				},
				"required":             []string{"codigo", "accion"},
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
			Name:        "orquesta.proyectos.control",
			Title:       "Control de proyecto",
			Description: "Devuelve el resumen canónico de control del proyecto con métricas operativas y Git",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"slug":  map[string]any{"type": "string"},
					"desde": map[string]any{"type": "string"},
				},
				"required":             []string{"slug"},
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
			Name:        "orquesta.lenguaje.politica.ver",
			Title:       "Ver politica de lenguaje",
			Description: "Devuelve la politica global de lenguaje activa",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.lenguaje.politica.fijar",
			Title:       "Fijar politica de lenguaje",
			Description: "Actualiza la politica global de lenguaje por la via canonica",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"default_language":               map[string]any{"type": "string"},
					"documentation_multilang":        map[string]any{"type": "boolean"},
					"apps_multilang":                 map[string]any{"type": "boolean"},
					"documentation_default_language": map[string]any{"type": "string"},
					"apps_default_language":          map[string]any{"type": "string"},
					"allowed_languages":              map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"notes":                          map[string]any{"type": "string"},
					"updated_by":                     map[string]any{"type": "string"},
				},
				"required":             []string{"default_language"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.lenguaje.matriz.listar",
			Title:       "Listar matriz de lenguaje",
			Description: "Lista las reglas activas de la matriz de resolucion de lenguaje",
			InputSchema: map[string]any{
				"type":                 "object",
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.lenguaje.matriz.fijar",
			Title:       "Fijar regla de lenguaje",
			Description: "Crea o actualiza una regla de la matriz de lenguaje",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":      map[string]any{"type": "string"},
					"selector":   map[string]any{"type": "string"},
					"contexto":   map[string]any{"type": "string"},
					"language":   map[string]any{"type": "string"},
					"reason":     map[string]any{"type": "string"},
					"updated_by": map[string]any{"type": "string"},
				},
				"required":             []string{"scope", "selector", "contexto", "language"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.lenguaje.matriz.borrar",
			Title:       "Borrar regla de lenguaje",
			Description: "Elimina una regla de la matriz de lenguaje",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"scope":    map[string]any{"type": "string"},
					"selector": map[string]any{"type": "string"},
					"contexto": map[string]any{"type": "string"},
				},
				"required":             []string{"scope", "selector", "contexto"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.lenguaje.resolver",
			Title:       "Resolver lenguaje",
			Description: "Resuelve el idioma efectivo para un proyecto o una tarea",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"proyecto": map[string]any{"type": "string"},
					"tarea_id": map[string]any{"type": "integer"},
					"contexto": map[string]any{"type": "string"},
				},
				"required":             []string{"proyecto"},
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
			Name:        "orquesta.asignaciones.activar",
			Title:       "Activar asignación",
			Description: "Activa la asignación de un agente a un proyecto por la vía canónica",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"nota":     map[string]any{"type": "string"},
				},
				"required":             []string{"agente", "proyecto"},
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
			Name:        "orquesta.worktrees.preparar",
			Title:       "Preparar worktree",
			Description: "Prepara una worktree por la vía canónica de coordinación",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"tarea_id": map[string]any{"type": "integer"},
					"lock_id":  map[string]any{"type": "integer"},
					"nombre":   map[string]any{"type": "string"},
					"branch":   map[string]any{"type": "string"},
					"base_ref": map[string]any{"type": "string"},
					"motivo":   map[string]any{"type": "string"},
				},
				"required":             []string{"agente", "proyecto"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.worktrees.cerrar",
			Title:       "Cerrar worktree",
			Description: "Cierra una worktree por la vía canónica de coordinación",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":       map[string]any{"type": "integer"},
					"eliminar": map[string]any{"type": "boolean"},
					"motivo":   map[string]any{"type": "string"},
				},
				"required":             []string{"id"},
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
			Name:        "orquesta.locks.adquirir",
			Title:       "Adquirir lock",
			Description: "Adquiere un lock por la vía canónica de coordinación",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":        map[string]any{"type": "string"},
					"proyecto":      map[string]any{"type": "string"},
					"tarea_id":      map[string]any{"type": "integer"},
					"scope_type":    map[string]any{"type": "string"},
					"scope_key":     map[string]any{"type": "string"},
					"ruta":          map[string]any{"type": "string"},
					"branch":        map[string]any{"type": "string"},
					"motivo":        map[string]any{"type": "string"},
					"lease_seconds": map[string]any{"type": "integer"},
				},
				"required":             []string{"agente", "scope_type", "scope_key"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.locks.liberar",
			Title:       "Liberar lock",
			Description: "Libera un lock por la vía canónica de coordinación",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":          map[string]any{"type": "integer"},
					"agente":      map[string]any{"type": "string"},
					"lease_token": map[string]any{"type": "string"},
					"motivo":      map[string]any{"type": "string"},
				},
				"required":             []string{"id", "agente", "lease_token"},
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
			Name:        "orquesta.sesiones.fin",
			Title:       "Cerrar sesión activa",
			Description: "Cierra una sesión activa por la vía canónica del servidor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente": map[string]any{"type": "string"},
				},
				"required":             []string{"agente"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.listar",
			Title:       "Listar agentes",
			Description: "Lista agentes, opcionalmente filtrados por activos",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"activos": map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.pausar",
			Title:       "Pausar agente",
			Description: "Pausa temporalmente un agente por la vía canónica del servidor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":  map[string]any{"type": "string"},
					"minutos": map[string]any{"type": "integer"},
					"motivo":  map[string]any{"type": "string"},
					"accion":  map[string]any{"type": "string"},
					"entidad": map[string]any{"type": "string"},
					"detalle": map[string]any{"type": "string"},
				},
				"required":             []string{"agente", "minutos", "motivo"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.registrar",
			Title:       "Registrar agente",
			Description: "Registra un agente nuevo por la vía canónica; acepta nombre explícito o alta automática por proveedor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nombre":    map[string]any{"type": "string"},
					"rol":       map[string]any{"type": "string"},
					"proveedor": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.accion",
			Title:       "Acción de agente",
			Description: "Aplica una acción canónica de estado sobre un agente",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente": map[string]any{"type": "string"},
					"accion": map[string]any{"type": "string", "enum": []string{"retirar", "rehabilitar", "reset-reanimacion"}},
				},
				"required":             []string{"agente", "accion"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.lanzar",
			Title:       "Lanzar agente",
			Description: "Prepara y arranca un agente por la vía canónica unificada del servidor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":       map[string]any{"type": "string"},
					"proyecto":     map[string]any{"type": "string"},
					"conector":     map[string]any{"type": "string"},
					"modelo":       map[string]any{"type": "string"},
					"razonamiento": map[string]any{"type": "string"},
					"perfil":       map[string]any{"type": "string"},
					"motivo":       map[string]any{"type": "string"},
					"por":          map[string]any{"type": "string"},
				},
				"required":             []string{"agente", "proyecto"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.control",
			Title:       "Control de ciclo de vida de agente",
			Description: "Encola una orden canónica de start, pause, resume o stop para un agente",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":       map[string]any{"type": "string"},
					"accion":       map[string]any{"type": "string", "enum": []string{"start", "pause", "resume", "stop"}},
					"proyecto":     map[string]any{"type": "string"},
					"conector":     map[string]any{"type": "string"},
					"modelo":       map[string]any{"type": "string"},
					"razonamiento": map[string]any{"type": "string"},
					"perfil":       map[string]any{"type": "string"},
					"motivo":       map[string]any{"type": "string"},
					"por":          map[string]any{"type": "string"},
					"tarea_id":     map[string]any{"type": "integer"},
				},
				"required":             []string{"agente", "accion"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.start",
			Title:       "Arrancar agente",
			Description: "Encola una orden canónica de start para un agente",
			InputSchema: mcpAgentLifecycleToolInputSchema("start"),
		},
		{
			Name:        "orquesta.agentes.pause",
			Title:       "Pausar runtime de agente",
			Description: "Encola una orden canónica de pause para un agente",
			InputSchema: mcpAgentLifecycleToolInputSchema("pause"),
		},
		{
			Name:        "orquesta.agentes.resume",
			Title:       "Reanudar agente",
			Description: "Encola una orden canónica de resume para un agente",
			InputSchema: mcpAgentLifecycleToolInputSchema("resume"),
		},
		{
			Name:        "orquesta.agentes.stop",
			Title:       "Detener agente",
			Description: "Encola una orden canónica de stop para un agente",
			InputSchema: mcpAgentLifecycleToolInputSchema("stop"),
		},
		{
			Name:        "orquesta.agentes.handoff",
			Title:       "Handoff entre agentes",
			Description: "Crea un handoff canónico entre agente origen y destino",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente_origen":       map[string]any{"type": "string"},
					"agente_destino":      map[string]any{"type": "string"},
					"tarea_id":            map[string]any{"type": "integer"},
					"motivo":              map[string]any{"type": "string"},
					"resumen":             map[string]any{"type": "string"},
					"external_session_id": map[string]any{"type": "string"},
				},
				"required":             []string{"agente_origen", "agente_destino"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.preparar",
			Title:       "Preparar agente",
			Description: "Construye el bundle canónico de preparación de un agente",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":       map[string]any{"type": "string"},
					"proyecto":     map[string]any{"type": "string"},
					"conector":     map[string]any{"type": "string"},
					"modelo":       map[string]any{"type": "string"},
					"razonamiento": map[string]any{"type": "string"},
					"perfil":       map[string]any{"type": "string"},
				},
				"required":             []string{"agente", "proyecto"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.investigar",
			Title:       "Investigar agente",
			Description: "Ejecuta la investigación canónica de runtime/transcript",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":    map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"limit":    map[string]any{"type": "integer"},
				},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.overview",
			Title:       "Overview de agente",
			Description: "Devuelve el contexto operativo canónico completo de un agente",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente": map[string]any{"type": "string"},
					"schema": map[string]any{"type": "string"},
				},
				"required":             []string{"agente"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.panel",
			Title:       "Panel canónico de agentes",
			Description: "Devuelve la vista canónica de flota por agente",
			InputSchema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.actividad",
			Title:       "Actividad de agente",
			Description: "Resume actividad operativa y Git de un agente en una ventana temporal",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":           map[string]any{"type": "string"},
					"proyecto":         map[string]any{"type": "string"},
					"desde":            map[string]any{"type": "string"},
					"audit_limit":      map[string]any{"type": "integer"},
					"transcript_limit": map[string]any{"type": "integer"},
				},
				"required":             []string{"agente"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.handles.listar",
			Title:       "Listar runtime handles",
			Description: "Lista runtime handles, con resincronización supervisada cuando aplica",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtimes.tree",
			Title:       "Árbol de runtimes",
			Description: "Devuelve la vista canónica del árbol de runtimes por agente, proyecto o filtro de activos",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"activos":  map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.events.listar",
			Title:       "Listar runtime events",
			Description: "Lista runtime events compactados por agente, proyecto, kind o level",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"kind":     map[string]any{"type": "string"},
					"level":    map[string]any{"type": "string"},
					"limit":    map[string]any{"type": "integer"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.trace",
			Title:       "Leer runtime trace raw",
			Description: "Lee la traza raw PTY canónica por handle o agente/proyecto",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":    map[string]any{"type": "string"},
					"proyecto":  map[string]any{"type": "string"},
					"handle_id": map[string]any{"type": "integer"},
					"bytes":     map[string]any{"type": "integer"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.transcript.listar",
			Title:       "Listar runtime transcript",
			Description: "Lista runtime transcript por agente, proyecto, handle, runtime, stream o clasificación",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":          map[string]any{"type": "string"},
					"proyecto":        map[string]any{"type": "string"},
					"runtime_id":      map[string]any{"type": "integer"},
					"handle_id":       map[string]any{"type": "integer"},
					"stream":          map[string]any{"type": "string"},
					"classification":  map[string]any{"type": "string"},
					"q":               map[string]any{"type": "string"},
					"desde":           map[string]any{"type": "string", "description": "RFC3339 timestamp"},
					"signals_pending": map[string]any{"type": "boolean"},
					"limit":           map[string]any{"type": "integer"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.ordenes.listar",
			Title:       "Listar runtime orders",
			Description: "Lista runtime orders por agente, proyecto o estado",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"estado":   map[string]any{"type": "string"},
					"limit":    map[string]any{"type": "integer"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.mailbox.listar",
			Title:       "Listar runtime mailbox",
			Description: "Lista mensajes del runtime mailbox por destino, origen, proyecto o estado",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"to_agente":   map[string]any{"type": "string"},
					"from_agente": map[string]any{"type": "string"},
					"proyecto":    map[string]any{"type": "string"},
					"estado":      map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.mailbox.enviar",
			Title:       "Enviar runtime mailbox",
			Description: "Envia un mensaje al runtime mailbox por la vía canónica del servidor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from_agente":      map[string]any{"type": "string"},
					"to_agente":        map[string]any{"type": "string"},
					"kind":             map[string]any{"type": "string"},
					"proyecto":         map[string]any{"type": "string"},
					"runtime_order_id": map[string]any{"type": "integer"},
					"payload_json":     map[string]any{"type": "string"},
				},
				"required":             []string{"from_agente", "to_agente", "kind"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.mailbox.entregar",
			Title:       "Entregar runtime mailbox",
			Description: "Marca un mensaje de runtime mailbox como entregado",
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
			Name:        "orquesta.runtime.mailbox.consumir",
			Title:       "Consumir runtime mailbox",
			Description: "Marca un mensaje de runtime mailbox como consumido",
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
			Name:        "orquesta.runtime.handles.cerrar",
			Title:       "Cerrar runtime handles residuales",
			Description: "Cierra runtime handles residuales por agente o proyecto",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"actor":    map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.handles.purgar",
			Title:       "Purgar runtime handles inactivos",
			Description: "Purga runtime handles inactivos filtrando por agente, proyecto o estados",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"actor":    map[string]any{"type": "string"},
					"estados": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtimes.cerrar",
			Title:       "Cerrar runtimes residuales",
			Description: "Cierra runtimes residuales activos por agente o proyecto",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"actor":    map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.ordenes.cancelar",
			Title:       "Cancelar runtime order",
			Description: "Cancela una runtime order existente",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "integer"},
					"actor":  map[string]any{"type": "string"},
					"motivo": map[string]any{"type": "string"},
				},
				"required":             []string{"id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.ordenes.purgar",
			Title:       "Purgar runtime orders terminales",
			Description: "Purga runtime orders terminales por agente, proyecto, estados o tipos",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":             map[string]any{"type": "string"},
					"proyecto":           map[string]any{"type": "string"},
					"actor":              map[string]any{"type": "string"},
					"older_than_minutes": map[string]any{"type": "integer"},
					"estados": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"tipos": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.mailbox.limpiar",
			Title:       "Limpiar runtime mailbox",
			Description: "Marca como consumidos mensajes del runtime mailbox filtrados por destino, origen, proyecto, estado o kind",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"to_agente":   map[string]any{"type": "string"},
					"from_agente": map[string]any{"type": "string"},
					"proyecto":    map[string]any{"type": "string"},
					"estados": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"kinds": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.wake",
			Title:       "Wake runtime control plane",
			Description: "Despierta explícitamente los carriles orders, mailbox o warm del control plane",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"orders":  map[string]any{"type": "boolean"},
					"mailbox": map[string]any{"type": "boolean"},
					"warm":    map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.process.orders",
			Title:       "Procesar runtime orders",
			Description: "Fuerza el batch canónico de runtime orders",
			InputSchema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.process.transcript",
			Title:       "Procesar runtime transcript",
			Description: "Fuerza el reprocesado canónico del transcript completo, por handle o por agente/proyecto",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wait":      map[string]any{"type": "boolean"},
					"handle_id": map[string]any{"type": "integer"},
					"agente":    map[string]any{"type": "string"},
					"proyecto":  map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.process.mailbox",
			Title:       "Procesar runtime mailbox",
			Description: "Fuerza el batch canónico de runtime mailbox con filtros opcionales",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"to_agente": map[string]any{"type": "string"},
					"proyecto":  map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.process.autonomia",
			Title:       "Procesar runtime autonomia",
			Description: "Fuerza el batch canónico de autonomía del runtime",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wait": map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.process.degradados",
			Title:       "Procesar runtime degradados",
			Description: "Fuerza el batch canónico de recuperación de agentes degradados",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wait": map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.process.hygiene",
			Title:       "Procesar runtime hygiene",
			Description: "Fuerza el batch canónico de higiene runtime",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wait": map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.process.reanimaciones",
			Title:       "Procesar runtime reanimaciones",
			Description: "Fuerza el batch canónico de reanimaciones de agentes",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"wait": map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.transcript.purgar_ruido",
			Title:       "Purgar ruido de transcript",
			Description: "Purge histórico de ruido UI/spinner del runtime transcript",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"all": map[string]any{"type": "boolean"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.checkpoints.crear",
			Title:       "Crear runtime checkpoint",
			Description: "Crea un checkpoint explícito de runtime por la vía canónica",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":          map[string]any{"type": "string"},
					"proyecto":        map[string]any{"type": "string"},
					"sesion_id":       map[string]any{"type": "integer"},
					"runtime_id":      map[string]any{"type": "integer"},
					"checkpoint_kind": map[string]any{"type": "string"},
					"resumen":         map[string]any{"type": "string"},
					"branch":          map[string]any{"type": "string"},
					"cwd":             map[string]any{"type": "string"},
					"payload":         map[string]any{"type": "string"},
					"resume_strategy": map[string]any{"type": "string"},
					"source":          map[string]any{"type": "string"},
				},
				"required":             []string{"agente"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.checkpoints.listar",
			Title:       "Listar runtime checkpoints",
			Description: "Lista checkpoints de runtime por agente, proyecto, kind o source",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
					"kind":     map[string]any{"type": "string"},
					"source":   map[string]any{"type": "string"},
					"limit":    map[string]any{"type": "integer"},
				},
				"required":             []string{"agente"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.checkpoints.latest",
			Title:       "Último runtime checkpoint",
			Description: "Devuelve el último checkpoint de runtime para un agente y proyecto opcional",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":   map[string]any{"type": "string"},
					"proyecto": map[string]any{"type": "string"},
				},
				"required":             []string{"agente"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.agentes.tick",
			Title:       "Tick de agente",
			Description: "Procesa un tick canónico de agente y devuelve la siguiente acción recomendada",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"agente":                 map[string]any{"type": "string"},
					"proyecto":               map[string]any{"type": "string"},
					"host":                   map[string]any{"type": "string"},
					"pid":                    map[string]any{"type": "integer"},
					"cuota_pct":              map[string]any{"type": "integer"},
					"finalizado":             map[string]any{"type": "boolean"},
					"motivo":                 map[string]any{"type": "string"},
					"ack_start_order_id":     map[string]any{"type": "integer"},
					"ack_bootstrap_order_id": map[string]any{"type": "integer"},
					"ack_mailbox_ids": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "integer"},
					},
				},
				"required":             []string{"agente", "proyecto"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.runtime.nudge",
			Title:       "Nudge runtime",
			Description: "Encola un nudge canónico a un agente a través del control plane",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"to_agente":   map[string]any{"type": "string"},
					"texto":       map[string]any{"type": "string"},
					"from_agente": map[string]any{"type": "string"},
					"proyecto":    map[string]any{"type": "string"},
					"kind":        map[string]any{"type": "string"},
				},
				"required":             []string{"to_agente", "texto"},
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
		{
			Name:        "orquesta.review_gates.listar",
			Title:       "Listar review gates",
			Description: "Lista review gates por proyecto, estado o reviewer",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"proyecto":        map[string]any{"type": "string"},
					"estado":          map[string]any{"type": "string"},
					"reviewer_agente": map[string]any{"type": "string"},
					"limit":           map[string]any{"type": "integer"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.review_gates.resolver",
			Title:       "Resolver review gate",
			Description: "Resuelve un review gate existente con estado y findings",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":              map[string]any{"type": "integer"},
					"estado":          map[string]any{"type": "string"},
					"reviewer_agente": map[string]any{"type": "string"},
					"findings_json":   map[string]any{"type": "string"},
				},
				"required":             []string{"id", "estado"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.git.merges.listar",
			Title:       "Listar merges",
			Description: "Lista solicitudes de merge por proyecto o estado",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"proyecto": map[string]any{"type": "string"},
					"estado":   map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.git.stats",
			Title:       "Git stats canónicos",
			Description: "Devuelve métricas Git canónicas por proyecto o path/cwd",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"proyecto": map[string]any{"type": "string"},
					"path":     map[string]any{"type": "string"},
					"cwd":      map[string]any{"type": "string"},
					"desde":    map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.git.merges.guardar",
			Title:       "Guardar merge",
			Description: "Crea o actualiza una solicitud de merge por la vía canónica de Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":            map[string]any{"type": "integer"},
					"proyecto":      map[string]any{"type": "string"},
					"source_branch": map[string]any{"type": "string"},
					"target_branch": map[string]any{"type": "string"},
					"requested_by":  map[string]any{"type": "string"},
					"estado":        map[string]any{"type": "string"},
					"commit_origen": map[string]any{"type": "string"},
					"commit_merge":  map[string]any{"type": "string"},
					"notas":         map[string]any{"type": "string"},
					"metadata_json": map[string]any{"type": "string"},
				},
				"required":             []string{"proyecto", "source_branch", "target_branch", "requested_by"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.revision",
			Title:       "Cola de revisión del supervisor",
			Description: "Devuelve en JSON la cola estrecha de revisión: gates, señales, merges y colisiones",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.eventos",
			Title:       "Eventos normalizados del supervisor",
			Description: "Devuelve en JSON la cola compacta de eventos normalizados para OpenClaw",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
					"limit":      map[string]any{"type": "integer"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.threads.registrar",
			Title:       "Registrar thread de supervisor",
			Description: "Registra un turno de leader/subagent para OpenClaw sin duplicar la sesión runtime",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
					"proyecto":   map[string]any{"type": "string"},
					"session_id": map[string]any{"type": "string"},
					"thread_id":  map[string]any{"type": "string"},
					"kind":       map[string]any{"type": "string", "enum": []string{"leader", "subagent"}},
					"mode":       map[string]any{"type": "string"},
					"status":     map[string]any{"type": "string", "enum": []string{"active", "idle", "closed"}},
					"source":     map[string]any{"type": "string"},
					"turn_id":    map[string]any{"type": "string"},
				},
				"required":             []string{"thread_id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.threads.listar",
			Title:       "Listar threads del supervisor",
			Description: "Devuelve en JSON el tracking ligero de leader/subagents del supervisor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
					"session_id": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.pipeline.actualizar",
			Title:       "Actualizar pipeline del supervisor",
			Description: "Persiste la fase actual y el estado explícito del pipeline del supervisor/OpenClaw",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor":       map[string]any{"type": "string"},
					"proyecto":         map[string]any{"type": "string"},
					"pipeline_name":    map[string]any{"type": "string"},
					"current_phase":    map[string]any{"type": "string"},
					"status":           map[string]any{"type": "string", "enum": []string{"active", "paused", "blocked", "completed", "failed"}},
					"current_task_id":  map[string]any{"type": "integer"},
					"current_gate_id":  map[string]any{"type": "integer"},
					"current_merge_id": map[string]any{"type": "integer"},
					"artifacts_json":   map[string]any{"type": "string"},
					"metadata_json":    map[string]any{"type": "string"},
				},
				"required":             []string{"pipeline_name", "current_phase"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.pipeline.listar",
			Title:       "Listar pipeline del supervisor",
			Description: "Devuelve en JSON la pipeline operativa explícita del supervisor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
					"proyecto":   map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.subagentes.registrar",
			Title:       "Registrar subagente del supervisor",
			Description: "Persiste un subagente explícito del supervisor/OpenClaw por la vía canónica",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor":        map[string]any{"type": "string"},
					"proyecto":          map[string]any{"type": "string"},
					"session_id":        map[string]any{"type": "string"},
					"parent_thread_id":  map[string]any{"type": "string"},
					"thread_id":         map[string]any{"type": "string"},
					"subagent_name":     map[string]any{"type": "string"},
					"subagent_type":     map[string]any{"type": "string"},
					"tool_profile_json": map[string]any{"type": "string"},
					"status":            map[string]any{"type": "string", "enum": []string{"running", "completed", "failed", "cancelled"}},
					"manifest_path":     map[string]any{"type": "string"},
					"output_path":       map[string]any{"type": "string"},
					"error_message":     map[string]any{"type": "string"},
					"metadata_json":     map[string]any{"type": "string"},
				},
				"required":             []string{"thread_id"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.subagentes.listar",
			Title:       "Listar subagentes del supervisor",
			Description: "Devuelve en JSON el modelo persistente de subagentes explícitos del supervisor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor":    map[string]any{"type": "string"},
					"proyecto":      map[string]any{"type": "string"},
					"session_id":    map[string]any{"type": "string"},
					"status":        map[string]any{"type": "string"},
					"subagent_type": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.subagentes.refrescar_store",
			Title:       "Refrescar store de subagentes Claude",
			Description: "Importa manifests/output del store .clawd-agents al modelo canónico de subagentes",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
					"proyecto":   map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.subagentes.lanzar_externo",
			Title:       "Lanzar subagente Claude externo",
			Description: "Ejecuta un launcher externo configurable y luego sincroniza el store .clawd-agents con Orquesta",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor":    map[string]any{"type": "string"},
					"proyecto":      map[string]any{"type": "string"},
					"name":          map[string]any{"type": "string"},
					"description":   map[string]any{"type": "string"},
					"prompt":        map[string]any{"type": "string"},
					"subagent_type": map[string]any{"type": "string"},
					"model":         map[string]any{"type": "string"},
				},
				"required":             []string{"description", "prompt"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.acciones.aplicar",
			Title:       "Aplicar acción del supervisor",
			Description: "Aplica una acción canónica sugerida al supervisor cuando la semántica es segura",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
					"action":     map[string]any{"type": "string"},
					"target":     map[string]any{"type": "string"},
					"assignee":   map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.acciones.aplicar_lote",
			Title:       "Aplicar lote seguro del supervisor",
			Description: "Aplica en lote acciones canónicas seguras del supervisor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
					"max_items":  map[string]any{"type": "integer"},
					"kind":       map[string]any{"type": "string", "description": "Filtro opcional: proposal, dispatch o guidance"},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "orquesta.supervision.acciones.aplicar_siguiente",
			Title:       "Aplicar siguiente acción segura del supervisor",
			Description: "Aplica la siguiente acción segura de la cola viva del supervisor",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"supervisor": map[string]any{"type": "string"},
				},
				"additionalProperties": false,
			},
		},
	}
	return append(tools, microprogramacionMCPTools()...)
}

func mcpAgentLifecycleToolInputSchema(action string) map[string]any {
	properties := map[string]any{
		"agente":       map[string]any{"type": "string"},
		"proyecto":     map[string]any{"type": "string"},
		"conector":     map[string]any{"type": "string"},
		"modelo":       map[string]any{"type": "string"},
		"razonamiento": map[string]any{"type": "string"},
		"perfil":       map[string]any{"type": "string"},
		"motivo":       map[string]any{"type": "string"},
		"por":          map[string]any{"type": "string"},
		"tarea_id":     map[string]any{"type": "integer"},
	}
	required := []string{"agente"}
	if strings.EqualFold(strings.TrimSpace(action), "start") {
		required = append(required, "proyecto")
	}
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func callMCPAgentLifecycleTool(args map[string]any, forcedAction string) (map[string]any, error) {
	agente, err := requiredStringArg(args, "agente")
	if err != nil {
		return nil, err
	}
	accion := strings.TrimSpace(forcedAction)
	if accion == "" {
		accion, err = requiredStringArg(args, "accion")
		if err != nil {
			return nil, err
		}
	}
	req := apiAgenteControlRequest{
		Agente:       agente,
		Accion:       accion,
		Proyecto:     optionalStringArg(args, "proyecto"),
		Conector:     optionalStringArg(args, "conector"),
		Modelo:       optionalStringArg(args, "modelo"),
		Razonamiento: optionalStringArg(args, "razonamiento"),
		Perfil:       optionalStringArg(args, "perfil"),
		Motivo:       optionalStringArg(args, "motivo"),
		Por:          optionalStringArg(args, "por"),
	}
	if tareaID := optionalInt64Arg(args, "tarea_id"); tareaID > 0 {
		req.TareaID = &tareaID
	}
	orderID, accionCanonica, err := encolarControlAgenteLocalWithTimeout(req)
	if err != nil {
		return toolResult(err.Error(), nil, true), nil
	}
	result := apiAgenteControlResponse{
		OK:     true,
		ID:     orderID,
		Agente: strings.TrimSpace(req.Agente),
		Accion: strings.TrimSpace(accionCanonica),
	}
	return toolResult(prettyJSON(result), result, false), nil
}

func callMCPTool(name string, args map[string]any) (map[string]any, error) {
	if result, handled, err := callMicroprogramacionMCPTool(name, args); handled {
		return result, err
	}
	switch strings.TrimSpace(name) {
	case "orquesta.estado.resumen":
		resumen, err := buildEstadoResumen()
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(resumen), resumen, false), nil

	case "orquesta.server.operational":
		return callMCPServerOperational()

	case "orquesta.server.self_heal":
		return callMCPServerSelfHeal(args)

	case "orquesta.server.rearm":
		result, err := serverOperationalApplyNextAction(optionalStringArg(args, "supervisor"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.workspace.control":
		return callMCPWorkspaceControl(args)

	case "orquesta.openclaw.operator":
		return callMCPOpenClawOperator(args)

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

	case "orquesta.tareas.crear":
		id, err := tareasService.Create(tareasapp.CreateTaskInput{
			Titulo:          optionalStringArg(args, "titulo"),
			Descripcion:     optionalStringArg(args, "descripcion"),
			Modulo:          optionalStringArg(args, "modulo"),
			Prioridad:       db.PrioridadTarea(optionalStringArg(args, "prioridad")),
			CreadoPor:       optionalStringArg(args, "creado_por"),
			Agente:          optionalStringArg(args, "agente"),
			Proyecto:        optionalStringArg(args, "proyecto"),
			PropuestaCodigo: optionalStringArg(args, "propuesta"),
			Notas:           optionalStringArg(args, "notas"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		t, err := tareasService.Get(id)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(t), t, false), nil

	case "orquesta.tareas.accion":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, err
		}
		accion, err := requiredStringArg(args, "accion")
		if err != nil {
			return nil, err
		}
		switch accion {
		case "tomar":
			err = tareasService.Take(id, optionalStringArg(args, "agente"))
		case "iniciar":
			err = tareasService.Start(id, optionalStringArg(args, "agente"))
		case "completar":
			err = tareasService.Complete(id, optionalStringArg(args, "agente"), optionalStringArg(args, "commit"))
		case "bloquear":
			err = tareasService.Block(id, optionalStringArg(args, "agente"), optionalStringArg(args, "motivo"))
		case "desbloquear":
			err = tareasService.Unblock(id, optionalStringArg(args, "agente"), optionalStringArg(args, "resolucion"))
		case "nota":
			err = tareasService.Note(id, optionalStringArg(args, "agente"), optionalStringArg(args, "nota"))
		case "backlog":
			err = tareasService.MoveToBacklog(id)
		case "cancelar":
			err = db.CancelarTarea(id, optionalStringArg(args, "agente"), valorConFallback(optionalStringArg(args, "motivo"), "cancelada via MCP"))
		case "reasignar":
			err = tareasService.Reassign(id, optionalStringArg(args, "nuevo_agente"))
		default:
			return nil, fmt.Errorf("acción desconocida: %s", accion)
		}
		if err != nil {
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

	case "orquesta.propuestas.crear":
		id, propuesta, err := propuestasService.Create(propuestasapp.CreateProposalInput{
			Codigo:       optionalStringArg(args, "codigo"),
			Titulo:       optionalStringArg(args, "titulo"),
			Descripcion:  optionalStringArg(args, "descripcion"),
			Tipo:         optionalStringArg(args, "tipo"),
			Proyecto:     optionalStringArg(args, "proyecto"),
			PropuestoPor: optionalStringArg(args, "propuesto_por"),
			Distribuidor: optionalStringArg(args, "distribuidor"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{
			"id":        id,
			"propuesta": propuesta,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.propuestas.accion":
		codigo, err := requiredStringArg(args, "codigo")
		if err != nil {
			return nil, err
		}
		accion, err := requiredStringArg(args, "accion")
		if err != nil {
			return nil, err
		}
		agente := valorConFallback(optionalStringArg(args, "agente"), "OpenClaw")
		switch accion {
		case "cerrar":
			estadoCierre := valorConFallback(optionalStringArg(args, "estado_cierre"), string(db.PropuestaConsenso))
			if err := propuestasService.Close(codigo, estadoCierre, agente); err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
		case "reabrir":
			if _, err := propuestasService.Reopen(codigo, agente); err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
		case "reparar_votos":
			if _, err := propuestasService.RepairPendingVotes(codigo, agente); err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
		default:
			return nil, fmt.Errorf("acción desconocida: %s", accion)
		}
		detail, err := propuestasService.GetDetail(codigo)
		if err != nil {
			return nil, err
		}
		detail.Proposal.Votos = detail.Votes
		return toolResult(prettyJSON(detail.Proposal), detail.Proposal, false), nil

	case "orquesta.proyectos.listar":
		proyectos, err := listarProyectosFiltrados(optionalStringArg(args, "tipo"), boolPtrArg(args, "activo"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(proyectos), proyectos, false), nil

	case "orquesta.proyectos.control":
		slug, err := requiredStringArg(args, "slug")
		if err != nil {
			return nil, err
		}
		since, err := parseMCPProjectControlSince(optionalStringArg(args, "desde"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		report, err := mcpProjectControlReport(slug, since)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := apiProyectoControlResponse{Control: report}
		return toolResult(prettyJSON(result), result, false), nil

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

	case "orquesta.lenguaje.politica.ver":
		politica, err := lenguajeService.GetPolicy()
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(politica), politica, false), nil

	case "orquesta.lenguaje.politica.fijar":
		defaultLanguage, err := requiredStringArg(args, "default_language")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		politica := &lenguajeapp.LanguagePolicy{
			DefaultLanguage:          defaultLanguage,
			DocumentationMultilang:   boolArgOrFalse(args, "documentation_multilang"),
			AppsMultilang:            boolArgOrFalse(args, "apps_multilang"),
			DocumentationDefaultLang: optionalStringArg(args, "documentation_default_language"),
			AppsDefaultLang:          optionalStringArg(args, "apps_default_language"),
			AllowedLanguages:         optionalStringSliceArg(args, "allowed_languages"),
			Notes:                    optionalStringArg(args, "notes"),
		}
		if err := lenguajeService.SetPolicy(politica, optionalStringArg(args, "updated_by")); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		actualizada, err := lenguajeService.GetPolicy()
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(actualizada), actualizada, false), nil

	case "orquesta.lenguaje.matriz.listar":
		entries, err := lenguajeService.ListMatrixEntries()
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(entries), entries, false), nil

	case "orquesta.lenguaje.matriz.fijar":
		scope, err := requiredStringArg(args, "scope")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		selector, err := requiredStringArg(args, "selector")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		contexto, err := requiredStringArg(args, "contexto")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		language, err := requiredStringArg(args, "language")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		entry, err := lenguajeService.SetMatrixEntry(lenguajeapp.SetMatrixEntryInput{
			Scope:     scope,
			Selector:  selector,
			Contexto:  contexto,
			Language:  language,
			Reason:    optionalStringArg(args, "reason"),
			UpdatedBy: optionalStringArg(args, "updated_by"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(entry), entry, false), nil

	case "orquesta.lenguaje.matriz.borrar":
		scope, err := requiredStringArg(args, "scope")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		selector, err := requiredStringArg(args, "selector")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		contexto, err := requiredStringArg(args, "contexto")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		if err := lenguajeService.DeleteMatrixEntry(scope, selector, contexto); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{
			"ok":       true,
			"scope":    scope,
			"selector": selector,
			"contexto": contexto,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.lenguaje.resolver":
		proyecto, err := requiredStringArg(args, "proyecto")
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		resolucion, err := lenguajeService.Resolve(lenguajeapp.ResolveInput{
			Proyecto: proyecto,
			TareaID:  optionalInt64PtrArg(args, "tarea_id"),
			Contexto: optionalStringArg(args, "contexto"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(resolucion), resolucion, false), nil

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

	case "orquesta.asignaciones.activar":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		proyecto, err := requiredStringArg(args, "proyecto")
		if err != nil {
			return nil, err
		}
		item, err := sesionesAPIService.ActivateAssignment(agente, proyecto, optionalStringArg(args, "nota"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(item), item, false), nil

	case "orquesta.worktrees.listar":
		worktrees, err := listarWorktreesFiltradas(optionalStringArg(args, "estado"), optionalStringArg(args, "agente"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(worktrees), worktrees, false), nil

	case "orquesta.worktrees.preparar":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		proyecto, err := requiredStringArg(args, "proyecto")
		if err != nil {
			return nil, err
		}
		var tareaID *int64
		if value := optionalInt64Arg(args, "tarea_id"); value > 0 {
			tareaID = &value
		}
		var lockID *int64
		if value := optionalInt64Arg(args, "lock_id"); value > 0 {
			lockID = &value
		}
		worktree, err := newCoordinationService().PrepareWorktree(coordinacion.PrepareWorktreeInput{
			ProjectRef: proyecto,
			Agent:      agente,
			TaskID:     tareaID,
			LockID:     lockID,
			Name:       optionalStringArg(args, "nombre"),
			Branch:     optionalStringArg(args, "branch"),
			BaseRef:    optionalStringArg(args, "base_ref"),
			Reason:     optionalStringArg(args, "motivo"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(worktree), worktree, false), nil

	case "orquesta.worktrees.cerrar":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, err
		}
		worktree, err := newCoordinationService().CloseWorktree(id, boolArgOrFalse(args, "eliminar"), optionalStringArg(args, "motivo"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(worktree), worktree, false), nil

	case "orquesta.locks.listar":
		locks, err := listarLocksFiltrados(optionalStringArg(args, "estado"), optionalStringArg(args, "agente"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(locks), locks, false), nil

	case "orquesta.locks.adquirir":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		scopeType, err := requiredStringArg(args, "scope_type")
		if err != nil {
			return nil, err
		}
		scopeKey, err := requiredStringArg(args, "scope_key")
		if err != nil {
			return nil, err
		}
		var projectID *int64
		proyectoRef := optionalStringArg(args, "proyecto")
		if strings.TrimSpace(proyectoRef) != "" {
			project, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			projectID = &project.ID
		}
		var taskID *int64
		if value := optionalInt64Arg(args, "tarea_id"); value > 0 {
			taskID = &value
		}
		var sessionID *int64
		if projectID != nil {
			sesion, err := sesionesAPIService.GetActiveSession(agente, proyectoRef)
			if err == nil && sesion != nil {
				sessionID = &sesion.ID
			}
		}
		lock, err := newCoordinationService().AcquireLock(coordinacion.AcquireLockInput{
			ProjectID:    projectID,
			TaskID:       taskID,
			SessionID:    sessionID,
			Agent:        agente,
			ScopeType:    scopeType,
			ScopeKey:     scopeKey,
			Path:         optionalStringArg(args, "ruta"),
			Branch:       optionalStringArg(args, "branch"),
			Reason:       optionalStringArg(args, "motivo"),
			LeaseSeconds: optionalIntArg(args, "lease_seconds"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(lock), lock, false), nil

	case "orquesta.locks.liberar":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, err
		}
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		leaseToken, err := requiredStringArg(args, "lease_token")
		if err != nil {
			return nil, err
		}
		lock, err := newCoordinationService().ReleaseLock(coordinacion.ReleaseLockInput{
			ID:         id,
			Agent:      agente,
			LeaseToken: leaseToken,
			Reason:     optionalStringArg(args, "motivo"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(lock), lock, false), nil

	case "orquesta.sesiones.activas":
		sesiones, err := listarSesionesActivas()
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(sesiones), sesiones, false), nil

	case "orquesta.sesiones.fin":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		sesion, err := sesionesAPIService.Finish(agente)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(sesion), sesion, false), nil

	case "orquesta.agentes.listar":
		agentes, err := agentesService.ListAgents()
		if err != nil {
			return nil, err
		}
		if activos, ok := boolArg(args, "activos"); ok && activos {
			filtrados := make([]*db.Agente, 0, len(agentes))
			for _, agente := range agentes {
				if agente != nil && agente.Activo {
					filtrados = append(filtrados, agente)
				}
			}
			agentes = filtrados
		}
		return toolResult(prettyJSON(agentes), agentes, false), nil

	case "orquesta.agentes.pausar":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		minutos := optionalIntArg(args, "minutos")
		motivo, err := requiredStringArg(args, "motivo")
		if err != nil {
			return nil, err
		}
		if err := agentesService.PauseTemporarily(agente, minutos, motivo, optionalStringArg(args, "accion"), optionalStringArg(args, "entidad"), optionalStringArg(args, "detalle")); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		detail, err := agentesService.GetAgent(agente)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(detail), detail, false), nil

	case "orquesta.agentes.accion":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		accion, err := requiredStringArg(args, "accion")
		if err != nil {
			return nil, err
		}
		if err := agentesService.ApplyStateAction(agente, accion); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		detail, err := agentesService.GetAgent(agente)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(detail), detail, false), nil

	case "orquesta.agentes.control":
		return callMCPAgentLifecycleTool(args, "")

	case "orquesta.agentes.lanzar":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		proyecto, err := requiredStringArg(args, "proyecto")
		if err != nil {
			return nil, err
		}
		out, err := orquestacionAgentesService.Launch(orquestacionagentesapp.LaunchRequest{
			Agente:       agente,
			Proyecto:     proyecto,
			Conector:     optionalStringArg(args, "conector"),
			Modelo:       optionalStringArg(args, "modelo"),
			Razonamiento: optionalStringArg(args, "razonamiento"),
			Perfil:       optionalStringArg(args, "perfil"),
			Motivo:       optionalStringArg(args, "motivo"),
			Por:          optionalStringArg(args, "por"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := apiAgenteLanzarResponse{
			OK:       true,
			OrderID:  out.OrderID,
			Agente:   out.Agente,
			Proyecto: out.Proyecto,
			Conector: out.Conector,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.agentes.start":
		return callMCPAgentLifecycleTool(args, "start")

	case "orquesta.agentes.pause":
		return callMCPAgentLifecycleTool(args, "pause")

	case "orquesta.agentes.resume":
		return callMCPAgentLifecycleTool(args, "resume")

	case "orquesta.agentes.stop":
		return callMCPAgentLifecycleTool(args, "stop")

	case "orquesta.agentes.handoff":
		origen, err := requiredStringArg(args, "agente_origen")
		if err != nil {
			return nil, err
		}
		destino, err := requiredStringArg(args, "agente_destino")
		if err != nil {
			return nil, err
		}
		var tareaID *int64
		if value := optionalInt64Arg(args, "tarea_id"); value > 0 {
			tareaID = &value
		}
		id, err := db.CrearHandoffAgenteVivo(
			origen,
			destino,
			tareaID,
			optionalStringArg(args, "motivo"),
			optionalStringArg(args, "resumen"),
			optionalStringArg(args, "external_session_id"),
		)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{
			"id":             id,
			"order_id":       id,
			"agente_origen":  origen,
			"agente_destino": destino,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.agentes.preparar":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		proyecto, err := requiredStringArg(args, "proyecto")
		if err != nil {
			return nil, err
		}
		out, err := agentesService.BuildPrepare(agentesapp.PrepareInput{
			Agente:       agente,
			Proyecto:     proyecto,
			Conector:     optionalStringArg(args, "conector"),
			Modelo:       optionalStringArg(args, "modelo"),
			Razonamiento: optionalStringArg(args, "razonamiento"),
			Perfil:       optionalStringArg(args, "perfil"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(out), out, false), nil

	case "orquesta.agentes.investigar":
		query, err := requiredStringArg(args, "query")
		if err != nil {
			return nil, err
		}
		out, err := agentesService.Investigate(query, optionalStringArg(args, "proyecto"), intArgOrDefault(args, "limit", 25))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(out), out, false), nil

	case "orquesta.agentes.registrar":
		nombre := strings.TrimSpace(optionalStringArg(args, "nombre"))
		rol := strings.TrimSpace(optionalStringArg(args, "rol"))
		if rol == "" {
			rol = "programador"
		}
		proveedor := strings.TrimSpace(optionalStringArg(args, "proveedor"))
		if nombre == "" {
			if proveedor == "" {
				return toolResult("nombre o proveedor requerido", nil, true), nil
			}
			nombreAuto, err := agentesService.RegisterAgentAuto(proveedor, rol)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			nombre = nombreAuto
		} else if err := agentesService.RegisterAgent(nombre, rol); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		invalidateAgentStatusCaches()
		out := map[string]any{"ok": true, "nombre": nombre, "rol": rol}
		return toolResult(prettyJSON(out), out, false), nil

	case "orquesta.agentes.overview":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		detail, err := agentesService.BuildDetailCompact(agente)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		if strings.EqualFold(optionalStringArg(args, "schema"), "canonical") {
			out := apiAgenteOverviewResponse{
				Agent:  buildCanonicalAgentOverview(detail),
				Detail: detail,
			}
			return toolResult(prettyJSON(out), out, false), nil
		}
		return toolResult(prettyJSON(detail), detail, false), nil

	case "orquesta.agentes.panel":
		rows, err := fetchAgentPanelRowsCached(apiAgentsPanelTimeout)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		now := time.Now().UTC()
		out := apiAgentesPanelCanonicalResponse{
			Agents: agentesService.BuildPanelEntities(normalizeAgentPanelRowsForAPI(rows, now), now),
		}
		return toolResult(prettyJSON(out), out, false), nil

	case "orquesta.agentes.actividad":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		since, err := parseStatsSince(optionalStringArg(args, "desde"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		report, err := buildAgentActivityReportLocal(
			agente,
			optionalStringArg(args, "proyecto"),
			since,
			intArgOrDefault(args, "audit_limit", 200),
			intArgOrDefault(args, "transcript_limit", 400),
		)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(report), report, false), nil

	case "orquesta.runtime.handles.listar":
		agente := optionalStringArg(args, "agente")
		handles, err := runtimesService.ListRuntimeHandles(stringPtrOrNil(agente))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(handles), handles, false), nil

	case "orquesta.runtimes.tree":
		activos := boolPtrArg(args, "activos")
		resp, err := mcpRuntimeTreeResponse(
			optionalStringArg(args, "agente"),
			optionalStringArg(args, "proyecto"),
			activos,
		)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(resp), resp, false), nil

	case "orquesta.runtime.events.listar":
		level := optionalStringArg(args, "level")
		filter := db.FiltroRuntimeEvents{
			Agente: stringPtrOrNil(optionalStringArg(args, "agente")),
			Kind:   stringPtrOrNil(optionalStringArg(args, "kind")),
			Limit:  intArgOrDefault(args, "limit", 50),
		}
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			filter.ProyectoID = &p.ID
		}
		events, err := runtimesService.ListRuntimeEvents(filter)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(level) != "" {
			filtered := make([]*db.RuntimeEvent, 0, len(events))
			for _, item := range events {
				if item != nil && strings.EqualFold(strings.TrimSpace(item.Level), strings.TrimSpace(level)) {
					filtered = append(filtered, item)
				}
			}
			events = filtered
		}
		events = compactarRuntimeEventsAPI(events)
		return toolResult(prettyJSON(events), events, false), nil

	case "orquesta.runtime.trace":
		maxBytes := 0
		if raw, ok := args["bytes"]; ok && raw != nil {
			value := optionalIntArg(args, "bytes")
			if value <= 0 {
				return toolResult("bytes inválido", nil, true), nil
			}
			maxBytes = value
		}
		resp, err := mcpRuntimeTraceResponse(
			optionalInt64Arg(args, "handle_id"),
			optionalStringArg(args, "agente"),
			optionalStringArg(args, "proyecto"),
			maxBytes,
		)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(resp), resp, false), nil

	case "orquesta.runtime.transcript.listar":
		filter := db.FiltroRuntimeTranscript{
			Agente:          stringPtrOrNil(optionalStringArg(args, "agente")),
			Stream:          stringPtrOrNil(optionalStringArg(args, "stream")),
			Classification:  stringPtrOrNil(optionalStringArg(args, "classification")),
			Query:           stringPtrOrNil(optionalStringArg(args, "q")),
			SoloSenalesPend: boolArgOrFalse(args, "signals_pending"),
			Limit:           intArgOrDefault(args, "limit", 50),
		}
		if sinceRaw := strings.TrimSpace(optionalStringArg(args, "desde")); sinceRaw != "" {
			since, err := apiParseRFC3339QueryTime(sinceRaw)
			if err != nil {
				return toolResult("desde inválido", nil, true), nil
			}
			filter.Desde = since
		}
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			filter.ProyectoID = &p.ID
		}
		if runtimeID := optionalInt64Arg(args, "runtime_id"); runtimeID > 0 {
			filter.RuntimeID = &runtimeID
		}
		if handleID := optionalInt64Arg(args, "handle_id"); handleID > 0 {
			filter.HandleID = &handleID
		}
		items, err := apiListRuntimeTranscriptFn(filter)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(items), items, false), nil

	case "orquesta.runtime.ordenes.listar":
		filter := db.FiltroRuntimeOrders{
			Agente: stringPtrOrNil(optionalStringArg(args, "agente")),
			Estado: stringPtrOrNil(optionalStringArg(args, "estado")),
			Limit:  intArgOrDefault(args, "limit", 50),
		}
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			filter.ProyectoID = &p.ID
		}
		orders, err := runtimesService.ListRuntimeOrders(filter)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(orders), orders, false), nil

	case "orquesta.runtime.mailbox.listar":
		filter := db.FiltroRuntimeMailbox{
			ToAgente:   stringPtrOrNil(optionalStringArg(args, "to_agente")),
			FromAgente: stringPtrOrNil(optionalStringArg(args, "from_agente")),
			Estado:     stringPtrOrNil(optionalStringArg(args, "estado")),
		}
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			filter.ProyectoID = &p.ID
		}
		mailbox, err := runtimesService.ListRuntimeMailbox(filter)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(mailbox), mailbox, false), nil

	case "orquesta.runtime.mailbox.enviar":
		fromAgente, err := requiredStringArg(args, "from_agente")
		if err != nil {
			return nil, err
		}
		toAgente, err := requiredStringArg(args, "to_agente")
		if err != nil {
			return nil, err
		}
		kind, err := requiredStringArg(args, "kind")
		if err != nil {
			return nil, err
		}
		payloadJSON := strings.TrimSpace(optionalStringArg(args, "payload_json"))
		if payloadJSON == "" {
			payloadJSON = "{}"
		}
		var proyectoID *int64
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			proyectoID = &p.ID
		}
		var runtimeOrderID *int64
		if value := optionalInt64Arg(args, "runtime_order_id"); value > 0 {
			runtimeOrderID = &value
		}
		id, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:     fromAgente,
			ToAgente:       toAgente,
			ProyectoID:     proyectoID,
			RuntimeOrderID: runtimeOrderID,
			Kind:           kind,
			PayloadJSON:    payloadJSON,
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{"id": id, "from_agente": fromAgente, "to_agente": toAgente, "kind": kind}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.mailbox.entregar":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, err
		}
		if err := runtimesService.MarkRuntimeMailboxDelivered(id); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{"id": id, "estado": "entregado"}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.mailbox.consumir":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, err
		}
		if err := runtimesService.MarkRuntimeMailboxConsumed(id); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{"id": id, "estado": "consumido"}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.handles.cerrar":
		resultado, err := runtimesService.CloseResidualRuntimeHandles(runtimesapp.RuntimeHandleResidualCloseRequest{
			Agente:   apiNombreAgenteCanonico(optionalStringArg(args, "agente")),
			Proyecto: strings.TrimSpace(optionalStringArg(args, "proyecto")),
			Actor:    strings.TrimSpace(optionalStringArg(args, "actor")),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := apiRuntimeHandleResidualCloseResponse{
			OK:     true,
			Closed: resultado.Closed,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.handles.purgar":
		resultado, err := runtimesService.PurgeInactiveRuntimeHandles(runtimesapp.RuntimeHandlePurgeRequest{
			Agente:   apiNombreAgenteCanonico(optionalStringArg(args, "agente")),
			Proyecto: strings.TrimSpace(optionalStringArg(args, "proyecto")),
			Estados:  optionalStringSliceArg(args, "estados"),
			Actor:    strings.TrimSpace(optionalStringArg(args, "actor")),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := apiRuntimeHandlesPurgeResponse{
			OK:         true,
			Deleted:    resultado.Deleted,
			DeletedIDs: resultado.DeletedIDs,
			Estados:    resultado.Estados,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtimes.cerrar":
		resultado, err := runtimesService.CloseResidualRuntimes(runtimesapp.RuntimeResidualCloseRequest{
			Agente:   apiNombreAgenteCanonico(optionalStringArg(args, "agente")),
			Proyecto: strings.TrimSpace(optionalStringArg(args, "proyecto")),
			Actor:    strings.TrimSpace(optionalStringArg(args, "actor")),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := apiRuntimeResidualCloseResponse{
			OK:        true,
			Closed:    resultado.Closed,
			ClosedIDs: resultado.ClosedIDs,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.ordenes.cancelar":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, err
		}
		if err := runtimesService.CancelRuntimeOrder(id, strings.TrimSpace(optionalStringArg(args, "actor")), strings.TrimSpace(optionalStringArg(args, "motivo"))); err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{"ok": true, "id": id}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.ordenes.purgar":
		resultado, err := runtimesService.PurgeTerminalRuntimeOrders(runtimesapp.RuntimeOrderPurgeRequest{
			Agente:           apiNombreAgenteCanonico(optionalStringArg(args, "agente")),
			Proyecto:         strings.TrimSpace(optionalStringArg(args, "proyecto")),
			Estados:          optionalStringSliceArg(args, "estados"),
			Tipos:            optionalStringSliceArg(args, "tipos"),
			OlderThanMinutes: optionalIntArg(args, "older_than_minutes"),
			Actor:            strings.TrimSpace(optionalStringArg(args, "actor")),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := apiRuntimeOrdersPurgeResponse{
			OK:         true,
			Deleted:    resultado.Deleted,
			DeletedIDs: resultado.DeletedIDs,
			Estados:    resultado.Estados,
			Tipos:      resultado.Tipos,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.mailbox.limpiar":
		filter := db.FiltroRuntimeMailbox{}
		if toAgente := strings.TrimSpace(optionalStringArg(args, "to_agente")); toAgente != "" {
			toAgente = apiNombreAgenteCanonico(toAgente)
			filter.ToAgente = &toAgente
		}
		if fromAgente := strings.TrimSpace(optionalStringArg(args, "from_agente")); fromAgente != "" {
			fromAgente = apiNombreAgenteCanonico(fromAgente)
			filter.FromAgente = &fromAgente
		}
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			filter.ProyectoID = &p.ID
		}
		estados := normalizarSliceFlags(optionalStringSliceArg(args, "estados"))
		if len(estados) == 0 {
			estados = []string{"pendiente"}
		}
		kinds := normalizarSliceFlags(optionalStringSliceArg(args, "kinds"))
		ids := make([]int64, 0)
		for _, estado := range estados {
			estadoLocal := estado
			filter.Estado = &estadoLocal
			items, err := runtimesService.ListRuntimeMailbox(filter)
			if err != nil {
				return nil, err
			}
			for _, item := range items {
				if item == nil || item.ID <= 0 {
					continue
				}
				if len(kinds) > 0 {
					match := false
					for _, kind := range kinds {
						if strings.EqualFold(strings.TrimSpace(item.Kind), kind) {
							match = true
							break
						}
					}
					if !match {
						continue
					}
				}
				if err := runtimesService.MarkRuntimeMailboxConsumed(item.ID); err != nil {
					return toolResult(err.Error(), nil, true), nil
				}
				ids = append(ids, item.ID)
			}
		}
		result := apiRuntimeMailboxClearResponse{
			OK:         true,
			Cleared:    len(ids),
			ClearedIDs: ids,
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.wake":
		result := apiRuntimeWakeResponse{}
		if boolArgOrFalse(args, "orders") {
			result.Orders = wakeControlPlaneRuntimeOrders()
		}
		if boolArgOrFalse(args, "mailbox") {
			result.Mailbox = wakeControlPlaneRuntimeMailbox()
		}
		if boolArgOrFalse(args, "warm") {
			result.Warm = wakeControlPlaneWarm()
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.process.orders":
		count, err := apiRuntimeProcessOrdersExecutor()
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := apiRuntimeProcessOrdersResponse{OK: true, Count: count}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.process.transcript":
		if handleID := optionalInt64Arg(args, "handle_id"); handleID > 0 {
			count, err := db.IngestarRuntimeTranscriptHandle(handleID)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: count}
			return toolResult(prettyJSON(result), result, false), nil
		}
		if agente := strings.TrimSpace(optionalStringArg(args, "agente")); agente != "" {
			agente = apiNombreAgenteCanonico(agente)
			var (
				handle     *db.RuntimeHandle
				err        error
				proyectoID *int64
			)
			if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
				p, err := apiRuntimeProcessMailboxProjectFn(proyecto)
				if err != nil {
					return toolResult(err.Error(), nil, true), nil
				}
				proyectoID = &p.ID
				handle, err = db.GetRuntimeHandleCanonicoRecienteAgenteProyecto(agente, proyectoID)
			} else {
				handle, err = db.GetRuntimeHandleCanonicoRecienteAgente(agente)
			}
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			if handle == nil || handle.ID <= 0 {
				result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: 0}
				return toolResult(prettyJSON(result), result, false), nil
			}
			count, err := db.IngestarRuntimeTranscriptHandle(handle.ID)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: count}
			return toolResult(prettyJSON(result), result, false), nil
		}
		if boolArgOrFalse(args, "wait") {
			count, err := runtimeProcessTranscriptBatch()
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: count}
			return toolResult(prettyJSON(result), result, false), nil
		}
		started := runtimeProcessTranscriptEnCurso.CompareAndSwap(false, true)
		if started {
			go func() {
				defer runtimeProcessTranscriptEnCurso.Store(false)
				if _, err := runtimeProcessTranscriptBatch(); err != nil {
					db.Audit("server", "runtime_process_transcript_error", "runtime", 0, err.Error())
				}
			}()
		}
		result := apiRuntimeProcessAutonomiaResponse{
			OK:       true,
			Count:    0,
			Accepted: started,
			Running:  runtimeProcessTranscriptEnCurso.Load(),
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.process.mailbox":
		estado := "pendiente"
		filter := db.FiltroRuntimeMailbox{Estado: &estado}
		if toAgente := strings.TrimSpace(optionalStringArg(args, "to_agente")); toAgente != "" {
			toAgente = apiNombreAgenteCanonico(toAgente)
			filter.ToAgente = &toAgente
		}
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := apiRuntimeProcessMailboxProjectFn(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			filter.ProyectoID = &p.ID
		}
		count, err := apiRuntimeProcessMailboxExecutor(filter)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := apiRuntimeProcessMailboxResponse{OK: true, Count: count}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.process.autonomia":
		if boolArgOrFalse(args, "wait") {
			count, err := runtimeProcessAutonomiaBatch()
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: count}
			return toolResult(prettyJSON(result), result, false), nil
		}
		started := runtimeProcessAutonomiaEnCurso.CompareAndSwap(false, true)
		if started {
			go func() {
				defer runtimeProcessAutonomiaEnCurso.Store(false)
				if _, err := runtimeProcessAutonomiaBatch(); err != nil {
					db.Audit("server", "runtime_process_autonomia_error", "runtime", 0, err.Error())
				}
			}()
		}
		result := apiRuntimeProcessAutonomiaResponse{
			OK:       true,
			Count:    0,
			Accepted: started,
			Running:  runtimeProcessAutonomiaEnCurso.Load(),
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.process.degradados":
		if boolArgOrFalse(args, "wait") {
			started := runtimeProcessDegradadosEnCurso.CompareAndSwap(false, true)
			if !started {
				result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: 0, Accepted: false, Running: true}
				return toolResult(prettyJSON(result), result, false), nil
			}
			defer runtimeProcessDegradadosEnCurso.Store(false)
			summary, err := runtimeProcessDegradadosBatchDetailed()
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			result := apiRuntimeProcessAutonomiaResponse{
				OK:                        true,
				Count:                     summary.Count,
				GhostAssignmentsCompacted: summary.GhostAssignmentsCompacted,
				ReactivatedWithoutRuntime: summary.ReactivatedWithoutRuntime,
				IdleAutoassigned:          summary.IdleAutoassigned,
			}
			return toolResult(prettyJSON(result), result, false), nil
		}
		started := runtimeProcessDegradadosEnCurso.CompareAndSwap(false, true)
		if started {
			go func() {
				defer runtimeProcessDegradadosEnCurso.Store(false)
				if _, err := runtimeProcessDegradadosBatch(); err != nil {
					db.Audit("server", "runtime_process_degradados_error", "runtime", 0, err.Error())
				}
			}()
		}
		result := apiRuntimeProcessAutonomiaResponse{
			OK:       true,
			Count:    0,
			Accepted: started,
			Running:  runtimeProcessDegradadosEnCurso.Load(),
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.process.hygiene":
		if boolArgOrFalse(args, "wait") {
			count, err := runtimeProcessHygieneBatch()
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: count}
			return toolResult(prettyJSON(result), result, false), nil
		}
		go func() {
			if _, err := runtimeProcessHygieneBatch(); err != nil {
				db.Audit("server", "runtime_process_hygiene_error", "runtime", 0, err.Error())
			}
		}()
		result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: 0, Accepted: true, Running: true}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.process.reanimaciones":
		if boolArgOrFalse(args, "wait") {
			result := runtimeProcessReanimationsBatchFn()
			return toolResult(prettyJSON(result), result, false), nil
		}
		started := runtimeProcessReanimacionesEnCurso.CompareAndSwap(false, true)
		if started {
			go func() {
				defer runtimeProcessReanimacionesEnCurso.Store(false)
				resp := runtimeProcessReanimationsBatchFn()
				if resp.Errors > 0 {
					db.Audit("server", "runtime_process_reanimaciones_background_errors", "runtime", 0,
						fmt.Sprintf("candidates=%d reactivated=%d cooldown_sustained=%d capacity_blocked=%d errors=%d", resp.Candidates, resp.Reactivated, resp.CooldownSustained, resp.CapacityBlocked, resp.Errors))
				}
			}()
		}
		result := apiRuntimeProcessReanimationsResponse{
			OK:       true,
			Count:    0,
			Accepted: started,
			Running:  runtimeProcessReanimacionesEnCurso.Load(),
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.transcript.purgar_ruido":
		total := 0
		maxBatches := 20
		purgeFn := apiRuntimePurgeTranscriptNoiseExecutor
		if boolArgOrFalse(args, "all") {
			purgeFn = apiRuntimePurgeTranscriptNoiseAllExecutor
			maxBatches = 1
		}
		for i := 0; i < maxBatches; i++ {
			count, err := purgeFn()
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			total += count
			if !boolArgOrFalse(args, "all") || count == 0 {
				break
			}
		}
		result := apiRuntimeProcessAutonomiaResponse{OK: true, Count: total}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.checkpoints.crear":
		req := &db.RuntimeCheckpoint{
			Agente:         strings.TrimSpace(optionalStringArg(args, "agente")),
			CheckpointKind: strings.TrimSpace(optionalStringArg(args, "checkpoint_kind")),
			Resumen:        strings.TrimSpace(optionalStringArg(args, "resumen")),
			Branch:         strings.TrimSpace(optionalStringArg(args, "branch")),
			CWD:            strings.TrimSpace(optionalStringArg(args, "cwd")),
			PayloadJSON:    strings.TrimSpace(optionalStringArg(args, "payload")),
			ResumeStrategy: strings.TrimSpace(optionalStringArg(args, "resume_strategy")),
			Source:         strings.TrimSpace(optionalStringArg(args, "source")),
		}
		if req.PayloadJSON == "" {
			req.PayloadJSON = "{}"
		}
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			req.ProyectoID = &p.ID
		}
		if sesionID := optionalInt64Arg(args, "sesion_id"); sesionID > 0 {
			req.SesionID = &sesionID
		}
		if runtimeID := optionalInt64Arg(args, "runtime_id"); runtimeID > 0 {
			req.RuntimeID = &runtimeID
		}
		id, err := runtimesService.CreateRuntimeCheckpoint(req)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{"ok": true, "id": id}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.runtime.checkpoints.listar":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		filter := db.FiltroRuntimeCheckpoints{Agente: &agente}
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			filter.ProyectoID = &p.ID
		}
		if checkpointKind := strings.TrimSpace(optionalStringArg(args, "kind")); checkpointKind != "" {
			filter.CheckpointKind = &checkpointKind
		}
		if source := strings.TrimSpace(optionalStringArg(args, "source")); source != "" {
			filter.Source = &source
		}
		if limit := optionalIntArg(args, "limit"); limit > 0 {
			filter.Limit = limit
		}
		checkpoints, err := runtimesService.ListRuntimeCheckpoints(filter)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(checkpoints), checkpoints, false), nil

	case "orquesta.runtime.checkpoints.latest":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		var proyectoID *int64
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			proyectoID = &p.ID
		}
		checkpoint, err := runtimesService.LatestRuntimeCheckpoint(agente, proyectoID)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(checkpoint), checkpoint, false), nil

	case "orquesta.agentes.tick":
		agente, err := requiredStringArg(args, "agente")
		if err != nil {
			return nil, err
		}
		proyecto, err := requiredStringArg(args, "proyecto")
		if err != nil {
			return nil, err
		}
		out, err := agentesService.ProcessTick(agentesapp.TickInput{
			Agente:              agente,
			Proyecto:            proyecto,
			Host:                optionalStringArg(args, "host"),
			PID:                 optionalInt64Arg(args, "pid"),
			CuotaPct:            optionalIntArg(args, "cuota_pct"),
			Finalizado:          boolArgOrFalse(args, "finalizado"),
			Motivo:              optionalStringArg(args, "motivo"),
			AckStartOrderID:     optionalInt64Arg(args, "ack_start_order_id"),
			AckBootstrapOrderID: optionalInt64Arg(args, "ack_bootstrap_order_id"),
			AckMailboxIDs:       optionalInt64SliceArg(args, "ack_mailbox_ids"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(out), out, false), nil

	case "orquesta.runtime.nudge":
		toAgente, err := requiredStringArg(args, "to_agente")
		if err != nil {
			return nil, err
		}
		texto, err := requiredStringArg(args, "texto")
		if err != nil {
			return nil, err
		}
		kind := optionalStringArg(args, "kind")
		if strings.TrimSpace(kind) == "" {
			kind = "nudge"
		}
		payload := map[string]any{
			"from_agente": optionalStringArg(args, "from_agente"),
			"to_agente":   toAgente,
			"kind":        kind,
			"texto":       texto,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("serializando payload runtime.nudge: %w", err)
		}
		var proyectoID *int64
		if proyecto := strings.TrimSpace(optionalStringArg(args, "proyecto")); proyecto != "" {
			p, err := runtimesService.GetProject(proyecto)
			if err != nil {
				return toolResult(err.Error(), nil, true), nil
			}
			proyectoID = &p.ID
		}
		id, err := runtimesService.EnqueueRuntimeOrder(&db.RuntimeOrder{
			Agente:      toAgente,
			ProyectoID:  proyectoID,
			Tipo:        "nudge",
			PayloadJSON: string(payloadJSON),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		result := map[string]any{"id": id, "to_agente": toAgente, "kind": kind, "texto": texto}
		return toolResult(prettyJSON(result), result, false), nil

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

	case "orquesta.review_gates.listar":
		gates, err := reviewService.List(reviewapp.ListInput{
			ProyectoRef:    optionalStringArg(args, "proyecto"),
			Estado:         optionalStringArg(args, "estado"),
			ReviewerAgente: optionalStringArg(args, "reviewer_agente"),
			Limit:          intArgOrDefault(args, "limit", 50),
		})
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(gates), gates, false), nil

	case "orquesta.review_gates.resolver":
		id, err := requiredInt64Arg(args, "id")
		if err != nil {
			return nil, err
		}
		estado, err := requiredStringArg(args, "estado")
		if err != nil {
			return nil, err
		}
		gate, err := reviewService.Resolve(reviewapp.ResolveGateInput{
			ID:             id,
			Estado:         estado,
			ReviewerAgente: optionalStringArg(args, "reviewer_agente"),
			FindingsJSON:   optionalStringArg(args, "findings_json"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(gate), gate, false), nil

	case "orquesta.git.merges.listar":
		merges, err := gitgobernanza.NewService(gitgobernanza.Repository{}).ListRequests(optionalStringArg(args, "proyecto"), optionalStringArg(args, "estado"))
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(merges), merges, false), nil

	case "orquesta.git.stats":
		since, err := parseStatsSince(optionalStringArg(args, "desde"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		resp, err := mcpGitStatsResponse(
			optionalStringArg(args, "proyecto"),
			optionalStringArg(args, "path"),
			optionalStringArg(args, "cwd"),
			since,
		)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(resp), resp, false), nil

	case "orquesta.git.merges.guardar":
		proyecto, err := requiredStringArg(args, "proyecto")
		if err != nil {
			return nil, err
		}
		sourceBranch, err := requiredStringArg(args, "source_branch")
		if err != nil {
			return nil, err
		}
		targetBranch, err := requiredStringArg(args, "target_branch")
		if err != nil {
			return nil, err
		}
		requestedBy, err := requiredStringArg(args, "requested_by")
		if err != nil {
			return nil, err
		}
		var mergeID int64
		if raw, ok := args["id"]; ok && raw != nil {
			value, err := requiredInt64Arg(args, "id")
			if err != nil {
				return nil, err
			}
			mergeID = value
		}
		id, err := gitgobernanza.NewService(gitgobernanza.Repository{}).SaveRequest(gitgobernanza.SaveMergeRequestInput{
			ID:           mergeID,
			ProyectoSlug: proyecto,
			SourceBranch: sourceBranch,
			TargetBranch: targetBranch,
			RequestedBy:  requestedBy,
			Estado:       optionalStringArg(args, "estado"),
			CommitOrigen: optionalStringArg(args, "commit_origen"),
			CommitMerge:  optionalStringArg(args, "commit_merge"),
			Notas:        optionalStringArg(args, "notas"),
			MetadataJSON: optionalStringArg(args, "metadata_json"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		merge, err := gitgobernanza.NewService(gitgobernanza.Repository{}).GetRequest(id)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(merge), merge, false), nil

	case "orquesta.supervision.revision":
		supervisor := optionalStringArg(args, "supervisor")
		resumen, err := buildSupervisorRevisionSnapshot(supervisor)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(resumen), resumen, false), nil

	case "orquesta.supervision.eventos":
		return callMCPSupervisionEvents(args)

	case "orquesta.supervision.threads.registrar":
		item, err := registrarSupervisorThreadLigero(
			optionalStringArg(args, "supervisor"),
			optionalStringArg(args, "proyecto"),
			optionalStringArg(args, "session_id"),
			optionalStringArg(args, "thread_id"),
			optionalStringArg(args, "kind"),
			optionalStringArg(args, "mode"),
			optionalStringArg(args, "status"),
			optionalStringArg(args, "source"),
			optionalStringArg(args, "turn_id"),
		)
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(item), item, false), nil

	case "orquesta.supervision.threads.listar":
		resumen, err := buildSupervisorThreadsSnapshot(optionalStringArg(args, "supervisor"), optionalStringArg(args, "session_id"), 100)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(resumen), resumen, false), nil

	case "orquesta.supervision.pipeline.actualizar":
		currentTaskID := optionalInt64PtrArg(args, "current_task_id")
		currentGateID := optionalInt64PtrArg(args, "current_gate_id")
		currentMergeID := optionalInt64PtrArg(args, "current_merge_id")
		item, err := db.UpsertSupervisorPipelineState(db.UpsertSupervisorPipelineStateInput{
			Supervisor:     resolveSupervisorName(optionalStringArg(args, "supervisor")),
			ProyectoSlug:   optionalStringArg(args, "proyecto"),
			PipelineName:   optionalStringArg(args, "pipeline_name"),
			CurrentPhase:   optionalStringArg(args, "current_phase"),
			Status:         optionalStringArg(args, "status"),
			CurrentTaskID:  currentTaskID,
			CurrentGateID:  currentGateID,
			CurrentMergeID: currentMergeID,
			ArtifactsJSON:  optionalStringArg(args, "artifacts_json"),
			MetadataJSON:   optionalStringArg(args, "metadata_json"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(item), item, false), nil

	case "orquesta.supervision.pipeline.listar":
		resumen, err := buildSupervisorPipelineSnapshot(optionalStringArg(args, "supervisor"), optionalStringArg(args, "proyecto"), 20)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(resumen), resumen, false), nil

	case "orquesta.supervision.subagentes.registrar":
		item, err := db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
			Supervisor:      resolveSupervisorName(optionalStringArg(args, "supervisor")),
			ProyectoSlug:    optionalStringArg(args, "proyecto"),
			SessionID:       optionalStringArg(args, "session_id"),
			ParentThreadID:  optionalStringArg(args, "parent_thread_id"),
			ThreadID:        optionalStringArg(args, "thread_id"),
			SubagentName:    optionalStringArg(args, "subagent_name"),
			SubagentType:    optionalStringArg(args, "subagent_type"),
			ToolProfileJSON: optionalStringArg(args, "tool_profile_json"),
			Status:          optionalStringArg(args, "status"),
			ManifestPath:    optionalStringArg(args, "manifest_path"),
			OutputPath:      optionalStringArg(args, "output_path"),
			ErrorMessage:    optionalStringArg(args, "error_message"),
			MetadataJSON:    optionalStringArg(args, "metadata_json"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(item), item, false), nil

	case "orquesta.supervision.subagentes.listar":
		resumen, err := buildSupervisorSubagentsSnapshot(
			optionalStringArg(args, "supervisor"),
			optionalStringArg(args, "proyecto"),
			optionalStringArg(args, "session_id"),
			100,
		)
		if err != nil {
			return nil, err
		}
		return toolResult(prettyJSON(resumen), resumen, false), nil

	case "orquesta.supervision.subagentes.refrescar_store":
		result, err := refreshSupervisorSubagentsFromStore(optionalStringArg(args, "supervisor"), optionalStringArg(args, "proyecto"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.supervision.subagentes.lanzar_externo":
		result, err := launchClaudeSubagentExternal(supervisorSubagentLaunchRequest{
			Supervisor:   optionalStringArg(args, "supervisor"),
			Proyecto:     optionalStringArg(args, "proyecto"),
			Name:         optionalStringArg(args, "name"),
			Description:  optionalStringArg(args, "description"),
			Prompt:       optionalStringArg(args, "prompt"),
			SubagentType: optionalStringArg(args, "subagent_type"),
			Model:        optionalStringArg(args, "model"),
		})
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.supervision.acciones.aplicar":
		result, err := applySupervisorRecommendedAction(optionalStringArg(args, "supervisor"), optionalStringArg(args, "action"), optionalStringArg(args, "target"), optionalStringArg(args, "assignee"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.supervision.acciones.aplicar_lote":
		result, err := applySupervisorRecommendedActionsBatch(optionalStringArg(args, "supervisor"), optionalIntArg(args, "max_items"), optionalStringArg(args, "kind"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(result), result, false), nil

	case "orquesta.supervision.acciones.aplicar_siguiente":
		result, err := applySupervisorNextAction(optionalStringArg(args, "supervisor"))
		if err != nil {
			return toolResult(err.Error(), nil, true), nil
		}
		return toolResult(prettyJSON(result), result, false), nil
	}

	return nil, fmt.Errorf("tool no soportada: %s", name)
}

type estadoResumen struct {
	Generado            string                              `json:"generado"`
	Agentes             []*db.Agente                        `json:"agentes,omitempty"`
	TareasPorEstado     map[string]int                      `json:"tareasPorEstado"`
	AgentesActivos      []*db.Agente                        `json:"agentesActivos"`
	AgentesTrabajando   []*db.Agente                        `json:"agentesTrabajando,omitempty"`
	AgentesSaturados    []*db.Agente                        `json:"agentesSaturados,omitempty"`
	AgentesAtascados    []*db.Agente                        `json:"agentesAtascados,omitempty"`
	AgentesAuthManual   []*db.Agente                        `json:"agentesAuthManual,omitempty"`
	AgentesQuotaBlocked []*db.Agente                        `json:"agentesQuotaBlocked,omitempty"`
	Proyectos           []map[string]any                    `json:"proyectos"`
	Pools               []map[string]any                    `json:"pools"`
	Asignaciones        []map[string]any                    `json:"asignaciones"`
	PropuestasAbiertas  []propuestaLite                     `json:"propuestasAbiertas"`
	TareasActivas       []tareaLite                         `json:"tareasActivas"`
	TareasEnProgreso    []tareaLite                         `json:"tareasEnProgreso,omitempty"`
	TareasReservadas    []tareaLite                         `json:"tareasReservadas,omitempty"`
	PoolsLocales        []*capacidadapp.PoolLocalCompartido `json:"poolsLocales,omitempty"`
	DeudaDispatch       deudaDispatchResumen                `json:"deudaDispatch,omitempty"`
	Autonomia           autonomiaResumen                    `json:"autonomia,omitempty"`
	AutonomySurface     *autonomySurface                    `json:"autonomySurface,omitempty"`
	WorkersConectados   int                                 `json:"workersConectados,omitempty"`
	WorkersTrabajando   int                                 `json:"workersTrabajando,omitempty"`
	SupervisoresActivos int                                 `json:"supervisoresActivos,omitempty"`
	WorktreesActivas    []map[string]any                    `json:"worktreesActivas"`
	LocksActivos        []map[string]any                    `json:"locksActivos"`
	SesionesActivas     []map[string]any                    `json:"sesionesActivas"`
	Conectores          []map[string]any                    `json:"conectores"`
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
	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
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
	autonomySurface, err := resolveMCPStatusAutonomySurface(status)
	if err != nil {
		return nil, err
	}

	return &estadoResumen{
		Generado:            time.Now().UTC().Format(time.RFC3339),
		Agentes:             status.Agentes,
		TareasPorEstado:     status.TareasPorEstado,
		AgentesActivos:      status.AgentesActivos,
		AgentesTrabajando:   status.AgentesTrabajando,
		AgentesSaturados:    status.AgentesSaturados,
		AgentesAtascados:    status.AgentesAtascados,
		AgentesAuthManual:   status.AgentesAuthManual,
		AgentesQuotaBlocked: status.AgentesQuotaBlocked,
		Proyectos:           proyectos,
		Pools:               pools,
		Asignaciones:        asignaciones,
		PropuestasAbiertas:  status.PropuestasResumen,
		TareasActivas:       status.TareasActivas,
		TareasEnProgreso:    filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaEnProgreso, db.TareaBloqueada),
		TareasReservadas:    filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaAsignada),
		PoolsLocales:        status.PoolsLocales,
		WorktreesActivas:    worktrees,
		LocksActivos:        locks,
		SesionesActivas:     sesiones,
		Conectores:          conectores,
		AutonomySurface:     autonomySurface,
		WorkersConectados:   status.WorkersConectados,
		WorkersTrabajando:   status.WorkersTrabajando,
		SupervisoresActivos: status.SupervisoresActivos,
	}, nil
}

func resolveMCPStatusAutonomySurface(status apiStatusResponse) (*autonomySurface, error) {
	surface, _, _ := normalizeStatusAutonomyPayload(status.AutonomySurface, status.AutonomyHighlights, status.CriticalProjectRisk)
	if surface != nil {
		return surface, nil
	}
	return buildVisibleProjectAutonomySurfaceLocal()
}

func buildEstadoResumenLigero() (*estadoResumen, error) {
	status := degradedAPIStatusResponse()
	switch {
	case func() bool {
		snapshot, ok := readStatusSnapshotFresh()
		if ok {
			status = snapshot
		}
		return ok
	}():
	case func() bool {
		snapshot, ok := fetchStatusReadOnlyLiteDirect(statusFastTimeout)
		if ok {
			status = snapshot
			storeStatusSnapshot(snapshot, statusNowFunc().UTC())
		}
		return ok
	}():
	case func() bool {
		snapshot, ok := readStatusSnapshotAny()
		if ok {
			status = snapshot
			ensureStatusRefreshAsync()
		}
		return ok
	}():
	default:
		ensureStatusRefreshAsync()
	}
	return &estadoResumen{
		Generado:            time.Now().UTC().Format(time.RFC3339),
		Agentes:             status.Agentes,
		TareasPorEstado:     status.TareasPorEstado,
		AgentesActivos:      status.AgentesActivos,
		AgentesTrabajando:   status.AgentesTrabajando,
		AgentesSaturados:    status.AgentesSaturados,
		AgentesAtascados:    status.AgentesAtascados,
		AgentesAuthManual:   status.AgentesAuthManual,
		AgentesQuotaBlocked: status.AgentesQuotaBlocked,
		PropuestasAbiertas:  status.PropuestasResumen,
		TareasActivas:       status.TareasActivas,
		TareasEnProgreso:    filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaEnProgreso, db.TareaBloqueada),
		TareasReservadas:    filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaAsignada),
		PoolsLocales:        status.PoolsLocales,
		WorkersConectados:   status.WorkersConectados,
		WorkersTrabajando:   status.WorkersTrabajando,
		SupervisoresActivos: status.SupervisoresActivos,
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
	b.WriteString("## Doctrina canonica\n")
	fmt.Fprintf(&b, "- Lectura obligatoria: %s\n", db.RutaDoctrinaCanonica)
	b.WriteString("- Si hay conflicto entre documentos, manda esta doctrina y el estado vivo consultado en Orquesta.\n")
	b.WriteString("- Tareas, propuestas, sesiones y runtimes vivos se leen desde Orquesta, no desde documentos estaticos.\n\n")

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

func buildAgentGuidance(agente string) (string, error) {
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
	fmt.Fprintf(&b, "# Guidance canónica de %s\n\n", actual.Nombre)
	b.WriteString("## Role & Intent\n")
	fmt.Fprintf(&b, "- Eres `%s` con rol `%s` dentro de Orquesta.\n", actual.Nombre, strings.TrimSpace(actual.Rol))
	b.WriteString("- Tu objetivo es cerrar trabajo real sin crear rutas paralelas ni segundas fuentes de verdad.\n")
	b.WriteString("- Trabajas sobre el estado vivo del daemon y la doctrina canónica, no sobre recuerdos de conversaciones.\n\n")

	b.WriteString("## Operating Principles\n")
	b.WriteString("- Server-first siempre: API, web, MCP y daemon antes que accesos locales.\n")
	b.WriteString("- No reescribas piezas sanas solo por estilo; mejora solo cuando el cambio añade claridad, fiabilidad o cierre operativo.\n")
	b.WriteString("- Evita solapes de módulo y cambios que abran deriva entre agentes.\n")
	b.WriteString("- Usa pruebas dirigidas y evidencia real antes de afirmar cierre.\n\n")

	b.WriteString("## Execution Protocol\n")
	b.WriteString("- Lee la biblia y luego el estado vivo de Orquesta.\n")
	b.WriteString("- Revisa tarea asignada, módulo, propuesta asociada y riesgo de colisión.\n")
	b.WriteString("- Implementa por la vía canónica del servidor.\n")
	b.WriteString("- Valida con tests dirigidos y, si aplica, verificación viva.\n")
	b.WriteString("- Deja evidencia en tarea/diario antes de dar algo por terminado.\n\n")

	b.WriteString("## Constraints & Safety\n")
	fmt.Fprintf(&b, "- Doctrina obligatoria: %s\n", db.RutaDoctrinaCanonica)
	b.WriteString("- No uses scripts laterales ni SQL manual salvo bloqueo real y diagnóstico explícito.\n")
	b.WriteString("- No inventes contratos nuevos si ya existe uno en API/MCP/daemon.\n")
	b.WriteString("- Si detectas programación cíclica, paras y escalas al supervisor.\n\n")

	b.WriteString("## Verification & Completion\n")
	b.WriteString("- Una tarea no se da por cerrada sin pruebas dirigidas o evidencia operativa suficiente.\n")
	b.WriteString("- Si tocas núcleo, valida también la proyección visible afectada.\n")
	b.WriteString("- Registra notas o hallazgos útiles para el siguiente agente.\n\n")

	b.WriteString("## Recovery & Lifecycle\n")
	if strings.TrimSpace(actual.EstadoSesion) != "" {
		fmt.Fprintf(&b, "- Estado de sesión actual: %s.\n", strings.TrimSpace(actual.EstadoSesion))
	}
	b.WriteString("- Si quedas bloqueado, deja checkpoint útil y prepara handoff claro.\n")
	b.WriteString("- Si el runtime cae por cuota o cooldown, no intentes reanimación manual fuera del control plane.\n")
	b.WriteString("- Si falta contexto, usa briefing, revisión MCP y estado vivo antes de pedirlo fuera.\n")

	return strings.TrimSpace(b.String()) + "\n", nil
}

func buildSupervisorBriefing(supervisor string) (string, error) {
	supervisor = strings.TrimSpace(supervisor)
	if supervisor == "" {
		if cfg, err := db.ConfigGet("openclaw_gateway_operator"); err == nil && strings.TrimSpace(cfg) != "" {
			supervisor = strings.TrimSpace(cfg)
		} else {
			supervisor = "OpenClaw"
		}
	}
	status, err := statusService.FetchStatus()
	if err != nil {
		return "", err
	}
	autonomySurface, err := resolveMCPStatusAutonomySurface(status)
	if err != nil {
		return "", err
	}
	snapshot, err := buildSupervisorReviewSnapshot(supervisor)
	if err != nil {
		return "", err
	}
	globalRiskFocus := supervisorCriticalProjectRiskFromSnapshot(snapshot)

	conectados := status.AgentesActivos
	enCuota := agentesNoActivosConCuota(status.Agentes)
	retenidas := tareasRetenidasPorCuota(status.TareasActivas, status.Agentes, status.AgentesQuotaBlocked)
	workersConectados, workersTrabajando, supervisoresActivos := apiStatusVisibleCounters(status)

	var b strings.Builder
	fmt.Fprintf(&b, "# Briefing de supervisor: %s\n\n", supervisor)
	b.WriteString("## Doctrina canonica\n")
	fmt.Fprintf(&b, "- Lectura obligatoria: %s\n", db.RutaDoctrinaCanonica)
	b.WriteString("- Orquesta es la fuente de verdad para flota, cuota, tareas, handoffs y runtime.\n")
	b.WriteString("- El supervisor no recompone estado desde documentos sueltos ni desde rutas locales; decide sobre el estado vivo del daemon.\n\n")

	fmt.Fprintf(&b, "## Estado de flota\n")
	fmt.Fprintf(&b, "- Conectados y disponibles: %d\n", workersConectados)
	fmt.Fprintf(&b, "- Supervisores activos: %d\n", supervisoresActivos)
	fmt.Fprintf(&b, "- En enfriamiento/cuota: %d\n", len(enCuota))
	fmt.Fprintf(&b, "- Con trabajo activo: %d\n\n", workersTrabajando)

	if len(conectados) > 0 {
		b.WriteString("### Workers disponibles\n")
		for _, agente := range conectados {
			if agente == nil {
				continue
			}
			fmt.Fprintf(&b, "- %s: %s\n", strings.TrimSpace(agente.Nombre), resumenSupervisorAgente(agente))
		}
		b.WriteString("\n")
	}
	if len(enCuota) > 0 {
		b.WriteString("### Workers fuera del pool por cuota\n")
		for _, agente := range enCuota {
			if agente == nil {
				continue
			}
			fmt.Fprintf(&b, "- %s: %s\n", strings.TrimSpace(agente.Nombre), resumenSupervisorAgente(agente))
		}
		b.WriteString("\n")
	}
	if len(status.TareasActivas) > 0 {
		b.WriteString("## Frentes activos\n")
		for _, tarea := range status.TareasActivas {
			fmt.Fprintf(&b, "- #%d [%s] %s", tarea.ID, tarea.Estado, strings.TrimSpace(tarea.Titulo))
			if strings.TrimSpace(tarea.Agente) != "" {
				fmt.Fprintf(&b, " -> %s", strings.TrimSpace(tarea.Agente))
			}
			if strings.TrimSpace(tarea.Modulo) != "" {
				fmt.Fprintf(&b, " (%s)", strings.TrimSpace(tarea.Modulo))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	if len(retenidas) > 0 {
		b.WriteString("## Tareas retenidas por cuota\n")
		for _, tarea := range retenidas {
			fmt.Fprintf(&b, "- #%d %s -> %s\n", tarea.ID, strings.TrimSpace(tarea.Titulo), strings.TrimSpace(tarea.Agente))
		}
		b.WriteString("\n")
	}
	if len(status.PropuestasResumen) > 0 {
		b.WriteString("## Propuestas abiertas\n")
		for _, propuesta := range status.PropuestasResumen {
			fmt.Fprintf(&b, "- %s: %s (✓%d ✗%d ⏳%d)\n", strings.TrimSpace(propuesta.Codigo), strings.TrimSpace(propuesta.Titulo), propuesta.Acuerdo, propuesta.Desacuerdo, propuesta.Pendiente)
		}
		b.WriteString("\n")
	}
	if focus := buildSupervisorCriticalAutonomyFocus(status, autonomySurface, nil); focus != nil {
		b.WriteString("## Frente crítico actual\n")
		if strings.TrimSpace(focus.Project) != "" {
			fmt.Fprintf(&b, "- proyecto=%s · %s -> %s · %s\n\n", strings.TrimSpace(focus.Project), strings.TrimSpace(focus.Target), strings.TrimSpace(focus.Action), strings.TrimSpace(focus.Reason))
		} else {
			fmt.Fprintf(&b, "- %s -> %s · %s\n\n", strings.TrimSpace(focus.Target), strings.TrimSpace(focus.Action), strings.TrimSpace(focus.Reason))
		}
	}
	if globalRiskFocus != nil {
		b.WriteString("## Riesgo global de integración\n")
		fmt.Fprintf(&b, "- Proyecto crítico: %s · integracion_bloqueada=%d\n", strings.TrimSpace(globalRiskFocus.Project), globalRiskFocus.Blocking)
		if len(globalRiskFocus.Highlights) > 0 {
			fmt.Fprintf(&b, "- Señales: %s\n", strings.Join(globalRiskFocus.Highlights, " | "))
		}
		b.WriteString("\n")
	}
	if queueContext := formatSupervisorQueueContextLine(snapshot); queueContext != "" {
		b.WriteString("## Cola canónica del supervisor\n")
		fmt.Fprintf(&b, "- %s\n\n", queueContext)
	}
	if autonomySurface != nil && autonomySurface.Events > 0 {
		b.WriteString("## Autonomía reciente\n")
		fmt.Fprintf(&b, "- %s\n", formatAutonomySurfaceSummary(autonomySurface))
		if len(autonomySurface.Highlights) > 0 {
			fmt.Fprintf(&b, "- Focos canónicos: %s\n", strings.Join(autonomySurface.Highlights, " | "))
		}
		for _, item := range autonomySurface.Recent {
			fmt.Fprintf(&b, "- [%s] %s", strings.TrimSpace(item.Project), strings.TrimSpace(item.Kind))
			if strings.TrimSpace(item.Agent) != "" {
				fmt.Fprintf(&b, " agente=%s", strings.TrimSpace(item.Agent))
			}
			if strings.TrimSpace(item.TargetAgent) != "" {
				fmt.Fprintf(&b, " destino=%s", strings.TrimSpace(item.TargetAgent))
			}
			if strings.TrimSpace(item.Reason) != "" {
				fmt.Fprintf(&b, " · %s", strings.TrimSpace(item.Reason))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	revision, err := buildSupervisorReviewOverviewFromSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(revision) != "" {
		b.WriteString(revision)
		b.WriteString("\n\n")
	}
	b.WriteString("## Instrucciones operativas para el supervisor\n")
	b.WriteString("- Reparte trabajo evitando solapes de modulo cuando haya alternativa.\n")
	b.WriteString("- Prioriza workers con cuota activa y frentes no bloqueados.\n")
	b.WriteString("- Si un worker cae a cuota, no lo reanimes a mano antes de `reanimar_at` y del presupuesto visible.\n")
	b.WriteString("- Integra solo cambios con pruebas dirigidas y sin abrir rutas paralelas.\n")

	return strings.TrimSpace(b.String()) + "\n", nil
}

func buildSupervisorGuidance(supervisor string) (string, error) {
	supervisor = strings.TrimSpace(supervisor)
	if supervisor == "" {
		if cfg, err := db.ConfigGet("openclaw_gateway_operator"); err == nil && strings.TrimSpace(cfg) != "" {
			supervisor = strings.TrimSpace(cfg)
		} else {
			supervisor = "OpenClaw"
		}
	}
	status, err := statusService.FetchStatus()
	if err != nil {
		return "", err
	}
	autonomySurface, err := resolveMCPStatusAutonomySurface(status)
	if err != nil {
		return "", err
	}
	snapshot, err := buildSupervisorRevisionSnapshot(supervisor)
	if err != nil {
		return "", err
	}
	globalRiskFocus := supervisorCriticalProjectRiskFromSnapshot(snapshot)
	workersConectados, workersTrabajando, supervisoresActivos := apiStatusVisibleCounters(status)

	var b strings.Builder
	fmt.Fprintf(&b, "# Guidance canónica del supervisor: %s\n\n", supervisor)
	b.WriteString("## Role & Intent\n")
	b.WriteString("- Eres el supervisor operativo de la flota y del cierre de integración.\n")
	b.WriteString("- Decides reparto, arbitraje, review, handoff e integración sin crear otra fuente de verdad.\n")
	b.WriteString("- OpenClaw consume estado estructurado de Orquesta; no recompone el sistema desde documentos dispersos.\n\n")

	b.WriteString("## Operating Principles\n")
	b.WriteString("- Prioriza trabajo real, no movimiento cosmético de tareas.\n")
	b.WriteString("- No asignes dos agentes al mismo módulo si hay alternativa equivalente.\n")
	b.WriteString("- El presupuesto semanal manda; la ventana corta solo decide operativa si la semanal sigue viva.\n")
	b.WriteString("- Integra por la vía canónica: review gates, merge queue, MCP y web server-first.\n\n")

	b.WriteString("## Execution Protocol\n")
	fmt.Fprintf(&b, "- Flota conectada ahora: %d.\n", workersConectados)
	fmt.Fprintf(&b, "- Flota trabajando ahora: %d.\n", workersTrabajando)
	fmt.Fprintf(&b, "- Supervisores activos ahora: %d.\n", supervisoresActivos)
	if focus := buildSupervisorCriticalAutonomyFocus(status, autonomySurface, nil); focus != nil {
		if strings.TrimSpace(focus.Project) != "" {
			fmt.Fprintf(&b, "- Frente crítico actual: proyecto=%s · %s -> %s. %s.\n", strings.TrimSpace(focus.Project), strings.TrimSpace(focus.Target), strings.TrimSpace(focus.Action), strings.TrimSpace(focus.Reason))
		} else {
			fmt.Fprintf(&b, "- Frente crítico actual: %s -> %s. %s.\n", strings.TrimSpace(focus.Target), strings.TrimSpace(focus.Action), strings.TrimSpace(focus.Reason))
		}
	}
	if globalRiskFocus != nil {
		fmt.Fprintf(&b, "- Proyecto crítico por riesgo: %s · integracion_bloqueada=%d.\n", strings.TrimSpace(globalRiskFocus.Project), globalRiskFocus.Blocking)
		if len(globalRiskFocus.Highlights) > 0 {
			fmt.Fprintf(&b, "- Señales canónicas del proyecto crítico: %s.\n", strings.Join(globalRiskFocus.Highlights, " | "))
		}
	}
	if queueContext := formatSupervisorQueueContextLine(snapshot); queueContext != "" {
		fmt.Fprintf(&b, "- Cola canónica del supervisor: %s.\n", queueContext)
	}
	if autonomySurface != nil && autonomySurface.Events > 0 {
		fmt.Fprintf(&b, "- Autonomía reciente: %s.\n", formatAutonomySurfaceSummary(autonomySurface))
		if len(autonomySurface.Highlights) > 0 {
			fmt.Fprintf(&b, "- Focos canónicos de autonomía: %s.\n", strings.Join(autonomySurface.Highlights, " | "))
		}
	}
	b.WriteString("- Lee briefing, guidance y cola de revisión antes de tomar decisiones.\n")
	b.WriteString("- Usa `next_safe_action`, `safe_action_queue` y `eventos_normalizados` como resumen operativo canónico; `action_queue` queda como vista amplia, no como vía de ejecución.\n")
	b.WriteString("- Si un gate, señal o merge necesita acción, prioriza arbitraje e integración antes que abrir nuevos frentes.\n\n")

	b.WriteString("## Constraints & Safety\n")
	fmt.Fprintf(&b, "- Doctrina obligatoria: %s\n", db.RutaDoctrinaCanonica)
	b.WriteString("- No fuerces reanimaciones manuales durante cooldown/cuota.\n")
	b.WriteString("- No aceptes cambios sin pruebas dirigidas o evidencia suficiente.\n")
	b.WriteString("- No derives semántica distinta en cada cliente; MCP, API y web deben reutilizar el mismo contrato.\n\n")

	b.WriteString("## Verification & Completion\n")
	b.WriteString("- Antes de integrar, revisa colisiones de módulo, review gates abiertos y merges vivos.\n")
	b.WriteString("- Antes de reasignar, verifica cuota visible, trabajo activo y continuidad.\n")
	b.WriteString("- Antes de cerrar un frente, deja evidencia operativa o test dirigida.\n\n")

	b.WriteString("## Recovery & Lifecycle\n")
	b.WriteString("- Si un worker cae, decide entre checkpoint, handoff o enfriamiento, no entre parches locales.\n")
	b.WriteString("- Si aparece deriva o bucle, corta guidance repetitiva y vuelve al control plane.\n")
	b.WriteString("- Si la integración queda bloqueada, usa review queue y eventos normalizados para arbitrar la siguiente acción.\n")

	return strings.TrimSpace(b.String()) + "\n", nil
}

func buildSupervisorWorkspaceRiskFocus() (*workspaceAutonomyProjectSummary, error) {
	report, err := buildWorkspaceControlReport()
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, nil
	}
	top, ok := workspaceTopRiskProject(report.AutonomyProjects)
	if !ok || strings.TrimSpace(top.Project) == "" || top.Blocking <= 0 {
		return nil, nil
	}
	item := top
	item.Project = strings.TrimSpace(item.Project)
	item.Highlights = append([]string(nil), item.Highlights...)
	return &item, nil
}

func buildSupervisorWorkspaceRiskFocusFromActions(actions []supervisorRecommendedAction) (*workspaceAutonomyProjectSummary, error) {
	report, err := buildWorkspaceControlReport()
	if err != nil {
		return nil, err
	}
	if report == nil || len(report.AutonomyProjects) == 0 {
		return nil, nil
	}
	projectsBySlug := make(map[string]workspaceAutonomyProjectSummary, len(report.AutonomyProjects))
	for _, item := range report.AutonomyProjects {
		slug := strings.ToLower(strings.TrimSpace(item.Project))
		if slug == "" || item.Blocking <= 0 {
			continue
		}
		copy := item
		copy.Project = strings.TrimSpace(copy.Project)
		copy.Highlights = append([]string(nil), copy.Highlights...)
		projectsBySlug[slug] = copy
	}
	if len(projectsBySlug) == 0 {
		return nil, nil
	}
	for _, action := range actions {
		if strings.TrimSpace(action.Kind) != "autonomy_event" {
			continue
		}
		if supervisorActionIsHealthyAbsorbed(action) {
			continue
		}
		project := strings.ToLower(strings.TrimSpace(supervisorCriticalAutonomyProjectFromTarget(action)))
		if project == "" {
			continue
		}
		if item, ok := projectsBySlug[project]; ok {
			return &item, nil
		}
	}
	top, ok := workspaceTopRiskProject(report.AutonomyProjects)
	if !ok || strings.TrimSpace(top.Project) == "" || top.Blocking <= 0 {
		return nil, nil
	}
	item := top
	item.Project = strings.TrimSpace(item.Project)
	item.Highlights = append([]string(nil), item.Highlights...)
	return &item, nil
}

func apiStatusVisibleCounters(status apiStatusResponse) (int, int, int) {
	return reconciledVisibleWorkerCounters(
		status.WorkersConectados,
		status.WorkersTrabajando,
		status.SupervisoresActivos,
		status.AgentesActivos,
		status.AgentesTrabajando,
		status.Autonomia,
	)
}

func resumenSupervisorAgente(a *db.Agente) string {
	if a == nil {
		return ""
	}
	partes := make([]string, 0, 6)
	if strings.TrimSpace(a.Rol) != "" {
		partes = append(partes, strings.TrimSpace(a.Rol))
	}
	if a.CuotaRestantePct != nil {
		partes = append(partes, fmt.Sprintf("efectivo %d%%", *a.CuotaRestantePct))
	}
	if strings.TrimSpace(a.PresupuestoVentana) != "" {
		partes = append(partes, "ventana "+strings.TrimSpace(a.PresupuestoVentana))
	}
	if a.ReanimarAt != nil && !a.ReanimarAt.IsZero() && a.ReanimarAt.After(time.Now().UTC()) {
		partes = append(partes, "cooldown hasta "+a.ReanimarAt.Local().Format("2006-01-02 15:04"))
	}
	if cuenta := resumenCuentaCanonica(a.CuentaID, a.CuentaEmail, a.CuentaUsuario); cuenta != "" {
		partes = append(partes, cuenta)
	}
	if strings.TrimSpace(a.EstadoCuota) != "" && strings.TrimSpace(a.EstadoCuota) != "activo" {
		partes = append(partes, "cuota:"+strings.TrimSpace(a.EstadoCuota))
	}
	return strings.Join(partes, " · ")
}

func buildSupervisorReviewOverview() (string, error) {
	openGates, signals, merges, err := listSupervisorReviewOverviewData()
	if err != nil {
		return "", err
	}
	return buildSupervisorReviewOverviewFromData(openGates, signals, merges)
}

func buildSupervisorReviewBriefing(supervisor string) (string, error) {
	supervisor = strings.TrimSpace(supervisor)
	if supervisor == "" {
		if cfg, err := db.ConfigGet("openclaw_gateway_operator"); err == nil && strings.TrimSpace(cfg) != "" {
			supervisor = strings.TrimSpace(cfg)
		} else {
			supervisor = "OpenClaw"
		}
	}
	snapshot, err := buildSupervisorReviewSnapshot(supervisor)
	if err != nil {
		return "", err
	}
	revision, err := buildSupervisorReviewOverviewFromSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# Cola de revisión del supervisor: %s\n\n", supervisor)
	fmt.Fprintf(&b, "- Doctrina: %s\n", db.RutaDoctrinaCanonica)
	b.WriteString("- Usa esta vista para decidir si integras, pides cambios, reasignas reviewer o relanzas trabajo.\n")
	b.WriteString("- No recompongas el estado de revisión desde varias rutas; esta cola ya resume gates y señales recientes del daemon.\n\n")
	if strings.TrimSpace(revision) != "" {
		b.WriteString(revision)
		b.WriteString("\n\n")
	} else {
		b.WriteString("## Revisión e integración\n- Sin gates abiertos ni señales recientes.\n\n")
	}
	conflicts := supervisorModuleConflictsFromSnapshot(snapshot)
	if len(conflicts) == 0 {
		if fallback, err := listarSupervisorModuleConflictsSnapshotSafe(nil); err == nil {
			conflicts = fallback
		}
	}
	if conflictsOverview := buildSupervisorConflictOverview(conflicts); strings.TrimSpace(conflictsOverview) != "" {
		b.WriteString(conflictsOverview)
		b.WriteString("\n\n")
	}
	if queueContext := formatSupervisorQueueContextLine(snapshot); queueContext != "" {
		b.WriteString("## Cola canónica del supervisor\n")
		fmt.Fprintf(&b, "- %s\n", queueContext)
		if risk := supervisorCriticalProjectRiskFromSnapshot(snapshot); risk != nil {
			fmt.Fprintf(&b, "- critical_project_risk=%s(%d)\n", strings.TrimSpace(risk.Project), risk.Blocking)
		}
		b.WriteString("- safe_action_queue refleja solo la cola autoaplicable actual.\n\n")
	}
	b.WriteString("## Criterio de decisión\n")
	b.WriteString("- `approval_request`: arbitra y desbloquea sin abrir espera humana innecesaria.\n")
	b.WriteString("- `waiting_human`: reencuadra y relanza si el bloqueo no es externo real.\n")
	b.WriteString("- `ready_for_review`: inspecciona, decide gate y encamina merge o cambios.\n")
	return strings.TrimSpace(b.String()) + "\n", nil
}

func buildSupervisorConflictOverview(conflicts []supervisorModuleConflict) string {
	lines := make([]string, 0, 8)
	for _, conflict := range conflicts {
		partes := make([]string, 0, len(conflict.Tareas))
		for _, tarea := range conflict.Tareas {
			partes = append(partes, fmt.Sprintf("#%d %s -> %s", tarea.ID, compactMCPLine(strings.TrimSpace(tarea.Titulo), 80), strings.TrimSpace(tarea.Agente)))
		}
		sort.Strings(partes)
		lines = append(lines, fmt.Sprintf("- módulo=%s · %s", conflict.Modulo, strings.Join(partes, " | ")))
	}
	sort.Strings(lines)
	if len(lines) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Riesgos de colisión detectados\n")
	for _, line := range lines {
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String())
}

func supervisorModuleConflictsFromSnapshot(snapshot map[string]any) []supervisorModuleConflict {
	if len(snapshot) == 0 {
		return nil
	}
	if conflicts, ok := snapshot["module_conflicts"].([]supervisorModuleConflict); ok {
		return conflicts
	}
	return nil
}

type supervisorReviewSignal struct {
	Event   *db.RuntimeEvent
	Agent   string
	Project string
}

type supervisorModuleConflict struct {
	Modulo string      `json:"modulo"`
	Tareas []tareaLite `json:"tareas"`
}

type supervisorRecommendedAction struct {
	Kind          string `json:"kind"`
	Target        string `json:"target"`
	Action        string `json:"action"`
	Reason        string `json:"reason"`
	Priority      string `json:"priority"`
	Assignee      string `json:"assignee,omitempty"`
	AutoAplicable bool   `json:"auto_aplicable,omitempty"`
}

func listarSignalsRevisionSupervisor(limit int) ([]*supervisorReviewSignal, error) {
	if limit <= 0 {
		limit = 12
	}
	kinds := []string{"approval_request", "waiting_human", "ready_for_review"}
	runtimeCache := map[int64]*db.RuntimeInstance{}
	items := make([]*supervisorReviewSignal, 0, limit*len(kinds))
	seen := make(map[int64]struct{}, limit*len(kinds))
	for _, kind := range kinds {
		kind := kind
		events, err := db.ListarRuntimeEvents(db.FiltroRuntimeEvents{Kind: &kind, Limit: limit})
		if err != nil {
			return nil, err
		}
		for _, event := range events {
			if event == nil {
				continue
			}
			if _, ok := seen[event.ID]; ok {
				continue
			}
			seen[event.ID] = struct{}{}
			runtime := runtimeCache[event.RuntimeID]
			if runtime == nil {
				runtime, err = db.GetRuntime(event.RuntimeID)
				if err != nil {
					return nil, err
				}
				runtimeCache[event.RuntimeID] = runtime
			}
			item := &supervisorReviewSignal{Event: event}
			if runtime != nil {
				item.Agent = strings.TrimSpace(runtime.Agente)
				item.Project = strings.TrimSpace(runtime.ProyectoSlug)
			}
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Event.CreatedAt.After(items[j].Event.CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func listarMergesRevisionSupervisor(limit int) ([]*db.GitMerge, error) {
	if limit <= 0 {
		limit = 12
	}
	svc := gitgobernanza.NewService(gitgobernanza.Repository{})
	estados := []string{"pendiente", "validando", "aprobado", "ejecutando", "fallido"}
	items := make([]*db.GitMerge, 0, limit*len(estados))
	seen := make(map[int64]struct{}, limit*len(estados))
	for _, estado := range estados {
		rows, err := svc.ListRequestsLimit("", estado, limit)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row == nil {
				continue
			}
			if _, ok := seen[row.ID]; ok {
				continue
			}
			seen[row.ID] = struct{}{}
			items = append(items, adaptarGitMergeGobernanzaDB(row))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func adaptarGitMergeGobernanzaDB(item *gitgobernanza.GitMerge) *db.GitMerge {
	if item == nil {
		return nil
	}
	return &db.GitMerge{
		ID:            item.ID,
		ProyectoID:    item.ProyectoID,
		ProyectoSlug:  item.ProyectoSlug,
		SourceBranch:  item.SourceBranch,
		TargetBranch:  item.TargetBranch,
		RequestedBy:   item.RequestedBy,
		SolicitadoPor: item.SolicitadoPor,
		Estado:        item.Estado,
		SourceCommit:  item.SourceCommit,
		MergeCommit:   item.MergeCommit,
		CommitOrigen:  item.CommitOrigen,
		CommitMerge:   item.CommitMerge,
		Notas:         item.Notas,
		MetadataJSON:  item.MetadataJSON,
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func buildSupervisorReviewSnapshot(supervisor string) (map[string]any, error) {
	status := degradedAPIStatusResponse()
	switch {
	case func() bool {
		snapshot, ok := readStatusSnapshotFresh()
		if ok {
			status = snapshot
		}
		return ok
	}():
	case func() bool {
		snapshot, ok := fetchStatusReadOnlyLiteDirect(statusFastTimeout)
		if ok {
			status = snapshot
			storeStatusSnapshot(snapshot, statusNowFunc().UTC())
		}
		return ok
	}():
	case func() bool {
		snapshot, ok := readStatusSnapshotAny()
		if ok {
			status = snapshot
			ensureStatusRefreshAsync()
		}
		return ok
	}():
	default:
		ensureStatusRefreshAsync()
	}
	return buildSupervisorReviewSnapshotWithStatus(supervisor, status)
}

var (
	supervisorReviewThreadsSnapshotBuilder   = buildSupervisorThreadsSnapshot
	supervisorReviewSubagentsSnapshotBuilder = buildSupervisorSubagentsSnapshot
	supervisorReviewPipelineSnapshotBuilder  = buildSupervisorPipelineSnapshotFromInputs
	supervisorReviewSignalsBuilder           = listarSignalsRevisionSupervisor
	supervisorReviewMergesBuilder            = listarMergesRevisionSupervisor
	supervisorReviewCriticalRiskBuilder      = buildSupervisorWorkspaceRiskFocusFromActions
)

func buildSupervisorReviewSnapshotWithStatus(supervisor string, status apiStatusResponse) (map[string]any, error) {
	supervisor = strings.TrimSpace(supervisor)
	if supervisor == "" {
		if cfg, err := db.ConfigGet("openclaw_gateway_operator"); err == nil && strings.TrimSpace(cfg) != "" {
			supervisor = strings.TrimSpace(cfg)
		} else {
			supervisor = "OpenClaw"
		}
	}
	gates, err := db.ListarReviewGates(db.FiltroReviewGates{Limit: supervisorReviewOpenGatesLimit})
	if err != nil {
		return nil, err
	}
	openGates := make([]*db.ReviewGate, 0, len(gates))
	for _, gate := range gates {
		if gate == nil || gate.Estado == db.ReviewGateAprobado {
			continue
		}
		openGates = append(openGates, gate)
	}
	signalsCh := supervisorReviewOptionalAsync(150*time.Millisecond, func() ([]*supervisorReviewSignal, error) {
		return supervisorReviewSignalsBuilder(supervisorReviewSignalsLimit)
	})
	mergesCh := supervisorReviewOptionalAsync(150*time.Millisecond, func() ([]*db.GitMerge, error) {
		return supervisorReviewMergesBuilder(supervisorReviewMergesLimit)
	})
	conflictsCh := supervisorReviewOptionalAsync(100*time.Millisecond, func() ([]supervisorModuleConflict, error) {
		return supervisorReviewConflictsBuilder(status.TareasActivas)
	})
	mailboxCh := supervisorReviewOptionalAsync(100*time.Millisecond, func() ([]apiOpenClawMailboxLite, error) {
		return supervisorReviewMailboxBuilder(status.Agentes)
	})
	outboxCh := supervisorReviewOptionalAsync(100*time.Millisecond, func() (notificaciones.OutboxSummary, error) {
		return supervisorReviewOutboxBuilder(supervisorReviewOutboxLimit)
	})
	threadsCh := supervisorReviewOptionalAsync(150*time.Millisecond, func() (map[string]any, error) {
		return supervisorReviewThreadsSnapshotBuilder(supervisor, "", 100)
	})
	subagentsCh := supervisorReviewOptionalAsync(150*time.Millisecond, func() (map[string]any, error) {
		return supervisorReviewSubagentsSnapshotBuilder(supervisor, "", "", 100)
	})
	worktreeDriftCh := supervisorReviewOptionalAsync(openClawWorktreeDriftTimeout, func() ([]apiOpenClawWorktreeDrift, error) {
		return openClawWorktreeDriftBuilder(status)
	})

	signals := []*supervisorReviewSignal{}
	if snapshot, ok := supervisorReviewAwait(signalsCh); ok && snapshot != nil {
		signals = snapshot
	}
	merges := []*db.GitMerge{}
	if snapshot, ok := supervisorReviewAwait(mergesCh); ok && snapshot != nil {
		merges = snapshot
	}
	conflicts := []supervisorModuleConflict{}
	if snapshot, ok := supervisorReviewAwait(conflictsCh); ok && snapshot != nil {
		conflicts = snapshot
	}
	mailboxPendiente := []apiOpenClawMailboxLite{}
	if snapshot, ok := supervisorReviewAwait(mailboxCh); ok && snapshot != nil {
		mailboxPendiente = snapshot
	}
	outbox := notificaciones.OutboxSummary{}
	if snapshot, ok := supervisorReviewAwait(outboxCh); ok {
		outbox = snapshot
	}
	normalizedEvents := buildOpenClawNormalizedEventsFromData(openGates, signals, merges, outbox.Recientes, supervisorReviewOutboxLimit)
	threadSessions := map[string]any{}
	if snapshot, ok := supervisorReviewAwait(threadsCh); ok && snapshot != nil {
		threadSessions = snapshot
	}
	subagents := map[string]any{}
	if snapshot, ok := supervisorReviewAwait(subagentsCh); ok && snapshot != nil {
		subagents = snapshot
	}
	pipelineState := map[string]any{}
	if snapshot, timeoutErr := runAPITimeboxed(200*time.Millisecond, func() (map[string]any, error) {
		return supervisorReviewPipelineSnapshotBuilder(supervisor, "", 20, status, openGates, signals, merges, conflicts, mailboxPendiente)
	}, errStatusFetchTimeout); timeoutErr == nil && snapshot != nil {
		pipelineState = snapshot
	}
	worktreeDrift := []apiOpenClawWorktreeDrift(nil)
	if snapshot, ok := supervisorReviewAwait(worktreeDriftCh); ok {
		worktreeDrift = snapshot
	}
	recommended := buildSupervisorRecommendedActions(openGates, signals, merges, conflicts)
	recommended = append(recommended, buildSupervisorWorktreeDriftActions(worktreeDrift)...)
	recommended = append(recommended, buildSupervisorOperationalActionsFast(status, mailboxPendiente)...)
	recommended = append(recommended, buildSupervisorObservedSessionActions(status, threadSessions)...)
	recommended = append(recommended, buildSupervisorSubagentActions(subagents)...)
	recommended = append(recommended, buildSupervisorProposalActions(status.PropuestasResumen)...)
	sortSupervisorRecommendedActions(recommended)
	markSupervisorRecommendedActions(recommended)
	actionQueue := cloneSupervisorRecommendedActions(recommended)
	safeQueue := buildSupervisorSafeActionQueueFast(actionQueue)
	var criticalProjectRisk *workspaceAutonomyProjectSummary
	if snapshot, timeoutErr := runAPITimeboxed(200*time.Millisecond, func() (*workspaceAutonomyProjectSummary, error) {
		return supervisorReviewCriticalRiskBuilder(actionQueue)
	}, errStatusFetchTimeout); timeoutErr == nil {
		criticalProjectRisk = snapshot
	}
	var nextAction any
	if len(actionQueue) > 0 {
		nextAction = actionQueue[0]
	}
	var nextSafeAction any
	if len(safeQueue) > 0 {
		nextSafeAction = safeQueue[0]
	}
	capacitySummary := buildOpenClawCapacitySummary(status.AgentesActivos, status.AgentesTrabajando, status.TareasActivas, status.TareasPorEstado)
	saturatedAgents := buildOpenClawSaturatedAgents(status.AgentesActivos, status.TareasActivas, nil)
	queueSummary := buildOpenClawQueueSummaryFromActions(actionQueue, safeQueue)
	return map[string]any{
		"supervisor":            supervisor,
		"review_gates":          openGates,
		"signals":               signals,
		"merges":                merges,
		"module_conflicts":      conflicts,
		"mailbox_pending":       mailboxPendiente,
		"normalized_events":     normalizedEvents,
		"thread_sessions":       threadSessions,
		"subagents":             subagents,
		"worktree_drift":        worktreeDrift,
		"pipeline_state":        pipelineState,
		"recommended_actions":   recommended,
		"action_queue":          actionQueue,
		"next_action":           nextAction,
		"queue_kind":            "safe",
		"safe_action_queue":     safeQueue,
		"next_safe_action":      nextSafeAction,
		"critical_project_risk": criticalProjectRisk,
		"capacity_summary":      capacitySummary,
		"saturated_agents":      saturatedAgents,
		"queue_summary":         queueSummary,
	}, nil
}

func supervisorQueueKindFromSnapshot(snapshot map[string]any) string {
	if snapshot == nil {
		return ""
	}
	queueKind, _ := snapshot["queue_kind"].(string)
	return strings.TrimSpace(queueKind)
}

func supervisorNextSafeActionFromSnapshot(snapshot map[string]any) *supervisorRecommendedAction {
	if snapshot == nil {
		return nil
	}
	nextSafeAction, ok := parseSupervisorRecommendedActionSnapshotValue(snapshot["next_safe_action"])
	if !ok {
		return nil
	}
	return nextSafeAction
}

func formatSupervisorQueueContextLine(snapshot map[string]any) string {
	queueKind := supervisorQueueKindFromSnapshot(snapshot)
	nextSafeAction := supervisorNextSafeActionFromSnapshot(snapshot)
	if queueKind == "" && nextSafeAction == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if queueKind != "" {
		parts = append(parts, "queue_kind="+queueKind)
	}
	if nextSafeAction != nil {
		parts = append(parts, fmt.Sprintf("next_safe_action=%s -> %s", strings.TrimSpace(nextSafeAction.Target), strings.TrimSpace(nextSafeAction.Action)))
	} else if queueKind != "" {
		parts = append(parts, "next_safe_action=none")
	}
	return strings.Join(parts, " · ")
}

func intArgOrDefault(args map[string]any, name string, fallback int) int {
	raw, ok := args[name]
	if !ok || raw == nil {
		return fallback
	}
	switch v := raw.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case float64:
		if int(v) > 0 {
			return int(v)
		}
	}
	return fallback
}

func listarSupervisorModuleConflicts() ([]supervisorModuleConflict, error) {
	status, err := statusService.FetchStatus()
	if err != nil {
		return nil, err
	}
	return listarSupervisorModuleConflictsFromTasks(status.TareasActivas), nil
}

func listarSupervisorModuleConflictsSnapshotSafe(tareas []tareaLite) ([]supervisorModuleConflict, error) {
	conflicts := listarSupervisorModuleConflictsFromTasks(tareas)
	if len(conflicts) > 0 || len(tareas) > 0 {
		return conflicts, nil
	}
	activa := db.TareaEnProgreso
	asignada := db.TareaAsignada
	tareasActivas, err := db.ListarTareas(db.FiltroTareas{Estado: &activa})
	if err != nil {
		return nil, err
	}
	tareasAsignadas, err := db.ListarTareas(db.FiltroTareas{Estado: &asignada})
	if err != nil {
		return nil, err
	}
	flat := make([]tareaLite, 0, len(tareasActivas)+len(tareasAsignadas))
	appendLite := func(src []*db.Tarea) {
		for _, tarea := range src {
			if tarea == nil {
				continue
			}
			agente := ""
			if tarea.Agente != nil {
				agente = strings.TrimSpace(*tarea.Agente)
			}
			flat = append(flat, tareaLite{
				ID:        tarea.ID,
				Estado:    tarea.Estado,
				Prioridad: tarea.Prioridad,
				Agente:    agente,
				Titulo:    strings.TrimSpace(tarea.Titulo),
				Modulo:    strings.TrimSpace(tarea.Modulo),
			})
		}
	}
	appendLite(tareasActivas)
	appendLite(tareasAsignadas)
	return listarSupervisorModuleConflictsFromTasks(flat), nil
}

func listarSupervisorModuleConflictsFromTasks(tareas []tareaLite) []supervisorModuleConflict {
	moduleGroups := map[string][]tareaLite{}
	for _, tarea := range tareas {
		modulo := strings.TrimSpace(tarea.Modulo)
		agente := strings.TrimSpace(tarea.Agente)
		if modulo == "" || agente == "" {
			continue
		}
		moduleGroups[modulo] = append(moduleGroups[modulo], tarea)
	}
	conflicts := make([]supervisorModuleConflict, 0, len(moduleGroups))
	for modulo, tareas := range moduleGroups {
		agents := map[string]struct{}{}
		for _, tarea := range tareas {
			agents[strings.TrimSpace(tarea.Agente)] = struct{}{}
		}
		if len(agents) < 2 {
			continue
		}
		sort.Slice(tareas, func(i, j int) bool { return tareas[i].ID < tareas[j].ID })
		conflicts = append(conflicts, supervisorModuleConflict{Modulo: modulo, Tareas: tareas})
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].Modulo < conflicts[j].Modulo })
	return conflicts
}

func buildSupervisorRecommendedActions(gates []*db.ReviewGate, signals []*supervisorReviewSignal, merges []*db.GitMerge, conflicts []supervisorModuleConflict) []supervisorRecommendedAction {
	actions := make([]supervisorRecommendedAction, 0, len(gates)+len(signals)+len(merges)+len(conflicts))
	for _, gate := range gates {
		if gate == nil {
			continue
		}
		target := fmt.Sprintf("review_gate:%d", gate.ID)
		switch gate.Estado {
		case db.ReviewGateBloqueado:
			actions = append(actions, supervisorRecommendedAction{
				Kind:     "review_gate",
				Target:   target,
				Action:   "escalar_bloqueo_real",
				Reason:   "Gate bloqueado; revisar dependencia externa o decisión arquitectónica pendiente.",
				Priority: "alta",
			})
		case db.ReviewGateCambiosPed:
			actions = append(actions, supervisorRecommendedAction{
				Kind:     "review_gate",
				Target:   target,
				Action:   "relanzar_correccion",
				Reason:   "Gate con cambios pedidos; reenfocar al worker con findings actuales.",
				Priority: "alta",
			})
		default:
			actions = append(actions, supervisorRecommendedAction{
				Kind:     "review_gate",
				Target:   target,
				Action:   "asignar_o_confirmar_reviewer",
				Reason:   "Gate abierto sin cierre; revisar o confirmar responsible reviewer antes de integrar.",
				Priority: "media",
			})
		}
	}
	for _, signal := range signals {
		if signal == nil || signal.Event == nil {
			continue
		}
		action := "inspeccionar_signal"
		reason := "Señal de revisión reciente."
		priority := "media"
		switch strings.TrimSpace(signal.Event.Kind) {
		case "approval_request":
			action = "arbitrar_y_desbloquear"
			reason = "El worker pide criterio del supervisor para continuar o integrar."
			priority = "alta"
		case "waiting_human":
			action = "reencuadrar_sin_espera"
			reason = "El worker parece esperando humano; comprobar si el bloqueo es realmente externo."
			priority = "alta"
		case "ready_for_review":
			action = "inspeccionar_y_decidir_gate"
			reason = "Hay trabajo listo para revisión; decidir gate y siguiente paso de integración."
			priority = "media"
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "signal",
			Target:   fmt.Sprintf("runtime_event:%d", signal.Event.ID),
			Action:   action,
			Reason:   reason,
			Priority: priority,
		})
	}
	for _, merge := range merges {
		if merge == nil {
			continue
		}
		action := "inspeccionar_merge"
		reason := "Solicitud de merge viva."
		priority := "media"
		switch strings.TrimSpace(merge.Estado) {
		case "fallido":
			action = "reabrir_integracion"
			reason = "La integración falló; revisar causa y decidir corrección o reapertura."
			priority = "alta"
		case "aprobado":
			action = "confirmar_ejecutar_merge"
			reason = "El merge está aprobado; confirmar que no hay colisión ni gate pendiente antes de fusionar."
			priority = "media"
		case "pendiente", "validando":
			action = "revisar_merge_pendiente"
			reason = "Hay merge vivo aún sin completar ciclo de validación."
			priority = "media"
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "merge",
			Target:   fmt.Sprintf("git_merge:%d", merge.ID),
			Action:   action,
			Reason:   reason,
			Priority: priority,
		})
	}
	for _, conflict := range conflicts {
		if strings.TrimSpace(conflict.Modulo) == "" {
			continue
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "module_conflict",
			Target:   "modulo:" + strings.TrimSpace(conflict.Modulo),
			Action:   "repartir_o_serializar",
			Reason:   "Hay varias tareas activas del mismo módulo con agentes distintos; riesgo alto de pisada.",
			Priority: "alta",
		})
	}
	sortSupervisorRecommendedActions(actions)
	markSupervisorRecommendedActions(actions)
	return actions
}

func supervisorActionIsAutomaticallyApplicable(action string) bool {
	switch strings.TrimSpace(action) {
	case "asignar_tarea_libre", "reservar_tarea_libre", "replanificar_por_cuota", "seguir_guidance_durable", "rebalancear_reserva", "cerrar_propuesta_rechazada", "revisar_worktree_desfasada", "refrescar_worktree_limpia", "refrescar_store_subagentes":
		return true
	default:
		return false
	}
}

func markSupervisorRecommendedActions(actions []supervisorRecommendedAction) {
	for i := range actions {
		actions[i].AutoAplicable = supervisorActionIsAutomaticallyApplicable(actions[i].Action)
	}
}

func cloneSupervisorRecommendedActions(actions []supervisorRecommendedAction) []supervisorRecommendedAction {
	if len(actions) == 0 {
		return nil
	}
	out := make([]supervisorRecommendedAction, len(actions))
	copy(out, actions)
	return out
}

func buildSupervisorSafeActionQueue(actions []supervisorRecommendedAction) []supervisorRecommendedAction {
	if len(actions) == 0 {
		return nil
	}
	safeQueue := make([]supervisorRecommendedAction, 0, len(actions))
	for _, item := range actions {
		if supervisorActionIsAutomaticallyApplicable(item.Action) {
			safeQueue = append(safeQueue, item)
		}
	}
	projectRisk := supervisorWorkspaceProjectRiskSnapshotBuilder()
	sort.SliceStable(safeQueue, func(i, j int) bool {
		ri := supervisorActionProjectRisk(safeQueue[i], projectRisk)
		rj := supervisorActionProjectRisk(safeQueue[j], projectRisk)
		if ri != rj {
			return ri > rj
		}
		return false
	})
	return safeQueue
}

func buildSupervisorSafeActionQueueFast(actions []supervisorRecommendedAction) []supervisorRecommendedAction {
	if len(actions) == 0 {
		return nil
	}
	safeQueue := make([]supervisorRecommendedAction, 0, len(actions))
	for _, item := range actions {
		if supervisorActionIsAutomaticallyApplicable(item.Action) {
			safeQueue = append(safeQueue, item)
		}
	}
	return safeQueue
}

var supervisorPanelRowsBuilder = buildPanelRowsForControlPlane

var supervisorSemanticProgressRecentThreshold = 5 * time.Minute

func buildSupervisorOperationalActions(status apiStatusResponse, mailboxPendiente []apiOpenClawMailboxLite) []supervisorRecommendedAction {
	return buildSupervisorOperationalActionsWithOptions(status, mailboxPendiente, false)
}

func buildSupervisorOperationalActionsFast(status apiStatusResponse, mailboxPendiente []apiOpenClawMailboxLite) []supervisorRecommendedAction {
	return buildSupervisorOperationalActionsWithOptions(status, mailboxPendiente, true)
}

func buildSupervisorOperationalActionsWithOptions(status apiStatusResponse, mailboxPendiente []apiOpenClawMailboxLite, fast bool) []supervisorRecommendedAction {
	actions := make([]supervisorRecommendedAction, 0, 4+len(mailboxPendiente))
	idle := idleSupervisorWorkers(status.AgentesActivos, status.AgentesTrabajando)
	if len(idle) > 0 {
		assignee := idle[0]
		target := "backlog:libre"
		reason := fmt.Sprintf("Hay workers conectados sin trabajo (%s) y backlog libre disponible.", strings.Join(idle, ", "))
		priority := "media"
		if len(status.AgentesTrabajando) == 0 {
			priority = "alta"
		}
		if !fast {
			libres, err := db.ListarTareas(db.FiltroTareas{Libre: true})
			if err == nil && len(libres) > 0 && libres[0] != nil {
				target = fmt.Sprintf("tarea:%d", libres[0].ID)
			}
		}
		if libresCount := countLibreTasks(status); libresCount > 0 || strings.HasPrefix(target, "tarea:") {
			actions = append(actions, supervisorRecommendedAction{
				Kind:     "dispatch",
				Target:   target,
				Action:   "asignar_tarea_libre",
				Reason:   reason,
				Priority: priority,
				Assignee: assignee,
			})
		}
	}
	if len(idle) == 0 && len(status.AgentesActivos) > 0 {
		assignee := preferredSupervisorWorker(status)
		target := "backlog:libre"
		reason := "Hay backlog libre y workers conectados, pero ninguno ocioso; conviene reservar el siguiente frente sin arrancarlo aún."
		if !fast {
			libres, err := db.ListarTareas(db.FiltroTareas{Libre: true})
			if err == nil && len(libres) > 0 && libres[0] != nil {
				target = fmt.Sprintf("tarea:%d", libres[0].ID)
			}
		}
		if libresCount := countLibreTasks(status); libresCount > 0 || strings.HasPrefix(target, "tarea:") {
			actions = append(actions, supervisorRecommendedAction{
				Kind:     "dispatch",
				Target:   target,
				Action:   "reservar_tarea_libre",
				Reason:   reason,
				Priority: "baja",
				Assignee: assignee,
			})
		}
	}
	actions = append(actions, buildSupervisorReserveRebalanceActions(status)...)
	retenidas := tareasRetenidasPorCuota(status.TareasActivas, status.Agentes, status.AgentesQuotaBlocked)
	if len(retenidas) > 0 {
		priority := "media"
		if len(status.AgentesActivos) == 0 {
			priority = "alta"
		}
		assignee := preferredSupervisorWorker(status)
		for _, retenida := range retenidas {
			actions = append(actions, supervisorRecommendedAction{
				Kind:     "quota_hold",
				Target:   fmt.Sprintf("tarea:%d", retenida.ID),
				Action:   "replanificar_por_cuota",
				Reason:   "Hay trabajo retenido por cuota; revisar reasignación o secuenciación sin esperar al agente bloqueado.",
				Priority: priority,
				Assignee: assignee,
			})
		}
	}
	for _, item := range mailboxPendiente {
		if strings.TrimSpace(item.Agente) == "" || item.Count <= 0 {
			continue
		}
		if strings.Contains(strings.TrimSpace(item.SupervisorActionsCSV), "revisar_worktree_desfasada") {
			continue
		}
		priority := "baja"
		if item.OldestAgeMin >= 30 {
			priority = "media"
		}
		if item.OldestAgeMin >= 120 {
			priority = "alta"
		}
		reason := fmt.Sprintf("El worker mantiene guidance durable pendiente (%d mensaje(s): %s); conviene observar continuidad y drenado.", item.Count, strings.TrimSpace(item.KindsCSV))
		if item.OldestAgeMin > 0 {
			reason = fmt.Sprintf("El worker mantiene guidance durable pendiente desde hace %d min (%d mensaje(s): %s); conviene observar continuidad y drenado.", item.OldestAgeMin, item.Count, strings.TrimSpace(item.KindsCSV))
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "mailbox_pending",
			Target:   "agente:" + strings.TrimSpace(item.Agente),
			Action:   "seguir_guidance_durable",
			Reason:   reason,
			Priority: priority,
			Assignee: strings.TrimSpace(item.Agente),
		})
	}
	actions = append(actions, buildSupervisorAutonomyActions(status.Autonomia.Recent)...)
	if !fast {
		actions = supervisorApplyLiveProgressToAutonomyActions(actions, status)
		sortSupervisorRecommendedActions(actions)
		return actions
	}
	sortSupervisorRecommendedActionsFast(actions)
	return actions
}

func supervisorApplyLiveProgressToAutonomyActions(actions []supervisorRecommendedAction, status apiStatusResponse) []supervisorRecommendedAction {
	if len(actions) == 0 || supervisorPanelRowsBuilder == nil {
		return actions
	}
	rows, err := supervisorPanelRowsBuilder()
	if err != nil || len(rows) == 0 {
		return actions
	}
	now := time.Now().UTC()
	rowByAgent := make(map[string]agentesapp.Row, len(rows))
	for _, row := range rows {
		if row.Agente == nil || strings.TrimSpace(row.Agente.Nombre) == "" {
			continue
		}
		rowByAgent[nombreAgenteCanonico(row.Agente.Nombre)] = row
	}
	visibleTaskIDs := supervisorVisibleTasks(status)
	visibleTasksByAgent := supervisorVisibleTasksByAgent(status)
	out := make([]supervisorRecommendedAction, 0, len(actions))
	for _, action := range actions {
		if adjusted, keep := supervisorAdjustAutonomyActionByRows(action, rowByAgent, visibleTaskIDs, visibleTasksByAgent, now); keep {
			out = append(out, adjusted)
		}
	}
	return out
}

func supervisorAdjustAutonomyActionByRows(action supervisorRecommendedAction, rowByAgent map[string]agentesapp.Row, visibleTasks map[int64]tareaLite, visibleTasksByAgent map[string]tareaLite, now time.Time) (supervisorRecommendedAction, bool) {
	if strings.TrimSpace(action.Kind) != "autonomy_event" {
		return action, true
	}
	if taskID := parseSupervisorActionTargetID("tarea:", action.Target); taskID != nil {
		switch strings.TrimSpace(action.Action) {
		case "verificar_handoff_consolidado", "seguir_reasignacion":
			if _, ok := visibleTasks[*taskID]; !ok {
				return action, false
			}
		}
	}
	if adjusted, keep, handled := supervisorAdjustAutonomyFollowupByVisibleTask(action, visibleTasks, visibleTasksByAgent); handled {
		return adjusted, keep
	}
	agent := nombreAgenteCanonico(strings.TrimSpace(action.Assignee))
	if agent == "" {
		if strings.HasPrefix(strings.TrimSpace(action.Target), "agente:") {
			agent = nombreAgenteCanonico(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(action.Target), "agente:")))
		}
	}
	row, ok := rowByAgent[agent]
	if !ok {
		return action, true
	}
	task, hasVisibleTask := supervisorVisibleTaskForAutonomyAction(action, visibleTasks, visibleTasksByAgent)
	if !supervisorRowShowsRecentSemanticProgress(row, now) {
		if strings.TrimSpace(action.Action) == "inspeccionar_handoff_fallido" && hasVisibleTask {
			switch task.Estado {
			case db.TareaBloqueada:
				action.Priority = "alta"
				action.Reason = strings.TrimSpace(action.Reason + supervisorBlockedFrontReasonSuffix(task))
				if supervisorTaskBlocksRealIntegration(action, task) {
					action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración real.")
				}
			case db.TareaAsignada:
				if strings.TrimSpace(action.Priority) == "" || strings.EqualFold(strings.TrimSpace(action.Priority), "baja") {
					action.Priority = "media"
				}
				action.Reason = strings.TrimSpace(action.Reason + " El frente sigue asignado y sin progreso confirmado.")
				if supervisorTaskBlocksRealIntegration(action, task) {
					action.Priority = "alta"
					action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración real.")
				}
			case db.TareaEnProgreso:
				if strings.TrimSpace(action.Priority) == "" || strings.EqualFold(strings.TrimSpace(action.Priority), "alta") {
					action.Priority = "media"
				}
				action.Reason = strings.TrimSpace(action.Reason + " El frente reapareció en progreso, pero aún sin confirmación semántica reciente.")
			}
		}
		return action, true
	}
	switch strings.TrimSpace(action.Action) {
	case "verificar_handoff_consolidado":
		return action, false
	case "seguir_reasignacion", "seguir_reinicio_runtime", "resolver_followup_bloqueado":
		action.Priority = "baja"
		if supervisorActionCarriesStrongEvidence(action) && strings.TrimSpace(action.Action) == "resolver_followup_bloqueado" {
			action.Priority = "media"
		}
		if hasVisibleTask && task.Estado == db.TareaEnProgreso {
			action.Reason = strings.TrimSpace(action.Reason + " Ya hay progreso reciente del worker destino. El frente ya está en progreso visible y con progreso sin bloqueo visible.")
			if supervisorActionBlocksGlobalProgress(action, task) {
				action.Priority = "media"
				action.Reason = strings.TrimSpace(action.Reason + " Aun así, sigue afectando integración/progreso global. Frente crítico del proyecto/workspace.")
			} else {
				action.Reason = strings.TrimSpace(action.Reason + " No parece bloquear integración/progreso global. Frente local o no bloqueante.")
			}
			return action, true
		}
		action.Reason = strings.TrimSpace(action.Reason + " Ya hay progreso reciente del worker destino.")
		return action, true
	case "inspeccionar_handoff_fallido":
		if !hasVisibleTask {
			if supervisorActionCarriesStrongEvidence(action) {
				action.Priority = "media"
				action.Reason = strings.TrimSpace(action.Reason + " La evidencia fuerte aconseja mantener seguimiento aunque el frente ya no sea visible.")
				return action, true
			}
			if taskID := parseSupervisorActionTargetID("tarea:", action.Target); taskID != nil {
				if _, ok := visibleTasks[*taskID]; !ok {
					return action, false
				}
			}
			action.Priority = "media"
			action.Reason = strings.TrimSpace(action.Reason + " El destino ya muestra progreso reciente.")
			return action, true
		}
		switch task.Estado {
		case db.TareaBloqueada:
			action.Priority = "alta"
			action.Reason = strings.TrimSpace(action.Reason + supervisorBlockedFrontReasonSuffix(task))
			if supervisorTaskBlocksRealIntegration(action, task) {
				action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración real.")
			}
			if supervisorActionBlocksGlobalProgress(action, task) {
				action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración/progreso global. Frente crítico del proyecto/workspace.")
			}
			return action, true
		case db.TareaAsignada:
			action.Priority = "media"
			action.Reason = strings.TrimSpace(action.Reason + " El frente sigue asignado y requiere validar el relevo.")
			if supervisorTaskBlocksRealIntegration(action, task) {
				action.Priority = "alta"
				action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración real.")
			}
			if supervisorActionBlocksGlobalProgress(action, task) {
				action.Priority = "alta"
				action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración/progreso global. Frente crítico del proyecto/workspace.")
			}
			return action, true
		case db.TareaEnProgreso:
			if supervisorActionCarriesStrongEvidence(action) {
				action.Reason = strings.TrimSpace(action.Reason + " El frente ya progresa, pero la evidencia fuerte aconseja cerrar el seguimiento antes de descartarlo.")
				if supervisorActionBlocksGlobalProgress(action, task) {
					action.Priority = "media"
					action.Reason = strings.TrimSpace(action.Reason + " Sigue afectando integración/progreso global. Frente crítico del proyecto/workspace.")
				} else {
					action.Priority = "baja"
					action.Reason = strings.TrimSpace(action.Reason + " Ya hay progreso reciente del worker destino. No parece bloquear integración/progreso global. Frente local o ya mitigado.")
				}
				return action, true
			}
			return action, false
		}
		return action, false
	default:
		return action, true
	}
}

func supervisorRowShowsRecentSemanticProgress(row agentesapp.Row, now time.Time) bool {
	if strings.EqualFold(strings.TrimSpace(row.LastAutonomyState), "work_confirmed") &&
		strings.EqualFold(strings.TrimSpace(row.EstadoOperativo), "trabajando") {
		return true
	}
	cutoff := now.Add(-supervisorSemanticProgressRecentThreshold)
	if row.WorkerLastProgress != nil && !row.WorkerLastProgress.IsZero() && !row.WorkerLastProgress.Before(cutoff) &&
		strings.EqualFold(strings.TrimSpace(row.EstadoOperativo), "trabajando") {
		return true
	}
	return false
}

func supervisorVisibleTasks(status apiStatusResponse) map[int64]tareaLite {
	out := make(map[int64]tareaLite, len(status.TareasActivas)+len(status.TareasReservadas))
	for _, item := range status.TareasActivas {
		if item.ID > 0 {
			out[item.ID] = item
		}
	}
	for _, item := range status.TareasReservadas {
		if item.ID > 0 {
			out[item.ID] = item
		}
	}
	return out
}

func supervisorVisibleTasksByAgent(status apiStatusResponse) map[string]tareaLite {
	out := make(map[string]tareaLite, len(status.TareasActivas)+len(status.TareasReservadas))
	for _, item := range status.TareasActivas {
		supervisorStorePreferredVisibleTaskByAgent(out, item)
	}
	for _, item := range status.TareasReservadas {
		supervisorStorePreferredVisibleTaskByAgent(out, item)
	}
	return out
}

func supervisorStorePreferredVisibleTaskByAgent(out map[string]tareaLite, item tareaLite) {
	agent := nombreAgenteCanonico(strings.TrimSpace(item.Agente))
	if agent == "" || item.ID <= 0 {
		return
	}
	current, ok := out[agent]
	if !ok || supervisorVisibleTaskPriority(item) < supervisorVisibleTaskPriority(current) {
		out[agent] = item
	}
}

func supervisorVisibleTaskPriority(task tareaLite) int {
	switch task.Estado {
	case db.TareaBloqueada:
		return 0
	case db.TareaAsignada:
		return 1
	case db.TareaEnProgreso:
		return 2
	default:
		return 3
	}
}

func supervisorAdjustAutonomyFollowupByVisibleTask(action supervisorRecommendedAction, visibleTasks map[int64]tareaLite, visibleTasksByAgent map[string]tareaLite) (supervisorRecommendedAction, bool, bool) {
	if !supervisorAutonomyActionUsesVisibleFrontState(action.Action) {
		return action, true, false
	}
	task, ok := supervisorVisibleTaskForAutonomyAction(action, visibleTasks, visibleTasksByAgent)
	if !ok {
		return action, true, false
	}
	switch task.Estado {
	case db.TareaBloqueada:
		action.Priority = "alta"
		action.Reason = strings.TrimSpace(action.Reason + supervisorBlockedFrontReasonSuffix(task))
		if supervisorTaskBlocksRealIntegration(action, task) {
			action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración real.")
		}
		if supervisorActionBlocksGlobalProgress(action, task) {
			action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración/progreso global. Frente crítico del proyecto/workspace.")
		}
		return action, true, true
	case db.TareaAsignada:
		if strings.TrimSpace(action.Action) == "inspeccionar_handoff_fallido" {
			action.Priority = "media"
			action.Reason = strings.TrimSpace(action.Reason + " El frente sigue asignado, retenido y visible.")
			if supervisorTaskBlocksRealIntegration(action, task) {
				action.Priority = "alta"
				action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración real.")
			}
			if supervisorActionBlocksGlobalProgress(action, task) {
				action.Priority = "alta"
				action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración/progreso global. Frente crítico del proyecto/workspace.")
			}
			return action, true, true
		} else if strings.TrimSpace(action.Priority) == "" || strings.EqualFold(strings.TrimSpace(action.Priority), "baja") {
			action.Priority = "media"
		}
		action.Reason = strings.TrimSpace(action.Reason + " El frente sigue retenido y visible.")
		if supervisorTaskBlocksRealIntegration(action, task) {
			action.Priority = "alta"
			action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración real.")
		}
		if supervisorActionBlocksGlobalProgress(action, task) {
			action.Priority = "alta"
			action.Reason = strings.TrimSpace(action.Reason + " Bloquea integración/progreso global. Frente crítico del proyecto/workspace.")
		}
		return action, true, true
	default:
		return action, true, false
	}
}

func supervisorBlockedFrontReasonSuffix(task tareaLite) string {
	suffix := " El frente sigue bloqueado y visible."
	switch task.Prioridad {
	case db.PrioridadAlta:
		suffix = " El frente sigue bloqueado, visible y es de prioridad alta."
	}
	if modulo := strings.TrimSpace(task.Modulo); modulo != "" {
		suffix += " Módulo: " + modulo + "."
	}
	return suffix
}

func supervisorActionCarriesStrongEvidence(action supervisorRecommendedAction) bool {
	reason := strings.ToLower(strings.TrimSpace(action.Reason))
	if strings.Contains(reason, "evidencia fuerte") {
		return true
	}
	for _, marker := range []string{
		"evidencia: checkpoint:",
		"evidencia: runtime_order:",
		"evidencia: runtime:",
		"evidencia: handle:",
		"evidencia: patch:",
		"evidencia: test:",
		"evidencia: transcript:",
		" checkpoint ",
		" patch ",
		" test ",
	} {
		if strings.Contains(reason, marker) {
			return true
		}
	}
	return false
}

func supervisorAutonomyActionUsesVisibleFrontState(action string) bool {
	switch strings.TrimSpace(action) {
	case "verificar_handoff_consolidado", "seguir_reasignacion", "seguir_reinicio_runtime", "inspeccionar_handoff_fallido", "resolver_followup_bloqueado":
		return true
	default:
		return false
	}
}

func supervisorTaskBlocksRealIntegration(action supervisorRecommendedAction, task tareaLite) bool {
	if task.Estado != db.TareaBloqueada && task.Estado != db.TareaAsignada {
		return false
	}
	switch strings.TrimSpace(action.Action) {
	case "resolver_followup_bloqueado":
		return true
	case "inspeccionar_handoff_fallido":
		if task.Prioridad == db.PrioridadAlta {
			return true
		}
		text := strings.ToLower(strings.TrimSpace(task.Titulo + " " + task.Modulo))
		for _, marker := range []string{"integr", "merge", "release", "deploy", "review", "handoff", "api", "mcp"} {
			if strings.Contains(text, marker) {
				return true
			}
		}
	}
	return false
}

func supervisorActionBlocksGlobalProgress(action supervisorRecommendedAction, task tareaLite) bool {
	if !supervisorActionCarriesStrongEvidence(action) {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		action.Reason,
		task.Titulo,
		task.Modulo,
	}, " ")))
	if strings.Contains(text, "review gate") {
		return true
	}
	for _, marker := range []string{"merge", "api", "mcp", "deploy", "release", "integr"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func supervisorVisibleTaskForAutonomyAction(action supervisorRecommendedAction, visibleTasks map[int64]tareaLite, visibleTasksByAgent map[string]tareaLite) (tareaLite, bool) {
	if taskID := parseSupervisorActionTargetID("tarea:", action.Target); taskID != nil {
		task, ok := visibleTasks[*taskID]
		return task, ok
	}
	agent := nombreAgenteCanonico(strings.TrimSpace(action.Assignee))
	if agent == "" && strings.HasPrefix(strings.TrimSpace(action.Target), "agente:") {
		agent = nombreAgenteCanonico(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(action.Target), "agente:")))
	}
	task, ok := visibleTasksByAgent[agent]
	return task, ok
}

func buildSupervisorAutonomyActions(items []autonomyEventSummary) []supervisorRecommendedAction {
	if len(items) == 0 {
		return nil
	}
	actions := make([]supervisorRecommendedAction, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	taskActionIndex := make(map[string]int, len(items))
	for _, item := range items {
		action, ok := supervisorActionFromAutonomyEvent(item)
		if !ok {
			continue
		}
		key := supervisorAutonomyActionSeenKey(action)
		if target := strings.TrimSpace(action.Target); strings.HasPrefix(target, "tarea:") {
			if rank, collapsible := supervisorAutonomyTaskCollapseRank(action.Action); collapsible {
				if existingIdx, ok := taskActionIndex[target]; ok && existingIdx >= 0 && existingIdx < len(actions) {
					existing := actions[existingIdx]
					if existingRank, existingCollapsible := supervisorAutonomyTaskCollapseRank(existing.Action); existingCollapsible {
						if existingRank <= rank {
							continue
						}
						delete(seen, supervisorAutonomyActionSeenKey(existing))
						seen[key] = struct{}{}
						actions[existingIdx] = action
						continue
					}
				}
				taskActionIndex[target] = len(actions)
			}
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		actions = append(actions, action)
	}
	return actions
}

func supervisorAutonomyTaskCollapseRank(action string) (int, bool) {
	switch strings.TrimSpace(action) {
	case "inspeccionar_handoff_fallido":
		return 0, true
	case "resolver_followup_bloqueado":
		return 1, true
	case "verificar_handoff_consolidado":
		return 2, true
	case "seguir_reasignacion":
		return 3, true
	default:
		return 0, false
	}
}

func supervisorAutonomyActionSeenKey(action supervisorRecommendedAction) string {
	return strings.Join([]string{
		strings.TrimSpace(action.Kind),
		strings.TrimSpace(action.Target),
		strings.TrimSpace(action.Action),
	}, "|")
}

func supervisorActionFromAutonomyEvent(item autonomyEventSummary) (supervisorRecommendedAction, bool) {
	target := supervisorAutonomyActionTarget(item)
	kind := "autonomy_event"
	reasonSuffix := strings.TrimSpace(item.Reason)
	artifactSuffix := ""
	strongEvidence := supervisorAutonomyEventHasStrongEvidence(item)
	if len(item.Artifacts) > 0 {
		artifactSuffix = " Evidencia: " + strings.Join(item.Artifacts, ", ")
		if item.ArtifactsMore > 0 {
			artifactSuffix += fmt.Sprintf(" (+%d)", item.ArtifactsMore)
		}
	}
	if strongEvidence {
		artifactSuffix += " Evidencia fuerte."
	}
	switch strings.TrimSpace(item.Kind) {
	case "runtime_restart_requested":
		reason := "Hay un reinicio de runtime pedido por el control plane; verificar convergencia antes de abrir más trabajo."
		if reasonSuffix != "" {
			reason += " Motivo: " + reasonSuffix + "."
		}
		return supervisorRecommendedAction{
			Kind:     kind,
			Target:   target,
			Action:   "seguir_reinicio_runtime",
			Reason:   strings.TrimSpace(reason + artifactSuffix),
			Priority: "media",
			Assignee: firstNonEmpty(strings.TrimSpace(item.TargetAgent), strings.TrimSpace(item.Agent), strings.TrimSpace(item.Supervisor)),
		}, true
	case "task_reassigned":
		reason := "Hubo una reasignación reciente; confirmar que el nuevo worker absorbió el frente sin deriva."
		if reasonSuffix != "" {
			reason += " Motivo: " + reasonSuffix + "."
		}
		return supervisorRecommendedAction{
			Kind:     kind,
			Target:   target,
			Action:   "seguir_reasignacion",
			Reason:   strings.TrimSpace(reason + artifactSuffix),
			Priority: "media",
			Assignee: firstNonEmpty(strings.TrimSpace(item.TargetAgent), strings.TrimSpace(item.Supervisor), strings.TrimSpace(item.Agent)),
		}, true
	case "handoff_completed":
		reason := "El handoff ya se consolidó; conviene verificar progreso real del destino y cerrar seguimiento residual."
		if reasonSuffix != "" {
			reason += " Motivo: " + reasonSuffix + "."
		}
		return supervisorRecommendedAction{
			Kind:     kind,
			Target:   target,
			Action:   "verificar_handoff_consolidado",
			Reason:   strings.TrimSpace(reason + artifactSuffix),
			Priority: "baja",
			Assignee: firstNonEmpty(strings.TrimSpace(item.TargetAgent), strings.TrimSpace(item.Supervisor), strings.TrimSpace(item.Agent)),
		}, true
	case "handoff_failed":
		reason := "Handoff autónomo fallido reciente; revisar reasignación, destino y consolidación."
		if reasonSuffix != "" {
			reason += " Motivo: " + reasonSuffix + "."
		}
		return supervisorRecommendedAction{
			Kind:     kind,
			Target:   target,
			Action:   "inspeccionar_handoff_fallido",
			Reason:   strings.TrimSpace(reason + artifactSuffix),
			Priority: "alta",
			Assignee: firstNonEmpty(strings.TrimSpace(item.Supervisor), strings.TrimSpace(item.TargetAgent), strings.TrimSpace(item.Agent)),
		}, true
	case "post_remediation_blocked_escalated":
		reason := "Follow-up bloqueado tras remediación; decidir escalado, cambio de slice o desbloqueo real."
		if reasonSuffix != "" {
			reason += " Motivo: " + reasonSuffix + "."
		}
		return supervisorRecommendedAction{
			Kind:     kind,
			Target:   target,
			Action:   "resolver_followup_bloqueado",
			Reason:   strings.TrimSpace(reason + artifactSuffix),
			Priority: "alta",
			Assignee: firstNonEmpty(strings.TrimSpace(item.Supervisor), strings.TrimSpace(item.Agent), strings.TrimSpace(item.TargetAgent)),
		}, true
	case "worker_recovery_paused_external":
		reason := "Recuperación pausada por dependencia externa; revisar conector o ventana operativa antes de reintentar."
		if reasonSuffix != "" {
			reason += " Motivo: " + reasonSuffix + "."
		}
		return supervisorRecommendedAction{
			Kind:     kind,
			Target:   target,
			Action:   "revisar_pausa_externa",
			Reason:   strings.TrimSpace(reason + artifactSuffix),
			Priority: "media",
			Assignee: firstNonEmpty(strings.TrimSpace(item.Supervisor), strings.TrimSpace(item.Agent)),
		}, true
	case "repair_helper_opened":
		reason := "Hay un repair-helper activo; seguir si converge o si necesita relevo limpio."
		if reasonSuffix != "" {
			reason += " Motivo: " + reasonSuffix + "."
		}
		return supervisorRecommendedAction{
			Kind:     kind,
			Target:   target,
			Action:   "seguir_repair_helper",
			Reason:   strings.TrimSpace(reason + artifactSuffix),
			Priority: "media",
			Assignee: firstNonEmpty(strings.TrimSpace(item.Supervisor), strings.TrimSpace(item.TargetAgent), strings.TrimSpace(item.Agent)),
		}, true
	default:
		return supervisorRecommendedAction{}, false
	}
}

func supervisorAutonomyEventHasStrongEvidence(item autonomyEventSummary) bool {
	for _, artifact := range item.Artifacts {
		value := strings.ToLower(strings.TrimSpace(artifact))
		for _, prefix := range []string{"checkpoint:", "runtime_order:", "runtime:", "handle:", "patch:", "test:", "transcript:"} {
			if strings.HasPrefix(value, prefix) {
				return true
			}
		}
	}
	reason := strings.ToLower(strings.TrimSpace(item.Reason))
	for _, marker := range []string{"checkpoint", "runtime", "patch", "test", "transcript"} {
		if strings.Contains(reason, marker) {
			return true
		}
	}
	return item.ArtifactsMore > 0 && len(item.Artifacts) > 0
}

func supervisorAutonomyActionTarget(item autonomyEventSummary) string {
	switch {
	case item.TaskID != nil && *item.TaskID > 0:
		return fmt.Sprintf("tarea:%d", *item.TaskID)
	case strings.TrimSpace(item.TargetAgent) != "":
		return "agente:" + strings.TrimSpace(item.TargetAgent)
	case strings.TrimSpace(item.Agent) != "":
		return "agente:" + strings.TrimSpace(item.Agent)
	case item.RuntimeID != nil && *item.RuntimeID > 0:
		return fmt.Sprintf("runtime:%d", *item.RuntimeID)
	case item.HandleID != nil && *item.HandleID > 0:
		return fmt.Sprintf("handle:%d", *item.HandleID)
	default:
		return "autonomia:reciente"
	}
}

var (
	openClawWorktreeDriftBuilder = buildOpenClawWorktreeDriftFromAPIStatusCached
	openClawWorktreeDriftTimeout = 150 * time.Millisecond
)
var supervisorRefreshCleanWorktreeDrift = refreshSupervisorCleanWorktreeDrift

func buildSupervisorWorktreeDriftActions(items []apiOpenClawWorktreeDrift) []supervisorRecommendedAction {
	if len(items) == 0 {
		return nil
	}
	actions := make([]supervisorRecommendedAction, 0, len(items))
	for _, item := range items {
		agente := strings.TrimSpace(item.Agente)
		if agente == "" {
			continue
		}
		if pending, err := hasPendingWorktreeDriftCheckpointRequest(agente); err == nil && pending {
			continue
		}
		reason := "La worktree del agente está desfasada respecto al HEAD del servidor; conviene refrescar base antes de integrar cambios."
		if item.CurrentHead != "" || item.ExpectedHead != "" {
			reason = fmt.Sprintf("La worktree del agente está desfasada (%s -> %s); conviene refrescar base antes de integrar cambios.", strings.TrimSpace(item.CurrentHead), strings.TrimSpace(item.ExpectedHead))
		}
		if strings.TrimSpace(item.DirtySummary) != "" {
			reason += " Estado local: " + strings.TrimSpace(item.DirtySummary) + "."
		}
		actionName := "revisar_worktree_desfasada"
		if !item.Dirty {
			actionName = "refrescar_worktree_limpia"
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "worktree_drift",
			Target:   "agente:" + agente,
			Action:   actionName,
			Reason:   reason,
			Priority: "alta",
			Assignee: agente,
		})
	}
	return actions
}

func hasPendingWorktreeDriftCheckpointRequest(agente string) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, nil
	}
	estadoPendiente := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{
		Agente: &agente,
		Estado: &estadoPendiente,
		Limit:  20,
	})
	if err != nil {
		return false, err
	}
	for _, order := range orders {
		if runtimePayloadHasSupervisorAction(order.Tipo, order.PayloadJSON, "revisar_worktree_desfasada") {
			return true, nil
		}
	}
	mailbox, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente: &agente,
		Estado:   &estadoPendiente,
	})
	if err != nil {
		return false, err
	}
	for _, msg := range mailbox {
		if runtimePayloadHasSupervisorAction(msg.Kind, msg.PayloadJSON, "revisar_worktree_desfasada") {
			return true, nil
		}
	}
	return false, nil
}

func runtimePayloadHasSupervisorAction(kind, payloadJSON, expected string) bool {
	if strings.TrimSpace(payloadJSON) == "" {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return false
	}
	if action := strings.TrimSpace(supervisorActionString(payload["supervisor_action"])); action == strings.TrimSpace(expected) {
		return true
	}
	if action := strings.TrimSpace(supervisorActionString(payload["accion"])); action == strings.TrimSpace(expected) && strings.TrimSpace(kind) == "autonomia" {
		return true
	}
	if drift, ok := payload["worktree_drift"].(map[string]any); ok && strings.TrimSpace(expected) == "revisar_worktree_desfasada" {
		return len(drift) > 0
	}
	return false
}

func supervisorActionString(v any) string {
	switch item := v.(type) {
	case string:
		return item
	case fmt.Stringer:
		return item.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func buildSupervisorObservedSessionActions(status apiStatusResponse, snapshot map[string]any) []supervisorRecommendedAction {
	candidates := buildOpenClawSessionCandidates(snapshot)
	if len(candidates) == 0 {
		return nil
	}
	activos := make(map[string]struct{}, len(status.AgentesActivos))
	for _, agente := range status.AgentesActivos {
		if agente == nil {
			continue
		}
		activos[strings.ToLower(strings.TrimSpace(agente.Nombre))] = struct{}{}
	}
	enCuota := make(map[string]struct{}, len(status.Agentes))
	for _, agente := range agentesNoActivosConCuota(status.Agentes) {
		if agente == nil {
			continue
		}
		enCuota[strings.ToLower(strings.TrimSpace(agente.Nombre))] = struct{}{}
	}
	actions := make([]supervisorRecommendedAction, 0, len(candidates))
	for _, candidate := range candidates {
		agente := strings.TrimSpace(candidate.Agente)
		if agente == "" {
			continue
		}
		key := strings.ToLower(agente)
		if _, ok := activos[key]; ok {
			continue
		}
		if _, ok := enCuota[key]; ok {
			continue
		}
		reasonParts := make([]string, 0, 4)
		if candidate.ExternalSessionID != "" {
			reasonParts = append(reasonParts, "session_id="+candidate.ExternalSessionID)
		}
		if candidate.ObservedSessionPath != "" {
			reasonParts = append(reasonParts, "path="+candidate.ObservedSessionPath)
		}
		if candidate.Herramienta != "" {
			reasonParts = append(reasonParts, "tool="+candidate.Herramienta)
		}
		if candidate.UsageSummary != "" {
			reasonParts = append(reasonParts, candidate.UsageSummary)
		}
		reason := "Existe una sesión observada reutilizable; conviene inspeccionarla antes de relanzar el agente."
		if len(reasonParts) > 0 {
			reason += " (" + strings.Join(reasonParts, " · ") + ")"
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "session_candidate",
			Target:   "agente:" + agente,
			Action:   "inspeccionar_sesion_observada",
			Reason:   reason,
			Priority: "baja",
			Assignee: agente,
		})
	}
	return actions
}

func buildSupervisorSubagentActions(snapshot map[string]any) []supervisorRecommendedAction {
	items, _ := snapshot["subagents"].([]*db.SupervisorSubagent)
	store, _ := snapshot["store"].(supervisorSubagentStoreSummary)
	actions := make([]supervisorRecommendedAction, 0, len(items)+1)
	if store.Exists && store.ManifestCount > 0 && len(items) == 0 {
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "subagent_store",
			Target:   "supervisor:" + resolveSupervisorName(""),
			Action:   "refrescar_store_subagentes",
			Reason:   "Hay manifests en .clawd-agents pero aún no se han importado al modelo canónico de subagentes.",
			Priority: "media",
			Assignee: resolveSupervisorName(""),
		})
	}
	for _, item := range items {
		if item == nil || item.ID <= 0 {
			continue
		}
		actionName := ""
		reason := ""
		priority := "baja"
		metadata := metadataSupervisorSubagente(item)
		isPipelineParallel := strings.EqualFold(strings.TrimSpace(stringSupervisorSubagente(metadata["source"])), "pipeline_local_parallel")
		switch strings.TrimSpace(item.Status) {
		case "failed":
			actionName = "revisar_subagente_fallido"
			reason = "El subagente terminó en fallo; revisar artefactos y decidir relanzamiento o descarte."
			priority = "alta"
		case "completed":
			actionName = "recoger_resultado_subagente"
			reason = "El subagente completó su trabajo; recoger resultado y decidir integración."
			priority = "media"
			if isPipelineParallel {
				priority = "alta"
			}
		case "cancelled":
			actionName = "limpiar_subagente_cancelado"
			reason = "El subagente quedó cancelado; confirmar limpieza y estado terminal."
			priority = "baja"
		default:
			continue
		}
		if name := strings.TrimSpace(item.SubagentName); name != "" {
			reason = fmt.Sprintf("%s (subagente=%s tipo=%s)", reason, name, strings.TrimSpace(item.SubagentType))
		}
		if isPipelineParallel {
			sliceInfo := ""
			if idx := int64SupervisorSubagente(metadata["slice_index"]); idx > 0 {
				if total := int64SupervisorSubagente(metadata["slice_total"]); total > 0 {
					sliceInfo = fmt.Sprintf(" slice=%d/%d", idx, total)
				} else {
					sliceInfo = fmt.Sprintf(" slice=%d", idx)
				}
			}
			taskTitle := strings.TrimSpace(stringSupervisorSubagente(metadata["task_title"]))
			if taskTitle != "" {
				reason = fmt.Sprintf("%s%s tarea=%s", reason, sliceInfo, taskTitle)
			} else if sliceInfo != "" {
				reason = reason + sliceInfo
			}
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "subagent",
			Target:   fmt.Sprintf("subagente:%d", item.ID),
			Action:   actionName,
			Reason:   reason,
			Priority: priority,
			Assignee: resolveSupervisorName(item.Supervisor),
		})
	}
	return actions
}

func buildSupervisorReserveRebalanceActions(status apiStatusResponse) []supervisorRecommendedAction {
	reservadas := filtrarOpenClawTareasPorEstado(status.TareasActivas, db.TareaAsignada)
	if len(reservadas) == 0 || len(status.AgentesActivos) < 2 {
		return nil
	}
	cargaActiva, cargaReservada := buildOpenClawAgentLoad(status.TareasActivas)
	actions := make([]supervisorRecommendedAction, 0, len(reservadas))
	for _, tarea := range reservadas {
		origen := strings.TrimSpace(tarea.Agente)
		if origen == "" {
			continue
		}
		destino, cargaOrigen, cargaDestino := lowerLoadActiveWorker(status.AgentesActivos, cargaActiva, cargaReservada, origen)
		if destino == "" {
			continue
		}
		if cargaOrigen-cargaDestino < 2 {
			continue
		}
		priority := "media"
		if cargaOrigen-cargaDestino >= 3 {
			priority = "alta"
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "dispatch",
			Target:   fmt.Sprintf("tarea:%d", tarea.ID),
			Action:   "rebalancear_reserva",
			Reason:   fmt.Sprintf("La reserva #%d está en %s con carga %d; %s tiene carga %d y puede absorberla sin tocar trabajo en progreso.", tarea.ID, origen, cargaOrigen, destino, cargaDestino),
			Priority: priority,
			Assignee: destino,
		})
	}
	return actions
}

func buildSupervisorProposalActions(propuestas []propuestaLite) []supervisorRecommendedAction {
	if len(propuestas) == 0 {
		return nil
	}
	actions := make([]supervisorRecommendedAction, 0, len(propuestas))
	for _, propuesta := range propuestas {
		if propuesta.Estado != db.PropuestaAbierta || propuesta.Pendiente > 0 {
			continue
		}
		if propuesta.Desacuerdo <= propuesta.Acuerdo {
			continue
		}
		actions = append(actions, supervisorRecommendedAction{
			Kind:     "proposal",
			Target:   "propuesta:" + strings.TrimSpace(propuesta.Codigo),
			Action:   "cerrar_propuesta_rechazada",
			Reason:   fmt.Sprintf("La propuesta %s ya terminó la votación y tiene mayoría de desacuerdo (%d vs %d); conviene cerrarla canónicamente.", strings.TrimSpace(propuesta.Codigo), propuesta.Desacuerdo, propuesta.Acuerdo),
			Priority: "media",
			Assignee: "OpenClaw",
		})
	}
	return actions
}

func lowerLoadActiveWorker(activos []*db.Agente, cargaActiva, cargaReservada map[string]int, exclude string) (string, int, int) {
	exclude = strings.TrimSpace(exclude)
	bestName := ""
	bestLoad := 0
	sourceLoad := 0
	foundSource := false
	for _, agente := range activos {
		if agente == nil {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if nombre == "" {
			continue
		}
		total := cargaActiva[nombre] + cargaReservada[nombre]
		if strings.EqualFold(nombre, exclude) {
			sourceLoad = total
			foundSource = true
			continue
		}
		if bestName == "" || total < bestLoad || (total == bestLoad && nombre < bestName) {
			bestName = nombre
			bestLoad = total
		}
	}
	if !foundSource {
		sourceLoad = cargaActiva[exclude] + cargaReservada[exclude]
	}
	return bestName, sourceLoad, bestLoad
}

func countLibreTasks(status apiStatusResponse) int {
	if status.ResumenTareas != nil {
		if n, ok := status.ResumenTareas[string(db.TareaLibre)]; ok {
			return n
		}
	}
	if status.TareasPorEstado != nil {
		if n, ok := status.TareasPorEstado[string(db.TareaLibre)]; ok {
			return n
		}
	}
	return 0
}

func idleSupervisorWorkers(conectados, trabajando []*db.Agente) []string {
	conectados = visibleNonSupervisorAgents(conectados)
	trabajando = visibleNonSupervisorAgents(trabajando)
	trabajandoSet := make(map[string]struct{}, len(trabajando))
	for _, agente := range trabajando {
		if agente == nil {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if nombre != "" {
			trabajandoSet[nombre] = struct{}{}
		}
	}
	out := make([]string, 0, len(conectados))
	for _, agente := range conectados {
		if agente == nil {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if nombre == "" {
			continue
		}
		if _, busy := trabajandoSet[nombre]; busy {
			continue
		}
		out = append(out, nombre)
	}
	sort.Strings(out)
	return out
}

func firstIdleSupervisorWorker(idle []string) string {
	if len(idle) == 0 {
		return ""
	}
	return strings.TrimSpace(idle[0])
}

func preferredSupervisorWorker(status apiStatusResponse) string {
	workersActivos := visibleNonSupervisorAgents(status.AgentesActivos)
	workersTrabajando := visibleNonSupervisorAgents(status.AgentesTrabajando)
	idle := idleSupervisorWorkers(workersActivos, workersTrabajando)
	if len(idle) > 0 {
		return firstIdleSupervisorWorker(idle)
	}
	if len(workersActivos) == 0 {
		return ""
	}
	load := make(map[string]int, len(workersActivos))
	for _, agente := range workersActivos {
		if agente == nil {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if nombre != "" {
			load[nombre] = 0
		}
	}
	for _, tarea := range status.TareasActivas {
		nombre := strings.TrimSpace(tarea.Agente)
		if nombre == "" {
			continue
		}
		if _, ok := load[nombre]; !ok {
			continue
		}
		switch tarea.Estado {
		case db.TareaAsignada, db.TareaEnProgreso, db.TareaBloqueada:
			load[nombre]++
		}
	}
	best := ""
	bestLoad := 0
	for nombre, n := range load {
		if best == "" || n < bestLoad || (n == bestLoad && nombre < best) {
			best = nombre
			bestLoad = n
		}
	}
	return best
}

var supervisorWorkspaceProjectRiskSnapshotBuilder = supervisorWorkspaceProjectRiskSnapshot

func sortSupervisorRecommendedActions(actions []supervisorRecommendedAction) {
	projectRisk := supervisorWorkspaceProjectRiskSnapshotBuilder()
	sort.SliceStable(actions, func(i, j int) bool {
		wi, wj := supervisorActionPriorityWeight(actions[i].Priority), supervisorActionPriorityWeight(actions[j].Priority)
		if wi != wj {
			return wi < wj
		}
		gi, gj := supervisorActionGlobalImpactSortWeight(actions[i]), supervisorActionGlobalImpactSortWeight(actions[j])
		if gi != gj {
			return gi < gj
		}
		ri, rj := supervisorActionProjectRisk(actions[i], projectRisk), supervisorActionProjectRisk(actions[j], projectRisk)
		if ri != rj {
			return ri > rj
		}
		return actions[i].Target < actions[j].Target
	})
}

func sortSupervisorRecommendedActionsFast(actions []supervisorRecommendedAction) {
	sort.SliceStable(actions, func(i, j int) bool {
		wi, wj := supervisorActionPriorityWeight(actions[i].Priority), supervisorActionPriorityWeight(actions[j].Priority)
		if wi != wj {
			return wi < wj
		}
		gi, gj := supervisorActionGlobalImpactSortWeight(actions[i]), supervisorActionGlobalImpactSortWeight(actions[j])
		if gi != gj {
			return gi < gj
		}
		return actions[i].Target < actions[j].Target
	})
}

func supervisorActionPriorityWeight(v string) int {
	switch v {
	case "alta":
		return 0
	case "media":
		return 1
	default:
		return 2
	}
}

func supervisorWorkspaceProjectRiskSnapshot() map[string]int {
	report, err := buildWorkspaceControlReport()
	if err != nil || report == nil || len(report.AutonomyProjects) == 0 {
		return nil
	}
	out := make(map[string]int, len(report.AutonomyProjects))
	for _, item := range report.AutonomyProjects {
		project := strings.ToLower(strings.TrimSpace(item.Project))
		if project == "" || item.Blocking <= 0 {
			continue
		}
		out[project] = item.Blocking
	}
	return out
}

func supervisorActionProjectRisk(action supervisorRecommendedAction, projectRisk map[string]int) int {
	if len(projectRisk) == 0 {
		return 0
	}
	if supervisorActionIsHealthyAbsorbed(action) {
		return 0
	}
	project := strings.ToLower(strings.TrimSpace(supervisorCriticalAutonomyProjectFromTarget(action)))
	if project == "" {
		return 0
	}
	return projectRisk[project]
}

func supervisorActionGlobalImpactSortWeight(action supervisorRecommendedAction) int {
	if supervisorActionIsHealthyAbsorbed(action) {
		return 3
	}
	if strings.TrimSpace(action.Kind) != "autonomy_event" {
		return 2
	}
	if supervisorActionHasCriticalGlobalImpact(action) {
		return 0
	}
	if supervisorActionHasGlobalImpact(action) {
		return 1
	}
	return 2
}

func supervisorActionHasCriticalGlobalImpact(action supervisorRecommendedAction) bool {
	if supervisorActionIsHealthyAbsorbed(action) {
		return false
	}
	reason := strings.ToLower(strings.TrimSpace(action.Reason))
	if strings.Contains(reason, "bloquea integración/progreso global") {
		return true
	}
	return false
}

func supervisorActionHasGlobalImpact(action supervisorRecommendedAction) bool {
	if supervisorActionIsHealthyAbsorbed(action) {
		return false
	}
	reason := strings.ToLower(strings.TrimSpace(action.Reason))
	return strings.Contains(reason, "integración/progreso global")
}

func supervisorActionIsHealthyAbsorbed(action supervisorRecommendedAction) bool {
	reason := strings.ToLower(strings.TrimSpace(action.Reason))
	if !strings.Contains(reason, "ya hay progreso reciente") {
		return false
	}
	if !strings.Contains(reason, "no parece bloquear integración/progreso global") {
		return false
	}
	return true
}

type supervisorCriticalAutonomyFocus struct {
	supervisorRecommendedAction
	Project string
}

func buildSupervisorCriticalAutonomyFocus(status apiStatusResponse, surface *autonomySurface, mailboxPendiente []apiOpenClawMailboxLite) *supervisorCriticalAutonomyFocus {
	actions := buildSupervisorOperationalActions(status, mailboxPendiente)
	for i := range actions {
		if strings.TrimSpace(actions[i].Kind) != "autonomy_event" {
			continue
		}
		if supervisorActionHasGlobalImpact(actions[i]) {
			item := supervisorCriticalAutonomyFocus{supervisorRecommendedAction: actions[i]}
			item.Project = supervisorCriticalAutonomyProject(actions[i], surface)
			return &item
		}
	}
	return nil
}

func supervisorCriticalAutonomyProject(action supervisorRecommendedAction, surface *autonomySurface) string {
	if surface == nil {
		return supervisorCriticalAutonomyProjectFromTarget(action)
	}
	for _, item := range surface.Recent {
		candidate, ok := supervisorActionFromAutonomyEvent(item.autonomyEventSummary)
		if !ok {
			continue
		}
		if strings.TrimSpace(candidate.Action) == strings.TrimSpace(action.Action) &&
			strings.TrimSpace(candidate.Target) == strings.TrimSpace(action.Target) {
			return strings.TrimSpace(item.Project)
		}
	}
	return supervisorCriticalAutonomyProjectFromTarget(action)
}

func supervisorCriticalAutonomyProjectFromTarget(action supervisorRecommendedAction) string {
	taskID := parseSupervisorActionTargetID("tarea:", action.Target)
	if taskID == nil || *taskID <= 0 {
		return ""
	}
	task, err := db.GetTarea(*taskID)
	if err != nil || task == nil || task.ProyectoID == nil || *task.ProyectoID <= 0 {
		return ""
	}
	project, err := db.GetProyecto(strconv.FormatInt(*task.ProyectoID, 10))
	if err != nil || project == nil {
		return ""
	}
	return strings.TrimSpace(project.Slug)
}

func supervisorCriticalProjectRiskFromSnapshot(snapshot map[string]any) *workspaceAutonomyProjectSummary {
	if snapshot == nil {
		return nil
	}
	risk, _ := parseWorkspaceAutonomyProjectSummarySnapshotValue(snapshot["critical_project_risk"])
	return risk
}

func parseSupervisorRecommendedActionSnapshotValue(raw any) (*supervisorRecommendedAction, bool) {
	switch value := raw.(type) {
	case supervisorRecommendedAction:
		item := value
		return &item, true
	case *supervisorRecommendedAction:
		if value == nil {
			return nil, false
		}
		item := *value
		return &item, true
	case map[string]any:
		item := supervisorRecommendedAction{
			Kind:     strings.TrimSpace(fmt.Sprint(value["kind"])),
			Target:   strings.TrimSpace(fmt.Sprint(value["target"])),
			Action:   strings.TrimSpace(fmt.Sprint(value["action"])),
			Reason:   strings.TrimSpace(fmt.Sprint(value["reason"])),
			Priority: strings.TrimSpace(fmt.Sprint(value["priority"])),
			Assignee: strings.TrimSpace(fmt.Sprint(value["assignee"])),
		}
		if rawAuto, ok := value["auto_aplicable"]; ok {
			switch v := rawAuto.(type) {
			case bool:
				item.AutoAplicable = v
			case string:
				item.AutoAplicable = strings.EqualFold(strings.TrimSpace(v), "true")
			}
		}
		if item.Action == "" && item.Target == "" && item.Kind == "" {
			return nil, false
		}
		return &item, true
	default:
		return nil, false
	}
}

func parseWorkspaceAutonomyProjectSummarySnapshotValue(raw any) (*workspaceAutonomyProjectSummary, bool) {
	switch value := raw.(type) {
	case *workspaceAutonomyProjectSummary:
		if value == nil {
			return nil, false
		}
		item := *value
		item.Project = strings.TrimSpace(item.Project)
		item.Highlights = append([]string(nil), item.Highlights...)
		return &item, true
	case workspaceAutonomyProjectSummary:
		item := value
		item.Project = strings.TrimSpace(item.Project)
		item.Highlights = append([]string(nil), item.Highlights...)
		return &item, true
	case map[string]any:
		item := &workspaceAutonomyProjectSummary{
			Project: strings.TrimSpace(fmt.Sprint(value["project"])),
		}
		if rawEvents, ok := value["events"]; ok {
			switch v := rawEvents.(type) {
			case int:
				item.Events = v
			case int64:
				item.Events = int(v)
			case float64:
				item.Events = int(v)
			}
		}
		if rawBlocking, ok := value["blocking"]; ok {
			switch v := rawBlocking.(type) {
			case int:
				item.Blocking = v
			case int64:
				item.Blocking = int(v)
			case float64:
				item.Blocking = int(v)
			}
		}
		if rawLastAt, ok := value["last_at"]; ok {
			switch v := rawLastAt.(type) {
			case string:
				if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(v)); err == nil {
					item.LastAt = &parsed
				}
			case time.Time:
				parsed := v
				item.LastAt = &parsed
			}
		}
		if rawHighlights, ok := value["highlights"]; ok {
			switch v := rawHighlights.(type) {
			case []string:
				item.Highlights = append(item.Highlights, v...)
			case []any:
				for _, candidate := range v {
					value := strings.TrimSpace(fmt.Sprint(candidate))
					if value == "" {
						continue
					}
					item.Highlights = append(item.Highlights, value)
				}
			}
		}
		if item.Project == "" && item.Blocking <= 0 && len(item.Highlights) == 0 {
			return nil, false
		}
		return item, true
	default:
		return nil, false
	}
}

func attachSupervisorCriticalProjectRisk(result map[string]any, snapshot map[string]any) map[string]any {
	if result == nil {
		return nil
	}
	if risk := supervisorCriticalProjectRiskFromSnapshot(snapshot); risk != nil {
		result["critical_project_risk"] = risk
	}
	return result
}

func supervisorRecommendedActionEquals(a, b supervisorRecommendedAction) bool {
	return strings.TrimSpace(a.Kind) == strings.TrimSpace(b.Kind) &&
		strings.TrimSpace(a.Action) == strings.TrimSpace(b.Action) &&
		strings.TrimSpace(a.Target) == strings.TrimSpace(b.Target) &&
		strings.TrimSpace(a.Assignee) == strings.TrimSpace(b.Assignee)
}

func attachSupervisorQueueContext(result map[string]any, snapshot map[string]any, selected *supervisorRecommendedAction) map[string]any {
	if result == nil || snapshot == nil || selected == nil {
		return result
	}
	safeQueue, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
	for _, item := range safeQueue {
		if !supervisorRecommendedActionEquals(item, *selected) {
			continue
		}
		result["queue_kind"] = "safe"
		result["safe_action_queue"] = safeQueue
		if nextSafeAction, ok := snapshot["next_safe_action"].(supervisorRecommendedAction); ok {
			result["next_safe_action"] = nextSafeAction
		}
		return result
	}
	actionQueue, _ := snapshot["action_queue"].([]supervisorRecommendedAction)
	for _, item := range actionQueue {
		if !supervisorRecommendedActionEquals(item, *selected) {
			continue
		}
		result["queue_kind"] = "recommended"
		result["action_queue"] = actionQueue
		if nextAction, ok := snapshot["next_action"].(supervisorRecommendedAction); ok {
			result["next_action"] = nextAction
		}
		return result
	}
	return result
}

func applySupervisorRecommendedAction(supervisor, actionName, target, assignee string) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	snapshot, err := buildSupervisorReviewSnapshot(supervisor)
	if err != nil {
		return nil, err
	}
	actions, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
	selected := pickSupervisorRecommendedAction(actions, strings.TrimSpace(actionName), strings.TrimSpace(target), strings.TrimSpace(assignee))
	if selected == nil {
		selected = fallbackSupervisorRecommendedAction(strings.TrimSpace(actionName), strings.TrimSpace(target), strings.TrimSpace(assignee))
	}
	if selected == nil {
		return nil, fmt.Errorf("no se encontró una acción aplicable del supervisor")
	}
	worker := strings.TrimSpace(selected.Assignee)
	if worker == "" {
		worker = strings.TrimSpace(assignee)
	}
	if worker == "" {
		return nil, fmt.Errorf("la acción no tiene assignee sugerido ni explícito")
	}
	if strings.TrimSpace(selected.Action) == "asignar_tarea_libre" ||
		strings.TrimSpace(selected.Action) == "reservar_tarea_libre" ||
		strings.TrimSpace(selected.Action) == "replanificar_por_cuota" ||
		strings.TrimSpace(selected.Action) == "rebalancear_reserva" {
		if err := revalidarYVerificarAgenteDisponibleParaTrabajo(worker); err != nil {
			return nil, err
		}
	}

	switch strings.TrimSpace(selected.Action) {
	case "asignar_tarea_libre":
		taskID, err := resolveSupervisorActionTaskID(*selected)
		if err != nil {
			return nil, err
		}
		task, err := tareasService.Get(taskID)
		if err != nil {
			return nil, err
		}
		if task == nil {
			return nil, fmt.Errorf("tarea #%d no encontrada", taskID)
		}
		switch task.Estado {
		case db.TareaLibre, db.TareaBacklog:
			if err := tareasService.Take(taskID, worker); err != nil {
				return nil, err
			}
			if err := tareasService.Start(taskID, worker); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("la tarea #%d no está libre para dispatch", taskID)
		}
		result := map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"action":     selected,
			"task_id":    taskID,
			"assignee":   worker,
		}
		attachSupervisorQueueContext(result, snapshot, selected)
		return attachSupervisorCriticalProjectRisk(result, snapshot), nil
	case "reservar_tarea_libre":
		taskID, err := resolveSupervisorActionTaskID(*selected)
		if err != nil {
			return nil, err
		}
		task, err := tareasService.Get(taskID)
		if err != nil {
			return nil, err
		}
		if task == nil {
			return nil, fmt.Errorf("tarea #%d no encontrada", taskID)
		}
		switch task.Estado {
		case db.TareaLibre, db.TareaBacklog:
			if err := tareasService.Take(taskID, worker); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("la tarea #%d no está libre para reserva", taskID)
		}
		result := map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"action":     selected,
			"task_id":    taskID,
			"assignee":   worker,
		}
		attachSupervisorQueueContext(result, snapshot, selected)
		return attachSupervisorCriticalProjectRisk(result, snapshot), nil
	case "replanificar_por_cuota":
		taskID, err := resolveSupervisorActionTaskID(*selected)
		if err != nil {
			return nil, err
		}
		task, err := tareasService.Get(taskID)
		if err != nil {
			return nil, err
		}
		if task == nil {
			return nil, fmt.Errorf("tarea #%d no encontrada", taskID)
		}
		wasInProgress := task.Estado == db.TareaEnProgreso
		if err := tareasService.Reassign(taskID, worker); err != nil {
			return nil, err
		}
		if wasInProgress {
			if err := tareasService.Start(taskID, worker); err != nil {
				return nil, err
			}
		}
		result := map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"action":     selected,
			"task_id":    taskID,
			"assignee":   worker,
		}
		attachSupervisorQueueContext(result, snapshot, selected)
		return attachSupervisorCriticalProjectRisk(result, snapshot), nil
	case "rebalancear_reserva":
		taskID, err := resolveSupervisorActionTaskID(*selected)
		if err != nil {
			return nil, err
		}
		task, err := tareasService.Get(taskID)
		if err != nil {
			return nil, err
		}
		if task == nil {
			return nil, fmt.Errorf("tarea #%d no encontrada", taskID)
		}
		if task.Estado != db.TareaAsignada {
			return nil, fmt.Errorf("la tarea #%d no está reservada para rebalanceo", taskID)
		}
		if err := tareasService.Reassign(taskID, worker); err != nil {
			return nil, err
		}
		result := map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"action":     selected,
			"task_id":    taskID,
			"assignee":   worker,
		}
		attachSupervisorQueueContext(result, snapshot, selected)
		return attachSupervisorCriticalProjectRisk(result, snapshot), nil
	case "cerrar_propuesta_rechazada":
		codigo, err := resolveSupervisorActionProposalCode(*selected)
		if err != nil {
			return nil, err
		}
		if err := propuestasService.Close(codigo, string(db.PropuestaRechazada), supervisor); err != nil {
			return nil, err
		}
		result := map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"action":     selected,
			"codigo":     codigo,
			"assignee":   worker,
		}
		attachSupervisorQueueContext(result, snapshot, selected)
		return attachSupervisorCriticalProjectRisk(result, snapshot), nil
	case "seguir_guidance_durable":
		agenteObjetivo := strings.TrimSpace(worker)
		if strings.HasPrefix(strings.TrimSpace(selected.Target), "agente:") {
			agenteObjetivo = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(selected.Target), "agente:"))
		}
		if agenteObjetivo == "" {
			return nil, fmt.Errorf("target de agente inválido para guidance durable: %s", strings.TrimSpace(selected.Target))
		}
		estadoPendiente := "pendiente"
		filter := db.FiltroRuntimeMailbox{
			ToAgente: strPtr(strings.TrimSpace(apiNombreAgenteCanonico(agenteObjetivo))),
			Estado:   &estadoPendiente,
		}
		processedCount, err := apiRuntimeProcessMailboxExecutor(filter)
		if err != nil {
			return nil, err
		}
		remaining, err := db.ListarRuntimeMailbox(filter)
		if err != nil {
			return nil, err
		}
		pendingIDs := make([]int64, 0, len(remaining))
		for _, item := range remaining {
			if item == nil {
				continue
			}
			pendingIDs = append(pendingIDs, item.ID)
		}
		result := map[string]any{
			"ok":                    true,
			"supervisor":            supervisor,
			"action":                selected,
			"agente":                agenteObjetivo,
			"assignee":              worker,
			"processed_count":       processedCount,
			"mailbox_pending_after": len(pendingIDs),
			"pending_mailbox_ids":   pendingIDs,
			"guidance_resolved":     len(pendingIDs) == 0,
			"needs_followup":        len(pendingIDs) > 0,
		}
		attachSupervisorQueueContext(result, snapshot, selected)
		return attachSupervisorCriticalProjectRisk(result, snapshot), nil
	case "inspeccionar_sesion_observada":
		target := strings.TrimSpace(selected.Target)
		if !strings.HasPrefix(target, "agente:") {
			return nil, fmt.Errorf("target no soportado para sesión observada: %s", target)
		}
		agente := strings.TrimSpace(strings.TrimPrefix(target, "agente:"))
		if agente == "" {
			return nil, fmt.Errorf("target de agente inválido: %s", target)
		}
		snapshot, err := buildSupervisorThreadsSnapshot(supervisor, "", 100)
		if err != nil {
			return nil, err
		}
		candidates := buildOpenClawSessionCandidates(snapshot)
		for _, item := range candidates {
			if strings.EqualFold(strings.TrimSpace(item.Agente), agente) {
				return attachSupervisorCriticalProjectRisk(map[string]any{
					"ok":                true,
					"supervisor":        supervisor,
					"action":            selected,
					"agente":            agente,
					"assignee":          worker,
					"session_candidate": item,
				}, snapshot), nil
			}
		}
		return nil, fmt.Errorf("no hay sesión observada para %s", agente)
	case "revisar_worktree_desfasada":
		target := strings.TrimSpace(selected.Target)
		if !strings.HasPrefix(target, "agente:") {
			return nil, fmt.Errorf("target no soportado para worktree drift: %s", target)
		}
		agente := strings.TrimSpace(strings.TrimPrefix(target, "agente:"))
		if agente == "" {
			return nil, fmt.Errorf("target de agente inválido: %s", target)
		}
		status, err := statusService.FetchStatus()
		if err != nil {
			return nil, err
		}
		items, err := openClawWorktreeDriftBuilder(status)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if strings.EqualFold(strings.TrimSpace(item.Agente), agente) {
				result := map[string]any{
					"ok":                 true,
					"supervisor":         supervisor,
					"action":             selected,
					"agente":             agente,
					"assignee":           worker,
					"worktree_drift":     item,
					"checkpoint_request": buildSupervisorWorktreeCheckpointInstruction(item),
					"nudge_enqueued":     false,
				}
				enqueued, proyecto, err := requestSupervisorWorktreeCheckpoint(agente, item)
				if err == nil {
					result["nudge_enqueued"] = enqueued
					result["proyecto"] = proyecto
				} else {
					result["request_error"] = err.Error()
				}
				attachSupervisorQueueContext(result, snapshot, selected)
				return attachSupervisorCriticalProjectRisk(result, snapshot), nil
			}
		}
		return nil, fmt.Errorf("no hay worktree desfasada para %s", agente)
	case "refrescar_worktree_limpia":
		target := strings.TrimSpace(selected.Target)
		if !strings.HasPrefix(target, "agente:") {
			return nil, fmt.Errorf("target no soportado para refresco de worktree: %s", target)
		}
		agente := strings.TrimSpace(strings.TrimPrefix(target, "agente:"))
		if agente == "" {
			return nil, fmt.Errorf("target de agente inválido: %s", target)
		}
		status, err := statusService.FetchStatus()
		if err != nil {
			return nil, err
		}
		items, err := openClawWorktreeDriftBuilder(status)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if !strings.EqualFold(strings.TrimSpace(item.Agente), agente) {
				continue
			}
			if item.Dirty {
				return nil, fmt.Errorf("la worktree de %s no es limpia; requiere arbitraje manual", agente)
			}
			refreshed, err := supervisorRefreshCleanWorktreeDrift(item)
			if err != nil {
				return nil, err
			}
			return attachSupervisorCriticalProjectRisk(map[string]any{
				"ok":               true,
				"supervisor":       supervisor,
				"action":           selected,
				"agente":           agente,
				"assignee":         worker,
				"worktree_drift":   item,
				"worktree_refresh": refreshed,
			}, snapshot), nil
		}
		return nil, fmt.Errorf("no hay worktree limpia desfasada para %s", agente)
	case "recoger_resultado_subagente", "revisar_subagente_fallido", "limpiar_subagente_cancelado":
		subagentID, err := resolveSupervisorActionSubagentID(selected.Target)
		if err != nil {
			return nil, err
		}
		item, err := db.GetSupervisorSubagentByID(subagentID)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, fmt.Errorf("subagente #%d no encontrado", subagentID)
		}
		entregaGit, err := mcpRecogerResultadoGitSubagenteFn(item)
		if err != nil {
			return nil, err
		}
		pipelineSlice := metadataSupervisorSubagente(item)
		pipelineFollowup, err := maybePromoverPipelineTrasEntregaSidecar(item, entregaGit, pipelineSlice)
		if err != nil {
			return nil, err
		}
		out := map[string]any{
			"ok":             true,
			"supervisor":     supervisor,
			"action":         selected,
			"assignee":       worker,
			"subagente":      item,
			"entrega_git":    entregaGit,
			"pipeline_slice": pipelineSlice,
		}
		if pipelineFollowup != nil {
			out["pipeline_followup"] = pipelineFollowup
		}
		attachSupervisorQueueContext(out, snapshot, selected)
		return attachSupervisorCriticalProjectRisk(out, snapshot), nil
	case "refrescar_store_subagentes":
		storeSync, err := refreshSupervisorSubagentsFromStore(supervisor, "")
		if err != nil {
			return nil, err
		}
		result := map[string]any{
			"ok":         true,
			"supervisor": supervisor,
			"action":     selected,
			"assignee":   worker,
			"store_sync": storeSync,
		}
		attachSupervisorQueueContext(result, snapshot, selected)
		return attachSupervisorCriticalProjectRisk(result, snapshot), nil
	default:
		return nil, fmt.Errorf("acción no aplicable automáticamente: %s", strings.TrimSpace(selected.Action))
	}
}

func maybePromoverPipelineTrasEntregaSidecar(item *db.SupervisorSubagent, entregaGit map[string]any, pipelineSlice map[string]any) (*capacidadapp.ResultadoEjecucionPasoPipelineLocal, error) {
	if item == nil || entregaGit == nil || len(pipelineSlice) == 0 {
		return nil, nil
	}
	if !strings.EqualFold(strings.TrimSpace(stringSupervisorSubagente(pipelineSlice["source"])), "pipeline_local_parallel") {
		return nil, nil
	}
	if sidecarPipelineFollowupYaDespachado(pipelineSlice) {
		return nil, nil
	}
	proyectoSlug := strings.TrimSpace(item.ProyectoSlug)
	if proyectoSlug == "" || mcpPipelineDispatchFn == nil {
		return nil, nil
	}
	resultado, err := mcpPipelineDispatchFn(proyectoSlug)
	if err != nil {
		return nil, err
	}
	if err := persistirFollowupPipelineSidecar(item, pipelineSlice, entregaGit, resultado); err != nil {
		return nil, err
	}
	return resultado, nil
}

func sidecarPipelineFollowupYaDespachado(metadata map[string]any) bool {
	if len(metadata) == 0 {
		return false
	}
	switch v := metadata["pipeline_parent_followup_dispatched"].(type) {
	case bool:
		return v
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		return v == "true" || v == "1" || v == "yes"
	}
	return false
}

func persistirFollowupPipelineSidecar(item *db.SupervisorSubagent, metadata map[string]any, entregaGit map[string]any, resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal) error {
	if item == nil {
		return nil
	}
	next := make(map[string]any, len(metadata)+6)
	for k, v := range metadata {
		next[k] = v
	}
	next["pipeline_parent_followup_dispatched"] = true
	next["pipeline_parent_followup_dispatched_at"] = time.Now().UTC().Format(time.RFC3339)
	if mergeID := int64SupervisorSubagente(entregaGit["git_merge_id"]); mergeID > 0 {
		next["pipeline_parent_followup_git_merge_id"] = mergeID
	}
	if resultado != nil && resultado.Paso != nil {
		if fase := strings.TrimSpace(resultado.Paso.FaseObjetivo); fase != "" {
			next["pipeline_parent_followup_phase"] = fase
		}
		if accion := strings.TrimSpace(resultado.Paso.Accion); accion != "" {
			next["pipeline_parent_followup_action"] = accion
		}
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	_, err = db.UpsertSupervisorSubagent(db.UpsertSupervisorSubagentInput{
		Supervisor:      strings.TrimSpace(item.Supervisor),
		ProyectoSlug:    strings.TrimSpace(item.ProyectoSlug),
		SessionID:       strings.TrimSpace(item.SessionID),
		ParentThreadID:  strings.TrimSpace(item.ParentThreadID),
		ThreadID:        strings.TrimSpace(item.ThreadID),
		SubagentName:    strings.TrimSpace(item.SubagentName),
		SubagentType:    strings.TrimSpace(item.SubagentType),
		ToolProfileJSON: strings.TrimSpace(item.ToolProfileJSON),
		Status:          strings.TrimSpace(item.Status),
		ManifestPath:    strings.TrimSpace(item.ManifestPath),
		OutputPath:      strings.TrimSpace(item.OutputPath),
		ErrorMessage:    strings.TrimSpace(item.ErrorMessage),
		MetadataJSON:    string(raw),
		CreatedAt:       &item.CreatedAt,
		StartedAt:       &item.StartedAt,
		CompletedAt:     item.CompletedAt,
	})
	return err
}

func resolveSupervisorActionProjectForAgent(agente string) (*db.Proyecto, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil, fmt.Errorf("agente obligatorio")
	}
	if asignacion, err := db.GetAsignacionActivaAgente(agente); err == nil && asignacion != nil {
		return db.GetProyecto(strconv.FormatInt(asignacion.ProyectoID, 10))
	}
	projectSlug := strings.TrimSpace(configOrDefault("server_autobootstrap_project_slug", "orquestador"))
	if projectSlug == "" {
		projectSlug = "orquestador"
	}
	return db.GetProyecto(projectSlug)
}

func buildSupervisorWorktreeCheckpointInstruction(item apiOpenClawWorktreeDrift) string {
	parts := []string{
		"tu worktree esta desfasada respecto al HEAD del servidor",
	}
	if strings.TrimSpace(item.CurrentHead) != "" || strings.TrimSpace(item.ExpectedHead) != "" {
		parts = append(parts, fmt.Sprintf("base %s -> %s", strings.TrimSpace(item.CurrentHead), strings.TrimSpace(item.ExpectedHead)))
	}
	if strings.TrimSpace(item.DirtySummary) != "" {
		parts = append(parts, "estado local "+strings.TrimSpace(item.DirtySummary))
	}
	parts = append(parts, "haz checkpoint util, resume cambios locales y deja la worktree lista para refresco controlado")
	return strings.Join(parts, "; ")
}

func requestSupervisorWorktreeCheckpoint(agente string, item apiOpenClawWorktreeDrift) (bool, string, error) {
	proyecto, err := resolveSupervisorActionProjectForAgent(agente)
	if err != nil {
		return false, "", err
	}
	if proyecto == nil {
		return false, "", nil
	}
	instruction := buildSupervisorWorktreeCheckpointInstruction(item)
	enqueued, err := encolarNudgeAutonomiaDetallado(
		agente,
		proyecto,
		"continuar_trabajo",
		"worktree_desfasada",
		instruction,
		map[string]any{
			"worktree_drift": map[string]any{
				"branch":        strings.TrimSpace(item.Branch),
				"path":          strings.TrimSpace(item.Path),
				"current_head":  strings.TrimSpace(item.CurrentHead),
				"expected_head": strings.TrimSpace(item.ExpectedHead),
				"dirty":         item.Dirty,
				"dirty_summary": strings.TrimSpace(item.DirtySummary),
			},
			"supervisor_action": "revisar_worktree_desfasada",
		},
	)
	if err != nil {
		return false, "", err
	}
	return enqueued, strings.TrimSpace(proyecto.Slug), nil
}

func normalizeSupervisorBatchKind(kind string) string {
	switch strings.TrimSpace(strings.ToLower(kind)) {
	case "proposal", "dispatch", "guidance", "worktree_drift":
		return strings.TrimSpace(strings.ToLower(kind))
	case "mailbox_pending":
		return "guidance"
	default:
		return ""
	}
}

func supervisorActionMatchesBatchKind(action supervisorRecommendedAction, batchKind string) bool {
	batchKind = normalizeSupervisorBatchKind(batchKind)
	if batchKind == "" {
		return true
	}
	return normalizeSupervisorBatchKind(action.Kind) == batchKind
}

func applySupervisorRecommendedActionsBatch(supervisor string, maxItems int, batchKind string) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	snapshot, err := buildSupervisorReviewSnapshot(supervisor)
	if err != nil {
		return nil, err
	}
	actions, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
	criticalProjectRisk, _ := snapshot["critical_project_risk"].(*workspaceAutonomyProjectSummary)
	batchKind = normalizeSupervisorBatchKind(batchKind)
	if batchKind != "" {
		filtered := make([]supervisorRecommendedAction, 0, len(actions))
		for _, item := range actions {
			if !supervisorActionMatchesBatchKind(item, batchKind) {
				continue
			}
			filtered = append(filtered, item)
		}
		actions = filtered
	}
	if maxItems <= 0 || maxItems > len(actions) {
		maxItems = len(actions)
	}
	var nextSafeAction any
	if len(actions) > 0 {
		nextSafeAction = actions[0]
	}
	applied := make([]map[string]any, 0, maxItems)
	for _, item := range actions {
		if len(applied) >= maxItems {
			break
		}
		result, err := applySupervisorRecommendedAction(supervisor, item.Action, item.Target, item.Assignee)
		if err != nil {
			return nil, err
		}
		applied = append(applied, result)
	}
	return map[string]any{
		"ok":                    true,
		"supervisor":            supervisor,
		"count":                 len(applied),
		"applied":               applied,
		"kind":                  batchKind,
		"queue_kind":            "safe",
		"safe_action_queue":     actions,
		"next_safe_action":      nextSafeAction,
		"critical_project_risk": criticalProjectRisk,
	}, nil
}

func applySupervisorNextAction(supervisor string) (map[string]any, error) {
	supervisor = resolveSupervisorName(supervisor)
	snapshot, err := buildSupervisorReviewSnapshot(supervisor)
	if err != nil {
		return nil, err
	}
	actions, _ := snapshot["safe_action_queue"].([]supervisorRecommendedAction)
	for _, item := range actions {
		result, err := applySupervisorRecommendedAction(supervisor, item.Action, item.Target, item.Assignee)
		if err != nil {
			return nil, err
		}
		result["queue_kind"] = "safe"
		result["next_safe_action"] = item
		attachSupervisorCriticalProjectRisk(result, snapshot)
		return result, nil
	}
	return nil, fmt.Errorf("no hay acción segura aplicable en la cola del supervisor")
}

func pickSupervisorRecommendedAction(actions []supervisorRecommendedAction, actionName, target, assignee string) *supervisorRecommendedAction {
	actionName = strings.TrimSpace(actionName)
	target = strings.TrimSpace(target)
	assignee = strings.TrimSpace(assignee)
	for _, item := range actions {
		if actionName != "" && strings.TrimSpace(item.Action) != actionName {
			continue
		}
		if target != "" && strings.TrimSpace(item.Target) != target {
			continue
		}
		if assignee != "" && strings.TrimSpace(item.Assignee) != assignee {
			continue
		}
		copy := item
		return &copy
	}
	if actionName == "" && target == "" && assignee == "" && len(actions) > 0 {
		copy := actions[0]
		return &copy
	}
	return nil
}

func fallbackSupervisorRecommendedAction(actionName, target, assignee string) *supervisorRecommendedAction {
	if actionName == "" || target == "" {
		return nil
	}
	switch actionName {
	case "asignar_tarea_libre":
		return &supervisorRecommendedAction{
			Kind:     "dispatch",
			Action:   actionName,
			Target:   target,
			Priority: "media",
			Assignee: assignee,
		}
	case "replanificar_por_cuota":
		return &supervisorRecommendedAction{
			Kind:     "quota_hold",
			Action:   actionName,
			Target:   target,
			Priority: "media",
			Assignee: assignee,
		}
	case "reservar_tarea_libre":
		return &supervisorRecommendedAction{
			Kind:     "dispatch",
			Action:   actionName,
			Target:   target,
			Priority: "baja",
			Assignee: assignee,
		}
	case "rebalancear_reserva":
		return &supervisorRecommendedAction{
			Kind:     "dispatch",
			Action:   actionName,
			Target:   target,
			Priority: "media",
			Assignee: assignee,
		}
	case "cerrar_propuesta_rechazada":
		return &supervisorRecommendedAction{
			Kind:     "proposal",
			Action:   actionName,
			Target:   target,
			Priority: "media",
			Assignee: assignee,
		}
	case "seguir_guidance_durable":
		return &supervisorRecommendedAction{
			Kind:     "mailbox_pending",
			Action:   actionName,
			Target:   target,
			Priority: "media",
			Assignee: assignee,
		}
	case "inspeccionar_sesion_observada":
		return &supervisorRecommendedAction{
			Kind:     "session_candidate",
			Action:   actionName,
			Target:   target,
			Priority: "baja",
			Assignee: assignee,
		}
	case "revisar_worktree_desfasada":
		return &supervisorRecommendedAction{
			Kind:     "worktree_drift",
			Action:   actionName,
			Target:   target,
			Priority: "alta",
			Assignee: assignee,
		}
	case "refrescar_worktree_limpia":
		return &supervisorRecommendedAction{
			Kind:     "worktree_drift",
			Action:   actionName,
			Target:   target,
			Priority: "alta",
			Assignee: assignee,
		}
	case "recoger_resultado_subagente":
		return &supervisorRecommendedAction{
			Kind:     "subagent",
			Action:   actionName,
			Target:   target,
			Priority: "media",
			Assignee: assignee,
		}
	case "revisar_subagente_fallido":
		return &supervisorRecommendedAction{
			Kind:     "subagent",
			Action:   actionName,
			Target:   target,
			Priority: "alta",
			Assignee: assignee,
		}
	case "limpiar_subagente_cancelado":
		return &supervisorRecommendedAction{
			Kind:     "subagent",
			Action:   actionName,
			Target:   target,
			Priority: "baja",
			Assignee: assignee,
		}
	case "refrescar_store_subagentes":
		return &supervisorRecommendedAction{
			Kind:     "subagent_store",
			Action:   actionName,
			Target:   target,
			Priority: "media",
			Assignee: assignee,
		}
	default:
		return nil
	}
}

func resolveSupervisorActionTaskID(action supervisorRecommendedAction) (int64, error) {
	target := strings.TrimSpace(action.Target)
	switch {
	case strings.HasPrefix(target, "tarea:"):
		id, err := strconv.ParseInt(strings.TrimPrefix(target, "tarea:"), 10, 64)
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("target de tarea inválido: %s", target)
		}
		return id, nil
	case target == "backlog:libre":
		libres, err := db.ListarTareas(db.FiltroTareas{Libre: true})
		if err != nil {
			return 0, err
		}
		if len(libres) == 0 || libres[0] == nil {
			return 0, fmt.Errorf("no hay tareas libres para dispatch")
		}
		return libres[0].ID, nil
	default:
		return 0, fmt.Errorf("target no soportado para aplicación automática: %s", target)
	}
}

func resolveSupervisorActionProposalCode(action supervisorRecommendedAction) (string, error) {
	target := strings.TrimSpace(action.Target)
	if strings.HasPrefix(target, "propuesta:") {
		target = strings.TrimPrefix(target, "propuesta:")
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("target de propuesta inválido: %q", action.Target)
	}
	return target, nil
}

func resolveSupervisorActionMailboxID(action supervisorRecommendedAction, agente string) (int64, error) {
	agente = strings.TrimSpace(agente)
	target := strings.TrimSpace(action.Target)
	if !strings.HasPrefix(target, "agente:") {
		return 0, fmt.Errorf("target no soportado para mailbox: %s", target)
	}
	targetAgente := strings.TrimSpace(strings.TrimPrefix(target, "agente:"))
	if targetAgente == "" {
		return 0, fmt.Errorf("target de agente inválido: %s", target)
	}
	if agente != "" && !strings.EqualFold(targetAgente, agente) {
		return 0, fmt.Errorf("assignee %s no coincide con target %s", agente, targetAgente)
	}
	estado := "pendiente"
	rows, err := db.ListarRuntimeMailbox(db.FiltroRuntimeMailbox{ToAgente: &targetAgente, Estado: &estado})
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 || rows[0] == nil {
		return 0, fmt.Errorf("no hay mailbox pendiente para %s", targetAgente)
	}
	return rows[0].ID, nil
}

func compactMCPLine(text string, maxRunes int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.Join(strings.Fields(text), " ")
	if maxRunes <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
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
- El nucleo de Orquesta no debe depender de MCP, del backend local heredado ni de un proveedor concreto.
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

func boolArgOrFalse(args map[string]any, key string) bool {
	v, _ := boolArg(args, key)
	return v
}

func optionalIntArg(args map[string]any, key string) int {
	if args == nil {
		return 0
	}
	v, ok := args[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		value, _ := n.Int64()
		return int(value)
	case string:
		value, _ := strconv.Atoi(strings.TrimSpace(n))
		return value
	default:
		return 0
	}
}

func optionalInt64Arg(args map[string]any, key string) int64 {
	if args == nil {
		return 0
	}
	v, ok := args[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		value, _ := n.Int64()
		return value
	case string:
		value, _ := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		return value
	default:
		return 0
	}
}

func optionalInt64PtrArg(args map[string]any, key string) *int64 {
	value := optionalInt64Arg(args, key)
	if value <= 0 {
		return nil
	}
	return &value
}

func optionalInt64SliceArg(args map[string]any, key string) []int64 {
	if args == nil {
		return nil
	}
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]int64, 0, len(items))
	for _, item := range items {
		switch n := item.(type) {
		case float64:
			out = append(out, int64(n))
		case int:
			out = append(out, int64(n))
		case int64:
			out = append(out, n)
		case json.Number:
			if value, err := n.Int64(); err == nil {
				out = append(out, value)
			}
		case string:
			if value, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64); err == nil {
				out = append(out, value)
			}
		}
	}
	return out
}

func stringPtrOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
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
