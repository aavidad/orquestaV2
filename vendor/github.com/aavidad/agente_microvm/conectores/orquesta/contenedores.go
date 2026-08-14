package microvm

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	EsquemaPlanLanzamientoContenedorV1 = "agentmicrovm.plan-lanzamiento-contenedor.v1"
	maximoBultosContenedorV1           = 64
	maximoCuerpoSolicitudContenedorV1  = 16 * 1_024
)

var dominioPlanLanzamientoContenedorV1 = []byte("agentmicrovm.plan-lanzamiento-contenedor.v1\x00")

// PlanLanzamientoContenedorV1 es el plan firmado específico de Docker. No
// representa CID, kernel, initramfs, vsock, red, mounts ni rutas del host.
type PlanLanzamientoContenedorV1 struct {
	Esquema              string   `json:"esquema"`
	OperacionRef         string   `json:"operacion_ref"`
	EjecucionRef         string   `json:"ejecucion_ref"`
	RunRef               string   `json:"run_ref"`
	Cerca                uint64   `json:"cerca"`
	EspecificacionRef    string   `json:"especificacion_ref"`
	BultosRef            []string `json:"bultos_ref"`
	ImagenRef            string   `json:"imagen_ref"`
	VCPU                 uint8    `json:"vcpu"`
	MemoriaMiB           uint32   `json:"memoria_mib"`
	MaximoPIDs           uint32   `json:"maximo_pids"`
	LimiteTiempoMS       uint64   `json:"limite_tiempo_ms"`
	LimiteDiscoPicoBytes uint64   `json:"limite_disco_pico_bytes"`
	LimiteTokensAgente   uint64   `json:"limite_tokens_agente"`
}

// SolicitudLanzarORecuperarContenedorV1 conserva el envelope firmado sin
// duplicar en este cliente la autoridad criptográfica del servidor.
type SolicitudLanzarORecuperarContenedorV1 struct {
	Plan      PlanLanzamientoContenedorV1 `json:"plan"`
	Concesion json.RawMessage             `json:"concesion"`
}

type IdentidadContenedorV1 struct {
	ContainerID    string `json:"container_id"`
	PIDObservado   uint32 `json:"pid_observado"`
	OwnerLabel     string `json:"owner_label"`
	ExecutionLabel string `json:"execution_label"`
	RunLabel       string `json:"run_label"`
	Generation     uint64 `json:"generation"`
}

type RespuestaContenedorV1 struct {
	Referencia  string                 `json:"referencia"`
	Estado      string                 `json:"estado"`
	Revision    uint64                 `json:"revision"`
	Cerca       uint64                 `json:"cerca"`
	VCPU        uint8                  `json:"vcpu"`
	MemoriaMiB  uint32                 `json:"memoria_mib"`
	Identidad   *IdentidadContenedorV1 `json:"identidad"`
	RecursoVivo *bool                  `json:"recurso_vivo"`
	EstadoMotor *string                `json:"estado_motor"`
}

type RespuestaOperacionContenedorV1 struct {
	OperacionRef string                `json:"operacion_ref"`
	PlanSHA256   string                `json:"plan_sha256"`
	ConcesionRef string                `json:"concesion_ref"`
	Contenedor   RespuestaContenedorV1 `json:"contenedor"`
}

// SolicitudDetenerContenedorV1 cerca la retirada al snapshot durable observado.
type SolicitudDetenerContenedorV1 struct {
	RevisionEsperada uint64 `json:"revision_esperada"`
	Cerca            uint64 `json:"cerca"`
}

type ErrorContenedorV1 struct{ Codigo string }

func (e *ErrorContenedorV1) Error() string {
	if e == nil {
		return ""
	}
	return e.Codigo
}

var operacionesDockerV1 = [...]string{
	"lanzar_o_recuperar_contenedor",
	"consultar_operacion_contenedor",
	"observar_contenedor",
	"detener_contenedor",
	"ejecutar_orden_contenedor",
	"sincronizar_entrada_contenedor",
	"sincronizar_salida_contenedor",
	"iniciar_sesion_contenedor",
	"enviar_entrada_sesion_contenedor",
	"leer_eventos_sesion_contenedor",
}

