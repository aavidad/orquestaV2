package microvm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"
)

const (
	maximoArgumentosOrdenContenedorV1      = 128
	maximoBytesArgumentoContenedorV1       = 4_096
	maximoBytesArgumentosContenedorV1      = 60 * 1_024
	maximoVariablesEntornoContenedorV1     = 128
	maximoBytesVariableEntornoContenedorV1 = 8_192
	maximoBytesEntornoContenedorV1         = 64 * 1_024
	maximoBytesStdinContenedorV1           = 1_048_576
	maximoPlazoOrdenContenedorV1           = 300_000
	maximoSalidaOrdenContenedorV1          = 1_048_576
	maximoCuerpoOrdenContenedorV1          = 2 * 1_024 * 1_024
	maximoCuerpoEntradaContenedorV1        = 12 * 1_024 * 1_024
	maximoCuerpoSalidaContenedorV1         = 16 * 1_024
)

// SolicitudOrdenContenedorV1 cerca una orden no interactiva al snapshot
// lifecycle de un contenedor activo.
type SolicitudOrdenContenedorV1 struct {
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

// SolicitudSincronizacionEntradaContenedorV1 materializa un árbol sellado sin
// compartir rutas del host con Agente MicroVM.
type SolicitudSincronizacionEntradaContenedorV1 struct {
	RevisionEsperada uint64             `json:"revision_esperada"`
	Cerca            uint64             `json:"cerca"`
	DestinoRelativo  string             `json:"destino_relativo"`
	RaizSHA256       string             `json:"raiz_sha256"`
	Archivos         []ArchivoContenido `json:"archivos"`
}

type SolicitudSincronizacionSalidaContenedorV1 struct {
	RevisionEsperada uint64 `json:"revision_esperada"`
	Cerca            uint64 `json:"cerca"`
	OrigenRelativo   string `json:"origen_relativo"`
	MaximoArchivos   uint32 `json:"maximo_archivos"`
	MaximoBytes      uint64 `json:"maximo_bytes"`
}

// RespuestaSincronizacionContenedorV1 es el receipt durable de la operación
// física ya confirmada por el huésped.
type RespuestaSincronizacionContenedorV1 struct {
	Referencia                  string `json:"referencia"`
	IdentificadorSincronizacion string `json:"identificador_sincronizacion"`
	Cerca                       uint64 `json:"cerca"`
	RutaRelativa                string `json:"ruta_relativa"`
	RaizSHA256                  string `json:"raiz_sha256"`
	Archivos                    uint32 `json:"archivos"`
	Bytes                       uint64 `json:"bytes"`
}

type RespuestaSincronizacionSalidaContenedorV1 struct {
	Comprobante RespuestaSincronizacionContenedorV1 `json:"comprobante"`
	Archivos    []ArchivoContenido                  `json:"archivos"`
}

var esquemaRespuestaOrdenContenedorV1 = esquemaObjetoJSONEstricto{
	"referencia": esquemaEscalarJSONEstricto, "codigo_salida": esquemaEscalarJSONEstrictoOpcional,
	"senal": esquemaEscalarJSONEstrictoOpcional, "agotada": esquemaEscalarJSONEstricto,
	"stdout_base64": esquemaEscalarJSONEstricto, "stderr_base64": esquemaEscalarJSONEstricto,
	"salida_truncada": esquemaEscalarJSONEstricto,
}

var esquemaVariableEntornoContenedorV1 = esquemaObjetoJSONEstricto{
	"nombre": esquemaEscalarJSONEstricto,
	"valor":  esquemaEscalarJSONEstricto,
}

var esquemaSolicitudOrdenContenedorV1 = esquemaObjetoJSONEstricto{
	"revision_esperada": esquemaEscalarJSONEstricto, "cerca": esquemaEscalarJSONEstricto,
	"programa":           esquemaEscalarJSONEstricto,
	"argumentos":         esquemaArrayEstricto(esquemaEscalarJSONEstricto),
	"directorio_trabajo": esquemaEscalarJSONEstricto,
	"entorno":            esquemaArrayEstricto(esquemaObjetoEstricto(esquemaVariableEntornoContenedorV1)),
	"stdin_base64":       esquemaEscalarJSONEstricto, "plazo_milisegundos": esquemaEscalarJSONEstricto,
	"maximo_salida_bytes": esquemaEscalarJSONEstricto,
}

var esquemaArchivoContenidoContenedorV1 = esquemaObjetoJSONEstricto{
	"ruta_relativa": esquemaEscalarJSONEstricto, "ejecutable": esquemaEscalarJSONEstricto,
	"sha256": esquemaEscalarJSONEstricto, "contenido_base64": esquemaEscalarJSONEstricto,
}

var esquemaSolicitudSincronizacionEntradaContenedorV1 = esquemaObjetoJSONEstricto{
	"revision_esperada": esquemaEscalarJSONEstricto, "cerca": esquemaEscalarJSONEstricto,
	"destino_relativo": esquemaEscalarJSONEstricto, "raiz_sha256": esquemaEscalarJSONEstricto,
	"archivos": esquemaArrayEstricto(esquemaObjetoEstricto(esquemaArchivoContenidoContenedorV1)),
}

var esquemaSolicitudSincronizacionSalidaContenedorV1 = esquemaObjetoJSONEstricto{
	"revision_esperada": esquemaEscalarJSONEstricto, "cerca": esquemaEscalarJSONEstricto,
	"origen_relativo": esquemaEscalarJSONEstricto, "maximo_archivos": esquemaEscalarJSONEstricto,
	"maximo_bytes": esquemaEscalarJSONEstricto,
}

var esquemaRespuestaSincronizacionContenedorV1 = esquemaObjetoJSONEstricto{
	"referencia":                   esquemaEscalarJSONEstricto,
	"identificador_sincronizacion": esquemaEscalarJSONEstricto,
	"cerca":                        esquemaEscalarJSONEstricto, "ruta_relativa": esquemaEscalarJSONEstricto,
	"raiz_sha256": esquemaEscalarJSONEstricto, "archivos": esquemaEscalarJSONEstricto,
	"bytes": esquemaEscalarJSONEstricto,
}

var esquemaRespuestaSincronizacionSalidaContenedorV1 = esquemaObjetoJSONEstricto{
	"comprobante": esquemaObjetoEstricto(esquemaRespuestaSincronizacionContenedorV1),
	"archivos":    esquemaArrayEstricto(esquemaObjetoEstricto(esquemaArchivoContenidoContenedorV1)),
}

func (c *Cliente) EjecutarOrdenContenedor(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudOrdenContenedorV1,
) (RespuestaOrden, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaOrden{}, err
	}
	cuerpo, err := CodificarSolicitudOrdenContenedorV1(solicitud)
	if err != nil {
		return RespuestaOrden{}, err
	}
	return c.ejecutarOrdenContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

// CodificarSolicitudOrdenContenedorV1 fija el cuerpo de una orden antes de su
// primer intento.
func CodificarSolicitudOrdenContenedorV1(
	solicitud SolicitudOrdenContenedorV1,
) (json.RawMessage, error) {
	if err := validarRevisionCercaTrabajoContenedor(solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return nil, err
	}
	if err := validarOrdenContenedor(solicitud); err != nil {
		return nil, err
	}
	return codificarSolicitudTrabajoContenedor(solicitud, maximoCuerpoOrdenContenedorV1)
}

// EjecutarOrdenContenedorCodificada reenvía sin reserializar una orden
// previamente persistida.
func (c *Cliente) EjecutarOrdenContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	cuerpo json.RawMessage,
) (RespuestaOrden, error) {
	var solicitud SolicitudOrdenContenedorV1
	if len(cuerpo) > maximoCuerpoOrdenContenedorV1 ||
		!decodificarObjetoJSONEstricto(cuerpo, esquemaSolicitudOrdenContenedorV1, &solicitud) {
		return RespuestaOrden{}, errorContenedor("contenedores.solicitud_orden_invalida")
	}
	return c.ejecutarOrdenContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

func (c *Cliente) ejecutarOrdenContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudOrdenContenedorV1,
	cuerpo []byte,
) (RespuestaOrden, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaOrden{}, err
	}
	if len(cuerpo) > maximoCuerpoOrdenContenedorV1 {
		return RespuestaOrden{}, errorContenedor("contenedores.solicitud_orden_invalida")
	}
	if err := validarOrdenContenedor(solicitud); err != nil {
		return RespuestaOrden{}, err
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaOrden](
		ctx,
		c,
		http.MethodPost,
		"/v1/contenedores/"+segmentoRutaContenedor(referencia)+"/ordenes",
		clave,
		cuerpo,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaOrden{}, err
	}
	if err := validarRespuestaOrdenContenedor(referencia, solicitud.MaximoSalidaBytes, respuesta); err != nil {
		return RespuestaOrden{}, err
	}
	return respuesta, nil
}

