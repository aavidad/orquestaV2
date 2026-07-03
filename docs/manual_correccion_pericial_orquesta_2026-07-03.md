<!--
Manual de corrección pericial — Orquesta
Autor: revisión externa asistida (Claude Code, Fable 5)
Fecha: 2026-07-03
Origen: docs/informe_pericial_claude_orquesta_2026-07-03.md
Destinatarios: agentes de programación (Codex u otros) que ejecuten las
tareas T-PER-*. Cada tarea es autocontenida y puede asignarse a un agente
distinto. Bitácora de ejecución: docs/bitacora_correccion_pericial_2026-07-03.md
-->

# Manual de corrección pericial — cómo arreglar Orquesta

Este manual traduce los hallazgos P1-P6 del informe pericial a tareas
ejecutables `T-PER-NNN`. Léelo entero antes de tocar código aunque solo te
toque una tarea: la sección 1 (reglas comunes) es obligatoria para todas.

## 0. Cómo usar este manual

- Cada tarea tiene: objetivo, contexto, write-set, diseño exacto, pasos,
  tests con nombre, comandos de verificación y criterio de cierre.
- **Antes de empezar una tarea**: apunta en la bitácora
  (`docs/bitacora_correccion_pericial_2026-07-03.md`) tu claim con fecha,
  agente y tarea. Al terminar, apunta evidencia (tests verdes, commit).
