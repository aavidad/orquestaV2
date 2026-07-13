package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	orquestaconfig "orquesta/modulos/orquesta-config"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const serverProjectConfigMaxBytesV0 = orquestaconfig.DefaultMaxDocumentBytesV0

// serverProjectConfigLoadV0 is the one-read product consumed by composition.
// Config, its canonical representation and the catalog are produced from the
// same validated candidate, so callers cannot accidentally mix revisions.
type serverProjectConfigLoadV0 struct {
	Config         serverProjectConfigFileV0
	CanonicalBytes []byte
	Revision       string
	Catalog        []orquestaconfig.CatalogEntryV0
}

// Startup retains the complete product load between configuration projection
// and composition. ConfigV0 transports only its immutable Revision; the
// canonical bytes, typed document and catalog remain private and are selected
// by path+revision without cross-talk between concurrent startups.
var serverProjectConfigStartupLoadsV0 = struct {
	sync.RWMutex
	byKey map[string]serverProjectConfigStartupLoadEntryV0
}{byKey: make(map[string]serverProjectConfigStartupLoadEntryV0)}

var serverProjectConfigReadObserverV0 = struct {
	sync.RWMutex
	observe func(string)
}{}

type serverProjectConfigStartupLoadEntryV0 struct {
	load   serverProjectConfigLoadV0
	owners int
}

func serverProjectConfigStartupLoadKeyV0(projectDir, configPath string, revisions ...string) string {
	base := "dir:" + strings.TrimSpace(projectDir)
	if path := strings.TrimSpace(configPath); path != "" {
		base = "path:" + path
	}
	if len(revisions) > 0 && strings.TrimSpace(revisions[0]) != "" {
		return base + "\x00revision:" + strings.TrimSpace(revisions[0])
	}
	return base
}

func rememberServerProjectConfigStartupLoadV0(projectDir, configPath string, load serverProjectConfigLoadV0) {
	load = cloneServerProjectConfigLoadV0(load)
	serverProjectConfigStartupLoadsV0.Lock()
	defer serverProjectConfigStartupLoadsV0.Unlock()
	if projectDir = strings.TrimSpace(projectDir); projectDir != "" {
		retainServerProjectConfigStartupLoadKeyV0(serverProjectConfigStartupLoadKeyV0(projectDir, "", load.Revision), load)
	}
	if configPath = strings.TrimSpace(configPath); configPath != "" {
		retainServerProjectConfigStartupLoadKeyV0(serverProjectConfigStartupLoadKeyV0("", configPath, load.Revision), load)
	}
}

func retainServerProjectConfigStartupLoadKeyV0(key string, load serverProjectConfigLoadV0) {
	entry := serverProjectConfigStartupLoadsV0.byKey[key]
	entry.load = load
	entry.owners++
	serverProjectConfigStartupLoadsV0.byKey[key] = entry
}

func serverProjectConfigStartupLoadV0(projectDir, configPath string, revisions ...string) (serverProjectConfigLoadV0, bool) {
	serverProjectConfigStartupLoadsV0.RLock()
	defer serverProjectConfigStartupLoadsV0.RUnlock()
	for _, base := range []string{
		serverProjectConfigStartupLoadKeyV0("", configPath),
		serverProjectConfigStartupLoadKeyV0(projectDir, ""),
	} {
		if base == "dir:" || base == "path:" {
			continue
		}
		key, ok := serverProjectConfigStartupLoadRevisionKeyV0(base, revisions...)
		if !ok {
			continue
		}
		if entry, exists := serverProjectConfigStartupLoadsV0.byKey[key]; exists {
			return cloneServerProjectConfigLoadV0(entry.load), true
		}
	}
	return serverProjectConfigLoadV0{}, false
}

