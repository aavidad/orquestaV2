package runtimesapp

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
	partes := []string{
		"PROTOCOLO_MICROPROGRAMACION_INLINE",
		"MODO: sin herramientas y sin acceso a filesystem o shell.",
		"REGLAS: trabaja solo con el contexto inline; no inventes archivos ocultos; no propongas refactors laterales; no cambies nada fuera del WRITE_SET.",
		"SI_FALTA_CONTEXTO: responde solo `BLOQUEO: <motivo concreto>`.",
		"SALIDA_OBLIGATORIA:",
		"PATCH_UNIFICADO: devuelve solo un diff unificado que toque unicamente archivos del WRITE_SET.",
		"EVIDENCIA: añade al final una seccion breve con 1-3 lineas maximo indicando que cambiaste y que test objetivo quedaria cubierto.",
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

func cargarArchivosContextoInline(rutaProyecto string, req MicroprogramacionDispatchRequest) ([]archivoContextoInline, error) {
	writeSet := normalizarWriteSetContexto(req.ArchivoObjetivo, req.WriteSet)
	if len(writeSet) == 0 {
		return nil, fmt.Errorf("write_set vacio para microprogramacion inline")
	}
	contexto := make([]archivoContextoInline, 0, len(writeSet)+2)
	vistos := map[string]struct{}{}
	for _, rutaRel := range writeSet {
		archivo, err := leerArchivoContextoInline(rutaProyecto, rutaRel, "WRITE_SET")
		if err != nil {
			return nil, err
		}
		contexto = append(contexto, archivo)
		vistos[rutaRel] = struct{}{}
	}
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
	return contexto, nil
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
