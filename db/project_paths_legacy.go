package db

import (
	"path/filepath"
	"strings"

	"orquesta/coordinacion"
)

func rutaProyectoWorktreeRaw(proyectoID int64) string {
	if proyectoID <= 0 {
		return ""
	}
	proyecto, err := GetProyecto(jsonNumber(proyectoID))
	if err != nil || proyecto == nil {
		return ""
	}
	return normalizarRutaProyecto(proyecto.RutaAbs)
}

func rutaProyectoWorktreeEfectiva(proyectoID int64) string {
	if proyectoID <= 0 {
		return ""
	}
	proyecto, err := GetProyectoConRutaEfectiva(jsonNumber(proyectoID), "")
	if err != nil || proyecto == nil {
		return rutaProyectoWorktreeRaw(proyectoID)
	}
	return normalizarRutaProyecto(proyecto.RutaAbs)
}

func rutaProyectoEscopadaCanonica(rutaProyecto, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	rutaProyecto = normalizarRutaProyecto(rutaProyecto)
	if rutaProyecto == "" {
		return normalizarRutaProyecto(raw)
	}
	clean := filepath.Clean(raw)
	if clean == "." {
		return rutaProyecto
	}
	return filepath.Clean(filepath.Join(rutaProyecto, clean))
}

func rutaWorktreeCanonicaProyecto(proyectoID int64, raw string) string {
	return rutaProyectoEscopadaCanonica(rutaProyectoWorktreeEfectiva(proyectoID), raw)
}

func rutaWorktreeCanonicaProyectoRaw(proyectoID int64, raw string) string {
	return rutaProyectoEscopadaCanonica(rutaProyectoWorktreeRaw(proyectoID), raw)
}

func rutaRuntimeCanonicaProyecto(agente string, proyectoID *int64, cwd string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return ""
	}
	if canonical, err := CanonicalizeAgentName(agente); err == nil {
		agente = canonical
	}
	if proyectoID == nil || *proyectoID <= 0 {
		return normalizarRutaProyecto(cwd)
	}
	rutaProyecto := rutaProyectoWorktreeEfectiva(*proyectoID)
	rutaWorktree := rutaWorktreeActivaAgenteProyecto(*proyectoID, strings.TrimSpace(agente))
	return rutaRuntimeCanonicaConProyecto(rutaProyecto, rutaWorktree, cwd)
}

func rutaRuntimeCanonicaConProyecto(rutaProyecto, rutaWorktree, cwd string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return ""
	}
	rutaProyecto = normalizarRutaProyecto(rutaProyecto)
	rutaWorktree = normalizarRutaProyecto(rutaWorktree)
	canonica := rutaProyectoEscopadaCanonica(rutaProyecto, cwd)
	if rutaWorktree != "" {
		if coordinacion.PathWithin(rutaWorktree, canonica) {
			return canonica
		}
		if remapeada := remapearRutaHistoricaAWorktreeActiva(canonica, rutaWorktree); remapeada != "" {
			return remapeada
		}
	}
	if rutaProyecto != "" {
		if coordinacion.WorkPathBelongsToProject(canonica, rutaProyecto, rutaWorktree) {
			return canonica
		}
		return rutaProyecto
	}
	return canonica
}

func remapearRutaHistoricaAWorktreeActiva(raw, rutaWorktree string) string {
	nombre, sufijo, ok := descomponerRutaWorktree(raw)
	if !ok || filepath.Base(rutaWorktree) != nombre {
		return ""
	}
	if sufijo == "" {
		return rutaWorktree
	}
	return filepath.Clean(filepath.Join(rutaWorktree, sufijo))
}

func descomponerRutaWorktree(raw string) (nombre, sufijo string, ok bool) {
	raw = normalizarRutaProyecto(raw)
	if raw == "" {
		return "", "", false
	}
	marker := string(filepath.Separator) + ".orquesta-worktrees" + string(filepath.Separator)
	pos := strings.Index(raw, marker)
	if pos < 0 {
		return "", "", false
	}
	resto := strings.TrimLeft(raw[pos+len(marker):], string(filepath.Separator))
	if resto == "" {
		return "", "", false
	}
	partes := strings.Split(resto, string(filepath.Separator))
	if len(partes) == 0 || strings.TrimSpace(partes[0]) == "" {
		return "", "", false
	}
	nombre = strings.TrimSpace(partes[0])
	if len(partes) > 1 {
		sufijo = filepath.Join(partes[1:]...)
	}
	return nombre, sufijo, true
}