- **Una tarea = un commit o serie corta de commits**. Asunto en castellano,
  imperativo, corto (estilo del repo: "Estructura artefactos parciales en
  Goal").
- Respeta las dependencias de la sección 2. No empieces una tarea cuyo
  prerequisito no esté `hecho` en la bitácora.
- Si descubres un bug nuevo mientras trabajas: fila nueva en
  `docs/inventario_bugs_orquesta_2026-06-30.md` (formato existente), no lo
  arregles en silencio ni abras frente nuevo.

## 1. Reglas comunes obligatorias (para TODAS las tareas)

1. **Nunca ejecutes `go test ./...` global**: mata la sesión por memoria.
   Ejecuta solo paquetes focales: `go test -count=1 ./modulos/<modulo> ./cmd/orquesta-server`.
2. **Siempre antes de commitear**:
   - `go build ./...` (esto sí es seguro y debe quedar limpio);
   - `go test -count=1 ./` (tests de frontera en raíz, ~1s);
   - tests focales de los paquetes tocados con `-count=1`.
3. **Presupuesto de fichero**: ningún `.go` de producción puede superar
   **900 líneas** (`TestResidualGoFileBudgetT90V0` lo vigila en
   `cmd/orquesta-server`; aplica el mismo criterio en módulos).
4. **Convención de nombres**: ficheros nuevos con sufijo `_v0.go`. Módulos y
   paquetes nuevos de dominio/adaptador en **castellano** (doctrina en
   `ARQUITECTURA.md`, sección "Módulos e Idioma").
5. **Fronteras**: el núcleo neutral no importa `net/http`, `os/exec`,
   adaptadores de producto ni `cmd/`. Si creas módulo neutral nuevo, añádelo
   a la cobertura de `architecture_boundaries_test.go`.
6. **No toques OPES productivo** ni exportes `ORQUESTA_OPES_BASE_URL` en
   pruebas. Smokes reales solo con sus flags de confirmación y en directorio
   aislado.
7. **Write-set ajeno**: hay un agente con write-set declarado sobre
   `cmd/orquesta-server/codex_goal_app_server_v0.go`,
   `codex_goal_app_server_rpc_v0.go`, `codex_goal_backend_env_v0.go`,
   `codex_goal_app_server_tmux_v0_test.go`,
   `smoke_goal_first_script_guard_v0_test.go`,
   `scripts/smoke_goal_first_app_server_real.sh` y
   `docs/inventario_bugs_orquesta_2026-06-30.md` (ver
   `docs/auditoria_claude_estado_orquesta_2026-07-03.md`). Las tareas que
   tocan esos ficheros están marcadas **BLOQUEADA-POR-WRITE-SET** y no se
   empiezan hasta que ese trabajo esté commiteado/integrado.
8. **No borrar generaciones sin sunset documentado**: retirar código legacy
   requiere la tarea T-PER-201/203 aprobada, no iniciativa individual.
9. **i18n**: cualquier texto visible nuevo en web/UI nace localizable.
10. **Evidencia**: al cerrar una tarea deja: nombres de tests verdes,
    comando exacto ejecutado, y (si aplica) fila de inventario actualizada.

## 2. Mapa de fases, tareas y dependencias

| Tarea | Fase | Hallazgo | Depende de | Reparto |
| --- | --- | --- | --- | --- |
| T-PER-101 módulo `orquesta-estado-vivo` (contrato + proyección pura) | 1 | P1 | — | 1 agente |
| T-PER-102 adaptadores de evidencia en el stack Codex | 1 | P1 | 101 | 1 agente |
| T-PER-103 migrar `autoprogramming/status` a la proyección | 1 | P1 | 102 | 1 agente |
| T-PER-104 migrar `director/stats` y `queue/global-status` | 1 | P1 | 102 | 1 agente |
| T-PER-105 migrar `domain-work/status` y `observe` goal | 1 | P1 | 102 | 1 agente |
| T-PER-106 test de frontera ratchet de estado vivo | 1 | P1 | 103-105 | 1 agente |
| T-PER-201 sunset formal del loop legacy + fix opt-in smoke T18 | 2 | P2 | — | 1 agente |
| T-PER-202 congelar espina Director V2 (ratchet) | 2 | P2 | — | 1 agente |
| T-PER-203 mapa canónico de los 17 módulos director | 2 | P2 | 201, 202 | 1 agente |
| T-PER-301 mover backend Codex Goal de `cmd/` a módulo | 3 | P3 | write-set ajeno integrado | 1 agente |
| T-PER-302 máquina de estados explícita del backend | 3 | P3 | 301 | 1 agente |
| T-PER-401 contrato de shutdown en dos fases | 4 | P4/inventario | write-set ajeno integrado | 1 agente |
| T-PER-402 smoke E2E nightly | 4 | P4 | 401 | 1 agente |
| T-PER-501 script de métricas de deuda | 5 | P5 | — | 1 agente |
| T-PER-502 presupuesto de endpoints de status (ratchet) | 5 | P5 | 501 | 1 agente |
| T-PER-601 sincronizar doctrina (persistencia, autoridad) | 6 | P6 | — | 1 agente |
| T-PER-701 restricción dura de lenguaje en Nueva App | 7 | informe preliminar | — | 1 agente |

Tareas sin dependencias que pueden arrancar hoy en paralelo:
**T-PER-101, T-PER-201, T-PER-202, T-PER-501, T-PER-601, T-PER-701**.

---

## FASE 1 — Grafo operativo único del estado vivo (P1, la más importante)

### T-PER-101 — Crear módulo `orquesta-estado-vivo`

**Objetivo**: que exista UNA función pura que, dadas todas las evidencias
disponibles, calcule la fase de vida de cada trabajo y exponga los
conflictos entre fuentes, en vez de que cada endpoint re-derive su propia
foto.

**Contexto**: hoy hay 64 interfaces de estado y ≥15 endpoints de status que
reconcilian por su cuenta. Todos los bugs `running_stale` / `falso 100%` /
`ready falso` nacen de esa divergencia.

**Write-set** (todo nuevo, no toca nada existente):

- `modulos/orquesta-estado-vivo/tipos_v0.go`
- `modulos/orquesta-estado-vivo/puertos_v0.go`
- `modulos/orquesta-estado-vivo/proyeccion_v0.go`
- `modulos/orquesta-estado-vivo/reglas_precedencia_v0.go`
- `modulos/orquesta-estado-vivo/*_v0_test.go`
- `modulos/orquesta-estado-vivo/AGENTS.md` y `docs/` (contratos, decisiones)
- `architecture_boundaries_test.go` (añadir el módulo a la frontera neutral)

**Diseño exacto** (paquete `orquestaestadovivo`; es núcleo neutral: sin
`net/http`, sin `os/exec`, sin imports de producto):

```go
// Fase única calculada. Valores cerrados:
type FaseCicloVidaV0 string

const (
    FaseSolicitadoV0       FaseCicloVidaV0 = "solicitado"
    FaseLanzadoV0          FaseCicloVidaV0 = "lanzado"
    FaseProcesoVivoV0      FaseCicloVidaV0 = "proceso_vivo"
    FaseEntregadoParcialV0 FaseCicloVidaV0 = "entregado_parcial"
    FaseBloqueadoV0        FaseCicloVidaV0 = "bloqueado"
    FaseTerminalAceptadoV0 FaseCicloVidaV0 = "terminal_aceptado"
    FaseTerminalReworkV0   FaseCicloVidaV0 = "terminal_rework"
    FaseHuerfanoV0         FaseCicloVidaV0 = "huerfano"
    FaseConflictoV0        FaseCicloVidaV0 = "conflicto"
    FaseDesconocidoV0      FaseCicloVidaV0 = "desconocido"
)

// Una observación de UNA fuente. Las fuentes NO deciden fase; solo aportan.
type EvidenciaEstadoV0 struct {
    RunRef          string
    GoalRef         string
    ExternalGoalRef string
    Fuente          string // "run_store"|"goal_state"|"run_marker"|"process_registry"|"process_snapshot"|"receipt"|"outbox"|"queue"|"checkpoint"
    Estado          string // estado bruto tal y como lo publica la fuente
    ProcesoVivo     bool   // solo fuentes de proceso
    Terminal        bool   // solo receipt/estado terminal validado
    Aceptado        bool   // closure accepted
    ObservadoEn     string // RFC3339
    EvidenceRefs    []string
}

type ConflictoEstadoV0 struct {
    RunRef  string
    Codigo  string // p.ej. "proceso_vivo_tras_terminal", "terminal_sin_receipt", "marker_sin_estado"
    Fuentes []string
}

type NodoCicloVidaV0 struct {
    RunRef          string
    GoalRef         string
    ExternalGoalRef string
    Fase            FaseCicloVidaV0
    Evidencias      []EvidenciaEstadoV0 // las usadas para decidir
    Conflictos      []ConflictoEstadoV0
}

type ProyeccionCicloVidaV0 struct {
    SchemaVersion string // "orquesta_proyeccion_ciclo_vida.v0"
    GeneradaEn    string
    Nodos         []NodoCicloVidaV0
}

// Función PURA y determinista. Sin IO, sin reloj interno (now se inyecta).
func ConstruirProyeccionCicloVidaV0(
    evidencias []EvidenciaEstadoV0,
    ahora time.Time,
    umbralHuerfano time.Duration,
) ProyeccionCicloVidaV0
```

**Reglas de precedencia** (implementar EXACTAMENTE estas, en
`reglas_precedencia_v0.go`, cada una con test propio):

1. Evidencia con `ProcesoVivo=true` y sin evidencia terminal → fase
   `proceso_vivo`. Domina sobre cualquier estado persistido no terminal
   (mata los `running_stale` con procesos vivos).
2. Evidencia `Terminal=true && Aceptado=true` y sin proceso vivo → fase
   `terminal_aceptado`.
3. Evidencia `Terminal=true && Aceptado=false` y sin proceso vivo → fase
   `terminal_rework`.
4. `ProcesoVivo=true` **y** `Terminal=true` simultáneos → fase `conflicto`
   + conflicto `proceso_vivo_tras_terminal`. **Prohibido** resolverlo en
   silencio: el conflicto es salida de primera clase.
5. Estado persistido `blocked`/`bloqueado` sin proceso vivo y sin terminal
   → fase `bloqueado`.
6. Solo marker/launch (fuente `run_marker`) sin estado ni proceso: si
   `ahora - ObservadoEn > umbralHuerfano` → `huerfano`; si no → `lanzado`.
7. Entregas parciales (`receipt` no terminal con artefactos) → `entregado_parcial`.
8. Ninguna evidencia → `desconocido` (nunca inventar verde).
9. Agregación por nodo: agrupar evidencias por `RunRef` (y si falta, por
   `GoalRef`); un nodo por trabajo.

**Tests obligatorios** (mínimo):

- `TestConstruirProyeccionCicloVidaV0ProcesoVivoDominaEstadoStale`
- `TestConstruirProyeccionCicloVidaV0TerminalAceptadoSinProcesoVivo`
- `TestConstruirProyeccionCicloVidaV0ProcesoVivoTrasTerminalEsConflicto`
- `TestConstruirProyeccionCicloVidaV0MarkerViejoEsHuerfano`
- `TestConstruirProyeccionCicloVidaV0SinEvidenciasEsDesconocido`
- `TestConstruirProyeccionCicloVidaV0EsDeterminista` (misma entrada en
  distinto orden → misma salida; ordenar internamente)

**Verificación**: `go build ./... && go test -count=1 ./modulos/orquesta-estado-vivo ./`

**Criterio de cierre**: módulo compilando, tests verdes, cubierto por
`architecture_boundaries_test.go` como paquete neutral, documentado en su
`AGENTS.md` con la frase: "Las fuentes aportan evidencia; SOLO
`ConstruirProyeccionCicloVidaV0` decide fase".

**Qué NO hacer**: no migrar todavía ningún endpoint; no añadir IO al módulo;
no añadir dependencias a otros módulos de Orquesta.

### T-PER-102 — Adaptadores de evidencia en el stack Codex

**Objetivo**: que las fuentes reales (RunStore, GoalWorkStateStore,
GoalWorkRunMarkerStore, ProcessRegistry, snapshots, receipts) se proyecten a
`[]EvidenciaEstadoV0` mediante adaptadores finos.

**Write-set**:

- `modulos/orquesta-app-codex-stack/evidencia_estado_runstore_v0.go`
- `modulos/orquesta-app-codex-stack/evidencia_estado_goalstate_v0.go`
- `modulos/orquesta-app-codex-stack/evidencia_estado_marker_v0.go`
- `modulos/orquesta-app-codex-stack/evidencia_estado_procesos_v0.go`
- `modulos/orquesta-app-codex-stack/evidencia_estado_receipts_v0.go`
- tests `*_v0_test.go` por adaptador
- un agregador: `modulos/orquesta-app-codex-stack/evidencia_estado_agregador_v0.go`
  con `ListarEvidenciasEstadoV0(ctx, filtro) ([]orquestaestadovivo.EvidenciaEstadoV0, error)`

**Reglas**:

- Cada adaptador SOLO traduce; nada de lógica de fase (eso vive en
  T-PER-101). Si un adaptador necesita decidir algo, es señal de que la
  regla falta en `reglas_precedencia_v0.go`: añádela allí con test.
- `ProcesoVivo=true` solo si el registro de procesos confirma PID vivo en el
  momento de la consulta (usar los mismos checks que ya usa
  `ProcessRegistry`/snapshot; no inventar).
- Traducciones de estado bruto documentadas en tabla en
  `modulos/orquesta-app-codex-stack/docs/contratos.md`.

**Tests**: uno por adaptador con dobles in-memory ya existentes en el repo
(p.ej. `orquesta-run-memory`, `orquesta-agent-process-registry-memory`), más
`TestAgregadorEvidenciaEstadoV0CombinaFuentesSinPerderRefs`.

**Criterio de cierre**: agregador devuelve evidencias de las 5 familias de
fuente para un run de prueba; ningún adaptador importa `net/http`.

### T-PER-103 — Migrar `autoprogramming/status` a la proyección

**Objetivo**: que `GET /api/v0/autoprogramming/status` derive TODOS sus
campos de vida (`running_live`, `running_stale`, `agents_live`,
`efficiency_summary`, `overall_percentage`) desde
`ConstruirProyeccionCicloVidaV0`, no desde consultas propias.

**Contexto**: bugs BUG-ORQ-20260630-006 y -038 nacieron aquí (stale con
procesos vivos; 100% con goal bloqueado).

**Write-set**: `modulos/orquesta-mcp/autoprogramming_status_*.go` (los
ficheros del executor de status) + tests.

**Pasos**:

1. Localiza el executor: `grep -rn "AutoprogrammingStatus" modulos/orquesta-mcp --include='*.go' -l`.
2. Inyecta el agregador de T-PER-102 como puerto nuevo
   (`FuenteEvidenciaEstadoPortV0`).
3. Sustituye el cálculo de vida propio por lectura de la proyección.
   **El schema de salida NO cambia** (los consumidores OPES dependen de él);
   cambia solo de dónde salen los valores.
4. Regla de mapeo obligatoria: `fase=conflicto` → `attention_required`,
   nunca verde; `fase=desconocido` → no computar como 100%.
5. Los tests existentes del executor deben seguir verdes SIN reescribirlos
   (si uno falla, o el mapeo está mal o el test fijaba un comportamiento
   erróneo; en el segundo caso documenta el cambio en el commit).

**Tests nuevos**:

- `TestMCPAutoprogrammingStatusExecutorV0DerivaVidaDesdeProyeccionV0`
- `TestMCPAutoprogrammingStatusExecutorV0ConflictoNuncaProyectaVerdeV0`

**Verificación**: `go test -count=1 ./modulos/orquesta-mcp`

### T-PER-104 — Migrar `director/stats` y `queue/global-status`

Igual que T-PER-103 pero para esos dos endpoints (bugs de origen:
BUG-ORQ-20260630-045 falso 100% en stats; -040 acciones colapsadas en
queue). Mismas reglas: schema estable, vida desde proyección, conflicto
nunca verde, tests equivalentes
(`TestMCPDirectorStatsToolExecutorV0DerivaVidaDesdeProyeccionV0`,
`TestMCPQueueGlobalStatusHTTPHandlerV0DerivaVidaDesdeProyeccionV0`).

### T-PER-105 — Migrar `domain-work/status` y `observe` goal

Igual que T-PER-103 para `GET /api/v0/domain-work/status` (fachada de
BUG-ORQ-20260630-047: debe pasar a leer la proyección en vez de reutilizar
otros dos endpoints) y para `observe_app_director_goal` (el snapshot parcial
de BUG-035 debe construirse desde la proyección). Tests:
`TestMCPDomainWorkStatusHTTPHandlerV0DerivaVidaDesdeProyeccionV0`,
`TestMCPObserveAppDirectorGoalToolExecutorV0SnapshotDesdeProyeccionV0`.

### T-PER-106 — Test de frontera ratchet del estado vivo

**Objetivo**: impedir por test que vuelvan a aparecer derivaciones de vida
fuera de la proyección.

**Write-set**: `estado_vivo_boundaries_test.go` (raíz del repo, junto a
`architecture_boundaries_test.go`).

**Diseño**: test que parsea imports (mismo mecanismo que el boundary test
existente; cópialo como plantilla) y verifica:

1. Lista `paquetesStatus` = los paquetes que publican status/observe
   (mínimo: los tocados en 103-105).
2. Esos paquetes NO importan directamente stores de estado
   (`orquesta-run-file`, `orquesta-state-file`, registries de proceso...)
   **salvo** los que estén en una allowlist literal en el propio test.
3. La allowlist arranca con los imports que existan hoy y **solo puede
   menguar**: comentario en el test: "PROHIBIDO añadir entradas; si tu
   cambio necesita añadir una, tu diseño es incorrecto: pasa por
   orquesta-estado-vivo".

**Criterio de cierre**: test verde hoy; quitar una entrada usada lo pone
rojo (probado); añadir un import prohibido en un paquete status lo pone rojo.

---

## FASE 2 — Retirada de generaciones del director (P2)

### T-PER-201 — Sunset formal del loop legacy + fix del opt-in del smoke (T18)

**Objetivo**: el loop legacy (`legacy_director_loop`) queda oficialmente en
extinción con fecha, y el smoke de proveedores deja de fallar por opt-in no
exportado.

**Parte A (docs+diagnóstico)**:

1. Añade a `AGENTS.md` (sección "Foto vigente") y a
   `docs/inventario_bugs_orquesta_2026-06-30.md` (fila arquitectónica) el
   sunset: "legacy_director_loop: solo mantenimiento correctivo; sin
   features nuevas; candidato a retirada cuando goal-first cubra Claude y
   Gemini (ver T18)".
2. En el código, donde se acepta `director_execution_mode=legacy_director_loop`,
   añade al resultado/diagnóstico el campo `legacy_sunset_notice` con ese
   texto corto. Test:
   `TestStartAppDirectorV0LegacyLoopPublicaSunsetNoticeV0`.

**Parte B (fix real, incidencia T18 de
`modulos/orquesta-opes-connector/docs/tareas.md`)**:

1. `scripts/smoke_opes_reviews_providers_real.sh` usa
   `director_execution_mode=legacy_director_loop` pero no exporta
   `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true` → 400
   `legacy_run_supervise_requires_explicit_opt_in`.
2. Arréglalo exportando la variable en el script SOLO cuando el modo sea
   legacy, con comentario de sunset al lado.
3. `bash -n scripts/smoke_opes_reviews_providers_real.sh` + test guard del
   script si existe patrón (mira `smoke_goal_first_script_guard_v0_test.go`
   como plantilla de cómo se testean guards de scripts).

**Qué NO hacer**: no borrar el loop legacy; no tocar su lógica.

### T-PER-202 — Congelar la espina Director V2 (ratchet)

**Objetivo**: que la espina `director-cycle → director-runner →
director-scheduler → director-cycle-outbox → director-tick-input` no crezca
salvo decisión explícita.

**Pasos**:

1. Añade a cada `AGENTS.md` de esos 5 módulos: "MÓDULO CONGELADO 2026-07-03:
   solo fixes correctivos con bug enlazado. Features nuevas requieren
   decisión documentada en docs/ raíz. Motivo: informe pericial P2".
2. Crea `director_v2_freeze_test.go` en la raíz: cuenta ficheros `.go` de
   producción en esos módulos y compara contra un mapa literal con los
   valores actuales. Mensaje de fallo: "espina V2 congelada; si el aumento
   es un fix aprobado, actualiza el mapa en el mismo commit y enlaza el bug".

**Criterio de cierre**: test verde; añadir un fichero a
`orquesta-director-cycle` lo pone rojo.

### T-PER-203 — Mapa canónico de los 17 módulos `*director*`

**Objetivo**: un documento único que diga, módulo a módulo, cuál es la
generación viva, cuál está congelada y cuál está en sunset.

**Write-set**: `docs/mapa_generaciones_director_2026-07-03.md` + enlace
desde `AGENTS.md` (orden de autoridad, nivel 2).

**Contenido mínimo**: tabla con los 17 módulos
(`ls modulos | grep director`), columnas: módulo, generación
(legacy/V1/V2/goal-first/soporte), estado (vivo/congelado/sunset),
consumidores actuales (grep de imports), y decisión. Goal-first queda
declarado como único camino de producción.

---

## FASE 3 — Backend Codex Goal fuera de `cmd/` (P3)

> **BLOQUEADA-POR-WRITE-SET**: no empezar hasta que el trabajo local/remoto
> descrito en `docs/auditoria_claude_estado_orquesta_2026-07-03.md` esté
> commiteado e integrado en la rama. Verifica con `git status` que
> `cmd/orquesta-server/codex_goal_app_server*.go` está limpio.

### T-PER-301 — Mover el backend a `modulos/orquesta-runtime-codex-appserver`

**Objetivo**: el componente con más bugs del sistema pasa a ser un módulo
gobernado, con docs, AGENTS.md y fronteras, en vez de vivir en la raíz de
composición.

**Pasos**:

1. Crea `modulos/orquesta-runtime-codex-appserver/` (paquete
   `orquestaruntimecodexappserver`).
2. Mueve con `git mv` (histórico limpio) estos ficheros desde
   `cmd/orquesta-server/`: `codex_goal_app_server_v0.go`,
   `codex_goal_app_server_rpc_v0.go`, `codex_goal_app_server_tmux_v0.go`,
   `codex_goal_app_server_websocket_protocol_v0.go`,
   `codex_goal_backend_env_v0.go` y TODOS sus `_test.go`.
3. Cambia `package main` → `package orquestaruntimecodexappserver` y exporta
   solo lo que `cmd/orquesta-server` realmente usa (empieza exportando lo
   mínimo; el compilador te dirá qué falta).
4. En `cmd/orquesta-server`, deja solo el wiring (construcción del backend
   desde env) importando el módulo nuevo.
5. **El commit del movimiento NO cambia lógica**: `git diff` debe ser
   renombres + cambios de paquete/visibilidad. Cualquier fix va en commit
   separado.
6. Añade el módulo a `architecture_boundaries_test.go`: puede usar
   `os/exec` (es adaptador de runtime) pero NO puede importar
   `orquesta-persistence`, stores de estado ni `orquesta-web`.

**Verificación**: `go build ./... && go test -count=1 ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server ./`

**Avance 2026-07-03**: se creó el módulo y el wiring principal ya instancia el
backend desde `modulos/orquesta-runtime-codex-appserver`, con boundary de
arquitectura. El modulo ya no importa `orquesta-server.ConfigV0`; `cmd` adapta
su configuracion al `ConfigV0` neutral del modulo. No está cerrado: siguen duplicados legacy en
`cmd/orquesta-server/codex_goal_app_server*.go` y los tests principales aún no
viven dentro del módulo. Incidencia de seguimiento:
`BUG-ORQ-20260703-150`.

### T-PER-302 — Máquina de estados explícita del backend

**Objetivo**: sustituir los checks dispersos de vida del app-server/tmux por
una máquina de estados con transiciones nombradas y testeadas.

**Diseño** (`estado_backend_v0.go` en el módulo nuevo):

```go
type EstadoBackendAppServerV0 string

const (
    BackendInexistenteV0    EstadoBackendAppServerV0 = "inexistente"
    BackendPreparandoV0     EstadoBackendAppServerV0 = "preparando"      // Ensure en curso
    BackendSocketPendienteV0 EstadoBackendAppServerV0 = "socket_pendiente"
    BackendListoV0          EstadoBackendAppServerV0 = "listo"
    BackendDegradadoV0      EstadoBackendAppServerV0 = "degradado"       // con issue_code
    BackendApagandoV0       EstadoBackendAppServerV0 = "apagando"        // kill pedido, pane_pid vivo
    BackendApagadoV0        EstadoBackendAppServerV0 = "apagado"         // pane_pid desaparecido
    BackendHuerfanoV0       EstadoBackendAppServerV0 = "huerfano"        // sesión tmux sin owner marker
)

// Única función de decisión de transición. Pura: observaciones entran, estado sale.
func TransicionBackendV0(actual EstadoBackendAppServerV0, obs ObservacionBackendV0) (EstadoBackendAppServerV0, []string /*issue codes*/)
```

**Pasos**: identifica cada punto donde hoy se decide "¿está listo/vivo/
apagado?" (Ensure, preflight, shutdown hook, cleanup post-fallo — los
cierres de BUG-023/029/033/043/044 los señalan) y haz que todos consulten
`TransicionBackendV0`. Un test por transición legal y otro
(`TestTransicionBackendV0RechazaTransicionesIlegales`) por las ilegales.

