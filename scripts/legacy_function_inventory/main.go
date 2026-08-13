// El paquete main expone el censador histórico como una orden autónoma.
// La entrada solo valida argumentos; la coordinación y los efectos viven en
// las responsabilidades internas probadas por este mismo paquete.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	var options options
	flag.StringVar(&options.repository, "repo", ".", "repositorio Git que se censará")
	flag.StringVar(&options.jsonl, "jsonl", "", "salida JSONL obligatoria")
	flag.StringVar(&options.manifest, "manifest", "", "manifiesto JSON obligatorio")
	lookupPath := flag.String("lookup-index-path", "", "ruta Go productiva exacta del índice Git")
	lookupSymbol := flag.String("lookup-symbol", "", "nombre exacto opcional de función o método")
	snapshotJSONL := flag.String("snapshot-index-jsonl", "", "índice estructural JSONL del snapshot Git")
	snapshotManifest := flag.String("snapshot-index-manifest", "", "manifiesto del índice estructural")
	flag.Parse()
	if *snapshotJSONL != "" || *snapshotManifest != "" {
		if *snapshotJSONL == "" || *snapshotManifest == "" || *lookupPath != "" ||
			options.jsonl != "" || options.manifest != "" {
			fatal(errors.New("el índice de snapshot exige sus dos salidas y no admite otros modos"))
		}
		if err := writeSnapshotFunctionIndex(options.repository, *snapshotJSONL, *snapshotManifest); err != nil {
			fatal(err)
		}
		return
	}
	if *lookupPath != "" {
		if options.jsonl != "" || options.manifest != "" {
			fatal(errors.New("la consulta del índice no admite salidas de censo"))
		}
		records, err := lookupIndexSymbols(options.repository, *lookupPath, *lookupSymbol)
		if err != nil {
			fatal(err)
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetEscapeHTML(false)
		for _, item := range records {
			if err := encoder.Encode(item); err != nil {
				fatal(err)
			}
		}
		return
	}
	if options.jsonl == "" || options.manifest == "" {
		fatal(errors.New("debes indicar --jsonl y --manifest"))
	}
	if err := run(options); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "legacy_function_inventory:", err)
	os.Exit(1)
}
