package microvm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maximoRespuestaBytes  int64  = 12*1024*1024 + 1
	maximoEnteroDurableV1 uint64 = 1<<63 - 1
)

// Cliente consume la API local sin conocer tipos internos de Agente MicroVM.
type Cliente struct {
	http *http.Client
}

// Nuevo construye un cliente confinado a una ruta Unix explícita.
func Nuevo(rutaSocket string) (*Cliente, error) {
	if !filepath.IsAbs(rutaSocket) || strings.ContainsRune(rutaSocket, '\x00') {
		return nil, &ErrorConfiguracion{Causa: "ruta_socket_invalida"}
	}
	marcador := &net.Dialer{}
	transporte := &http.Transport{
		Proxy:               nil,
		DisableCompression:  true,
		ForceAttemptHTTP2:   false,
		MaxIdleConns:        1,
		MaxIdleConnsPerHost: 1,
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return marcador.DialContext(ctx, "unix", rutaSocket)
		},
	}
	return &Cliente{
		http: &http.Client{Transport: transporte},
	}, nil
}

// LiberarConexiones cierra conexiones ociosas sin alterar ejecuciones remotas.
func (c *Cliente) LiberarConexiones() {
	if c != nil && c.http != nil {
		c.http.CloseIdleConnections()
	}
}

func (c *Cliente) Salud(ctx context.Context) (RespuestaSalud, error) {
	return solicitarSinCuerpo[RespuestaSalud](ctx, c, http.MethodGet, "/v1/salud")
}

func (c *Cliente) Capacidades(ctx context.Context) (RespuestaCapacidades, error) {
	return solicitarSinCuerpo[RespuestaCapacidades](ctx, c, http.MethodGet, "/v1/capacidades")
}

func (c *Cliente) Lanzar(
	ctx context.Context,
	clave string,
	solicitud SolicitudLanzamiento,
) (RespuestaEjecucion, error) {
	return mutar[RespuestaEjecucion](ctx, c, "/v1/ejecuciones", clave, solicitud)
}

func (c *Cliente) Observar(ctx context.Context, referencia string) (RespuestaEjecucion, error) {
	return solicitarSinCuerpo[RespuestaEjecucion](
		ctx,
		c,
		http.MethodGet,
		rutaEjecucion(referencia, ""),
	)
}

func (c *Cliente) RevisionTrabajo(
	ctx context.Context,
	referencia string,
) (RespuestaRevisionTrabajo, error) {
	return solicitarSinCuerpo[RespuestaRevisionTrabajo](
		ctx,
		c,
		http.MethodGet,
		rutaEjecucion(referencia, "/trabajo"),
	)
}

// IniciarSesion admite una sesión asíncrona en una ejecución disponible.
func (c *Cliente) IniciarSesion(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudIniciarSesionTrabajoV1,
) (RespuestaSesionTrabajoV1, error) {
	if err := validarInicioSesion(referencia, solicitud); err != nil {
		return RespuestaSesionTrabajoV1{}, err
	}
	respuesta, err := mutarEstricto[RespuestaSesionTrabajoV1](
		ctx,
		c,
		rutaEjecucion(referencia, "/sesiones"),
		clave,
		solicitud,
	)
	if err != nil {
		return RespuestaSesionTrabajoV1{}, err
	}
	if err := validarRespuestaSesion(referencia, solicitud.SesionRef, solicitud.Cerca, respuesta); err != nil {
		return RespuestaSesionTrabajoV1{}, err
	}
	if respuesta.Revision < solicitud.RevisionEsperada ||
		respuesta.RevisionTrabajo < solicitud.RevisionTrabajoEsperada {
		return RespuestaSesionTrabajoV1{}, errorSesion("sesion_trabajo.revision_invalida")
	}
	return respuesta, nil
}