**Criterio de cierre**: `grep -n "has-session\|pane_pid" *.go` dentro del
módulo muestra esas consultas SOLO en el recolector de observaciones, no en
lógica de decisión.

**Avance 2026-07-03**: existe `estado_backend_v0.go` con transiciones puras y
tests unitarios, pero aún no sustituye las decisiones dispersas de
Ensure/shutdown/cleanup/active-work. Cierre pendiente junto a
`BUG-ORQ-20260703-150`.

---

## FASE 4 — Shutdown fiable y smoke nightly (P4)

> T-PER-401 quedó desbloqueada y cerrada sobre rama limpia el 2026-07-03; no se
> integró el WIP remoto completo de `BUG-ORQ-20260703-149`.

### T-PER-401 — Contrato de shutdown en dos fases

**Objetivo**: eliminar la clase de bug "shutdown_ready=true pero el proceso
sigue vivo" (residual del smoke del 2026-07-03).

**Contrato a implementar** (mismo patrón que ya se aplicó al pane tmux en
BUG-ORQ-20260630-023, un nivel arriba):

1. `shutdown_ready=true` significa EXACTAMENTE: sin trabajo vivo + limpieza
   solicitada + salida programada. No significa "el proceso ya salió".
2. La respuesta HTTP de shutdown incluye `exit_pending=true` y el `pid`.
3. Tras responder (flush), el servidor termina de verdad (salida ordenada
   del runtime; si a los N segundos no ha salido, `os.Exit` como último
   recurso, N configurable con default 10s).
