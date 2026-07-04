# Diseño: wizard de programación conversacional (TAREA-7/7b)

Fecha: 2026-07-04. Autor: Claude (director). Implementa: Orquesta/Codex.
Estado: diseño aprobado por el operador; implementacion base G1+G2 integrada
por Codex en `modulos/orquesta-web`, `modulos/orquesta-mcp` y
`modulos/orquesta-app-codex-stack` el 2026-07-04. No tratar como cierre total:
queda pendiente la ampliacion de taxonomia universal U1-U12 y packs de dominio
definida al final de este documento.

Orden del operador: partir de "quiero una app para una agenda" y llegar a la
app completa preguntando por los huecos, mostrando SIEMPRE la mejor opción y
su porqué (incluso si el usuario elige otra), y aplicando por defecto
hexagonal, i18n y todas las buenas prácticas sin preguntar.

## 1. Qué existe ya (base a evolucionar, no reescribir)

- `modulos/orquesta-web/nueva_app_intake_session_v0.go`: sesión con `Form`,
  `PendingQuestions []string`, `RecommendedQuestions []string`, decisiones y
  handoff. Limitación: los "pendientes" son nombres de campo planos; no hay
  opciones, ni recomendación razonada, ni huecos cruzados.
- `nueva_app_intake_guided_turn_v0.go`: switch por campo (`tipo_app`,
  `datos`, `deploy`, `calidad`...) que ya sabe mapear respuestas al form.
- `modulos/orquesta-factory/appspec_request_v0.go`: catálogos validados
  (tipo_app: web/api/cli/desktop/mobile/...; storage.tipo: mixta/...;
  deploy.target: local/contenedor/paas/...). Fuente única de opciones.
- i18n con test de propiedad (`nueva_app_i18n_owner_v0_test.go`).

## 2. Contratos nuevos (módulo `orquesta-web`, tipos puros)

```go
type WizardQuestionV0 struct {
    QuestionRef string            // estable, ej. "wizard-q-plataformas"
    Field       string            // campo del AppSpecRequestV0 que resuelve
    TopicGroup  string            // "uso" | "datos" | "entrega" (turnos temáticos)
    Importance  string            // "alta" | "media" (alta bloquea cierre)
    PromptKey   string            // clave i18n de la pregunta
    WhyKey      string            // clave i18n de por qué importa
    Options     []WizardOptionV0  // 2-4 + siempre texto libre implícito
}

type WizardOptionV0 struct {
    Value       string // valor canónico del catálogo factory
    LabelKey    string // i18n
    Recommended bool   // exactamente una por pregunta
    RationaleKey string // i18n: por qué es la mejor (obligatoria en la recomendada)
}

type WizardAnswerV0 struct {
    QuestionRef string
    UserChoice  string // valor elegido o texto libre
    FreeText    bool
}

type WizardTurnResultV0 struct {
    SessionRef       string
    Turn             int
    Questions        []WizardQuestionV0   // máx 3-4, un solo TopicGroup
    Decisions        []WizardDecisionV0   // lo consolidado hasta ahora
    Contrasts        []WizardContrastV0   // elecciones del usuario ≠ recomendada
    EngineeringDefaults []WizardDefaultV0 // sección "decisiones tomadas por ti"
    SpecComplete     bool
    SpecPreview      *orquestafactory.AppSpecRequestV0 // solo si SpecComplete
    LaunchReady      bool // SpecComplete && validación factory ok
}

type WizardContrastV0 struct { // regla 7b.1: recomendación siempre visible
    QuestionRef string
    UserChoice  string
    Recommended string
    RationaleKey string
}

type WizardDefaultV0 struct { // regla 7b.2
    Area  string // "arquitectura" | "i18n" | "calidad" | "docs" | "observabilidad"
    Value string // ej. "hexagonal_puertos_adaptadores"
    WhyKey string
}
```

## 3. Motor de huecos (nuevo fichero `nueva_app_wizard_gaps_v0.go`)

Dos fuentes de preguntas, en este orden:

1. **Campos requeridos vacíos** (ya existe): se convierten en
   `WizardQuestionV0` con opciones del catálogo factory + recomendación.
