// Este fichero prueba que raíces, denegaciones y salidas se resuelven sin alias peligrosos.
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseRootsRequiresExplicitUniqueRoots(t *testing.T) {
	root := t.TempDir()
	roots, err := parseRoots([]string{"historia=" + root})
	if err != nil {
		t.Fatal(err)
	}
	if roots[0].mode != modeMetadata || roots[0].alias != "historia" {
		t.Fatalf("raíz predeterminada inesperada: %#v", roots[0])
	}
	contentRoots, err := parseRoots([]string{"codigo=" + modeContent + ":" + root})
	if err != nil {
		t.Fatal(err)
	}
	if contentRoots[0].mode != modeContent {
		t.Fatalf("modo explícito perdido: %#v", contentRoots[0])
	}
	if _, err := parseRoots([]string{"historia=" + root, "historia=" + root}); err == nil {
		t.Fatal("se admitió un alias repetido")
	}
	link := filepath.Join(t.TempDir(), "raiz-enlace")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	values, err := parseRoots([]string{"enlace=" + link})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := anchorRoot(values[0]); err == nil {
		t.Fatal("se ancló una raíz que era enlace simbólico")
	}
}
func TestCommandHelpAndErrorsRemainInSpanishOrMachineCodes(t *testing.T) {
	var output bytes.Buffer
	if err := executeCommand([]string{"--help"}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Uso del censador físico histórico:") ||
		strings.Contains(output.String(), "Usage of") ||
		strings.Contains(output.String(), "(default ") ||
		strings.Contains(output.String(), " duration") ||
		strings.Contains(output.String(), " string") ||
		strings.Contains(output.String(), " value") {
		t.Fatalf("ayuda no castellana: %q", output.String())
	}
	output.Reset()
	err := executeCommand([]string{"--bandera-inexistente"}, &output)
	if err == nil {
		t.Fatal("una bandera desconocida fue aceptada")
	}
	writeCommandError(&output, err)
	if !strings.Contains(output.String(), "el censo físico no pudo completarse [invalid_input]") ||
		strings.Contains(output.String(), "flag provided") ||
		strings.Contains(output.String(), "Usage of") ||
		strings.Contains(output.String(), "default") ||
		strings.Contains(output.String(), "duration") ||
		strings.Contains(output.String(), "string") ||
		strings.Contains(output.String(), "value") {
		t.Fatalf("error visible no catalogado: %q", output.String())
	}
	if err := executeCommand([]string{"residuo"}, &output); !errors.Is(err, errUnexpectedArguments) {
		t.Fatalf("argumento posicional aceptado: %v", err)
	}
}
func TestValidateOptionsResolvesOutputParentAliases(t *testing.T) {
	root := t.TempDir()
	aliasParent := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, aliasParent); err != nil {
		t.Fatal(err)
	}
	opts := options{
		roots:        []rootOption{{alias: "historia", mode: modeMetadata, path: root}},
		jsonlPath:    filepath.Join(aliasParent, "censo.jsonl"),
		manifestPath: filepath.Join(t.TempDir(), "censo.json"),
		budget: budgetOptions{
			maxEntries: 10, maxDirectoryEntries: 10, maxFileBytes: 10,
			maxDepth: 10, maxPathBytes: 100, maxOutputBytes: 10 << 10,
			maxHashBytes: 10, timeout: time.Second,
		},
	}
	if err := validateOptions(&opts); err == nil {
		t.Fatal("se admitió una salida dentro de la raíz a través de un alias")
	}
}
func TestValidateOptionsRejectsExistingNonRegularOutput(t *testing.T) {
	root := t.TempDir()
	outputParent := t.TempDir()
	outputDirectory := filepath.Join(outputParent, "ocupado")
	if err := os.Mkdir(outputDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	opts := options{
		roots:        []rootOption{{alias: "historia", mode: modeMetadata, path: root}},
		jsonlPath:    outputDirectory,
		manifestPath: filepath.Join(outputParent, "censo.json"),
		budget: budgetOptions{
			maxEntries: 10, maxDirectoryEntries: 10, maxFileBytes: 10,
			maxDepth: 10, maxPathBytes: 100, maxOutputBytes: 10 << 10,
			maxHashBytes: 10, timeout: time.Second,
		},
	}
	if err := validateOptions(&opts); err != nil {
		t.Fatal(err)
	}
	if _, err := newPublicationForTest(opts); err == nil {
		t.Fatal("se admitió un directorio como salida final")
	}
}
func TestDeniedPathsAreRelativeAndScoped(t *testing.T) {
	root := rootOption{alias: "historia"}
	roots := []rootOption{root}
	if err := applyDenied(roots, []string{"historia=secreto/claves"}); err != nil {
		t.Fatal(err)
	}
	if !denied([]string{"secreto", "claves", "actual"}, roots[0].denied) {
		t.Fatal("la denegación no cubrió el subárbol")
	}
	for _, invalid := range []string{"otra=secreto", "historia=../fuera", "historia=/absoluta"} {
		candidate := []rootOption{{alias: "historia"}}
		if err := applyDenied(candidate, []string{invalid}); err == nil {
			t.Fatalf("se admitió denegación inválida %q", invalid)
		}
	}
}
func TestInputEnvelopeAndPreScanDeadlineAreBounded(t *testing.T) {
	if err := executeCommand(make([]string, maxInputArguments+1), &bytes.Buffer{}); !errors.Is(err, errInvalidInput) {
		t.Fatalf("envolvente de argumentos sin límite: %v", err)
	}
	if err := validateInputEnvelope([]string{strings.Repeat("x", int(maxInputBytes))}); err != nil {
		t.Fatalf("la frontera de bytes fue rechazada: %v", err)
	}
	if err := validateInputEnvelope([]string{strings.Repeat("x", int(maxInputBytes+1))}); !errors.Is(err, errInvalidInput) {
		t.Fatalf("el byte sobre la frontera fue aceptado: %v", err)
	}
	opts := testOptions(t, t.TempDir(), modeMetadata)
	opts.roots = make([]rootOption, maxRootInputs+1)
	if err := validateOptions(&opts); !errors.Is(err, errInvalidInput) {
		t.Fatalf("cantidad de raíces sin límite: %v", err)
	}
	opts = testOptions(t, t.TempDir(), modeMetadata)
	opts.roots[0].denied = make([][]string, maxDeniedInputs+1)
	if err := validateOptions(&opts); !errors.Is(err, errInvalidInput) {
		t.Fatalf("cantidad de exclusiones sin límite: %v", err)
	}
	opts = testOptions(t, t.TempDir(), modeMetadata)
	err := executeCommandStarted([]string{
		"--root", "prueba=" + opts.roots[0].path,
		"--jsonl", opts.jsonlPath, "--manifest", opts.manifestPath,
	}, &bytes.Buffer{}, time.Now().Add(-defaultTimeout))
	if !errors.Is(err, errBudget) {
		t.Fatalf("el parseo completo ignoró el plazo inicial: %v", err)
	}
}
func TestCanonicalBudgetCeilingsAcceptBoundaryAndRejectOverflow(t *testing.T) {
	opts := testOptions(t, t.TempDir(), modeMetadata)
	opts.budget = budgetOptions{
		maxEntries: defaultMaxEntries, maxDirectoryEntries: defaultMaxDirectoryEntries,
		maxDepth: defaultMaxDepth, maxPathBytes: defaultMaxPathBytes,
		maxOutputBytes: defaultMaxOutputBytes, maxHashBytes: defaultMaxHashBytes,
		maxFileBytes: defaultMaxFileBytes, timeout: defaultTimeout,
	}
	if err := validateOptions(&opts); err != nil {
		t.Fatalf("los máximos canónicos fueron rechazados: %v", err)
	}
	overflows := []func(*budgetOptions){
		func(value *budgetOptions) { value.maxEntries++ },
		func(value *budgetOptions) { value.maxDirectoryEntries++ },
		func(value *budgetOptions) { value.maxDepth++ },
		func(value *budgetOptions) { value.maxPathBytes++ },
		func(value *budgetOptions) { value.maxOutputBytes++ },
		func(value *budgetOptions) { value.maxHashBytes++ },
		func(value *budgetOptions) { value.maxFileBytes++ },
		func(value *budgetOptions) { value.timeout++ },
	}
	for index, overflow := range overflows {
		candidate := opts
		overflow(&candidate.budget)
		if err := validateOptions(&candidate); !errors.Is(err, errInvalidInput) {
			t.Fatalf("techo %d sin ratchet: %v", index, err)
		}
	}
}