4. El wrapper/cliente (smoke) espera la desaparición del PID con timeout,
   y SOLO falla si el PID sigue vivo pasado el timeout.

**Write-set**: `modulos/orquesta-server/` (runtime shutdown),
`cmd/orquesta-server` (wiring), `scripts/smoke_goal_first_app_server_real.sh`
(espera de PID), tests.

**Tests**:

- `TestRuntimeV0ShutdownReadySignificaSalidaProgramadaV0`
- `TestRuntimeV0ProcesoSaleTrasShutdownReadyV0` (con proceso de test real,
  estilo `TestCodexAppServerTmuxBackendV0EsperaPanePIDAntesDeReadyV0`)
- guard del script: la espera de PID existe y tiene timeout.

**Cierre 2026-07-03**: implementado como `exit_pending/pid` en el wrapper HTTP
del runtime, `ForceExitPortV0` inyectable en `orquesta-server` y espera de PID
en `scripts/smoke_goal_first_app_server_real.sh`. El caso de uso puro
`orquesta-server-shutdown` solo transporta campos opcionales; no conoce procesos.

### T-PER-402 — Smoke E2E nightly

**Objetivo**: que el bucle completo (Nueva App goal-first) se ejercite cada
noche sin humano, y sus fallos aparezcan como artefacto, no como incidencia
OPES días después.

