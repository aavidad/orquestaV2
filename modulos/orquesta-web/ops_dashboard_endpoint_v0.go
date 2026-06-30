package orquestaweb

import (
	"net/http"
	"strings"
)

const WebOpsDashboardPageEndpointV0 = "/ops"

type OpsDashboardWebEndpointV0 struct{}

func NewOpsDashboardWebEndpointV0() OpsDashboardWebEndpointV0 {
	return OpsDashboardWebEndpointV0{}
}

func (endpoint OpsDashboardWebEndpointV0) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handleWebPublicHTTPOptionsV0(w, r, http.MethodGet) {
		return
	}
	if r.Method != http.MethodGet {
		setWebPublicHTTPAllowV0(w, http.MethodGet)
		http.Error(w, "metodo no permitido", http.StatusMethodNotAllowed)
		return
	}
	locale := opsDashboardLocaleFromRequestV0(r)
	writeWebHTMLStringResponseV0(w, http.StatusOK, opsDashboardHTMLForLocaleV0(locale), locale)
}

func opsDashboardHTMLV0() string {
	return opsDashboardHTMLForLocaleV0("es")
}

func opsDashboardHTMLForLocaleV0(locale string) string {
	html := strings.TrimSpace(strings.Join([]string{
		opsDashboardHTMLChunk0V0,
		opsDashboardHTMLChunk1V0,
		opsDashboardHTMLChunk2V0,
		opsDashboardHTMLChunk3V0,
		opsDashboardHTMLChunk4V0,
		opsDashboardHTMLChunk5V0,
	}, "\n")) + "\n"
	if opsDashboardNormalizeLocaleV0(locale) == "en" {
		return opsDashboardEnglishReplacerV0().Replace(html)
	}
	return html
}

func opsDashboardLocaleFromRequestV0(r *http.Request) string {
	if r == nil {
		return "es"
	}
	return opsDashboardNormalizeLocaleV0(r.URL.Query().Get("lang"))
}

func opsDashboardNormalizeLocaleV0(locale string) string {
	normalized := strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(normalized, "en") {
		return "en"
	}
	return "es"
}

func opsDashboardEnglishReplacerV0() *strings.Replacer {
	return strings.NewReplacer(
		"Panel operativo live de servidor, automejora, cola, proyectos y agentes.", "Live operations panel for server, self-improvement, queue, projects, and agents.",
		"conectando", "connecting",
		"refresco", "refresh",
		"última lectura", "last read",
		"Inicio", "Home",
		"Nueva app", "New app",
		"Autoprogramar", "Self-program",
		"Cambio", "Change",
		"Actualizar", "Refresh",
		"indicadores", "indicators",
		"Servidor", "Server",
		"Proyectos activos", "Active projects",
		"Tareas en cola", "Queued tasks",
		"Agentes activos", "Active agents",
		"Memoria proceso", "Process memory",
		"Disco peor uso", "Worst disk use",
		"flujo operativo", "operational flow",
		"Cola", "Queue",
		"Ejecución", "Execution",
		"Atención", "Attention",
		"Cierre", "Closure",
		"director autónomo", "autonomous director",
		"Director", "Director",
		"Esperando datos", "Waiting for data",
		"Aún no hay snapshot operativo.", "No operational snapshot yet.",
		"Acción segura", "Safe action",
		"Proyectos / runs", "Projects / runs",
		"filtros de proyectos", "project filters",
		"Filtrar por tarea, run o proyecto", "Filter by task, run, or project",
		"Estado", "Status",
		"Todos", "All",
		"Activas", "Active",
		"Bloqueadas", "Blocked",
		"Completadas", "Completed",
		"Validación", "Validation",
		"Validada", "Validated",
		"Pendiente", "Pending",
		"Bloqueada", "Blocked",
		"Tarea", "Task",
		"Run", "Run",
		"Progreso", "Progress",
		"Agentes", "Agents",
		"Sin datos todavía", "No data yet",
		"Fases comparadas", "Compared phases",
		"esquema de fases", "phase diagram",
		"Sin fases observadas", "No phases observed",
		"Fase", "Phase",
		"Runs", "Runs",
		"Progreso medio", "Average progress",
		"Tareas", "Tasks",
		"Uso comparado", "Compared usage",
		"Cuota", "Quota",
		"Tokens", "Tokens",
		"Sin uso publicado por stats", "No usage published by stats",
		"Capacidad / uso", "Capacity / usage",
		"Filtrar por agente, run o tarea", "Filter by agent, run, or task",
		"Detalle y control", "Detail and control",
		"Admin", "Admin",
		"Fuentes", "Sources",
		"Variables pendientes para reinicio", "Variables pending restart",
		"Quitar de cola", "Remove from queue",
		"Observar goal", "Observe goal",
		"Avanzar run", "Advance run",
	)
}
