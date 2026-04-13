package capacidadapp

import (
	"fmt"
	"strings"
	"time"
)

type ModeloRuntimeActivo struct {
	Modelo     string     `json:"modelo"`
	ID         string     `json:"id,omitempty"`
	Tamano     string     `json:"tamano,omitempty"`
	Procesador string     `json:"procesador,omitempty"`
	Contexto   string     `json:"contexto,omitempty"`
	Hasta      string     `json:"hasta,omitempty"`
	Runtime    string     `json:"runtime,omitempty"`
	ActivoEn   *time.Time `json:"activo_en,omitempty"`
}

type GestorRuntimeModelos interface {
	ListarModelosActivos() ([]ModeloRuntimeActivo, error)
	DescargarModelo(modelo string) error
}

func (s *Service) SetRuntimeModelManager(manager GestorRuntimeModelos) {
	s.runtimeModelManager = manager
}

func (s *Service) ListarModelosRuntimeActivos() ([]ModeloRuntimeActivo, error) {
	if s.runtimeModelManager == nil {
		return nil, nil
	}
	items, err := s.runtimeModelManager.ListarModelosActivos()
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) DescargarModeloRuntime(modelo string) error {
	if s.runtimeModelManager == nil {
		return fmt.Errorf("gestor de runtime de modelos no configurado")
	}
	modelo = strings.TrimSpace(modelo)
	if modelo == "" {
		return fmt.Errorf("modelo obligatorio")
	}
	return s.runtimeModelManager.DescargarModelo(modelo)
}

func (s *Service) DescargarModelosRuntimeActivos() ([]string, error) {
	if s.runtimeModelManager == nil {
		return nil, fmt.Errorf("gestor de runtime de modelos no configurado")
	}
	items, err := s.runtimeModelManager.ListarModelosActivos()
	if err != nil {
		return nil, err
	}
	descargados := make([]string, 0, len(items))
	for _, item := range items {
		modelo := strings.TrimSpace(item.Modelo)
		if modelo == "" {
			continue
		}
		if err := s.runtimeModelManager.DescargarModelo(modelo); err != nil {
			return descargados, err
		}
		descargados = append(descargados, modelo)
	}
	return descargados, nil
}