**Write-set**: `scripts/orquesta_smoke_nightly.sh` + unidad
systemd/cron de usuario documentada en
`docs/runbooks/smoke_nightly_2026-07.md`.

**Diseño del script**:

1. Modo por defecto: preflight
   (`ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1`) — barato, sin cuota.
2. Modo real solo si `ORQUESTA_NIGHTLY_REAL_CONFIRM=1` está en el
   environment de la unidad (decisión del operador, no del script).
3. Resultado SIEMPRE como JSON datado en
   `~/.orquesta-nightly/resultado_YYYYMMDD.json` con: exit code, fase
   alcanzada, refs, duración. Nunca commitea ni toca el repo.
4. Retención: borra resultados >30 días.
5. Si el último resultado fue fallo, el script imprime el diff de fases
   contra el último verde (para que el primer agente de la mañana abra
   fila de inventario con contexto).

**Cierre 2026-07-03**: implementado `scripts/orquesta_smoke_nightly.sh` con
preflight por defecto, modo real opt-in mediante `ORQUESTA_NIGHTLY_REAL_CONFIRM=1`,
resultado JSON diario, retencion, diff contra ultimo verde y bloqueo de entorno
OPES/productivo. Runbook: `docs/runbooks/smoke_nightly_2026-07.md`.

---

