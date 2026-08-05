package microvm

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	// ProtocoloAuthJSONCodexOneShotV1 identifica el canal aislado para
	// proyectar una sola credencial Codex en memoria del huésped.
	ProtocoloAuthJSONCodexOneShotV1 = "orquesta.codex-auth-json.one-shot.v1"

	BytesLongitudCabeceraAuthJSONCodexV1 = 4
	MaximoCabeceraAuthJSONCodexV1        = 4 * 1024
	MaximoMaterialAuthJSONCodexV1        = 1024 * 1024
	MaximoSesionRefAuthJSONCodexV1       = 512
)

type OperacionAuthJSONCodexV1 string

const OperacionAuthJSONCodexAdquirir OperacionAuthJSONCodexV1 = "adquirir"

// SolicitudAuthJSONCodexV1 no permite que el huésped seleccione credencial,
// propietario, alcance ni versión. Esa autoridad pertenece al host.
type SolicitudAuthJSONCodexV1 struct {
	Protocolo string                   `json:"protocolo"`
	Operacion OperacionAuthJSONCodexV1 `json:"operacion"`
	SesionRef string                   `json:"sesion_ref"`
	Cerca     uint64                   `json:"cerca"`
}

type EstadoRespuestaAuthJSONCodexV1 string

const (
	EstadoRespuestaAuthJSONCodexDisponible EstadoRespuestaAuthJSONCodexV1 = "disponible"
	EstadoRespuestaAuthJSONCodexRechazada  EstadoRespuestaAuthJSONCodexV1 = "rechazada"
)

// CodigoErrorAuthJSONCodexV1 es una lista cerrada. Sus valores nunca incluyen
// referencias, material secreto ni texto recibido del peer.
type CodigoErrorAuthJSONCodexV1 string

const (
	CodigoErrorAuthJSONCodexSolicitudInvalida   CodigoErrorAuthJSONCodexV1 = "credencial_codex.solicitud_invalida"
	CodigoErrorAuthJSONCodexAutoridadInvalida   CodigoErrorAuthJSONCodexV1 = "credencial_codex.autoridad_invalida"
	CodigoErrorAuthJSONCodexNoDisponible        CodigoErrorAuthJSONCodexV1 = "credencial_codex.no_disponible"
	CodigoErrorAuthJSONCodexRepeticionRechazada CodigoErrorAuthJSONCodexV1 = "credencial_codex.repeticion_rechazada"
	CodigoErrorAuthJSONCodexMaterialInvalido    CodigoErrorAuthJSONCodexV1 = "credencial_codex.material_invalido"
	CodigoErrorAuthJSONCodexProyeccionFallida   CodigoErrorAuthJSONCodexV1 = "credencial_codex.proyeccion_fallida"
	CodigoErrorAuthJSONCodexPurgaFallida        CodigoErrorAuthJSONCodexV1 = "credencial_codex.purga_fallida"
	CodigoErrorAuthJSONCodexCanalInterrumpido   CodigoErrorAuthJSONCodexV1 = "credencial_codex.canal_interrumpido"
)

// CabeceraRespuestaAuthJSONCodexV1 describe los bytes crudos que siguen al
// frame JSON solo cuando Estado es disponible. El material no entra en JSON.
type CabeceraRespuestaAuthJSONCodexV1 struct {
	Protocolo        string                         `json:"protocolo"`
	Estado           EstadoRespuestaAuthJSONCodexV1 `json:"estado"`
	LongitudMaterial uint32                         `json:"longitud_material"`
	CodigoError      *CodigoErrorAuthJSONCodexV1    `json:"codigo_error"`
}

type EtapaAcuseAuthJSONCodexV1 string

const (
	EtapaAcuseAuthJSONCodexProyectada EtapaAcuseAuthJSONCodexV1 = "proyectada"
	EtapaAcuseAuthJSONCodexPurgada    EtapaAcuseAuthJSONCodexV1 = "purgada"
)

// AcuseAuthJSONCodexV1 mantiene el mismo canal hasta acreditar la purga.
type AcuseAuthJSONCodexV1 struct {
	Protocolo   string                      `json:"protocolo"`
	SesionRef   string                      `json:"sesion_ref"`
	Cerca       uint64                      `json:"cerca"`
	Etapa       EtapaAcuseAuthJSONCodexV1   `json:"etapa"`
	Correcto    bool                        `json:"correcto"`
	CodigoError *CodigoErrorAuthJSONCodexV1 `json:"codigo_error"`
}

// ErrorContratoAuthJSONCodexV1 conserva únicamente un código estable.
type ErrorContratoAuthJSONCodexV1 struct{ Codigo string }

