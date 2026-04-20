package rpclocal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"orquesta/storage"
)

const (
	envStatePath = "ORQUESTA_SERVER_STATE"
	envInfoPath  = "ORQUESTA_SERVER_INFO"
	envAddr      = "ORQUESTA_SERVER_ADDR"
)

func DefaultAddr() string {
	if addr := strings.TrimSpace(os.Getenv(envAddr)); addr != "" {
		return BaseURL(addr)
	}
	return "http://" + DefaultHost + ":" + DefaultPort
}

func DefaultStatePath() string {
	if path := strings.TrimSpace(os.Getenv(envInfoPath)); path != "" {
		return path
	}
	if path := strings.TrimSpace(os.Getenv(envStatePath)); path != "" {
		return path
	}
	return filepath.Join(os.TempDir(), scopedStateFile(CurrentScopeID()))
}

func SaveState(path string, state *State) error {
	if state == nil {
		return fmt.Errorf("state obligatorio")
	}
	if strings.TrimSpace(path) == "" {
		path = DefaultStatePath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "orquesta-localrpc-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func LoadState(path string) (*State, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultStatePath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if strings.TrimSpace(state.Addr) == "" {
		return nil, fmt.Errorf("state sin addr")
	}
	if !MatchesCurrentScope(&state) {
		return nil, fmt.Errorf("state fuera del scope actual")
	}
	return &state, nil
}

func RemoveState(path string) error {
	if strings.TrimSpace(path) == "" {
		path = DefaultStatePath()
	}
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func BaseURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = DefaultAddr()
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	return "http://" + addr
}

func DefaultInfoPath() string {
	return DefaultStatePath()
}

func CurrentScopeID() string {
	sum := sha256.Sum256([]byte(currentScopeSource()))
	return hex.EncodeToString(sum[:6])
}

func MatchesCurrentScope(state *State) bool {
	if state == nil {
		return false
	}
	scopeID := strings.TrimSpace(state.ScopeID)
	if scopeID == "" {
		return true
	}
	return scopeID == CurrentScopeID()
}

func ResolveServerAddr() string {
	info, err := LoadServerInfo()
	if err == nil && strings.TrimSpace(info.Addr) != "" {
		addr := strings.TrimSpace(info.Addr)
		if serverAddrReachable(addr) {
			return BaseURL(addr)
		}
	}
	return DefaultAddr()
}

func serverAddrReachable(addr string) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	return Ping(ctx, addr) == nil
}

func currentScopeSource() string {
	if storageExplicitlyConfigured() {
		cfg, err := storage.ResolveConfig(nil)
		if err == nil {
			driver := strings.TrimSpace(cfg.Driver)
			if path := strings.TrimSpace(cfg.Path); path != "" {
				return "db:" + driver + ":" + filepath.Clean(path)
			}
			if dsn := strings.TrimSpace(cfg.DSN); dsn != "" {
				return "db:" + driver + ":" + dsn
			}
		}
	}
	if wd, err := os.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
		return "cwd:" + filepath.Clean(wd)
	}
	return "global"
}

func storageExplicitlyConfigured() bool {
	for _, key := range []string{"ORQUESTA_DB_DRIVER", "ORQUESTA_DB_BACKEND", "ORQUESTA_DB_DSN", "ORQUESTA_DB"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

func scopedStateFile(scopeID string) string {
	ext := filepath.Ext(DefaultStateFile)
	base := strings.TrimSuffix(DefaultStateFile, ext)
	scopeID = strings.TrimSpace(scopeID)
	if scopeID == "" {
		return DefaultStateFile
	}
	return fmt.Sprintf("%s-%s%s", base, scopeID, ext)
}
