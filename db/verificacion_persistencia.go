package db

import (
	"database/sql"
	"fmt"
	"strings"

	"orquesta/storage"
)

type ComprobacionPersistencia struct {
	Nombre  string `json:"nombre"`
	Estado  string `json:"estado"`
	Detalle string `json:"detalle,omitempty"`
}

type InformePersistencia struct {
	Driver         string                     `json:"driver"`
	Target         string                     `json:"target"`
	Sano           bool                       `json:"sano"`
	Comprobaciones []ComprobacionPersistencia `json:"comprobaciones"`
}

var tablasCorePersistencia = []string{
	"config",
	"agentes",
	"sesiones",
	"tareas",
	"propuestas",
	"runtime_handles",
	"runtime_orders",
	"autonomia_ciclos",
}

func nuevoInformePersistencia(cfg storage.Config) *InformePersistencia {
	return &InformePersistencia{
		Driver: normalizedDriverName(cfg.Driver),
		Target: storage.DisplayTarget(cfg),
		Sano:   true,
	}
}

func registrarComprobacionPersistencia(informe *InformePersistencia, nombre, estado, detalle string) {
	if informe == nil {
		return
	}
	estado = strings.TrimSpace(strings.ToLower(estado))
	if estado == "" {
		estado = "ok"
	}
	informe.Comprobaciones = append(informe.Comprobaciones, ComprobacionPersistencia{
		Nombre:  strings.TrimSpace(nombre),
		Estado:  estado,
		Detalle: strings.TrimSpace(detalle),
	})
	if estado == "error" {
		informe.Sano = false
	}
}

func verificarConexionPersistencia(informe *InformePersistencia, raw *sql.DB) bool {
	if err := raw.Ping(); err != nil {
		registrarComprobacionPersistencia(informe, "conexion", "error", err.Error())
		return false
	}
	registrarComprobacionPersistencia(informe, "conexion", "ok", "ping correcto")
	return true
}

func verificarTablasCorePersistencia(informe *InformePersistencia, raw *sql.DB, driver string) {
	faltan := make([]string, 0)
	for _, tabla := range tablasCorePersistencia {
		existe, err := tablaExisteEnDB(raw, driver, tabla)
		if err != nil {
			registrarComprobacionPersistencia(informe, "schema_core", "error", fmt.Sprintf("verificando tabla %s: %v", tabla, err))
			return
		}
		if !existe {
			faltan = append(faltan, tabla)
		}
	}
	if len(faltan) > 0 {
		registrarComprobacionPersistencia(informe, "schema_core", "error", "faltan tablas core: "+strings.Join(faltan, ", "))
		return
	}
	registrarComprobacionPersistencia(informe, "schema_core", "ok", fmt.Sprintf("tablas core presentes (%d)", len(tablasCorePersistencia)))
}

func tablaExisteEnDB(raw *sql.DB, driver, tabla string) (bool, error) {
	var total int
	query := storage.RebindQuery(driver, tableExistsQuery(driver))
	if err := raw.QueryRow(query, tabla).Scan(&total); err != nil {
		return false, err
	}
	return total > 0, nil
}

func escalarCadena(raw *sql.DB, driver, query string, args ...any) (string, error) {
	query = storage.RebindQuery(driver, query)
	var out string
	if err := raw.QueryRow(query, args...).Scan(&out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}
