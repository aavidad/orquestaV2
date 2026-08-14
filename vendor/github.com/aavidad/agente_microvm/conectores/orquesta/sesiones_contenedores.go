package microvm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maximoContenidoEventoSesionContenedorV1 = 8_192
	maximoCuerpoInicioSesionContenedorV1    = 16 * 1_024
	maximoCuerpoEntradaSesionContenedorV1   = 2 * 1_024 * 1_024
)

type SolicitudInicioSesionContenedorV1 struct {
	SesionRef              string `json:"sesion_ref"`
	RevisionEsperada       uint64 `json:"revision_esperada"`
	Cerca                  uint64 `json:"cerca"`
	EjecutorRef            string `json:"ejecutor_ref"`
	DirectorioTrabajo      string `json:"directorio_trabajo"`
	PlazoTotalMilisegundos uint64 `json:"plazo_total_milisegundos"`
	MaximoEventosBytes     uint64 `json:"maximo_eventos_bytes"`
}

type RespuestaInicioSesionContenedorV1 struct {
	Referencia  string  `json:"referencia"`
	SesionRef   string  `json:"sesion_ref"`
	Cerca       uint64  `json:"cerca"`
	Iniciada    bool    `json:"iniciada"`
	CodigoError *string `json:"codigo_error"`
}

type SolicitudEntradaSesionContenedorV1 struct {
	RevisionEsperada uint64 `json:"revision_esperada"`
	Cerca            uint64 `json:"cerca"`
	ContenidoBase64  string `json:"contenido_base64"`
	CerrarStdin      bool   `json:"cerrar_stdin"`
}

type RespuestaEntradaSesionContenedorV1 struct {
	Referencia  string  `json:"referencia"`
	SesionRef   string  `json:"sesion_ref"`
	EntradaRef  string  `json:"entrada_ref"`
	Cerca       uint64  `json:"cerca"`
	Aceptada    bool    `json:"aceptada"`
	CodigoError *string `json:"codigo_error"`
}

type ConsultaEventosSesionContenedorV1 struct {
	RevisionEsperada uint64 `json:"revision_esperada"`
	Cerca            uint64 `json:"cerca"`
	DespuesDe        uint64 `json:"despues_de"`
	MaximoEventos    uint16 `json:"maximo_eventos"`
	MaximoBytes      uint64 `json:"maximo_bytes"`
}

type TipoEventoSesionContenedorV1 string

const (
	TipoEventoSesionContenedorIniciada       TipoEventoSesionContenedorV1 = "iniciada"
	TipoEventoSesionContenedorStdout         TipoEventoSesionContenedorV1 = "stdout"
	TipoEventoSesionContenedorStderr         TipoEventoSesionContenedorV1 = "stderr"
	TipoEventoSesionContenedorSalidaTruncada TipoEventoSesionContenedorV1 = "salida_truncada"
	TipoEventoSesionContenedorFinalizada     TipoEventoSesionContenedorV1 = "finalizada"
)

type EventoSesionContenedorV1 struct {
	SesionRef       string                       `json:"sesion_ref"`
	Secuencia       uint64                       `json:"secuencia"`
	Tipo            TipoEventoSesionContenedorV1 `json:"tipo"`
	ContenidoBase64 string                       `json:"contenido_base64"`
	CodigoSalida    *int32                       `json:"codigo_salida"`
	Senal           *int32                       `json:"senal"`
	Agotada         bool                         `json:"agotada"`
	CodigoError     *string                      `json:"codigo_error"`
}

type RespuestaEventosSesionContenedorV1 struct {
	Referencia      string                     `json:"referencia"`
	SesionRef       string                     `json:"sesion_ref"`
	Cerca           uint64                     `json:"cerca"`
	Eventos         []EventoSesionContenedorV1 `json:"eventos"`
	SiguienteCursor uint64                     `json:"siguiente_cursor"`
	Terminal        bool                       `json:"terminal"`
	CodigoError     *string                    `json:"codigo_error"`
}

var esquemaRespuestaInicioSesionContenedorV1 = esquemaObjetoJSONEstricto{
	"referencia": esquemaEscalarJSONEstricto, "sesion_ref": esquemaEscalarJSONEstricto,
	"cerca": esquemaEscalarJSONEstricto, "iniciada": esquemaEscalarJSONEstricto,
	"codigo_error": esquemaEscalarJSONEstrictoOpcional,
}

