// Package microvm ofrece a Orquesta un cliente del protocolo local de Agente MicroVM.
package microvm

import "encoding/json"

const (
	// ProtocoloLocal identifica la única versión negociada por este conector.
	ProtocoloLocal = "agentmicrovm.local.v1"
	// CabeceraProtocolo evita enviar una mutación a un servidor incompatible.
	CabeceraProtocolo = "x-agentmicrovm-protocolo"
	// CabeceraIdempotencia cerca cada mutación con una clave estable del consumidor.
	CabeceraIdempotencia = "idempotency-key"

	MaximoEntradaSesionBytesV1      = 1_048_576
	MaximoEventosPaginaSesionV1     = 256
	MaximoEventosSesionBytesV1      = 8 * 1_048_576
	MaximoPlazoSesionMilisegundosV1 = 86_400_000
)

type RespuestaSalud struct {
	Estado    string `json:"estado"`
	Protocolo string `json:"protocolo"`
	Version   string `json:"version"`
}

type RespuestaCapacidades struct {
	Protocolo              string   `json:"protocolo"`
	Version                string   `json:"version"`
	Operaciones            []string `json:"operaciones"`
	KVMDisponible          bool     `json:"kvm_disponible"`
	FirecrackerConfigurado bool     `json:"firecracker_configurado"`
	FirecrackerEjecutable  bool     `json:"firecracker_ejecutable"`
	MaximoEjecuciones      uint32   `json:"maximo_ejecuciones"`
}

// SolicitudLanzamiento conserva los contratos firmados sin duplicar su schema en Go.
type SolicitudLanzamiento struct {
	Plan      json.RawMessage `json:"plan"`
	Concesion json.RawMessage `json:"concesion"`
}

type SolicitudDetencion struct {
	RevisionEsperada uint64 `json:"revision_esperada"`
	Cerca            uint64 `json:"cerca"`
}

type SolicitudPreservacion struct {
	RevisionEsperada uint64 `json:"revision_esperada"`
	Cerca            uint64 `json:"cerca"`
}

type SolicitudCierre struct {
	RevisionEsperada         uint64 `json:"revision_esperada"`
	Cerca                    uint64 `json:"cerca"`
	ManifiestoSHA256Esperado string `json:"manifiesto_sha256_esperado"`
}

type VariableEntorno struct {
	Nombre string `json:"nombre"`
	Valor  string `json:"valor"`
}

type SolicitudOrden struct {
	RevisionEsperada  uint64            `json:"revision_esperada"`
	Cerca             uint64            `json:"cerca"`
	Programa          string            `json:"programa"`
	Argumentos        []string          `json:"argumentos"`
	DirectorioTrabajo string            `json:"directorio_trabajo"`
	Entorno           []VariableEntorno `json:"entorno"`
	StdinBase64       string            `json:"stdin_base64"`
	PlazoMilisegundos uint64            `json:"plazo_milisegundos"`
	MaximoSalidaBytes uint32            `json:"maximo_salida_bytes"`
}

type ArchivoContenido struct {
	RutaRelativa    string `json:"ruta_relativa"`
	Ejecutable      bool   `json:"ejecutable"`
	SHA256          string `json:"sha256"`
	ContenidoBase64 string `json:"contenido_base64"`
}

type SolicitudSincronizacionEntrada struct {
	RevisionEsperada        uint64             `json:"revision_esperada"`
	RevisionTrabajoEsperada uint64             `json:"revision_trabajo_esperada"`
	Cerca                   uint64             `json:"cerca"`
	DestinoRelativo         string             `json:"destino_relativo"`
	RaizSHA256              string             `json:"raiz_sha256"`
	Archivos                []ArchivoContenido `json:"archivos"`
}

type SolicitudSincronizacionSalida struct {
	RevisionEsperada        uint64 `json:"revision_esperada"`
	RevisionTrabajoEsperada uint64 `json:"revision_trabajo_esperada"`
	Cerca                   uint64 `json:"cerca"`
	OrigenRelativo          string `json:"origen_relativo"`
	MaximoArchivos          uint32 `json:"maximo_archivos"`
	MaximoBytes             uint64 `json:"maximo_bytes"`
}

