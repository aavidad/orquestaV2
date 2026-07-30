// El paquete main expone el censador histórico como una orden autónoma.
// La entrada solo valida argumentos; la coordinación y los efectos viven en
// las responsabilidades internas probadas por este mismo paquete.
package main

import (
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
	flag.Parse()
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