func (e *ErrorContratoAuthJSONCodexV1) Error() string {
	if e == nil {
		return ""
	}
	return e.Codigo
}

var (
	esquemaSolicitudAuthJSONCodexV1 = esquemaObjetoJSONEstricto{
		"protocolo": esquemaEscalarJSONEstricto, "operacion": esquemaEscalarJSONEstricto,
		"sesion_ref": esquemaEscalarJSONEstricto, "cerca": esquemaEscalarJSONEstricto,
	}
	esquemaCabeceraRespuestaAuthJSONCodexV1 = esquemaObjetoJSONEstricto{
		"protocolo": esquemaEscalarJSONEstricto, "estado": esquemaEscalarJSONEstricto,
		"longitud_material": esquemaEscalarJSONEstricto,
		"codigo_error":      esquemaEscalarJSONEstrictoOpcional,
	}
	esquemaAcuseAuthJSONCodexV1 = esquemaObjetoJSONEstricto{
		"protocolo": esquemaEscalarJSONEstricto, "sesion_ref": esquemaEscalarJSONEstricto,
		"cerca": esquemaEscalarJSONEstricto, "etapa": esquemaEscalarJSONEstricto,
		"correcto": esquemaEscalarJSONEstricto, "codigo_error": esquemaEscalarJSONEstrictoOpcional,
	}
)

func ValidarSolicitudAuthJSONCodexV1(solicitud SolicitudAuthJSONCodexV1) error {
	if solicitud.Protocolo != ProtocoloAuthJSONCodexOneShotV1 ||
		solicitud.Operacion != OperacionAuthJSONCodexAdquirir ||
		!sesionRefAuthJSONCodexValida(solicitud.SesionRef) || solicitud.Cerca == 0 {
		return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexSolicitudInvalida)
	}
	return nil
}

func ValidarCabeceraRespuestaAuthJSONCodexV1(cabecera CabeceraRespuestaAuthJSONCodexV1) error {
	if cabecera.Protocolo != ProtocoloAuthJSONCodexOneShotV1 {
		return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexMaterialInvalido)
	}
	switch cabecera.Estado {
	case EstadoRespuestaAuthJSONCodexDisponible:
		if cabecera.CodigoError != nil || cabecera.LongitudMaterial == 0 ||
			cabecera.LongitudMaterial > MaximoMaterialAuthJSONCodexV1 {
			return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexMaterialInvalido)
		}
	case EstadoRespuestaAuthJSONCodexRechazada:
		if cabecera.LongitudMaterial != 0 || cabecera.CodigoError == nil ||
			!codigoRespuestaHostAuthJSONCodexValido(*cabecera.CodigoError) {
			return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexMaterialInvalido)
		}
	default:
		return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexMaterialInvalido)
	}
	return nil
}

func ValidarAcuseAuthJSONCodexV1(acuse AcuseAuthJSONCodexV1) error {
	valido := acuse.Protocolo == ProtocoloAuthJSONCodexOneShotV1 &&
		sesionRefAuthJSONCodexValida(acuse.SesionRef) && acuse.Cerca > 0 &&
		(acuse.Correcto == (acuse.CodigoError == nil))
	if !valido {
		return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexCanalInterrumpido)
	}
	if acuse.Correcto {
		if acuse.Etapa == EtapaAcuseAuthJSONCodexProyectada ||
			acuse.Etapa == EtapaAcuseAuthJSONCodexPurgada {
			return nil
		}
	} else if acuse.CodigoError != nil {
		switch acuse.Etapa {
		case EtapaAcuseAuthJSONCodexProyectada:
			if codigoAcuseProyeccionAuthJSONCodexValido(*acuse.CodigoError) {
				return nil
			}
		case EtapaAcuseAuthJSONCodexPurgada:
			if codigoAcusePurgaAuthJSONCodexValido(*acuse.CodigoError) {
				return nil
			}
		}
	}
	return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexCanalInterrumpido)
}

func CodificarSolicitudAuthJSONCodexV1(solicitud SolicitudAuthJSONCodexV1) ([]byte, error) {
	if err := ValidarSolicitudAuthJSONCodexV1(solicitud); err != nil {
		return nil, err
	}
	return codificarJSONEnmarcadoAuthJSONCodex(solicitud, CodigoErrorAuthJSONCodexSolicitudInvalida)
}

