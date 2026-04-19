package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"orquesta/storage"
)

type Backend interface {
	Name() string
	Open(storage.Config) (*sql.DB, error)
	Prepare(*sql.DB, storage.Config) error
	Backup(*sql.DB, string) error
	Verify(*sql.DB, storage.Config) *InformePersistencia
	IsBusy(error) bool
}

type BackgroundMaintainer interface {
	MaintenanceTick(*sql.DB, storage.Config) (map[string]any, error)
}

var (
	backendRegistry = map[string]Backend{}
	currentBackend  Backend
	currentConfig   storage.Config
	currentTarget   string
	persistenceMu   sync.RWMutex
)

func IsOpen() bool {
	return CurrentHandle() != nil
}

func CurrentHandle() *Handle {
	persistenceMu.RLock()
	defer persistenceMu.RUnlock()
	return DB
}

func currentPersistenceState() (Backend, storage.Config, string) {
	persistenceMu.RLock()
	defer persistenceMu.RUnlock()
	return currentBackend, currentConfig, currentTarget
}

func CurrentPersistenceState() (Backend, storage.Config, string) {
	return currentPersistenceState()
}

func CurrentDBPath() string {
	_, _, target := currentPersistenceState()
	if strings.TrimSpace(target) != "" {
		return target
	}
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return ""
	}
	return configTarget(cfg)
}

func CurrentStorageDisplayTarget() string {
	_, cfg, _ := currentPersistenceState()
	if strings.TrimSpace(cfg.Driver) != "" {
		return storage.DisplayTarget(cfg)
	}
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return ""
	}
	return storage.DisplayTarget(cfg)
}

func CurrentStorageDriver() string {
	_, cfg, _ := currentPersistenceState()
	if strings.TrimSpace(cfg.Driver) != "" {
		return normalizedDriverName(cfg.Driver)
	}
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return ""
	}
	return normalizedDriverName(cfg.Driver)
}

func CurrentBootstrapSchemaEnabled() bool {
	_, cfg, _ := currentPersistenceState()
	if strings.TrimSpace(cfg.Driver) != "" {
		return cfg.BootstrapSchema
	}
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return false
	}
	return cfg.BootstrapSchema
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
	return ""
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
	backend, _, _ := currentPersistenceState()
	if backend == nil {
		return ""
	}
	return backend.Name()
}

func DriverName() string {
	backend, _, _ := currentPersistenceState()
	if backend != nil {
		return normalizedDriverName(backend.Name())
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
	handle := CurrentHandle()
	if handle == nil {
		return fmt.Errorf("la base de datos no está inicializada")
	}
	backend, _, _ := currentPersistenceState()
	if backend == nil {
		return fmt.Errorf("backend de persistencia no inicializado")
	}
	return backend.Backup(handle.DB, path)
}

func VerificarPersistenciaActual() (*InformePersistencia, error) {
	handle := CurrentHandle()
	if handle == nil {
		return nil, fmt.Errorf("la base de datos no está inicializada")
	}
	backend, cfg, _ := currentPersistenceState()
	if backend == nil {
		return nil, fmt.Errorf("backend de persistencia no inicializado")
	}
	informe := backend.Verify(handle.DB, cfg)
	if informe == nil {
		return nil, fmt.Errorf("el backend de persistencia no ha devuelto informe de verificación")
	}
	if strings.TrimSpace(informe.Driver) == "" {
		informe.Driver = normalizedDriverName(cfg.Driver)
	}
	if strings.TrimSpace(informe.Target) == "" {
		informe.Target = storage.DisplayTarget(cfg)
	}
	return informe, nil
}