type IdentidadProceso struct {
	PID         uint32 `json:"pid"`
	InicioTicks uint64 `json:"inicio_ticks"`
}

type RespuestaEjecucion struct {
	Referencia      string            `json:"referencia"`
	Estado          string            `json:"estado"`
	Revision        uint64            `json:"revision"`
	Cerca           uint64            `json:"cerca"`
	RevisionTrabajo *uint64           `json:"revision_trabajo,omitempty"`
	VCPU            uint8             `json:"vcpu"`
	MemoriaMiB      uint32            `json:"memoria_mib"`
	Identidad       *IdentidadProceso `json:"identidad"`
	ProcesoVivo     *bool             `json:"proceso_vivo"`
	EstadoMotor     *string           `json:"estado_motor"`
}

type RespuestaOrden struct {
	Referencia     string `json:"referencia"`
	CodigoSalida   *int32 `json:"codigo_salida"`
	Senal          *int32 `json:"senal"`
	Agotada        bool   `json:"agotada"`
	StdoutBase64   string `json:"stdout_base64"`
	StderrBase64   string `json:"stderr_base64"`
	SalidaTruncada bool   `json:"salida_truncada"`
}

type RespuestaSincronizacion struct {
	Referencia      string `json:"referencia"`
	Direccion       string `json:"direccion"`
	RutaRelativa    string `json:"ruta_relativa"`
	RaizSHA256      string `json:"raiz_sha256"`
	Archivos        uint32 `json:"archivos"`
	Bytes           uint64 `json:"bytes"`
	RevisionTrabajo uint64 `json:"revision_trabajo"`
}

type RespuestaSincronizacionSalida struct {
	Sincronizacion RespuestaSincronizacion `json:"sincronizacion"`
	Archivos       []ArchivoContenido      `json:"archivos"`
}

type RespuestaRevisionTrabajo struct {
	Referencia      string `json:"referencia"`
	RevisionTrabajo uint64 `json:"revision_trabajo"`
}

// EstadoSesionTrabajoV1 describe el trabajo alojado, no el lifecycle físico.
type EstadoSesionTrabajoV1 string

const (
	EstadoSesionAdmitida   EstadoSesionTrabajoV1 = "admitida"
	EstadoSesionIniciando  EstadoSesionTrabajoV1 = "iniciando"
	EstadoSesionActiva     EstadoSesionTrabajoV1 = "activa"
	EstadoSesionAmbigua    EstadoSesionTrabajoV1 = "ambigua"
	EstadoSesionFinalizada EstadoSesionTrabajoV1 = "finalizada"
	EstadoSesionFallida    EstadoSesionTrabajoV1 = "fallida"
)

// TipoEventoSesionTrabajoV1 es la clase estable de un evento durable.
type TipoEventoSesionTrabajoV1 string

const (
	TipoEventoSesionIniciada       TipoEventoSesionTrabajoV1 = "iniciada"
	TipoEventoSesionStdout         TipoEventoSesionTrabajoV1 = "stdout"
	TipoEventoSesionStderr         TipoEventoSesionTrabajoV1 = "stderr"
	TipoEventoSesionSalidaTruncada TipoEventoSesionTrabajoV1 = "salida_truncada"
	TipoEventoSesionFinalizada     TipoEventoSesionTrabajoV1 = "finalizada"
)

type SolicitudIniciarSesionTrabajoV1 struct {
	SesionRef               string `json:"sesion_ref"`
	RevisionEsperada        uint64 `json:"revision_esperada"`
	RevisionTrabajoEsperada uint64 `json:"revision_trabajo_esperada"`
	Cerca                   uint64 `json:"cerca"`
	EjecutorRef             string `json:"ejecutor_ref"`
	DirectorioTrabajo       string `json:"directorio_trabajo"`
	PlazoTotalMilisegundos  uint64 `json:"plazo_total_milisegundos"`
	MaximoEventosBytes      uint64 `json:"maximo_eventos_bytes"`
}

type SolicitudEntradaSesionTrabajoV1 struct {
	RevisionEsperada        uint64 `json:"revision_esperada"`
	RevisionTrabajoEsperada uint64 `json:"revision_trabajo_esperada"`
	RevisionSesionEsperada  uint64 `json:"revision_sesion_esperada"`
	Cerca                   uint64 `json:"cerca"`
	ContenidoBase64         string `json:"contenido_base64"`
	CerrarStdin             bool   `json:"cerrar_stdin"`
}