func DecodificarSolicitudAuthJSONCodexV1(lector io.Reader) (SolicitudAuthJSONCodexV1, error) {
	contenido, err := leerJSONEnmarcadoAuthJSONCodex(lector, CodigoErrorAuthJSONCodexSolicitudInvalida)
	if err != nil {
		return SolicitudAuthJSONCodexV1{}, err
	}
	var solicitud SolicitudAuthJSONCodexV1
	if !decodificarObjetoJSONEstricto(contenido, esquemaSolicitudAuthJSONCodexV1, &solicitud) {
		return SolicitudAuthJSONCodexV1{}, errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexSolicitudInvalida)
	}
	if err := ValidarSolicitudAuthJSONCodexV1(solicitud); err != nil {
		return SolicitudAuthJSONCodexV1{}, err
	}
	return solicitud, nil
}

func CodificarCabeceraRespuestaAuthJSONCodexV1(cabecera CabeceraRespuestaAuthJSONCodexV1) ([]byte, error) {
	if err := ValidarCabeceraRespuestaAuthJSONCodexV1(cabecera); err != nil {
		return nil, err
	}
	return codificarJSONEnmarcadoAuthJSONCodex(cabecera, CodigoErrorAuthJSONCodexMaterialInvalido)
}

func DecodificarCabeceraRespuestaAuthJSONCodexV1(lector io.Reader) (CabeceraRespuestaAuthJSONCodexV1, error) {
	contenido, err := leerJSONEnmarcadoAuthJSONCodex(lector, CodigoErrorAuthJSONCodexMaterialInvalido)
	if err != nil {
		return CabeceraRespuestaAuthJSONCodexV1{}, err
	}
	var cabecera CabeceraRespuestaAuthJSONCodexV1
	if !decodificarObjetoJSONEstricto(contenido, esquemaCabeceraRespuestaAuthJSONCodexV1, &cabecera) {
		return CabeceraRespuestaAuthJSONCodexV1{}, errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexMaterialInvalido)
	}
	if err := ValidarCabeceraRespuestaAuthJSONCodexV1(cabecera); err != nil {
		return CabeceraRespuestaAuthJSONCodexV1{}, err
	}
	return cabecera, nil
}

func CodificarAcuseAuthJSONCodexV1(acuse AcuseAuthJSONCodexV1) ([]byte, error) {
	if err := ValidarAcuseAuthJSONCodexV1(acuse); err != nil {
		return nil, err
	}
	return codificarJSONEnmarcadoAuthJSONCodex(acuse, CodigoErrorAuthJSONCodexCanalInterrumpido)
}

func DecodificarAcuseAuthJSONCodexV1(lector io.Reader) (AcuseAuthJSONCodexV1, error) {
	contenido, err := leerJSONEnmarcadoAuthJSONCodex(lector, CodigoErrorAuthJSONCodexCanalInterrumpido)
	if err != nil {
		return AcuseAuthJSONCodexV1{}, err
	}
	var acuse AcuseAuthJSONCodexV1
	if !decodificarObjetoJSONEstricto(contenido, esquemaAcuseAuthJSONCodexV1, &acuse) {
		return AcuseAuthJSONCodexV1{}, errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexCanalInterrumpido)
	}
	if err := ValidarAcuseAuthJSONCodexV1(acuse); err != nil {
		return AcuseAuthJSONCodexV1{}, err
	}
	return acuse, nil
}

// EscribirRespuestaAuthJSONCodexV1 escribe frame JSON y, para una respuesta
// disponible, los bytes crudos exactos. Nunca incorpora el material a errores.
func EscribirRespuestaAuthJSONCodexV1(
	escritor io.Writer,
	cabecera CabeceraRespuestaAuthJSONCodexV1,
	material []byte,
) error {
	if escritor == nil || ValidarCabeceraRespuestaAuthJSONCodexV1(cabecera) != nil ||
		len(material) != int(cabecera.LongitudMaterial) {
		return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexMaterialInvalido)
	}
	trama, err := CodificarCabeceraRespuestaAuthJSONCodexV1(cabecera)
	if err != nil {
		return err
	}
	if err := escribirCompletoAuthJSONCodex(escritor, trama); err != nil {
		return err
	}
	if len(material) > 0 {
		return escribirCompletoAuthJSONCodex(escritor, material)
	}
	return nil
}

func leerJSONEnmarcadoAuthJSONCodex(
	lector io.Reader,
	codigoInvalido CodigoErrorAuthJSONCodexV1,
) ([]byte, error) {
	if lector == nil {
		return nil, errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexCanalInterrumpido)
	}
	var prefijo [BytesLongitudCabeceraAuthJSONCodexV1]byte
	if _, err := io.ReadFull(lector, prefijo[:]); err != nil {
		return nil, errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexCanalInterrumpido)
	}
	longitud := binary.BigEndian.Uint32(prefijo[:])
	if longitud == 0 || longitud > MaximoCabeceraAuthJSONCodexV1 {
		return nil, errorContratoAuthJSONCodex(codigoInvalido)
	}
	contenido := make([]byte, int(longitud))
	if _, err := io.ReadFull(lector, contenido); err != nil {
		return nil, errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexCanalInterrumpido)
	}
	return contenido, nil
}

