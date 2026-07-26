# V23: contrato parcial de intake único para Wizard

Fecha: 2026-07-25.

Estado: **partial_green_unsealed**. Este write-set implementa únicamente el
contrato puro de `internal/intake` para `WIZ-03`, `WIZ-04` y `WIZ-15`. No
acredita `AC-V23-WIZARD`, no crea receipt, no promueve el roadmap y no afirma
que V23 esté completa.

> Actualización 2026-07-26: este documento conserva el corte puro inicial.
> La dependencia inmediatamente posterior —writer de aplicación, CAS/restart
> SQLite y bindings públicos— ya tiene implementación candidata en
> `docs/reconstruccion/corte_v23_intake_durable_2026-07-26.md`. Las frases
> posteriores que la enumeran como pendiente describen el baseline del
> 2026-07-25, no el estado vigente. La capa de aplicación ya tiene un builder
> candidato de dossier en `internal/application/intake_dossier.go`: recibe el
> `IntakeRecord` durable y el `PlanSpec` concreto, recalcula el digest del
> estado, deriva el del plan y proyecta las decisiones completas. Su
> persistencia, validación ejecutable por el orquestador y comandos públicos
> del dossier ya están integrados. Confirmación, freeze y creación atómica del
> Goal siguen pendientes. V23 continúa
> `partial_green_unsealed`.
>
> Corte posterior: `docs/reconstruccion/corte_v23_dossier_durable_2026-07-26.md`
> ya acredita persistencia, replay, restart, recovery y comandos públicos del
> dossier. Siguen pendientes su generación editorial, confirmación, freeze y
> creación atómica del Goal.

Contexto causal:

- proyecto: `project:v23`;
- Goal: `goal:a7b8704cc9b368213fd7c201dfc8e6c9`;
- WorkItem: `work-item:7e8333c215d71ea3199bb4b48fea7a15`;
- ejecución: `execution:1e8b0c8164d7dda5e626f43e108996f8`;
- generación de plan: `1`;
- generación de AppSpec: `1`.

## 1. Declaración obligatoria de tarea

```text
capability IDs: WIZ-03, WIZ-04, WIZ-15
invariante: chat y form mutan por expected revision una identidad única; cada pregunta deriva de gap/contradiction y conserva elección y recomendación
autoridad que escribe: transición pura internal/intake.Apply; application y persistencia no se adelantan
puertos afectados: ninguno
adaptadores afectados: ninguno
write-set: internal/intake; acceptance/v23_wizard_test.go; acceptance/fixtures/v23_wizard.json; este documento
dependencias causales: contratos V04/V05/V20/V21; la política canónica de rondas depende de otro Goal con lease L-CONFIG
código antiguo que permitirá retirar: estados wizard/intake de modulos/orquesta-web y modulos/orquesta-app-director-intake, únicamente durante V34
test de contrato: TestAcceptanceV23WizardIntakeContract más los dos negativos del RequiredTest focal
negativo/mutación: revisión stale, refs inválidas, cero/múltiples recomendaciones, creación por canal, límite global, atomicidad y rechazo de [no tests to run]
E2E/gate: aceptación parcial offline; no CAS durable, idempotencia application, dossier, confirmación, freeze, binding, i18n completo, plan, web, roadmap, receipt o sello
presupuesto LOC/tokens/tiempo/disco: <=1.700 LOC totales; 0 puertos/stores/writers externos; <=50 min; <100 MB
```

La ejecución por Codex directo se limita al WorkItem ya dirigido por la nueva
Orquesta y a su write-set. No usa, arranca ni modifica el runtime antiguo.

## 2. Autoridad y lectura histórica

La autoridad vigente es `AGENTS.md`, `internal/AGENTS.md`,
`product/roadmap.json#AC-V23-WIZARD` y
`docs/reconstruccion/ruta_total_100.md`. La aceptación total V23 exige más que
este corte: chat/form, rondas configurables, dossier confirmado y creación
causal del plan sobre el único motor.

### V01–V22 del rebuild

No existía un Wizard nuevo que pudiera copiarse. Los cortes previos aportan
contratos que este paquete respeta sin importarlos:

