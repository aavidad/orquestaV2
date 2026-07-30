// Este fichero concentra el catálogo castellano de la superficie de línea de órdenes.
package main

import (
	"fmt"
	"io"
)

var spanishCatalog = map[string]string{
	"flag.root":                  "raíz explícita alias=[modo:]ruta; se puede repetir",
	"flag.deny":                  "subárbol sensible alias=ruta/relativa; se puede repetir",
	"flag.jsonl":                 "salida JSONL obligatoria",
	"flag.manifest":              "manifiesto JSON obligatorio",
	"flag.max_entries":           "máximo de entradas, incluida una reserva de diagnóstico",
	"flag.max_directory_entries": "máximo de nombres cargados por directorio",
	"flag.max_depth":             "profundidad máxima por raíz",
	"flag.max_path_bytes":        "bytes máximos de ruta relativa",
	"flag.max_output_bytes":      "bytes máximos del flujo JSONL",
	"flag.max_hash_bytes":        "bytes globales que se pueden resumir",
	"flag.max_file_bytes":        "tamaño máximo por fichero",
	"flag.timeout":               "tiempo máximo total",
	"flag.help":                  "muestra esta ayuda",
	"usage.title":                "Uso del censador físico histórico:",
	"error.prefix":               "el censo físico no pudo completarse",
}

func renderSpanishHelp(output io.Writer) {
	lines := []string{
		"  --root alias=[metadata_only|source_content]:/ruta",
		"      " + spanishText("flag.root"),
		"  --deny alias=ruta/relativa",
		"      " + spanishText("flag.deny"),
		"  --jsonl /ruta/censo.jsonl",
		"      " + spanishText("flag.jsonl"),
		"  --manifest /ruta/censo.manifest.json",
		"      " + spanishText("flag.manifest"),
		"  --max-entries, --max-directory-entries, --max-depth",
		"      " + spanishText("flag.max_entries") + "; " + spanishText("flag.max_directory_entries") + "; " + spanishText("flag.max_depth"),
		"  --max-path-bytes, --max-output-bytes",
		"      " + spanishText("flag.max_path_bytes") + "; " + spanishText("flag.max_output_bytes"),
		"  --max-hash-bytes, --max-file-bytes, --timeout",
		"      " + spanishText("flag.max_hash_bytes") + "; " + spanishText("flag.max_file_bytes") + "; " + spanishText("flag.timeout"),
		"  --help, -h",
		"      " + spanishText("flag.help"),
	}
	fmt.Fprintln(output, spanishText("usage.title"))
	for _, line := range lines {
		fmt.Fprintln(output, line)
	}
}
func spanishText(key string) string {
	if value := spanishCatalog[key]; value != "" {
		return value
	}
	return key
}