var esquemaIdentidadContenedorV1 = esquemaObjetoJSONEstricto{
	"container_id": esquemaEscalarJSONEstricto, "pid_observado": esquemaEscalarJSONEstricto,
	"owner_label": esquemaEscalarJSONEstricto, "execution_label": esquemaEscalarJSONEstricto,
	"run_label": esquemaEscalarJSONEstricto, "generation": esquemaEscalarJSONEstricto,
}

var esquemaRespuestaContenedorV1 = esquemaObjetoJSONEstricto{
	"referencia": esquemaEscalarJSONEstricto, "estado": esquemaEscalarJSONEstricto,
	"revision": esquemaEscalarJSONEstricto, "cerca": esquemaEscalarJSONEstricto,
	"vcpu": esquemaEscalarJSONEstricto, "memoria_mib": esquemaEscalarJSONEstricto,
	"identidad":    esquemaObjetoEstrictoOpcional(esquemaIdentidadContenedorV1),
	"recurso_vivo": esquemaEscalarJSONEstrictoOpcional,
	"estado_motor": esquemaEscalarJSONEstrictoOpcional,
}

var esquemaRespuestaOperacionContenedorV1 = esquemaObjetoJSONEstricto{
	"operacion_ref": esquemaEscalarJSONEstricto,
	"plan_sha256":   esquemaEscalarJSONEstricto,
	"concesion_ref": esquemaEscalarJSONEstricto,
	"contenedor":    esquemaObjetoEstricto(esquemaRespuestaContenedorV1),
}

var esquemaPlanLanzamientoContenedorV1 = esquemaObjetoJSONEstricto{
	"esquema": esquemaEscalarJSONEstricto, "operacion_ref": esquemaEscalarJSONEstricto,
	"ejecucion_ref": esquemaEscalarJSONEstricto, "run_ref": esquemaEscalarJSONEstricto,
	"cerca": esquemaEscalarJSONEstricto, "especificacion_ref": esquemaEscalarJSONEstricto,
	"bultos_ref": esquemaArrayEstricto(esquemaEscalarJSONEstricto),
	"imagen_ref": esquemaEscalarJSONEstricto, "vcpu": esquemaEscalarJSONEstricto,
	"memoria_mib": esquemaEscalarJSONEstricto, "maximo_pids": esquemaEscalarJSONEstricto,
	"limite_tiempo_ms":        esquemaEscalarJSONEstricto,
	"limite_disco_pico_bytes": esquemaEscalarJSONEstricto,
	"limite_tokens_agente":    esquemaEscalarJSONEstricto,
}

var esquemaSolicitudLanzarORecuperarContenedorV1 = esquemaObjetoJSONEstricto{
	"plan":      esquemaObjetoEstricto(esquemaPlanLanzamientoContenedorV1),
	"concesion": esquemaObjetoEstricto(esquemaConcesionLanzamientoFirmadaV1),
}

var esquemaSolicitudDetenerContenedorV1 = esquemaObjetoJSONEstricto{
	"revision_esperada": esquemaEscalarJSONEstricto,
	"cerca":             esquemaEscalarJSONEstricto,
}

var esquemaRespuestaCapacidadesDockerV1 = esquemaObjetoJSONEstricto{
	"protocolo": esquemaEscalarJSONEstricto, "version": esquemaEscalarJSONEstricto,
	"operaciones":              esquemaArrayEstricto(esquemaEscalarJSONEstricto),
	"kvm_disponible":           esquemaEscalarJSONEstricto,
	"firecracker_configurado":  esquemaEscalarJSONEstricto,
	"firecracker_ejecutable":   esquemaEscalarJSONEstricto,
	"maximo_ejecuciones":       esquemaEscalarJSONEstricto,
	"backend_ejecuciones":      esquemaEscalarJSONEstricto,
	"docker_configurado":       esquemaEscalarJSONEstricto,
	"docker_engine_disponible": esquemaEscalarJSONEstricto,
}

