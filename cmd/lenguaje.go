/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var lenguajeCmd = &cobra.Command{
	Use:   "lenguaje",
	Short: "Politica de lenguaje, matriz de seleccion y multilenguaje",
}

var lenguajePoliticaCmd = &cobra.Command{
	Use:   "politica",
	Short: "Gestion de la politica global de lenguaje",
}

var lenguajePoliticaVerCmd = &cobra.Command{
	Use:   "ver",
	Short: "Muestra la politica global actual",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, ok, err := cargarPoliticaLenguajeDesdeAPI()
		if !ok {
			p, err = db.GetLanguagePolicy()
		}
		if err != nil {
			return err
		}
		fmt.Println("POLITICA GLOBAL DE LENGUAJE")
		fmt.Printf("  Idioma por defecto:              %s\n", p.DefaultLanguage)
		fmt.Printf("  Documentacion multilenguaje:     %t\n", p.DocumentationMultilang)
		fmt.Printf("  Apps multilenguaje:              %t\n", p.AppsMultilang)
		fmt.Printf("  Idioma defecto documentacion:    %s\n", p.DocumentationDefaultLang)
		fmt.Printf("  Idioma defecto apps:             %s\n", p.AppsDefaultLang)
		fmt.Printf("  Idiomas permitidos:              %s\n", strings.Join(p.AllowedLanguages, ", "))
		if p.Notes != "" {
			fmt.Printf("  Notas:                           %s\n", p.Notes)
		}
		if !p.UpdatedAt.IsZero() {
			fmt.Printf("  Actualizado:                     %s por %s\n", p.UpdatedAt.Format("2006-01-02 15:04"), p.UpdatedBy)
		}
		return nil
	},
}

var lenguajePoliticaSetCmd = &cobra.Command{
	Use:   "fijar",
	Short: "Actualiza la politica global de lenguaje",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, ok, err := cargarPoliticaLenguajeDesdeAPI()
		if !ok {
			p, err = db.GetLanguagePolicy()
		}
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("default") {
			v, _ := cmd.Flags().GetString("default")
			p.DefaultLanguage = v
		}
		if cmd.Flags().Changed("docs-default") {
			v, _ := cmd.Flags().GetString("docs-default")
			p.DocumentationDefaultLang = v
		}
		if cmd.Flags().Changed("apps-default") {
			v, _ := cmd.Flags().GetString("apps-default")
			p.AppsDefaultLang = v
		}
		if cmd.Flags().Changed("docs-multilang") {
			v, _ := cmd.Flags().GetBool("docs-multilang")
			p.DocumentationMultilang = v
		}
		if cmd.Flags().Changed("apps-multilang") {
			v, _ := cmd.Flags().GetBool("apps-multilang")
			p.AppsMultilang = v
		}
		if cmd.Flags().Changed("permitidos") {
			v, _ := cmd.Flags().GetString("permitidos")
			p.AllowedLanguages = splitLanguages(v)
		}
		if cmd.Flags().Changed("notas") {
			v, _ := cmd.Flags().GetString("notas")
			p.Notes = strings.TrimSpace(v)
		}
		por, _ := cmd.Flags().GetString("por")
		if ok, err := fijarPoliticaLenguajePorAPI(p, por); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Politica de lenguaje actualizada (%s)\n", p.DefaultLanguage)
			return nil
		}
		if err := db.SetLanguagePolicy(p, por); err != nil {
			return err
		}
		fmt.Printf("✓ Politica de lenguaje actualizada (%s)\n", p.DefaultLanguage)
		return nil
	},
}

var lenguajeMatrizCmd = &cobra.Command{
	Use:   "matriz",
	Short: "Gestion de la matriz de seleccion por proyecto y tarea",
}

var lenguajeMatrizListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista la matriz de lenguaje",
	RunE: func(cmd *cobra.Command, args []string) error {
		lista, ok, err := cargarMatrizLenguajeDesdeAPI()
		if !ok {
			lista, err = db.ListLanguageMatrixEntries()
		}
		if err != nil {
			return err
		}
		if len(lista) == 0 {
			fmt.Println("No hay entradas en la matriz de lenguaje.")
			return nil
		}
		fmt.Printf("%-8s %-16s %-8s %-8s %-18s %s\n", "TIPO", "SELECTOR", "CTX", "IDIOMA", "ACTUALIZADO", "RAZON")
		for _, item := range lista {
			fmt.Printf("%-8s %-16s %-8s %-8s %-18s %s\n",
				item.Scope, item.Selector, item.Context, item.Language,
				formatTime(item.UpdatedAt), item.Reason,
			)
		}
		return nil
	},
}

