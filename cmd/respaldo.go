/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"orquesta/db"
)

var ahoraRespaldo = time.Now

var respaldoCmd = &cobra.Command{
	Use:   "respaldo",
	Short: "Gestión de copias de seguridad de la base de datos",
}

var respaldoBDCmd = &cobra.Command{
	Use:   "bd",
	Short: "Crea un snapshot de la BD actual con nombre fechable",
	RunE: func(cmd *cobra.Command, args []string) error {
		destino, _ := cmd.Flags().GetString("destino")
		retener, _ := cmd.Flags().GetInt("retener")
		etiqueta, _ := cmd.Flags().GetString("etiqueta")
		ruta, err := ejecutarRespaldoBD(destino, etiqueta, retener)
		if err != nil {
			return err
		}
		fmt.Printf("✓ Respaldo creado: %s\n", ruta)
		return nil
	},
}

func ejecutarRespaldoBD(destino, etiqueta string, retener int) (string, error) {
	if db.DB == nil {
		return "", fmt.Errorf("la base de datos no está inicializada")
	}

	destino = strings.TrimSpace(destino)
	if destino == "" {
		destino = respaldoDestinoPorDefecto()
	}
	destinoAbs, err := filepath.Abs(destino)
	if err != nil {
		return "", fmt.Errorf("resolver destino: %w", err)
	}
	if err := os.MkdirAll(destinoAbs, 0o755); err != nil {
		return "", fmt.Errorf("crear destino %s: %w", destinoAbs, err)
	}

	nombre := nombreRespaldoFechable(ahoraRespaldo(), etiqueta)
	rutaSalida := filepath.Join(destinoAbs, nombre)
	if err := crearRespaldoSQLite(rutaSalida); err != nil {
		return "", err
	}

	if retener > 0 {
		if err := aplicarRetencionRespaldo(destinoAbs, retener); err != nil {
			return "", err
		}
	}

	return rutaSalida, nil
}

func crearRespaldoSQLite(rutaSalida string) error {
	rutaSalida = strings.TrimSpace(rutaSalida)
	if rutaSalida == "" {
		return fmt.Errorf("ruta de salida vacía")
	}
	if _, err := db.DB.Exec(fmt.Sprintf("VACUUM INTO %s", literalSQLite(rutaSalida))); err != nil {
		return fmt.Errorf("crear respaldo sqlite: %w", err)
	}
	return nil
}

func aplicarRetencionRespaldo(destino string, retener int) error {
	if retener <= 0 {
		return nil
	}

	patron := filepath.Join(destino, "*_orquesta.db.bak")
	ficheros, err := filepath.Glob(patron)
	if err != nil {
		return fmt.Errorf("listar respaldos: %w", err)
	}
	if len(ficheros) <= retener {
		return nil
	}

	sort.Strings(ficheros)
	exceso := len(ficheros) - retener
	for i := 0; i < exceso; i++ {
		if err := os.Remove(ficheros[i]); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("eliminar respaldo antiguo %s: %w", ficheros[i], err)
		}
	}
	return nil
}

func nombreRespaldoFechable(ts time.Time, etiqueta string) string {
	base := ts.Format("2006-01-02_15-04-05.000000000")
	if limpio := limpiarEtiquetaRespaldo(etiqueta); limpio != "" {
		base = base + "_" + limpio
	}
	return base + "_orquesta.db.bak"
}

func limpiarEtiquetaRespaldo(etiqueta string) string {
	etiqueta = strings.TrimSpace(strings.ToLower(etiqueta))
	if etiqueta == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(etiqueta))
	for _, r := range etiqueta {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func respaldoDestinoPorDefecto() string {
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_BACKUP_DIR")); v != "" {
		return v
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, "Trabajo", "backups", "orquestador")
	}
	return filepath.Join(".", "backups", "orquestador")
}

func literalSQLite(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func init() {
	respaldoBDCmd.Flags().String("destino", "", "Directorio destino del respaldo")
	respaldoBDCmd.Flags().String("etiqueta", "", "Etiqueta opcional para el nombre del fichero")
	respaldoBDCmd.Flags().Int("retener", 0, "Número máximo de respaldos a conservar en el destino")

	respaldoCmd.AddCommand(respaldoBDCmd)
	rootCmd.AddCommand(respaldoCmd)
}
