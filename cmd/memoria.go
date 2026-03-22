/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var memoriaCmd = &cobra.Command{
	Use:   "memoria",
	Short: "Memoria de proyecto, fuentes, hallazgos y derivas",
}

var memoriaVerCmd = &cobra.Command{
	Use:   "ver <proyecto>",
	Short: "Muestra la memoria viva de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto := strings.TrimSpace(args[0])

		memoria, memoriaErr := db.GetMemoriaProyecto(proyecto)
		if memoriaErr != nil && !errors.Is(memoriaErr, sql.ErrNoRows) {
			return memoriaErr
		}
		fuentes, err := db.ListarFuentesMemoria(proyecto)
		if err != nil {
			return err
		}
		hallazgos, err := db.ListarHallazgosMemoria(proyecto)
		if err != nil {
			return err
		}
		derivas, err := db.ListarDerivasMemoria(proyecto, false)
		if err != nil {
			return err
		}

		fmt.Printf("╔══════════════════════════════════════════════╗\n")
		fmt.Printf("║  MEMORIA — %s\n", proyecto)
		fmt.Printf("╚══════════════════════════════════════════════╝\n")
		if memoria == nil || errors.Is(memoriaErr, sql.ErrNoRows) {
			fmt.Println("Sin memoria resumida registrada todavía.")
		} else {
			fmt.Printf("Actualizado por: %s\n", memoria.ActualizadoPor)
			fmt.Printf("Actualizada:     %s\n", memoria.UpdatedAt.Format("2006-01-02 15:04"))
			if memoria.Resumen != "" {
				fmt.Printf("\nResumen:\n  %s\n", strings.ReplaceAll(memoria.Resumen, "\n", "\n  "))
			}
			if memoria.Contexto != "" {
				fmt.Printf("\nContexto:\n  %s\n", strings.ReplaceAll(memoria.Contexto, "\n", "\n  "))
			}
			if memoria.PreguntasAbiertas != "" {
				fmt.Printf("\nPreguntas abiertas:\n  %s\n", strings.ReplaceAll(memoria.PreguntasAbiertas, "\n", "\n  "))
			}
		}

		fmt.Printf("\nFuentes (%d):\n", len(fuentes))
		for _, f := range fuentes {
			sufijo := ""
			if f.Titulo != "" {
				sufijo = " — " + f.Titulo
			}
			fmt.Printf("  #%d [%s/%s] %s%s\n", f.ID, f.Tipo, f.Confianza, f.Referencia, sufijo)
		}

		fmt.Printf("\nHallazgos (%d):\n", len(hallazgos))
		for _, h := range hallazgos {
			rel := ""
			if h.FuenteID != nil {
				rel = fmt.Sprintf(" fuente=%d", *h.FuenteID)
			}
			fmt.Printf("  #%d [%s/%s/%s]%s %s\n", h.ID, h.Tipo, h.Impacto, h.Confianza, rel, h.Titulo)
		}

		fmt.Printf("\nDerivas (%d):\n", len(derivas))
		for _, d := range derivas {
			rel := ""
			if d.HallazgoID != nil {
				rel = fmt.Sprintf(" hallazgo=%d", *d.HallazgoID)
			}
			fmt.Printf("  #%d [%s/%s/%s]%s %s\n", d.ID, d.Tipo, d.Severidad, d.Estado, rel, d.Descripcion)
		}
		return nil
	},
}

