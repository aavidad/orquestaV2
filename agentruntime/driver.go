/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package agentruntime

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ConnectorConfig struct {
	Slug         string
	Nombre       string
	Transporte   string
	Comando      string
	ArgsJSON     string
	EnvJSON      string
	MetadataJSON string
	Activo       bool
}

type ResumeContext struct {
	ExternalSessionID  string
	ResumePayloadJSON  string
	ResumenContinuidad string
	Branch             string
	CWD                string
}

type LaunchRequest struct {
	Agente       string
	Rol          string
	ProyectoSlug string
	ProyectoRuta string
	Modelo       string
	Razonamiento string
	PerfilTarea  string
	Conector     ConnectorConfig
	Resume       ResumeContext
}

type LaunchPlan struct {
	Driver           string            `json:"driver"`
	Transporte       string            `json:"transporte"`
	Modo             string            `json:"modo"`
	Comando          string            `json:"comando"`
	Args             []string          `json:"args"`
	Env              map[string]string `json:"env"`
	WorkingDir       string            `json:"working_dir"`
	Modelo           string            `json:"modelo,omitempty"`
	Razonamiento     string            `json:"razonamiento,omitempty"`
	PerfilTarea      string            `json:"perfil_tarea,omitempty"`
	NativeResume     bool              `json:"native_resume"`
	ContinuityPrompt string            `json:"continuity_prompt,omitempty"`
	Notas            []string          `json:"notas,omitempty"`
}

type Driver interface {
	Transport() string
	Prepare(req LaunchRequest) (*LaunchPlan, error)
}

type Registry struct {
	drivers map[string]Driver
}

func NewRegistry(drivers ...Driver) *Registry {
	r := &Registry{drivers: make(map[string]Driver, len(drivers))}
	for _, driver := range drivers {
		if driver == nil {
			continue
		}
		r.drivers[driver.Transport()] = driver
	}
	return r
}

func DefaultRegistry() *Registry {
	return NewRegistry(
		commandDriver{transport: "cli"},
		commandDriver{transport: "mcp_stdio"},
		remoteDriver{transport: "mcp_http"},
		remoteDriver{transport: "api"},
		remoteDriver{transport: "otro"},
	)
}

func (r *Registry) Prepare(req LaunchRequest) (*LaunchPlan, error) {
	if strings.TrimSpace(req.Conector.Transporte) == "" {
		return nil, fmt.Errorf("el conector '%s' no define transporte", req.Conector.Slug)
	}
	driver, ok := r.drivers[req.Conector.Transporte]
	if !ok {
		return nil, fmt.Errorf("no existe driver para el transporte '%s'", req.Conector.Transporte)
	}
	return driver.Prepare(req)
}

type commandDriver struct {
	transport string
}

func (d commandDriver) Transport() string {
	return d.transport
}

func (d commandDriver) Prepare(req LaunchRequest) (*LaunchPlan, error) {
	if strings.TrimSpace(req.Conector.Comando) == "" {
		return nil, fmt.Errorf("el conector '%s' no define comando", req.Conector.Slug)
	}

	args, err := parseStringSliceJSON(req.Conector.ArgsJSON)
	if err != nil {
		return nil, fmt.Errorf("args_json inválido para '%s': %w", req.Conector.Slug, err)
	}
	env, err := parseStringMapJSON(req.Conector.EnvJSON)
	if err != nil {
		return nil, fmt.Errorf("env_json inválido para '%s': %w", req.Conector.Slug, err)
	}
	metadata, err := parseAnyMapJSON(req.Conector.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("metadata_json inválido para '%s': %w", req.Conector.Slug, err)
	}

	workingDir := strings.TrimSpace(req.Resume.CWD)
	if workingDir == "" {
		workingDir = strings.TrimSpace(req.ProyectoRuta)
	}
	plan := &LaunchPlan{
		Driver:       d.transport,
		Transporte:   d.transport,
		Modo:         "launch",
		Comando:      req.Conector.Comando,
		Args:         append([]string(nil), args...),
		Env:          env,
		WorkingDir:   workingDir,
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		PerfilTarea:  strings.TrimSpace(req.PerfilTarea),
	}
	applyExecutionProfile(plan, metadata)

	if applyLaunchHints(plan, metadata, req) {
		plan.Notas = append(plan.Notas, "El conector aporta hints de arranque desde metadata_json.")
	}

	if tieneContextoReanudable(req.Resume) {
		plan.Modo = "resume"
		plan.NativeResume = applyResumeHints(plan, metadata, req.Resume)
		if !plan.NativeResume {
			plan.ContinuityPrompt = construirPromptContinuidad(req)
			plan.Notas = append(plan.Notas, "El conector no declara estrategia nativa de reanudación; se devuelve prompt de continuidad.")
		}
	}

	return plan, nil
}

type remoteDriver struct {
	transport string
}

func (d remoteDriver) Transport() string {
	return d.transport
}

