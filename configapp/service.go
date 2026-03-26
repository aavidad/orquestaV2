package configapp

import (
	"strings"

	"orquesta/db"
)

type Store interface {
	Get(key string) (string, error)
	All() (map[string]string, error)
	Set(key, value string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Get(key string) (string, error) {
	return s.store.Get(strings.TrimSpace(key))
}

func (s *Service) All() (map[string]string, error) {
	return s.store.All()
}

func (s *Service) Set(key, value string) error {
	return s.store.Set(strings.TrimSpace(key), value)
}

type Repository struct{}

func (Repository) Get(key string) (string, error) {
	return db.ConfigGet(key)
}

func (Repository) All() (map[string]string, error) {
	return db.ConfigAll()
}

func (Repository) Set(key, value string) error {
	return db.ConfigSet(key, value)
}