2. **Reglas cruzadas** (`wizardCrossGapRulesV0`, tabla data-driven; cada
   regla = func(form) *WizardQuestionV0 + test focal):
   - R1 objetivo menciona uso personal/colaborativo ambiguo → "¿personal o
     para compartir?" (recomendada: compartir con auth simple, porqué:
     coste marginal bajo, evita migración futura).
   - R2 sin plataformas → "¿móvil, PC o ambas?" (recomendada: ambas vía web
     responsive salvo que el objetivo pida nativo).
   - R3 objetivo menciona dominio con ecosistema típico (agenda→calendarios
     de empresa Google/Microsoft/CalDAV; tienda→pagos; mapa→fuentes
     públicas) → pregunta de integración con opciones concretas.
   - R4 db_required sin storage.tipo → opciones del catálogo (recomendada
     según fuentes declaradas).
   - R5 integraciones sin auth/criticidad → completar por integración.
   - R6 tipo_app móvil/mixed sin plataformas concretas.
   - R7 deploy.target incompatible con tipo_app → re-preguntar con las
     compatibles.
   - R8 sin usuarios_objetivo cuando el uso es compartido.
   El detector de dominio (R3) es tabla keyword→pack de preguntas; ampliable
   sin tocar el motor.

## 4. Defaults de ingeniería (nuevo `nueva_app_wizard_defaults_v0.go`)

Se inyectan SIEMPRE en el spec sin preguntar (7b.2), declarados en
`preferencias_tecnicas.arquitectura="hexagonal"`, `preferencias_tecnicas.
preferencias=[...]` y `calidad`: i18n es/en por catálogo, tests unitarios +
test de arquitectura (ratchet), linters/format, errores tipados con catálogo
público, logging estructurado, scripts de verificación, y documentación
técnica (handoff_report, source_tree, technical_stack_manifest). Se muestran
en `EngineeringDefaults` en cada turno y en el resumen final. Solo se
pregunta si el usuario los contradice explícitamente; entonces se registra
`WizardContrastV0` (su elección manda, la recomendación queda visible).

## 5. Flujo de turnos

- Turno 1 `uso`: qué es, para quién, plataformas (R1, R2, campos nombre/
  objetivo si faltan).
- Turno 2 `datos`: datos/fuentes/storage/integraciones de dominio (R3-R5).
- Turno 3 `entrega`: deploy, calidad extra, idiomas adicionales (R6-R8).
- Turnos 4-6: solo lo que siga abierto; `SpecComplete` cuando no quedan
  preguntas de importancia alta Y la validación factory pasa.
- Acción `aceptar_recomendaciones`: responde todas las preguntas del turno
  (o de la sesión) con la opción recomendada de una vez.
- Cierre: `LaunchReady=true` → botón/campo `launch=goal_first` reutiliza el
  contrato existente de `POST /api/v0/apps/director` con el
  `AppSpecRequestV0` compuesto. El wizard NO duplica el lanzamiento.

## 6. Superficies

- HTTP: evolución del endpoint de intake guiado actual (misma ruta, campo
  nuevo `mode:"wizard"` para compatibilidad).
