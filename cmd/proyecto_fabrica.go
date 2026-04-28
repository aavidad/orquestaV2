package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/fabricaapp"
)

type projectAppFactory interface {
	Generate(spec fabricaapp.AppSpec) (fabricaapp.GenerationResult, error)
	GenerateLanguageExpansion(nombre string, idiomas []string) (fabricaapp.GenerationResult, error)
}

var newProjectAppFactory = func() projectAppFactory {
	return fabricaapp.NewService()
}

var proyectoFabricarAppCmd = &cobra.Command{
	Use:   "fabricar-app <slug|id>",
	Short: "Genera backlog base de una app completa para un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nombre, _ := cmd.Flags().GetString("nombre")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		tipo, _ := cmd.Flags().GetString("tipo")
		frontend, _ := cmd.Flags().GetBool("frontend")
		apiEnabled, _ := cmd.Flags().GetBool("api")
		auth, _ := cmd.Flags().GetBool("auth")
		database, _ := cmd.Flags().GetBool("db")
		docker, _ := cmd.Flags().GetBool("docker")
		i18nEnabled, _ := cmd.Flags().GetBool("i18n")
		idiomasRaw, _ := cmd.Flags().GetString("idiomas")
		actor, _ := cmd.Flags().GetString("por")
		platWeb, _ := cmd.Flags().GetBool("plat-web")
		platDesktop, _ := cmd.Flags().GetBool("plat-desktop")
		platMobile, _ := cmd.Flags().GetBool("plat-mobile")
		platCLI, _ := cmd.Flags().GetBool("plat-cli")
		platEmbedded, _ := cmd.Flags().GetBool("plat-embedded")
		soLinux, _ := cmd.Flags().GetBool("so-linux")
		soWindows, _ := cmd.Flags().GetBool("so-windows")
		soMacOS, _ := cmd.Flags().GetBool("so-macos")
		soAndroid, _ := cmd.Flags().GetBool("so-android")
		soIOS, _ := cmd.Flags().GetBool("so-ios")
		cRGPD, _ := cmd.Flags().GetBool("compliance-rgpd")
		cENS, _ := cmd.Flags().GetBool("compliance-ens")
		cLSSI, _ := cmd.Flags().GetBool("compliance-lssi")
		cWCAG, _ := cmd.Flags().GetBool("compliance-wcag")
		cFactura, _ := cmd.Flags().GetBool("compliance-factura-elec")
		cReutil, _ := cmd.Flags().GetBool("compliance-reutilizacion")
		ci, _ := cmd.Flags().GetBool("ci")
		kubernetes, _ := cmd.Flags().GetBool("kubernetes")
		terraform, _ := cmd.Flags().GetBool("terraform")
		monitoring, _ := cmd.Flags().GetBool("monitoring")
		if strings.TrimSpace(tipo) == "" {
			return fmt.Errorf("--tipo es obligatorio")
		}

		if resp, ok, err := fabricarAppProyectoPorAPI(args[0], apiProyectoFabricarAppRequest{
			Nombre:      strings.TrimSpace(nombre),
			Descripcion: strings.TrimSpace(descripcion),
			Tipo:        strings.TrimSpace(tipo),
			Frontend:    frontend,
			API:         apiEnabled,
			Auth:        auth,
			Database:    database,
			Docker:      docker,
			I18n:        i18nEnabled,
			Idiomas:     splitCSV(idiomasRaw),
			Por:         strings.TrimSpace(actor),

			PlatWeb:      platWeb,
			PlatDesktop:  platDesktop,
			PlatMobile:   platMobile,
			PlatCLI:      platCLI,
			PlatEmbedded: platEmbedded,

			SOLinux:   soLinux,
			SOWindows: soWindows,
			SOmacOS:   soMacOS,
			SOAndroid: soAndroid,
			SOiOS:     soIOS,

			ComplianceRGPD:          cRGPD,
			ComplianceENS:           cENS,
			ComplianceLSSI:          cLSSI,
			ComplianceWCAG:          cWCAG,
			ComplianceFacturaElec:   cFactura,
			ComplianceReutilizacion: cReutil,

			CI:         ci,
			Kubernetes: kubernetes,
			Terraform:  terraform,
			Monitoring: monitoring,
		}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Backlog de app generado para %s (%s)\n", resp.Slug, resp.Tipo)
			fmt.Printf("  Tareas creadas: %d\n", resp.Created)
			fmt.Printf("  En backlog:     %d\n", resp.Backlog)
			fmt.Printf("  Primeras libres: %d\n", resp.Created-resp.Backlog)
			return nil
		}
		return serverFirstCommandError("proyecto fabricar-app")
	},
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func init() {
	proyectoFabricarAppCmd.Flags().String("tipo", "", "Tipo de app: web, api, web_api, cli, desktop, mobile, embedded")
	proyectoFabricarAppCmd.Flags().String("nombre", "", "Nombre funcional de la app")
	proyectoFabricarAppCmd.Flags().String("descripcion", "", "Resumen funcional para construir el backlog")
	proyectoFabricarAppCmd.Flags().Bool("frontend", false, "Forzar tarea de frontend")
	proyectoFabricarAppCmd.Flags().Bool("api", false, "Forzar tarea de API")
	proyectoFabricarAppCmd.Flags().Bool("auth", false, "Incluir autenticacion")
	proyectoFabricarAppCmd.Flags().Bool("db", false, "Incluir persistencia/base de datos")
	proyectoFabricarAppCmd.Flags().Bool("docker", false, "Incluir dockerizacion/despliegue")
	proyectoFabricarAppCmd.Flags().Bool("i18n", true, "Incluir i18n y paquete inicial de idiomas")
	proyectoFabricarAppCmd.Flags().String("idiomas", "es,en", "Lista CSV de idiomas iniciales")
	proyectoFabricarAppCmd.Flags().String("por", "alberto", "Actor que genera el backlog")
	// Plataformas
	proyectoFabricarAppCmd.Flags().Bool("plat-web", false, "Plataforma: web (navegador)")
	proyectoFabricarAppCmd.Flags().Bool("plat-desktop", false, "Plataforma: escritorio (Electron, Tauri, Qt…)")
	proyectoFabricarAppCmd.Flags().Bool("plat-mobile", false, "Plataforma: móvil (Android/iOS)")
	proyectoFabricarAppCmd.Flags().Bool("plat-cli", false, "Plataforma: línea de comandos")
	proyectoFabricarAppCmd.Flags().Bool("plat-embedded", false, "Plataforma: embebido / IoT")
	// Sistemas operativos
	proyectoFabricarAppCmd.Flags().Bool("so-linux", false, "SO objetivo: Linux")
	proyectoFabricarAppCmd.Flags().Bool("so-windows", false, "SO objetivo: Windows")
	proyectoFabricarAppCmd.Flags().Bool("so-macos", false, "SO objetivo: macOS")
	proyectoFabricarAppCmd.Flags().Bool("so-android", false, "SO objetivo: Android")
	proyectoFabricarAppCmd.Flags().Bool("so-ios", false, "SO objetivo: iOS")
	// Compliance
	proyectoFabricarAppCmd.Flags().Bool("compliance-rgpd", false, "Cumplimiento RGPD (UE 2016/679)")
	proyectoFabricarAppCmd.Flags().Bool("compliance-ens", false, "Esquema Nacional de Seguridad (RD 311/2022)")
	proyectoFabricarAppCmd.Flags().Bool("compliance-lssi", false, "Ley de Servicios de la Sociedad de la Información")
	proyectoFabricarAppCmd.Flags().Bool("compliance-wcag", false, "Accesibilidad WCAG 2.1 AA (RD 1112/2018)")
	proyectoFabricarAppCmd.Flags().Bool("compliance-factura-elec", false, "Facturación electrónica (Ley Crea y Crece)")
	proyectoFabricarAppCmd.Flags().Bool("compliance-reutilizacion", false, "Reutilización información pública (Ley 37/2007)")
	// Infraestructura
	proyectoFabricarAppCmd.Flags().Bool("ci", false, "Pipeline CI/CD")
	proyectoFabricarAppCmd.Flags().Bool("kubernetes", false, "Orquestación Kubernetes")
	proyectoFabricarAppCmd.Flags().Bool("terraform", false, "Infraestructura como código (Terraform)")
	proyectoFabricarAppCmd.Flags().Bool("monitoring", false, "Observabilidad (métricas, trazas, logs)")
	proyectoCmd.AddCommand(proyectoFabricarAppCmd)
}
