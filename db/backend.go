package db

import (
	"database/sql"
	"fmt"
	"strings"

	"orquesta/storage"
)

type Backend interface {
	Name() string
	Open(storage.Config) (*sql.DB, error)
	Prepare(*sql.DB, storage.Config) error
	Backup(*sql.DB, string) error
	IsBusy(error) bool
}

var (
	backendRegistry = map[string]Backend{}
	currentBackend  Backend
	currentConfig   storage.Config
	currentTarget   string
)

func IsOpen() bool {
	return DB != nil
}

func CurrentDBPath() string {
	if strings.TrimSpace(currentTarget) != "" {
		return currentTarget
	}
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return ""
	}
	return configTarget(cfg)
}

func CurrentStorageDisplayTarget() string {
	if strings.TrimSpace(currentConfig.Driver) != "" {
		return storage.DisplayTarget(currentConfig)
	}
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return ""
	}
	return storage.DisplayTarget(cfg)
}

func CurrentStorageDriver() string {
	if strings.TrimSpace(currentConfig.Driver) != "" {
		return normalizedDriverName(currentConfig.Driver)
	}
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return ""
	}
	return normalizedDriverName(cfg.Driver)
}

func BackupFilenameSuffix() string {
	driver := strings.TrimSpace(CurrentStorageDriver())
	if driver == "" || driver == "sqlite" {
		return "_orquesta.db.bak"
	}
	return "_orquesta." + driver + ".bak"
}

func BackupFilenameGlob() string {
	return "*" + BackupFilenameSuffix()
}

func RegisterBackend(backend Backend) {
	if backend == nil {
		return
	}
	backendRegistry[strings.TrimSpace(strings.ToLower(backend.Name()))] = backend
}

func backendName() string {
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err == nil && cfg.Driver != "" {
		return cfg.Driver
	}
	return "sqlite"
}

func configTarget(cfg storage.Config) string {
	if strings.TrimSpace(cfg.Path) != "" {
		return cfg.Path
	}
	return strings.TrimSpace(cfg.DSN)
}

func resolveBackend() (Backend, storage.Config, error) {
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return nil, storage.Config{}, err
	}
	backend, ok := backendRegistry[cfg.Driver]
	if !ok {
		return nil, storage.Config{}, fmt.Errorf("backend de persistencia no soportado: %s", cfg.Driver)
	}
	if strings.TrimSpace(configTarget(cfg)) == "" {
		return nil, storage.Config{}, fmt.Errorf("target/DSN vacío para backend %s", cfg.Driver)
	}
	return backend, cfg, nil
}

func CurrentBackendName() string {
	if currentBackend == nil {
		return ""
	}
	return currentBackend.Name()
}

func DriverName() string {
	if currentBackend != nil {
		return normalizedDriverName(currentBackend.Name())
	}
	return normalizedDriverName(backendName())
}

func normalizedDriverName(driver string) string {
	return storage.DialectForDriver(driver).Name
}

func PlaceholderStyle() string {
	return storage.DialectForDriver(DriverName()).PlaceholderStyle()
}

func QueryRebindingEnabled() bool {
	return storage.DialectForDriver(DriverName()).RebindParameters()
}

func BackupTo(path string) error {
	if DB == nil {
		return fmt.Errorf("la base de datos no está inicializada")
	}
	if currentBackend == nil {
		return fmt.Errorf("backend de persistencia no inicializado")
	}
	return currentBackend.Backup(DB.DB, path)
}