## FASE 5 — Higiene de superficie (P5)

### T-PER-501 — Script de métricas de deuda

**Write-set**: `scripts/orquesta_metricas_deuda.sh` +
`scripts/test_orquesta_metricas_deuda.sh` (mismo patrón que
`test_orquesta_runtime_retention.sh`).

**El script imprime** (y opcionalmente escribe JSON con `--json`):

```
env_vars_orquesta=$(grep -rhoE 'ORQUESTA_[A-Z0-9_]+' --include='*.go' modulos cmd | sort -u | wc -l)
endpoints_status=$(grep -rhoE '"/api/v0/[a-z0-9/_{}-]+"' --include='*.go' cmd modulos | sort -u | grep -cE "status|stats|readiness|observe|health|cockpit|control")
interfaces_estado=$(grep -rhoE "type [A-Za-z0-9]+ interface" --include='*.go' modulos cmd | sort -u | grep -icE "store|registry|ledger|outbox|state|marker|snapshot|queue")
modulos_director=$(ls modulos | grep -c director)
```

Valores de línea base 2026-07-03 (déjalos en el propio script como
comentario): env=500, status=15, interfaces_estado=64, director=17.

### T-PER-502 — Presupuesto ratchet de endpoints de status

Test en raíz `status_surface_budget_test.go`: lista literal de los endpoints
de status/observe actuales; añadir uno nuevo pone el test rojo con el
mensaje "la superficie de status solo puede crecer con decisión documentada:
lee docs/informe_pericial_claude_orquesta_2026-07-03.md R5". Quitar
endpoints solo requiere actualizar la lista (celebrarlo en el commit).

---

## FASE 6 — Doctrina sincronizada (P6)

### T-PER-601 — Corregir ARQUITECTURA.md y autoridad documental

1. Sección "Restricciones" de `ARQUITECTURA.md`: sustituir la afirmación de
   PostgreSQL por la realidad: persistencia file-based
   (`orquesta-persistence` ledgers, `orquesta-state-file`, markers), con
   Postgres como dirección futura si se decide, no como estado actual.
2. Añadir al aviso de cabecera de `ARQUITECTURA.md` el enlace al informe
   pericial y a este manual.
3. Revisar que `AGENTS.md` no contradiga el mapa de T-PER-203 cuando exista.
4. **No** reescribir documentos históricos: solo la foto vigente.

---

## FASE 7 — Contrato duro de lenguaje en Nueva App (informe preliminar, pregunta 1)

### T-PER-701 — Restricción dura lenguaje/framework + gate de cierre

**Objetivo**: que "pedí Go y me generó Python" sea imposible de cerrar como
`complete`.

**Diseño**:

1. **Spec**: el compilador de `GoalWorkSpecV0` para Nueva App
   (búscalo con `grep -rn "buildStartAppDirectorGoalWorkSpecV0" modulos/orquesta-app-director-service`)
   debe transportar `lenguaje` y `framework` del `AppSpecRequestV0` como
   regla dura (no como prosa): añade a las `Rules`/campos del spec una
   entrada estable `technical_constraint:language=<x>` y
   `technical_constraint:framework=<y>` cuando el wizard las haya fijado.
