package smokehelper_gemini

import (
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// NormalizarTokens recibe un string y devuelve un slice de tokens normalizados.
// La normalización consiste en convertir a minúsculas y separar por caracteres no alfanuméricos.
func NormalizarTokens(input string) []string {
	// Convertir a minúsculas
	input = strings.ToLower(input)

	// Reemplazar caracteres no alfanuméricos por espacios
	cleaned := re.ReplaceAllString(input, " ")

	// Dividir por espacios
	parts := strings.Fields(cleaned)

	return parts
}
