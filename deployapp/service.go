package deployapp

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type DockerRemoteSpec struct {
	ProyectoSlug   string `json:"proyecto_slug"`
	Servicio       string `json:"servicio"`
	SSHHost        string `json:"ssh_host"`
	SSHPort        int    `json:"ssh_port"`
	SSHUser        string `json:"ssh_user"`
	RemoteDir      string `json:"remote_dir"`
	ComposeFile    string `json:"compose_file"`
	EnvFile        string `json:"env_file"`
	Image          string `json:"image"`
	Tag            string `json:"tag"`
	RollbackTag    string `json:"rollback_tag"`
	Dockerfile     string `json:"dockerfile"`
	BuildContext   string `json:"build_context"`
	HealthcheckCmd string `json:"healthcheck_cmd"`
	Strategy       string `json:"strategy"`
	Build          bool   `json:"build"`
}

type Step struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Shell       string `json:"shell"`
	Rollback    bool   `json:"rollback"`
}

type DockerRemotePlan struct {
	Target       string   `json:"target"`
	ReleaseDir   string   `json:"release_dir"`
	ImageRef     string   `json:"image_ref"`
	ArtifactName string   `json:"artifact_name,omitempty"`
	Strategy     string   `json:"strategy"`
	Warnings     []string `json:"warnings,omitempty"`
	Steps        []Step   `json:"steps"`
}

type ExecuteOptions struct {
	DryRun       bool
	AutoRollback bool
}

type ExecuteResult struct {
	Executed          []string `json:"executed"`
	FailedStep        string   `json:"failed_step,omitempty"`
	RollbackTriggered bool     `json:"rollback_triggered"`
}

type ShellRunner interface {
	Run(ctx context.Context, shell string) error
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, shell string) error {
	cmd := exec.CommandContext(ctx, "bash", "-lc", shell)
	return cmd.Run()
}

type Service struct {
	runner ShellRunner
}

func NewService() *Service {
	return &Service{runner: execRunner{}}
}

func NewServiceWithRunner(runner ShellRunner) *Service {
	return &Service{runner: runner}
}

