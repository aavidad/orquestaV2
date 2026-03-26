package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestCmdNoUsaSQLDirectoNiAperturasFueraDeExcepcionesControladas(t *testing.T) {
	t.Parallel()

	root := "."
	patronApertura := regexp.MustCompile(`db\.(Open|Close|IsOpen|CurrentDBPath)\(`)
	patronSQLDirecto := regexp.MustCompile(`db\.DB\.(Exec|Query|QueryRow|QueryContext|QueryRowContext|ExecContext)\(`)
	patronEnsureLocal := regexp.MustCompile(`ensureLocalDB\(`)
	patronesHexagonalesCmd := map[string]*regexp.Regexp{
		"api.go":                      regexp.MustCompile(`db\.(RegistrarAgente(Auto)?|RetirarAgente|RehabilitarAgente|EliminarAgente|FusionarAgentes|ResetReanimacion|PausarAgente|Config(Get|Set|All)|GetLanguagePolicy|SetLanguagePolicy|ListLanguageMatrixEntries|SetLanguageMatrixEntry|DeleteLanguageMatrixEntry|ResolveLanguage|GetAgente|CalcularResumenProgresoProyecto|ListarFasesProyecto|RegistrarFaseProyecto|GetFaseProyecto|ActualizarFaseProyecto|RegistrarAvanceTarea|UltimoPresupuestoSesion|EvaluarPresupuestoSesion|RegistrarPresupuestoSesion|GetSesionByID|GetSesionActiva|IniciarSesionContexto|ObtenerUltimaSesion|ActivarAsignacion|GetConector|PropuestasPendientesVoto|GetReglasAgente|GetSkillsAgente)\(`),
		"agente_control_lifecycle.go": regexp.MustCompile(`db\.(GetAgente|Audit)\(`),
	}
	permitidosPorFichero := map[string]map[string]bool{
		"root.go": {
			"Open":   true,
			"Close":  true,
			"IsOpen": true,
		},
		"server.go": {
			"Open":          true,
			"IsOpen":        true,
			"CurrentDBPath": true,
		},
		"cliente_servidor.go": {
			"Open": true,
		},
		"persistencia.go": {
			"CurrentDBPath": true,
		},
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		permitidosEnsureLocal := map[string]bool{
			"cliente_servidor.go": true,
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if filepath.Ext(base) != ".go" || strings.HasSuffix(base, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		texto := string(data)
		for _, match := range patronApertura.FindAllStringSubmatch(texto, -1) {
			llamada := match[1]
			if permitidosPorFichero[base][llamada] {
				continue
			}
			t.Errorf("%s usa db.%s() fuera de los puntos de apertura controlados", base, llamada)
		}
		for _, match := range patronSQLDirecto.FindAllStringSubmatch(texto, -1) {
			t.Errorf("%s usa db.DB.%s() con SQL directo", base, match[1])
		}
		if patronEnsureLocal.MatchString(texto) && !permitidosEnsureLocal[base] {
			t.Errorf("%s usa ensureLocalDB() fuera de la transicion controlada server-first", base)
		}
		if patron, ok := patronesHexagonalesCmd[base]; ok && patron.MatchString(texto) {
			t.Errorf("%s mantiene llamadas a db que deberian pasar por servicio hexagonal", base)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk cmd dir: %v", err)
	}
}