- V01–V03 clasifican capabilities y congelan legado; no aportan estado de
  intake productivo.
- V04 aporta la lección de snapshots inmutables, identidad y cambio por
  revisión esperada. `internal/intake` aplica la misma propiedad local, pero
  no importa `goal` ni crea otro lifecycle de Goal.
- V05 reserva la creación del DAG/plan a la autoridad causal existente. Una
  decisión de intake no crea WorkItems.
- V06 acredita CAS durable y persistencia state-centric en el writer de
  aplicación. Este paquete implementa solo una transición pura con revisión
  esperada; su repositorio, transacción e idempotencia application quedan
  pendientes y no se simulan aquí.
- V07 es el único owner de configuración. Por ello el contrato recibe
  `Policy{MaxQuestionRounds: ...}` y no fija seis rondas ni otra cifra por
  defecto. Registrar y componer el default requiere otro Goal con
  `L-CONFIG`.
- V10 reserva autenticación/autorización a application. `OriginChat` y
  `OriginForm` no son principals ni conceden permisos.
- V20 reserva comandos, bindings y errores públicos al registro único. Este
  corte define códigos de dominio estables, pero no añade comandos.
- V21 mantiene `wizard` como superficie futura con política
  `catalog_required_before_activation`. Aquí se conservan exclusivamente
  claves i18n tipadas; no se añaden literales ni se finge el catálogo completo.
- V22 declara explícitamente Wizard/web fuera de su vertical. No se acopla el
  intake al adapter Codex ni a sus ejecuciones.

### Legacy consultado en solo lectura

Se leyeron las implementaciones históricas relevantes:

- `modulos/orquesta-app-director-intake/wizard_types_v0.go`,
  `wizard_questions_v0.go`, `wizard_answers_v0.go`, `wizard_normalize_v0.go`,
  `wizard_v0.go` y sus tests;
- `modulos/orquesta-web/nueva_app_intake_session_v0.go`,
  `nueva_app_intake_session_decision_v0.go`,
  `nueva_app_wizard_types_v0.go`, `nueva_app_wizard_gaps_v0.go`,
  `nueva_app_wizard_turn_v0.go` y sus tests;
- `docs/diseno_wizard_programacion_2026-07-04.md` y la incidencia del contrato
  guiado del 2026-07-02.

Soluciones conceptuales conservadas:

- preguntas por campos ausentes y reglas cruzadas, no un cuestionario fijo;
- opciones explícitas, una recomendada y rationale;
- contraste que no sustituye la elección real;
- texto humano identificado por claves de catálogo;
- política de límite de rondas recibida desde fuera del motor.

Conductas legacy rechazadas:

- `Draft` o sesión propios de web/MCP y creación implícita por canal;
- nombres con sufijo de generación y cualquier import a `modulos/**`;
- aplicar respuestas una a una antes de descubrir un error posterior;
- ignorar silenciosamente preguntas u opciones desconocidas;
- convertir cero recomendaciones en “la primera” o eliminar recomendaciones
  múltiples durante normalización;
- hardcodear seis rondas cuando no llega una política;
- producir `AppSpec`, handoff, run o plan directamente desde el wizard puro;
- literales de UI, proveedores, DB, filesystem o configuración dentro del
  dominio.

## 3. Contrato implementado

### 3.1 Una identidad y una revisión

`NewState` es el único constructor y no recibe origen. Exige:

```text
Ref con prefijo intake:
Policy.MaxQuestionRounds > 0
Revision inicial = 1
Schema = orquesta.intake.state.v1
```

`Apply(current, change)` exige la misma `StateRef` y
`ExpectedRevision == current.Revision`. Chat y form son únicamente valores de
`Origin` registrados en `History`; no existe revisión, constructor, mapa,
sesión o lifecycle por origen.

La comparación de revisión es una precondición pura del snapshot. No constituye
CAS durable, persistencia, restart ni idempotencia de application.

Cada cambio aceptado incrementa una sola revisión. Un cambio con preguntas
consume a lo sumo una ronda global, con independencia del origen. Un cambio de
elección no consume otra ronda. El paquete no contiene default.

El estado es un snapshot inmutable: los slices son privados, los accessors
devuelven copias y `Apply` clona antes de construir la revisión siguiente.

