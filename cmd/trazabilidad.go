/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var decisionCmd = &cobra.Command{
	Use:   "decision",
	Short: "Registro CLI de decisiones por proyecto",
}

var decisionListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista decisiones por proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")
		lista, err := db.ListarDecisionesProyecto(strings.TrimSpace(proyecto))
		if err != nil {
			return err
		}
		if len(lista) == 0 {
			fmt.Println("No hay decisiones con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-16s %-8s %-16s %s\n", "ID", "PROYECTO", "IMPACTO", "PROPUESTA", "TITULO")
		for _, d := range lista {
			propuesta := d.PropuestaCodigo
			if propuesta == "" {
				propuesta = "—"
			}
			fmt.Printf("%-5d %-16s %-8s %-16s %s\n", d.ID, d.Proyecto, d.Impacto, propuesta, d.Titulo)
		}
		return nil
	},
}

var decisionRegistrarCmd = &cobra.Command{
	Use:   "registrar",
	Short: "Registra una decisión por proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")
		titulo, _ := cmd.Flags().GetString("titulo")
		solucion, _ := cmd.Flags().GetString("solucion")
		motivo, _ := cmd.Flags().GetString("motivo")
		alternativas, _ := cmd.Flags().GetString("alternativas")
		impacto, _ := cmd.Flags().GetString("impacto")
		propuesta, _ := cmd.Flags().GetString("propuesta")
		tareaID, err := int64FlagOpt(cmd, "tarea")
		if err != nil {
			return err
		}
		por, err := resolverAgenteMemoria(cmd)
		if err != nil {
			return err
		}

		id, err := db.RegistrarDecisionProyecto(&db.DecisionProyecto{
			Proyecto:                strings.TrimSpace(proyecto),
			Titulo:                  strings.TrimSpace(titulo),
			SolucionElegida:         strings.TrimSpace(solucion),
			Motivo:                  strings.TrimSpace(motivo),
			AlternativasDescartadas: strings.TrimSpace(alternativas),
			Impacto:                 strings.TrimSpace(impacto),
			PropuestaCodigo:         strings.TrimSpace(propuesta),
			TareaID:                 tareaID,
			RegistradoPor:           por,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Decisión #%d registrada\n", id)
		return nil
	},
}