func (c *Cliente) SincronizarEntradaContenedor(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudSincronizacionEntradaContenedorV1,
) (RespuestaSincronizacionContenedorV1, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaSincronizacionContenedorV1{}, err
	}
	cuerpo, err := CodificarSolicitudSincronizacionEntradaContenedorV1(solicitud)
	if err != nil {
		return RespuestaSincronizacionContenedorV1{}, err
	}
	return c.sincronizarEntradaContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

// CodificarSolicitudSincronizacionEntradaContenedorV1 fija el árbol sellado
// antes de su primer intento de materialización.
func CodificarSolicitudSincronizacionEntradaContenedorV1(
	solicitud SolicitudSincronizacionEntradaContenedorV1,
) (json.RawMessage, error) {
	if err := validarRevisionCercaTrabajoContenedor(solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return nil, err
	}
	if _, err := validarEntradaContenedor(solicitud); err != nil {
		return nil, err
	}
	return codificarSolicitudTrabajoContenedor(solicitud, maximoCuerpoEntradaContenedorV1)
}

// SincronizarEntradaContenedorCodificada reenvía sin reserializar un árbol
// sellado previamente persistido.
func (c *Cliente) SincronizarEntradaContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	cuerpo json.RawMessage,
) (RespuestaSincronizacionContenedorV1, error) {
	var solicitud SolicitudSincronizacionEntradaContenedorV1
	if len(cuerpo) > maximoCuerpoEntradaContenedorV1 ||
		!decodificarObjetoJSONEstricto(
			cuerpo,
			esquemaSolicitudSincronizacionEntradaContenedorV1,
			&solicitud,
		) {
		return RespuestaSincronizacionContenedorV1{}, errorContenedor("contenedores.solicitud_sincronizacion_invalida")
	}
	return c.sincronizarEntradaContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

func (c *Cliente) sincronizarEntradaContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudSincronizacionEntradaContenedorV1,
	cuerpo []byte,
) (RespuestaSincronizacionContenedorV1, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaSincronizacionContenedorV1{}, err
	}
	if len(cuerpo) > maximoCuerpoEntradaContenedorV1 {
		return RespuestaSincronizacionContenedorV1{}, errorContenedor("contenedores.solicitud_sincronizacion_invalida")
	}
	bytes, err := validarEntradaContenedor(solicitud)
	if err != nil {
		return RespuestaSincronizacionContenedorV1{}, err
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaSincronizacionContenedorV1](
		ctx,
		c,
		http.MethodPost,
		"/v1/contenedores/"+segmentoRutaContenedor(referencia)+"/sincronizaciones/entrada",
		clave,
		cuerpo,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaSincronizacionContenedorV1{}, err
	}
	if respuesta.Referencia != referencia || respuesta.IdentificadorSincronizacion != clave ||
		respuesta.Cerca != solicitud.Cerca || respuesta.RutaRelativa != solicitud.DestinoRelativo ||
		respuesta.RaizSHA256 != solicitud.RaizSHA256 ||
		respuesta.Archivos != uint32(len(solicitud.Archivos)) || respuesta.Bytes != bytes {
		return RespuestaSincronizacionContenedorV1{}, errorContenedor("contenedores.respuesta_sincronizacion_invalida")
	}
	return respuesta, nil
}

