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
		`ALTER TABLE proyectos ADD COLUMN origen_repo TEXT NOT NULL DEFAULT 'local'`,
		`ALTER TABLE proyectos ADD COLUMN remote_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE proyectos ADD COLUMN branch_base TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE proyectos_autonomia ADD COLUMN supervisor_agente TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE proyectos_autonomia ADD COLUMN reviewer_agente TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tareas ADD COLUMN contrato_definido INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE tareas ADD COLUMN blueprint_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runtime_orders ADD COLUMN available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE runtime_orders ADD COLUMN claimed_by TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runtime_orders ADD COLUMN lease_token TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runtime_orders ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE runtime_orders ADD COLUMN lease_expires_at DATETIME`,
		`ALTER TABLE agentes ADD COLUMN retirado INTEGER NOT NULL DEFAULT 0`,
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
		`CREATE TABLE IF NOT EXISTS especificaciones_funcion (
			id                           INTEGER PRIMARY KEY AUTOINCREMENT,
			tarea_id                     INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
			proyecto_id                  INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			titulo                       TEXT    NOT NULL,
			archivo_objetivo             TEXT    NOT NULL,
			simbolo_objetivo             TEXT    NOT NULL,
			descripcion                  TEXT    NOT NULL DEFAULT '',
			precondiciones_json          TEXT    NOT NULL DEFAULT '[]',
			postcondiciones_json         TEXT    NOT NULL DEFAULT '[]',
			dependencias_permitidas_json TEXT    NOT NULL DEFAULT '[]',
			dependencias_prohibidas_json TEXT    NOT NULL DEFAULT '[]',
			tests_obligatorios_json      TEXT    NOT NULL DEFAULT '[]',
			write_set_json               TEXT    NOT NULL DEFAULT '[]',
			formato_salida               TEXT    NOT NULL DEFAULT 'patch+evidencia',
			estado                       TEXT    NOT NULL DEFAULT 'activa'
			                                     CHECK (estado IN ('borrador','activa','reemplazada','archivada')),
			version                      INTEGER NOT NULL DEFAULT 1,
			creado_por                   TEXT    NOT NULL DEFAULT 'alberto',
			created_at                   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at                   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_especificaciones_funcion_tarea ON especificaciones_funcion(tarea_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_especificaciones_funcion_proyecto_estado ON especificaciones_funcion(proyecto_id, estado, id DESC)`,
		`CREATE TABLE IF NOT EXISTS governance_overrides (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			tipo_agente TEXT    NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
			scope_tipo  TEXT    NOT NULL CHECK (scope_tipo IN ('proyecto','agente')),
			scope_ref   TEXT    NOT NULL,
			entidad     TEXT    NOT NULL CHECK (entidad IN ('regla','skill','workflow')),
			entidad_id  INTEGER NOT NULL,
			accion      TEXT    NOT NULL CHECK (accion IN ('enable','disable')),
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(scope_tipo, scope_ref, entidad, entidad_id)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_tareas_proyecto_blueprint_key ON tareas(proyecto_id, blueprint_key) WHERE blueprint_key != ''`,
		`CREATE INDEX IF NOT EXISTS idx_sesiones_activa_id ON sesiones(activa, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sesiones_agente_activa_id ON sesiones(agente, activa, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sesiones_agente_id ON sesiones(agente, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_asignaciones_agente_estado_proyecto_id ON asignaciones(agente, estado, proyecto_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_asignaciones_proyecto_estado_agente_id ON asignaciones(proyecto_id, estado, agente, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_presupuestos_sesion_sesion_checked_id ON presupuestos_sesion(sesion_id, checked_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_presupuestos_sesion_sesion_fuente_checked_id ON presupuestos_sesion(sesion_id, budget_source, checked_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_propuestas_estado_proyecto_id ON propuestas(estado, proyecto_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_votos_agente_posicion_propuesta ON votos(agente, posicion, propuesta_id)`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_mailbox_estado_id ON runtime_mailbox(estado, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tareas_agente_estado_id ON tareas(agente, estado, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tareas_estado_id ON tareas(estado, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_accion_id ON audit_log(accion, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_agente_id ON audit_log(agente, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_entidad_entidadid_id ON audit_log(entidad, entidad_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_instances_agente_updated_id ON runtime_instances(agente, updated_at DESC, id DESC)`,
		`CREATE TABLE IF NOT EXISTS shared_context_items (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto_id      INTEGER REFERENCES proyectos(id) ON DELETE CASCADE,
			agente           TEXT    NOT NULL DEFAULT '',
			tipo             TEXT    NOT NULL DEFAULT 'nota',
			titulo           TEXT    NOT NULL,
			detalle          TEXT    NOT NULL DEFAULT '',
			payload_json     TEXT    NOT NULL DEFAULT '{}',
			peso             REAL    NOT NULL DEFAULT 5,
			origen           TEXT    NOT NULL DEFAULT '',
			expires_at       DATETIME,
			created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shared_context_items_project_agent_weight_id ON shared_context_items(proyecto_id, agente, peso DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_shared_context_items_tipo_id ON shared_context_items(tipo, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_handles_estado_last_seen ON runtime_handles(estado, last_seen_at, id) WHERE estado IN ('activo','pausado')`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_handles_estado_id ON runtime_handles(estado, id)`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_mailbox_destino_estado_id ON runtime_mailbox(to_agente, estado, id DESC)`,
		`CREATE TABLE IF NOT EXISTS agente_scores_locales (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			agente            TEXT    NOT NULL REFERENCES agentes(nombre) ON DELETE CASCADE,
			conector_slug     TEXT    NOT NULL DEFAULT '',
			materia           TEXT    NOT NULL DEFAULT 'codigo',
			score_base        REAL    NOT NULL DEFAULT 5.0,
			score_observado   REAL    NOT NULL DEFAULT 5.0,
			score_total       REAL    NOT NULL DEFAULT 5.0,
			confianza         REAL    NOT NULL DEFAULT 0.0,
			muestras          INTEGER NOT NULL DEFAULT 0,
			exitos            INTEGER NOT NULL DEFAULT 0,
			benchmarks        INTEGER NOT NULL DEFAULT 0,
			metadata_json     TEXT    NOT NULL DEFAULT '{}',
			last_benchmark_at DATETIME,
			last_observed_at  DATETIME,
			created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(agente, conector_slug, materia)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_agente_scores_locales_agente_materia ON agente_scores_locales(agente, materia, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_agente_scores_locales_conector_materia ON agente_scores_locales(conector_slug, materia, score_total DESC, updated_at DESC)`,
	}
}

func postMigrationStatementsForDriver(driver string) []string {
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver == "postgres" || driver == "postgresql" {
		stmts := append([]string{}, postMigrationStatements()...)
		for i, stmt := range stmts {
			stmts[i] = renderDriverColumnSyntax("postgres", stmt)
		}
		return stmts
	}
	if driver == "mysql" {
		return nil
	}
	stmts := append([]string{}, postMigrationStatements()...)
	return append(stmts, postMigrationDDLStatements()...)
}