// EnviarEntradaSesion entrega entrada idempotente o cierra stdin.
func (c *Cliente) EnviarEntradaSesion(
	ctx context.Context,
	clave string,
	referencia string,
	sesionRef string,
	solicitud SolicitudEntradaSesionTrabajoV1,
) (RespuestaSesionTrabajoV1, error) {
	if err := validarEntradaSesion(referencia, sesionRef, solicitud); err != nil {
		return RespuestaSesionTrabajoV1{}, err
	}
	respuesta, err := mutarEstricto[RespuestaSesionTrabajoV1](
		ctx,
		c,
		rutaSesion(referencia, sesionRef, "/entradas"),
		clave,
		solicitud,
	)
	if err != nil {
		return RespuestaSesionTrabajoV1{}, err
	}
	if err := validarRespuestaSesion(referencia, sesionRef, solicitud.Cerca, respuesta); err != nil {
		return RespuestaSesionTrabajoV1{}, err
	}
	if respuesta.Revision < solicitud.RevisionEsperada ||
		respuesta.RevisionTrabajo < solicitud.RevisionTrabajoEsperada ||
		respuesta.RevisionSesion < solicitud.RevisionSesionEsperada {
		return RespuestaSesionTrabajoV1{}, errorSesion("sesion_trabajo.revision_invalida")
	}
	return respuesta, nil
}

// LeerEventosSesion obtiene una página durable sin alterar la sesión.
func (c *Cliente) LeerEventosSesion(
	ctx context.Context,
	referencia string,
	sesionRef string,
	consulta ConsultaEventosSesionTrabajoV1,
) (PaginaEventosSesionTrabajoV1, error) {
	if err := validarConsultaEventosSesion(referencia, sesionRef, consulta); err != nil {
		return PaginaEventosSesionTrabajoV1{}, err
	}
	parametros := url.Values{}
	parametros.Set("cerca", strconv.FormatUint(consulta.Cerca, 10))
	parametros.Set("despues_de", strconv.FormatUint(consulta.DespuesDe, 10))
	parametros.Set("maximo_eventos", strconv.FormatUint(uint64(consulta.MaximoEventos), 10))
	respuesta, err := solicitarEstricto[PaginaEventosSesionTrabajoV1](
		ctx,
		c,
		http.MethodGet,
		rutaSesion(referencia, sesionRef, "/eventos")+"?"+parametros.Encode(),
		"",
		nil,
	)
	if err != nil {
		return PaginaEventosSesionTrabajoV1{}, err
	}
	if err := validarPaginaEventosSesion(referencia, sesionRef, consulta, respuesta); err != nil {
		return PaginaEventosSesionTrabajoV1{}, err
	}
	return respuesta, nil
}

func (c *Cliente) Ordenar(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudOrden,
) (RespuestaOrden, error) {
	return mutar[RespuestaOrden](
		ctx,
		c,
		rutaEjecucion(referencia, "/ordenes"),
		clave,
		solicitud,
	)
}

func (c *Cliente) SincronizarEntrada(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudSincronizacionEntrada,
) (RespuestaSincronizacion, error) {
	bytes, err := validarSolicitudEntrada(solicitud)
	if err != nil {
		return RespuestaSincronizacion{}, err
	}
	respuesta, err := mutar[RespuestaSincronizacion](
		ctx,
		c,
		rutaEjecucion(referencia, "/sincronizaciones/entrada"),
		clave,
		solicitud,
	)
	if err != nil {
		return RespuestaSincronizacion{}, err
	}
	if err := validarRespuestaEntrada(solicitud, bytes, respuesta); err != nil {
		return RespuestaSincronizacion{}, err
	}
	return respuesta, nil
}

func (c *Cliente) SincronizarSalida(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudSincronizacionSalida,
) (RespuestaSincronizacionSalida, error) {
	if err := validarSolicitudSalida(solicitud); err != nil {
		return RespuestaSincronizacionSalida{}, err
	}
	respuesta, err := mutar[RespuestaSincronizacionSalida](
		ctx,
		c,
		rutaEjecucion(referencia, "/sincronizaciones/salida"),
		clave,
		solicitud,
	)
	if err != nil {
		return RespuestaSincronizacionSalida{}, err
	}
	if err := validarRespuestaSalida(solicitud, respuesta); err != nil {
		return RespuestaSincronizacionSalida{}, err
	}
	return respuesta, nil
}