func serverProjectConfigStartupLoadRevisionKeyV0(base string, revisions ...string) (string, bool) {
	if len(revisions) > 0 && strings.TrimSpace(revisions[0]) != "" {
		return base + "\x00revision:" + strings.TrimSpace(revisions[0]), true
	}
	prefix := base + "\x00revision:"
	match := ""
	for key := range serverProjectConfigStartupLoadsV0.byKey {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if match != "" {
			return "", false
		}
		match = key
	}
	return match, match != ""
}

func forgetServerProjectConfigStartupLoadV0(projectDir, configPath string, revisions ...string) {
	serverProjectConfigStartupLoadsV0.Lock()
	defer serverProjectConfigStartupLoadsV0.Unlock()
	for _, base := range []string{
		serverProjectConfigStartupLoadKeyV0(projectDir, ""),
		serverProjectConfigStartupLoadKeyV0("", configPath),
	} {
		if key, ok := serverProjectConfigStartupLoadRevisionKeyV0(base, revisions...); ok {
			releaseServerProjectConfigStartupLoadKeyV0(key)
		}
	}
}

func releaseServerProjectConfigSnapshotV0(config orquestaserver.ConfigV0) {
	if strings.TrimSpace(config.ProjectConfigRevision) == "" {
		return
	}
	forgetServerProjectConfigStartupLoadV0(
		config.ProjectWorkDir,
		config.ProjectConfigFilePath,
		config.ProjectConfigRevision,
	)
}

func releaseServerProjectConfigStartupLoadKeyV0(key string) {
	entry, ok := serverProjectConfigStartupLoadsV0.byKey[key]
	if !ok {
		return
	}
	entry.owners--
	if entry.owners <= 0 {
		delete(serverProjectConfigStartupLoadsV0.byKey, key)
		return
	}
	serverProjectConfigStartupLoadsV0.byKey[key] = entry
}

func cloneServerProjectConfigLoadV0(load serverProjectConfigLoadV0) serverProjectConfigLoadV0 {
	load.Config = cloneServerProjectConfigFileV0(load.Config)
	load.CanonicalBytes = append([]byte(nil), load.CanonicalBytes...)
	load.Catalog = orquestaconfig.CloneCatalogV0(load.Catalog)
	return load
}

func cloneServerProjectConfigFileV0(config serverProjectConfigFileV0) serverProjectConfigFileV0 {
	return cloneServerProjectConfigValueV0(reflect.ValueOf(config)).Interface().(serverProjectConfigFileV0)
}

func cloneServerProjectConfigValueV0(value reflect.Value) reflect.Value {
	switch value.Kind() {
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.New(value.Type().Elem())
		cloned.Elem().Set(cloneServerProjectConfigValueV0(value.Elem()))
		return cloned
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := cloneServerProjectConfigValueV0(value.Elem())
		out := reflect.New(value.Type()).Elem()
		out.Set(cloned)
		return out
	case reflect.Struct:
		cloned := reflect.New(value.Type()).Elem()
		for index := 0; index < value.NumField(); index++ {
			cloned.Field(index).Set(cloneServerProjectConfigValueV0(value.Field(index)))
		}
		return cloned
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := 0; index < value.Len(); index++ {
			cloned.Index(index).Set(cloneServerProjectConfigValueV0(value.Index(index)))
		}
		return cloned
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.MakeMapWithSize(value.Type(), value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			cloned.SetMapIndex(
				cloneServerProjectConfigValueV0(iterator.Key()),
				cloneServerProjectConfigValueV0(iterator.Value()),
			)
		}
		return cloned
	case reflect.Array:
		cloned := reflect.New(value.Type()).Elem()
		for index := 0; index < value.Len(); index++ {
			cloned.Index(index).Set(cloneServerProjectConfigValueV0(value.Index(index)))
		}
		return cloned
	default:
		return value
	}
}