var esquemaSolicitudInicioSesionContenedorV1 = esquemaObjetoJSONEstricto{
	"sesion_ref": esquemaEscalarJSONEstricto, "revision_esperada": esquemaEscalarJSONEstricto,
	"cerca": esquemaEscalarJSONEstricto, "ejecutor_ref": esquemaEscalarJSONEstricto,
	"directorio_trabajo":       esquemaEscalarJSONEstricto,
	"plazo_total_milisegundos": esquemaEscalarJSONEstricto,
	"maximo_eventos_bytes":     esquemaEscalarJSONEstricto,
}

var esquemaSolicitudEntradaSesionContenedorV1 = esquemaObjetoJSONEstricto{
	"revision_esperada": esquemaEscalarJSONEstricto, "cerca": esquemaEscalarJSONEstricto,
	"contenido_base64": esquemaEscalarJSONEstricto, "cerrar_stdin": esquemaEscalarJSONEstricto,
}

var esquemaRespuestaEntradaSesionContenedorV1 = esquemaObjetoJSONEstricto{
	"referencia": esquemaEscalarJSONEstricto, "sesion_ref": esquemaEscalarJSONEstricto,
	"entrada_ref": esquemaEscalarJSONEstricto, "cerca": esquemaEscalarJSONEstricto,
	"aceptada": esquemaEscalarJSONEstricto, "codigo_error": esquemaEscalarJSONEstrictoOpcional,
}

var esquemaEventoSesionContenedorV1 = esquemaObjetoJSONEstricto{
	"sesion_ref": esquemaEscalarJSONEstricto, "secuencia": esquemaEscalarJSONEstricto,
	"tipo": esquemaEscalarJSONEstricto, "contenido_base64": esquemaEscalarJSONEstricto,
	"codigo_salida": esquemaEscalarJSONEstrictoOpcional, "senal": esquemaEscalarJSONEstrictoOpcional,
	"agotada": esquemaEscalarJSONEstricto, "codigo_error": esquemaEscalarJSONEstrictoOpcional,
}

var esquemaRespuestaEventosSesionContenedorV1 = esquemaObjetoJSONEstricto{
	"referencia": esquemaEscalarJSONEstricto, "sesion_ref": esquemaEscalarJSONEstricto,
	"cerca":            esquemaEscalarJSONEstricto,
	"eventos":          esquemaArrayEstricto(esquemaObjetoEstricto(esquemaEventoSesionContenedorV1)),
	"siguiente_cursor": esquemaEscalarJSONEstricto, "terminal": esquemaEscalarJSONEstricto,
	"codigo_error": esquemaEscalarJSONEstrictoOpcional,
}

func (c *Cliente) IniciarSesionContenedor(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudInicioSesionContenedorV1,
) (RespuestaInicioSesionContenedorV1, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaInicioSesionContenedorV1{}, err
	}
	cuerpo, err := CodificarSolicitudInicioSesionContenedorV1(solicitud)
	if err != nil {
		return RespuestaInicioSesionContenedorV1{}, err
	}
	return c.iniciarSesionContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

// CodificarSolicitudInicioSesionContenedorV1 fija los bytes que el consumidor
// debe persistir antes del primer intento de iniciar la sesión.
func CodificarSolicitudInicioSesionContenedorV1(
	solicitud SolicitudInicioSesionContenedorV1,
) (json.RawMessage, error) {
	if err := validarSolicitudInicioSesionContenedor(solicitud); err != nil {
		return nil, err
	}
	codificada, err := json.Marshal(solicitud)
	if err != nil {
		return nil, errorContenedor("contenedores.codificacion_fallida")
	}
	return codificada, nil
}

// IniciarSesionContenedorCodificada reenvía sin reserializar una intención
// previamente persistida. La forma y la semántica se revalidan antes del UDS.
func (c *Cliente) IniciarSesionContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	cuerpo json.RawMessage,
) (RespuestaInicioSesionContenedorV1, error) {
	var solicitud SolicitudInicioSesionContenedorV1
	if len(cuerpo) > maximoCuerpoInicioSesionContenedorV1 ||
		!decodificarObjetoJSONEstricto(cuerpo, esquemaSolicitudInicioSesionContenedorV1, &solicitud) {
		return RespuestaInicioSesionContenedorV1{}, errorContenedor("contenedores.inicio_sesion_invalido")
	}
	return c.iniciarSesionContenedorCodificada(ctx, clave, referencia, solicitud, cuerpo)
}

