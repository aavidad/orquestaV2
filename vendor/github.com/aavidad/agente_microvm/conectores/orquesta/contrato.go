// Package microvm ofrece a Orquesta un cliente del protocolo local de Agente MicroVM.
package microvm

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

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
	Perfil    json.RawMessage `json:"perfil"`
	Plan      json.RawMessage `json:"plan"`
	Concesion json.RawMessage `json:"concesion"`
}

// ExpiredLaunchContinuationTargetV1 fija una de las dos operaciones físicas
// ya existentes. No permite solicitar una operación, intención o clave nueva.
type ExpiredLaunchContinuationTargetV1 struct {
	Operation                string  `json:"operacion"`
	Schema                   uint16  `json:"esquema"`
	Code                     uint8   `json:"codigo"`
	IntentRef                string  `json:"intent_ref"`
	IdempotencyKey           string  `json:"clave_idempotencia"`
	RequestSHA256            string  `json:"request_sha256"`
	CommandSHA256            string  `json:"comando_sha256"`
	InnerAuthorizationSHA256 string  `json:"autorizacion_interior_sha256"`
	SignedRequestSHA256      *string `json:"solicitud_firmada_sha256"`
	DaemonState              string  `json:"estado_daemon"`
	RootState                string  `json:"estado_raiz"`
	RootRevision             *uint64 `json:"revision_raiz"`
}

// ExpiredLaunchContinuationManifestV1 es la foto inmutable devuelta por la
// preparación de Agente MicroVM para el intento ya registrado.
type ExpiredLaunchContinuationManifestV1 struct {
	Schema                string                            `json:"esquema"`
	Audience              string                            `json:"audiencia"`
	BrokerInstanceSHA256  string                            `json:"instancia_broker_sha256"`
	SourceDigest          string                            `json:"source_digest"`
	RequestKeySHA256      string                            `json:"clave_solicitud_sha256"`
	OriginalRequestSHA256 string                            `json:"solicitud_original_sha256"`
	LaunchRef             string                            `json:"launch_ref"`
	ExecutionRef          string                            `json:"execution_ref"`
	RunRef                string                            `json:"run_ref"`
	Fence                 uint64                            `json:"cerca"`
	Generation            uint64                            `json:"generacion"`
	CID                   uint32                            `json:"cid"`
	IdentitySHA256        string                            `json:"identidad_sha256"`
	ValidateLaunch        ExpiredLaunchContinuationTargetV1 `json:"validate_launch"`
	Launch                ExpiredLaunchContinuationTargetV1 `json:"launch"`
}

type ExpiredLaunchContinuationAuthorityContentV1 struct {
	Schema         string `json:"esquema"`
	Audience       string `json:"audiencia"`
	Purpose        string `json:"proposito"`
	AuthorityRef   string `json:"autoridad_ref"`
	ManifestSHA256 string `json:"manifiesto_sha256"`
	IssuedUnixMS   uint64 `json:"emitida_unix_ms"`
	ExpiresUnixMS  uint64 `json:"expira_unix_ms"`
	KeyID          string `json:"clave_id"`
	KeyEpoch       uint64 `json:"epoca_clave"`
	TrustRevision  uint64 `json:"revision_confianza"`
	Algorithm      string `json:"algoritmo"`
}

// ExpiredLaunchContinuationAuthorityV1 transporta el mismo manifiesto y una
// firma exterior separada de la concesión de lanzamiento ya caducada.
type ExpiredLaunchContinuationAuthorityV1 struct {
	Content         ExpiredLaunchContinuationAuthorityContentV1 `json:"contenido"`
	Manifest        ExpiredLaunchContinuationManifestV1         `json:"manifiesto"`
	SignatureBase64 string                                      `json:"firma_base64"`
}

type SolicitudContinuacionLanzamientoCaducadoV1 struct {
	SolicitudOriginal SolicitudLanzamiento                 `json:"solicitud_original"`
	Autoridad         ExpiredLaunchContinuationAuthorityV1 `json:"autoridad"`
}

type RespuestaPreparacionContinuacionLanzamientoCaducadoV1 struct {
	Manifiesto       ExpiredLaunchContinuationManifestV1 `json:"manifiesto"`
	ManifiestoSHA256 string                              `json:"manifiesto_sha256"`
}

