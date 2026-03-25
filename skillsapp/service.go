package skillsapp

import (
	"context"
	"fmt"
	"strings"

	"orquesta/db"
)

type Store interface {
	FindEquivalentSkill(s *db.Skill) (*db.Skill, error)
	CreateSkill(actor string, s *db.Skill) (int64, error)
	Audit(actor, accion, entidad string, entidadID int64, detalle string)
}

type Fetcher interface {
	Fetch(ctx context.Context, spec SourceSpec) (*RemoteSkill, error)
}

type Service struct {
	store   Store
	fetcher Fetcher
}

func NewService(store Store, fetcher Fetcher) *Service {
	return &Service{store: store, fetcher: fetcher}
}

type SourceSpec struct {
	URL   string
	Repo  string
	Skill string
}

type ImportInput struct {
	Actor      string
	TipoAgente string
	URL        string
	Repo       string
	Skill      string
}

type ImportResult struct {
	ID             int64     `json:"id"`
	Existente      bool      `json:"existente"`
	FuenteCanonica string    `json:"fuente_canonica"`
	Skill          *db.Skill `json:"skill"`
}

func (s *Service) ImportFromWeb(ctx context.Context, input ImportInput) (*ImportResult, error) {
	if s == nil || s.store == nil || s.fetcher == nil {
		return nil, fmt.Errorf("servicio de importacion de skills no inicializado")
	}
	actor := strings.TrimSpace(input.Actor)
	if actor == "" {
		return nil, fmt.Errorf("actor obligatorio")
	}
	tipoAgente := strings.TrimSpace(input.TipoAgente)
	if tipoAgente == "" {
		return nil, fmt.Errorf("tipo_agente obligatorio")
	}

	meta, err := s.fetcher.Fetch(ctx, SourceSpec{
		URL:   strings.TrimSpace(input.URL),
		Repo:  strings.TrimSpace(input.Repo),
		Skill: strings.TrimSpace(input.Skill),
	})
	if err != nil {
		return nil, err
	}

	skill := &db.Skill{
		TipoAgente:       tipoAgente,
		Nombre:           strings.TrimSpace(meta.Name),
		Descripcion:      strings.TrimSpace(meta.Description),
		CuandoUsar:       strings.TrimSpace(meta.Description),
		Escenario:        strings.TrimSpace(meta.SourceKind),
		AliasesJSON:      aliasesImportados(meta),
		HerramientasJSON: "[]",
		Origen:           "third_party",
		NivelRiesgo:      "medio",
		Activa:           false,
	}
	if existente, err := s.store.FindEquivalentSkill(skill); err != nil {
		return nil, err
	} else if existente != nil {
		s.store.Audit(actor, "importar_skill_web_existente", "skill", existente.ID, meta.CanonicalURL)
		return &ImportResult{
			ID:             existente.ID,
			Existente:      true,
			FuenteCanonica: meta.CanonicalURL,
			Skill:          existente,
		}, nil
	}

	id, err := s.store.CreateSkill(actor, skill)
	if err != nil {
		return nil, err
	}
	skill.ID = id
	s.store.Audit(actor, "importar_skill_web", "skill", id, meta.CanonicalURL)
	return &ImportResult{
		ID:             id,
		Existente:      false,
		FuenteCanonica: meta.CanonicalURL,
		Skill:          skill,
	}, nil
}

func aliasesImportados(meta *RemoteSkill) string {
	if meta == nil {
		return "[]"
	}
	aliases := []string{}
	if skill := strings.TrimSpace(meta.SkillRef); skill != "" && !strings.EqualFold(skill, meta.Name) {
		aliases = append(aliases, skill)
	}
	if repo := strings.TrimSpace(meta.Repo); repo != "" {
		aliases = append(aliases, repo+"/"+strings.TrimSpace(meta.SkillRef))
	}
	out, err := db.NormalizarListaJSONPublic(aliases)
	if err != nil {
		return "[]"
	}
	return out
}

type Repository struct{}

func (Repository) FindEquivalentSkill(s *db.Skill) (*db.Skill, error) {
	return db.BuscarSkillEquivalente(s, 0)
}

func (Repository) CreateSkill(actor string, s *db.Skill) (int64, error) {
	return db.CrearSkill(actor, s)
}

func (Repository) Audit(actor, accion, entidad string, entidadID int64, detalle string) {
	db.Audit(actor, accion, entidad, entidadID, detalle)
}