func (d remoteDriver) Prepare(req LaunchRequest) (*LaunchPlan, error) {
	metadata, err := parseAnyMapJSON(req.Conector.MetadataJSON)
	if err != nil {
		return nil, fmt.Errorf("metadata_json inválido para '%s': %w", req.Conector.Slug, err)
	}
	env, err := parseStringMapJSON(req.Conector.EnvJSON)
	if err != nil {
		return nil, fmt.Errorf("env_json inválido para '%s': %w", req.Conector.Slug, err)
	}

	plan := &LaunchPlan{
		Driver:       d.transport,
		Transporte:   d.transport,
		Modo:         "launch",
		Comando:      req.Conector.Comando,
		Env:          env,
		WorkingDir:   strings.TrimSpace(req.ProyectoRuta),
		Modelo:       strings.TrimSpace(req.Modelo),
		Razonamiento: strings.TrimSpace(req.Razonamiento),
		PerfilTarea:  strings.TrimSpace(req.PerfilTarea),
		Notas: []string{
			"El transporte remoto requiere un adaptador externo que consuma este plan.",
		},
	}
	applyExecutionProfile(plan, metadata)

	if endpoint := stringMetadata(metadata, "endpoint"); endpoint != "" {
		plan.Notas = append(plan.Notas, "Endpoint: "+endpoint)
	}
	if tieneContextoReanudable(req.Resume) {
		plan.Modo = "resume"
		plan.ContinuityPrompt = construirPromptContinuidad(req)
	}
	return plan, nil
}

func RenderCommand(plan *LaunchPlan) string {
	if plan == nil {
		return ""
	}
	partes := []string{strings.TrimSpace(plan.Comando)}
	partes = append(partes, plan.Args...)
	return strings.TrimSpace(strings.Join(partes, " "))
}

func parseStringSliceJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseStringMapJSON(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]string{}, nil
	}
	out := make(map[string]string)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseAnyMapJSON(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	out := make(map[string]any)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func tieneContextoReanudable(r ResumeContext) bool {
	return strings.TrimSpace(r.ExternalSessionID) != "" ||
		strings.TrimSpace(r.ResumePayloadJSON) != "" ||
		strings.TrimSpace(r.ResumenContinuidad) != ""
}

func applyLaunchHints(plan *LaunchPlan, metadata map[string]any, req LaunchRequest) bool {
	aplicado := false
	if subcmd := stringMetadata(metadata, "launch_subcommand"); subcmd != "" {
		plan.Args = append([]string{subcmd}, plan.Args...)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "cwd_flag"); flag != "" && strings.TrimSpace(plan.WorkingDir) != "" {
		plan.Args = append(plan.Args, flag, plan.WorkingDir)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "branch_flag"); flag != "" && strings.TrimSpace(req.Resume.Branch) != "" {
		plan.Args = append(plan.Args, flag, req.Resume.Branch)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "model_flag"); flag != "" && strings.TrimSpace(plan.Modelo) != "" {
		plan.Args = append(plan.Args, flag, plan.Modelo)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "reasoning_flag"); flag != "" && strings.TrimSpace(plan.Razonamiento) != "" {
		plan.Args = append(plan.Args, flag, plan.Razonamiento)
		aplicado = true
	}
	return aplicado
}

func applyExecutionProfile(plan *LaunchPlan, metadata map[string]any) {
	if plan == nil {
		return
	}
	if plan.Modelo == "" {
		plan.Modelo = stringMetadata(metadata, "default_model")
	}
	if plan.Razonamiento == "" {
		plan.Razonamiento = stringMetadata(metadata, "default_reasoning_effort")
	}
	if plan.PerfilTarea == "" {
		plan.PerfilTarea = stringMetadata(metadata, "default_task_profile")
	}
}

func applyResumeHints(plan *LaunchPlan, metadata map[string]any, resume ResumeContext) bool {
	aplicado := false
	if subcmd := stringMetadata(metadata, "resume_subcommand"); subcmd != "" {
		plan.Args = append([]string{subcmd}, plan.Args...)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "session_id_flag"); flag != "" && strings.TrimSpace(resume.ExternalSessionID) != "" {
		plan.Args = append(plan.Args, flag, resume.ExternalSessionID)
		aplicado = true
	} else if subcmd := stringMetadata(metadata, "resume_subcommand"); subcmd != "" && strings.TrimSpace(resume.ExternalSessionID) != "" {
		plan.Args = append(plan.Args, resume.ExternalSessionID)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "resume_payload_flag"); flag != "" && strings.TrimSpace(resume.ResumePayloadJSON) != "" {
		plan.Args = append(plan.Args, flag, resume.ResumePayloadJSON)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "branch_flag"); flag != "" && strings.TrimSpace(resume.Branch) != "" {
		plan.Args = append(plan.Args, flag, resume.Branch)
		aplicado = true
	}
	if flag := stringMetadata(metadata, "cwd_flag"); flag != "" && strings.TrimSpace(plan.WorkingDir) != "" {
		plan.Args = append(plan.Args, flag, plan.WorkingDir)
		aplicado = true
	}
	return aplicado
}

func construirPromptContinuidad(req LaunchRequest) string {
	partes := []string{
		fmt.Sprintf("Retoma el trabajo del agente %s", req.Agente),
	}
	if strings.TrimSpace(req.ProyectoSlug) != "" {
		partes = append(partes, "en el proyecto "+req.ProyectoSlug)
	}
	if strings.TrimSpace(req.Resume.Branch) != "" {
		partes = append(partes, "sobre la rama "+req.Resume.Branch)
	}

	var detalle []string
	if strings.TrimSpace(req.Resume.ResumenContinuidad) != "" {
		detalle = append(detalle, "Resumen previo: "+req.Resume.ResumenContinuidad)
	}
	if strings.TrimSpace(req.Resume.ExternalSessionID) != "" {
		detalle = append(detalle, "External session id: "+req.Resume.ExternalSessionID)
	}
	if strings.TrimSpace(req.Resume.ResumePayloadJSON) != "" {
		detalle = append(detalle, "Resume payload JSON: "+req.Resume.ResumePayloadJSON)
	}

	base := strings.Join(partes, ". ")
	if len(detalle) == 0 {
		return base + "."
	}
	return base + ". " + strings.Join(detalle, ". ")
}

func stringMetadata(metadata map[string]any, key string) string {
	v, ok := metadata[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