// NegociarDocker acredita selector, Engine y superficie completa antes de que
// el consumidor persista o envíe una mutación Docker.
func (c *Cliente) NegociarDocker(ctx context.Context) (RespuestaCapacidades, error) {
	capacidad, err := solicitarEstrictoConEstado[RespuestaCapacidades](
		ctx,
		c,
		http.MethodGet,
		"/v1/capacidades",
		"",
		nil,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaCapacidades{}, err
	}
	if capacidad.Protocolo != ProtocoloLocal || capacidad.Version == "" ||
		capacidad.BackendEjecuciones != BackendEjecucionesDocker ||
		!capacidad.DockerConfigurado || !capacidad.DockerEngineDisponible ||
		capacidad.MaximoEjecuciones == 0 {
		return RespuestaCapacidades{}, errorContenedor("contenedores.capacidad_incompatible")
	}
	disponibles := make(map[string]struct{}, len(capacidad.Operaciones))
	for _, operacion := range capacidad.Operaciones {
		if _, duplicada := disponibles[operacion]; duplicada {
			return RespuestaCapacidades{}, errorContenedor("contenedores.capacidad_incompatible")
		}
		disponibles[operacion] = struct{}{}
	}
	for _, requerida := range operacionesDockerV1 {
		if _, existe := disponibles[requerida]; !existe {
			return RespuestaCapacidades{}, errorContenedor("contenedores.capacidad_incompatible")
		}
	}
	return capacidad, nil
}

func (c *Cliente) LanzarORecuperarContenedor(
	ctx context.Context,
	clave string,
	solicitud SolicitudLanzarORecuperarContenedorV1,
) (RespuestaContenedorV1, error) {
	if clave != solicitud.Plan.OperacionRef || !claveIdempotenciaContenedorValida(clave) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.clave_invalida")
	}
	cuerpo, err := CodificarSolicitudLanzarORecuperarContenedorV1(solicitud)
	if err != nil {
		return RespuestaContenedorV1{}, err
	}
	return c.lanzarORecuperarContenedorCodificada(ctx, clave, solicitud, cuerpo)
}

// LanzarORecuperarContenedorCodificada reenvía sin reserializar el envelope
// firmado que el consumidor persistió antes del primer intento.
func (c *Cliente) LanzarORecuperarContenedorCodificada(
	ctx context.Context,
	clave string,
	cuerpo json.RawMessage,
) (RespuestaContenedorV1, error) {
	var solicitud SolicitudLanzarORecuperarContenedorV1
	if len(cuerpo) > maximoCuerpoSolicitudContenedorV1 ||
		!decodificarObjetoJSONEstricto(
			cuerpo,
			esquemaSolicitudLanzarORecuperarContenedorV1,
			&solicitud,
		) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.solicitud_invalida")
	}
	return c.lanzarORecuperarContenedorCodificada(ctx, clave, solicitud, cuerpo)
}

func (c *Cliente) lanzarORecuperarContenedorCodificada(
	ctx context.Context,
	clave string,
	solicitud SolicitudLanzarORecuperarContenedorV1,
	cuerpo []byte,
) (RespuestaContenedorV1, error) {
	if clave != solicitud.Plan.OperacionRef || !claveIdempotenciaContenedorValida(clave) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.clave_invalida")
	}
	if len(cuerpo) > maximoCuerpoSolicitudContenedorV1 {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.solicitud_invalida")
	}
	if err := validarSolicitudLanzarORecuperarContenedor(solicitud); err != nil {
		return RespuestaContenedorV1{}, err
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaContenedorV1](
		ctx, c, http.MethodPost, "/v1/contenedores", clave, cuerpo, http.StatusCreated,
	)
	if err != nil {
		return RespuestaContenedorV1{}, err
	}
	if err := validarRespuestaContenedorActiva(solicitud.Plan, respuesta); err != nil {
		return RespuestaContenedorV1{}, err
	}
	return respuesta, nil
}