### 3.2 Preguntas causales

Un `Issue` es exactamente uno de:

```text
gap
contradiction
```

Incluye una ref, un field máquina y `DetailKey`. Una `Question` contiene una o
más `DerivedFrom` que deben resolver Issues ya presentes o incluidos en el
mismo cambio. También contiene `PromptKey`, `WhyKey` y opciones. Una pregunta
sin fuente causal falla; no hay lista rígida embebida.

Este corte no detecta automáticamente huecos de una AppSpec concreta. El
caller propondrá Issues tipados y el writer de aplicación deberá autorizarlos
y persistir la transición. La pureza del paquete garantiza la relación causal,
no inventa análisis de dominio, templates o provider.

### 3.3 Recomendación y elección simultáneas

Cada opción lleva:

```text
OptionRef
LabelKey
RationaleKey
Recommended
```

Cada pregunta requiere exactamente una recomendación. No se normalizan cero o
varias. Todas las opciones conservan rationale por clave i18n, lo que permite
explicar tanto la mejor opción como sus alternativas sin literales privados.

Al elegir, `Decision` conserva:

```text
Choice
Recommendation
RecommendationRationale
Origin
Revision
```

Si `Choice != Recommendation`, ambos hechos permanecen juntos. Una elección
posterior añade otra decisión causal; `CurrentDecision` deriva la más reciente
sin reescribir la historia del snapshot.

### 3.4 Atomicidad

Antes de construir el snapshot siguiente se valida el lote completo:

1. estado inicializado, identidad, revisión y origen;
2. Issues y refs duplicadas;
3. Questions, fuentes causales, keys, opciones y cardinalidad de recomendación;
4. política global de rondas;
5. Choices, preguntas/opciones existentes y duplicación en el lote.

Solo después se clonan y anexan Issues, Questions, Decisions e History. En
error, el estado recibido no puede sufrir mutación parcial.

## 4. Códigos máquina

| Código | Condición |
|---|---|
| `intake.invalid_argument` | política, kind, field, cambio vacío o revisión no representable |
| `intake.invalid_ref` | ref sintácticamente inválida |
| `intake.invalid_origin` | origen distinto de `chat` o `form` |
| `intake.revision_conflict` | expected revision stale o futura |
| `intake.state_mismatch` | cambio apunta a otra identidad válida |
| `intake.channel_state_creation_forbidden` | se intenta aplicar un cambio sobre estado cero |
| `intake.duplicate_ref` | Issue, Question u Option duplicada |
| `intake.issue_not_found` | pregunta no deriva de Issue resoluble |
| `intake.question_not_found` | elección apunta a pregunta ausente |
| `intake.option_not_found` | elección apunta a opción ajena o ausente |
| `intake.recommendation_count` | no hay exactamente una recomendación |
| `intake.message_key_invalid` | prompt/why/label/rationale no es clave válida |
| `intake.round_limit` | otra ronda superaría la política recibida |
| `intake.choice_conflict` | un lote contiene más de una elección por pregunta |

`DomainError` contiene solo código y field máquina. La presentación humana
pertenece al catálogo V21 activado para Wizard en otro write-set.

## 5. Estado de capabilities WIZ

Este corte deja implementadas y ejercitadas parcialmente, pero no wired ni
acreditadas:

| ID | Cobertura local |
|---|---|
| `WIZ-03` | Question requiere refs a gaps/contradictions |
| `WIZ-04` | opción recomendada única y Decision conserva elección + recomendación |
| `WIZ-15` | chat/form comparten Ref, Revision, Policy e History |

Todos los demás WIZ permanecen no completados por este write-set. Los
aceptados quedan pendientes; los rechazados por roadmap no se reabren:

| Estado | IDs |
|---|---|
| pendiente | `WIZ-01`, `WIZ-02`, `WIZ-05..11`, `WIZ-13`, `WIZ-16..25` |
| rechazado por roadmap | `WIZ-12`, `WIZ-14` |

Pendientes explícitos del gate V23:

