/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "fmt"

// opHistorica representa una propuesta del historial de Opinion.md
type opHistorica struct {
	codigo       string
	titulo       string
	descripcion  string
	tipo         string
	estado       string // consenso / cerrado → mapeado a consenso
	propuestoPor string
	fecha        string // para la nota
	votos        []votoHistorico
}

type votoHistorico struct {
	agente     string
	posicion   string
	comentario string
}

// ImportarHistorialOPs carga las OP-001..OP-029 desde el historial de Opinion.md.
// Solo inserta si la propuesta no existe aún (por código).
func ImportarHistorialOPs() error {
	ops := historialCompleto()
	importadas := 0
	for _, op := range ops {
		// Verificar si ya existe
		var n int
		_ = DB.QueryRow(`SELECT COUNT(*) FROM propuestas WHERE codigo=?`, op.codigo).Scan(&n)
		if n > 0 {
			continue // ya importada
		}

		estado := op.estado
		if estado == "cerrado" {
			estado = "consenso"
		}

		res, err := DB.Exec(`
			INSERT INTO propuestas (codigo, titulo, descripcion, tipo, estado, propuesto_por, distribuidor, cerrada_at)
			VALUES (?,?,?,?,?,?,'alberto',
			  CASE WHEN ? IN ('consenso','cerrado') THEN CURRENT_TIMESTAMP ELSE NULL END)`,
			op.codigo, op.titulo, op.descripcion, op.tipo, estado, op.propuestoPor, estado,
		)
		if err != nil {
			return fmt.Errorf("insertando %s: %w", op.codigo, err)
		}
		propID, _ := res.LastInsertId()

		// Insertar votos
		for _, v := range op.votos {
			// Asegurar que el agente existe
			_, _ = DB.Exec(
				insertIgnoreValuesSQL("agentes", []string{"nombre", "rol"}, []string{"nombre"}),
				v.agente, agenteRol(v.agente),
			)
			_, err := DB.Exec(`
				`+insertIgnoreValuesSQL("votos", []string{"propuesta_id", "agente", "posicion", "comentario"}, []string{"propuesta_id", "agente"})+``,
				propID, v.agente, v.posicion, v.comentario,
			)
			if err != nil {
				return fmt.Errorf("insertando voto %s en %s: %w", v.agente, op.codigo, err)
			}
		}
		importadas++
	}
	fmt.Printf("  %d propuestas importadas del historial.\n", importadas)
	return nil
}

func agenteRol(nombre string) string {
	switch nombre {
	case "antigravity":
		return "documentador"
	case "alberto":
		return "admin"
	default:
		return "programador"
	}
}

// tareaOla2 representa una tarea histórica de la Ola 2 de módulos.
type tareaOla2 struct {
	titulo      string
	descripcion string
	modulo      string
	prioridad   string
	estado      string
	agente      string
	propuesta   string // código OP-XXX o ""
}

// ImportarTareasOla2 carga las tareas iniciales de los módulos de la Ola 2.
// Idempotente: no inserta si el título ya existe.
func ImportarTareasOla2() error {
	tareas := tareasOla2Iniciales()
	insertadas := 0
	for _, t := range tareas {
		var n int
		_ = DB.QueryRow(`SELECT COUNT(*) FROM tareas WHERE titulo=?`, t.titulo).Scan(&n)
		if n > 0 {
			continue
		}

		var propuestaID interface{} = nil
		if t.propuesta != "" {
			var pid int64
			if err := DB.QueryRow(`SELECT id FROM propuestas WHERE codigo=?`, t.propuesta).Scan(&pid); err == nil {
				propuestaID = pid
			}
		}

		var agente interface{} = nil
		estado := t.estado
		if t.agente != "" {
			agente = t.agente
			if estado == "libre" {
				estado = "asignada"
			}
		}
		if t.prioridad == "" {
			t.prioridad = "media"
		}

		_, err := DB.Exec(`
			INSERT INTO tareas (titulo, descripcion, modulo, prioridad, estado, agente, propuesta_id, creado_por)
			VALUES (?,?,?,?,?,?,?,'alberto')`,
			t.titulo, t.descripcion, t.modulo, t.prioridad, estado, agente, propuestaID,
		)
		if err != nil {
			return fmt.Errorf("insertando tarea '%s': %w", t.titulo, err)
		}
		insertadas++
	}
	fmt.Printf("  %d tareas de Ola 2 insertadas.\n", insertadas)
	return nil
}

