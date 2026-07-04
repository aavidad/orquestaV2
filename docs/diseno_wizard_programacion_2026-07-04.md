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