func (c *Cliente) SincronizarSalidaContenedor(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudSincronizacionSalidaContenedorV1,
) (RespuestaSincronizacionSalidaContenedorV1, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaSincronizacionSalidaContenedorV1{}, err
	}
	cuerpo, err := CodificarSolicitudSincronizacionSalidaContenedorV1(solicitud)
	if err != nil {
		return RespuestaSincronizacionSalidaContenedorV1{}, err
	}
	return c.sincronizarSalidaContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

// CodificarSolicitudSincronizacionSalidaContenedorV1 fija la solicitud de
// recuperación antes de su primer intento.
func CodificarSolicitudSincronizacionSalidaContenedorV1(
	solicitud SolicitudSincronizacionSalidaContenedorV1,
) (json.RawMessage, error) {
	if err := validarRevisionCercaTrabajoContenedor(solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return nil, err
	}
	if err := validarSalidaContenedor(solicitud); err != nil {
		return nil, err
	}
	return codificarSolicitudTrabajoContenedor(solicitud, maximoCuerpoSalidaContenedorV1)
}

// SincronizarSalidaContenedorCodificada reenvía sin reserializar una solicitud
// de recuperación previamente persistida.
func (c *Cliente) SincronizarSalidaContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	cuerpo json.RawMessage,
) (RespuestaSincronizacionSalidaContenedorV1, error) {
	var solicitud SolicitudSincronizacionSalidaContenedorV1
	if len(cuerpo) > maximoCuerpoSalidaContenedorV1 ||
		!decodificarObjetoJSONEstricto(
			cuerpo,
			esquemaSolicitudSincronizacionSalidaContenedorV1,
			&solicitud,
		) {
		return RespuestaSincronizacionSalidaContenedorV1{}, errorContenedor("contenedores.solicitud_sincronizacion_invalida")
	}
	return c.sincronizarSalidaContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

func (c *Cliente) sincronizarSalidaContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudSincronizacionSalidaContenedorV1,
	cuerpo []byte,
) (RespuestaSincronizacionSalidaContenedorV1, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaSincronizacionSalidaContenedorV1{}, err
	}
	if len(cuerpo) > maximoCuerpoSalidaContenedorV1 {
		return RespuestaSincronizacionSalidaContenedorV1{}, errorContenedor("contenedores.solicitud_sincronizacion_invalida")
	}
	if err := validarSalidaContenedor(solicitud); err != nil {
		return RespuestaSincronizacionSalidaContenedorV1{}, err
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaSincronizacionSalidaContenedorV1](
		ctx,
		c,
		http.MethodPost,
		"/v1/contenedores/"+segmentoRutaContenedor(referencia)+"/sincronizaciones/salida",
		clave,
		cuerpo,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaSincronizacionSalidaContenedorV1{}, err
	}
	bytes, raiz, err := acreditarArchivosContenedor(
		respuesta.Archivos,
		solicitud.MaximoArchivos,
		solicitud.MaximoBytes,
	)
	if err != nil {
		return RespuestaSincronizacionSalidaContenedorV1{}, err
	}
	comprobante := respuesta.Comprobante
	if comprobante.Referencia != referencia || comprobante.IdentificadorSincronizacion != clave ||
		comprobante.Cerca != solicitud.Cerca || comprobante.RutaRelativa != solicitud.OrigenRelativo ||
		comprobante.RaizSHA256 != raiz || comprobante.Archivos != uint32(len(respuesta.Archivos)) ||
		comprobante.Bytes != bytes {
		return RespuestaSincronizacionSalidaContenedorV1{}, errorContenedor("contenedores.respuesta_sincronizacion_invalida")
	}
	return respuesta, nil
}

