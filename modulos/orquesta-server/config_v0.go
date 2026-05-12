package orquestaserver

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

const (
	DefaultAddrV0         = "127.0.0.1:8787"
	DefaultStateFileV0    = "orquesta_server_state_v0.json"
	DefaultTickIntervalV0 = 5 * time.Second
)

type ConfigV0 struct {
	Addr              string
	StateDir          string
	StateFile         string
	ProjectWorkDir    string
	RuntimeWorkDir    string
	TickInterval      time.Duration
	SupervisorCommand orquestarunsupervisor.RunSupervisorCommandV0
}

func NormalizeConfigV0(config ConfigV0) ConfigV0 {
	config.Addr = strings.TrimSpace(config.Addr)
	if config.Addr == "" {
		config.Addr = DefaultAddrV0
	}
	config.StateDir = strings.TrimSpace(config.StateDir)
	config.StateFile = strings.TrimSpace(config.StateFile)
	if config.StateFile == "" {
		config.StateFile = DefaultStateFileV0
	}
	config.ProjectWorkDir = strings.TrimSpace(config.ProjectWorkDir)
	config.RuntimeWorkDir = strings.TrimSpace(config.RuntimeWorkDir)
	if config.TickInterval <= 0 {
		config.TickInterval = DefaultTickIntervalV0
	}
	config.SupervisorCommand.MaxTicks = 1
	return config
}

func ValidateConfigV0(config ConfigV0) error {
	config = NormalizeConfigV0(config)
	if strings.TrimSpace(config.StateDir) == "" || !filepath.IsAbs(config.StateDir) {
		return fmt.Errorf("orquesta_server: state_dir invalido")
	}
	if filepath.Base(config.StateFile) != config.StateFile {
		return fmt.Errorf("orquesta_server: state_file invalido")
	}
	if strings.TrimSpace(config.ProjectWorkDir) != "" && !filepath.IsAbs(config.ProjectWorkDir) {
		return fmt.Errorf("orquesta_server: project_work_dir invalido")
	}
	if strings.TrimSpace(config.RuntimeWorkDir) != "" && !filepath.IsAbs(config.RuntimeWorkDir) {
		return fmt.Errorf("orquesta_server: runtime_work_dir invalido")
	}
	return nil
}

func StatePathV0(config ConfigV0) string {
	config = NormalizeConfigV0(config)
	return filepath.Join(config.StateDir, config.StateFile)
}