- comandos públicos y wiring de la preparación/consulta del dossier;
- definición y bindings en `internal/commands/registry.json`;
- activación de la superficie Wizard y catálogo i18n completo;
- generación del dossier final;
- confirmación exacta por `dossier_ref` y freeze posterior del contenido
  confirmado;
- creación causal del plan únicamente después de la confirmación;
- templates `research`, `build_app`, `change_app`, `domain_production`,
  `deploy` y `self_change`;
- packs de dominio, preview, presupuesto, riesgos y efectos;
- web, accesibilidad, chat/form públicos y paridad con otros bindings;
- default canónico de rondas bajo un Goal que posea `L-CONFIG`;
- P/S/E y emisión de receipt;
- sello y promoción de roadmap/manifest sobre digests inmutables;
- retirada legacy V34.

## 6. Aceptación parcial

Fixture: `acceptance/fixtures/v23_wizard.json`.

Tests:

```text
internal/intake:
- chat y form avanzan una secuencia común
- pregunta causal y recomendación/elección
- stale/ref/creación por canal
- 0/N recomendaciones y ausencia de mutación parcial
- política tipada compartida
- copias defensivas

acceptance:
- TestAcceptanceV23WizardIntakeContract
- TestV23WizardIntakeNegativeAndAtomicContract
- TestV23WizardRoundPolicyHasNoPackageDefaultAndNoPerOriginReset
- RequiredTest focal ejecuta exactamente los tres nombres y rechaza
  `[no tests to run]`
- guard dinámico de todos los `.go` de `internal/intake`
```

RequiredTest focal:

```bash
go test -mod=vendor -race -count=1 -v ./acceptance -run '^(TestAcceptanceV23WizardIntakeContract|TestV23WizardIntakeNegativeAndAtomicContract|TestV23WizardRoundPolicyHasNoPackageDefaultAndNoPerOriginReset)$'
```

Esta aceptación verde demuestra semántica offline del paquete. La evidencia
de integración separada añade intake y dossier durables, replay, restart y
recovery. Todavía no demuestra generación editorial del dossier, comandos
públicos, confirmación, freeze, creación causal de plan, E2E web, promoción del
roadmap, receipt ni sello.

Verificación ejecutada:

```text
PASS  go test -mod=vendor -race -count=1 ./internal/intake
PASS  go test -mod=vendor -count=1 ./acceptance -run <tres tests V23>
PASS  go test -mod=vendor -count=1 ./internal/intake ./internal/application ./internal/commands ./internal/goal
PASS  GOFLAGS=-mod=vendor go vet ./internal/intake ./acceptance
PASS  GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
PASS  JSON estricto, gofmt, whitespace e import boundary del write-set
PASS  git diff --check
```

La suite amplia no es un gate verde de este candidato todavía. Los tests raíz
rechazan correctamente receipts V17/V22 cuyo source digest queda stale al
añadir fuentes sin sello; este WorkItem tiene prohibido reemitirlos. El sandbox
también impide listeners `httptest` y no reproduce los modos de filesystem
exigidos por V09/SQLite/Git local. No se corrigieron ni normalizaron esos
frentes ajenos. Los gates V23 anteriores sí se ejecutaron completos.

## 7. Cierre del WorkItem

```text
hecho: contrato puro de estado intake único, preguntas causales y recomendación visible
invariante restaurado: ningún origen posee identidad, revisión, policy o lifecycle privado
autoridad final: State inmutable + Apply puro; application seguirá siendo el único writer persistente
tests/negativos/mutaciones/E2E: unitarios y aceptación parcial; E2E público pendiente
receipts y revisión acreditada: ninguno; partial_green_unsealed
código o decisión retirados: ninguno; se rechazaron normalizaciones y defaults legacy
legacy retirado o bloqueo de retirada: legacy read-only hasta equivalencia V23 completa y cutover V34
LOC netas y complejidad: +1.694 líneas (477 producto, 220 unit, 470 acceptance, 197 fixture, 330 contrato); validación O(I+Q+O+C); 0 puertos, stores, adapters, comandos o loops residentes
riesgos/P0/P1: ningún P0/P1 reclamado; faltan integración y gates listados
siguiente dependencia causal: writer application + persistencia + command binding sobre este mismo expected revision; configuración por L-CONFIG
```
