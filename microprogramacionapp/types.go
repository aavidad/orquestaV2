package microprogramacionapp

import "time"

type EstadoEspecificacion string

const (
	EstadoEspecificacionBorrador    EstadoEspecificacion = "borrador"
	EstadoEspecificacionActiva      EstadoEspecificacion = "activa"
	EstadoEspecificacionReemplazada EstadoEspecificacion = "reemplazada"
	EstadoEspecificacionArchivada   EstadoEspecificacion = "archivada"
)

type EspecificacionFuncion struct {
	ID                     int64
	TareaID                *int64
	ProyectoID             *int64
	Titulo                 string
	ArchivoObjetivo        string
	SimboloObjetivo        string
	Descripcion            string
	Precondiciones         []string
	Postcondiciones        []string
	DependenciasPermitidas []string
	DependenciasProhibidas []string
	TestsObligatorios      []string
	WriteSet               []string
	FormatoSalida          string
	Estado                 EstadoEspecificacion
	Version                int
	CreadoPor              string
	CreadaAt               time.Time
	ActualizadaAt          time.Time
}

type EntradaCrearEspecificacion struct {
	TareaID                *int64
	ProyectoID             *int64
	Titulo                 string
	ArchivoObjetivo        string
	SimboloObjetivo        string
	Descripcion            string
	Precondiciones         []string
	Postcondiciones        []string
	DependenciasPermitidas []string
	DependenciasProhibidas []string
	TestsObligatorios      []string
	WriteSet               []string
	FormatoSalida          string
	CreadoPor              string
}

type FiltroEspecificaciones struct {
	TareaID    *int64
	ProyectoID *int64
	Estado     *EstadoEspecificacion
	Limit      int
}

type EntradaEmitirMicrotarea struct {
	Contexto string
}

type MicrotareaEmitida struct {
	EspecificacionID  int64    `json:"especificacion_id"`
	Titulo            string   `json:"titulo"`
	ArchivoObjetivo   string   `json:"archivo_objetivo"`
	SimboloObjetivo   string   `json:"simbolo_objetivo"`
	WriteSet          []string `json:"write_set"`
	TestsObligatorios []string `json:"tests_obligatorios"`
	FormatoSalida     string   `json:"formato_salida"`
	Mensaje           string   `json:"mensaje"`
}

type EntradaDespacharMicrotarea struct {
	AgenteDestino string
	ProyectoID    *int64
	Contexto      string
}

type SolicitudDespachoMicrotarea struct {
	EspecificacionID int64
	ProyectoID       *int64
	AgenteDestino    string
	Microtarea       *MicrotareaEmitida
}

type MicrotareaDespachada struct {
	EspecificacionID int64              `json:"especificacion_id"`
	ProyectoID       *int64             `json:"proyecto_id,omitempty"`
	AgenteDestino    string             `json:"agente_destino"`
	RuntimeOrderID   int64              `json:"runtime_order_id"`
	Microtarea       *MicrotareaEmitida `json:"microtarea,omitempty"`
}

type EntradaValidarEntrega struct {
	SimboloEntregado   string
	WriteSetEntregado  []string
	DependenciasUsadas []string
	TestsEjecutados    []string
	TestsFallidos      []string
	Evidencia          string
}

type ArchivoEntrega struct {
	RutaRelativa string
	Contenido    string
}

type EntradaMaterializarEntrega struct {
	RaizProyecto string
	Archivos     []ArchivoEntrega
	Evidencia    string
}

type ResultadoMaterializarEntrega struct {
	EspecificacionID       int64
	ArchivoObjetivo        string
	WriteSetPermitido      []string
	ArchivosMaterializados []string
}

type EntregaGitCapturada struct {
	ProyectoSlug        string
	WorktreeID          int64
	RutaWorktree        string
	Branch              string
	BaseRef             string
	HeadCommit          string
	ArchivosModificados []string
	Diff                string
}

type EntradaRegistrarEntregaGit struct {
	Agente                  string
	ProyectoID              *int64
	ProyectoSlug            string
	SolicitadoPor           string
	Evidencia               string
	PreferenciaWorktreeID   *int64
	PreferenciaRutaWorktree string
	PreferenciaBranch       string
	PreferenciaBaseRef      string
}

type ResultadoRegistrarEntregaGit struct {
	EspecificacionID   int64
	ArchivoObjetivo    string
	WriteSetPermitido  []string
	WorktreeID         int64
	RutaWorktree       string
	SourceBranch       string
	TargetBranch       string
	HeadCommit         string
	ArchivosEntregados []string
	GitMergeID         int64
}

type HallazgoValidacionEntrega struct {
	Campo   string
	Codigo  string
	Mensaje string
}

type ResultadoValidacionEntrega struct {
	EspecificacionID       int64
	Valida                 bool
	ArchivoObjetivo        string
	SimboloObjetivo        string
	WriteSetPermitido      []string
	DependenciasPermitidas []string
	DependenciasProhibidas []string
	TestsObligatorios      []string
	Hallazgos              []HallazgoValidacionEntrega
}
