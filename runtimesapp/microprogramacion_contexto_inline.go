package runtimesapp

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"orquesta/microprogramacionapp"
	"orquesta/runtimeagente"
)

const maxBytesContextoInlineMicro = 32 << 10

type archivoContextoInline struct {
	Ruta      string
	Etiqueta  string
	Contenido string
}

func (s *Service) prepararMensajeMicroprogramacion(req MicroprogramacionDispatchRequest) (string, error) {
	mensaje := strings.TrimSpace(req.Mensaje)
	if mensaje == "" {
		return "", fmt.Errorf("mensaje de microprogramacion obligatorio")
	}
	if !agenteUsaMicroprogramacionInline(req.AgenteDestino) {
		return mensaje, nil
	}
	if req.ProyectoID == nil || *req.ProyectoID <= 0 {
		return "", fmt.Errorf("proyecto obligatorio para microprogramacion inline")
	}
	proyecto, err := s.store.GetProject(strconv.FormatInt(*req.ProyectoID, 10))
	if err != nil {
		return "", err
	}
	if proyecto == nil || strings.TrimSpace(proyecto.RutaAbs) == "" {
		return "", fmt.Errorf("proyecto no disponible para microprogramacion inline")
	}
	return construirMensajeMicroprogramacionInline(strings.TrimSpace(proyecto.RutaAbs), req, mensaje)
}

func agenteUsaMicroprogramacionInline(agente string) bool {
	return strings.EqualFold(strings.TrimSpace(runtimeagente.ConectorPorDefectoAgente(strings.TrimSpace(agente))), "ollama-cli")
}

func construirMensajeMicroprogramacionInline(rutaProyecto string, req MicroprogramacionDispatchRequest, mensajeBase string) (string, error) {
	rutaProyecto = filepath.Clean(strings.TrimSpace(rutaProyecto))
	if rutaProyecto == "" {
		return "", fmt.Errorf("ruta de proyecto vacia")
	}
	archivos, err := cargarArchivosContextoInline(rutaProyecto, req)
	if err != nil {
		return "", err
	}
	mensajeBase = normalizarMensajeBaseMicroprogramacion(mensajeBase, req.FormatoSalida)
	usaBloquesArchivo := formatoSalidaUsaBloquesArchivo(req.FormatoSalida)
	usaGitWorktree := microprogramacionapp.FormatoSalidaUsaGitWorktree(req.FormatoSalida)
	salidaObligatoria := "PATCH_UNIFICADO: devuelve solo un diff unificado que toque unicamente archivos del WRITE_SET."
	if usaBloquesArchivo {
		salidaObligatoria = "FICHEROS: devuelve uno o varios bloques `// FILE: ruta/relativa` seguidos del contenido completo de cada fichero dentro del WRITE_SET."
	} else if usaGitWorktree {
		salidaObligatoria = "ENTREGA_GIT: trabaja dentro de tu worktree activa; modifica solo el WRITE_SET; ejecuta los tests obligatorios; responde solo con resumen breve, tests ejecutados y estado del diff/branch."
	}
	modo := "MODO: sin herramientas y sin acceso a filesystem o shell."
	reglas := "REGLAS: trabaja solo con el contexto inline; no inventes archivos ocultos; no propongas refactors laterales; no cambies nada fuera del WRITE_SET."
	if usaGitWorktree {
		modo = "MODO: usa tu worktree Git activa local; puedes editar archivos del WRITE_SET y ejecutar solo los tests obligatorios."
		reglas = "REGLAS: trabaja solo dentro de tu worktree activa; no inventes archivos ocultos; no propongas refactors laterales; no cambies nada fuera del WRITE_SET."
	}
	partes := []string{
		"PROTOCOLO_MICROPROGRAMACION_INLINE",
		modo,
		reglas,
		"RUTA_LITERAL_OBLIGATORIA: en cada bloque `// FILE:` usa exactamente una ruta del WRITE_SET, sin renombrarla, traducirla ni alterarla.",
		"SI_FALTA_CONTEXTO: responde solo `BLOQUEO: <motivo concreto>`.",
		"SALIDA_OBLIGATORIA:",
		salidaObligatoria,
		"",
		"MICROTAREA_ORIGINAL:",
		strings.TrimSpace(mensajeBase),
		"",
		"CONTEXTO_INLINE:",
	}
	for _, archivo := range archivos {
		partes = append(partes,
			fmt.Sprintf("=== %s: %s ===", archivo.Etiqueta, archivo.Ruta),
			"```",
			archivo.Contenido,
			"```",
			"",
		)
	}
	return strings.Join(partes, "\n"), nil
}

