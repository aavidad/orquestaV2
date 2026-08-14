package microvm

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"
)

type claseValorJSONEstricto uint8

const (
	valorJSONEscalar claseValorJSONEstricto = iota + 1
	valorJSONObject
	valorJSONArray
)

type esquemaValorJSONEstricto struct {
	clase      claseValorJSONEstricto
	objeto     esquemaObjetoJSONEstricto
	elemento   *esquemaValorJSONEstricto
	admiteNulo bool
}

type esquemaObjetoJSONEstricto map[string]esquemaValorJSONEstricto

var esquemaEscalarJSONEstricto = esquemaValorJSONEstricto{clase: valorJSONEscalar}

// esquemaEscalarJSONEstrictoOpcional representa un campo obligatorio cuya
// forma de red admite explícitamente `null`, como un Option de Serde. No se
// usa para campos omitibles: la completitud del objeto sigue siendo estricta.
var esquemaEscalarJSONEstrictoOpcional = esquemaValorJSONEstricto{
	clase:      valorJSONEscalar,
	admiteNulo: true,
}

func esquemaObjetoEstricto(campos esquemaObjetoJSONEstricto) esquemaValorJSONEstricto {
	return esquemaValorJSONEstricto{clase: valorJSONObject, objeto: campos}
}

// esquemaObjetoEstrictoOpcional representa un objeto obligatorio cuya forma
// de red admite `null`. La completitud del campo y, si existe, del objeto
// anidado siguen siendo estrictas.
func esquemaObjetoEstrictoOpcional(campos esquemaObjetoJSONEstricto) esquemaValorJSONEstricto {
	return esquemaValorJSONEstricto{
		clase:      valorJSONObject,
		objeto:     campos,
		admiteNulo: true,
	}
}

func esquemaArrayEstricto(elemento esquemaValorJSONEstricto) esquemaValorJSONEstricto {
	return esquemaValorJSONEstricto{clase: valorJSONArray, elemento: &elemento}
}

// decodificarObjetoJSONEstricto es la única autoridad sintáctica JSON de los
// contratos públicos Go. Posee UTF-8, surrogates, forma exacta, completitud,
// duplicados semánticos (también claves escapadas) y un único valor raíz.
func decodificarObjetoJSONEstricto(
	raw []byte,
	esquema esquemaObjetoJSONEstricto,
	destino any,
) bool {
	if len(raw) == 0 || !utf8.Valid(raw) || !escapesUnicodeJSONValidos(raw) {
		return false
	}
	validador := json.NewDecoder(bytes.NewReader(raw))
	validador.UseNumber()
	if !validarObjetoJSONEstricto(validador, esquema) || !finTokensJSONEstricto(validador) {
		return false
	}
	decodificador := json.NewDecoder(bytes.NewReader(raw))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return false
	}
	var trailing any
	return errors.Is(decodificador.Decode(&trailing), io.EOF)
}

func validarObjetoJSONEstricto(
	decoder *json.Decoder,
	esquema esquemaObjetoJSONEstricto,
) bool {
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return false
	}
	return validarContenidoObjetoJSONEstricto(decoder, esquema)
}

func validarContenidoObjetoJSONEstricto(
	decoder *json.Decoder,
	esquema esquemaObjetoJSONEstricto,
) bool {
	vistos := make(map[string]struct{}, len(esquema))
	for decoder.More() {
		token, err := decoder.Token()
		name, ok := token.(string)
		if err != nil || !ok {
			return false
		}
		campo, permitido := esquema[name]
		if !permitido {
			return false
		}
		if _, duplicado := vistos[name]; duplicado {
			return false
		}
		vistos[name] = struct{}{}
		if !validarValorJSONEstricto(decoder, campo) {
			return false
		}
	}
	closing, err := decoder.Token()
	return err == nil && closing == json.Delim('}') && len(vistos) == len(esquema)
}

func validarValorJSONEstricto(decoder *json.Decoder, esquema esquemaValorJSONEstricto) bool {
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	if token == nil {
		return esquema.admiteNulo
	}
	switch esquema.clase {
	case valorJSONEscalar:
		_, compuesto := token.(json.Delim)
		return !compuesto
	case valorJSONObject:
		return token == json.Delim('{') && validarContenidoObjetoJSONEstricto(decoder, esquema.objeto)
	case valorJSONArray:
		if token != json.Delim('[') || esquema.elemento == nil {
			return false
		}
		for decoder.More() {
			if !validarValorJSONEstricto(decoder, *esquema.elemento) {
				return false
			}
		}
		closing, err := decoder.Token()
		return err == nil && closing == json.Delim(']')
	default:
		return false
	}
}

func finTokensJSONEstricto(decoder *json.Decoder) bool {
	_, err := decoder.Token()
	return errors.Is(err, io.EOF)
}

// encoding/json sustituye surrogates UTF-16 aislados por U+FFFD. serde_json
// los rechaza; esta pasada conserva esa misma frontera antes de decodificar.
func escapesUnicodeJSONValidos(contenido []byte) bool {
	dentroCadena := false
	for indice := 0; indice < len(contenido); indice++ {
		switch contenido[indice] {
		case '"':
			dentroCadena = !dentroCadena
		case '\\':
			if !dentroCadena || indice+1 >= len(contenido) {
				continue
			}
			if contenido[indice+1] != 'u' {
				indice++
				continue
			}
			primero, ok := valorEscapeUnicode(contenido, indice)
			if !ok {
				return false
			}
			indice += 5
			if primero >= 0xd800 && primero <= 0xdbff {
				if indice+6 >= len(contenido) || contenido[indice+1] != '\\' || contenido[indice+2] != 'u' {
					return false
				}
				segundo, segundoOK := valorEscapeUnicode(contenido, indice+1)
				if !segundoOK || segundo < 0xdc00 || segundo > 0xdfff {
					return false
				}
				indice += 6
			} else if primero >= 0xdc00 && primero <= 0xdfff {
				return false
			}
		}
	}
	return !dentroCadena
}

func valorEscapeUnicode(contenido []byte, inicio int) (uint16, bool) {
	if inicio+5 >= len(contenido) || contenido[inicio] != '\\' || contenido[inicio+1] != 'u' {
		return 0, false
	}
	var valor uint16
	for _, digito := range contenido[inicio+2 : inicio+6] {
		valor <<= 4
		switch {
		case digito >= '0' && digito <= '9':
			valor += uint16(digito - '0')
		case digito >= 'a' && digito <= 'f':
			valor += uint16(digito-'a') + 10
		case digito >= 'A' && digito <= 'F':
			valor += uint16(digito-'A') + 10
		default:
			return 0, false
		}
	}
	return valor, true
}
