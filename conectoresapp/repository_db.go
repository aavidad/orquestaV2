package conectoresapp

import (
	"strings"

	"orquesta/db"
)

type Repository struct{}

func (Repository) ListConnectors() ([]*Conector, error) {
	items, err := db.ListarConectores()
	if err != nil {
		return nil, err
	}
	out := make([]*Conector, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, mapConector(item))
	}
	return out, nil
}

func (Repository) GetConnector(ref string) (*Conector, error) {
	item, err := db.GetConector(strings.TrimSpace(ref))
	if err != nil || item == nil {
		return nil, err
	}
	return mapConector(item), nil
}

func (Repository) SaveConnector(c *Conector) (int64, error) {
	return db.UpsertConector(mapDBConector(c))
}

func mapConector(in *db.Conector) *Conector {
	if in == nil {
		return nil
	}
	return &Conector{
		ID:           in.ID,
		Slug:         in.Slug,
		Nombre:       in.Nombre,
		Transporte:   in.Transporte,
		Comando:      in.Comando,
		ArgsJSON:     in.ArgsJSON,
		EnvJSON:      in.EnvJSON,
		MetadataJSON: in.MetadataJSON,
		Activo:       in.Activo,
		CreatedAt:    in.CreatedAt,
		UpdatedAt:    in.UpdatedAt,
	}
}

func mapDBConector(in *Conector) *db.Conector {
	if in == nil {
		return nil
	}
	return &db.Conector{
		ID:           in.ID,
		Slug:         in.Slug,
		Nombre:       in.Nombre,
		Transporte:   in.Transporte,
		Comando:      in.Comando,
		ArgsJSON:     in.ArgsJSON,
		EnvJSON:      in.EnvJSON,
		MetadataJSON: in.MetadataJSON,
		Activo:       in.Activo,
		CreatedAt:    in.CreatedAt,
		UpdatedAt:    in.UpdatedAt,
	}
}
