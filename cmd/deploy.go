package cmd

import (
	"context"
	"encoding/json"

	"github.com/spf13/cobra"
	"orquesta/deployapp"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Planifica y ejecuta despliegues opcionales",
}

var deployDockerRemoteCmd = &cobra.Command{
	Use:   "docker-remoto",
	Short: "Planifica o ejecuta un despliegue Docker remoto via SSH",
}

var deployDockerRemotePlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Genera el plan de despliegue Docker remoto",
	RunE: func(cmd *cobra.Command, args []string) error {
		spec, err := deployDockerRemoteSpecFromFlags(cmd)
		if err != nil {
			return err
		}
		plan, err := deployapp.NewService().Plan(spec)
		if err != nil {
			return err
		}
		return emitJSON(cmd, apiDeployDockerRemotePlanResponse{Plan: plan})
	},
}

var deployDockerRemoteExecuteCmd = &cobra.Command{
	Use:   "ejecutar",
	Short: "Ejecuta el plan de despliegue Docker remoto",
	RunE: func(cmd *cobra.Command, args []string) error {
		spec, err := deployDockerRemoteSpecFromFlags(cmd)
		if err != nil {
			return err
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		autoRollback, _ := cmd.Flags().GetBool("auto-rollback")
		service := deployapp.NewService()
		plan, err := service.Plan(spec)
		if err != nil {
			return err
		}
		result, err := service.Execute(context.Background(), plan, deployapp.ExecuteOptions{
			DryRun:       dryRun,
			AutoRollback: autoRollback,
		})
		if err != nil {
			return err
		}
		return emitJSON(cmd, apiDeployDockerRemoteExecuteResponse{
			Plan:   plan,
			Result: result,
		})
	},
}

func deployDockerRemoteSpecFromFlags(cmd *cobra.Command) (deployapp.DockerRemoteSpec, error) {
	spec := deployapp.DockerRemoteSpec{}
	spec.ProyectoSlug, _ = cmd.Flags().GetString("proyecto")
	spec.Servicio, _ = cmd.Flags().GetString("servicio")
	spec.SSHHost, _ = cmd.Flags().GetString("ssh-host")
	spec.SSHPort, _ = cmd.Flags().GetInt("ssh-port")
	spec.SSHUser, _ = cmd.Flags().GetString("ssh-user")
	spec.RemoteDir, _ = cmd.Flags().GetString("remote-dir")
	spec.ComposeFile, _ = cmd.Flags().GetString("compose-file")
	spec.EnvFile, _ = cmd.Flags().GetString("env-file")
	spec.Image, _ = cmd.Flags().GetString("image")
	spec.Tag, _ = cmd.Flags().GetString("tag")
	spec.RollbackTag, _ = cmd.Flags().GetString("rollback-tag")
	spec.Dockerfile, _ = cmd.Flags().GetString("dockerfile")
	spec.BuildContext, _ = cmd.Flags().GetString("build-context")
	spec.HealthcheckCmd, _ = cmd.Flags().GetString("healthcheck-cmd")
	spec.Strategy, _ = cmd.Flags().GetString("strategy")
	spec.Build, _ = cmd.Flags().GetBool("build")
	return spec, nil
}

func emitJSON(cmd *cobra.Command, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}

func init() {
	for _, sub := range []*cobra.Command{deployDockerRemotePlanCmd, deployDockerRemoteExecuteCmd} {
		sub.Flags().String("proyecto", "", "Slug del proyecto")
		sub.Flags().String("servicio", "", "Nombre del servicio docker compose")
		sub.Flags().String("ssh-host", "", "Host remoto SSH")
		sub.Flags().Int("ssh-port", 22, "Puerto SSH")
		sub.Flags().String("ssh-user", "", "Usuario SSH")
		sub.Flags().String("remote-dir", "", "Directorio remoto base para releases")
		sub.Flags().String("compose-file", "docker-compose.yml", "Ruta local del docker-compose")
		sub.Flags().String("env-file", ".env", "Ruta local del fichero de secretos/entorno")
		sub.Flags().String("image", "", "Nombre de la imagen Docker")
		sub.Flags().String("tag", "", "Tag de la imagen a desplegar")
		sub.Flags().String("rollback-tag", "", "Tag previo para rollback automatico")
		sub.Flags().String("dockerfile", "Dockerfile", "Ruta al Dockerfile")
		sub.Flags().String("build-context", ".", "Contexto de build Docker")
		sub.Flags().String("healthcheck-cmd", "", "Comando remoto de healthcheck")
		sub.Flags().String("strategy", "docker_save", "Estrategia de entrega: docker_save o registry")
		sub.Flags().Bool("build", true, "Construir imagen antes del despliegue")
	}
	deployDockerRemoteExecuteCmd.Flags().Bool("dry-run", false, "No ejecuta comandos; solo recorre el plan")
	deployDockerRemoteExecuteCmd.Flags().Bool("auto-rollback", true, "Dispara rollback si falla el healthcheck y hay rollback-tag")

	deployDockerRemoteCmd.AddCommand(deployDockerRemotePlanCmd, deployDockerRemoteExecuteCmd)
	deployCmd.AddCommand(deployDockerRemoteCmd)
	rootCmd.AddCommand(deployCmd)
}