func normalizarMensajeBaseMicroprogramacion(mensajeBase, formatoSalida string) string {
	mensajeBase = strings.TrimSpace(mensajeBase)
	if mensajeBase == "" {
		return mensajeBase
	}
	lineas := strings.Split(mensajeBase, "\n")
	resultado := make([]string, 0, len(lineas))
	formatoSalida = strings.TrimSpace(formatoSalida)
	usaGitWorktree := microprogramacionapp.FormatoSalidaUsaGitWorktree(formatoSalida)
	for _, linea := range lineas {
		trimmed := strings.TrimSpace(linea)
		switch {
		case strings.HasPrefix(trimmed, "SALIDA:"):
			resultado = append(resultado, "SALIDA: "+formatoSalida)
		case !usaGitWorktree && strings.HasPrefix(trimmed, "ENTREGA_GIT:"):
			continue
		case !usaGitWorktree && strings.HasPrefix(trimmed, "RESPUESTA_ESPERADA:") && strings.Contains(strings.ToLower(trimmed), "git"):
			continue
		default:
			resultado = append(resultado, linea)
		}
	}
	return strings.Join(resultado, "\n")
}

func formatoSalidaUsaBloquesArchivo(formato string) bool {
	formato = strings.TrimSpace(strings.ToLower(formato))
	if formato == "" {
		return false
	}
	if strings.Contains(formato, "ficheros") || strings.Contains(formato, "// file:") || strings.Contains(formato, "bloques // file") {
		return true
	}
	return false
}

func cargarArchivosContextoInline(rutaProyecto string, req MicroprogramacionDispatchRequest) ([]archivoContextoInline, error) {
	writeSet := normalizarWriteSetContexto(req.ArchivoObjetivo, req.WriteSet)
	if len(writeSet) == 0 {
		return nil, fmt.Errorf("write_set vacio para microprogramacion inline")
	}
	testsObjetivo := descubrirTestsObjetivoInline(req.TestsObligatorios)
	contexto := make([]archivoContextoInline, 0, len(writeSet)+2)
	vistos := map[string]struct{}{}
	incluyeTestExplicito := false
	for _, rutaRel := range writeSet {
		archivo, err := leerArchivoContextoInline(rutaProyecto, rutaRel, "WRITE_SET")
		if err != nil {
			return nil, err
		}
		if rutaRel == filepath.ToSlash(filepath.Clean(strings.TrimSpace(req.ArchivoObjetivo))) {
			archivo = recortarArchivoObjetivoInline(archivo, strings.TrimSpace(req.SimboloObjetivo))
		} else if strings.HasSuffix(strings.ToLower(rutaRel), "_test.go") {
			archivo = recortarArchivoTestsInline(archivo, testsObjetivo)
		}
		contexto = append(contexto, archivo)
		vistos[rutaRel] = struct{}{}
		if strings.HasSuffix(strings.ToLower(rutaRel), "_test.go") {
			incluyeTestExplicito = true
		}
	}
	if !incluyeTestExplicito {
		for _, rutaRel := range descubrirTestsRelacionados(rutaProyecto, req.ArchivoObjetivo) {
			if _, ok := vistos[rutaRel]; ok {
				continue
			}
			archivo, err := leerArchivoContextoInline(rutaProyecto, rutaRel, "TEST_REFERENCIA")
			if err != nil {
				return nil, err
			}
			contexto = append(contexto, archivo)
			vistos[rutaRel] = struct{}{}
		}
	}
	return contexto, nil
}

func descubrirTestsObjetivoInline(tests []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tests))
	re := regexp.MustCompile(`(?:^|[\s=])(-run)\s+([A-Za-z0-9_]+)|(?:^|[\s=])-run=([A-Za-z0-9_]+)`)
	for _, raw := range tests {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		matches := re.FindAllStringSubmatch(raw, -1)
		for _, match := range matches {
			candidato := ""
			if len(match) >= 4 {
				switch {
				case strings.TrimSpace(match[2]) != "":
					candidato = strings.TrimSpace(match[2])
				case strings.TrimSpace(match[3]) != "":
					candidato = strings.TrimSpace(match[3])
				}
			}
			if !strings.HasPrefix(candidato, "Test") {
				continue
			}
			if _, ok := seen[candidato]; ok {
				continue
			}
			seen[candidato] = struct{}{}
			out = append(out, candidato)
		}
	}
	sort.Strings(out)
	return out
}