func codificarJSONEnmarcadoAuthJSONCodex(
	valor any,
	codigoInvalido CodigoErrorAuthJSONCodexV1,
) ([]byte, error) {
	contenido, err := codificarJSONAuthJSONCodexCompatibleSerde(valor)
	if err != nil || len(contenido) == 0 || len(contenido) > MaximoCabeceraAuthJSONCodexV1 {
		return nil, errorContratoAuthJSONCodex(codigoInvalido)
	}
	trama := make([]byte, BytesLongitudCabeceraAuthJSONCodexV1, BytesLongitudCabeceraAuthJSONCodexV1+len(contenido))
	binary.BigEndian.PutUint32(trama, uint32(len(contenido)))
	return append(trama, contenido...), nil
}

func codificarJSONAuthJSONCodexCompatibleSerde(valor any) ([]byte, error) {
	var buffer bytes.Buffer
	codificador := json.NewEncoder(&buffer)
	codificador.SetEscapeHTML(false)
	if err := codificador.Encode(valor); err != nil {
		return nil, err
	}
	contenido := bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'})
	resultado := make([]byte, 0, len(contenido))
	for indice := 0; indice < len(contenido); indice++ {
		if contenido[indice] != '\\' || indice+1 >= len(contenido) {
			resultado = append(resultado, contenido[indice])
			continue
		}
		if contenido[indice+1] == '\\' {
			resultado = append(resultado, contenido[indice], contenido[indice+1])
			indice++
			continue
		}
		if indice+5 < len(contenido) && contenido[indice+1] == 'u' {
			switch string(contenido[indice+2 : indice+6]) {
			case "2028":
				resultado = append(resultado, '\xe2', '\x80', '\xa8')
				indice += 5
				continue
			case "2029":
				resultado = append(resultado, '\xe2', '\x80', '\xa9')
				indice += 5
				continue
			}
		}
		resultado = append(resultado, contenido[indice])
	}
	return resultado, nil
}

func escribirCompletoAuthJSONCodex(escritor io.Writer, contenido []byte) error {
	for len(contenido) > 0 {
		escritos, err := escritor.Write(contenido)
		if err != nil || escritos <= 0 || escritos > len(contenido) {
			return errorContratoAuthJSONCodex(CodigoErrorAuthJSONCodexCanalInterrumpido)
		}
		contenido = contenido[escritos:]
	}
	return nil
}

func sesionRefAuthJSONCodexValida(valor string) bool {
	return utf8.ValidString(valor) && valor != "" && len(valor) <= MaximoSesionRefAuthJSONCodexV1 &&
		strings.TrimSpace(valor) == valor && strings.HasPrefix(valor, "execution-session:") &&
		len(valor) > len("execution-session:") && !strings.ContainsAny(valor, "\x00\r\n/\\")
}

func codigoRespuestaHostAuthJSONCodexValido(codigo CodigoErrorAuthJSONCodexV1) bool {
	switch codigo {
	case CodigoErrorAuthJSONCodexSolicitudInvalida, CodigoErrorAuthJSONCodexAutoridadInvalida,
		CodigoErrorAuthJSONCodexNoDisponible, CodigoErrorAuthJSONCodexRepeticionRechazada,
		CodigoErrorAuthJSONCodexMaterialInvalido, CodigoErrorAuthJSONCodexCanalInterrumpido:
		return true
	default:
		return false
	}
}

func codigoAcuseProyeccionAuthJSONCodexValido(codigo CodigoErrorAuthJSONCodexV1) bool {
	return codigo == CodigoErrorAuthJSONCodexMaterialInvalido ||
		codigo == CodigoErrorAuthJSONCodexProyeccionFallida ||
		codigo == CodigoErrorAuthJSONCodexCanalInterrumpido
}

func codigoAcusePurgaAuthJSONCodexValido(codigo CodigoErrorAuthJSONCodexV1) bool {
	return codigo == CodigoErrorAuthJSONCodexPurgaFallida ||
		codigo == CodigoErrorAuthJSONCodexCanalInterrumpido
}

func errorContratoAuthJSONCodex(codigo CodigoErrorAuthJSONCodexV1) error {
	return &ErrorContratoAuthJSONCodexV1{Codigo: string(codigo)}
}
