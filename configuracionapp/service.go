package configuracionapp

import (
	"strings"

	"orquesta/db"
)

type Store interface {
	GetConfig(clave string) (string, error)
	ListConfig() (map[string]string, error)
	SetConfig(clave, valor string) error
	RegisterAgent(nombre, rol string) error
	RetireAgent(nombre string) error
	RehabilitateAgent(nombre string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Get(clave string) (string, error) {
	return s.store.GetConfig(strings.TrimSpace(clave))
}

func (s *Service) List() (map[string]string, error) {
	return s.store.ListConfig()
}

func (s *Service) Set(clave, valor string) error {
	return s.store.SetConfig(strings.TrimSpace(clave), strings.TrimSpace(valor))
}

func (s *Service) RegisterAgent(nombre, rol string) error {
	return s.store.RegisterAgent(strings.TrimSpace(nombre), strings.TrimSpace(rol))
}

func (s *Service) RetireAgent(nombre string) error {
	return s.store.RetireAgent(strings.TrimSpace(nombre))
}

func (s *Service) RehabilitateAgent(nombre string) error {
	return s.store.RehabilitateAgent(strings.TrimSpace(nombre))
}

type Repository struct{}

func (Repository) GetConfig(clave string) (string, error) {
	return db.ConfigGet(clave)
}

func (Repository) ListConfig() (map[string]string, error) {
	return db.ConfigAll()
}

func (Repository) SetConfig(clave, valor string) error {
	return db.ConfigSet(clave, valor)
}

func (Repository) RegisterAgent(nombre, rol string) error {
	return db.RegistrarAgente(nombre, rol)
}

func (Repository) RetireAgent(nombre string) error {
	return db.RetirarAgente(nombre)
}

func (Repository) RehabilitateAgent(nombre string) error {
	return db.RehabilitarAgente(nombre)
}
