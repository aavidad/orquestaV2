package microvm

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// ProtocoloAperturaServicioHostV1 identifica el encabezado que Agente
	// MicroVM antepone a cada conexión reenviada desde vsock al servicio host.
	ProtocoloAperturaServicioHostV1 = "agentmicrovm.servicio-host.v1"

	// MaximoAperturaServicioHostV1 limita solo el JSON de apertura. Los bytes
	// posteriores pertenecen al protocolo neutral del servicio conectado.
	MaximoAperturaServicioHostV1 = 4 * 1024
)

// PapelServicioHostV1 identifica el único servicio causal de una conexión.
type PapelServicioHostV1 string

const (
	PapelServicioHostControlBroker         PapelServicioHostV1 = "control_broker"
	PapelServicioHostControlledEgressProxy PapelServicioHostV1 = "controlled_egress_proxy"
)

// AperturaServicioHostV1 conserva la atribución causal y física que el
// intermediario ya verificó. No contiene rutas, endpoints, órdenes ni secretos.
// El orden de los campos también fija la codificación JSON emitida por Rust.
type AperturaServicioHostV1 struct {
	Protocolo       string              `json:"protocolo"`
	EjecucionRef    string              `json:"ejecucion_ref"`
	RunRef          string              `json:"run_ref"`
	CIDVsock        uint32              `json:"cid_vsock"`
	PlanSHA256      string              `json:"plan_sha256"`
	ConcesionSHA256 string              `json:"concesion_sha256"`
	Cerca           uint64              `json:"cerca"`
	Papel           PapelServicioHostV1 `json:"papel"`
	ServicioRef     string              `json:"servicio_ref"`
	IdentidadRef    string              `json:"identidad_ref"`
	IdentidadSHA256 string              `json:"identidad_sha256"`
	FirecrackerPID  uint32              `json:"firecracker_pid"`
	FirecrackerUID  uint32              `json:"firecracker_uid"`
}

// ErrorAperturaServicioHostV1 clasifica fallos de contrato sin exponer bytes.
type ErrorAperturaServicioHostV1 struct{ Codigo string }

func (e *ErrorAperturaServicioHostV1) Error() string {
	if e == nil {
		return ""
	}
	return e.Codigo
}

// ValidarAperturaServicioHostV1 valida únicamente el contrato portable. La
// identidad del peer y el lifecycle pertenecen al adaptador físico posterior.
func ValidarAperturaServicioHostV1(apertura AperturaServicioHostV1) error {
	if !cadenasAperturaServicioHostUTF8Validas(apertura) {
		return errorAperturaServicioHost("json_invalido")
	}
	if apertura.Protocolo != ProtocoloAperturaServicioHostV1 {
		return errorAperturaServicioHost("protocolo_incompatible")
	}
	if !referenciaValida(apertura.EjecucionRef, "ejecucion:", 96) ||
		!referenciaExternaServicioHostValida(apertura.RunRef) ||
		!referenciaValida(apertura.ServicioRef, "servicio:", 160) ||
		!referenciaValida(apertura.IdentidadRef, "identidad-servicio:", 160) {
		return errorAperturaServicioHost("referencia_invalida")
	}
	if !sha256Valido(apertura.PlanSHA256) || !sha256Valido(apertura.ConcesionSHA256) ||
		!sha256Valido(apertura.IdentidadSHA256) {
		return errorAperturaServicioHost("digest_invalido")
	}
	if apertura.Cerca == 0 {
		return errorAperturaServicioHost("cerca_invalida")
	}
	if apertura.CIDVsock < 3 {
		return errorAperturaServicioHost("cid_vsock_invalido")
	}
	if apertura.Papel != PapelServicioHostControlBroker &&
		apertura.Papel != PapelServicioHostControlledEgressProxy {
		return errorAperturaServicioHost("papel_invalido")
	}
	if apertura.FirecrackerPID == 0 || apertura.FirecrackerUID == 0 {
		return errorAperturaServicioHost("identidad_fisica_invalida")
	}
	return nil
}