func codificarSolicitudTrabajoContenedor[S any](
	solicitud S,
	maximo int,
) (json.RawMessage, error) {
	codificada, err := json.Marshal(solicitud)
	if err != nil {
		return nil, errorContenedor("contenedores.codificacion_fallida")
	}
	if len(codificada) > maximo {
		return nil, errorContenedor("contenedores.solicitud_sobredimensionada")
	}
	return codificada, nil
}

func validarSalidaContenedor(solicitud SolicitudSincronizacionSalidaContenedorV1) error {
	if !utf8.ValidString(solicitud.OrigenRelativo) || !rutaBaseValida(solicitud.OrigenRelativo) ||
		solicitud.MaximoArchivos == 0 || solicitud.MaximoArchivos > MaximoArchivosSincronizacion ||
		solicitud.MaximoBytes == 0 || solicitud.MaximoBytes > MaximoBytesSincronizacion {
		return errorContenedor("contenedores.solicitud_sincronizacion_invalida")
	}
	return nil
}

func validarFronteraTrabajoContenedor(
	clave string,
	referencia string,
	revision uint64,
	cerca uint64,
) error {
	if !claveIdempotenciaContenedorValida(clave) {
		return errorContenedor("contenedores.clave_invalida")
	}
	if !referenciaEjecucionContenedorValida(referencia) {
		return errorContenedor("contenedores.referencia_invalida")
	}
	return validarRevisionCercaTrabajoContenedor(revision, cerca)
}