func (s *Service) Plan(spec DockerRemoteSpec) (*DockerRemotePlan, error) {
	norm, err := normalizeSpec(spec)
	if err != nil {
		return nil, err
	}
	imageRef := fmt.Sprintf("%s:%s", norm.Image, norm.Tag)
	releaseDir := path.Join(norm.RemoteDir, "releases", sanitizeTag(norm.Tag))
	target := sshTarget(norm)
	composeBase := path.Base(norm.ComposeFile)

	plan := &DockerRemotePlan{
		Target:     target,
		ReleaseDir: releaseDir,
		ImageRef:   imageRef,
		Strategy:   norm.Strategy,
	}

	switch norm.Strategy {
	case "docker_save":
		artifactName := fmt.Sprintf("%s-%s-image.tar.gz", safeSlug(norm.ProyectoSlug), sanitizeTag(norm.Tag))
		plan.ArtifactName = artifactName
		if norm.Build {
			plan.Steps = append(plan.Steps, Step{
				Name:        "build_image",
				Description: "Construir imagen Docker local",
				Shell:       fmt.Sprintf("docker build -t %s -f %s %s", shellQuote(imageRef), shellQuote(norm.Dockerfile), shellQuote(norm.BuildContext)),
			})
		}
		plan.Steps = append(plan.Steps,
			Step{
				Name:        "prepare_remote_dir",
				Description: "Crear release dir remoto",
				Shell:       shellSSH(norm, fmt.Sprintf("mkdir -p %s", releaseDir)),
			},
			Step{
				Name:        "package_image",
				Description: "Empaquetar imagen Docker para transporte",
				Shell:       fmt.Sprintf("docker save %s | gzip > %s", shellQuote(imageRef), shellQuote(artifactName)),
			},
			Step{
				Name:        "transfer_release",
				Description: "Transferir artefactos y manifiestos al servidor",
				Shell:       fmt.Sprintf("scp -P %d %s %s %s", norm.SSHPort, shellQuote(norm.ComposeFile), shellQuote(norm.EnvFile), shellQuote(fmt.Sprintf("%s:%s/", target, releaseDir))),
			},
			Step{
				Name:        "transfer_image",
				Description: "Transferir imagen al servidor",
				Shell:       fmt.Sprintf("scp -P %d %s %s", norm.SSHPort, shellQuote(artifactName), shellQuote(fmt.Sprintf("%s:%s/", target, releaseDir))),
			},
			Step{
				Name:        "deploy_remote",
				Description: "Cargar imagen y desplegar contenedor en remoto",
				Shell: shellSSH(norm, fmt.Sprintf(
					"cd %s && docker load -i %s && IMAGE_REF=%s docker compose -f %s up -d",
					releaseDir,
					artifactName,
					imageRef,
					composeBase,
				)),
			},
		)
	case "registry":
		if norm.Build {
			plan.Steps = append(plan.Steps,
				Step{
					Name:        "build_image",
					Description: "Construir imagen Docker local",
					Shell:       fmt.Sprintf("docker build -t %s -f %s %s", shellQuote(imageRef), shellQuote(norm.Dockerfile), shellQuote(norm.BuildContext)),
				},
				Step{
					Name:        "push_image",
					Description: "Publicar imagen en el registry",
					Shell:       fmt.Sprintf("docker push %s", shellQuote(imageRef)),
				},
			)
		}
		plan.Steps = append(plan.Steps,
			Step{
				Name:        "prepare_remote_dir",
				Description: "Crear release dir remoto",
				Shell:       shellSSH(norm, fmt.Sprintf("mkdir -p %s", releaseDir)),
			},
			Step{
				Name:        "transfer_release",
				Description: "Transferir compose y env al servidor",
				Shell:       fmt.Sprintf("scp -P %d %s %s %s", norm.SSHPort, shellQuote(norm.ComposeFile), shellQuote(norm.EnvFile), shellQuote(fmt.Sprintf("%s:%s/", target, releaseDir))),
			},
			Step{
				Name:        "deploy_remote",
				Description: "Actualizar imagen y desplegar en remoto",
				Shell: shellSSH(norm, fmt.Sprintf(
					"cd %s && IMAGE_REF=%s docker compose -f %s pull && IMAGE_REF=%s docker compose -f %s up -d",
					releaseDir,
					imageRef,
					composeBase,
					imageRef,
					composeBase,
				)),
			},
		)
	default:
		return nil, fmt.Errorf("estrategia no soportada: %s", norm.Strategy)
	}

	plan.Steps = append(plan.Steps, Step{
		Name:        "healthcheck",
		Description: "Verificar que el despliegue ha quedado sano",
		Shell:       shellSSH(norm, fmt.Sprintf("cd %s && %s", releaseDir, norm.HealthcheckCmd)),
	})
	if strings.TrimSpace(norm.RollbackTag) != "" {
		plan.Steps = append(plan.Steps, Step{
			Name:        "rollback",
			Description: "Volver a la version anterior si falla el healthcheck",
			Shell: shellSSH(norm, fmt.Sprintf(
				"cd %s && IMAGE_REF=%s:%s docker compose -f %s up -d",
				releaseDir,
				norm.Image,
				norm.RollbackTag,
				composeBase,
			)),
			Rollback: true,
		})
	} else {
		plan.Warnings = append(plan.Warnings, "no rollback_tag definido; si falla el healthcheck no habra rollback automatico")
	}
	plan.Warnings = append(plan.Warnings, "el fichero de secretos debe viajar fuera del repositorio y revisarse antes del despliegue")
	if norm.Strategy == "docker_save" {
		plan.Warnings = append(plan.Warnings, "la estrategia docker_save requiere transferir la imagen completa al servidor")
	}
	return plan, nil
}

func (s *Service) Execute(ctx context.Context, plan *DockerRemotePlan, opts ExecuteOptions) (*ExecuteResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan obligatorio")
	}
	if s.runner == nil {
		s.runner = execRunner{}
	}
	result := &ExecuteResult{}
	var rollbackStep *Step
	for i := range plan.Steps {
		if plan.Steps[i].Rollback {
			rollbackStep = &plan.Steps[i]
			break
		}
	}
	for i := range plan.Steps {
		step := plan.Steps[i]
		if step.Rollback {
			rollbackStep = &step
			continue
		}
		result.Executed = append(result.Executed, step.Name)
		if opts.DryRun {
			continue
		}
		if err := s.runner.Run(ctx, step.Shell); err != nil {
			result.FailedStep = step.Name
			if opts.AutoRollback && rollbackStep != nil {
				result.RollbackTriggered = true
				_ = s.runner.Run(ctx, rollbackStep.Shell)
			}
			return result, fmt.Errorf("fallo en paso %s: %w", step.Name, err)
		}
	}
	return result, nil
}