func (c *Cliente) iniciarSesionContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudInicioSesionContenedorV1,
	cuerpo []byte,
) (RespuestaInicioSesionContenedorV1, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaInicioSesionContenedorV1{}, err
	}
	if err := validarSolicitudInicioSesionContenedor(solicitud); err != nil {
		return RespuestaInicioSesionContenedorV1{}, err
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaInicioSesionContenedorV1](
		ctx,
		c,
		http.MethodPost,
		"/v1/contenedores/"+segmentoRutaContenedor(referencia)+"/sesiones",
		clave,
		cuerpo,
		http.StatusCreated,
	)
	if err != nil {
		return RespuestaInicioSesionContenedorV1{}, err
	}
	if respuesta.Referencia != referencia || respuesta.SesionRef != solicitud.SesionRef ||
		respuesta.Cerca != solicitud.Cerca ||
		!resultadoSesionContenedorCoherente(respuesta.Iniciada, respuesta.CodigoError) {
		return RespuestaInicioSesionContenedorV1{}, errorContenedor("contenedores.respuesta_sesion_invalida")
	}
	return respuesta, nil
}

func (c *Cliente) EnviarEntradaSesionContenedor(
	ctx context.Context,
	clave string,
	referencia string,
	sesionRef string,
	solicitud SolicitudEntradaSesionContenedorV1,
) (RespuestaEntradaSesionContenedorV1, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaEntradaSesionContenedorV1{}, err
	}
	cuerpo, err := CodificarSolicitudEntradaSesionContenedorV1(solicitud)
	if err != nil {
		return RespuestaEntradaSesionContenedorV1{}, err
	}
	return c.enviarEntradaSesionContenedorCodificada(
		ctx, clave, referencia, sesionRef, solicitud, cuerpo,
	)
}

// CodificarSolicitudEntradaSesionContenedorV1 fija los bytes que el
// consumidor debe persistir antes del primer intento de entregar stdin.
func CodificarSolicitudEntradaSesionContenedorV1(
	solicitud SolicitudEntradaSesionContenedorV1,
) (json.RawMessage, error) {
	if err := validarSolicitudEntradaSesionContenedor(solicitud); err != nil {
		return nil, err
	}
	codificada, err := json.Marshal(solicitud)
	if err != nil {
		return nil, errorContenedor("contenedores.codificacion_fallida")
	}
	return codificada, nil
}

// EnviarEntradaSesionContenedorCodificada reenvía sin reserializar una
// intención previamente persistida. La forma y semántica se revalidan antes
// del UDS.
func (c *Cliente) EnviarEntradaSesionContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	sesionRef string,
	cuerpo json.RawMessage,
) (RespuestaEntradaSesionContenedorV1, error) {
	var solicitud SolicitudEntradaSesionContenedorV1
	if len(cuerpo) > maximoCuerpoEntradaSesionContenedorV1 ||
		!decodificarObjetoJSONEstricto(cuerpo, esquemaSolicitudEntradaSesionContenedorV1, &solicitud) {
		return RespuestaEntradaSesionContenedorV1{}, errorContenedor("contenedores.entrada_sesion_invalida")
	}
	return c.enviarEntradaSesionContenedorCodificada(
		ctx, clave, referencia, sesionRef, solicitud, cuerpo,
	)
}

func (c *Cliente) enviarEntradaSesionContenedorCodificada(
	ctx context.Context,
	clave string,
	referencia string,
	sesionRef string,
	solicitud SolicitudEntradaSesionContenedorV1,
	cuerpo []byte,
) (RespuestaEntradaSesionContenedorV1, error) {
	if err := validarFronteraTrabajoContenedor(clave, referencia, solicitud.RevisionEsperada, solicitud.Cerca); err != nil {
		return RespuestaEntradaSesionContenedorV1{}, err
	}
	if !referenciaSesionValida(sesionRef) {
		return RespuestaEntradaSesionContenedorV1{}, errorContenedor("contenedores.entrada_sesion_invalida")
	}
	if err := validarSolicitudEntradaSesionContenedor(solicitud); err != nil {
		return RespuestaEntradaSesionContenedorV1{}, err
	}
	respuesta, err := solicitarEstrictoConEstado[RespuestaEntradaSesionContenedorV1](
		ctx,
		c,
		http.MethodPost,
		"/v1/contenedores/"+segmentoRutaContenedor(referencia)+"/sesiones/"+
			segmentoRutaContenedor(sesionRef)+"/entradas",
		clave,
		cuerpo,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaEntradaSesionContenedorV1{}, err
	}
	if respuesta.Referencia != referencia || respuesta.SesionRef != sesionRef ||
		respuesta.EntradaRef != clave || respuesta.Cerca != solicitud.Cerca ||
		!resultadoSesionContenedorCoherente(respuesta.Aceptada, respuesta.CodigoError) {
		return RespuestaEntradaSesionContenedorV1{}, errorContenedor("contenedores.respuesta_sesion_invalida")
	}
	return respuesta, nil
}

