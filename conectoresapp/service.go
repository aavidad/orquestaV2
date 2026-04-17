package conectoresapp

import (
	"strings"
	"time"
)

type Conector struct {
	ID           int64
	Slug         string
	Nombre       string
	Transporte   string
	Comando      string
	ArgsJSON     string
	EnvJSON      string
	MetadataJSON string
	Activo       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Store interface {
	ListConnectors() ([]*Conector, error)
	GetConnector(ref string) (*Conector, error)
	SaveConnector(c *Conector) (int64, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type SaveConnectorInput struct {
	Slug         string
	Nombre       string
	Transporte   string
	Comando      string
	ArgsJSON     string
	EnvJSON      string
	MetadataJSON string
	Activo       bool
}

func (s *Service) ListConnectors() ([]*Conector, error) {
	return s.store.ListConnectors()
}

func (s *Service) GetConnector(ref string) (*Conector, error) {
	return s.store.GetConnector(strings.TrimSpace(ref))
}

func (s *Service) SaveConnector(input SaveConnectorInput) (int64, error) {
	return s.store.SaveConnector(&Conector{
		Slug:         strings.TrimSpace(input.Slug),
		Nombre:       strings.TrimSpace(input.Nombre),
		Transporte:   strings.TrimSpace(input.Transporte),
		Comando:      strings.TrimSpace(input.Comando),
		ArgsJSON:     strings.TrimSpace(input.ArgsJSON),
		EnvJSON:      strings.TrimSpace(input.EnvJSON),
		MetadataJSON: strings.TrimSpace(input.MetadataJSON),
		Activo:       input.Activo,
	})
}