type ModoDetencionSolicitadoV1 string

const (
	ModoDetencionCooperativa ModoDetencionSolicitadoV1 = "cooperativa"
	ModoDetencionForzada     ModoDetencionSolicitadoV1 = "forzada"
)

type ModoDetencionEfectivoV1 string

const (
	ModoDetencionEfectivoCooperativa ModoDetencionEfectivoV1 = "cooperativa"
	ModoDetencionEfectivoForzada     ModoDetencionEfectivoV1 = "forzada"
	ModoDetencionEfectivoYaAusente   ModoDetencionEfectivoV1 = "ya_ausente"
)

type EstadoDetencionV1 string

const (
	EstadoDetencionPendiente  EstadoDetencionV1 = "pendiente"
	EstadoDetencionConfirmada EstadoDetencionV1 = "confirmada"
)

type SolicitudDetencion struct {
	RevisionEsperada uint64                    `json:"revision_esperada"`
	Cerca            uint64                    `json:"cerca"`
	Modo             ModoDetencionSolicitadoV1 `json:"modo"`
}

type SolicitudPreservacion struct {
	RevisionEsperada uint64 `json:"revision_esperada"`
	Cerca            uint64 `json:"cerca"`
}

// SolicitudRecuperacionManifiestoPreservacion identifica un único sello durable.
type SolicitudRecuperacionManifiestoPreservacion struct {
	Cerca            uint64 `json:"cerca"`
	RevisionTrabajo  uint64 `json:"revision_trabajo"`
	ManifiestoRef    string `json:"manifiesto_ref"`
	ManifiestoSHA256 string `json:"manifiesto_sha256"`
	ManifiestoBytes  uint64 `json:"manifiesto_bytes"`
}

type SolicitudCierre struct {
	RevisionEsperada         uint64 `json:"revision_esperada"`
	Cerca                    uint64 `json:"cerca"`
	ManifiestoSHA256Esperado string `json:"manifiesto_sha256_esperado"`
}

type EstadoCierreV1 string

const (
	EstadoCierrePendiente  EstadoCierreV1 = "pendiente"
	EstadoCierreConfirmada EstadoCierreV1 = "confirmada"
)

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

// RespuestaDetencion conserva por separado la intención solicitada y el modo
// físico observado. Una respuesta pendiente no acredita todavía una parada.
type RespuestaDetencion struct {
	Ejecucion         RespuestaEjecucion        `json:"ejecucion"`
	ClaveIdempotencia string                    `json:"clave_idempotencia"`
	Estado            EstadoDetencionV1         `json:"estado"`
	ModoSolicitado    ModoDetencionSolicitadoV1 `json:"modo_solicitado"`
	ModoEfectivo      *ModoDetencionEfectivoV1  `json:"modo_efectivo,omitempty"`
	ReceiptRef        *string                   `json:"receipt_ref,omitempty"`
	ConfirmadaUnixMS  *uint64                   `json:"confirmada_unix_ms,omitempty"`
}