func validarSolicitudInicioSesionContenedor(solicitud SolicitudInicioSesionContenedorV1) error {
	if solicitud.RevisionEsperada == 0 || solicitud.RevisionEsperada > maximoEnteroDurableV1 ||
		solicitud.Cerca == 0 || solicitud.Cerca > maximoEnteroDurableV1 ||
		!referenciaSesionValida(solicitud.SesionRef) || !ejecutorSesionValido(solicitud.EjecutorRef) ||
		!utf8.ValidString(solicitud.DirectorioTrabajo) || !rutaBaseValida(solicitud.DirectorioTrabajo) ||
		solicitud.PlazoTotalMilisegundos == 0 ||
		solicitud.PlazoTotalMilisegundos > MaximoPlazoSesionMilisegundosV1 ||
		solicitud.MaximoEventosBytes == 0 || solicitud.MaximoEventosBytes > MaximoEventosSesionBytesV1 {
		return errorContenedor("contenedores.inicio_sesion_invalido")
	}
	return nil
}

func validarSolicitudEntradaSesionContenedor(solicitud SolicitudEntradaSesionContenedorV1) error {
	if solicitud.RevisionEsperada == 0 || solicitud.RevisionEsperada > maximoEnteroDurableV1 ||
		solicitud.Cerca == 0 || solicitud.Cerca > maximoEnteroDurableV1 ||
		len(solicitud.ContenidoBase64) > longitudBase64MaximaContenedor(MaximoEntradaSesionBytesV1) {
		return errorContenedor("contenedores.entrada_sesion_invalida")
	}
	contenido, correcto := decodificarBase64CanonicoContenedor(solicitud.ContenidoBase64)
	if !correcto || len(contenido) > MaximoEntradaSesionBytesV1 ||
		len(contenido) == 0 && !solicitud.CerrarStdin {
		return errorContenedor("contenedores.entrada_sesion_invalida")
	}
	return nil
}

func (c *Cliente) LeerEventosSesionContenedor(
	ctx context.Context,
	referencia string,
	sesionRef string,
	consulta ConsultaEventosSesionContenedorV1,
) (RespuestaEventosSesionContenedorV1, error) {
	if !referenciaEjecucionContenedorValida(referencia) || !referenciaSesionValida(sesionRef) {
		return RespuestaEventosSesionContenedorV1{}, errorContenedor("contenedores.referencia_invalida")
	}
	if consulta.RevisionEsperada == 0 || consulta.RevisionEsperada > maximoEnteroDurableV1 ||
		consulta.Cerca == 0 || consulta.Cerca > maximoEnteroDurableV1 ||
		consulta.DespuesDe > maximoEnteroDurableV1 || consulta.MaximoEventos == 0 ||
		consulta.MaximoEventos > MaximoEventosPaginaSesionV1 || consulta.MaximoBytes == 0 ||
		consulta.MaximoBytes > MaximoEventosSesionBytesV1 {
		return RespuestaEventosSesionContenedorV1{}, errorContenedor("contenedores.consulta_sesion_invalida")
	}
	parametros := url.Values{}
	parametros.Set("revision_esperada", strconv.FormatUint(consulta.RevisionEsperada, 10))
	parametros.Set("cerca", strconv.FormatUint(consulta.Cerca, 10))
	parametros.Set("despues_de", strconv.FormatUint(consulta.DespuesDe, 10))
	parametros.Set("maximo_eventos", strconv.FormatUint(uint64(consulta.MaximoEventos), 10))
	parametros.Set("maximo_bytes", strconv.FormatUint(consulta.MaximoBytes, 10))
	respuesta, err := solicitarEstrictoConEstado[RespuestaEventosSesionContenedorV1](
		ctx,
		c,
		http.MethodGet,
		"/v1/contenedores/"+segmentoRutaContenedor(referencia)+"/sesiones/"+
			segmentoRutaContenedor(sesionRef)+"/eventos?"+parametros.Encode(),
		"",
		nil,
		http.StatusOK,
	)
	if err != nil {
		return RespuestaEventosSesionContenedorV1{}, err
	}
	if err := validarPaginaEventosSesionContenedor(referencia, sesionRef, consulta, respuesta); err != nil {
		return RespuestaEventosSesionContenedorV1{}, err
	}
	return respuesta, nil
}