func recortarArchivoObjetivoInline(archivo archivoContextoInline, simbolo string) archivoContextoInline {
	simbolo = strings.TrimSpace(simbolo)
	if simbolo == "" || !strings.HasSuffix(strings.ToLower(strings.TrimSpace(archivo.Ruta)), ".go") {
		return archivo
	}
	recortado, ok := extraerContextoSimboloGo(archivo.Contenido, simbolo)
	if !ok {
		return archivo
	}
	recortado = strings.TrimSpace(recortado)
	if recortado == "" || len(recortado) >= len(archivo.Contenido) {
		return archivo
	}
	archivo.Contenido = recortado
	return archivo
}

func recortarArchivoTestsInline(archivo archivoContextoInline, testsObjetivo []string) archivoContextoInline {
	if len(testsObjetivo) == 0 || !strings.HasSuffix(strings.ToLower(strings.TrimSpace(archivo.Ruta)), "_test.go") {
		return archivo
	}
	recortado, ok := extraerContextoTestsGo(archivo.Contenido, testsObjetivo)
	if !ok {
		return archivo
	}
	recortado = strings.TrimSpace(recortado)
	if recortado == "" || len(recortado) >= len(archivo.Contenido) {
		return archivo
	}
	archivo.Contenido = recortado
	return archivo
}

func extraerContextoSimboloGo(contenido, simbolo string) (string, bool) {
	fs := token.NewFileSet()
	archivo, err := parser.ParseFile(fs, "inline.go", contenido, parser.ParseComments)
	if err != nil || archivo == nil {
		return "", false
	}
	var declaracion ast.Decl
	for _, decl := range archivo.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			if typed != nil && typed.Name != nil && strings.EqualFold(strings.TrimSpace(typed.Name.Name), simbolo) {
				declaracion = typed
			}
		case *ast.GenDecl:
			for _, spec := range typed.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					if item != nil && item.Name != nil && strings.EqualFold(strings.TrimSpace(item.Name.Name), simbolo) {
						declaracion = typed
					}
				case *ast.ValueSpec:
					for _, nombre := range item.Names {
						if nombre != nil && strings.EqualFold(strings.TrimSpace(nombre.Name), simbolo) {
							declaracion = typed
							break
						}
					}
				}
				if declaracion != nil {
					break
				}
			}
		}
		if declaracion != nil {
			break
		}
	}
	if declaracion == nil {
		return "", false
	}
	var out bytes.Buffer
	out.WriteString("package ")
	out.WriteString(strings.TrimSpace(archivo.Name.Name))
	out.WriteString("\n\n")
	if len(archivo.Imports) > 0 {
		var imports bytes.Buffer
		gen := &ast.GenDecl{Tok: token.IMPORT}
		for _, imp := range archivo.Imports {
			gen.Specs = append(gen.Specs, imp)
		}
		if err := printer.Fprint(&imports, fs, gen); err == nil {
			out.WriteString(strings.TrimSpace(imports.String()))
			out.WriteString("\n\n")
		}
	}
	if err := printer.Fprint(&out, fs, declaracion); err != nil {
		return "", false
	}
	return out.String(), true
}