- MCP: tool `orquesta.nueva_app.wizard.v0` con el mismo contrato de turno.
- Web: render del turno (preguntas con radio + texto libre, chip "mejor
  opción" con el porqué, panel "decisiones tomadas por ti").

## 7. Plan de tests (mínimo)

- `TestWizardAgendaDesdeSoloObjetivoV0`: entrada "quiero una app para una
  agenda" → verifica familias móvil/pc-ambas, personal/compartir,
  integración agenda empresa; spec completo válido en ≤6 turnos con
  `aceptar_recomendaciones`.
- `TestWizardContrasteRecomendacionVisibleV0`: usuario elige lo no
  recomendado → `Contrasts` conserva recommended+rationale.
- `TestWizardDefaultsIngenieriaSiempreV0`: spec final sin input técnico
  incluye hexagonal+i18n+calidad; y aparecen en `EngineeringDefaults`.
- Un test focal por regla R1-R8.
- Test de propiedad i18n: toda PromptKey/WhyKey/RationaleKey existe en es y
  en (patrón owner existente).

## 8. Reparto de implementación (Orquesta/Codex, prepare-run)

- G1 (web core): tipos + motor de huecos + defaults + sesión (write-set:
  modulos/orquesta-web, docs). Tests R1-R8 + los 3 de aceptación.
- G2 (superficies): endpoint mode=wizard + tool MCP + render (write-set:
  modulos/orquesta-web, modulos/orquesta-mcp, modulos/orquesta-app-gateway).
  Depende de G1.
- G3 (i18n): catálogos es/en de todas las claves nuevas (write-set:
  modulos/orquesta-web i18n, modulos/orquesta-i18n-docs). Paralelo a G2.

## 9. Estado Codex 2026-07-04

Implementado y verificado:

- G1 web core: `WizardQuestionV0`, `WizardOptionV0`, `WizardAnswerV0`,
  `WizardTurnResultV0`, reglas cruzadas R1-R8, defaults de ingenieria y accion
  `AcceptWebNuevaAppWizardRecommendationsV0`.
- Caso canonico del operador: `"quiero una app para una agenda"` genera
  preguntas de plataformas, uso personal/compartido e integracion
  Google/Microsoft/CalDAV; aceptando recomendaciones llega a
  `AppSpecRequestV0` valido en <=6 turnos.
- G2 parcial: el endpoint guided devuelve `wizard` y acepta
  `wizard_answers []WizardAnswerV0`, conservando `Contrasts` cuando el usuario
  elige algo distinto de la recomendacion.
- G3 parcial/cerrado para web core: catalogos es/en y lista de claves
  requeridas cubren todas las preguntas/opciones/defaults generables por el
  motor actual, con test exacto por locale.

Pruebas ejecutadas en este corte:

- `go test -count=1 ./modulos/orquesta-web -run 'TestWizard|TestNuevaAppIntakeGuidedResponseV0IncluyeWizardRicoV0|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0|TestNuevaAppI18nCatalogV0CatalogosCubrenClavesRequeridas'`
- `go test -count=1 ./modulos/orquesta-web`
- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`

Pendiente real tras el corte inicial:

- Tool MCP `orquesta.nueva_app.wizard.v0` con el mismo contrato de turno
  (cerrado en la continuacion Codex 2026-07-04 tarde 5).
- Render web usable de preguntas/opciones/recomendacion/contraste/defaults
  (cerrado en la continuacion Codex 2026-07-04 tarde 5).
- Si se exige `mode:"wizard"` explicito, anadirlo como alias compatible; el
  endpoint actual mantiene compatibilidad y acepta `wizard_answers` sin campo
  de modo.

## 10. Taxonomía de preguntas (vinculante; generaliza R1-R8 y el ejemplo agenda)

El operador aclaró que la agenda era solo un ejemplo. El wizard debe cubrir
DOS capas: dimensiones universales (aplican a cualquier app) y packs de
dominio (se activan por keywords del objetivo). Cada pregunta lleva 2-4
opciones con una recomendada y su porqué.

### 10.1 Dimensiones universales (toda app, en este orden de turnos)

U1 Uso y audiencia: ¿personal, equipo, o público? ¿cuántos usuarios
   esperas? ¿roles distintos (admin/editor/lector)? (rec.: empezar simple
   con roles básicos; ampliar después cuesta poco si el dominio lo separa).
U2 Plataformas: ¿móvil, PC o ambas? ¿necesita funcionar sin internet?
   (rec.: web responsive/PWA salvo necesidad nativa: una base de código).
U3 Identidad y acceso: ¿sin login / login local / Google-Microsoft /
   invitaciones por enlace? ¿2FA? (rec. si compartida: OAuth, sin gestionar
   contraseñas propias).
U4 Datos y ciclo de vida: ¿los datos los crea el usuario, se importan o
   vienen de fuentes externas? ¿import/export (CSV/Excel/JSON/PDF)?
   ¿papelera/deshacer? ¿copias de seguridad? (rec.: export CSV+JSON y
   backup automático siempre que haya datos de usuario).
U5 Privacidad y cumplimiento: ¿datos personales de terceros (RGPD)?
   ¿datos sensibles (salud/finanzas/menores)? ¿registro de auditoría?
   (rec.: minimizar datos y auditoría si hay más de un rol).
U6 Colaboración y avisos: ¿edición simultánea? ¿comentarios? ¿notificaciones
   (email/push/in-app)? ¿compartir por enlace público? (rec.: in-app
   primero; email solo para eventos importantes).
U7 Integraciones: ¿conectar con servicios que ya usas? (ofrecer las del
   pack de dominio + genéricas: email, almacenamiento en la nube, webhooks,
   API propia para terceros).
U8 Búsqueda y organización: ¿búsqueda de texto completo? ¿etiquetas/
   categorías/filtros? (rec.: etiquetas + búsqueda simple; full-text solo
   con volumen alto).
U9 Idiomas y accesibilidad: es/en ya es default silencioso; ¿más idiomas?
   ¿formatos regionales (fecha/moneda)? ¿accesibilidad reforzada/modo
   oscuro? (rec.: seguir preferencia del sistema).
U10 Despliegue y operación: ¿solo en tu equipo, servidor propio, o nube?
   ¿dominio propio con HTTPS? ¿actualizaciones automáticas? (rec.:
   contenedor local si personal; contenedor+nube si compartida).
U11 Escala y rendimiento: ¿usuarios a la vez estimados? ¿algo debe ser
   instantáneo? (solo se pregunta si U1=equipo/público; rec.: dimensionar
   para 10x lo declarado).
U12 Histórico y versiones: ¿ver cambios pasados / restaurar versiones?
   (rec.: sí para documentos/datos editables; no para datos efímeros).

Prioridad: score = importancia(alta=2, media=1) x incertidumbre(sin dato=2,
inferido=1). Máx 3-4 preguntas/turno; alta sin responder bloquea cierre;
`aceptar_recomendaciones` responde el resto de una vez.

### 10.2 Packs de dominio (tabla keyword→pack, ampliable sin tocar el motor)

Cada pack añade 3-6 preguntas específicas con opciones y recomendación:

- **agenda/calendario/citas**: recurrencia; recordatorios (email/push/SMS);
  invitados y confirmaciones; sincronización Google/Microsoft/CalDAV;
  zonas horarias; reservas con disponibilidad.
- **tienda/venta/ecommerce**: catálogo y variantes; carrito e invitado o
  cuenta; pagos (Stripe/PayPal/manual); envíos y zonas; stock; facturas e
  impuestos regionales.
- **inventario/almacén**: códigos de barras/QR; ubicaciones/almacenes;
  movimientos entrada/salida; stock mínimo con alertas; proveedores;
  valoración (FIFO/medio).
- **notas/documentos/wiki**: markdown o texto enriquecido; adjuntos;
  plantillas; versionado; cifrado local; publicación selectiva.
- **tareas/proyectos**: vista lista/kanban/calendario; asignación y
  vencimientos; subtareas y dependencias; prioridades; informes de avance.
- **finanzas/gastos/presupuesto**: categorías y reglas automáticas;
  multi-moneda; importación extractos (CSV/OFX); presupuestos y alertas;
  informes mensuales/fiscales; objetivos de ahorro.
- **contactos/CRM**: etapas de pipeline; actividades y seguimientos;
  deduplicación; importación vCard/CSV/Google; recordatorios de contacto.
- **reservas/turnos**: calendario de disponibilidad; duración y buffers;
  confirmación/cancelación con plazos; recordatorios; pago o señal
  anticipada; lista de espera.
- **mapa/geolocalización**: fuentes (OSM/publicas/propias); capas y
  filtros; rutas; geocodificación; uso offline; privacidad de ubicación.
- **salud/fitness/hábitos**: métricas y unidades; objetivos y rachas;
  gráficas de evolución; recordatorios; dispositivos/wearables; privacidad
  reforzada (datos sensibles → activa U5 en alta).
- **educación/cursos**: lecciones y orden; progreso por alumno;
  ejercicios/evaluaciones; certificados; cohortes/grupos.
- **comunidad/foro**: perfiles; hilos y votos; moderación y reportes;
  menciones y notificaciones; reputación.
- **domótica/IoT/sensores**: dispositivos y protocolos (MQTT/HTTP);
  paneles en tiempo real; umbrales y alertas; histórico de series;
  control remoto seguro.
- **galería/media**: subida masiva; miniaturas y transcodificación;
  álbumes y etiquetas; compartir por enlace; almacenamiento (local/nube).
- **facturación/documentos legales**: series y numeración; plantillas PDF;
  clientes; impuestos; envío por email; archivado legal.

Si el objetivo casa con VARIOS packs (ej. "tienda con reservas"), se
combinan y se deduplica por Field. Si no casa con ninguno, el wizard hace
una pregunta abierta de dominio en turno 1 ("¿qué debe poder hacer un
usuario un día normal?") y reevalúa los packs con la respuesta.

### 10.3 Tests adicionales de la taxonomía

- `TestWizardDimensionesUniversalesCubiertasV0`: spec vacío genera preguntas
  de todas las U con importancia alta.
- `TestWizardPackDominioPorKeywordV0` (tabla): al menos tienda, finanzas,
  reservas e IoT activan su pack.
- `TestWizardPacksCombinadosSinDuplicadosV0`: "tienda con reservas" combina
  packs sin preguntas repetidas por Field.
- `TestWizardSinDominioPreguntaAbiertaV0`: objetivo sin keywords → pregunta
  abierta y reevaluación.

## 11. Estado Codex 2026-07-04 tarde 5

Cerrado en el repo principal:

- MCP tool `orquesta.nueva_app.wizard.v0` con entrada compatible con
  `WebNuevaAppIntakeGuidedRequestV0`: `need`, `action_id`, `answer`,
  `wizard_answers`, `session`, `locale`, `nombre` e `idea`. Devuelve `turn`,
  `session` y `wizard` como JSON durable y conserva errores publicos.
- Stack Codex cablea el executor MCP real para el wizard, usando el mismo
  handler guiado y sin duplicar reglas de negocio.
- Web `/nueva-app` renderiza preguntas ricas del primer turno, opciones,
  insignia de recomendacion, racionales y defaults de ingenieria. Los clicks en
  opciones envian `wizard_answers` al endpoint guiado y el cliente vuelve a
  pintar preguntas, contrastes y defaults desde la respuesta.
- El mapa i18n del wizard se serializa a cliente desde el catalogo es/en para
  evitar mostrar claves internas en turnos dinamicos.

Pruebas ejecutadas en este cierre:

- `go test -count=1 ./modulos/orquesta-web -run 'TestNuevaAppHTMLHandlerV0GETMuestraFormularioUsableSinDelegar|TestWizard|TestNuevaAppIntakeGuidedResponseV0IncluyeWizardRicoV0|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0|TestNuevaAppI18nCatalogV0CatalogosCubrenClavesRequeridas'`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-web`

Pendiente real despues de este cierre:

- Implementar la taxonomia universal U1-U12 y los packs de dominio de la
  seccion 10. El motor actual sigue cubriendo R1-R8 y packs iniciales
  agenda/pagos/mapas/storage/mobile/deploy/usuarios, no toda la taxonomia.
- Anadir los tests adicionales de la seccion 10.3 antes de declarar el wizard
  conversacional completo para cualquier app.
- Opcional: alias explicito `mode:"wizard"` si se decide hacerlo obligatorio
  para clientes nuevos.

## 10. Capa técnica T1-T8 y motor de exclusión (vinculante, orden del operador)

Reglas del operador: (a) "todo de todo" en lo técnico; (b) por defecto SIEMPRE
la mejor opción, y solo se desvía si el humano lo elige (quedando el contraste
registrado, 7b.1); (c) las respuestas EXCLUYEN preguntas y opciones que dejan
de tener sentido — no se pregunta lo irrelevante ni lo ya resuelto por
exclusión (ej.: app para el núcleo de Linux en C → cero preguntas de aspecto
web).

### 10.1 Dimensiones técnicas (se preguntan SOLO si los hechos las activan;
si no, se aplican como default silencioso en EngineeringDefaults)

