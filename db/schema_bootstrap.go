package db

import (
	"database/sql"
	"fmt"
	"strings"

	"orquesta/storage"
)

type schemaSeedGroup struct {
	table           string
	columns         []string
	conflictColumns []string
	rows            [][]string
}

var schemaSeedGroups = []schemaSeedGroup{
	{
		table:           "agentes",
		columns:         []string{"nombre", "rol"},
		conflictColumns: []string{"nombre"},
		rows: [][]string{
			{"alberto", "admin"},
			{"claude", "programador"},
			{"codex1", "programador"},
			{"codex2", "programador"},
			{"antigravity", "documentador"},
		},
	},
	{
		table:           "config",
		columns:         []string{"clave", "valor"},
		conflictColumns: []string{"clave"},
		rows:            defaultConfigSeedRows(),
	},
	{
		table:           "reglas",
		columns:         []string{"tipo_agente", "categoria", "titulo", "descripcion"},
		conflictColumns: []string{"tipo_agente", "titulo"},
		rows: [][]string{
			{"programador", "financiero", "decimal.Decimal obligatorio", "Usar decimal.Decimal para cualquier cifra monetaria. Cero float32/float64 para importes."},
			{"programador", "financiero", "Redondeo HALF_UP", "Redondear siempre a 2 decimales con HALF_UP."},
			{"programador", "seguridad", "JWT RS256", "Autenticación con JWT RS256. Argon2id para contraseñas (memory=64MB, iterations=3, parallelism=2)."},
			{"programador", "seguridad", "Bloqueo tras 5 intentos", "Bloquear cuenta tras 5 intentos de autenticación fallidos."},
			{"programador", "seguridad", "RBAC completo", "Control de acceso en cada endpoint: user → AD groups → roles internos → permisos → acción."},
			{"programador", "seguridad", "Filtrado por tenant_id", "Filtrar siempre por tenant_id (BD por tenant). No usar sede_id como frontera de seguridad."},
			{"programador", "seguridad", "Auditoría obligatoria", "Toda mutación requiere auditoría: SHA-256 encadenado, retención 5 años, AES-256-GCM at rest."},
			{"programador", "seguridad", "Sin secretos hardcodeados", "Ningún secreto en código fuente. TLS obligatorio en todos los endpoints."},
			{"programador", "arquitectura", "Clean Architecture + DDD", "Capas: api → service → repository → domain. Interfaces primero."},
			{"programador", "arquitectura", "main.go intocable", "main.go es intocable por agentes de módulo. Solo el agente de infraestructura lo integra."},
			{"programador", "arquitectura", "Multi-tenant aislado", "1 ayuntamiento = 1 base de datos PostgreSQL aislada. Patrón H-72: tenantDB(ctx)."},
			{"programador", "arquitectura", "Propiedad exclusiva de ficheros", "Cada módulo tiene propiedad exclusiva de sus ficheros. No editar ficheros de otro módulo."},
			{"programador", "arquitectura", "Módulos opcionales", "Todos los módulos deben ser opcionales vía variable MODULES. Disabled = sin instanciación."},
			{"programador", "calidad", "Commits frecuentes", "Commit tras cada bloque probado: entidades / repositorio / servicio / handler / migración. Formato: feat(MXX): descripción."},
			{"programador", "calidad", "Gate antes de commit", "Antes de cada commit: go test ./internal/... debe pasar."},
			{"programador", "calidad", "Gate de cierre de módulo", "Al cerrar módulo: go build ./..., go vet ./..., go test ./... deben pasar."},
			{"programador", "calidad", "Propuesta antes de código", "No escribir código sin propuesta OP-XXX aprobada en la app de orquestación."},
			{"programador", "calidad", "Idioma castellano", "Todo en castellano. Inglés solo cuando lo exija framework, librería o protocolo."},
			{"programador", "calidad", "Cabecera GPLv3", "Incluir cabecera de licencia GPLv3 en todos los ficheros nuevos."},
			{"programador", "sesion", "Fuente de verdad: BD de orquesta", "Reglas, skills y workflows viven en la BD de orquesta — no en ficheros. Ejecutar siempre \"orquesta sesion inicio <nombre>\" al comenzar: muestra el briefing completo desde la BD."},
			{"programador", "sesion", "Protocolo de inicio", "Paso 1: orquesta sesion inicio <nombre>  |  Paso 2: ver tareas asignadas (orquesta tarea listar --agente <nombre>)  |  Paso 3: votar propuestas pendientes  |  Paso 4: registrar tarea antes de tocar código."},
			{"programador", "sesion", "Protocolo de fin", "Paso 1: git status limpio (todo commiteado)  |  Paso 2: orquesta sesion fin <nombre>."},
			{"programador", "sesion", "Panel web de orquestación", "Alberto arranca el panel web con \"orquesta serve\" (http://localhost:8080). Muestra en tiempo real: agentes activos, progreso de tareas, propuestas abiertas con votos. Se auto-refresca cada 30 s. Secciones: Dashboard, Tareas (con filtros por estado), Propuestas (expandibles con votos). Los agentes NO necesitan arrancarlo; es para Alberto y para generar capturas de estado."},
			{"programador", "comandos", "Gestión de tareas", "Iniciar: orquesta tarea iniciar <id> <agente>  |  Completar: orquesta tarea completar <id> <agente> --commit \"feat(MXX): ...\"  |  Bloquear: orquesta tarea bloquear <id> <agente> --motivo \"razón\"  |  Desbloquear: orquesta tarea desbloquear <id> <agente> --resolucion \"cómo se resolvió\"  |  Ver mis tareas: orquesta tarea listar --agente <nombre>"},
			{"programador", "comandos", "Propuestas y votación", "Nueva propuesta: orquesta propuesta nueva \"Título\" --descripcion \"Descripción\" --agente <nombre>  |  Votar: orquesta votar <OP-XXX> <acuerdo|desacuerdo|abstencion> --agente <nombre> --comentario \"razón\"  |  Ver propuesta: orquesta propuesta ver <OP-XXX>  |  Ver pendientes de voto: incluidas en el briefing de sesion inicio."},
			{"documentador", "general", "Fuente de verdad: BD de orquesta", "Reglas, skills y workflows viven en la BD — no en ficheros. Ejecutar \"orquesta sesion inicio antigravity\" al comenzar para recibir el briefing completo."},
			{"documentador", "general", "Solo documentación", "Antigravity no toca código fuente (.go, .sql, .ts, .yaml). Solo docs/."},
			{"documentador", "general", "Ficheros asignados", "Propiedad exclusiva: docs/modulos/MXX_*.md e índice maestro docs/00_INDICE.md."},
			{"documentador", "general", "Activación por notificación", "Documentar un módulo solo tras notificación explícita del agente programador que lo cierra."},
			{"documentador", "general", "Idioma castellano", "Toda la documentación en castellano. Términos técnicos en inglés solo si no tienen traducción."},
			{"documentador", "sesion", "Protocolo de inicio", "Paso 1: orquesta sesion inicio antigravity  |  Paso 2: identificar módulos cerrados sin documentar  |  Paso 3: ver tareas asignadas."},
			{"documentador", "sesion", "Protocolo de fin", "Paso 1: actualizar docs/00_INDICE.md si hay módulos nuevos  |  Paso 2: orquesta sesion fin antigravity."},
		},
	},
	{
		table:           "skills",
		columns:         []string{"tipo_agente", "nombre", "descripcion", "cuando_usar"},
		conflictColumns: []string{"tipo_agente", "nombre"},
		rows: [][]string{
			{"programador", "create-module", "Crear un módulo nuevo completo (14 pasos): dominio, repositorio, servicio, handler, migración, tests, integración.", "Cuando se asigna un módulo nuevo de la lista de Ola 2 o posterior."},
			{"programador", "develop-feature", "Desarrollar una feature dentro de un módulo existente sin romper otros módulos.", "Cuando hay una tarea de nueva funcionalidad dentro de un módulo ya existente."},
			{"programador", "fix-bug", "Corregir un bug con mínimo impacto, sin refactorizar código no relacionado.", "Cuando hay un bug confirmado con reproducción conocida."},
			{"programador", "security-review", "Auditar seguridad de un módulo: RBAC, cifrado, auditoría, inyección SQL, OWASP Top 10.", "Al cerrar un módulo o cuando se detecta deuda técnica de seguridad (ej: OP-022)."},
			{"programador", "autofirma-integration", "Integrar firma digital AutoFirma según estándar @firma de la AEAT.", "Cuando se implementa firma electrónica en flujos administrativos. Ver OP-027 (📋 BACKLOG)."},
			{"programador", "administracion-publica-segura", "Referencia normativa ENS/LOPDGDD para diseño de módulos de administración pública.", "Al diseñar módulos con datos personales o procesos sujetos a ENS."},
			{"documentador", "document-module", "Generar documentación completa de un módulo cerrado: descripción, endpoints, entidades, flujos.", "Al recibir notificación de cierre de módulo de un agente programador."},
			{"documentador", "update-index", "Actualizar el índice maestro docs/00_INDICE.md con nuevos módulos documentados.", "Después de documentar cualquier módulo."},
			{"documentador", "review-docs", "Revisar documentación existente por coherencia, completitud y actualidad.", "Cuando se detectan discrepancias entre el código y la documentación."},
		},
	},
	{
		table:           "workflows",
		columns:         []string{"tipo_agente", "nombre", "descripcion", "pasos"},
		conflictColumns: []string{"nombre"},
		rows: [][]string{
			{"programador", "inicio-sesion", "Protocolo obligatorio al comenzar cualquier sesión de trabajo.", `["1. Ejecutar: orquesta sesion inicio <mi-nombre>","2. Ver tareas asignadas: orquesta tarea listar --agente <mi-nombre>","3. Votar todas las propuestas con posicion pendiente para mi agente","4. Iniciar la tarea en la app: orquesta tarea iniciar <id> <mi-nombre>","5. Leer el doc del módulo asignado en docs/modulos/MXX_*.md"]`},
			{"programador", "fin-sesion", "Protocolo obligatorio al terminar cualquier sesión de trabajo.", `["1. Asegurar que todo el trabajo está commiteado (git status limpio)","2. Completar o bloquear mis tareas en la app de orquestación según corresponda","3. Ejecutar: orquesta sesion fin <mi-nombre>"]`},
			{"programador", "crear-modulo", "Flujo completo para implementar un módulo nuevo (14 pasos).", `["1. Crear propuesta OP-XXX con orquesta propuesta nueva y esperar consenso en la app","2. Crear fichero de dominio: internal/domain/<modulo>_entities.go","3. Crear interfaces: internal/domain/<modulo>_interfaces.go","4. Crear migración SQL: migrations/XXXXXX_<modulo>.up.sql","5. Crear repositorio: internal/repository/postgres_<modulo>.go","6. Commit: feat(MXX): dominio e interfaces","7. Crear servicio: internal/service/<modulo>_service.go","8. Commit: feat(MXX): servicio","9. Crear tests: internal/service/<modulo>_service_test.go","10. Ejecutar go test ./internal/service/... → debe pasar","11. Commit: feat(MXX): tests servicio","12. Crear handler REST: internal/api/<modulo>_handler.go","13. Commit: feat(MXX): handler REST","14. Gate final: go build ./... && go vet ./... && go test ./... → notificar a Antigravity"]`},
			{"programador", "votar-propuesta", "Proceso para votar una propuesta OP-XXX.", `["1. Leer la propuesta completa: orquesta propuesta ver <codigo>","2. Analizar impacto técnico en módulos asignados","3. Votar: orquesta votar <codigo> <acuerdo|desacuerdo|abstencion> --agente <mi-nombre> --comentario \"razón\"","4. Si desacuerdo: añadir comentario técnico con alternativa concreta"]`},
			{"documentador", "inicio-sesion", "Protocolo obligatorio al comenzar cualquier sesión de trabajo.", `["1. Ejecutar: orquesta sesion inicio antigravity","2. Ver tareas asignadas: orquesta tarea listar --agente antigravity","3. Votar propuestas con posicion pendiente para antigravity","4. Revisar docs/00_INDICE.md para detectar gaps"]`},
			{"documentador", "fin-sesion", "Protocolo obligatorio al terminar cualquier sesión de trabajo.", `["1. Actualizar docs/00_INDICE.md si se añadieron módulos","2. Confirmar con el agente programador que la documentación es correcta","3. Ejecutar: orquesta sesion fin antigravity"]`},
			{"documentador", "documentar-modulo", "Flujo para documentar un módulo cerrado.", `["1. Recibir notificación del agente programador con el código del módulo","2. Leer el código fuente del módulo (solo lectura, nunca editar)","3. Crear docs/modulos/MXX_<nombre>.md con: descripción, entidades, endpoints, flujos, seguridad","4. Actualizar docs/00_INDICE.md","5. Notificar al programador que la documentación está lista"]`},
		},
	},
}