func validarRevisionCercaTrabajoContenedor(revision uint64, cerca uint64) error {
	if revision == 0 || revision > maximoEnteroDurableV1 ||
		cerca == 0 || cerca > maximoEnteroDurableV1 {
		return errorContenedor("contenedores.trabajo_invalido")
	}
	return nil
}

func validarOrdenContenedor(solicitud SolicitudOrdenContenedorV1) error {
	if !utf8.ValidString(solicitud.Programa) || !strings.HasPrefix(solicitud.Programa, "/") ||
		len(solicitud.Programa) > 256 || strings.ContainsRune(solicitud.Programa, '\x00') ||
		contieneSegmentoPadre(solicitud.Programa) || solicitud.Argumentos == nil ||
		len(solicitud.Argumentos) > maximoArgumentosOrdenContenedorV1 || solicitud.Entorno == nil ||
		!utf8.ValidString(solicitud.DirectorioTrabajo) || !rutaBaseValida(solicitud.DirectorioTrabajo) ||
		solicitud.PlazoMilisegundos == 0 || solicitud.PlazoMilisegundos > maximoPlazoOrdenContenedorV1 ||
		solicitud.MaximoSalidaBytes == 0 || solicitud.MaximoSalidaBytes > maximoSalidaOrdenContenedorV1 {
		return errorContenedor("contenedores.solicitud_orden_invalida")
	}
	bytesArgumentos := len(solicitud.Programa)
	for _, argumento := range solicitud.Argumentos {
		if !utf8.ValidString(argumento) || len(argumento) > maximoBytesArgumentoContenedorV1 ||
			strings.ContainsRune(argumento, '\x00') {
			return errorContenedor("contenedores.solicitud_orden_invalida")
		}
		bytesArgumentos += len(argumento)
	}
	if bytesArgumentos > maximoBytesArgumentosContenedorV1 || !entornoContenedorValido(solicitud.Entorno) {
		return errorContenedor("contenedores.solicitud_orden_invalida")
	}
	if len(solicitud.StdinBase64) > longitudBase64MaximaContenedor(maximoBytesStdinContenedorV1) {
		return errorContenedor("contenedores.solicitud_orden_invalida")
	}
	stdin, correcto := decodificarBase64CanonicoContenedor(solicitud.StdinBase64)
	if !correcto || len(stdin) > maximoBytesStdinContenedorV1 {
		return errorContenedor("contenedores.solicitud_orden_invalida")
	}
	return nil
}

func validarRespuestaOrdenContenedor(
	referencia string,
	maximoSalida uint32,
	respuesta RespuestaOrden,
) error {
	if respuesta.Referencia != referencia ||
		len(respuesta.StdoutBase64) > longitudBase64MaximaContenedor(uint64(maximoSalida)) ||
		len(respuesta.StderrBase64) > longitudBase64MaximaContenedor(uint64(maximoSalida)) {
		return errorContenedor("contenedores.respuesta_orden_invalida")
	}
	stdout, stdoutValido := decodificarBase64CanonicoContenedor(respuesta.StdoutBase64)
	stderr, stderrValido := decodificarBase64CanonicoContenedor(respuesta.StderrBase64)
	resultadoAusente := respuesta.CodigoSalida == nil && respuesta.Senal == nil
	if !stdoutValido || !stderrValido ||
		uint64(len(stdout))+uint64(len(stderr)) > uint64(maximoSalida) ||
		respuesta.CodigoSalida != nil && respuesta.Senal != nil || resultadoAusente && !respuesta.Agotada ||
		respuesta.CodigoSalida != nil && *respuesta.CodigoSalida < 0 ||
		respuesta.Senal != nil && *respuesta.Senal <= 0 {
		return errorContenedor("contenedores.respuesta_orden_invalida")
	}
	return nil
}