T1 Control de acceso: ¿RBAC por roles, permisos finos por recurso (ACL),
   o ambos? ¿aislamiento multi-tenant? ¿MFA/passkeys? Sesiones (JWT vs
   sesión de servidor, expiración, revocación). Activada por: U1=equipo o
   público. Default si equipo: RBAC simple (admin/editor/lector) + MFA
   opcional (porqué: cubre el 90% de casos sin matriz de permisos cara).
T2 Identidad corporativa: ¿integrar con Active Directory/Samba AD/LDAP?
   ¿SSO (OIDC/SAML/Kerberos)? ¿aprovisionamiento SCIM? ¿mapear grupos del
   directorio a roles de la app? Activada por: keywords empresa/oficina/
   dominio/AD/LDAP o U1=equipo en contexto corporativo. Default: OIDC si
   hay SSO disponible (porqué: estándar, sin custodiar contraseñas);
   LDAP-bind como alternativa si solo hay AD clásico.
T3 Observabilidad: logging estructurado con ROTACIÓN completa (por tamaño y
   tiempo, retención configurable, compresión, integración
   journald/logrotate), niveles, log de auditoría separado e inmutable;
   métricas Prometheus; trazas OpenTelemetry; healthchecks liveness/
   readiness; alertas por umbral. Default SIEMPRE (silencioso): logs
   estructurados rotados + healthchecks. Se pregunta solo el destino
   (fichero local/journald/colector central) cuando U10=servidor o nube.
