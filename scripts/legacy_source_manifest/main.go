// Este fichero contiene únicamente la entrada y la coordinación del manifiesto.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type repeatedFlag []string

func (values *repeatedFlag) String() string {
	return strings.Join(*values, ",")
}

func (values *repeatedFlag) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("la ruta no puede estar vacía")
	}
	*values = append(*values, value)
	return nil
}

func main() {
	var roots repeatedFlag
	var excluded repeatedFlag
	var output string
	var timeout time.Duration
	flag.Var(&roots, "root", "raíz histórica explícita; se puede repetir")
	flag.Var(&excluded, "exclude", "ruta exacta que no debe inspeccionarse; se puede repetir")
	flag.StringVar(&output, "manifest", "", "fichero JSON de salida obligatorio")
	flag.DurationVar(&timeout, "git-timeout", defaultGitTimeout, "límite por consulta Git")
	flag.Parse()

	if len(roots) == 0 || strings.TrimSpace(output) == "" {
		fatal(errors.New("debes indicar al menos un --root y un --manifest"))
	}
	if timeout <= 0 {
		fatal(errors.New("--git-timeout debe ser positivo"))
	}
	if err := run(options{
		roots:        roots,
		excluded:     excluded,
		manifestPath: output,
		gitTimeout:   timeout,
	}); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "legacy_source_manifest:", err)
	os.Exit(1)
}

func run(opts options) error {
	normalizedRoots, err := normalizeUniquePaths(opts.roots)
	if err != nil {
		return fmt.Errorf("raíces: %w", err)
	}
	normalizedExcluded, err := normalizeUniquePaths(opts.excluded)
	if err != nil {
		return fmt.Errorf("exclusiones: %w", err)
	}
	manifestPath, err := filepath.Abs(opts.manifestPath)
	if err != nil {
		return fmt.Errorf("salida: %w", err)
	}
	manifestPath = filepath.Clean(manifestPath)
	for _, root := range normalizedRoots {
		if pathWithin(root, manifestPath) {
			return fmt.Errorf("la salida no puede escribirse dentro de la raíz histórica %q", root)
		}
	}
	for _, excluded := range normalizedExcluded {
		if !withinAnyRoot(normalizedRoots, excluded) {
			return fmt.Errorf("la exclusión %q queda fuera de las raíces explícitas", excluded)
		}
	}

	if opts.gitTimeout <= 0 {
		opts.gitTimeout = defaultGitTimeout
	}
	collected := collector{sources: make(map[string]sourceRecord)}
	for _, excluded := range normalizedExcluded {
		collected.add(sealSource(sourceRecord{
			Path:   excluded,
			Kind:   "path",
			Status: "excluded",
			Reason: "operator_excluded",
		}))
	}

	roots := make([]rootRecord, 0, len(normalizedRoots))
	for _, root := range normalizedRoots {
		roots = append(roots, scanRoot(root, normalizedExcluded, opts.gitTimeout, &collected))
	}
	sources := collected.sorted()
	result := manifest{
		SchemaVersion: manifestSchemaVersion,
		Algorithm:     manifestAlgorithm,
		Roots:         roots,
		ExcludedPaths: normalizedExcluded,
		Sources:       sources,
		Summary:       summarize(roots, sources),
	}
	result.ManifestSHA256 = digestJSON(manifestAlgorithm, result)
	content, err := marshalManifest(result)
	if err != nil {
		return err
	}
	return writeAtomic(manifestPath, content, 0o600)
}
