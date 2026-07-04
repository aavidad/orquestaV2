# Diseño: wizard de programación conversacional (TAREA-7/7b)

Fecha: 2026-07-04. Autor: Claude (director). Implementa: Orquesta/Codex.
Estado: diseño aprobado por el operador; parcialmente implementado por Codex
en `modulos/orquesta-web` el 2026-07-04. No tratar como cierre total: quedan
pendientes tool MCP equivalente y render web completo del turno.

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

Pendiente real:

- Tool MCP `orquesta.nueva_app.wizard.v0` con el mismo contrato de turno.
- Render web usable de preguntas/opciones/recomendacion/contraste/defaults.
- Si se exige `mode:"wizard"` explicito, anadirlo como alias compatible; el
  endpoint actual mantiene compatibilidad y acepta `wizard_answers` sin campo
  de modo.

## 9. Taxonomía de preguntas (vinculante; generaliza R1-R8 y el ejemplo agenda)

El operador aclaró que la agenda era solo un ejemplo. El wizard debe cubrir
DOS capas: dimensiones universales (aplican a cualquier app) y packs de
dominio (se activan por keywords del objetivo). Cada pregunta lleva 2-4
opciones con una recomendada y su porqué.

### 9.1 Dimensiones universales (toda app, en este orden de turnos)

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

### 9.2 Packs de dominio (tabla keyword→pack, ampliable sin tocar el motor)

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

### 9.3 Tests adicionales de la taxonomía

- `TestWizardDimensionesUniversalesCubiertasV0`: spec vacío genera preguntas
  de todas las U con importancia alta.
- `TestWizardPackDominioPorKeywordV0` (tabla): al menos tienda, finanzas,
  reservas e IoT activan su pack.
- `TestWizardPacksCombinadosSinDuplicadosV0`: "tienda con reservas" combina
  packs sin preguntas repetidas por Field.
- `TestWizardSinDominioPreguntaAbiertaV0`: objetivo sin keywords → pregunta
  abierta y reevaluación.