type ConsultaEventosSesionTrabajoV1 struct {
	Cerca         uint64 `json:"cerca"`
	DespuesDe     uint64 `json:"despues_de"`
	MaximoEventos uint16 `json:"maximo_eventos"`
}

type ResultadoTerminalSesionTrabajoV1 struct {
	CodigoSalida   *int32  `json:"codigo_salida"`
	Senal          *int32  `json:"senal"`
	Agotada        bool    `json:"agotada"`
	SalidaTruncada bool    `json:"salida_truncada"`
	CodigoError    *string `json:"codigo_error"`
}

type RespuestaSesionTrabajoV1 struct {
	EjecucionRef    string                            `json:"ejecucion_ref"`
	SesionRef       string                            `json:"sesion_ref"`
	Estado          EstadoSesionTrabajoV1             `json:"estado"`
	Revision        uint64                            `json:"revision"`
	RevisionTrabajo uint64                            `json:"revision_trabajo"`
	RevisionSesion  uint64                            `json:"revision_sesion"`
	Cerca           uint64                            `json:"cerca"`
	Terminal        bool                              `json:"terminal"`
	Resultado       *ResultadoTerminalSesionTrabajoV1 `json:"resultado"`
}

type EventoSesionTrabajoV1 struct {
	SesionRef       string                            `json:"sesion_ref"`
	Secuencia       uint64                            `json:"secuencia"`
	Tipo            TipoEventoSesionTrabajoV1         `json:"tipo"`
	Estado          EstadoSesionTrabajoV1             `json:"estado"`
	Revision        uint64                            `json:"revision"`
	RevisionTrabajo uint64                            `json:"revision_trabajo"`
	RevisionSesion  uint64                            `json:"revision_sesion"`
	Cerca           uint64                            `json:"cerca"`
	ContenidoBase64 string                            `json:"contenido_base64"`
	Terminal        bool                              `json:"terminal"`
	Resultado       *ResultadoTerminalSesionTrabajoV1 `json:"resultado"`
}

type PaginaEventosSesionTrabajoV1 struct {
	EjecucionRef    string                  `json:"ejecucion_ref"`
	SesionRef       string                  `json:"sesion_ref"`
	Estado          EstadoSesionTrabajoV1   `json:"estado"`
	Revision        uint64                  `json:"revision"`
	RevisionTrabajo uint64                  `json:"revision_trabajo"`
	RevisionSesion  uint64                  `json:"revision_sesion"`
	Cerca           uint64                  `json:"cerca"`
	DespuesDe       uint64                  `json:"despues_de"`
	Eventos         []EventoSesionTrabajoV1 `json:"eventos"`
	SiguienteCursor uint64                  `json:"siguiente_cursor"`
	Terminal        bool                    `json:"terminal"`
}

// ErrorSesionTrabajoV1 identifica un DTO de sesión inválido antes o después del socket.
type ErrorSesionTrabajoV1 struct{ Codigo string }

func (e *ErrorSesionTrabajoV1) Error() string {
	if e == nil {
		return ""
	}
	return e.Codigo
}

type ArtefactoPreservado struct {
	Origen          string  `json:"origen"`
	Clase           string  `json:"clase"`
	ContenidoSHA256 string  `json:"contenido_sha256"`
	ContenidoBytes  uint64  `json:"contenido_bytes"`
	RaizSHA256      *string `json:"raiz_sha256"`
	Archivos        *uint32 `json:"archivos"`
	BytesUtiles     uint64  `json:"bytes_utiles"`
}

type RespuestaPreservacion struct {
	Ejecucion        RespuestaEjecucion    `json:"ejecucion"`
	ManifiestoSHA256 string                `json:"manifiesto_sha256"`
	ManifiestoBytes  uint64                `json:"manifiesto_bytes"`
	Artefactos       []ArtefactoPreservado `json:"artefactos"`
	BytesUtiles      uint64                `json:"bytes_utiles"`
	RevisionTrabajo  uint64                `json:"revision_trabajo"`
}

type Problema struct {
	Codigo  string `json:"codigo"`
	Detalle string `json:"detalle"`
}