var decisionEditarCmd = &cobra.Command{
	Use:   "editar <id>",
	Short: "Edita una decisión por proyecto",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		d, err := db.GetDecisionProyecto(id)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("proyecto") {
			v, _ := cmd.Flags().GetString("proyecto")
			d.Proyecto = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("titulo") {
			v, _ := cmd.Flags().GetString("titulo")
			d.Titulo = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("solucion") {
			v, _ := cmd.Flags().GetString("solucion")
			d.SolucionElegida = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("motivo") {
			v, _ := cmd.Flags().GetString("motivo")
			d.Motivo = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("alternativas") {
			v, _ := cmd.Flags().GetString("alternativas")
			d.AlternativasDescartadas = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("impacto") {
			v, _ := cmd.Flags().GetString("impacto")
			d.Impacto = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("propuesta") {
			v, _ := cmd.Flags().GetString("propuesta")
			d.PropuestaCodigo = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("tarea") {
			tareaID, err := int64FlagOpt(cmd, "tarea")
			if err != nil {
				return err
			}
			d.TareaID = tareaID
		}
		if cmd.Flags().Changed("por") || cmd.Flags().Changed("agente") {
			por, err := resolverAgenteMemoria(cmd)
			if err != nil {
				return err
			}
			d.RegistradoPor = por
		}
		if err := db.ActualizarDecisionProyecto(d); err != nil {
			return err
		}
		fmt.Printf("✓ Decisión #%d actualizada\n", id)
		return nil
	},
}

var documentacionExternaCmd = &cobra.Command{
	Use:   "documentacion-externa",
	Short: "Registro CLI de documentación externa referenciada",
}

var documentacionExternaListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista documentación externa por proyecto",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")
		lista, err := db.ListarDocumentosExternos(strings.TrimSpace(proyecto))
		if err != nil {
			return err
		}
		if len(lista) == 0 {
			fmt.Println("No hay documentación externa con ese filtro.")
			return nil
		}
		fmt.Printf("%-5s %-16s %-12s %-10s %s\n", "ID", "PROYECTO", "TIPO", "ESTADO", "RUTA")
		for _, doc := range lista {
			fmt.Printf("%-5d %-16s %-12s %-10s %s\n", doc.ID, doc.Proyecto, doc.TipoDocumento, doc.Estado, doc.Ruta)
		}
		return nil
	},
}

var documentacionExternaRegistrarCmd = &cobra.Command{
	Use:   "registrar",
	Short: "Registra una referencia de documentación externa",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyecto, _ := cmd.Flags().GetString("proyecto")
		tipo, _ := cmd.Flags().GetString("tipo")
		ruta, _ := cmd.Flags().GetString("ruta")
		resumen, _ := cmd.Flags().GetString("resumen")
		estado, _ := cmd.Flags().GetString("estado")
		por, err := resolverAgenteMemoria(cmd)
		if err != nil {
			return err
		}
		id, err := db.RegistrarDocumentoExterno(&db.DocumentoExterno{
			Proyecto:      strings.TrimSpace(proyecto),
			TipoDocumento: strings.TrimSpace(tipo),
			Ruta:          strings.TrimSpace(ruta),
			Resumen:       strings.TrimSpace(resumen),
			Estado:        strings.TrimSpace(estado),
			RegistradoPor: por,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Documento externo #%d registrado\n", id)
		return nil
	},
}

var documentacionExternaEditarCmd = &cobra.Command{
	Use:   "editar <id>",
	Short: "Edita una referencia de documentación externa",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("id inválido")
		}
		doc, err := db.GetDocumentoExterno(id)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed("proyecto") {
			v, _ := cmd.Flags().GetString("proyecto")
			doc.Proyecto = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("tipo") {
			v, _ := cmd.Flags().GetString("tipo")
			doc.TipoDocumento = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("ruta") {
			v, _ := cmd.Flags().GetString("ruta")
			doc.Ruta = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("resumen") {
			v, _ := cmd.Flags().GetString("resumen")
			doc.Resumen = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("estado") {
			v, _ := cmd.Flags().GetString("estado")
			doc.Estado = strings.TrimSpace(v)
		}
		if cmd.Flags().Changed("por") || cmd.Flags().Changed("agente") {
			por, err := resolverAgenteMemoria(cmd)
			if err != nil {
				return err
			}
			doc.RegistradoPor = por
		}
		if err := db.ActualizarDocumentoExterno(doc); err != nil {
			return err
		}
		fmt.Printf("✓ Documento externo #%d actualizado\n", id)
		return nil
	},
}

func init() {
	decisionListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")
	decisionRegistrarCmd.Flags().String("proyecto", "", "Proyecto")
	decisionRegistrarCmd.Flags().String("titulo", "", "Título de la decisión")
	decisionRegistrarCmd.Flags().String("solucion", "", "Solución elegida")
	decisionRegistrarCmd.Flags().String("motivo", "", "Motivo")
	decisionRegistrarCmd.Flags().String("alternativas", "", "Alternativas descartadas")
	decisionRegistrarCmd.Flags().String("impacto", "medio", "Impacto: alto, medio, bajo")
	decisionRegistrarCmd.Flags().String("propuesta", "", "Código de propuesta relacionada")
	decisionRegistrarCmd.Flags().Int64("tarea", 0, "ID de tarea relacionada")
	decisionRegistrarCmd.Flags().String("por", "", "Registrado por")
	decisionRegistrarCmd.Flags().String("agente", "", "Alias de --por")
	decisionEditarCmd.Flags().String("proyecto", "", "Proyecto")
	decisionEditarCmd.Flags().String("titulo", "", "Título")
	decisionEditarCmd.Flags().String("solucion", "", "Solución elegida")
	decisionEditarCmd.Flags().String("motivo", "", "Motivo")
	decisionEditarCmd.Flags().String("alternativas", "", "Alternativas descartadas")
	decisionEditarCmd.Flags().String("impacto", "", "Impacto")
	decisionEditarCmd.Flags().String("propuesta", "", "Código de propuesta")
	decisionEditarCmd.Flags().Int64("tarea", 0, "ID de tarea relacionada")
	decisionEditarCmd.Flags().String("por", "", "Registrado por")
	decisionEditarCmd.Flags().String("agente", "", "Alias de --por")
	decisionCmd.AddCommand(decisionListarCmd, decisionRegistrarCmd, decisionEditarCmd)

	documentacionExternaListarCmd.Flags().String("proyecto", "", "Filtrar por proyecto")
	documentacionExternaRegistrarCmd.Flags().String("proyecto", "", "Proyecto")
	documentacionExternaRegistrarCmd.Flags().String("tipo", "referencia", "Tipo de documento")
	documentacionExternaRegistrarCmd.Flags().String("ruta", "", "Ruta o referencia")
	documentacionExternaRegistrarCmd.Flags().String("resumen", "", "Resumen")
	documentacionExternaRegistrarCmd.Flags().String("estado", "vigente", "Estado")
	documentacionExternaRegistrarCmd.Flags().String("por", "", "Registrado por")
	documentacionExternaRegistrarCmd.Flags().String("agente", "", "Alias de --por")
	documentacionExternaEditarCmd.Flags().String("proyecto", "", "Proyecto")
	documentacionExternaEditarCmd.Flags().String("tipo", "", "Tipo")
	documentacionExternaEditarCmd.Flags().String("ruta", "", "Ruta")
	documentacionExternaEditarCmd.Flags().String("resumen", "", "Resumen")
	documentacionExternaEditarCmd.Flags().String("estado", "", "Estado")
	documentacionExternaEditarCmd.Flags().String("por", "", "Registrado por")
	documentacionExternaEditarCmd.Flags().String("agente", "", "Alias de --por")
	documentacionExternaCmd.AddCommand(documentacionExternaListarCmd, documentacionExternaRegistrarCmd, documentacionExternaEditarCmd)

	rootCmd.AddCommand(decisionCmd, documentacionExternaCmd)
}