// RespuestaCierre distingue admision de confirmacion fisica. Solo confirmada
// porta el receipt durable y el instante observado por Agente MicroVM.
type RespuestaCierre struct {
	Ejecucion         RespuestaEjecucion `json:"ejecucion"`
	ClaveIdempotencia string             `json:"clave_idempotencia"`
	Estado            EstadoCierreV1     `json:"estado"`
	ReceiptRef        *string            `json:"receipt_ref,omitempty"`
	ConfirmadaUnixMS  *uint64            `json:"confirmada_unix_ms,omitempty"`
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

// EstadoReconciliacionEntradaSesionTrabajoV1 distingue espera de receipt durable.
type EstadoReconciliacionEntradaSesionTrabajoV1 string

const (
	EstadoReconciliacionEntradaPendiente EstadoReconciliacionEntradaSesionTrabajoV1 = "pendiente"
	EstadoReconciliacionEntradaResuelta  EstadoReconciliacionEntradaSesionTrabajoV1 = "resuelta"
)

// RespuestaReconciliacionEntradaSesionTrabajoV1 nunca contiene entrada ni secretos.
type RespuestaReconciliacionEntradaSesionTrabajoV1 struct {
	EjecucionRef string                                     `json:"ejecucion_ref"`
	SesionRef    string                                     `json:"sesion_ref"`
	Cerca        uint64                                     `json:"cerca"`
	Estado       EstadoReconciliacionEntradaSesionTrabajoV1 `json:"estado"`
	Comprobante  *RespuestaSesionTrabajoV1                  `json:"comprobante"`
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

// ErrorDetencionV1 identifica un contrato de parada inválido antes o después
// del socket. Codigo es una lista cerrada y no contiene texto remoto.
type ErrorDetencionV1 struct{ Codigo string }

func (e *ErrorDetencionV1) Error() string {
	if e == nil {
		return ""
	}
	return e.Codigo
}

// ErrorCierreV1 identifica una solicitud o respuesta de cierre que no prueba
// el efecto terminal durable.
type ErrorCierreV1 struct{ Codigo string }

func (e *ErrorCierreV1) Error() string {
	if e == nil || e.Codigo == "" {
		return "cierre.contrato_invalido"
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

// RespuestaManifiestoPreservacion transporta un manifiesto exacto sin exponer rutas.
type RespuestaManifiestoPreservacion struct {
	Referencia       string `json:"referencia"`
	Cerca            uint64 `json:"cerca"`
	RevisionTrabajo  uint64 `json:"revision_trabajo"`
	ManifiestoRef    string `json:"manifiesto_ref"`
	ManifiestoSHA256 string `json:"manifiesto_sha256"`
	ManifiestoBytes  uint64 `json:"manifiesto_bytes"`
	SelladaUnixMS    uint64 `json:"sellada_unix_ms"`
	ContenidoBase64  string `json:"contenido_base64"`
}

type Problema struct {
	Codigo  string `json:"codigo"`
	Detalle string `json:"detalle"`
}

const (
	ExpiredLaunchContinuationManifestSchemaV1  = "agentmicrovm.manifiesto-continuacion-lanzamiento-caducado.v1"
	ExpiredLaunchContinuationAuthoritySchemaV1 = "agentmicrovm.autoridad-continuacion-lanzamiento-caducado.v1"
	ExpiredLaunchContinuationAudienceV1        = "agentmicrovm.continuacion-lanzamiento-caducado.v1"
	ExpiredLaunchContinuationPurposeV1         = "continuar_intento_lanzamiento_caducado"
	ExpiredLaunchContinuationAlgorithmV1       = "ed25519"

	expiredContinuationMaximumValidityMS = uint64(5 * 60 * 1000)
	expiredContinuationMaximumField      = 512
	expiredContinuationMaximumBlock      = 64 * 1024
)

var (
	expiredContinuationManifestDomain  = []byte("agentmicrovm.manifiesto-continuacion-lanzamiento-caducado.v1\x00")
	expiredContinuationSignatureDomain = []byte("agentmicrovm.firma-autoridad-continuacion-lanzamiento-caducado.v1\x00")
	expiredContinuationAuthorityDomain = []byte("agentmicrovm.autoridad-continuacion-lanzamiento-caducado.v1\x00")

	expiredContinuationTargetSchema = esquemaObjetoEstricto(esquemaObjetoJSONEstricto{
		"operacion":                    esquemaEscalarJSONEstricto,
		"esquema":                      esquemaEscalarJSONEstricto,
		"codigo":                       esquemaEscalarJSONEstricto,
		"intent_ref":                   esquemaEscalarJSONEstricto,
		"clave_idempotencia":           esquemaEscalarJSONEstricto,
		"request_sha256":               esquemaEscalarJSONEstricto,
		"comando_sha256":               esquemaEscalarJSONEstricto,
		"autorizacion_interior_sha256": esquemaEscalarJSONEstricto,
		"solicitud_firmada_sha256":     esquemaEscalarJSONEstrictoOpcional,
		"estado_daemon":                esquemaEscalarJSONEstricto,
		"estado_raiz":                  esquemaEscalarJSONEstricto,
		"revision_raiz":                esquemaEscalarJSONEstrictoOpcional,
	})
	expiredContinuationManifestSchema = esquemaObjetoJSONEstricto{
		"esquema":                   esquemaEscalarJSONEstricto,
		"audiencia":                 esquemaEscalarJSONEstricto,
		"instancia_broker_sha256":   esquemaEscalarJSONEstricto,
		"source_digest":             esquemaEscalarJSONEstricto,
		"clave_solicitud_sha256":    esquemaEscalarJSONEstricto,
		"solicitud_original_sha256": esquemaEscalarJSONEstricto,
		"launch_ref":                esquemaEscalarJSONEstricto,
		"execution_ref":             esquemaEscalarJSONEstricto,
		"run_ref":                   esquemaEscalarJSONEstricto,
		"cerca":                     esquemaEscalarJSONEstricto,
		"generacion":                esquemaEscalarJSONEstricto,
		"cid":                       esquemaEscalarJSONEstricto,
		"identidad_sha256":          esquemaEscalarJSONEstricto,
		"validate_launch":           expiredContinuationTargetSchema,
		"launch":                    expiredContinuationTargetSchema,
	}
	expiredContinuationAuthorityContentSchema = esquemaObjetoEstricto(esquemaObjetoJSONEstricto{
		"esquema":            esquemaEscalarJSONEstricto,
		"audiencia":          esquemaEscalarJSONEstricto,
		"proposito":          esquemaEscalarJSONEstricto,
		"autoridad_ref":      esquemaEscalarJSONEstricto,
		"manifiesto_sha256":  esquemaEscalarJSONEstricto,
		"emitida_unix_ms":    esquemaEscalarJSONEstricto,
		"expira_unix_ms":     esquemaEscalarJSONEstricto,
		"clave_id":           esquemaEscalarJSONEstricto,
		"epoca_clave":        esquemaEscalarJSONEstricto,
		"revision_confianza": esquemaEscalarJSONEstricto,
		"algoritmo":          esquemaEscalarJSONEstricto,
	})
	expiredContinuationAuthoritySchema = esquemaObjetoJSONEstricto{
		"contenido":    expiredContinuationAuthorityContentSchema,
		"manifiesto":   esquemaObjetoEstricto(expiredContinuationManifestSchema),
		"firma_base64": esquemaEscalarJSONEstricto,
	}
	esquemaRespuestaPreparacionContinuacionLanzamientoCaducadoV1 = esquemaObjetoJSONEstricto{
		"manifiesto":        esquemaObjetoEstricto(expiredContinuationManifestSchema),
		"manifiesto_sha256": esquemaEscalarJSONEstricto,
	}
)

// DecodeExpiredLaunchContinuationManifestV1 decodifica y valida la forma
// completa del manifiesto antes de que el consumidor lo persista o firme.
func DecodeExpiredLaunchContinuationManifestV1(data []byte) (ExpiredLaunchContinuationManifestV1, error) {
	var value ExpiredLaunchContinuationManifestV1
	if !decodificarObjetoJSONEstricto(data, expiredContinuationManifestSchema, &value) {
		return ExpiredLaunchContinuationManifestV1{}, errors.New("agentmicrovm.expired_launch_continuation.invalid_manifest_json")
	}
	if err := validateExpiredLaunchContinuationManifest(value); err != nil {
		return ExpiredLaunchContinuationManifestV1{}, err
	}
	return value, nil
}

// DecodeExpiredLaunchContinuationAuthorityV1 decodifica una autoridad sin
// aceptar campos ausentes, desconocidos, duplicados ni JSON adicional.
func DecodeExpiredLaunchContinuationAuthorityV1(data []byte) (ExpiredLaunchContinuationAuthorityV1, error) {
	var value ExpiredLaunchContinuationAuthorityV1
	if !decodificarObjetoJSONEstricto(data, expiredContinuationAuthoritySchema, &value) {
		return ExpiredLaunchContinuationAuthorityV1{}, errors.New("agentmicrovm.expired_launch_continuation.invalid_authority_json")
	}
	if err := validateExpiredLaunchContinuationAuthority(value); err != nil {
		return ExpiredLaunchContinuationAuthorityV1{}, err
	}
	return value, nil
}

func ExpiredLaunchContinuationManifestMessageV1(manifest ExpiredLaunchContinuationManifestV1) ([]byte, error) {
	if err := validateExpiredLaunchContinuationManifest(manifest); err != nil {
		return nil, err
	}
	output := append([]byte(nil), expiredContinuationManifestDomain...)
	for _, field := range []string{
		manifest.Schema, manifest.Audience, manifest.BrokerInstanceSHA256,
		manifest.SourceDigest, manifest.RequestKeySHA256, manifest.OriginalRequestSHA256,
		manifest.LaunchRef, manifest.ExecutionRef, manifest.RunRef,
	} {
		if err := appendExpiredContinuationField(&output, field); err != nil {
			return nil, err
		}
	}
	output = binary.BigEndian.AppendUint64(output, manifest.Fence)
	output = binary.BigEndian.AppendUint64(output, manifest.Generation)
	output = binary.BigEndian.AppendUint32(output, manifest.CID)
	if err := appendExpiredContinuationField(&output, manifest.IdentitySHA256); err != nil {
		return nil, err
	}
	if err := appendExpiredContinuationTarget(&output, manifest.ValidateLaunch); err != nil {
		return nil, err
	}
	if err := appendExpiredContinuationTarget(&output, manifest.Launch); err != nil {
		return nil, err
	}
	return output, nil
}

func ExpiredLaunchContinuationManifestSHA256V1(manifest ExpiredLaunchContinuationManifestV1) (string, error) {
	message, err := ExpiredLaunchContinuationManifestMessageV1(manifest)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(message)
	return hex.EncodeToString(digest[:]), nil
}

func ExpiredLaunchContinuationSigningMessageV1(content ExpiredLaunchContinuationAuthorityContentV1) ([]byte, error) {
	if err := validateExpiredLaunchContinuationContent(content); err != nil {
		return nil, err
	}
	output := append([]byte(nil), expiredContinuationSignatureDomain...)
	for _, field := range []string{content.Schema, content.Audience, content.Purpose, content.AuthorityRef, content.ManifestSHA256} {
		if err := appendExpiredContinuationField(&output, field); err != nil {
			return nil, err
		}
	}
	output = binary.BigEndian.AppendUint64(output, content.IssuedUnixMS)
	output = binary.BigEndian.AppendUint64(output, content.ExpiresUnixMS)
	if err := appendExpiredContinuationField(&output, content.KeyID); err != nil {
		return nil, err
	}
	output = binary.BigEndian.AppendUint64(output, content.KeyEpoch)
	output = binary.BigEndian.AppendUint64(output, content.TrustRevision)
	if err := appendExpiredContinuationField(&output, content.Algorithm); err != nil {
		return nil, err
	}
	return output, nil
}

func ExpiredLaunchContinuationAuthoritySHA256V1(authority ExpiredLaunchContinuationAuthorityV1) (string, error) {
	if err := validateExpiredLaunchContinuationAuthority(authority); err != nil {
		return "", err
	}
	content, err := ExpiredLaunchContinuationSigningMessageV1(authority.Content)
	if err != nil {
		return "", err
	}
	manifest, err := ExpiredLaunchContinuationManifestMessageV1(authority.Manifest)
	if err != nil {
		return "", err
	}
	output := append([]byte(nil), expiredContinuationAuthorityDomain...)
	if err := appendExpiredContinuationBlock(&output, content); err != nil {
		return "", err
	}
	if err := appendExpiredContinuationBlock(&output, manifest); err != nil {
		return "", err
	}
	if err := appendExpiredContinuationField(&output, authority.SignatureBase64); err != nil {
		return "", err
	}
	digest := sha256.Sum256(output)
	return hex.EncodeToString(digest[:]), nil
}

func SignExpiredLaunchContinuationAuthorityV1(
	privateKey ed25519.PrivateKey,
	content ExpiredLaunchContinuationAuthorityContentV1,
	manifest ExpiredLaunchContinuationManifestV1,
) (ExpiredLaunchContinuationAuthorityV1, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return ExpiredLaunchContinuationAuthorityV1{}, errors.New("agentmicrovm.expired_launch_continuation.invalid_private_key")
	}
	manifestDigest, err := ExpiredLaunchContinuationManifestSHA256V1(manifest)
	if err != nil || manifestDigest != content.ManifestSHA256 {
		return ExpiredLaunchContinuationAuthorityV1{}, errors.New("agentmicrovm.expired_launch_continuation.manifest_mismatch")
	}
	message, err := ExpiredLaunchContinuationSigningMessageV1(content)
	if err != nil {
		return ExpiredLaunchContinuationAuthorityV1{}, err
	}
	signature := ed25519.Sign(privateKey, message)
	value := ExpiredLaunchContinuationAuthorityV1{
		Content: content, Manifest: manifest,
		SignatureBase64: base64.StdEncoding.EncodeToString(signature),
	}
	if err := validateExpiredLaunchContinuationAuthority(value); err != nil {
		return ExpiredLaunchContinuationAuthorityV1{}, err
	}
	return value, nil
}

func VerifyExpiredLaunchContinuationAuthorityV1(
	publicKey ed25519.PublicKey,
	authority ExpiredLaunchContinuationAuthorityV1,
) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New("agentmicrovm.expired_launch_continuation.invalid_public_key")
	}
	if err := validateExpiredLaunchContinuationAuthority(authority); err != nil {
		return err
	}
	signature, valid := decodeExpiredContinuationSignatureBase64(authority.SignatureBase64)
	if !valid {
		return errors.New("agentmicrovm.expired_launch_continuation.invalid_signature")
	}
	message, err := ExpiredLaunchContinuationSigningMessageV1(authority.Content)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, message, signature) {
		return errors.New("agentmicrovm.expired_launch_continuation.signature_rejected")
	}
	return nil
}