func (c *Cliente) ConsultarOperacionContenedor(
	ctx context.Context,
	operacionRef string,
) (RespuestaOperacionContenedorV1, error) {
	if !claveIdempotenciaContenedorValida(operacionRef) {
		return RespuestaOperacionContenedorV1{}, errorContenedor("contenedores.clave_invalida")
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaOperacionContenedorV1](
		ctx,
		c,
		http.MethodGet,
		"/v1/contenedores/operaciones/"+segmentoRutaContenedor(operacionRef),
		"",
		nil,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaOperacionContenedorV1{}, err
	}
	if respuesta.OperacionRef != operacionRef || !sha256Valido(respuesta.PlanSHA256) ||
		!referenciaValida(respuesta.ConcesionRef, "concesion:", 160) {
		return RespuestaOperacionContenedorV1{}, errorContenedor("contenedores.respuesta_invalida")
	}
	if err := validarRespuestaContenedorBasica(respuesta.Contenedor); err != nil {
		return RespuestaOperacionContenedorV1{}, err
	}
	if respuesta.Contenedor.RecursoVivo != nil || respuesta.Contenedor.EstadoMotor != nil {
		return RespuestaOperacionContenedorV1{}, errorContenedor("contenedores.liveness_invalida")
	}
	return respuesta, nil
}

func (c *Cliente) ObservarContenedor(
	ctx context.Context,
	referencia string,
) (RespuestaContenedorV1, error) {
	if !referenciaEjecucionContenedorValida(referencia) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.referencia_invalida")
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaContenedorV1](
		ctx,
		c,
		http.MethodGet,
		"/v1/contenedores/"+segmentoRutaContenedor(referencia),
		"",
		nil,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaContenedorV1{}, err
	}
	if respuesta.Referencia != referencia {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.referencia_invalida")
	}
	if err := validarRespuestaContenedorObservada(respuesta); err != nil {
		return RespuestaContenedorV1{}, err
	}
	return respuesta, nil
}

// DetenerContenedor solicita la retirada durable exacta. Un retorno correcto
// acredita el estado terminal publicado, no sólo la admisión de una señal.
func (c *Cliente) DetenerContenedor(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudDetenerContenedorV1,
) (RespuestaContenedorV1, error) {
	if !claveIdempotenciaContenedorValida(clave) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.clave_invalida")
	}
	if !referenciaEjecucionContenedorValida(referencia) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.referencia_invalida")
	}
	cuerpo, err := CodificarSolicitudDetenerContenedorV1(solicitud)
	if err != nil {
		return RespuestaContenedorV1{}, err
	}
	return c.detenerContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

// CodificarSolicitudDetenerContenedorV1 fija el cuerpo que debe persistirse
// antes del primer intento de retirar el contenedor.
func CodificarSolicitudDetenerContenedorV1(
	solicitud SolicitudDetenerContenedorV1,
) (json.RawMessage, error) {
	if err := validarSolicitudDetenerContenedor(solicitud); err != nil {
		return nil, err
	}
	codificada, err := json.Marshal(solicitud)
	if err != nil {
		return nil, errorContenedor("contenedores.codificacion_fallida")
	}
	return codificada, nil
}

// DetenerContenedorCodificada reenvía sin reserializar una intención de
// detención previamente persistida.
func (c *Cliente) DetenerContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	cuerpo json.RawMessage,
) (RespuestaContenedorV1, error) {
	var solicitud SolicitudDetenerContenedorV1
	if len(cuerpo) > maximoCuerpoSolicitudContenedorV1 ||
		!decodificarObjetoJSONEstricto(cuerpo, esquemaSolicitudDetenerContenedorV1, &solicitud) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.detencion_invalida")
	}
	return c.detenerContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

