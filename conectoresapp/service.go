package conectoresapp

import (
	"strings"

	"orquesta/db"
)

type Store interface {
	ListConnectors() ([]*db.Conector, error)
	GetConnector(ref string) (*db.Conector, error)
	SaveConnector(c *db.Conector) (int64, error)
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

func (s *Service) ListConnectors() ([]*db.Conector, error) {
	return s.store.ListConnectors()
}

func (s *Service) GetConnector(ref string) (*db.Conector, error) {
	return s.store.GetConnector(strings.TrimSpace(ref))
}

func (s *Service) SaveConnector(input SaveConnectorInput) (int64, error) {
	return s.store.SaveConnector(&db.Conector{
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

type Repository struct{}

func (Repository) ListConnectors() ([]*db.Conector, error) {
	return db.ListarConectores()
}

func (Repository) GetConnector(ref string) (*db.Conector, error) {
	return db.GetConector(strings.TrimSpace(ref))
}

func (Repository) SaveConnector(c *db.Conector) (int64, error) {
	return db.UpsertConector(c)
}