func validateExpiredLaunchContinuationAuthority(value ExpiredLaunchContinuationAuthorityV1) error {
	if err := validateExpiredLaunchContinuationContent(value.Content); err != nil {
		return err
	}
	manifestDigest, err := ExpiredLaunchContinuationManifestSHA256V1(value.Manifest)
	if err != nil || manifestDigest != value.Content.ManifestSHA256 {
		return errors.New("agentmicrovm.expired_launch_continuation.manifest_mismatch")
	}
	_, valid := decodeExpiredContinuationSignatureBase64(value.SignatureBase64)
	if !valid {
		return errors.New("agentmicrovm.expired_launch_continuation.invalid_signature")
	}
	return nil
}

func decodeExpiredContinuationSignatureBase64(value string) ([]byte, bool) {
	if strings.ContainsAny(value, "\r\n") {
		return nil, false
	}
	signature, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil || len(signature) != ed25519.SignatureSize ||
		base64.StdEncoding.EncodeToString(signature) != value {
		return nil, false
	}
	return signature, true
}

func validateExpiredLaunchContinuationContent(value ExpiredLaunchContinuationAuthorityContentV1) error {
	validity, validWindow := value.ExpiresUnixMS-value.IssuedUnixMS, value.ExpiresUnixMS > value.IssuedUnixMS
	if value.Schema != ExpiredLaunchContinuationAuthoritySchemaV1 || value.Audience != ExpiredLaunchContinuationAudienceV1 ||
		value.Purpose != ExpiredLaunchContinuationPurposeV1 || !validExpiredContinuationMachineReference(value.AuthorityRef, "continuation:", 160) ||
		!digestValido(value.ManifestSHA256) || value.IssuedUnixMS == 0 || !validWindow || validity > expiredContinuationMaximumValidityMS ||
		!validExpiredContinuationMachineText(value.KeyID, 128) || value.KeyEpoch == 0 || value.TrustRevision == 0 ||
		value.Algorithm != ExpiredLaunchContinuationAlgorithmV1 {
		return errors.New("agentmicrovm.expired_launch_continuation.invalid_content")
	}
	return nil
}

