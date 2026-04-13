package microprogramacionapp

import (
	"path/filepath"
	"strings"
)

// ExtraerArchivosEntrega analiza bloques "// FILE: ruta/relativa.go" y devuelve
// los ficheros declarados por la respuesta del agente.
func ExtraerArchivosEntrega(respuesta string) []ArchivoEntrega {
	const marcador = "// FILE:"
	var resultado []ArchivoEntrega

	bloques := strings.Split(respuesta, marcador)
	for _, bloque := range bloques[1:] {
		bloque = strings.TrimLeft(bloque, " \t")
		fin := strings.IndexAny(bloque, "\r\n")
		if fin <= 0 {
			continue
		}
		ruta := normalizarRutaEntrega(strings.TrimSpace(bloque[:fin]))
		if ruta == "" {
			continue
		}
		contenido := strings.TrimLeft(bloque[fin:], "\r\n")
		contenido = recortarSeccionEvidencia(contenido)
		contenido = recortarFenceMarkdown(contenido)
		if strings.TrimSpace(contenido) == "" {
			continue
		}
		resultado = append(resultado, ArchivoEntrega{
			RutaRelativa: ruta,
			Contenido:    contenido,
		})
	}
	return resultado
}

func recortarFenceMarkdown(contenido string) string {
	contenido = strings.TrimSpace(contenido)
	if !strings.HasPrefix(contenido, "```") {
		return strings.TrimRight(contenido, "\n")
	}
	lineas := strings.Split(contenido, "\n")
	if len(lineas) < 3 {
		return strings.TrimRight(contenido, "\n")
	}
	if !strings.HasPrefix(strings.TrimSpace(lineas[0]), "```") {
		return strings.TrimRight(contenido, "\n")
	}
	cierre := -1
	for i := len(lineas) - 1; i >= 1; i-- {
		if strings.TrimSpace(lineas[i]) == "```" {
			cierre = i
			break
		}
	}
	if cierre <= 0 {
		return strings.TrimRight(contenido, "\n")
	}
	return strings.TrimRight(strings.Join(lineas[1:cierre], "\n"), "\n")
}

func recortarSeccionEvidencia(contenido string) string {
	lineas := strings.Split(contenido, "\n")
	for idx, linea := range lineas {
		if strings.EqualFold(strings.TrimSpace(linea), "evidencia:") {
			return strings.TrimRight(strings.Join(lineas[:idx], "\n"), "\n")
		}
	}
	return strings.TrimRight(contenido, "\n")
}

func normalizarRutaEntrega(ruta string) string {
	ruta = filepath.ToSlash(strings.TrimSpace(ruta))
	if ruta == "" || filepath.IsAbs(ruta) {
		return ""
	}
	limpia := filepath.Clean(ruta)
	if strings.HasPrefix(limpia, "..") {
		return ""
	}
	return limpia
}
