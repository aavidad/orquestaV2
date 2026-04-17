package lenguajeapp

import (
	"strings"
	"time"
)

type LanguagePolicy struct {
	DefaultLanguage          string    `json:"default_language"`
	DocumentationMultilang   bool      `json:"documentation_multilang"`
	AppsMultilang            bool      `json:"apps_multilang"`
	DocumentationDefaultLang string    `json:"documentation_default_language"`
	AppsDefaultLang          string    `json:"apps_default_language"`
	AllowedLanguages         []string  `json:"allowed_languages"`
	Notes                    string    `json:"notes"`
	UpdatedBy                string    `json:"updated_by"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type LanguageMatrixEntry struct {
	Scope     string    `json:"scope"`
	Selector  string    `json:"selector"`
	Context   string    `json:"context"`
	Language  string    `json:"language"`
	Reason    string    `json:"reason"`
	UpdatedBy string    `json:"updated_by"`
	UpdatedAt time.Time `json:"updated_at"`
	ConfigKey string    `json:"-"`
}

type LanguageResolution struct {
	Proyecto string               `json:"proyecto"`
	TareaID  *int64               `json:"tarea_id"`
	Contexto string               `json:"contexto"`
	Idioma   string               `json:"idioma"`
	Origen   string               `json:"origen"`
	Entrada  *LanguageMatrixEntry `json:"entrada"`
	Politica *LanguagePolicy      `json:"politica"`
}

type Store interface {
	GetLanguagePolicy() (*LanguagePolicy, error)
	SetLanguagePolicy(p *LanguagePolicy, updatedBy string) error
	ListLanguageMatrixEntries() ([]*LanguageMatrixEntry, error)
	SetLanguageMatrixEntry(kind, selector, contexto, language, reason, updatedBy string) (*LanguageMatrixEntry, error)
	DeleteLanguageMatrixEntry(kind, selector, contexto string) error
	ResolveLanguage(project string, taskID *int64, contexto string) (*LanguageResolution, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

type ResolveInput struct {
	Proyecto string
	TareaID  *int64
	Contexto string
}

type SetMatrixEntryInput struct {
	Scope     string
	Selector  string
	Contexto  string
	Language  string
	Reason    string
	UpdatedBy string
}

func (s *Service) GetPolicy() (*LanguagePolicy, error) {
	return s.store.GetLanguagePolicy()
}

func (s *Service) SetPolicy(policy *LanguagePolicy, updatedBy string) error {
	return s.store.SetLanguagePolicy(policy, strings.TrimSpace(updatedBy))
}

func (s *Service) ListMatrixEntries() ([]*LanguageMatrixEntry, error) {
	return s.store.ListLanguageMatrixEntries()
}

func (s *Service) Resolve(input ResolveInput) (*LanguageResolution, error) {
	return s.store.ResolveLanguage(strings.TrimSpace(input.Proyecto), input.TareaID, strings.TrimSpace(input.Contexto))
}

func (s *Service) SetMatrixEntry(input SetMatrixEntryInput) (*LanguageMatrixEntry, error) {
	return s.store.SetLanguageMatrixEntry(
		strings.TrimSpace(input.Scope),
		strings.TrimSpace(input.Selector),
		strings.TrimSpace(input.Contexto),
		strings.TrimSpace(input.Language),
		strings.TrimSpace(input.Reason),
		strings.TrimSpace(input.UpdatedBy),
	)
}

func (s *Service) DeleteMatrixEntry(scope, selector, contexto string) error {
	return s.store.DeleteLanguageMatrixEntry(strings.TrimSpace(scope), strings.TrimSpace(selector), strings.TrimSpace(contexto))
}