func validateExpiredLaunchContinuationManifest(value ExpiredLaunchContinuationManifestV1) error {
	if value.Schema != ExpiredLaunchContinuationManifestSchemaV1 || value.Audience != ExpiredLaunchContinuationAudienceV1 ||
		!digestValido(value.BrokerInstanceSHA256) || !digestValido(value.SourceDigest) ||
		!digestValido(value.RequestKeySHA256) || !digestValido(value.OriginalRequestSHA256) ||
		!validExpiredContinuationMachineReference(value.LaunchRef, "launch:", 160) ||
		!validExpiredContinuationMachineReference(value.ExecutionRef, "ejecucion:", 160) ||
		!validExpiredContinuationMachineReference(value.RunRef, "run:", 160) || value.Fence == 0 || value.Generation == 0 || value.CID < 3 ||
		!digestValido(value.IdentitySHA256) || !validExpiredContinuationTarget(value.ValidateLaunch, true) ||
		!validExpiredContinuationTarget(value.Launch, false) || value.ValidateLaunch.IntentRef == value.Launch.IntentRef ||
		value.ValidateLaunch.IdempotencyKey == value.Launch.IdempotencyKey {
		return errors.New("agentmicrovm.expired_launch_continuation.invalid_manifest")
	}
	return nil
}