func (c *Cliente) detenerContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudDetenerContenedorV1,
	cuerpo []byte,
) (RespuestaContenedorV1, error) {
	if !claveIdempotenciaContenedorValida(clave) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.clave_invalida")
	}
	if !referenciaEjecucionContenedorValida(referencia) {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.referencia_invalida")
	}
	if len(cuerpo) > maximoCuerpoSolicitudContenedorV1 {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.detencion_invalida")
	}
	if err := validarSolicitudDetenerContenedor(solicitud); err != nil {
		return RespuestaContenedorV1{}, err
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaContenedorV1](
		ctx,
		c,
		http.MethodPost,
		"/v1/contenedores/"+segmentoRutaContenedor(referencia)+"/detencion",
		clave,
		cuerpo,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaContenedorV1{}, err
	}
	if respuesta.Referencia != referencia || respuesta.Cerca != solicitud.Cerca ||
		respuesta.Revision != solicitud.RevisionEsperada+2 {
		return RespuestaContenedorV1{}, errorContenedor("contenedores.respuesta_invalida")
	}
	if err := validarRespuestaContenedorBasica(respuesta); err != nil {
		return RespuestaContenedorV1{}, err
	}
	if err := validarLivenessContenedor(respuesta, "detenido", false, "removed"); err != nil {
		return RespuestaContenedorV1{}, err
	}
	return respuesta, nil
}

func validarSolicitudDetenerContenedor(solicitud SolicitudDetenerContenedorV1) error {
	if solicitud.RevisionEsperada == 0 ||
		solicitud.RevisionEsperada > maximoEnteroDurableV1-2 ||
		solicitud.Cerca == 0 || solicitud.Cerca > maximoEnteroDurableV1 {
		return errorContenedor("contenedores.detencion_invalida")
	}
	return nil
}

// CodificarPlanLanzamientoContenedorV1 fija los bytes JSON que el consumidor
// persiste antes de preparar o intentar una admisión.
func CodificarPlanLanzamientoContenedorV1(
	plan PlanLanzamientoContenedorV1,
) (json.RawMessage, error) {
	if err := validarPlanLanzamientoContenedor(plan); err != nil {
		return nil, err
	}
	codificado, err := json.Marshal(plan)
	if err != nil {
		return nil, errorContenedor("plan_contenedor.codificacion_fallida")
	}
	return codificado, nil
}

// CodificarSolicitudLanzarORecuperarContenedorV1 fija el cuerpo exacto que
// LanzarORecuperarContenedor enviará por el socket Unix.
func CodificarSolicitudLanzarORecuperarContenedorV1(
	solicitud SolicitudLanzarORecuperarContenedorV1,
) (json.RawMessage, error) {
	if err := validarSolicitudLanzarORecuperarContenedor(solicitud); err != nil {
		return nil, err
	}
	codificada, err := json.Marshal(solicitud)
	if err != nil {
		return nil, errorContenedor("contenedores.codificacion_fallida")
	}
	if len(codificada) > maximoCuerpoSolicitudContenedorV1 {
		return nil, errorContenedor("contenedores.solicitud_invalida")
	}
	return codificada, nil
}