T4 Persistencia técnica: motor (SQLite/PostgreSQL/MySQL/KV embebido),
   migraciones versionadas con rollback, pooling, backups automáticos con
   restore PROBADO, cache (memoria/Redis), tareas en segundo plano y cron.
   Activada por: datos de usuario (U4). Default: SQLite si personal/local,
   PostgreSQL si compartida (porqué: concurrencia y tipos ricos); backup
   diario + restore verificado semanal.
T5 API y contratos: ¿API pública propia? REST/gRPC/GraphQL/eventos;
   versionado de API; OpenAPI generada; rate limiting; idempotencia en
   mutaciones; webhooks salientes firmados con reintentos. Activada por:
   U7=API propia o integraciones entrantes. Default: REST versionada +
   OpenAPI (porqué: interoperable y autodocumentada).
T6 Despliegue avanzado: contenedor/systemd; entornos dev/staging/prod;
   CI/CD con gates de test; config 12-factor por entorno; feature flags;
   actualizaciones con migración automática y rollback; HA/failover y
   objetivos DR (RPO/RTO). Activada por: U10=servidor/nube. Default:
   contenedor + systemd + CI con suites obligatorias.
T7 Resiliencia y rendimiento: timeouts/reintentos/circuit breakers en toda
   llamada externa; paginación por defecto en listados; límites de recursos;
   presupuestos de latencia si U11 lo declara. Default SIEMPRE silencioso.
T8 Cumplimiento técnico: auditoría inmutable de acciones sensibles,
   retención de logs/datos conforme a normativa declarada en U5,
   anonimización/pseudonimización, borrado real bajo petición (RGPD).
   Activada por: U5=datos personales/sensibles.