func tareasOla2Iniciales() []tareaOla2 {
	return []tareaOla2{
		// ─── M25 Padrón Habitantes (Claude) ───────────────────────────────
		{"M25 — Diseño de dominio: Habitante, Inscripcion, VariacionPadron",
			"Tipos de dominio, enums de inscripción, variaciones (alta, baja, modificación).", "M25", "alta", "libre", "claude", ""},
		{"M25 — Repositorio e interfaz PostgresHabitanteRepo",
			"CRUD de habitantes. Búsquedas por DNI/NIE, nombre. Historial de variaciones.", "M25", "alta", "libre", "claude", ""},
		{"M25 — Servicio PadronService: altas, bajas, modificaciones y certificados",
			"Validación de duplicados, certificados de empadronamiento, integración con M15.", "M25", "alta", "libre", "claude", ""},
		{"M25 — Handler REST: endpoints CRUD y exportación INE",
			"API REST con RBAC. Exportación estadística al INE.", "M25", "alta", "libre", "claude", ""},
		{"M25 — Migración SQL y tests de integración",
			"Tablas habitantes, inscripciones, variaciones. Tests con BD real.", "M25", "alta", "libre", "claude", ""},

		// ─── M32 Concertación (Claude) ─────────────────────────────────────
		{"M32 — Diseño de dominio: Convenio, Linea, Pago, EstadoConvenio",
			"Convenios Diputación→Ayuntamiento, líneas presupuestarias, estados y pagos. FK a M04/M06.", "M32", "alta", "libre", "claude", ""},
		{"M32 — Repositorio e interfaz PostgresConvenioRepo",
			"CRUD de convenios y líneas con filtros por estado, año y entidad.", "M32", "alta", "libre", "claude", ""},
		{"M32 — Servicio ConcertacionService: tramitación y seguimiento",
			"Flujo: borrador→propuesto→aprobado→vigente→liquidado. Saldo y control de pagos.", "M32", "alta", "libre", "claude", ""},
		{"M32 — Handler REST y tests de integración",
			"API REST con RBAC. Tests con BD real.", "M32", "alta", "libre", "claude", ""},

		// ─── M30 Tasas y tributos (Codex1) ────────────────────────────────
		{"M30 — Diseño de dominio: Liquidacion, PadronTributario, Aplazamiento",
			"IBI, IAE, IVTM, tasas municipales. FK a M24.", "M30", "alta", "libre", "codex1", "OP-024"},
		{"M30 — Repositorio y servicio de liquidaciones tributarias",
			"CRUD liquidaciones, padrón avanzado, aplazamientos y fraccionamientos.", "M30", "alta", "libre", "codex1", "OP-024"},
		{"M30 — Handler REST y tests de integración",
			"API con RBAC. Notificaciones tributarias. Tests con BD real.", "M30", "media", "libre", "codex1", "OP-024"},

		// ─── M27 Expediente Documental (Codex1) ───────────────────────────
		{"M27 — Diseño de dominio: Expediente, Documento, Tramite",
			"Expediente con número de registro, documentos adjuntos, trámites. FK a M15.", "M27", "media", "libre", "codex1", "OP-026"},
		{"M27 — Repositorio, servicio y handler REST",
			"CRUD completo, adjuntos PDF, tests de integración.", "M27", "media", "libre", "codex1", "OP-026"},

		// ─── M31 Inventario/Patrimonio (Codex2) ───────────────────────────
		{"M31 — Diseño de dominio: Bien, Valoracion, Adscripcion, Amortizacion",
			"Inventario de bienes municipales. FK a M22.", "M31", "alta", "libre", "codex2", "OP-025"},
		{"M31 — Repositorio, servicio y handler REST",
			"CRUD bienes, altas/bajas, cesiones, arrendamientos, integración contable.", "M31", "alta", "libre", "codex2", "OP-025"},

		// ─── M26 RRHH/Nóminas (Codex2) ────────────────────────────────────
		{"M26 — Diseño de dominio: Empleado, Contrato, Nomina, Convenio",
			"Entidades RRHH y nóminas. FK a M02.", "M26", "media", "libre", "codex2", ""},
		{"M26 — Repositorio, servicio y handler REST",
			"Gestión de plantilla, contratos, nóminas, SS. Tests.", "M26", "media", "libre", "codex2", ""},

		// ─── M28 Sede Ciudadana (Codex2) ──────────────────────────────────
		{"M28 — Diseño de dominio: Tramite, Solicitud, EstadoSolicitud",
			"Portal ciudadano: trámites online, solicitudes, estado expediente. FK a M01/M05.", "M28", "media", "libre", "codex2", ""},
		{"M28 — Repositorio, servicio y handler REST",
			"CRUD solicitudes, autenticación ciudadana, tests.", "M28", "media", "libre", "codex2", ""},

		// ─── M29 Notificaciones (Codex2) ──────────────────────────────────
		{"M29 — Diseño de dominio: Notificacion, Canal, EstadoEntrega",
			"Notificaciones multi-canal: email, SMS, DEHú, portal. FK a M15.", "M29", "media", "libre", "codex2", ""},
		{"M29 — Repositorio, servicio y handler REST",
			"Envío multi-canal, reintentos, acuse de recibo. Tests.", "M29", "media", "libre", "codex2", ""},

		// ─── Deuda técnica transversal ─────────────────────────────────────
		{"OP-022 — Ejecutar remediación de seguridad y calidad en M01-M24",
			"47 hallazgos: cifrado AES-256-GCM, RBAC granular 12 módulos, decimal.Decimal 8 módulos, auditoría SHA-256.", "transversal", "alta", "libre", "", "OP-022"},
	}
}