func validarEntradaContenedor(solicitud SolicitudSincronizacionEntradaContenedorV1) (uint64, error) {
	if solicitud.Archivos == nil || !utf8.ValidString(solicitud.DestinoRelativo) ||
		!rutaBaseValida(solicitud.DestinoRelativo) {
		return 0, errorContenedor("contenedores.solicitud_sincronizacion_invalida")
	}
	bytes, raiz, err := acreditarArchivosContenedor(
		solicitud.Archivos,
		MaximoArchivosSincronizacion,
		MaximoBytesSincronizacion,
	)
	if err != nil {
		return 0, err
	}
	if solicitud.RaizSHA256 != raiz {
		return 0, errorContenedor("contenedores.raiz_no_acreditada")
	}
	return bytes, nil
}

func acreditarArchivosContenedor(
	archivos []ArchivoContenido,
	maximoArchivos uint32,
	maximoBytes uint64,
) (uint64, string, error) {
	if archivos == nil {
		return 0, "", errorContenedor("contenedores.contenido_invalido")
	}
	for _, archivo := range archivos {
		if !utf8.ValidString(archivo.RutaRelativa) {
			return 0, "", errorContenedor("contenedores.contenido_invalido")
		}
		if len(archivo.ContenidoBase64) > longitudBase64MaximaContenedor(maximoBytes) {
			return 0, "", errorContenedor("contenedores.contenido_invalido")
		}
		if _, correcto := decodificarBase64CanonicoContenedor(archivo.ContenidoBase64); !correcto {
			return 0, "", errorContenedor("contenedores.contenido_invalido")
		}
	}
	descriptores, bytes, err := validarArchivos(archivos, maximoArchivos, maximoBytes)
	if err != nil {
		return 0, "", errorContenedor("contenedores.contenido_invalido")
	}
	return bytes, calcularRaizDescriptores(descriptores), nil
}

func entornoContenedorValido(entorno []VariableEntorno) bool {
	if len(entorno) > maximoVariablesEntornoContenedorV1 {
		return false
	}
	vistas := make(map[string]struct{}, len(entorno))
	bytes := 0
	for _, variable := range entorno {
		if !nombreVariableContenedorValido(variable.Nombre) ||
			!utf8.ValidString(variable.Valor) || len(variable.Valor) > maximoBytesVariableEntornoContenedorV1 ||
			strings.ContainsRune(variable.Valor, '\x00') {
			return false
		}
		if _, repetida := vistas[variable.Nombre]; repetida {
			return false
		}
		vistas[variable.Nombre] = struct{}{}
		bytes += len(variable.Nombre) + len(variable.Valor)
	}
	return bytes <= maximoBytesEntornoContenedorV1
}

func nombreVariableContenedorValido(nombre string) bool {
	if nombre == "" || len(nombre) > 128 || strings.HasPrefix(nombre, "AGENTE_MICROVM_") {
		return false
	}
	for indice, caracter := range []byte(nombre) {
		if indice == 0 {
			if caracter != '_' && (caracter < 'A' || caracter > 'Z') && (caracter < 'a' || caracter > 'z') {
				return false
			}
			continue
		}
		if caracter != '_' && (caracter < 'A' || caracter > 'Z') &&
			(caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') {
			return false
		}
	}
	return true
}

func contieneSegmentoPadre(ruta string) bool {
	for _, segmento := range strings.Split(ruta, "/") {
		if segmento == ".." {
			return true
		}
	}
	return false
}

func decodificarBase64CanonicoContenedor(valor string) ([]byte, bool) {
	if strings.ContainsAny(valor, "\r\n") {
		return nil, false
	}
	contenido, err := base64.StdEncoding.Strict().DecodeString(valor)
	return contenido, err == nil && base64.StdEncoding.EncodeToString(contenido) == valor
}

func longitudBase64MaximaContenedor(maximo uint64) int {
	return int((maximo + 2) / 3 * 4)
}