func validExpiredContinuationTarget(value ExpiredLaunchContinuationTargetV1, validate bool) bool {
	if !validExpiredContinuationMachineReference(value.IntentRef, "intent:", 160) ||
		!validExpiredContinuationMachineText(value.IdempotencyKey, 128) || !digestValido(value.RequestSHA256) ||
		!digestValido(value.CommandSHA256) || !digestValido(value.InnerAuthorizationSHA256) {
		return false
	}
	if validate {
		return value.Operation == "validate_launch" && value.Schema == 2 && value.Code == 8 &&
			(value.DaemonState == "signed_request" || value.DaemonState == "ambiguous") && value.RootState == "prepared" &&
			value.RootRevision != nil && *value.RootRevision == 1 && value.SignedRequestSHA256 != nil && digestValido(*value.SignedRequestSHA256)
	}
	return value.Operation == "launch" && value.Schema == 1 && value.Code == 1 && value.DaemonState == "reserved" &&
		value.RootState == "absent" && value.RootRevision == nil && value.SignedRequestSHA256 == nil
}

func appendExpiredContinuationTarget(output *[]byte, value ExpiredLaunchContinuationTargetV1) error {
	if err := appendExpiredContinuationField(output, value.Operation); err != nil {
		return err
	}
	*output = binary.BigEndian.AppendUint16(*output, value.Schema)
	*output = append(*output, value.Code)
	for _, field := range []string{value.IntentRef, value.IdempotencyKey, value.RequestSHA256, value.CommandSHA256, value.InnerAuthorizationSHA256} {
		if err := appendExpiredContinuationField(output, field); err != nil {
			return err
		}
	}
	if value.SignedRequestSHA256 == nil {
		*output = append(*output, 0)
	} else {
		*output = append(*output, 1)
		if err := appendExpiredContinuationField(output, *value.SignedRequestSHA256); err != nil {
			return err
		}
	}
	if err := appendExpiredContinuationField(output, value.DaemonState); err != nil {
		return err
	}
	if err := appendExpiredContinuationField(output, value.RootState); err != nil {
		return err
	}
	if value.RootRevision == nil {
		*output = append(*output, 0)
	} else {
		*output = append(*output, 1)
		*output = binary.BigEndian.AppendUint64(*output, *value.RootRevision)
	}
	return nil
}

