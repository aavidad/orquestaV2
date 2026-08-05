package microvm

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"sort"
	"strings"
)

const (
	MaximoArchivosSincronizacion = 2_048
	MaximoBytesSincronizacion    = 8 * 1_048_576
)

// ErrorContenido identifica una entrada o respuesta que no acredita su árbol.
type ErrorContenido struct{ Causa string }

func (err *ErrorContenido) Error() string {
	if err == nil {
		return ""
	}
	return "contenido." + err.Causa
}

type descriptorContenido struct {
	ruta       string
	ejecutable bool
	contenido  []byte
	digest     [sha256.Size]byte
}

// CalcularRaizContenido verifica cada blob y devuelve la raíz canónica v1.
func CalcularRaizContenido(archivos []ArchivoContenido) (string, error) {
	descriptores, _, err := validarArchivos(archivos, MaximoArchivosSincronizacion, MaximoBytesSincronizacion)
	if err != nil {
		return "", err
	}
	return calcularRaizDescriptores(descriptores), nil
}

func calcularRaizDescriptores(descriptores []descriptorContenido) string {
	sort.Slice(descriptores, func(i, j int) bool { return descriptores[i].ruta < descriptores[j].ruta })
	digest := sha256.New()
	_, _ = digest.Write([]byte("agentmicrovm.contenido.v1\x00"))
	var entero [8]byte
	for _, descriptor := range descriptores {
		binary.BigEndian.PutUint32(entero[:4], uint32(len(descriptor.ruta)))
		_, _ = digest.Write(entero[:4])
		_, _ = digest.Write([]byte(descriptor.ruta))
		if descriptor.ejecutable {
			_, _ = digest.Write([]byte{1})
		} else {
			_, _ = digest.Write([]byte{0})
		}
		binary.BigEndian.PutUint64(entero[:], uint64(len(descriptor.contenido)))
		_, _ = digest.Write(entero[:])
		_, _ = digest.Write(descriptor.digest[:])
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func validarSolicitudEntrada(solicitud SolicitudSincronizacionEntrada) (uint64, error) {
	if solicitud.RevisionEsperada == 0 || solicitud.RevisionTrabajoEsperada == 0 || solicitud.Cerca == 0 ||
		!rutaBaseValida(solicitud.DestinoRelativo) {
		return 0, &ErrorContenido{Causa: "solicitud_entrada_invalida"}
	}
	descriptores, bytes, err := validarArchivos(
		solicitud.Archivos,
		MaximoArchivosSincronizacion,
		MaximoBytesSincronizacion,
	)
	if err != nil {
		return 0, err
	}
	raiz := calcularRaizDescriptores(descriptores)
	if !digestValido(solicitud.RaizSHA256) || raiz != solicitud.RaizSHA256 {
		return 0, &ErrorContenido{Causa: "raiz_no_acreditada"}
	}
	return bytes, nil
}

func validarSolicitudSalida(solicitud SolicitudSincronizacionSalida) error {
	if solicitud.RevisionEsperada == 0 || solicitud.RevisionTrabajoEsperada == 0 || solicitud.Cerca == 0 ||
		!rutaBaseValida(solicitud.OrigenRelativo) || solicitud.MaximoArchivos == 0 ||
		solicitud.MaximoArchivos > MaximoArchivosSincronizacion || solicitud.MaximoBytes == 0 ||
		solicitud.MaximoBytes > MaximoBytesSincronizacion {
		return &ErrorContenido{Causa: "solicitud_salida_invalida"}
	}
	return nil
}

func validarRespuestaEntrada(solicitud SolicitudSincronizacionEntrada, bytes uint64, respuesta RespuestaSincronizacion) error {
	if respuesta.Direccion != "entrada" || respuesta.RutaRelativa != solicitud.DestinoRelativo ||
		respuesta.RaizSHA256 != solicitud.RaizSHA256 || respuesta.Archivos != uint32(len(solicitud.Archivos)) ||
		respuesta.Bytes != bytes || respuesta.RevisionTrabajo <= solicitud.RevisionTrabajoEsperada {
		return &ErrorContenido{Causa: "respuesta_entrada_invalida"}
	}
	return nil
}

func validarRespuestaSalida(solicitud SolicitudSincronizacionSalida, respuesta RespuestaSincronizacionSalida) error {
	sincronizacion := respuesta.Sincronizacion
	descriptores, bytes, err := validarArchivos(respuesta.Archivos, solicitud.MaximoArchivos, solicitud.MaximoBytes)
	if err != nil {
		return err
	}
	raiz := calcularRaizDescriptores(descriptores)
	if sincronizacion.Direccion != "salida" || sincronizacion.RutaRelativa != solicitud.OrigenRelativo ||
		sincronizacion.RaizSHA256 != raiz || sincronizacion.Archivos != uint32(len(respuesta.Archivos)) ||
		sincronizacion.Bytes != bytes || sincronizacion.RevisionTrabajo != solicitud.RevisionTrabajoEsperada {
		return &ErrorContenido{Causa: "respuesta_salida_invalida"}
	}
	return nil
}

func validarArchivos(archivos []ArchivoContenido, maximoArchivos uint32, maximoBytes uint64) ([]descriptorContenido, uint64, error) {
	if uint64(len(archivos)) > uint64(maximoArchivos) {
		return nil, 0, &ErrorContenido{Causa: "demasiados_archivos"}
	}
	descriptores := make([]descriptorContenido, 0, len(archivos))
	rutas := make(map[string]struct{}, len(archivos))
	var total uint64
	for _, archivo := range archivos {
		if !rutaArchivoValida(archivo.RutaRelativa) {
			return nil, 0, &ErrorContenido{Causa: "ruta_invalida"}
		}
		if _, repetida := rutas[archivo.RutaRelativa]; repetida {
			return nil, 0, &ErrorContenido{Causa: "ruta_repetida"}
		}
		rutas[archivo.RutaRelativa] = struct{}{}
		contenido, err := base64.StdEncoding.Strict().DecodeString(archivo.ContenidoBase64)
		if err != nil {
			return nil, 0, &ErrorContenido{Causa: "base64_invalido"}
		}
		digest := sha256.Sum256(contenido)
		if !digestValido(archivo.SHA256) || hex.EncodeToString(digest[:]) != archivo.SHA256 {
			return nil, 0, &ErrorContenido{Causa: "digest_invalido"}
		}
		if uint64(len(contenido)) > maximoBytes-total {
			return nil, 0, &ErrorContenido{Causa: "contenido_excedido"}
		}
		total += uint64(len(contenido))
		descriptores = append(descriptores, descriptorContenido{
			ruta: archivo.RutaRelativa, ejecutable: archivo.Ejecutable, contenido: contenido, digest: digest,
		})
	}
	return descriptores, total, nil
}

func rutaBaseValida(ruta string) bool { return ruta == "." || rutaArchivoValida(ruta) }

func rutaArchivoValida(ruta string) bool {
	if ruta == "" || len(ruta) > 240 || strings.HasPrefix(ruta, "/") || strings.ContainsAny(ruta, "\x00\\") {
		return false
	}
	for _, segmento := range strings.Split(ruta, "/") {
		if segmento == "" || segmento == "." || segmento == ".." {
			return false
		}
	}
	return true
}

func digestValido(valor string) bool {
	if len(valor) != sha256.Size*2 || strings.ToLower(valor) != valor {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}
