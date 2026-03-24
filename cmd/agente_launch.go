/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type agenteLanzarPlanOutput struct {
	ScriptPath string            `json:"script_path"`
	Args       []string          `json:"args"`
	Env        map[string]string `json:"env,omitempty"`
	Backend    string            `json:"backend"`
	LaunchMode string            `json:"launch_mode"`
	PlanFile   string            `json:"plan_file"`
	Ejecutar   bool              `json:"ejecutar"`
}

var agenteLanzarPlanCmd = &cobra.Command{
	Use:   "lanzar-plan <fichero.plan>",
	Short: "Lanza un plan de agentes a través del adaptador terminal genérico",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend, _ := cmd.Flags().GetString("backend")
		launcher, _ := cmd.Flags().GetString("launcher")
		wrapper, _ := cmd.Flags().GetString("wrapper")
		tmuxSession, _ := cmd.Flags().GetString("tmux-session")
		tmuxAttach, _ := cmd.Flags().GetBool("tmux-attach")
		ejecutar, _ := cmd.Flags().GetBool("ejecutar")
		jsonOut, _ := cmd.Flags().GetBool("json")

		launchMode, err := resolveAgenteLaunchMode(cmd)
		if err != nil {
			return err
		}
		spec, err := buildAgenteLanzarPlanSpec(args[0], backend, launcher, wrapper, tmuxSession, tmuxAttach, launchMode, ejecutar)
		if err != nil {
			return err
		}
		if jsonOut {
			data, err := json.MarshalIndent(spec, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		proc := exec.Command(spec.ScriptPath, spec.Args...)
		proc.Env = mergeAgenteLaunchEnv(os.Environ(), spec.Env)
		proc.Stdin = os.Stdin
		proc.Stdout = os.Stdout
		proc.Stderr = os.Stderr
		return proc.Run()
	},
}

func init() {
	agenteLanzarPlanCmd.Flags().String("backend", "", "Backend terminal: terminator, tmux o custom")
	agenteLanzarPlanCmd.Flags().String("launcher", "", "Launcher custom a usar cuando backend=custom")
	agenteLanzarPlanCmd.Flags().String("wrapper", "", "Wrapper para envolver la llamada al backend terminal")
	agenteLanzarPlanCmd.Flags().String("tmux-session", "", "Nombre base de sesión tmux")
	agenteLanzarPlanCmd.Flags().Bool("tmux-attach", true, "Adjunta automáticamente a tmux cuando backend=tmux y modo=tabs")
	agenteLanzarPlanCmd.Flags().Bool("ejecutar", false, "Aplica el plan y lanza los agentes")
	agenteLanzarPlanCmd.Flags().Bool("tabs", false, "Lanza en pestañas cuando el backend lo soporte")
	agenteLanzarPlanCmd.Flags().Bool("ventanas", false, "Lanza en ventanas/sesiones separadas")
	agenteLanzarPlanCmd.Flags().Bool("json", false, "Imprime la especificación del lanzamiento sin ejecutarla")

	agenteCmd.AddCommand(agenteLanzarPlanCmd)
}

func resolveAgenteLaunchMode(cmd *cobra.Command) (string, error) {
	tabs, _ := cmd.Flags().GetBool("tabs")
	ventanas, _ := cmd.Flags().GetBool("ventanas")
	if tabs && ventanas {
		return "", fmt.Errorf("usa solo uno de --tabs o --ventanas")
	}
	if tabs {
		return "tabs", nil
	}
	return "windows", nil
}

func buildAgenteLanzarPlanSpec(planFile, backend, launcher, wrapper, tmuxSession string, tmuxAttach bool, launchMode string, ejecutar bool) (*agenteLanzarPlanOutput, error) {
	scriptPath, err := resolveOrquestaScriptPath("launch_agentes.sh")
	if err != nil {
		return nil, err
	}
	planAbs, err := filepath.Abs(strings.TrimSpace(planFile))
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(planAbs); err != nil {
		return nil, fmt.Errorf("plan no encontrado: %s", planAbs)
	}

	args := make([]string, 0, 4)
	if ejecutar {
		args = append(args, "--ejecutar")
	}
	switch strings.TrimSpace(launchMode) {
	case "tabs":
		args = append(args, "--tabs")
	default:
		args = append(args, "--ventanas")
	}
	args = append(args, planAbs)

	env := map[string]string{}
	if strings.TrimSpace(backend) != "" {
		env["ORQUESTA_TERMINAL_BACKEND"] = strings.TrimSpace(backend)
	}
	if strings.TrimSpace(launcher) != "" {
		launcherAbs, err := filepath.Abs(strings.TrimSpace(launcher))
		if err != nil {
			return nil, err
		}
		env["ORQUESTA_TERMINAL_LAUNCHER"] = launcherAbs
	}
	if strings.TrimSpace(wrapper) != "" {
		env["ORQUESTA_TERMINAL_WRAPPER"] = strings.TrimSpace(wrapper)
	}
	if strings.TrimSpace(tmuxSession) != "" {
		env["ORQUESTA_TMUX_SESSION_NAME"] = strings.TrimSpace(tmuxSession)
	}
	if !tmuxAttach {
		env["ORQUESTA_TMUX_ATTACH"] = "0"
	}

	return &agenteLanzarPlanOutput{
		ScriptPath: scriptPath,
		Args:       args,
		Env:        env,
		Backend:    strings.TrimSpace(backend),
		LaunchMode: launchMode,
		PlanFile:   planAbs,
		Ejecutar:   ejecutar,
	}, nil
}

func resolveOrquestaScriptPath(scriptName string) (string, error) {
	scriptName = strings.TrimSpace(scriptName)
	if scriptName == "" {
		return "", fmt.Errorf("script vacío")
	}
	var candidates []string
	if dir := strings.TrimSpace(os.Getenv("ORQUESTA_SCRIPTS_DIR")); dir != "" {
		candidates = append(candidates, filepath.Join(dir, scriptName))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "scripts", scriptName))
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "scripts", scriptName),
			filepath.Join(exeDir, "..", "scripts", scriptName),
		)
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no encuentro scripts/%s; usa ORQUESTA_SCRIPTS_DIR para indicarlo", scriptName)
}

func mergeAgenteLaunchEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return append([]string(nil), base...)
	}
	envMap := make(map[string]string, len(base)+len(extra))
	order := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, exists := envMap[key]; !exists {
			order = append(order, key)
		}
		envMap[key] = value
	}
	for key, value := range extra {
		if _, exists := envMap[key]; !exists {
			order = append(order, key)
		}
		envMap[key] = value
	}
	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, key+"="+envMap[key])
	}
	return out
}