func (c *Cliente) Detener(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudDetencion,
) (RespuestaEjecucion, error) {
	return mutar[RespuestaEjecucion](
		ctx,
		c,
		rutaEjecucion(referencia, "/detencion"),
		clave,
		solicitud,
	)
}

func (c *Cliente) Preservar(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudPreservacion,
) (RespuestaPreservacion, error) {
	return mutar[RespuestaPreservacion](
		ctx,
		c,
		rutaEjecucion(referencia, "/preservacion"),
		clave,
		solicitud,
	)
}

func (c *Cliente) Cerrar(
	ctx context.Context,
	clave string,
	referencia string,
	solicitud SolicitudCierre,
) (RespuestaEjecucion, error) {
	return mutar[RespuestaEjecucion](
		ctx,
		c,
		rutaEjecucion(referencia, "/cierre"),
		clave,
		solicitud,
	)
}

func rutaEjecucion(referencia string, sufijo string) string {
	return "/v1/ejecuciones/" + url.PathEscape(referencia) + sufijo
}

func rutaSesion(referencia string, sesionRef string, sufijo string) string {
	return rutaEjecucion(referencia, "/sesiones/") + url.PathEscape(sesionRef) + sufijo
}

func solicitarSinCuerpo[T any](
	ctx context.Context,
	cliente *Cliente,
	metodo string,
	ruta string,
) (T, error) {
	return solicitar[T](ctx, cliente, metodo, ruta, "", nil)
}

func mutar[T any, S any](
	ctx context.Context,
	cliente *Cliente,
	ruta string,
	clave string,
	solicitud S,
) (T, error) {
	var cero T
	if clave == "" || strings.ContainsAny(clave, "\r\n") {
		return cero, &ErrorConfiguracion{Causa: "clave_idempotencia_invalida"}
	}
	cuerpo, err := json.Marshal(solicitud)
	if err != nil {
		return cero, fmt.Errorf("serializar solicitud: %w", err)
	}
	return solicitar[T](ctx, cliente, http.MethodPost, ruta, clave, cuerpo)
}

func mutarEstricto[T any, S any](
	ctx context.Context,
	cliente *Cliente,
	ruta string,
	clave string,
	solicitud S,
) (T, error) {
	var cero T
	if clave == "" || strings.ContainsAny(clave, "\r\n") {
		return cero, &ErrorConfiguracion{Causa: "clave_idempotencia_invalida"}
	}
	cuerpo, err := json.Marshal(solicitud)
	if err != nil {
		return cero, fmt.Errorf("serializar solicitud: %w", err)
	}
	return solicitarEstricto[T](ctx, cliente, http.MethodPost, ruta, clave, cuerpo)
}

func solicitar[T any](
	ctx context.Context,
	cliente *Cliente,
	metodo string,
	ruta string,
	clave string,
	cuerpo []byte,
) (T, error) {
	return solicitarDecodificando[T](ctx, cliente, metodo, ruta, clave, cuerpo, false)
}

func solicitarEstricto[T any](
	ctx context.Context,
	cliente *Cliente,
	metodo string,
	ruta string,
	clave string,
	cuerpo []byte,
) (T, error) {
	return solicitarDecodificando[T](ctx, cliente, metodo, ruta, clave, cuerpo, true)
}