func extraerContextoTestsGo(contenido string, testsObjetivo []string) (string, bool) {
	if len(testsObjetivo) == 0 {
		return "", false
	}
	lookup := make(map[string]struct{}, len(testsObjetivo))
	for _, test := range testsObjetivo {
		test = strings.TrimSpace(test)
		if test != "" {
			lookup[test] = struct{}{}
		}
	}
	if len(lookup) == 0 {
		return "", false
	}
	fs := token.NewFileSet()
	archivo, err := parser.ParseFile(fs, "inline_test.go", contenido, parser.ParseComments)
	if err != nil || archivo == nil {
		return "", false
	}
	var declaraciones []ast.Decl
	for _, decl := range archivo.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn == nil || fn.Name == nil {
			continue
		}
		if _, ok := lookup[strings.TrimSpace(fn.Name.Name)]; ok {
			declaraciones = append(declaraciones, decl)
		}
	}
	if len(declaraciones) == 0 {
		return "", false
	}
	var out bytes.Buffer
	out.WriteString("package ")
	out.WriteString(strings.TrimSpace(archivo.Name.Name))
	out.WriteString("\n\n")
	if len(archivo.Imports) > 0 {
		var imports bytes.Buffer
		gen := &ast.GenDecl{Tok: token.IMPORT}
		for _, imp := range archivo.Imports {
			gen.Specs = append(gen.Specs, imp)
		}
		if err := printer.Fprint(&imports, fs, gen); err == nil {
			out.WriteString(strings.TrimSpace(imports.String()))
			out.WriteString("\n\n")
		}
	}
	for idx, decl := range declaraciones {
		if idx > 0 {
			out.WriteString("\n\n")
		}
		if err := printer.Fprint(&out, fs, decl); err != nil {
			return "", false
		}
	}
	return out.String(), true
}

func normalizarWriteSetContexto(archivoObjetivo string, writeSet []string) []string {
	out := make([]string, 0, len(writeSet)+1)
	add := func(raw string) {
		raw = filepath.ToSlash(filepath.Clean(strings.TrimSpace(raw)))
		if raw == "" || raw == "." || raw == ".." || strings.HasPrefix(raw, "../") {
			return
		}
		for _, existente := range out {
			if existente == raw {
				return
			}
		}
		out = append(out, raw)
	}
	add(archivoObjetivo)
	for _, item := range writeSet {
		add(item)
	}
	return out
}

func descubrirTestsRelacionados(rutaProyecto, archivoObjetivo string) []string {
	archivoObjetivo = filepath.ToSlash(filepath.Clean(strings.TrimSpace(archivoObjetivo)))
	if archivoObjetivo == "" || archivoObjetivo == "." || archivoObjetivo == ".." || strings.HasPrefix(archivoObjetivo, "../") {
		return nil
	}
	dirRel := filepath.ToSlash(filepath.Dir(archivoObjetivo))
	if dirRel == "." {
		dirRel = ""
	}
	patron := filepath.Join(rutaProyecto, filepath.FromSlash(dirRel), "*_test.go")
	matches, err := filepath.Glob(patron)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		rel, err := filepath.Rel(rutaProyecto, match)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
		if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
			continue
		}
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}

func leerArchivoContextoInline(rutaProyecto, rutaRel, etiqueta string) (archivoContextoInline, error) {
	rutaRel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rutaRel)))
	if rutaRel == "" || rutaRel == "." || rutaRel == ".." || strings.HasPrefix(rutaRel, "../") {
		return archivoContextoInline{}, fmt.Errorf("ruta inline invalida: %q", rutaRel)
	}
	rutaAbs := filepath.Join(rutaProyecto, filepath.FromSlash(rutaRel))
	absProyecto := filepath.Clean(rutaProyecto) + string(os.PathSeparator)
	absArchivo := filepath.Clean(rutaAbs)
	if absArchivo != filepath.Clean(rutaProyecto) && !strings.HasPrefix(absArchivo, absProyecto) {
		return archivoContextoInline{}, fmt.Errorf("ruta inline fuera del proyecto: %q", rutaRel)
	}
	contenido, err := os.ReadFile(absArchivo)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return archivoContextoInline{
				Ruta:     rutaRel,
				Etiqueta: strings.TrimSpace(etiqueta),
				Contenido: strings.Join([]string{
					"// ARCHIVO_NO_EXISTE_TODAVIA",
					"// Crea este archivo solo si la microtarea lo requiere.",
					"// Debe permanecer dentro del WRITE_SET.",
				}, "\n"),
			}, nil
		}
		return archivoContextoInline{}, fmt.Errorf("leyendo contexto inline %q: %w", rutaRel, err)
	}
	if len(contenido) > maxBytesContextoInlineMicro {
		return archivoContextoInline{}, fmt.Errorf("el archivo %q excede el limite de contexto inline (%d bytes)", rutaRel, maxBytesContextoInlineMicro)
	}
	return archivoContextoInline{
		Ruta:      rutaRel,
		Etiqueta:  strings.TrimSpace(etiqueta),
		Contenido: strings.TrimRight(string(contenido), "\n"),
	}, nil
}
