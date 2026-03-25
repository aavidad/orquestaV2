/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var memoriaCmd = &cobra.Command{
	Use:   "memoria",
	Short: "Gestión de memoria persistente de entidades",
}

func memoriaModoRecuperacionLocalExplicito() bool {
	return strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1"
}

func memoriaErrorServerFirst() error {
	return fmt.Errorf("este comando exige servidor/daemon de Orquesta; usa --local solo en recuperacion explicita o exporta ORQUESTA_FORCE_LOCAL_DB=1")
}

var memoriaListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Lista entidades de memoria persistente",
	RunE: func(cmd *cobra.Command, args []string) error {
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		tipo, _ := cmd.Flags().GetString("tipo")
		params := url.Values{}
		if strings.TrimSpace(proyectoRef) != "" {
			params.Set("proyecto", strings.TrimSpace(proyectoRef))
		}
		if strings.TrimSpace(tipo) != "" {
			params.Set("tipo", strings.TrimSpace(tipo))
		}

		var entidades []*db.EntidadMemoria
		var resp apiMemoriaEntidadesResponse
		if ok, err := apiGetQuery("/api/memoria", params, &resp); err != nil {
			return err
		} else if ok {
			entidades = resp.Entidades
		} else {
			if !memoriaModoRecuperacionLocalExplicito() {
				return memoriaErrorServerFirst()
			}
			if err := ensureLocalDB(); err != nil {
				return err
			}
			filter := db.FiltroEntidadesMemoria{}
			if strings.TrimSpace(proyectoRef) != "" {
				proyecto, err := db.GetProyecto(proyectoRef)
				if err != nil {
					return err
				}
				filter.ProyectoID = &proyecto.ID
			}
			if strings.TrimSpace(tipo) != "" {
				filter.Tipo = &tipo
			}

			var err error
			entidades, err = db.ListarEntidadesMemoria(filter)
			if err != nil {
				return err
			}
		}
		if len(entidades) == 0 {
			fmt.Println("No hay entidades de memoria con ese filtro.")
			return nil
		}

		fmt.Printf("%-18s %-10s %-15s %s\n", "NOMBRE", "TIPO", "VERIFICADO_POR", "VALOR")
		for _, entidad := range entidades {
			fmt.Printf("%-18s %-10s %-15s %s\n",
				truncar(entidad.Nombre, 18),
				truncar(entidad.Tipo, 10),
				truncar(entidad.VerificadoPor, 15),
				truncar(entidad.ValorJSON, 48),
			)
		}
		return nil
	},
}

var memoriaVerCmd = &cobra.Command{
	Use:   "ver <nombre>",
	Short: "Muestra una entidad de memoria",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		params := url.Values{}
		if strings.TrimSpace(proyectoRef) != "" {
			params.Set("proyecto", strings.TrimSpace(proyectoRef))
		}

		var entidad *db.EntidadMemoria
		var resp apiMemoriaEntidadResponse
		if ok, err := apiGetQuery("/api/memoria/"+args[0], params, &resp); err != nil {
			return err
		} else if ok {
			entidad = resp.Entidad
		} else {
			if !memoriaModoRecuperacionLocalExplicito() {
				return memoriaErrorServerFirst()
			}
			if err := ensureLocalDB(); err != nil {
				return err
			}
			var proyectoID *int64
			if strings.TrimSpace(proyectoRef) != "" {
				proyecto, err := db.GetProyecto(proyectoRef)
				if err != nil {
					return err
				}
				proyectoID = &proyecto.ID
			}

			var err error
			entidad, err = db.GetEntidadMemoria(args[0], proyectoID)
			if err != nil {
				return err
			}
			if entidad == nil {
				return fmt.Errorf("entidad '%s' no encontrada", args[0])
			}
		}

		fmt.Printf("Entidad:      %s\n", entidad.Nombre)
		fmt.Printf("Tipo:         %s\n", entidad.Tipo)
		fmt.Printf("Verificado:   %s\n", valorVacio(entidad.VerificadoPor))
		fmt.Printf("Última verif: %s\n", entidad.UltimaVerificacion.Format("2006-01-02 15:04:05"))
		fmt.Printf("Valor:        %s\n", entidad.ValorJSON)
		fmt.Printf("Metadata:     %s\n", entidad.MetadataJSON)
		return nil
	},
}

var memoriaGuardarCmd = &cobra.Command{
	Use:   "guardar <nombre> <tipo>",
	Short: "Crea o actualiza una entidad de memoria",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		nombre := strings.TrimSpace(args[0])
		tipo := strings.TrimSpace(args[1])
		valor, _ := cmd.Flags().GetString("valor")
		metadata, _ := cmd.Flags().GetString("metadata")
		verificadoPor, _ := cmd.Flags().GetString("verificado-por")
		proyectoRef, _ := cmd.Flags().GetString("proyecto")
		if strings.TrimSpace(valor) == "" {
			return fmt.Errorf("--valor es obligatorio")
		}

		if ok, err := apiPost("/api/memoria", apiMemoriaEntidadRequest{
			Nombre:        nombre,
			Tipo:          tipo,
			Valor:         valor,
			Metadata:      metadata,
			VerificadoPor: verificadoPor,
			Proyecto:      proyectoRef,
		}, &map[string]any{}); err != nil {
			return err
		} else if ok {
			fmt.Printf("✓ Entidad de memoria guardada: %s\n", nombre)
			return nil
		}

		if !memoriaModoRecuperacionLocalExplicito() {
			return memoriaErrorServerFirst()
		}
		if err := ensureLocalDB(); err != nil {
			return err
		}
		var proyectoID *int64
		if strings.TrimSpace(proyectoRef) != "" {
			proyecto, err := db.GetProyecto(proyectoRef)
			if err != nil {
				return err
			}
			proyectoID = &proyecto.ID
		}
		id, err := db.UpsertEntidadMemoria(&db.EntidadMemoria{
			Nombre:        nombre,
			Tipo:          tipo,
			ValorJSON:     valor,
			MetadataJSON:  metadata,
			VerificadoPor: verificadoPor,
			ProyectoID:    proyectoID,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Entidad de memoria #%d guardada: %s\n", id, nombre)
		return nil
	},
}

func init() {
	memoriaListarCmd.Flags().String("proyecto", "", "Proyecto asociado a la memoria")
	memoriaListarCmd.Flags().String("tipo", "", "Filtrar por tipo de entidad")
	memoriaVerCmd.Flags().String("proyecto", "", "Proyecto asociado a la memoria")
	memoriaGuardarCmd.Flags().String("proyecto", "", "Proyecto asociado a la memoria")
	memoriaGuardarCmd.Flags().String("valor", "", "Valor JSON de la entidad")
	memoriaGuardarCmd.Flags().String("metadata", "{}", "Metadata JSON de la entidad")
	memoriaGuardarCmd.Flags().String("verificado-por", "", "Agente que verifica la entidad")
	memoriaCmd.AddCommand(memoriaListarCmd, memoriaVerCmd, memoriaGuardarCmd)
}