func solicitarDecodificando[T any](
	ctx context.Context,
	cliente *Cliente,
	metodo string,
	ruta string,
	clave string,
	cuerpo []byte,
	estricto bool,
) (T, error) {
	var cero T
	if cliente == nil || cliente.http == nil {
		return cero, &ErrorConfiguracion{Causa: "cliente_no_inicializado"}
	}
	peticion, err := http.NewRequestWithContext(
		ctx,
		metodo,
		"http://agente-microvm.local"+ruta,
		bytes.NewReader(cuerpo),
	)
	if err != nil {
		return cero, fmt.Errorf("crear solicitud: %w", err)
	}
	peticion.Header.Set(CabeceraProtocolo, ProtocoloLocal)
	if clave != "" {
		peticion.Header.Set(CabeceraIdempotencia, clave)
		peticion.Header.Set("content-type", "application/json")
	}

	respuesta, err := cliente.http.Do(peticion)
	if err != nil {
		return cero, fmt.Errorf("solicitar agente microvm: %w", err)
	}
	defer respuesta.Body.Close()
	if respuesta.Header.Get(CabeceraProtocolo) != ProtocoloLocal {
		return cero, &ErrorProtocolo{Recibido: respuesta.Header.Get(CabeceraProtocolo)}
	}
	bytesRespuesta, err := io.ReadAll(io.LimitReader(respuesta.Body, maximoRespuestaBytes))
	if err != nil {
		return cero, fmt.Errorf("leer respuesta: %w", err)
	}
	if int64(len(bytesRespuesta)) >= maximoRespuestaBytes {
		return cero, &ErrorRespuestaGrande{}
	}
	if respuesta.StatusCode < 200 || respuesta.StatusCode >= 300 {
		problema := Problema{Codigo: "api.respuesta_rechazada"}
		_ = json.Unmarshal(bytesRespuesta, &problema)
		return cero, &ErrorRespuesta{
			Estado:  respuesta.StatusCode,
			Codigo:  problema.Codigo,
			Detalle: problema.Detalle,
		}
	}
	if len(bytesRespuesta) == 0 {
		return cero, errors.New("agente microvm devolvio una respuesta vacia")
	}
	if !estricto {
		if err := json.Unmarshal(bytesRespuesta, &cero); err != nil {
			return cero, fmt.Errorf("decodificar respuesta: %w", err)
		}
		return cero, nil
	}
	if err := validarEstructuraJSONSesion[T](bytesRespuesta); err != nil {
		return cero, fmt.Errorf("decodificar respuesta: %w", err)
	}
	decodificador := json.NewDecoder(bytes.NewReader(bytesRespuesta))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&cero); err != nil {
		return cero, fmt.Errorf("decodificar respuesta: %w", err)
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return cero, errors.New("decodificar respuesta: contenido JSON adicional")
	}
	return cero, nil
}

func validarInicioSesion(referencia string, solicitud SolicitudIniciarSesionTrabajoV1) error {
	if !referenciaSesionValida(referencia) || !referenciaSesionValida(solicitud.SesionRef) ||
		!ejecutorSesionValido(solicitud.EjecutorRef) {
		return errorSesion("sesion_trabajo.referencia_invalida")
	}
	if !revisionSesionValida(solicitud.RevisionEsperada) ||
		!revisionSesionValida(solicitud.RevisionTrabajoEsperada) {
		return errorSesion("sesion_trabajo.revision_invalida")
	}
	if !cercaSesionValida(solicitud.Cerca) {
		return errorSesion("sesion_trabajo.cerca_invalida")
	}
	if !utf8.ValidString(solicitud.DirectorioTrabajo) || !rutaBaseValida(solicitud.DirectorioTrabajo) ||
		solicitud.PlazoTotalMilisegundos == 0 ||
		solicitud.PlazoTotalMilisegundos > MaximoPlazoSesionMilisegundosV1 ||
		solicitud.MaximoEventosBytes == 0 || solicitud.MaximoEventosBytes > MaximoEventosSesionBytesV1 {
		return errorSesion("sesion_trabajo.limite_invalido")
	}
	return nil
}

func validarEntradaSesion(
	referencia string,
	sesionRef string,
	solicitud SolicitudEntradaSesionTrabajoV1,
) error {
	if !referenciaSesionValida(referencia) || !referenciaSesionValida(sesionRef) {
		return errorSesion("sesion_trabajo.referencia_invalida")
	}
	if !revisionSesionValida(solicitud.RevisionEsperada) ||
		!revisionSesionValida(solicitud.RevisionTrabajoEsperada) ||
		!revisionSesionValida(solicitud.RevisionSesionEsperada) {
		return errorSesion("sesion_trabajo.revision_invalida")
	}
	if !cercaSesionValida(solicitud.Cerca) {
		return errorSesion("sesion_trabajo.cerca_invalida")
	}
	maximoCodificado := (MaximoEntradaSesionBytesV1 + 2) / 3 * 4
	if len(solicitud.ContenidoBase64) > maximoCodificado {
		return errorSesion("sesion_trabajo.base64_invalido")
	}
	contenido, err := base64.StdEncoding.Strict().DecodeString(solicitud.ContenidoBase64)
	if err != nil || len(contenido) > MaximoEntradaSesionBytesV1 {
		return errorSesion("sesion_trabajo.base64_invalido")
	}
	if len(contenido) == 0 && !solicitud.CerrarStdin {
		return errorSesion("sesion_trabajo.limite_invalido")
	}
	return nil
}