func validateServerProjectConfigLoadV0(load serverProjectConfigLoadV0) error {
	canonical, err := canonicalServerProjectConfigV0(load.Config)
	if err != nil {
		return err
	}
	if canonical.Revision != load.Revision || !bytes.Equal(canonical.Bytes, load.CanonicalBytes) {
		return errors.New("server project config load: canonical revision mismatch")
	}
	catalog, err := serverProjectConfigCatalogV0()
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(catalog, load.Catalog) {
		return errors.New("server project config load: catalog mismatch")
	}
	return nil
}

// loadServerProjectConfigBytesV0 reads at most maxBytes+1 bytes before JSON
// decoding. This prevents an oversized config from ever being materialized.
func loadServerProjectConfigBytesV0(path string, maxBytes int) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = orquestaconfig.DefaultMaxDocumentBytesV0
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	serverProjectConfigReadObserverV0.RLock()
	observe := serverProjectConfigReadObserverV0.observe
	serverProjectConfigReadObserverV0.RUnlock()
	if observe != nil {
		observe(path)
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBytes {
		return nil, orquestaconfig.ErrDocumentTooLargeV0
	}
	return raw, nil
}

func serverProjectConfigFileNotFoundV0(err error) bool { return errors.Is(err, os.ErrNotExist) }

func loadServerProjectConfigPathV0(path string) (serverProjectConfigFileV0, bool, error) {
	loaded, ok, err := loadServerProjectConfigProductPathV0(path)
	if err != nil || !ok {
		return serverProjectConfigFileV0{}, ok, err
	}
	return loaded.Config, true, nil
}

func loadServerProjectConfigProductPathV0(path string) (serverProjectConfigLoadV0, bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return serverProjectConfigLoadV0{}, false, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return serverProjectConfigLoadV0{}, false, fmt.Errorf("%s: path", configFileInvalidPublicCodeV0)
	}
	raw, err := loadServerProjectConfigBytesV0(abs, serverProjectConfigMaxBytesV0)
	if serverProjectConfigFileNotFoundV0(err) {
		return serverProjectConfigLoadV0{}, false, nil
	}
	if err != nil {
		return serverProjectConfigLoadV0{}, false, fmt.Errorf("%s: read", configFileInvalidPublicCodeV0)
	}
	var candidate serverProjectConfigFileV0
	if err := orquestaconfig.DecodeAndValidateJSONV0(raw, &candidate, serverProjectConfigMaxBytesV0, validateServerProjectConfigCandidateV0); err != nil {
		if strings.Contains(err.Error(), configFileUnsupportedSchemaCodeV0) {
			return serverProjectConfigLoadV0{}, false, fmt.Errorf("%s: %s", configFileUnsupportedSchemaCodeV0, serverProjectConfigSchemaVersionV0)
		}
		return serverProjectConfigLoadV0{}, false, fmt.Errorf("%s: json", configFileInvalidPublicCodeV0)
	}
	canonical, err := canonicalServerProjectConfigV0(candidate)
	if err != nil {
		return serverProjectConfigLoadV0{}, false, fmt.Errorf("%s: canonical", configFileInvalidPublicCodeV0)
	}
	catalog, err := serverProjectConfigCatalogV0()
	if err != nil {
		return serverProjectConfigLoadV0{}, false, fmt.Errorf("%s: catalog", configFileInvalidPublicCodeV0)
	}
	return serverProjectConfigLoadV0{Config: candidate, CanonicalBytes: canonical.Bytes, Revision: canonical.Revision, Catalog: catalog}, true, nil
}

func validateServerProjectConfigCandidateV0(candidate any) error {
	config, ok := candidate.(*serverProjectConfigFileV0)
	if !ok || config == nil {
		return errors.New("invalid project config candidate")
	}
	return validateServerProjectConfigSemanticV0(*config)
}

func canonicalServerProjectConfigV0(config serverProjectConfigFileV0) (orquestaconfig.CanonicalDocumentV0, error) {
	if err := validateServerProjectConfigSemanticV0(config); err != nil {
		return orquestaconfig.CanonicalDocumentV0{}, err
	}
	return orquestaconfig.EncodeCanonicalJSONV0(config)
}
