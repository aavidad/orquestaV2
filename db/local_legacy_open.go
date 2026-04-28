package db

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"orquesta/storage"
)

func shouldUseLocalRecoveryOpen(cfg storage.Config) bool {
	if !supportsPrepareLiteReadOnlyBackend(cfg.Driver) {
		return false
	}
	if !envBoolEnabled("ORQUESTA_FORCE_LOCAL_DB") && !envBoolEnabled("ORQUESTA_FORCE_LOCAL") {
		return false
	}
	target := strings.TrimSpace(cfg.Path)
	if target == "" {
		return false
	}
	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Size() > 0
}

func envBoolEnabled(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return value == "1" || value == "true" || value == "yes" || value == "si" || value == "on"
}

// Orden de resolución del target por fichero local cuando el backend activo usa
// persistencia basada en ruta. Solo devuelve rutas existentes; no inventa un
// almacenamiento implícito por omisión:
//  1. Variable de entorno ORQUESTA_DB
//  2. Repositorio `orquesta`/`orquestador` del workspace actual con fichero existente
//  3. Si el git-root ya es ese repo, <git-root>/orquesta.db existente
func resolverRuta() string {
	if v := os.Getenv("ORQUESTA_DB"); strings.TrimSpace(v) != "" {
		return v
	}
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return resolverRutaDesdeGitRoot(strings.TrimSpace(string(out)))
	}
	if wd, err := os.Getwd(); err == nil {
		if ruta := buscarRutaRepoOrquesta(wd); ruta != "" {
			return ruta
		}
	}
	return ""
}

func resolverRutaDesdeGitRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	if filepath.Base(root) == "orquesta" {
		ruta := filepath.Join(root, "orquesta.db")
		if existeFichero(ruta) {
			return ruta
		}
		return ""
	}
	if ruta := rutaRepoOrquestaEnDirectorio(filepath.Dir(root)); ruta != "" {
		return ruta
	}
	return ""
}

func buscarRutaRepoOrquesta(inicio string) string {
	actual := filepath.Clean(inicio)
	for {
		if ruta := rutaRepoOrquestaEnDirectorio(actual); ruta != "" {
			return ruta
		}
		siguiente := filepath.Dir(actual)
		if siguiente == actual {
			return ""
		}
		actual = siguiente
	}
}

func rutaRepoOrquestaEnDirectorio(base string) string {
	for _, nombre := range []string{"orquesta", "orquestador"} {
		candidato := filepath.Join(base, nombre)
		rutaDB := filepath.Join(candidato, "orquesta.db")
		if existeFichero(filepath.Join(candidato, "go.mod")) && existeFichero(rutaDB) {
			return rutaDB
		}
	}
	return ""
}

func existeFichero(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func usesLegacyLocalDriver(driver string) bool {
	return normalizedDriverName(driver) == "sqlite"
}
