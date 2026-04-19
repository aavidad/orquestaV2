package db

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"orquesta/storage"
)

func TestOpenPostgresBootstrapLimpio(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ORQUESTA_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("ORQUESTA_TEST_POSTGRES_DSN no definido; se omite el test de integración con Postgres")
	}

	Close()

	adminDB, err := storage.Open(storage.Config{
		Driver:       "postgres",
		DSN:          dsn,
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("abriendo conexion de limpieza: %v", err)
	}
	defer adminDB.Close()

	if err := adminDB.Ping(); err != nil {
		t.Fatalf("ping conexion de limpieza: %v", err)
	}
	schemaName := fmt.Sprintf("orquesta_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + quotePostgresIdent(schemaName)); err != nil {
		t.Fatalf("creando schema de prueba: %v", err)
	}
	defer func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + quotePostgresIdent(schemaName) + ` CASCADE`)
	}()

	schemaDSN, err := postgresTestDSNWithSearchPath(dsn, schemaName)
	if err != nil {
		t.Fatalf("construyendo dsn con search_path: %v", err)
	}

	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_DSN", schemaDSN)
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "true")
	t.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", "1")

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	if got := DriverName(); got != "postgres" {
		t.Fatalf("DriverName=%q, want postgres", got)
	}
	if !BootstrapSchemaEnabled() {
		t.Fatalf("BootstrapSchemaEnabled deberia estar activo para esta prueba")
	}

	for _, table := range []string{
		"agentes",
		"config",
		"reglas",
		"skills",
		"workflows",
		"proyectos",
		"sesiones",
		"runtime_handles",
		"runtime_orders",
		"pools_capacidad",
		"pool_modelos",
		"decisiones_proyecto",
		"documentos_externos",
		"git_merges",
	} {
		exists, err := TableExists(table)
		if err != nil {
			t.Fatalf("TableExists(%s): %v", table, err)
		}
		if !exists {
			t.Fatalf("tabla %s no creada por el bootstrap de postgres", table)
		}
	}

	for _, table := range []string{"reglas", "skills", "workflows"} {
		got, err := tableRowCount(table)
		if err != nil {
			t.Fatalf("tableRowCount(%s): %v", table, err)
		}
		if got == 0 {
			t.Fatalf("bootstrap postgres deberia dejar semillas minimas en %s", table)
		}
	}

	var version string
	if err := DB.QueryRow(`SELECT valor FROM config WHERE clave = ?`, "version").Scan(&version); err != nil {
		t.Fatalf("leyendo config semilla version: %v", err)
	}
	if version != "1.0.0" {
		t.Fatalf("valor version inesperado: %s", version)
	}
	for _, column := range []struct {
		table string
		name  string
	}{
		{table: "agentes", name: "estado_sesion"},
		{table: "agentes", name: "estado_cuota"},
		{table: "sesiones", name: "heartbeat_at"},
		{table: "runtime_orders", name: "available_at"},
	} {
		exists, err := ColumnExists(column.table, column.name)
		if err != nil {
			t.Fatalf("ColumnExists(%s.%s): %v", column.table, column.name, err)
		}
		if !exists {
			t.Fatalf("columna %s.%s no creada por bootstrap/migracion postgres", column.table, column.name)
		}
	}

	Close()

	if err := Open(); err != nil {
		t.Fatalf("Open segundo pase idempotente: %v", err)
	}
	defer Close()
}

func TestPostgresUpsertProyectoAceptaActivoBooleano(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ORQUESTA_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("ORQUESTA_TEST_POSTGRES_DSN no definido; se omite el test de integración con Postgres")
	}

	Close()

	adminDB, err := storage.Open(storage.Config{
		Driver:       "postgres",
		DSN:          dsn,
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("abriendo conexion de limpieza: %v", err)
	}
	defer adminDB.Close()

	schemaName := fmt.Sprintf("orquesta_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + quotePostgresIdent(schemaName)); err != nil {
		t.Fatalf("creando schema de prueba: %v", err)
	}
	defer func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + quotePostgresIdent(schemaName) + ` CASCADE`)
	}()

	schemaDSN, err := postgresTestDSNWithSearchPath(dsn, schemaName)
	if err != nil {
		t.Fatalf("construyendo dsn con search_path: %v", err)
	}

	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_DSN", schemaDSN)
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "true")
	t.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", "1")

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer Close()

	id, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if id == 0 {
		t.Fatalf("id de proyecto inesperado: %d", id)
	}

	activo := true
	proyectos, err := ListarProyectos(FiltroProyectos{Activo: &activo})
	if err != nil {
		t.Fatalf("ListarProyectos: %v", err)
	}
	if len(proyectos) != 1 || proyectos[0].Slug != "orquesta" {
		t.Fatalf("proyectos activos inesperados: %+v", proyectos)
	}
}