### 10.2 Motor de exclusión (hechos → poda de preguntas y opciones)

Contrato nuevo en el motor de huecos:

```go
type WizardFactV0 struct { Key, Value string } // ej. {"runtime","kernel_c"}

// En WizardQuestionV0 y WizardOptionV0:
//   RequiresFacts []WizardFactV0   // solo aparece si TODOS los hechos casan
//   ExcludedByFacts []WizardFactV0 // desaparece si ALGUNO casa
// En WizardTurnResultV0:
//   ResolvedByExclusion []WizardDecisionV0 // respondidas sin preguntar
```

- Cada respuesta consolida hechos (ej. U2=cli → {"ui","none"}; U1=personal →
  {"audience","single"}; "sin red" → {"network","none"}; tipo_app=kernel/C →
  {"runtime","kernel_c"},{"ui","none"},{"web","none"}).
- Poda de PREGUNTAS: una pregunta cuyos RequiresFacts no casan o cuyos
  ExcludedByFacts casan no se emite nunca. Ej. canónico del operador: app
  para el núcleo de Linux en C → se excluyen aspecto web, i18n de UI,
  accesibilidad de navegador, PWA/offline, RBAC de usuarios finales; siguen
  activas T3 (logging kernel: printk/trace niveles), T6 (build/CI), pruebas.