var memoriaGuardarCmd = &cobra.Command{
	Use:   "guardar <proyecto>",
	Short: "Crea o actualiza la memoria resumida de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto := strings.TrimSpace(args[0])
		agente, err := resolverAgenteMemoria(cmd)
		if err != nil {
			return err
		}
		resumen, _ := cmd.Flags().GetString("resumen")
		contexto, _ := cmd.Flags().GetString("contexto")
		preguntas, _ := cmd.Flags().GetString("preguntas")

		existente, err := db.GetMemoriaProyecto(proyecto)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if existente != nil {
			if strings.TrimSpace(resumen) == "" {
				resumen = existente.Resumen
			}
			if strings.TrimSpace(contexto) == "" {
				contexto = existente.Contexto
			}
			if strings.TrimSpace(preguntas) == "" {
				preguntas = existente.PreguntasAbiertas
			}
		}
		if strings.TrimSpace(resumen) == "" && strings.TrimSpace(contexto) == "" && strings.TrimSpace(preguntas) == "" {
			return fmt.Errorf("debe indicar al menos uno de --resumen, --contexto o --preguntas")
		}

		id, err := db.GuardarMemoriaProyecto(&db.MemoriaProyecto{
			Proyecto:          proyecto,
			Resumen:           strings.TrimSpace(resumen),
			Contexto:          strings.TrimSpace(contexto),
			PreguntasAbiertas: strings.TrimSpace(preguntas),
			ActualizadoPor:    agente,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Memoria del proyecto %s guardada (id: %d)\n", proyecto, id)
		return nil
	},
}

var memoriaFuenteCmd = &cobra.Command{
	Use:   "fuente",
	Short: "Gestión de fuentes de memoria",
}

var memoriaFuenteRegistrarCmd = &cobra.Command{
	Use:   "registrar <proyecto>",
	Short: "Registra una fuente para la memoria de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, err := resolverAgenteMemoria(cmd)
		if err != nil {
			return err
		}
		referencia, _ := cmd.Flags().GetString("referencia")
		if strings.TrimSpace(referencia) == "" {
			return fmt.Errorf("--referencia es obligatorio")
		}
		tipo, _ := cmd.Flags().GetString("tipo")
		titulo, _ := cmd.Flags().GetString("titulo")
		url, _ := cmd.Flags().GetString("url")
		confianza, _ := cmd.Flags().GetString("confianza")
		detalle, _ := cmd.Flags().GetString("detalle")

		id, err := db.RegistrarFuenteMemoria(&db.FuenteMemoria{
			Proyecto:      strings.TrimSpace(args[0]),
			Tipo:          strings.TrimSpace(tipo),
			Referencia:    strings.TrimSpace(referencia),
			Titulo:        strings.TrimSpace(titulo),
			URL:           strings.TrimSpace(url),
			Confianza:     strings.TrimSpace(confianza),
			Detalle:       strings.TrimSpace(detalle),
			RegistradoPor: agente,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Fuente registrada en %s (id: %d)\n", strings.TrimSpace(args[0]), id)
		return nil
	},
}

var memoriaFuenteListarCmd = &cobra.Command{
	Use:   "listar <proyecto>",
	Short: "Lista las fuentes registradas de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fuentes, err := db.ListarFuentesMemoria(strings.TrimSpace(args[0]))
		if err != nil {
			return err
		}
		if len(fuentes) == 0 {
			fmt.Println("No hay fuentes registradas.")
			return nil
		}
		for _, f := range fuentes {
			fmt.Printf("#%d [%s/%s] %s", f.ID, f.Tipo, f.Confianza, f.Referencia)
			if f.Titulo != "" {
				fmt.Printf(" — %s", f.Titulo)
			}
			if f.URL != "" {
				fmt.Printf(" <%s>", f.URL)
			}
			fmt.Println()
		}
		return nil
	},
}

var memoriaHallazgoCmd = &cobra.Command{
	Use:   "hallazgo",
	Short: "Gestión de hallazgos de memoria",
}

var memoriaHallazgoRegistrarCmd = &cobra.Command{
	Use:   "registrar <proyecto>",
	Short: "Registra un hallazgo asociado a una memoria de proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, err := resolverAgenteMemoria(cmd)
		if err != nil {
			return err
		}
		titulo, _ := cmd.Flags().GetString("titulo")
		if strings.TrimSpace(titulo) == "" {
			return fmt.Errorf("--titulo es obligatorio")
		}
		fuenteID, err := int64FlagOpt(cmd, "fuente")
		if err != nil {
			return err
		}
		tipo, _ := cmd.Flags().GetString("tipo")
		descripcion, _ := cmd.Flags().GetString("descripcion")
		impacto, _ := cmd.Flags().GetString("impacto")
		confianza, _ := cmd.Flags().GetString("confianza")

		id, err := db.RegistrarHallazgoMemoria(&db.HallazgoMemoria{
			Proyecto:      strings.TrimSpace(args[0]),
			FuenteID:      fuenteID,
			Tipo:          strings.TrimSpace(tipo),
			Titulo:        strings.TrimSpace(titulo),
			Descripcion:   strings.TrimSpace(descripcion),
			Impacto:       strings.TrimSpace(impacto),
			Confianza:     strings.TrimSpace(confianza),
			RegistradoPor: agente,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Hallazgo registrado en %s (id: %d)\n", strings.TrimSpace(args[0]), id)
		return nil
	},
}

var memoriaHallazgoListarCmd = &cobra.Command{
	Use:   "listar <proyecto>",
	Short: "Lista los hallazgos registrados de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hallazgos, err := db.ListarHallazgosMemoria(strings.TrimSpace(args[0]))
		if err != nil {
			return err
		}
		if len(hallazgos) == 0 {
			fmt.Println("No hay hallazgos registrados.")
			return nil
		}
		for _, h := range hallazgos {
			fmt.Printf("#%d [%s/%s/%s] %s", h.ID, h.Tipo, h.Impacto, h.Confianza, h.Titulo)
			if h.FuenteID != nil {
				fmt.Printf(" (fuente=%d)", *h.FuenteID)
			}
			fmt.Println()
		}
		return nil
	},
}

var memoriaDerivaCmd = &cobra.Command{
	Use:   "deriva",
	Short: "Gestión de detecciones de deriva",
}

var memoriaDerivaRegistrarCmd = &cobra.Command{
	Use:   "registrar <proyecto>",
	Short: "Registra una detección de deriva para un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, err := resolverAgenteMemoria(cmd)
		if err != nil {
			return err
		}
		descripcion, _ := cmd.Flags().GetString("descripcion")
		if strings.TrimSpace(descripcion) == "" {
			return fmt.Errorf("--descripcion es obligatorio")
		}
		hallazgoID, err := int64FlagOpt(cmd, "hallazgo")
		if err != nil {
			return err
		}
		tipo, _ := cmd.Flags().GetString("tipo")
		severidad, _ := cmd.Flags().GetString("severidad")
		estado, _ := cmd.Flags().GetString("estado")
		evidencia, _ := cmd.Flags().GetString("evidencia")

		id, err := db.RegistrarDerivaMemoria(&db.DerivaMemoria{
			Proyecto:     strings.TrimSpace(args[0]),
			HallazgoID:   hallazgoID,
			Tipo:         strings.TrimSpace(tipo),
			Severidad:    strings.TrimSpace(severidad),
			Estado:       db.EstadoDeriva(strings.TrimSpace(estado)),
			Descripcion:  strings.TrimSpace(descripcion),
			Evidencia:    strings.TrimSpace(evidencia),
			DetectadaPor: agente,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Deriva registrada en %s (id: %d)\n", strings.TrimSpace(args[0]), id)
		return nil
	},
}

var memoriaDerivaListarCmd = &cobra.Command{
	Use:   "listar <proyecto>",
	Short: "Lista las derivas registradas de un proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		abiertas, _ := cmd.Flags().GetBool("abiertas")
		derivas, err := db.ListarDerivasMemoria(strings.TrimSpace(args[0]), abiertas)
		if err != nil {
			return err
		}
		if len(derivas) == 0 {
			fmt.Println("No hay derivas registradas.")
			return nil
		}
		for _, d := range derivas {
			fmt.Printf("#%d [%s/%s/%s] %s", d.ID, d.Tipo, d.Severidad, d.Estado, d.Descripcion)
			if d.HallazgoID != nil {
				fmt.Printf(" (hallazgo=%d)", *d.HallazgoID)
			}
			fmt.Println()
		}
		return nil
	},
}

var memoriaDerivaResolverCmd = &cobra.Command{
	Use:   "resolver <id>",
	Short: "Marca una deriva como en revisión, resuelta o descartada",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agente, err := resolverAgenteMemoria(cmd)
		if err != nil {
			return err
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		estado, _ := cmd.Flags().GetString("estado")
		resolucion, _ := cmd.Flags().GetString("resolucion")
		if strings.TrimSpace(resolucion) == "" {
			return fmt.Errorf("--resolucion es obligatorio")
		}
		if err := db.ResolverDerivaMemoria(id, agente, strings.TrimSpace(estado), strings.TrimSpace(resolucion)); err != nil {
			return err
		}
		fmt.Printf("✓ Deriva #%d actualizada a %s\n", id, strings.TrimSpace(estado))
		return nil
	},
}

func init() {
	memoriaGuardarCmd.Flags().String("agente", "", "Agente que actualiza la memoria")
	memoriaGuardarCmd.Flags().String("por", "alberto", "Alias de --agente para compatibilidad")
	memoriaGuardarCmd.Flags().String("resumen", "", "Resumen actual del proyecto")
	memoriaGuardarCmd.Flags().String("contexto", "", "Contexto y estado observado")
	memoriaGuardarCmd.Flags().String("preguntas", "", "Preguntas abiertas o incertidumbres")

	memoriaFuenteRegistrarCmd.Flags().String("agente", "", "Agente que registra la fuente")
	memoriaFuenteRegistrarCmd.Flags().String("por", "alberto", "Alias de --agente para compatibilidad")
	memoriaFuenteRegistrarCmd.Flags().String("tipo", "documentacion", "Tipo de fuente")
	memoriaFuenteRegistrarCmd.Flags().String("referencia", "", "Referencia corta obligatoria")
	memoriaFuenteRegistrarCmd.Flags().String("titulo", "", "Título ampliado")
	memoriaFuenteRegistrarCmd.Flags().String("url", "", "URL si aplica")
	memoriaFuenteRegistrarCmd.Flags().String("confianza", "media", "Confianza: alta, media, baja")
	memoriaFuenteRegistrarCmd.Flags().String("detalle", "", "Detalle adicional")

	memoriaHallazgoRegistrarCmd.Flags().String("agente", "", "Agente que registra el hallazgo")
	memoriaHallazgoRegistrarCmd.Flags().String("por", "alberto", "Alias de --agente para compatibilidad")
	memoriaHallazgoRegistrarCmd.Flags().String("tipo", "hecho", "Tipo: hecho, inferencia, riesgo, decision, pregunta")
	memoriaHallazgoRegistrarCmd.Flags().String("titulo", "", "Título corto obligatorio")
	memoriaHallazgoRegistrarCmd.Flags().String("descripcion", "", "Descripción del hallazgo")
	memoriaHallazgoRegistrarCmd.Flags().String("impacto", "medio", "Impacto: alto, medio, bajo")
	memoriaHallazgoRegistrarCmd.Flags().String("confianza", "media", "Confianza: alta, media, baja")
	memoriaHallazgoRegistrarCmd.Flags().Int64("fuente", 0, "ID de fuente asociada")

	memoriaDerivaRegistrarCmd.Flags().String("agente", "", "Agente que detecta la deriva")
	memoriaDerivaRegistrarCmd.Flags().String("por", "alberto", "Alias de --agente para compatibilidad")
	memoriaDerivaRegistrarCmd.Flags().String("tipo", "documental", "Tipo de deriva")
	memoriaDerivaRegistrarCmd.Flags().String("severidad", "media", "Severidad: alta, media, baja")
	memoriaDerivaRegistrarCmd.Flags().String("estado", string(db.DerivaAbierta), "Estado inicial")
	memoriaDerivaRegistrarCmd.Flags().String("descripcion", "", "Descripción obligatoria")
	memoriaDerivaRegistrarCmd.Flags().String("evidencia", "", "Evidencia o traza observada")
	memoriaDerivaRegistrarCmd.Flags().Int64("hallazgo", 0, "ID de hallazgo relacionado")

	memoriaDerivaListarCmd.Flags().Bool("abiertas", false, "Mostrar solo derivas abiertas o en revisión")

	memoriaDerivaResolverCmd.Flags().String("agente", "", "Agente que resuelve la deriva")
	memoriaDerivaResolverCmd.Flags().String("por", "alberto", "Alias de --agente para compatibilidad")
	memoriaDerivaResolverCmd.Flags().String("estado", string(db.DerivaResuelta), "Nuevo estado: en_revision, resuelta, descartada")
	memoriaDerivaResolverCmd.Flags().String("resolucion", "", "Resolución obligatoria")

	memoriaFuenteCmd.AddCommand(memoriaFuenteRegistrarCmd, memoriaFuenteListarCmd)
	memoriaHallazgoCmd.AddCommand(memoriaHallazgoRegistrarCmd, memoriaHallazgoListarCmd)
	memoriaDerivaCmd.AddCommand(memoriaDerivaRegistrarCmd, memoriaDerivaListarCmd, memoriaDerivaResolverCmd)
	memoriaCmd.AddCommand(memoriaVerCmd, memoriaGuardarCmd, memoriaFuenteCmd, memoriaHallazgoCmd, memoriaDerivaCmd)
}

func resolverAgenteMemoria(cmd *cobra.Command) (string, error) {
	agente, err := resolverValorFlag(cmd, "agente", "por")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(agente) == "" {
		return "alberto", nil
	}
	return strings.TrimSpace(agente), nil
}

func int64FlagOpt(cmd *cobra.Command, nombre string) (*int64, error) {
	v, err := cmd.Flags().GetInt64(nombre)
	if err != nil {
		return nil, err
	}
	if v <= 0 {
		return nil, nil
	}
	return &v, nil
}