2. **Gate de cierre**: en el validador de cierre goal-first del stack Codex
   (donde ya viven los gates OPES tipo
   `TestCodexStackV0ExternalWorkGoalFirstNoCierraOPESVisualFinalSVG`),
   añade detección por manifiesto de artefactos: si la constraint es
   `language=go` y los artefactos materializados no contienen `go.mod` pero
   sí `pyproject.toml`/`package.json` (o viceversa), el cierre queda
   `rework` con razón `wrong_language_generated` y acción
   `replan_with_language_constraint`.
3. La detección es por presencia de ficheros manifest, no por parsear
   código: tabla cerrada `go→go.mod, python→pyproject.toml|setup.py,
   node→package.json, rust→Cargo.toml` documentada en el validador.
4. Si el wizard NO fijó lenguaje: no hay constraint y no hay gate (la
   preferencia blanda sigue siendo válida).

**Tests**:

- `TestBuildStartAppDirectorGoalWorkSpecV0TransportaConstraintLenguajeV0`
- `TestCodexStackV0GoalFirstNoCierraNuevaAppConLenguajeEquivocadoV0`
- `TestCodexStackV0GoalFirstCierraNuevaAppSinConstraintDeLenguajeV0`

---

## 8. Ejecutar estas tareas CON Orquesta (el pilotaje)

La intención es que Orquesta programe estas tareas (dogfooding). Protocolo:

### 8.1 Precondiciones

- Codex CLI autenticado (`~/.codex` con `auth.json` y `config.toml`) y con
  cuota. Backend goal: `app_server_tmux` (canon).
- Entorno aislado SIEMPRE (los smokes ya lo hacen). Nunca contra el árbol de
  trabajo principal si hay write-set ajeno sin commitear.

### 8.2 Validación barata del entorno (sin cuota)

```bash
ORQUESTA_GOAL_FIRST_SMOKE_PREFLIGHT_ONLY=1 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
./scripts/smoke_goal_first_app_server_real.sh
```

Si el preflight falla: fila de inventario + arreglar antes de gastar cuota.

### 8.3 Lanzar una tarea T-PER como goal real

El flujo soportado para que Orquesta se programe a sí misma es el
self-programming goal-first. Camino recomendado, de menos a más ambición:

1. **Primera vez / validación del bucle**: smoke compuesto en raíz aislada:

   ```bash
   ORQUESTA_SELF_PROGRAMMING_COMPOSITE_SMOKE_CONFIRM=1 \
   ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
   ORQUESTA_KEEP_SMOKE_DIR=1 \
   ./scripts/smoke_self_programming_composite_goal_first.sh
   ```

2. **Tarea concreta**: registra la tarea T-PER elegida como entrada de
   backlog en el `docs/tareas.md` del módulo afectado (formato
   `## XXX-TASK-NNN: título` + Objetivo + Write-set previsto + Validación,
   copiando el contenido de este manual para esa tarea), y lanza el flujo de
   automejora idle apuntando a un worktree limpio:
   `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_PROJECT_WORKDIR=<worktree>`
   (+ `_PROJECT_REF`, `_BRANCH_REF`, `_AREA` según
   `cmd/orquesta-server/server_env_registry_v0.go`).
3. **Regla de pilotaje**: una tarea cada vez, empezar por T-PER-501 (pequeña,
   write-set solo `scripts/`) o T-PER-101 (módulo nuevo, cero colisiones).
   Revisar el diff resultante ANTES de integrar. Si Orquesta falla en el
   intento: eso también es resultado — fila de inventario con el fallo, y la
   tarea se hace a mano por un agente siguiendo este manual.

### 8.4 Si te quedas sin cuota o se corta la sesión

1. Todo lo necesario para continuar está en: este manual + el informe
   pericial + la bitácora (`docs/bitacora_correccion_pericial_2026-07-03.md`).
2. El siguiente agente: lee bitácora → toma la primera tarea sin claim cuyo
   prerequisito esté hecho → claim → ejecuta → evidencia.
3. No reinterpretes las tareas: si una instrucción parece imposible o
   contradictoria con el código actual, NO improvises: anota el conflicto en
   la bitácora y pasa a otra tarea.

## 8.5 Adenda tras el primer pilotaje real (2026-07-03 tarde)

El pilotaje de T-PER-501 se ejecutó y encontró bugs reales. Lecciones que
CORRIGEN lo dicho arriba (esta adenda manda sobre la sección 8):

1. **Formato de sección de backlog que el planner realmente entiende**
   (`cmd/orquesta-server/idle_self_improvement_backlog_parser_v0.go`):
   etiquetas `Objetivo:` (una línea), `Estado: pendiente.`, `Alcance:`
   (lista = write-set), `Criterios:` (lista), `Tests:` (lista),
   `Dependencias:`. **"Write-set previsto:" NO se parsea** y el spec cae al
   write-set enlatado del área por defecto. Ejemplo vivo correcto: sección
   T265 en la rama `pericial/pilot-t501`.
2. **Bug arreglado en local** (commit en rama de trabajo): el prepare del
   backend Goal convertía en directorio o abortaba con
   `codex_app_server_write_set_prepare_failed` cualquier write-set con
   ficheros existentes no-Markdown. Ver bitácora (entrada "pilotaje real")
   para detalle y tests. Si un launch goal-first falla con ese código en un
   árbol sin el fix, ese es el motivo.