func validarConsultaEventosSesion(
	referencia string,
	sesionRef string,
	consulta ConsultaEventosSesionTrabajoV1,
) error {
	if !referenciaSesionValida(referencia) || !referenciaSesionValida(sesionRef) {
		return errorSesion("sesion_trabajo.referencia_invalida")
	}
	if !cercaSesionValida(consulta.Cerca) {
		return errorSesion("sesion_trabajo.cerca_invalida")
	}
	if consulta.DespuesDe > maximoEnteroDurableV1 {
		return errorSesion("sesion_trabajo.cursor_invalido")
	}
	if consulta.MaximoEventos == 0 || consulta.MaximoEventos > MaximoEventosPaginaSesionV1 {
		return errorSesion("sesion_trabajo.limite_invalido")
	}
	return nil
}

func validarRespuestaSesion(
	referencia string,
	sesionRef string,
	cerca uint64,
	respuesta RespuestaSesionTrabajoV1,
) error {
	if respuesta.EjecucionRef != referencia || respuesta.SesionRef != sesionRef ||
		!referenciaSesionValida(respuesta.EjecucionRef) || !referenciaSesionValida(respuesta.SesionRef) {
		return errorSesion("sesion_trabajo.referencia_invalida")
	}
	if !revisionSesionValida(respuesta.Revision) || !revisionSesionValida(respuesta.RevisionTrabajo) ||
		!revisionSesionValida(respuesta.RevisionSesion) {
		return errorSesion("sesion_trabajo.revision_invalida")
	}
	if respuesta.Cerca != cerca {
		return errorSesion("sesion_trabajo.cerca_invalida")
	}
	if !estadoSesionValido(respuesta.Estado) || respuesta.Terminal != estadoSesionTerminal(respuesta.Estado) ||
		respuesta.Terminal != (respuesta.Resultado != nil) {
		return errorSesion("sesion_trabajo.estado_terminal_incoherente")
	}
	return validarResultadoSesion(respuesta.Estado, respuesta.Resultado)
}

func validarPaginaEventosSesion(
	referencia string,
	sesionRef string,
	consulta ConsultaEventosSesionTrabajoV1,
	pagina PaginaEventosSesionTrabajoV1,
) error {
	if pagina.EjecucionRef != referencia || pagina.SesionRef != sesionRef ||
		!referenciaSesionValida(pagina.EjecucionRef) || !referenciaSesionValida(pagina.SesionRef) {
		return errorSesion("sesion_trabajo.referencia_invalida")
	}
	if !revisionSesionValida(pagina.Revision) || !revisionSesionValida(pagina.RevisionTrabajo) ||
		!revisionSesionValida(pagina.RevisionSesion) {
		return errorSesion("sesion_trabajo.revision_invalida")
	}
	if pagina.Cerca != consulta.Cerca {
		return errorSesion("sesion_trabajo.cerca_invalida")
	}
	if pagina.DespuesDe != consulta.DespuesDe || !estadoSesionValido(pagina.Estado) {
		return errorSesion("sesion_trabajo.cursor_invalido")
	}
	if pagina.Eventos == nil || len(pagina.Eventos) > int(consulta.MaximoEventos) ||
		len(pagina.Eventos) > MaximoEventosPaginaSesionV1 || pagina.SiguienteCursor < pagina.DespuesDe ||
		pagina.SiguienteCursor > maximoEnteroDurableV1 || (pagina.Terminal && !estadoSesionTerminal(pagina.Estado)) {
		return errorSesion("sesion_trabajo.cursor_invalido")
	}
	anterior := pagina.DespuesDe
	var revisionAnterior, trabajoAnterior, sesionAnterior uint64
	for indice := range pagina.Eventos {
		evento := pagina.Eventos[indice]
		if err := validarEventoSesion(evento); err != nil {
			return err
		}
		if evento.SesionRef != pagina.SesionRef || evento.Secuencia != anterior+1 ||
			evento.Revision > pagina.Revision || evento.RevisionTrabajo > pagina.RevisionTrabajo ||
			evento.RevisionSesion > pagina.RevisionSesion || evento.Cerca != pagina.Cerca ||
			(indice > 0 && (evento.Revision < revisionAnterior || evento.RevisionTrabajo < trabajoAnterior ||
				evento.RevisionSesion <= sesionAnterior)) {
			return errorSesion("sesion_trabajo.secuencia_invalida")
		}
		if evento.Terminal && indice+1 != len(pagina.Eventos) {
			return errorSesion("sesion_trabajo.cursor_invalido")
		}
		anterior = evento.Secuencia
		revisionAnterior, trabajoAnterior, sesionAnterior = evento.Revision, evento.RevisionTrabajo, evento.RevisionSesion
	}
	if pagina.SiguienteCursor != anterior ||
		(len(pagina.Eventos) > 0 && pagina.Eventos[len(pagina.Eventos)-1].Terminal != pagina.Terminal) {
		return errorSesion("sesion_trabajo.cursor_invalido")
	}
	return nil
}