func appendExpiredContinuationField(output *[]byte, value string) error {
	if len(value) > expiredContinuationMaximumField {
		return errors.New("agentmicrovm.expired_launch_continuation.field_too_large")
	}
	*output = binary.BigEndian.AppendUint32(*output, uint32(len(value)))
	*output = append(*output, value...)
	return nil
}

func appendExpiredContinuationBlock(output *[]byte, value []byte) error {
	if len(value) > expiredContinuationMaximumBlock {
		return errors.New("agentmicrovm.expired_launch_continuation.block_too_large")
	}
	*output = binary.BigEndian.AppendUint32(*output, uint32(len(value)))
	*output = append(*output, value...)
	return nil
}

func validExpiredContinuationMachineReference(value, prefix string, maximum int) bool {
	return strings.HasPrefix(value, prefix) && len(value) > len(prefix) && len(value) <= maximum &&
		validExpiredContinuationMachineBytes(value)
}

func validExpiredContinuationMachineText(value string, maximum int) bool {
	return value != "" && len(value) <= maximum && validExpiredContinuationMachineBytes(value)
}

func validExpiredContinuationMachineBytes(value string) bool {
	for _, value := range []byte(value) {
		if !((value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9') ||
			value == '.' || value == '_' || value == ':' || value == '-') {
			return false
		}
	}
	return true
}
