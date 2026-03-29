/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "strings"

func postMigrationStatements() []string {
	return []string{
		`ALTER TABLE sesiones ADD COLUMN conector_id INTEGER REFERENCES conectores(id)`,
		`ALTER TABLE agentes ADD COLUMN estado_sesion TEXT DEFAULT NULL`,
		`ALTER TABLE tareas ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`ALTER TABLE propuestas ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`ALTER TABLE sesiones ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`ALTER TABLE sesiones ADD COLUMN pool_id INTEGER REFERENCES pools_capacidad(id)`,
		`ALTER TABLE sesiones ADD COLUMN estado TEXT NOT NULL DEFAULT 'activa'`,
		`ALTER TABLE sesiones ADD COLUMN cwd TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN herramienta TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN external_session_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN resume_payload_json TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN resumen_continuidad TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN branch TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN heartbeat_at DATETIME`,
		`ALTER TABLE sesiones ADD COLUMN host TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN pid INTEGER`,
		`ALTER TABLE proyectos_autonomia ADD COLUMN supervisor_agente TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE proyectos_autonomia ADD COLUMN reviewer_agente TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tareas ADD COLUMN contrato_definido INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tareas ADD COLUMN blueprint_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runtime_orders ADD COLUMN available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE agentes ADD COLUMN consumo_dia_segundos INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE agentes ADD COLUMN consumo_semanal_segundos INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE agentes ADD COLUMN limite_dia_segundos INTEGER NOT NULL DEFAULT 14400`,
		`ALTER TABLE agentes ADD COLUMN limite_semanal_segundos INTEGER NOT NULL DEFAULT 43200`,
		`ALTER TABLE agentes ADD COLUMN last_usage_reset_at DATETIME`,
		`ALTER TABLE agentes ADD COLUMN estado_cuota TEXT NOT NULL DEFAULT 'activo'`,
		`ALTER TABLE agentes ADD COLUMN reanimar_at DATETIME`,
		`ALTER TABLE agentes ADD COLUMN motivo_pausa TEXT`,
		`ALTER TABLE skills ADD COLUMN escenario TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE skills ADD COLUMN prioridad INTEGER NOT NULL DEFAULT 100`,
		`ALTER TABLE skills ADD COLUMN aliases_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE skills ADD COLUMN herramientas_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE skills ADD COLUMN origen TEXT NOT NULL DEFAULT 'builtin'`,
		`ALTER TABLE skills ADD COLUMN nivel_riesgo TEXT NOT NULL DEFAULT 'bajo'`,
		`ALTER TABLE skills ADD COLUMN requiere_aprobacion INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE skills_versiones ADD COLUMN escenario TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE skills_versiones ADD COLUMN prioridad INTEGER NOT NULL DEFAULT 100`,
		`ALTER TABLE skills_versiones ADD COLUMN aliases_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE skills_versiones ADD COLUMN herramientas_json TEXT NOT NULL DEFAULT '[]'`,
		`ALTER TABLE skills_versiones ADD COLUMN origen TEXT NOT NULL DEFAULT 'builtin'`,
		`ALTER TABLE skills_versiones ADD COLUMN nivel_riesgo TEXT NOT NULL DEFAULT 'bajo'`,
		`ALTER TABLE skills_versiones ADD COLUMN requiere_aprobacion INTEGER NOT NULL DEFAULT 0`,
		`UPDATE propuestas
		 SET cerrada_at = NULL
		 WHERE estado = 'abierta' AND cerrada_at IS NOT NULL`,
		`UPDATE reglas
		 SET descripcion='No escribir código sin propuesta OP-XXX aprobada en la app de orquestación.'
		 WHERE tipo_agente='programador' AND categoria='calidad' AND titulo='Propuesta antes de código'`,
		`UPDATE reglas
		 SET descripcion='Nueva propuesta: orquesta propuesta nueva "Título" --descripcion "Descripción" --agente <nombre>  |  Votar: orquesta votar <OP-XXX> <acuerdo|desacuerdo|abstencion> --agente <nombre> --comentario "razón"  |  Ver propuesta: orquesta propuesta ver <OP-XXX>  |  Ver pendientes de voto: incluidas en el briefing de sesion inicio.'
		 WHERE tipo_agente='programador' AND categoria='comandos' AND titulo='Propuestas y votación'`,
		`UPDATE workflows
		 SET pasos='["1. Leer la propuesta completa: orquesta propuesta ver <codigo>","2. Analizar impacto técnico en módulos asignados","3. Votar: orquesta votar <codigo> <acuerdo|desacuerdo|abstencion> --agente <mi-nombre> --comentario \"razón\"","4. Si desacuerdo: añadir comentario técnico con alternativa concreta"]'
		 WHERE tipo_agente='programador' AND nombre='votar-propuesta'`,
		`UPDATE workflows
		 SET pasos='["1. Ejecutar: orquesta sesion inicio <mi-nombre>","2. Ver tareas asignadas: orquesta tarea listar --agente <mi-nombre>","3. Votar todas las propuestas con posicion pendiente para mi agente","4. Iniciar la tarea en la app: orquesta tarea iniciar <id> <mi-nombre>","5. Leer el doc del módulo asignado en docs/modulos/MXX_*.md","6. Operar con autonomía para acciones normales; si una acción es destructiva o peligrosa, consultar antes con Orquesta o con otro agente del mismo proyecto"]'
		 WHERE tipo_agente='programador' AND nombre='inicio-sesion'`,
		`UPDATE workflows
		 SET pasos='["1. Asegurar que todo el trabajo está commiteado (git status limpio)","2. Completar o bloquear mis tareas en la app de orquestación según corresponda","3. Ejecutar: orquesta sesion fin <mi-nombre>"]'
		 WHERE tipo_agente='programador' AND nombre='fin-sesion'`,
		`UPDATE workflows
		 SET pasos='["1. Crear propuesta OP-XXX con orquesta propuesta nueva y esperar consenso en la app","2. Crear fichero de dominio: internal/domain/<modulo>_entities.go","3. Crear interfaces: internal/domain/<modulo>_interfaces.go","4. Crear migración SQL: migrations/XXXXXX_<modulo>.up.sql","5. Crear repositorio: internal/repository/postgres_<modulo>.go","6. Commit: feat(MXX): dominio e interfaces","7. Crear servicio: internal/service/<modulo>_service.go","8. Commit: feat(MXX): servicio","9. Crear tests: internal/service/<modulo>_service_test.go","10. Ejecutar go test ./internal/service/... → debe pasar","11. Commit: feat(MXX): tests servicio","12. Crear handler REST: internal/api/<modulo>_handler.go","13. Commit: feat(MXX): handler REST","14. Gate final: go build ./... && go vet ./... && go test ./... → notificar a Antigravity"]'
		 WHERE tipo_agente='programador' AND nombre='crear-modulo'`,
		`UPDATE workflows
		 SET pasos='["1. Ejecutar: orquesta sesion inicio antigravity","2. Ver tareas asignadas: orquesta tarea listar --agente antigravity","3. Votar propuestas con posicion pendiente para antigravity","4. Revisar docs/00_INDICE.md para detectar gaps"]'
		 WHERE tipo_agente='documentador' AND nombre='inicio-sesion'`,
	}
}

func postMigrationDDLStatements() []string {
	return []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tareas_proyecto_blueprint_key ON tareas(proyecto_id, blueprint_key) WHERE blueprint_key != ''`,
	}
}

func postMigrationStatementsForDriver(driver string) []string {
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver == "postgres" || driver == "postgresql" || driver == "mysql" {
		return nil
	}
	stmts := append([]string{}, postMigrationStatements()...)
	return append(stmts, postMigrationDDLStatements()...)
}
