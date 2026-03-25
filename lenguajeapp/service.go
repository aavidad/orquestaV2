package lenguajeapp

import (
	"strings"

	"orquesta/db"
)

type Store interface {
	GetLanguagePolicy() (*db.LanguagePolicy, error)
	SetLanguagePolicy(p *db.LanguagePolicy, updatedBy string) error
	ListLanguageMatrixEntries() ([]*db.LanguageMatrixEntry, error)
	SetLanguageMatrixEntry(kind, selector, contexto, language, reason, updatedBy string) (*db.LanguageMatrixEntry, error)
	DeleteLanguageMatrixEntry(kind, selector, contexto string) error
	ResolveLanguage(project string, taskID *int64, contexto string) (*db.LanguageResolution, error)
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

func (s *Service) GetPolicy() (*db.LanguagePolicy, error) {
	return s.store.GetLanguagePolicy()
}

func (s *Service) SetPolicy(policy *db.LanguagePolicy, updatedBy string) error {
	return s.store.SetLanguagePolicy(policy, strings.TrimSpace(updatedBy))
}

func (s *Service) ListMatrixEntries() ([]*db.LanguageMatrixEntry, error) {
	return s.store.ListLanguageMatrixEntries()
}

func (s *Service) Resolve(input ResolveInput) (*db.LanguageResolution, error) {
	return s.store.ResolveLanguage(strings.TrimSpace(input.Proyecto), input.TareaID, strings.TrimSpace(input.Contexto))
}

func (s *Service) SetMatrixEntry(input SetMatrixEntryInput) (*db.LanguageMatrixEntry, error) {
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

type Repository struct{}

func (Repository) GetLanguagePolicy() (*db.LanguagePolicy, error) {
	return db.GetLanguagePolicy()
}

func (Repository) SetLanguagePolicy(p *db.LanguagePolicy, updatedBy string) error {
	return db.SetLanguagePolicy(p, updatedBy)
}

func (Repository) ListLanguageMatrixEntries() ([]*db.LanguageMatrixEntry, error) {
	return db.ListLanguageMatrixEntries()
}

func (Repository) SetLanguageMatrixEntry(kind, selector, contexto, language, reason, updatedBy string) (*db.LanguageMatrixEntry, error) {
	return db.SetLanguageMatrixEntry(kind, selector, contexto, language, reason, updatedBy)
}

func (Repository) DeleteLanguageMatrixEntry(kind, selector, contexto string) error {
	return db.DeleteLanguageMatrixEntry(kind, selector, contexto)
}

func (Repository) ResolveLanguage(project string, taskID *int64, contexto string) (*db.LanguageResolution, error) {
	return db.ResolveLanguage(project, taskID, contexto)
}
