package microprogramacionapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EscritorArchivos interface {
	Escribir(raizProyecto string, archivos []ArchivoEntrega) ([]string, error)
}

type EscritorArchivosDisco struct{}

func (EscritorArchivosDisco) Escribir(raizProyecto string, archivos []ArchivoEntrega) ([]string, error) {
	raiz := filepath.Clean(strings.TrimSpace(raizProyecto))
	if raiz == "" {
		return nil, fmt.Errorf("raiz de proyecto obligatoria")
	}
	var escritos []string
	for _, archivo := range archivos {
		ruta := strings.TrimSpace(archivo.RutaRelativa)
		if ruta == "" {
			return nil, fmt.Errorf("ruta de archivo obligatoria")
		}
		destino := filepath.Join(raiz, ruta)
		destino = filepath.Clean(destino)
		if !strings.HasPrefix(destino+string(filepath.Separator), raiz+string(filepath.Separator)) {
			return nil, fmt.Errorf("ruta materializada fuera del proyecto: %s", ruta)
		}
		if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(destino, []byte(archivo.Contenido), 0o644); err != nil {
			return nil, err
		}
		escritos = append(escritos, ruta)
	}
	return escritos, nil
}