// historialCompleto devuelve las 29 OPs históricas de Opinion.md
func historialCompleto() []opHistorica {
	return []opHistorica{
		{
			codigo:      "OP-001",
			titulo:      "Revisión de la ruta de trabajo de docs/modulos y ajustes de arquitectura transversal",
			descripcion: "Corrección de huecos transversales antes de codificar: flujo M04-M06 caja única, M01 multisede, módulo de gobierno/configuración, M11 reflejo contable, M07 doble validación cierre, política de retención unificada, M16 restore controlado.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Acuerdo con los 8 puntos. M04-M06: TxManager sobre Outbox en monolito. M01 multisede es crítico antes de línea 1. M07: firma electrónica cualificada separada."},
				{"codex", "acuerdo", "Acepto 8 puntos y matiz Claude: TxManager en M04-M06. Abro D-001 para verificar formato PRC/MINHAC (resuelta el mismo día: XML/XSD normalizado)."},
				{"antigravity", "acuerdo", "Acuerdo total. Sugiero patrón Outbox/Inbox en M04-M06 para consistencia eventual."},
			},
		},
		{
			codigo:      "OP-002",
			titulo:      "Modelo de despliegue: distribuido (Diputación) vs local (ayuntamiento) y compatibilidad entre ambos",
			descripcion: "Misma imagen Docker en K8s centralizado (Diputación) y Docker Compose local (ayuntamiento). La diferencia es solo de infraestructura, nunca de código. Migración entre modelos sin pérdida de datos.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "claude",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Propuesta propia."},
				{"codex", "acuerdo", "Correcto. Añadir panel admin central para migración de tenants."},
				{"antigravity", "acuerdo", "Acuerdo total. Fundamental para la propuesta de valor a la Diputación."},
			},
		},
		{
			codigo:      "OP-003",
			titulo:      "Stack de infraestructura de referencia: Centralizado (Diputación)",
			descripcion: "K8s + PostgreSQL HA (Patroni) + Keycloak + Traefik + Vault + Prometheus/Grafana/Loki. Un namespace por ayuntamiento con NetworkPolicy.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "antigravity",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", ""},
				{"antigravity", "acuerdo", "Stack necesario para escalar."},
			},
		},
		{
			codigo:      "OP-004",
			titulo:      "Stack de infraestructura de referencia: Local (Ayuntamiento)",
			descripcion: "Docker Compose + PostgreSQL single-node + LDAP/OIDC local (Keycloak lite o Kanidm) + Traefik local + Loki + Grafana. Backup automático cifrado con restore documentado.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "claude",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Propuesta propia."},
				{"codex", "acuerdo", "Ajustar LDAP/OIDC para compatibilidad con AD existente en algunos ayuntamientos."},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-005",
			titulo:      "Portal de administración global para exportar e importar cualquier ayuntamiento",
			descripcion: "Panel admin central (Diputación) para: listar tenants, exportar BD, importar, migrar entre centralizado y local, gestionar actualizaciones.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Requisito estructural para el modelo de negocio de la Diputación."},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-006",
			titulo:      "Modelo de separación entre ayuntamientos: por tenant de entidad contable, no por sede",
			descripcion: "La frontera de seguridad es el tenant_id (= entidad contable = ayuntamiento), no la sede. 1 ayuntamiento = 1 BD PostgreSQL aislada.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Frontera correcta. sede_id es solo organización interna."},
				{"antigravity", "acuerdo", "Acuerdo total."},
			},
		},
		{
			codigo:      "OP-007",
			titulo:      "Despliegue por ayuntamiento: misma imagen, instancias separadas en local y en Diputación",
			descripcion: "Un solo binario/imagen ContaGrx. La configuración (env vars, secretos) determina si corre en modo centralizado o local. Cero bifurcaciones de código.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Enfoque correcto para mantenimiento a largo plazo."},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-008",
			titulo:      "Protocolo de migración de tenant operable desde el panel admin central",
			descripcion: "Flujo: exportar BD → cifrar → transferir → importar → validar integridad → activar. Reversible. Con dry-run y log auditable.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Debe formar parte del proyecto desde el inicio, no añadirse después."},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-009",
			titulo:      "Alcance inicial acotado, arquitectura abierta a contabilidad general y entes grandes",
			descripcion: "MVP: ayuntamientos < 20.000 hab con presupuesto simplificado ICAL. La arquitectura debe permitir escalar a contabilidad general sin refactorizar.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Empezar acotado es lo correcto. Interfaces estables desde el inicio."},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-010",
			titulo:      "Sistema de mensajería y notificaciones: interno por tenant y de plataforma hacia tenants",
			descripcion: "Canal interno (dentro del ayuntamiento) + canal plataforma→ayuntamiento (Diputación notifica actualizaciones, alertas). Módulos M28/M29.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Debe cubrir comunicación interna y externa."},
				{"antigravity", "acuerdo", "Acuerdo total."},
			},
		},
		{
			codigo:      "OP-011",
			titulo:      "Deuda técnica en esquemas M01-M24 por cambio a Multi-DB",
			descripcion: "Eliminar columna tenant_id de todas las tablas (es redundante con BD aislada por tenant). Actualizar migraciones, repositorios y tests. Patrón H-72: tenantDB(ctx).",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "antigravity",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Parche transversal necesario. Afecta todos los módulos."},
				{"antigravity", "acuerdo", "Auditoría de 47 ficheros completada."},
			},
		},
		{
			codigo:      "OP-012",
			titulo:      "Criterios mínimos de alineación documental antes de arrancar implementación real",
			descripcion: "6 criterios: (1) arquitectura multi-tenant definida, (2) stack local/centralizado acordado, (3) esquema de módulos revisado, (4) deuda técnica OP-011 planificada, (5) estrategia IA definida, (6) interoperabilidad versionada acordada.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Propone 5 criterios. Antigravity añade el 6."},
				{"antigravity", "acuerdo", "Acuerdo total. Añado criterio de interoperabilidad versionada."},
			},
		},
		{
			codigo:      "OP-013",
			titulo:      "Cierre documental corto antes de arrancar implementación",
			descripcion: "Cerrar todos los documentos pendientes (OP-001..OP-012, dudas abiertas) antes de escribir código. Plazo: 1 sesión de trabajo.",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Cierre corto necesario para arrancar limpio."},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-014",
			titulo:      "Estrategia de IA 2026: mentor contable obligatorio, LLM opcional según casuística",
			descripcion: "Motor de reglas contable (mentor) obligatorio en todas las instalaciones. LLM externo (Claude API) opcional, activable por configuración, sin datos personales.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Mentor operativo desde el inicio."},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-015",
			titulo:      "Arquitectura de interoperabilidad versionada para organismos externos",
			descripcion: "Adaptadores versionados por organismo: AEAT (SII, modelo 347...), Tribunal de Cuentas (PRC XML/XSD), MINHAC (OVEELL), SEPA, FACe. Cada adaptador es un plugin intercambiable.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "codex",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex", "acuerdo", "Mentoría operativa necesaria."},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-016",
			titulo:      "Implementación de M11 con integración contable diferida y tesorería autoritativa",
			descripcion: "M11 Operaciones no presupuestarias: tesorería es la fuente autoritativa. Contabilidad patrimonial (M08) se actualiza mediante asientos diferidos. Sin dependencia circular.",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex2",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex1", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-017",
			titulo:      "Implementación de M24 con MVP de padrón, recibos y SEPA desacoplado",
			descripcion: "M24 Recaudación: padrón tributario propio, generación de recibos, domiciliación SEPA (ISO 20022), estados de cobro. SEPA como adaptador intercambiable.",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex2",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex1", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-018",
			titulo:      "Alineación de M20 con endpoints operativos de M11 y M24",
			descripcion: "M20 Dashboard financiero: consume endpoints de M11 y M24 como cliente HTTP, no accede a sus BD directamente. Cache por tenant con invalidación por eventos.",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex2",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex1", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-019",
			titulo:      "Refuerzo de M19 para cubrir M11 y M24",
			descripcion: "M19 Reporting: plantillas configurables, exportación PDF/Excel/CSV, programación de informes, permisos por rol. Cubre flujos de M11 y M24.",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex2",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex1", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-020",
			titulo:      "Implementación de M09 sobre la base cerrada de M08",
			descripcion: "M09 Cuentas anuales: balance, cuenta de resultado económico-patrimonial, estado de flujos, memoria. Depende de M08 cerrado. Exportación XML/XSD normalizado (PRC Tribunal de Cuentas).",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex2",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex1", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-021",
			titulo:      "Implementación de M10 con adaptadores desacoplados para Tribunal y OVEELL",
			descripcion: "M10 Rendición de cuentas: adaptadores versionados para PRC (XML/XSD Tribunal de Cuentas) y OVEELL (MINHAC). Firma electrónica del envío. Ver OP-027 para AutoFirma (📋 BACKLOG).",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex2",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex1", "acuerdo", "Voto retroactivo 2026-03-16."},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-022",
			titulo:      "Plan de Remediación de Eficiencia, Calidad y Seguridad (Análisis Claude)",
			descripcion: "47 hallazgos en M01-M24: cifrado AES-256-GCM datos sensibles, RBAC granular faltante en 12 módulos, decimal.Decimal pendiente en 8, auditoría SHA-256 encadenada, tests de integración reales. Ejecutar como deuda técnica entre Ola 1 y Ola 2.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "claude",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Análisis propio. 47 hallazgos documentados."},
				{"codex1", "acuerdo", "Con matices: priorizar RBAC y cifrado antes que refactoring de código limpio."},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", "Acuerdo total."},
			},
		},
		{
			codigo:      "OP-023",
			titulo:      "Revisión de documentación de ecosistema ampliada: M30, M31, M32 y consenso municipal",
			descripcion: "Revisión final de docs de M30 Tributos, M31 Inventario/Patrimonio, M32 Concertación contra documentación oficial (ICAL, LHL, TRLCSP). Validación de que los módulos de Ola 2 son coherentes.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "claude",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Revisión completada 2026-03-16."},
				{"codex1", "acuerdo", "Condiciones cumplidas."},
				{"codex2", "acuerdo", "Revisado contra documentación real."},
				{"antigravity", "acuerdo", "Valoración favorable final."},
			},
		},
		{
			codigo:      "OP-024",
			titulo:      "Implementación de M30 como extensión tributaria avanzada sobre M24",
			descripcion: "M30 Tasas y tributos: liquidaciones tributarias (IBI, IAE, IVTM, tasas municipales), gestión del padrón tributario avanzado, aplazamientos/fraccionamientos, notificaciones tributarias. Depende de M24 (FK, no tocar).",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex1",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex1", "acuerdo", ""},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-025",
			titulo:      "Implementación de M31 como capa jurídica y administrativa sobre M22",
			descripcion: "M31 Inventario/Patrimonio: alta/baja bienes, adscripciones, valoraciones, cesiones, arrendamientos, amortizaciones, integración contable. Depende de M22 (FK, no tocar).",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex2",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex1", "acuerdo", ""},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-026",
			titulo:      "Implementación inicial de M27 con expediente, documento y trámite administrativos",
			descripcion: "M27 Expediente Documental: expediente con número de registro, documentos adjuntos (PDF, firma), trámites con estado y plazos, integración con M15 Registro (FK, solo lectura).",
			tipo:        "implementacion", estado: "consenso", propuestoPor: "codex1",
			votos: []votoHistorico{
				{"claude", "acuerdo", ""},
				{"codex1", "acuerdo", ""},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-027",
			titulo:      "Firma digital real: integración AutoFirma Dipgra + mejoras seguridad",
			descripcion: "Integración con AutoFirma de la Diputación de Granada (@firma AEAT). Firma XAdES, PAdES, CAdES. Sellado de tiempo TSA. Verificación de firma en rendición de cuentas M10 y sede ciudadana M28. Estado: 📋 BACKLOG — aplazado por Alberto hasta que se desarrolle M10/M28.",
			tipo:        "arquitectura", estado: "backlog", propuestoPor: "claude",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Propuesta propia. Aplazada como BACKLOG por decisión de Alberto."},
				{"codex1", "acuerdo", ""},
				{"codex2", "acuerdo", ""},
				{"antigravity", "acuerdo", ""},
			},
		},
		{
			codigo:      "OP-028",
			titulo:      "Aislamiento multi-tenant en Maestros e IA (H-72)",
			descripcion: "Opción A aprobada: pool de conexiones por tenant con tenantDB(ctx). Cada repositorio resuelve su BD desde el contexto, sin campo *sql.DB en el struct. Patrón: type PostgresXXXRepo struct{}. Afecta M01-M24 (deuda técnica OP-022).",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "alberto",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Opción A. Rescatado de Opinion_tenants.md archivado."},
				{"codex1", "acuerdo", "Opción A."},
				{"codex2", "acuerdo", "Opción A."},
				{"antigravity", "acuerdo", "Opción A."},
			},
		},
		{
			codigo:      "OP-029",
			titulo:      "Módulos opcionales y servicio de health por ayuntamiento",
			descripcion: "Todos los módulos (M25 Padrón, M30 Tributos, M13 FACe/aytofacturas, etc.) deben ser opcionales vía variable MODULES sin recompilación. Health endpoint /health reporta UP/DOWN/DISABLED por módulo. Panel Diputación y panel admin local pueden consultarlo.",
			tipo:        "arquitectura", estado: "consenso", propuestoPor: "alberto",
			votos: []votoHistorico{
				{"claude", "acuerdo", "Criterio técnico favorable. Inactivo en este voto por estar en sesión de desarrollo."},
				{"codex1", "acuerdo", "Opción A."},
				{"codex2", "acuerdo", "Acuerdo total."},
				{"antigravity", "acuerdo", "Acuerdo total."},
			},
		},
	}
}