// MensajeCanonicoPlanLanzamientoContenedorV1 reproduce byte a byte el dominio
// Rust firmado, ordenando bultos sin alterar el DTO del llamador.
func MensajeCanonicoPlanLanzamientoContenedorV1(
	plan PlanLanzamientoContenedorV1,
) ([]byte, error) {
	if err := validarPlanLanzamientoContenedor(plan); err != nil {
		return nil, err
	}
	bultos := append([]string(nil), plan.BultosRef...)
	sort.Strings(bultos)
	mensaje := append([]byte(nil), dominioPlanLanzamientoContenedorV1...)
	mensaje = anexarCampoContenedor(mensaje, plan.Esquema)
	mensaje = anexarCampoContenedor(mensaje, plan.OperacionRef)
	mensaje = anexarCampoContenedor(mensaje, plan.EjecucionRef)
	mensaje = anexarCampoContenedor(mensaje, plan.RunRef)
	mensaje = binary.BigEndian.AppendUint64(mensaje, plan.Cerca)
	mensaje = anexarCampoContenedor(mensaje, plan.EspecificacionRef)
	mensaje = append(mensaje, byte(len(bultos)))
	for _, bulto := range bultos {
		mensaje = anexarCampoContenedor(mensaje, bulto)
	}
	mensaje = anexarCampoContenedor(mensaje, plan.ImagenRef)
	mensaje = append(mensaje, plan.VCPU)
	mensaje = binary.BigEndian.AppendUint32(mensaje, plan.MemoriaMiB)
	mensaje = binary.BigEndian.AppendUint32(mensaje, plan.MaximoPIDs)
	mensaje = binary.BigEndian.AppendUint64(mensaje, plan.LimiteTiempoMS)
	mensaje = binary.BigEndian.AppendUint64(mensaje, plan.LimiteDiscoPicoBytes)
	mensaje = binary.BigEndian.AppendUint64(mensaje, plan.LimiteTokensAgente)
	return mensaje, nil
}

func CalcularSHA256PlanLanzamientoContenedorV1(
	plan PlanLanzamientoContenedorV1,
) (string, error) {
	mensaje, err := MensajeCanonicoPlanLanzamientoContenedorV1(plan)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(mensaje)
	return hex.EncodeToString(digest[:]), nil
}

func validarSolicitudLanzarORecuperarContenedor(
	solicitud SolicitudLanzarORecuperarContenedorV1,
) error {
	digest, err := CalcularSHA256PlanLanzamientoContenedorV1(solicitud.Plan)
	if err != nil {
		return err
	}
	var concesion concesionLanzamientoFirmadaV1
	if !decodificarObjetoJSONEstricto(
		solicitud.Concesion,
		esquemaConcesionLanzamientoFirmadaV1,
		&concesion,
	) || !contenidoConcesionValido(concesion.Contenido) {
		return errorContenedor("contenedores.concesion_invalida")
	}
	firma, err := base64.StdEncoding.Strict().DecodeString(concesion.FirmaBase64)
	if err != nil || len(firma) != 64 ||
		base64.StdEncoding.EncodeToString(firma) != concesion.FirmaBase64 {
		return errorContenedor("contenedores.concesion_invalida")
	}
	if concesion.Contenido.RunRef != solicitud.Plan.RunRef ||
		concesion.Contenido.Cerca != solicitud.Plan.Cerca ||
		concesion.Contenido.PlanSHA256 != digest {
		return errorContenedor("contenedores.vinculo_invalido")
	}
	return nil
}

func validarPlanLanzamientoContenedor(plan PlanLanzamientoContenedorV1) error {
	if plan.Esquema != EsquemaPlanLanzamientoContenedorV1 {
		return errorContenedor("plan_contenedor.esquema_incompatible")
	}
	if !claveIdempotenciaContenedorValida(plan.OperacionRef) {
		return errorContenedor("plan_contenedor.operacion_invalida")
	}
	if !referenciaEjecucionContenedorValida(plan.EjecucionRef) {
		return errorContenedor("plan_contenedor.ejecucion_invalida")
	}
	if !referenciaExternaContenedorValida(plan.RunRef) {
		return errorContenedor("plan_contenedor.run_ref_invalido")
	}
	if plan.Cerca == 0 {
		return errorContenedor("plan_contenedor.cerca_invalida")
	}
	if !referenciaExternaContenedorValida(plan.EspecificacionRef) {
		return errorContenedor("plan_contenedor.especificacion_invalida")
	}
	if len(plan.BultosRef) > maximoBultosContenedorV1 {
		return errorContenedor("plan_contenedor.bultos_invalidos")
	}
	bultos := append([]string(nil), plan.BultosRef...)
	for _, bulto := range bultos {
		if !referenciaExternaContenedorValida(bulto) {
			return errorContenedor("plan_contenedor.bultos_invalidos")
		}
	}
	sort.Strings(bultos)
	for indice := 1; indice < len(bultos); indice++ {
		if bultos[indice-1] == bultos[indice] {
			return errorContenedor("plan_contenedor.bultos_invalidos")
		}
	}
	if !imagenContenedorDigeridaValida(plan.ImagenRef) {
		return errorContenedor("plan_contenedor.imagen_invalida")
	}
	if plan.VCPU < 1 || plan.VCPU > 32 || plan.MemoriaMiB < 64 ||
		plan.MemoriaMiB > 32_768 || plan.MaximoPIDs < 1 || plan.MaximoPIDs > 4_096 {
		return errorContenedor("plan_contenedor.recursos_invalidos")
	}
	if plan.LimiteTiempoMS == 0 || plan.LimiteDiscoPicoBytes == 0 ||
		plan.LimiteTokensAgente == 0 {
		return errorContenedor("plan_contenedor.limites_invalidos")
	}
	return nil
}