func TestPostgresIniciarSesionContextoDevuelveSesionCreada(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ORQUESTA_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("ORQUESTA_TEST_POSTGRES_DSN no definido; se omite el test de integración con Postgres")
	}

	Close()

	adminDB, err := storage.Open(storage.Config{
		Driver:       "postgres",
		DSN:          dsn,
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("abriendo conexion de limpieza: %v", err)
	}
	defer adminDB.Close()

	schemaName := fmt.Sprintf("orquesta_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + quotePostgresIdent(schemaName)); err != nil {
		t.Fatalf("creando schema de prueba: %v", err)
	}
	defer func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + quotePostgresIdent(schemaName) + ` CASCADE`)
	}()

	schemaDSN, err := postgresTestDSNWithSearchPath(dsn, schemaName)
	if err != nil {
		t.Fatalf("construyendo dsn con search_path: %v", err)
	}

	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_DSN", schemaDSN)
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "true")
	t.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", "1")

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer Close()

	if err := RegistrarAgente("CodexPg2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}

	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "CodexPg2",
		ProyectoID:        &proyectoID,
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-postgres",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if sesion == nil || sesion.ID <= 0 {
		t.Fatalf("sesion inesperada: %+v", sesion)
	}
	if sesion.ProyectoID == nil || *sesion.ProyectoID != proyectoID {
		t.Fatalf("sesion proyecto inesperado: %+v", sesion)
	}
}

func TestPostgresInsertReturningPathsWriteSet(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("ORQUESTA_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("ORQUESTA_TEST_POSTGRES_DSN no definido; se omite el test de integración con Postgres")
	}

	Close()

	adminDB, err := storage.Open(storage.Config{
		Driver:       "postgres",
		DSN:          dsn,
		MaxOpenConns: 1,
	})
	if err != nil {
		t.Fatalf("abriendo conexion de limpieza: %v", err)
	}
	defer adminDB.Close()

	schemaName := fmt.Sprintf("orquesta_test_%d", time.Now().UnixNano())
	if _, err := adminDB.Exec(`CREATE SCHEMA ` + quotePostgresIdent(schemaName)); err != nil {
		t.Fatalf("creando schema de prueba: %v", err)
	}
	defer func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + quotePostgresIdent(schemaName) + ` CASCADE`)
	}()

	schemaDSN, err := postgresTestDSNWithSearchPath(dsn, schemaName)
	if err != nil {
		t.Fatalf("construyendo dsn con search_path: %v", err)
	}

	t.Setenv("ORQUESTA_DB_DRIVER", "postgres")
	t.Setenv("ORQUESTA_DB_DSN", schemaDSN)
	t.Setenv("ORQUESTA_DB_BOOTSTRAP", "true")
	t.Setenv("ORQUESTA_DB_MAX_OPEN_CONNS", "1")

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer Close()

	if err := RegistrarAgente("CodexPgSlice", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta",
		Nombre:  "Orquesta",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}

	faseID, err := RegistrarFaseProyecto(&FaseProyecto{
		Proyecto: "orquesta",
		Nombre:   "Portabilidad PG",
		Estado:   "activa",
	})
	if err != nil {
		t.Fatalf("RegistrarFaseProyecto: %v", err)
	}
	if faseID <= 0 {
		t.Fatalf("id de fase inesperado: %d", faseID)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:     "Validar RETURNING PG",
		ProyectoID: &proyectoID,
		Prioridad:  PrioridadAlta,
		CreadoPor:  "CodexPgSlice",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexPgSlice"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "CodexPgSlice"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	solicitud, err := SolicitarRefineria(tareaID, "CodexPgSlice", "feature/pg-returning", t.TempDir(), "true")
	if err != nil {
		t.Fatalf("SolicitarRefineria: %v", err)
	}
	if solicitud == nil || solicitud.ID <= 0 {
		t.Fatalf("solicitud de refineria inesperada: %+v", solicitud)
	}

	cicloID, err := RegistrarAutonomiaCiclo(&AutonomiaCiclo{
		ProyectoID: proyectoID,
		Kind:       "supervision",
		Agente:     "CodexPgSlice",
	})
	if err != nil {
		t.Fatalf("RegistrarAutonomiaCiclo: %v", err)
	}
	if cicloID <= 0 {
		t.Fatalf("id de autonomia_ciclo inesperado: %d", cicloID)
	}

	gateID, err := CrearReviewGate(&ReviewGate{
		ProyectoID:     &proyectoID,
		TareaID:        &tareaID,
		RequestedBy:    "orquesta",
		ReviewerAgente: "CodexPgSlice",
	})
	if err != nil {
		t.Fatalf("CrearReviewGate: %v", err)
	}
	if gateID <= 0 {
		t.Fatalf("id de review_gate inesperado: %d", gateID)
	}

	entregaID, err := CrearEntregaNotificacion("slack", "#ops", EventoNotificacion{
		Tipo:   "refineria",
		ID:     solicitud.ID,
		Codigo: "PG-SLICE",
		Texto:  "validacion postgres",
	})
	if err != nil {
		t.Fatalf("CrearEntregaNotificacion: %v", err)
	}
	if entregaID <= 0 {
		t.Fatalf("id de entrega inesperado: %d", entregaID)
	}

	if gate, err := GetReviewGate(gateID); err != nil || gate == nil {
		t.Fatalf("GetReviewGate: gate=%+v err=%v", gate, err)
	}
	if entrega, err := GetEntregaNotificacion(entregaID); err != nil || entrega == nil {
		t.Fatalf("GetEntregaNotificacion: entrega=%+v err=%v", entrega, err)
	}
}

func tableRowCount(table string) (int64, error) {
	var count int64
	if err := DB.QueryRow(`SELECT COUNT(*) FROM ` + quotePostgresIdent(table)).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func seedRowCount(table string) (int, bool) {
	for _, group := range schemaSeedGroups {
		if group.table == table {
			return len(group.rows), true
		}
	}
	return 0, false
}

func quotePostgresIdent(v string) string {
	return `"` + strings.ReplaceAll(v, `"`, `""`) + `"`
}

func postgresTestDSNWithSearchPath(baseDSN, schema string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseDSN))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
