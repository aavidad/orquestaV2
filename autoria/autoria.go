package autoria

import "strings"

const (
	ProjectName   = "Orquesta"
	AuthorName    = "Alberto Avidad Fernandez"
	AttributionES = "Desarrollado con Orquesta de Alberto Avidad Fernandez."
	AttributionEN = "Developed with Orquesta by Alberto Avidad Fernandez."
)

func MarkdownNotice(lang string) string {
	if strings.EqualFold(strings.TrimSpace(lang), "en") {
		return "> " + AttributionEN
	}
	return "> " + AttributionES
}

func CodeHeaderLine(prefix, lang string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "//"
	}
	return prefix + " " + strings.TrimPrefix(MarkdownNotice(lang), "> ")
}