func validarRespuestaContenedorActiva(
	plan PlanLanzamientoContenedorV1,
	respuesta RespuestaContenedorV1,
) error {
	if respuesta.Referencia != plan.EjecucionRef || respuesta.Cerca != plan.Cerca ||
		respuesta.VCPU != plan.VCPU || respuesta.MemoriaMiB != plan.MemoriaMiB {
		return errorContenedor("contenedores.respuesta_invalida")
	}
	if err := validarRespuestaContenedorBasica(respuesta); err != nil {
		return err
	}
	if respuesta.Identidad == nil || respuesta.Identidad.RunLabel != plan.RunRef {
		return errorContenedor("contenedores.identidad_invalida")
	}
	return validarLivenessContenedor(respuesta, "activo", true, "running")
}

func validarRespuestaContenedorObservada(respuesta RespuestaContenedorV1) error {
	if err := validarRespuestaContenedorBasica(respuesta); err != nil {
		return err
	}
	switch respuesta.Estado {
	case "activo":
		return validarLivenessContenedor(respuesta, "activo", true, "running")
	case "detenido":
		return validarLivenessContenedor(respuesta, "detenido", false, "removed")
	default:
		return errorContenedor("contenedores.estado_invalido")
	}
}

func validarRespuestaContenedorBasica(respuesta RespuestaContenedorV1) error {
	if !referenciaEjecucionContenedorValida(respuesta.Referencia) ||
		respuesta.Revision == 0 || respuesta.Revision > maximoEnteroDurableV1 ||
		respuesta.Cerca == 0 || respuesta.Cerca > maximoEnteroDurableV1 ||
		respuesta.VCPU < 1 || respuesta.VCPU > 32 || respuesta.MemoriaMiB < 64 ||
		respuesta.MemoriaMiB > 32_768 {
		return errorContenedor("contenedores.respuesta_invalida")
	}
	switch respuesta.Estado {
	case "reservado", "activo", "ambiguo", "detenido":
	default:
		return errorContenedor("contenedores.estado_invalido")
	}
	if (respuesta.RecursoVivo == nil) != (respuesta.EstadoMotor == nil) {
		return errorContenedor("contenedores.liveness_invalida")
	}
	if respuesta.Identidad == nil {
		if respuesta.Estado == "activo" {
			return errorContenedor("contenedores.identidad_invalida")
		}
		return nil
	}
	identidad := respuesta.Identidad
	if !sha256Valido(identidad.ContainerID) || identidad.PIDObservado == 0 ||
		!ownerContenedorValido(identidad.OwnerLabel) ||
		identidad.ExecutionLabel != respuesta.Referencia ||
		!referenciaExternaContenedorValida(identidad.RunLabel) || len(identidad.RunLabel) > 160 ||
		identidad.Generation != respuesta.Cerca {
		return errorContenedor("contenedores.identidad_invalida")
	}
	return nil
}