func normalizeSpec(spec DockerRemoteSpec) (DockerRemoteSpec, error) {
	spec.ProyectoSlug = strings.TrimSpace(spec.ProyectoSlug)
	spec.Servicio = strings.TrimSpace(spec.Servicio)
	spec.SSHHost = strings.TrimSpace(spec.SSHHost)
	spec.SSHUser = strings.TrimSpace(spec.SSHUser)
	spec.RemoteDir = strings.TrimSpace(spec.RemoteDir)
	spec.ComposeFile = strings.TrimSpace(spec.ComposeFile)
	spec.EnvFile = strings.TrimSpace(spec.EnvFile)
	spec.Image = strings.TrimSpace(spec.Image)
	spec.Tag = strings.TrimSpace(spec.Tag)
	spec.RollbackTag = strings.TrimSpace(spec.RollbackTag)
	spec.Dockerfile = strings.TrimSpace(spec.Dockerfile)
	spec.BuildContext = strings.TrimSpace(spec.BuildContext)
	spec.HealthcheckCmd = strings.TrimSpace(spec.HealthcheckCmd)
	spec.Strategy = strings.TrimSpace(strings.ToLower(spec.Strategy))
	if spec.SSHPort <= 0 {
		spec.SSHPort = 22
	}
	if spec.RemoteDir == "" {
		spec.RemoteDir = path.Join("/opt/orquesta", safeSlug(spec.ProyectoSlug))
	}
	if spec.ComposeFile == "" {
		spec.ComposeFile = "docker-compose.yml"
	}
	if spec.EnvFile == "" {
		spec.EnvFile = ".env"
	}
	if spec.Dockerfile == "" {
		spec.Dockerfile = "Dockerfile"
	}
	if spec.BuildContext == "" {
		spec.BuildContext = "."
	}
	if spec.HealthcheckCmd == "" {
		spec.HealthcheckCmd = fmt.Sprintf("IMAGE_REF=%s:%s docker compose -f %s ps", spec.Image, spec.Tag, path.Base(spec.ComposeFile))
	}
	if spec.Strategy == "" {
		spec.Strategy = "docker_save"
	}
	if !spec.Build {
		spec.Build = true
	}
	switch {
	case spec.ProyectoSlug == "":
		return spec, fmt.Errorf("proyecto_slug obligatorio")
	case spec.Servicio == "":
		return spec, fmt.Errorf("servicio obligatorio")
	case spec.SSHHost == "":
		return spec, fmt.Errorf("ssh_host obligatorio")
	case spec.SSHUser == "":
		return spec, fmt.Errorf("ssh_user obligatorio")
	case spec.Image == "":
		return spec, fmt.Errorf("image obligatorio")
	case spec.Tag == "":
		return spec, fmt.Errorf("tag obligatorio")
	}
	return spec, nil
}

func sshTarget(spec DockerRemoteSpec) string {
	return fmt.Sprintf("%s@%s", spec.SSHUser, spec.SSHHost)
}

func shellSSH(spec DockerRemoteSpec, remote string) string {
	return fmt.Sprintf("ssh -p %d %s %s", spec.SSHPort, shellQuote(sshTarget(spec)), shellQuote(remote))
}

var tagSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeTag(tag string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "latest"
	}
	return tagSanitizer.ReplaceAllString(tag, "_")
}

func safeSlug(slug string) string {
	slug = strings.TrimSpace(strings.ToLower(slug))
	slug = strings.ReplaceAll(slug, " ", "-")
	if slug == "" {
		return "app"
	}
	return tagSanitizer.ReplaceAllString(slug, "-")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func artifactPath(spec DockerRemoteSpec) string {
	return filepath.Join(".", fmt.Sprintf("%s-%s-image.tar.gz", safeSlug(spec.ProyectoSlug), sanitizeTag(spec.Tag)))
}