func validarEventoSesion(evento EventoSesionTrabajoV1) error {
	if !referenciaSesionValida(evento.SesionRef) || evento.Secuencia == 0 ||
		evento.Secuencia > maximoEnteroDurableV1 || !revisionSesionValida(evento.Revision) ||
		!revisionSesionValida(evento.RevisionTrabajo) || !revisionSesionValida(evento.RevisionSesion) ||
		!cercaSesionValida(evento.Cerca) || !estadoSesionValido(evento.Estado) || !tipoEventoSesionValido(evento.Tipo) {
		return errorSesion("sesion_trabajo.evento_incoherente")
	}
	maximoCodificado := (8_192 + 2) / 3 * 4
	if len(evento.ContenidoBase64) > maximoCodificado {
		return errorSesion("sesion_trabajo.base64_invalido")
	}
	contenido, err := base64.StdEncoding.Strict().DecodeString(evento.ContenidoBase64)
	if err != nil || len(contenido) > 8_192 {
		return errorSesion("sesion_trabajo.base64_invalido")
	}
	terminal := evento.Tipo == TipoEventoSesionFinalizada
	llevaContenido := evento.Tipo == TipoEventoSesionStdout || evento.Tipo == TipoEventoSesionStderr
	if evento.Terminal != terminal || evento.Terminal != estadoSesionTerminal(evento.Estado) ||
		evento.Terminal != (evento.Resultado != nil) || llevaContenido == (len(contenido) == 0) ||
		(evento.Tipo == TipoEventoSesionIniciada && evento.Estado != EstadoSesionActiva) ||
		((evento.Tipo == TipoEventoSesionStdout || evento.Tipo == TipoEventoSesionStderr ||
			evento.Tipo == TipoEventoSesionSalidaTruncada) &&
			evento.Estado != EstadoSesionActiva && evento.Estado != EstadoSesionAmbigua) {
		return errorSesion("sesion_trabajo.evento_incoherente")
	}
	return validarResultadoSesion(evento.Estado, evento.Resultado)
}

func validarResultadoSesion(estado EstadoSesionTrabajoV1, resultado *ResultadoTerminalSesionTrabajoV1) error {
	if resultado == nil {
		return nil
	}
	codigoValido := resultado.CodigoError == nil || codigoErrorSesionValido(*resultado.CodigoError)
	if !estadoSesionTerminal(estado) || resultado.CodigoSalida != nil && resultado.Senal != nil || !codigoValido ||
		estado == EstadoSesionFinalizada && resultado.CodigoError != nil ||
		estado == EstadoSesionFallida && resultado.CodigoError == nil {
		return errorSesion("sesion_trabajo.estado_terminal_incoherente")
	}
	return nil
}