func validarLivenessContenedor(
	respuesta RespuestaContenedorV1,
	estado string,
	vivo bool,
	estadoMotor string,
) error {
	if respuesta.Estado != estado || respuesta.RecursoVivo == nil ||
		*respuesta.RecursoVivo != vivo || respuesta.EstadoMotor == nil ||
		*respuesta.EstadoMotor != estadoMotor {
		return errorContenedor("contenedores.liveness_invalida")
	}
	return nil
}

func validarEstructuraRespuestaContenedor(datos []byte) error {
	var respuesta RespuestaContenedorV1
	if !decodificarObjetoJSONEstricto(datos, esquemaRespuestaContenedorV1, &respuesta) {
		return errors.New("respuesta JSON de contenedor invalida")
	}
	return nil
}

func validarEstructuraRespuestaOperacionContenedor(datos []byte) error {
	var respuesta RespuestaOperacionContenedorV1
	if !decodificarObjetoJSONEstricto(datos, esquemaRespuestaOperacionContenedorV1, &respuesta) {
		return errors.New("respuesta JSON de operacion contenedor invalida")
	}
	return nil
}

func claveIdempotenciaContenedorValida(valor string) bool {
	if valor == "" || len(valor) > 128 {
		return false
	}
	for _, caracter := range []byte(valor) {
		if !(caracter >= 'a' && caracter <= 'z' || caracter >= 'A' && caracter <= 'Z' ||
			caracter >= '0' && caracter <= '9' || caracter == '-' || caracter == '_' ||
			caracter == '.' || caracter == ':' || caracter == '/') {
			return false
		}
	}
	return true
}

func referenciaEjecucionContenedorValida(valor string) bool {
	sufijo := strings.TrimPrefix(valor, "ejecucion:")
	if sufijo == valor || sufijo == "" || len(valor) > 96 {
		return false
	}
	for _, caracter := range []byte(sufijo) {
		if !(caracter >= 'a' && caracter <= 'z' || caracter >= '0' && caracter <= '9' ||
			caracter == '-' || caracter == '_') {
			return false
		}
	}
	return true
}

func referenciaExternaContenedorValida(valor string) bool {
	if valor == "" || len(valor) > 512 || !utf8.ValidString(valor) || strings.TrimSpace(valor) != valor {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) {
			return false
		}
	}
	return true
}

func imagenContenedorDigeridaValida(imagen string) bool {
	indice := strings.LastIndex(imagen, "@sha256:")
	if indice <= 0 || indice > 255 || indice+8 >= len(imagen) {
		return false
	}
	repositorio, digest := imagen[:indice], imagen[indice+8:]
	if repositorio == "" || len(repositorio) > 255 || !utf8.ValidString(repositorio) {
		return false
	}
	for _, caracter := range repositorio {
		if unicode.IsControl(caracter) {
			return false
		}
	}
	return sha256Valido(digest)
}

func ownerContenedorValido(owner string) bool {
	if owner == "v1" {
		return true
	}
	digest := strings.TrimPrefix(owner, "v1:")
	return digest != owner && sha256Valido(digest)
}

func anexarCampoContenedor(destino []byte, campo string) []byte {
	destino = binary.BigEndian.AppendUint32(destino, uint32(len(campo)))
	return append(destino, campo...)
}

func segmentoRutaContenedor(valor string) string {
	const hexMayusculas = "0123456789ABCDEF"
	var resultado strings.Builder
	resultado.Grow(len(valor))
	for _, caracter := range []byte(valor) {
		if caracter >= 'a' && caracter <= 'z' || caracter >= 'A' && caracter <= 'Z' ||
			caracter >= '0' && caracter <= '9' || caracter == '-' || caracter == '.' ||
			caracter == '_' || caracter == '~' {
			resultado.WriteByte(caracter)
			continue
		}
		resultado.WriteByte('%')
		resultado.WriteByte(hexMayusculas[caracter>>4])
		resultado.WriteByte(hexMayusculas[caracter&0x0f])
	}
	return resultado.String()
}

func errorContenedor(codigo string) error { return &ErrorContenedorV1{Codigo: codigo} }