// CodificarAperturaServicioHostV1 devuelve u32 big-endian + JSON, exactamente
// la trama que escribe el intermediario Rust antes de los bytes del huésped.
func CodificarAperturaServicioHostV1(apertura AperturaServicioHostV1) ([]byte, error) {
	if err := ValidarAperturaServicioHostV1(apertura); err != nil {
		return nil, err
	}
	contenido, err := codificarJSONAperturaServicioHost(apertura)
	if err != nil {
		return nil, errorAperturaServicioHost("json_invalido")
	}
	if len(contenido) == 0 || len(contenido) > MaximoAperturaServicioHostV1 {
		return nil, errorAperturaServicioHost("trama_demasiado_grande")
	}
	trama := make([]byte, 4, 4+len(contenido))
	binary.BigEndian.PutUint32(trama, uint32(len(contenido)))
	return append(trama, contenido...), nil
}

// DecodificarAperturaServicioHostV1 consume una sola apertura y deja intactos
// en lector los bytes del huésped que el intermediario reenvía a continuación.
func DecodificarAperturaServicioHostV1(lector io.Reader) (AperturaServicioHostV1, error) {
	if lector == nil {
		return AperturaServicioHostV1{}, errorAperturaServicioHost("trama_invalida")
	}
	var prefijo [4]byte
	if _, err := io.ReadFull(lector, prefijo[:]); err != nil {
		return AperturaServicioHostV1{}, errorAperturaServicioHost("trama_incompleta")
	}
	longitud := binary.BigEndian.Uint32(prefijo[:])
	if longitud == 0 {
		return AperturaServicioHostV1{}, errorAperturaServicioHost("trama_invalida")
	}
	if longitud > MaximoAperturaServicioHostV1 {
		return AperturaServicioHostV1{}, errorAperturaServicioHost("trama_demasiado_grande")
	}
	contenido := make([]byte, int(longitud))
	if _, err := io.ReadFull(lector, contenido); err != nil {
		return AperturaServicioHostV1{}, errorAperturaServicioHost("trama_incompleta")
	}

	var apertura AperturaServicioHostV1
	if !decodificarObjetoJSONEstricto(contenido, esquemaAperturaServicioHostV1, &apertura) {
		return AperturaServicioHostV1{}, errorAperturaServicioHost("json_invalido")
	}
	if err := ValidarAperturaServicioHostV1(apertura); err != nil {
		return AperturaServicioHostV1{}, err
	}
	return apertura, nil
}

var esquemaAperturaServicioHostV1 = esquemaObjetoJSONEstricto{
	"protocolo": esquemaEscalarJSONEstricto, "ejecucion_ref": esquemaEscalarJSONEstricto,
	"run_ref": esquemaEscalarJSONEstricto, "cid_vsock": esquemaEscalarJSONEstricto,
	"plan_sha256": esquemaEscalarJSONEstricto, "concesion_sha256": esquemaEscalarJSONEstricto,
	"cerca": esquemaEscalarJSONEstricto, "papel": esquemaEscalarJSONEstricto,
	"servicio_ref": esquemaEscalarJSONEstricto, "identidad_ref": esquemaEscalarJSONEstricto,
	"identidad_sha256": esquemaEscalarJSONEstricto, "firecracker_pid": esquemaEscalarJSONEstricto,
	"firecracker_uid": esquemaEscalarJSONEstricto,
}

func cadenasAperturaServicioHostUTF8Validas(apertura AperturaServicioHostV1) bool {
	for _, valor := range []string{
		apertura.Protocolo, apertura.EjecucionRef, apertura.RunRef,
		apertura.PlanSHA256, apertura.ConcesionSHA256, string(apertura.Papel),
		apertura.ServicioRef, apertura.IdentidadRef, apertura.IdentidadSHA256,
	} {
		if !utf8.ValidString(valor) {
			return false
		}
	}
	return true
}

// encoding/json conserva U+2028/U+2029 escapados por compatibilidad con
// JavaScript. serde_json los emite como UTF-8; se desescapan únicamente cuando
// son el escape JSON activo, nunca dentro de una barra invertida ya escapada.
func codificarJSONAperturaServicioHost(apertura AperturaServicioHostV1) ([]byte, error) {
	var buffer bytes.Buffer
	codificador := json.NewEncoder(&buffer)
	codificador.SetEscapeHTML(false)
	if err := codificador.Encode(apertura); err != nil {
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

func referenciaExternaServicioHostValida(valor string) bool {
	if valor == "" || len(valor) > 512 || strings.TrimSpace(valor) != valor {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) {
			return false
		}
	}
	return true
}

func errorAperturaServicioHost(sufijo string) error {
	return &ErrorAperturaServicioHostV1{Codigo: "apertura_servicio_host." + sufijo}
}
