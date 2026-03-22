package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

type Backend interface {
	Name() string
	Open(target string) (*sql.DB, error)
	Prepare(*sql.DB) error
	Backup(*sql.DB, string) error
	IsBusy(error) bool
}

var (
	backendRegistry = map[string]Backend{}
	currentBackend  Backend
)

func RegisterBackend(backend Backend) {
	if backend == nil {
		return
	}
	backendRegistry[strings.TrimSpace(strings.ToLower(backend.Name()))] = backend
}

func backendName() string {
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("ORQUESTA_DB_BACKEND"))); v != "" {
		return v
	}
	return "sqlite"
}

func backendTarget(name string) string {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "sqlite":
		return resolverRuta()
	default:
		if v := strings.TrimSpace(os.Getenv("ORQUESTA_DB_DSN")); v != "" {
			return v
		}
		if v := strings.TrimSpace(os.Getenv("ORQUESTA_DB")); v != "" {
			return v
		}
		return ""
	}
}

func resolveBackend() (Backend, string, error) {
	name := backendName()
	backend, ok := backendRegistry[name]
	if !ok {
		return nil, "", fmt.Errorf("backend de persistencia no soportado: %s", name)
	}
	target := backendTarget(name)
	if strings.TrimSpace(target) == "" {
		return nil, "", fmt.Errorf("target/DSN vacío para backend %s", name)
	}
	return backend, target, nil
}

func CurrentBackendName() string {
	if currentBackend == nil {
		return ""
	}
	return currentBackend.Name()
}

func BackupTo(path string) error {
	if DB == nil {
		return fmt.Errorf("la base de datos no está inicializada")
	}
	if currentBackend == nil {
		return fmt.Errorf("backend de persistencia no inicializado")
	}
	return currentBackend.Backup(DB, path)
}
