package orquestaserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ServerBaseURLFileNameV0 = "base_url.txt"

func ServerBaseURLPathV0(config ConfigV0) string {
	config = NormalizeConfigV0(config)
	runtimeDir := strings.TrimSpace(config.RuntimeWorkDir)
	if runtimeDir == "" {
		return ""
	}
	return filepath.Join(runtimeDir, ServerBaseURLFileNameV0)
}

func (runtime *RuntimeV0) publishServingBaseURLV0(addr string) error {
	path := ServerBaseURLPathV0(runtime.config)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("server_base_url_file: mkdir_failed")
	}
	baseURL := serverBaseURLFromAddrV0(addr)
	if baseURL == "" {
		return fmt.Errorf("server_base_url_file: addr_required")
	}
	return writeServerDurableFileV0(path, []byte(baseURL+"\n"), "server_base_url_file")
}

func serverBaseURLFromAddrV0(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	return "http://" + addr
}