- Poda de OPCIONES: dentro de una pregunta viva se eliminan opciones
  incompatibles (ej. U1=personal → en T4 no se ofrece "PostgreSQL en
  clúster"; U10=local → en T3 no se ofrece "colector central").
- Resolución por exclusión: si tras la poda queda UNA sola opción elegible,
  no se pregunta: se adopta, se registra en ResolvedByExclusion con su
  porqué, y aparece en el panel "decisiones tomadas por ti". Si quedan cero,
  la pregunta se descarta y se anota el hecho que la cerró.
- Coherencia retroactiva: si el usuario CAMBIA una respuesta que era hecho
  excluyente, el motor reevalúa las podas: preguntas cerradas por exclusión
  vuelven a la cola si recuperan sentido (test focal obligatorio).

### 10.3 Regla "lo mejor por defecto, desviación justificada"

Toda dimensión T no activada se aplica como default silencioso con la mejor
práctica (visible en EngineeringDefaults). Toda dimensión activada llega con
la opción recomendada premarcada. Si el humano elige otra, el turno pide una
justificación breve OPCIONAL (campo libre `justification`) y registra el
WizardContrastV0 con ella; nunca se bloquea por no justificar, pero el
resumen final lista todas las desviaciones con su justificación (o "sin
justificar") para revisión consciente.

### 10.4 Tests adicionales

- `TestWizardKernelLinuxCExcluyeWebV0`: objetivo "módulo para el núcleo de
  Linux en C" → cero preguntas de web/UI/i18n-UI/PWA; T3/T6 siguen.
- `TestWizardExclusionPorAudienciaPersonalV0`: U1=personal → no se pregunta
  RBAC/AD/SSO; quedan resueltas por exclusión con porqué.
- `TestWizardOpcionUnicaSeAdoptaSinPreguntarV0`: poda deja 1 opción →
  ResolvedByExclusion la adopta con rationale.
- `TestWizardReevaluaExclusionesAlCambiarRespuestaV0`: cambiar U1
  personal→equipo devuelve T1/T2 a la cola.
- `TestWizardDesviacionConJustificacionV0`: elegir opción no recomendada
  registra contraste + justificación en el resumen final.
- `TestWizardActiveDirectoryPorContextoEmpresaV0`: objetivo con "para mi
  empresa con dominio Windows" activa T2 con opciones AD/LDAP/OIDC y
  recomendación razonada.

## 11. Documentación de cada opción para humanos no técnicos (vinculante)

Orden del operador: el humano puede no saber qué es cada cosa (ej.:
"multi-tenant"). TODA pregunta y TODA opción llevan explicación en lenguaje
llano, sin asumir conocimientos técnicos.

Contrato (ampliar tipos existentes):

```go
// En WizardQuestionV0:
//   HelpKey string // i18n: qué significa esta pregunta, en llano, con ejemplo
// En WizardOptionV0:
//   HelpKey    string // i18n: qué es esta opción explicada a un no técnico
//   ExampleKey string // i18n opcional: ejemplo cotidiano ("multi-tenant:
//                     // como un edificio de pisos: cada cliente tiene su
//                     // piso con llave propia dentro del mismo edificio")
// En WizardDefaultV0 y ResolvedByExclusion: WhyKey ya existe; añadir HelpKey
// con la explicación llana del concepto adoptado.
```

Reglas:

1. Redacción: una frase de qué es + una de cuándo conviene + un ejemplo
   cotidiano si el término es técnico (multi-tenant, RBAC, SSO, LDAP, JWT,
   circuit breaker, RPO/RTO, SCIM, webhook, full-text...). Prohibido definir
   un término técnico usando otro término técnico sin explicar.
2. Cobertura total verificada por test: `TestWizardTodaOpcionTieneAyudaV0`
   recorre todas las preguntas/opciones/defaults registrados y falla si
   falta HelpKey o si la clave no existe en es y en (reutilizar patrón
   owner). Sin excepciones: opción sin ayuda = build rojo.
3. Superficies: HTTP/MCP devuelven las claves resueltas al locale de la
   sesión; el render web muestra la ayuda plegada (icono "?") por opción y
   la de la pregunta bajo el enunciado.
4. Glosario generado: script/test dora un `docs/wizard_glosario_generado.md`
   con todos los términos y sus explicaciones es/en a partir del catálogo
   (fuente única: las claves i18n; el doc es artefacto derivado, no se edita
   a mano).

### 11.5 Botón "explícamelo todo" y preguntas libres (vinculante)

Además de la ayuda plegada por opción, debe existir una forma explícita de
pedir TODAS las explicaciones:

1. **Botón global por turno** ("¿Qué significa todo esto?" / "Explícamelo
   todo"): despliega de una vez la ayuda de la pregunta y de todas sus
   opciones, en llano, con sus ejemplos. Estado persistente en la sesión
   (si el usuario lo activó una vez, los turnos siguientes llegan ya
   desplegados hasta que lo cierre).
2. **Pregunta libre de comprensión**: en cualquier turno el usuario puede
   escribir "¿qué es X?" en el campo libre; si X casa con un término del
   glosario (matching por sinónimos del catálogo i18n), el wizard responde
   con la explicación y RE-EMITE la misma pregunta sin consumir turno ni
   registrar respuesta. Test: `TestWizardPreguntaQueEsRespondeYNoAvanzaV0`.
3. **Contrato**: `WizardTurnResultV0.GlossaryExpanded bool` +
   `WizardAnswerV0.ComprehensionQuery string` (si viene relleno, el motor
   trata el turno como consulta, no como respuesta). En MCP/HTTP, mismo
   contrato; en la web, botón visible junto al título del turno.
4. El glosario completo (11.4) queda además enlazado desde cada turno
   ("ver glosario completo").

## 12. Bot guía con RAG sobre el catálogo (vinculante, orden del operador)

El operador quiere que un bot conversacional sea quien le guíe, con RAG de
todas las opciones/explicaciones. Principio rector: el bot es una CAPA DE
CONVERSACIÓN sobre el motor del wizard; el motor sigue siendo la única
fuente de verdad. El bot nunca inventa preguntas ni opciones: solo
parafrasea, explica y rellena lo que el motor emite.

### 12.1 Corpus RAG (fuente única, ya estructurada)

- Catálogo i18n completo del wizard: preguntas, opciones, HelpKey,
  ExampleKey, RationaleKey (es/en).
- Glosario generado (11.4) con sinónimos por término.
- Packs de dominio (9.2) y dimensiones U/T con sus condiciones de
  activación y hechos de exclusión (10.2).
- Defaults de ingeniería con sus porqués.
El corpus se reconstruye en build desde el catálogo (artefacto derivado,
como el glosario): no hay segunda redacción que mantener.

### 12.2 Recuperación en dos niveles

1. Determinista (siempre, sin coste): índice léxico local sobre el corpus
   (términos + sinónimos del catálogo + matching difuso acentos/plurales).
   Cubre "¿qué es X?", "¿cuál me recomiendas?", "¿qué opciones hay de
   logs?". Testeable sin proveedor.
2. LLM opcional (si hay proveedor configurado): usa el director de
   escalada existente (claude/gemini/codex, perfil barato reasoning=low)
   SOLO para: parafrasear la pregunta del turno en tono conversacional,
   extraer respuestas de un texto libre largo (slot-filling: "quiero una
   agenda para mi empresa conectada al AD y que me avise por correo" →
   U1=equipo, T2=AD, U6=email), y reformular explicaciones. El prompt
   SIEMPRE lleva el fragmento recuperado del corpus como única base
   (grounding estricto); si el corpus no cubre la duda, el bot dice que no
   lo sabe y ofrece la pregunta abierta del turno.

### 12.3 Flujo del bot

- Usuario escribe libre → slot-filling responde todo lo que el texto ya
  contesta (cada asignación se registra como WizardAnswerV0 normal, con
  contraste si contradice la recomendación) → el motor recalcula huecos y
  exclusiones → el bot presenta el siguiente turno en conversación:
  pregunta parafraseada + opciones numeradas con la recomendada marcada y
  su porqué + "responde con el número, con tus palabras, o pregúntame qué
  significa cualquier cosa".
- "¿qué es X?" en cualquier momento → RAG responde y re-emite el turno
  (mismo contrato 11.5, sin consumir turno).
- "acepta lo que recomiendes" → acción aceptar_recomendaciones existente.
- Al cierre: el bot lee el resumen (decisiones, desviaciones con
  justificación, decisiones tomadas por ti) y pide confirmación explícita
  antes de LaunchReady.

### 12.4 Superficies y contrato

- `WizardBotTurnV0 { SessionRef, UserText, Locale }` →
  `WizardBotReplyV0 { Say string; TurnResult WizardTurnResultV0;
  FilledAnswers []WizardAnswerV0; GroundingRefs []string }`.
- Web: panel de chat como modo alternativo al formulario del wizard
  (mismo SessionRef: se puede alternar entre chat y formulario sin perder
  estado). MCP: tool `orquesta.nueva_app.wizard.bot.v0`.
- Sin proveedor LLM configurado el bot funciona en modo determinista
  (nivel 1): menos florido, misma capacidad de guiar/explicar/rellenar por
  número. Nunca es requisito tener LLM para completar el wizard.

### 12.5 Tests

- `TestWizardBotGroundingEstrictoV0`: toda Say referencia GroundingRefs
  del corpus; término fuera de corpus → respuesta honesta "no lo sé" +
  re-emisión del turno.
- `TestWizardBotSlotFillingRegistraRespuestasV0`: texto libre multi-dato
  registra las WizardAnswerV0 correctas y respeta exclusiones (10.2).
- `TestWizardBotSinProveedorFuncionaV0`: modo determinista completa una
  sesión entera de la agenda solo con números y "qué es X".
- `TestWizardBotNoInventaOpcionesV0`: la respuesta nunca contiene valores
  fuera de las opciones emitidas por el motor para ese turno.

### 12.6 Reparto

- G4: corpus derivado + índice léxico + bot determinista + tests (web).
- G5: slot-filling/parafraseo vía director de escalada + panel chat web +
  tool MCP. Depende de G4 y de G2.

### 12.7 Opcionalidad y coste (aclaración vinculante del operador)

El operador lo quiere, pero como tiene coste debe ser OPCIONAL:

1. El nivel LLM del bot (12.2.2) es opt-in explícito:
   `ORQUESTA_WIZARD_BOT_LLM_ENABLED=true` + proveedor configurado. Por
   defecto APAGADO: el wizard y el bot determinista funcionan completos sin
   gastar un token.
2. Presupuesto propio cuando está encendido:
   `ORQUESTA_WIZARD_BOT_DAILY_TOKEN_BUDGET` (por defecto conservador). Al
   agotarse, el bot degrada SOLO el parafraseo/slot-filling al modo
   determinista y lo dice en la conversación ("sigo contigo en modo básico
   por presupuesto"); la sesión nunca se corta.
3. En la web, activar el chat con LLM muestra una nota de coste una vez por
   sesión; el modo formulario y el chat determinista nunca la muestran.
4. Registro: tokens del bot van a la misma contabilidad thread_goals/uso
   que el resto de Orquesta para que el operador vea el gasto.
5. Test: `TestWizardBotDegradaAPresupuestoAgotadoV0` y
   `TestWizardBotApagadoPorDefectoV0`.

## 13. Estado vigente tras tanda Codex tarde 31

El wizard no queda completo. Lo integrado hasta esta tanda cubre la base
web/MCP y un avance parcial de dominio:

- Packs nuevos o ampliados para `inventario`, `notas/documentos`,
  `tareas/proyectos`, `finanzas`, `crm`, `reservas`, `salud`, `educacion`,
  `comunidad`, `iot`, `media`, `facturacion` y `ecommerce`.
- R3 puede emitir varias preguntas de dominio con campos `integraciones.N.tipo`
  distintos y aplica decisiones como conectores sin duplicar el mismo campo.
- La pregunta abierta de dominio se reserva para objetivos que no encajan con
  ningun pack conocido.
- Hay cobertura focal para tienda/ecommerce, finanzas, reservas, IoT,
  combinacion de packs, dominio abierto e i18n de las claves nuevas.

Pendiente vinculante antes de declarar "wizard universal para cualquier app":

- Taxonomia U1-U12 completa y efectiva.
- Capa tecnica T1-T8 completa y efectiva.
- Motor de exclusion runtime con hechos/requires/excluded/resolved.
- `HelpKey`/`ExampleKey`, glosario y boton "explicamelo todo".
- Bot determinista/RAG y nivel LLM opt-in con presupuesto.
- Tests de aceptacion completos de las secciones 9-12.
