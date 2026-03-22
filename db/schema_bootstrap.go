package db

import (
	"database/sql"
	"fmt"
	"strings"
)

type schemaSeedGroup struct {
	table   string
	columns []string
	rows    [][]string
}

var schemaSeedGroups = []schemaSeedGroup{
	{
		table:   "agentes",
		columns: []string{"nombre", "rol"},
		rows: [][]string{
			{"alberto", "admin"},
			{"claude", "programador"},
			{"codex1", "programador"},
			{"codex2", "programador"},
			{"antigravity", "documentador"},
		},
	},
	{
		table:   "config",
		columns: []string{"clave", "valor"},
		rows: [][]string{
			{"distribuidor", "claude"},
			{"version", "1.0.0"},
			{"pool_handoff_threshold_seconds", "1800"},
			{"pool_handoff_threshold_ratio", "0.10"},
			{"pool_default_budget_source", "manual"},
			{"model_policy_default_profile", "implementacion"},
			{"model_policy_default_reasoning", "high"},
		},
	},
	{
		table:   "reglas",
		columns: []string{"tipo_agente", "categoria", "titulo", "descripcion"},
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
}

func schemaDDL() string {
	return strings.TrimSpace(Schema)
}

func schemaSeedData() string {
	var b strings.Builder
	for _, group := range schemaSeedGroups {
		b.WriteString(renderSchemaSeedGroup(group))
	}
	return strings.TrimSpace(b.String())
}

func renderSchemaSeedGroup(group schemaSeedGroup) string {
	if len(group.columns) == 0 || len(group.rows) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("INSERT OR IGNORE INTO ")
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