func validarPaginaEventosSesionContenedor(
	referencia string,
	sesionRef string,
	consulta ConsultaEventosSesionContenedorV1,
	respuesta RespuestaEventosSesionContenedorV1,
) error {
	if respuesta.Referencia != referencia || respuesta.SesionRef != sesionRef ||
		respuesta.Cerca != consulta.Cerca || respuesta.Eventos == nil ||
		len(respuesta.Eventos) > int(consulta.MaximoEventos) ||
		respuesta.SiguienteCursor > maximoEnteroDurableV1 {
		return errorContenedor("contenedores.respuesta_eventos_invalida")
	}
	if respuesta.CodigoError != nil {
		if !codigoErrorSesionContenedorValido(*respuesta.CodigoError) || len(respuesta.Eventos) != 0 ||
			respuesta.SiguienteCursor != consulta.DespuesDe || respuesta.Terminal {
			return errorContenedor("contenedores.respuesta_eventos_invalida")
		}
		return nil
	}
	secuencia := consulta.DespuesDe
	finalizada := false
	bytes := uint64(0)
	for _, evento := range respuesta.Eventos {
		if secuencia == maximoEnteroDurableV1 {
			return errorContenedor("contenedores.respuesta_eventos_invalida")
		}
		secuencia++
		contenido, err := validarEventoSesionContenedor(sesionRef, secuencia, evento)
		if err != nil || finalizada {
			return errorContenedor("contenedores.respuesta_eventos_invalida")
		}
		if evento.Tipo == TipoEventoSesionContenedorFinalizada {
			finalizada = true
		}
		bytes += uint64(len(contenido))
		if bytes > consulta.MaximoBytes {
			return errorContenedor("contenedores.respuesta_eventos_invalida")
		}
	}
	if respuesta.SiguienteCursor != secuencia || finalizada && !respuesta.Terminal ||
		respuesta.Terminal && len(respuesta.Eventos) > 0 && !finalizada {
		return errorContenedor("contenedores.respuesta_eventos_invalida")
	}
	return nil
}

func validarEventoSesionContenedor(
	sesionRef string,
	secuencia uint64,
	evento EventoSesionContenedorV1,
) ([]byte, error) {
	if evento.SesionRef != sesionRef || evento.Secuencia != secuencia ||
		len(evento.ContenidoBase64) > longitudBase64MaximaContenedor(maximoContenidoEventoSesionContenedorV1) ||
		evento.CodigoError != nil && !codigoErrorSesionContenedorValido(*evento.CodigoError) {
		return nil, errorContenedor("contenedores.evento_sesion_invalido")
	}
	contenido, correcto := decodificarBase64CanonicoContenedor(evento.ContenidoBase64)
	if !correcto || len(contenido) > maximoContenidoEventoSesionContenedorV1 {
		return nil, errorContenedor("contenedores.evento_sesion_invalido")
	}
	coherente := false
	switch evento.Tipo {
	case TipoEventoSesionContenedorIniciada:
		coherente = len(contenido) == 0 && evento.CodigoSalida == nil && evento.Senal == nil &&
			!evento.Agotada && evento.CodigoError == nil
	case TipoEventoSesionContenedorStdout, TipoEventoSesionContenedorStderr:
		coherente = evento.CodigoSalida == nil && evento.Senal == nil && !evento.Agotada && evento.CodigoError == nil
	case TipoEventoSesionContenedorSalidaTruncada:
		coherente = len(contenido) == 0 && evento.CodigoSalida == nil && evento.Senal == nil &&
			!evento.Agotada && evento.CodigoError != nil
	case TipoEventoSesionContenedorFinalizada:
		coherente = len(contenido) == 0 && !(evento.CodigoSalida != nil && evento.Senal != nil)
	}
	if !coherente {
		return nil, errorContenedor("contenedores.evento_sesion_invalido")
	}
	return contenido, nil
}

func resultadoSesionContenedorCoherente(exito bool, codigo *string) bool {
	return exito && codigo == nil || !exito && codigo != nil && codigoErrorSesionContenedorValido(*codigo)
}

func codigoErrorSesionContenedorValido(codigo string) bool {
	if codigo == "" || len(codigo) > 96 || !utf8.ValidString(codigo) || strings.TrimSpace(codigo) != codigo {
		return false
	}
	for _, caracter := range []byte(codigo) {
		if (caracter < 'a' || caracter > 'z') && (caracter < 'A' || caracter > 'Z') &&
			(caracter < '0' || caracter > '9') && caracter != '.' && caracter != '_' && caracter != '-' {
			return false
		}
	}
	return true
}