func schemaDDL() string {
	return strings.TrimSpace(Schema)
}

func schemaSeedData() string {
	return schemaSeedDataForDriver(DriverName())
}

func schemaSeedDataForDriver(driver string) string {
	var b strings.Builder
	for _, group := range schemaSeedGroups {
		b.WriteString(renderSchemaSeedGroupForDriver(driver, group))
	}
	return strings.TrimSpace(b.String())
}

func renderSchemaSeedGroup(group schemaSeedGroup) string {
	return renderSchemaSeedGroupForDriver(DriverName(), group)
}

func renderSchemaSeedGroupForDriver(driver string, group schemaSeedGroup) string {
	if len(group.columns) == 0 || len(group.rows) == 0 {
		return ""
	}

	dialect := storage.DialectForDriver(driver)
	var b strings.Builder
	if dialect.Name == "mysql" {
		b.WriteString("INSERT IGNORE INTO ")
	} else {
		b.WriteString("INSERT INTO ")
	}
	b.WriteString(group.table)
	b.WriteString(" (")
	b.WriteString(strings.Join(group.columns, ", "))
	b.WriteString(") VALUES\n")
	for i, row := range group.rows {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("    (")
		for j, value := range row {
			if j > 0 {
				b.WriteString(", ")
			}
			b.WriteString(quoteSchemaSeedValue(value))
		}
		b.WriteString(")")
	}
	if dialect.Name != "mysql" && len(group.conflictColumns) > 0 {
		b.WriteString("\nON CONFLICT(")
		b.WriteString(strings.Join(group.conflictColumns, ", "))
		b.WriteString(") DO NOTHING")
	}
	b.WriteString(";\n\n")
	return b.String()
}

func quoteSchemaSeedValue(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func aplicarSchema(db *sql.DB) error {
	ddl := schemaDDL()
	if ddl == "" {
		return fmt.Errorf("schema DDL vacio")
	}
	_, err := db.Exec(ddl)
	return err
}

func aplicarSemillasSchema(db *sql.DB) error {
	seeds := schemaSeedData()
	if seeds == "" {
		return nil
	}
	_, err := db.Exec(seeds)
	return err
}