func referenciaSesionValida(referencia string) bool {
	if referencia == "" || len(referencia) > 128 || !utf8.ValidString(referencia) {
		return false
	}
	for _, caracter := range []byte(referencia) {
		if caracter < 0x21 || caracter > 0x7e || caracter == '/' || caracter == '\\' {
			return false
		}
	}
	return true
}

func ejecutorSesionValido(referencia string) bool {
	if !strings.HasPrefix(referencia, prefijoEjecutorPerfilV1) {
		return false
	}
	return sha256Valido(strings.TrimPrefix(referencia, prefijoEjecutorPerfilV1))
}

func revisionSesionValida(revision uint64) bool {
	return revision > 0 && revision <= maximoEnteroDurableV1
}

func cercaSesionValida(cerca uint64) bool { return revisionSesionValida(cerca) }

func estadoSesionValido(estado EstadoSesionTrabajoV1) bool {
	switch estado {
	case EstadoSesionAdmitida, EstadoSesionIniciando, EstadoSesionActiva,
		EstadoSesionAmbigua, EstadoSesionFinalizada, EstadoSesionFallida:
		return true
	default:
		return false
	}
}

func estadoSesionTerminal(estado EstadoSesionTrabajoV1) bool {
	return estado == EstadoSesionFinalizada || estado == EstadoSesionFallida
}

func tipoEventoSesionValido(tipo TipoEventoSesionTrabajoV1) bool {
	switch tipo {
	case TipoEventoSesionIniciada, TipoEventoSesionStdout, TipoEventoSesionStderr,
		TipoEventoSesionSalidaTruncada, TipoEventoSesionFinalizada:
		return true
	default:
		return false
	}
}

func codigoErrorSesionValido(codigo string) bool {
	if codigo == "" || len(codigo) > 96 {
		return false
	}
	for _, caracter := range []byte(codigo) {
		if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') &&
			caracter != '.' && caracter != '_' {
			return false
		}
	}
	return true
}

func errorSesion(codigo string) error { return &ErrorSesionTrabajoV1{Codigo: codigo} }

func validarEstructuraJSONSesion[T any](datos []byte) error {
	var cero T
	switch any(cero).(type) {
	case RespuestaSesionTrabajoV1:
		objeto, err := objetoJSONConCampos(datos, []string{
			"ejecucion_ref", "sesion_ref", "estado", "revision", "revision_trabajo",
			"revision_sesion", "cerca", "terminal", "resultado",
		})
		if err != nil {
			return err
		}
		return validarResultadoJSON(objeto["resultado"])
	case PaginaEventosSesionTrabajoV1:
		objeto, err := objetoJSONConCampos(datos, []string{
			"ejecucion_ref", "sesion_ref", "estado", "revision", "revision_trabajo",
			"revision_sesion", "cerca", "despues_de", "eventos", "siguiente_cursor", "terminal",
		})
		if err != nil {
			return err
		}
		var eventos []json.RawMessage
		if err := json.Unmarshal(objeto["eventos"], &eventos); err != nil || eventos == nil {
			return errors.New("campo eventos invalido")
		}
		for _, evento := range eventos {
			campos, err := objetoJSONConCampos(evento, []string{
				"sesion_ref", "secuencia", "tipo", "estado", "revision", "revision_trabajo",
				"revision_sesion", "cerca", "contenido_base64", "terminal", "resultado",
			})
			if err != nil {
				return err
			}
			if err := validarResultadoJSON(campos["resultado"]); err != nil {
				return err
			}
		}
	}
	return nil
}

func objetoJSONConCampos(datos []byte, esperados []string) (map[string]json.RawMessage, error) {
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(datos, &objeto); err != nil || objeto == nil {
		return nil, errors.New("objeto JSON de sesion invalido")
	}
	for _, campo := range esperados {
		if _, existe := objeto[campo]; !existe {
			return nil, fmt.Errorf("falta campo JSON de sesion %q", campo)
		}
	}
	return objeto, nil
}

func validarResultadoJSON(datos json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(datos), []byte("null")) {
		return nil
	}
	_, err := objetoJSONConCampos(datos, []string{
		"codigo_salida", "senal", "agotada", "salida_truncada", "codigo_error",
	})
	return err
}
