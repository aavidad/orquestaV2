package db

import "orquesta/coordinacion"

func WorktreeActivaCoherente(projectID int64, path string) bool {
	if projectID <= 0 {
		return true
	}
	path = normalizarRutaProyecto(path)
	if path == "" {
		return false
	}
	proyecto, err := GetProyecto(jsonNumber(projectID))
	if err != nil || proyecto == nil {
		return true
	}
	rutaBase := RutaProyectoEfectiva(proyecto.ID, proyecto.RutaAbs, "")
	if rutaBase == "" {
		return true
	}
	return coordinacion.ActiveWorktreePathCoherent(path, rutaBase)
}
