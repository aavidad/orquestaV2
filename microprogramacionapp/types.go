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
	EspecificacionID  int64
	Titulo            string
	ArchivoObjetivo   string
	SimboloObjetivo   string
	WriteSet          []string
	TestsObligatorios []string
	FormatoSalida     string
	Mensaje           string
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
	EspecificacionID int64
	ProyectoID       *int64
	AgenteDestino    string
	RuntimeOrderID   int64
	Microtarea       *MicrotareaEmitida
}

type EntradaValidarEntrega struct {
	SimboloEntregado   string
	WriteSetEntregado  []string
	DependenciasUsadas []string
	TestsEjecutados    []string
	TestsFallidos      []string
	Evidencia          string
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