3. **La receta completa y el diagnóstico paso a paso del pilotaje** están en
   `docs/bitacora_correccion_pericial_2026-07-03.md` (sección "Receta
   completa"). Usadla tal cual para pilotar T-PER-101, 201, 202, 601 y 701.
4. Bugs abiertos detectados por el pilotaje (candidatos a fila de inventario
   cuando el write-set ajeno se libere): receipt `invalid` sin causa
   (refuerza T-PER-101), etiquetas de backlog silenciosamente ignoradas,
   `shutdown forced` que responde `backend_still_running` sin salir del
   proceso (refuerza T-PER-401), y status/observe sin alias GET.

### T-PER-801 — Revisión del router de contexto híbrido (nueva, para 1 agente)

**Objetivo**: revisar `docs/diseno_router_contexto_hibrido_2026-07-03.md` y
decidir con evidencia si es el mejor camino de ahorro de tokens antes de
implementarlo.

**Pasos**:

1. Leer el diseño y extraer: qué problema exacto resuelve (tokens de contexto
   por agente), qué componentes propone y qué coste de implementación tiene.
2. Comparar con lo que YA existe en el repo: el broker central de contexto
   (`CodeContextQueryPortV0`, BUG-ORQ-20260630-032), el modo compacto
   `caveman` y las reglas de "workers con prompt compacto" de
   `ARQUITECTURA.md`. Pregunta clave: ¿el diseño duplica el broker o lo
   extiende?
3. Investigar alternativas maduras (búsqueda web) para reducción de tokens de
   contexto en agentes: al menos LLMLingua/LongLLMLingua (compresión de
   prompt), RAG selectivo sobre índice de código (tipo repomap de Aider,
   tree-sitter skeleton), y resúmenes jerárquicos con caché de prompt del
   proveedor. Evaluar madurez, coste de integración en Go, y compatibilidad
   con Codex CLI/app-server.
4. Entregar: decisión razonada (implementar tal cual / recortar / descartar)
   + si procede, secciones de backlog `## Txxx` ejecutables con `Alcance:` y
   `Tests:` para que Orquesta lo programe. Criterio del propietario:
   "todo lo que sea mejora de eficiencia, bienvenido" — pero sin duplicar el
   broker existente ni añadir una cuarta fuente de verdad de contexto.

### T-PER-901 — Control plane dormido: despertar por evento + watchdog (nueva, decisión del propietario 2026-07-03)

**Objetivo**: que ningún bucle residente de Orquesta gaste ciclos ni tokens
mientras hay trabajo delegado en curso: se duerme hasta que (a) llega el
evento de finalización/cambio, o (b) vence un watchdog largo por si el
trabajo se colgó. Es el patrón que el supervisor humano/Claude usó en los
pilotajes y el propietario quiere como norma de toda la app: "gastar cuanto
menos mejor, sin merma de eficiencia".

**Evidencia del problema** (auditoría del pilotaje 2026-07-03,
`state/audit/orquesta_server_audit_v0.jsonl`): con UN goal en vuelo y nada
más que hacer, el supervisor emitía `supervisor_tick_start/result` +
`idle_self_improvement_scheduled`→`check blocked` cada 5 segundos,
indefinidamente (pares idénticos con la misma request); el `goal_observer`
acumuló 31 ticks en ~2,5 min sin observaciones (`observed: 0`).

**Contrato a implementar** (por bucle residente: supervisor, idle
self-improvement, goal observer, resident director, outbox dispatch,
watchdog de leases):

1. Cada bucle declara sus **fuentes de despertar por evento**: terminal de
   run/goal (transición persistida), cambio de cola, ACK/delivery, marker
   nuevo en filesystem. Donde ya exista canal de wakeups (el resident
   director lo tiene: `residentDirectorWakeups`), usarlo; donde no, añadirlo.
2. El tick por reloj pasa a ser **solo watchdog**: intervalo largo
   (config por bucle, default 60-300s, no 5s) y su única función es detectar
   cuelgue/pérdida de evento, no dirigir el trabajo.
3. **Anti-churn idempotente**: si la decisión de un tick es idéntica a la
   anterior (misma request, mismo blocked), no se re-emite ni se re-audita
   como evento nuevo; se registra un contador compacto (`skipped_identical`).
4. Ningún componente re-pregunta a un agente/proveedor LLM por status: el
   status se lee de estado durable/proyección (cuando exista T-PER-101/102,
   de la proyección de ciclo de vida).
5. Config nueva mínima; preferir reutilizar `TickInterval` existente como
   watchdog y añadir el canal de eventos, no multiplicar env vars (regla R5).

**Write-set orientativo**: `modulos/orquesta-server` (loops residentes),
`cmd/orquesta-server` (wiring), tests focales por bucle.

**Criterios de cierre**:

- con un goal en vuelo y sin eventos, la auditoría de 10 minutos contiene
  como mucho los ticks watchdog esperados (no cientos de pares
  scheduled/blocked idénticos);
- un terminal de goal despierta al supervisor en <1s vía evento, no
  esperando al siguiente tick;
- un backend colgado se detecta por watchdog y produce diagnóstico, no
  silencio;
- tests: uno por bucle demostrando "duerme sin eventos" y "despierta por
  evento"; p.ej. `TestSupervisorLoopV0DuermeSinEventosYDespiertaPorTerminalV0`.

**Cómo pilotarlo**: mismo procedimiento de la bitácora, una sección `## Txxx`
por bucle (empezar por el par supervisor/idle, que es el churn peor).

## 9. Criterio global de éxito

El trabajo de este manual está terminado cuando:

1. Los 4 primeros números de `scripts/orquesta_metricas_deuda.sh` bajan o se
   mantienen (nunca suben) durante dos semanas seguidas.
2. El smoke nightly lleva 7 días consecutivos en verde en modo real.
3. `estado_vivo_boundaries_test.go` y `status_surface_budget_test.go` están
   verdes con allowlists estrictamente menores que las iniciales.
4. El inventario de bugs deja de recibir >5 filas/día de la familia
   "estado vivo/proyecciones".