var lenguajeMatrizFijarCmd = &cobra.Command{
	Use:   "fijar <proyecto|tarea> <selector> <idioma>",
	Short: "Define una regla de lenguaje para un proyecto o tarea",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		contexto, _ := cmd.Flags().GetString("contexto")
		razon, _ := cmd.Flags().GetString("razon")
		por, _ := cmd.Flags().GetString("por")
		if ok, err := fijarEntradaMatrizLenguajePorAPI(args[0], args[1], contexto, args[2], razon, por); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Matriz fijada: %s/%s [%s] = %s\n", args[0], args[1], normalizeContextCmd(contexto), args[2])
			return nil
		}
		entry, err := db.SetLanguageMatrixEntry(args[0], args[1], contexto, args[2], razon, por)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Matriz fijada: %s/%s [%s] = %s\n", entry.Scope, entry.Selector, entry.Context, entry.Language)
		return nil
	},
}

var lenguajeMatrizBorrarCmd = &cobra.Command{
	Use:   "borrar <proyecto|tarea> <selector>",
	Short: "Elimina una regla de la matriz de lenguaje",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		contexto, _ := cmd.Flags().GetString("contexto")
		if ok, err := borrarEntradaMatrizLenguajePorAPI(args[0], args[1], contexto); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Matriz borrada: %s/%s [%s]\n", args[0], args[1], normalizeContextCmd(contexto))
			return nil
		}
		if err := db.DeleteLanguageMatrixEntry(args[0], args[1], contexto); err != nil {
			return err
		}
		fmt.Printf("✓ Matriz borrada: %s/%s [%s]\n", args[0], args[1], normalizeContextCmd(contexto))
		return nil
	},
}

var lenguajeResolverCmd = &cobra.Command{
	Use:   "resolver",
	Short: "Resuelve el idioma efectivo para un proyecto o tarea",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")
		tareaID, err := int64FlagOpt(cmd, "tarea")
		if err != nil {
			return err
		}
		contexto, _ := cmd.Flags().GetString("contexto")
		var tareaPtr *int64
		if tareaID != nil && *tareaID > 0 {
			tareaPtr = tareaID
		}
		res, ok, err := resolverLenguajePorAPI(strings.TrimSpace(proyecto), tareaPtr, contexto)
		if !ok {
			res, err = db.ResolveLanguage(strings.TrimSpace(proyecto), tareaPtr, contexto)
		}
		if err != nil {
			return err
		}
		fmt.Printf("Idioma:   %s\n", res.Idioma)
		fmt.Printf("Contexto: %s\n", res.Contexto)
		if res.Entrada != nil {
			fmt.Printf("Origen:   %s\n", res.Origen)
			fmt.Printf("Clave:    %s/%s [%s]\n", res.Entrada.Scope, res.Entrada.Selector, res.Entrada.Context)
			if res.Entrada.Reason != "" {
				fmt.Printf("Razon:    %s\n", res.Entrada.Reason)
			}
		} else {
			fmt.Printf("Origen:   %s\n", res.Origen)
		}
		return nil
	},
}

func splitLanguages(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if lang := strings.TrimSpace(part); lang != "" {
			out = append(out, lang)
		}
	}
	return out
}

func normalizeContextCmd(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "all"
	}
	return v
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("2006-01-02 15:04")
}

func init() {
	lenguajePoliticaSetCmd.Flags().String("default", "", "Idioma por defecto")
	lenguajePoliticaSetCmd.Flags().Bool("docs-multilang", true, "Documentacion multilenguaje")
	lenguajePoliticaSetCmd.Flags().Bool("apps-multilang", true, "Apps multilenguaje")
	lenguajePoliticaSetCmd.Flags().String("docs-default", "", "Idioma por defecto de documentacion")
	lenguajePoliticaSetCmd.Flags().String("apps-default", "", "Idioma por defecto de apps")
	lenguajePoliticaSetCmd.Flags().String("permitidos", "", "Lista separada por comas de idiomas permitidos")
	lenguajePoliticaSetCmd.Flags().String("notas", "", "Notas de politica")
	lenguajePoliticaSetCmd.Flags().String("por", "orquesta", "Agente que actualiza la politica")

	lenguajeMatrizFijarCmd.Flags().String("contexto", "all", "Contexto: docs, apps o all")
	lenguajeMatrizFijarCmd.Flags().String("razon", "", "Motivo de la regla")
	lenguajeMatrizFijarCmd.Flags().String("por", "orquesta", "Agente que registra la regla")
	lenguajeMatrizBorrarCmd.Flags().String("contexto", "all", "Contexto: docs, apps o all")
	lenguajeResolverCmd.Flags().String("proyecto", "", "Proyecto")
	lenguajeResolverCmd.Flags().Int64("tarea", 0, "ID de tarea")
	lenguajeResolverCmd.Flags().String("contexto", "all", "Contexto: docs, apps o all")

	lenguajePoliticaCmd.AddCommand(lenguajePoliticaVerCmd, lenguajePoliticaSetCmd)
	lenguajeMatrizCmd.AddCommand(lenguajeMatrizListarCmd, lenguajeMatrizFijarCmd, lenguajeMatrizBorrarCmd)
	lenguajeCmd.AddCommand(lenguajePoliticaCmd, lenguajeMatrizCmd, lenguajeResolverCmd)
	rootCmd.AddCommand(lenguajeCmd)
}
