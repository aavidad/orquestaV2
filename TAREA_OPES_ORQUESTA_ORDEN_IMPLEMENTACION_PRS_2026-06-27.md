# Orden de implementación (PRs) — autonomía goal-first Orquesta

Fecha: 2026-06-27.
Para: agente programador. Este fichero es el **índice maestro**: fija el orden de
PRs y enlaza los tres documentos de detalle. No mezclar tareas de PRs distintos.

## Documentos de referencia
- **Auditoría / evidencia:** `TAREA_OPES_ORQUESTA_REVISION_CODIGO_Y_CONTRATOS_2026-06-27.md`
- **Encargo (tareas T1–T10):** `TAREA_OPES_ORQUESTA_ENCARGO_PROGRAMADOR_FIXES_2026-06-27.md`
- **Diseño + firmas exactas (T1/T2):** `TAREA_OPES_ORQUESTA_DISENO_OBSERVADOR_GOAL_FIRST_Y_ESTADO_VIVO_2026-06-27.md`

## Principio
Capacidad principal = **programar de forma autónoma** (idle → programar → observar
→ cerrar → siguiente). OPES es solo un conector. El lazo se cierra **dentro** de
Orquesta, sin polling externo. Cada PR deja `go build`, `go vet`, `go test ./...`
y `go test -race` en verde.

---

## Secuencia de PRs

### PR 0 — Higiene base (rápido, desbloquea CI)
- T4: copia por valor mutex/Builder en `orquesta-runtime-required-test`.
- T3: data races de tests `orquesta-server` (usar `waitAsyncWorkV0`, no `Sleep`).
- T10: añadir a CI `go vet`, `go test -race`, `staticcheck`, `govulncheck`.
- T5: subir toolchain Go ≥1.25.7 + `golang.org/x/text@latest`.
Sin dependencias. Hacerlo primero para que el resto se valide con `-race` y gates.

### PR 1 — Goal store con lister + persistencia unificada (prerrequisito de T1)
Depende de: PR 0.
- Hacer que el `GoalWorkStateStorePortV0` de producción implemente también
  `GoalWorkStateListPortV0` (`ListGoalWorkStatesV0`). Ref: diseño §Prerrequisito.
- Persistir los goals de **auto-mejora** en el mismo `GoalStateStore`
  (`launchIdleSelfImprovementGoalsV0`), no solo en el tracker.
- Test: `ListGoalWorkStatesV0(ActiveOnly:true)` devuelve goals de auto-mejora y de
  external-work juntos.
Sin esto, el observador activo no ve nada → PR 2 no sirve.

### PR 2 — Observador goal-first residente (núcleo de T1)
Depende de: PR 1.
- Puerto `GoalActiveObservationPortV0` en `orquesta-server` + implementación en el
  codex-stack delegando en `ObserveActiveGoalWorksV0`. Ref: diseño §T1 firmas.
- Ticker `runGoalObservationLoopV0`/`...TickAsyncV0`/`...TickV0` (espejo del
  supervisor, coalescente) arrancado incondicional en `runtime_v0.go`.
- Config `GoalObserverEnabled` (default true) + env
  `ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED`; **gating propio**, NO el del director
  residente.
- Reemplazar `observePendingIdleSelfImprovementGoalV0` (un-goal-por-tick) por el
  ticker multi-goal una vez verificado.
- Verificación clave (diseño §Verificación T1): con director residente OFF y sin
  trabajo externo, Orquesta lanza auto-mejora y la **cierra sola**; ídem un goal
  de `external-work/run`. Sincronizar con `waitAsyncWorkV0`.

### PR 3 — Aviso honesto si no hay observador (cierre de T1)
Depende de: PR 2.
- `external-work/run` goal-first devuelve `next_action=goal_observer_required`
  cuando `GoalObserverEnabled=false` y no hay residente, en vez de `ok` huérfano.
  Punto: `external_work_goal_first_executor_v0.go`.

### PR 4 — Estado vivo `running_live`/`running_stale` (T2)
Depende de: PR 0 (independiente de PR 1–3; puede ir en paralelo).
- Función pura `ClassifyRunLivenessV0` en `orquesta-run-coordinator`, extrayendo
  la lógica viva ya presente en `supervisor_loop_v0.go`. Ref: diseño §T2.
- Cablearla en la proyección de `autoprogramming/status`; **borrar** la copia
  muerta `hasMCPAutoprogrammingLiveProcessSignalV0`.
- Poblar `processObserved` con `ProcessRegistry.ResolveAgentProcessV0`.
- Reconciliar runs `running` sin proceso vivo ni ACK → `running_stale`/cierre.
- Verificación (diseño §Verificación T2): run sin proceso → `running_stale` con
  `recommended_action`; status siempre con cuerpo y acotado.

### PR 5 — Intake de artefactos (T6)
Depende de: nada. Confirmar contra contrato si el ACK puede traer varios ficheros
(`domain_work_delivery_artifact_intake_v0.go`): filtrar por `artifactType` o
reescribir como `if len(ack.Files) > 0`.

### PR 6 — DECISIÓN validación muerta (T7) — antes de borrar nada del grupo B
Depende de: PR 2/PR 4 (para saber qué se usa ya). Por cada bloque del Grupo B
(política de ACK Codex, rails de detalle sensible, señales de proceso vivo,
validación de records de proceso): **cablear** o **borrar**, con test si se cabló.

### PR 7 — Borrado de código muerto obsoleto (T8) + limpiezas (T9)
Depende de: PR 6 (para no borrar lo que se decida cablear) y PR 2 (restos del loop
forzado ya superados). Borrar Grupo A (idle no-causal, fingerprints legacy, CLI
bootstrap-appspec, helpers sueltos, 21 helpers de test) + limpiezas SA4006/SA4009/
SA4031/SA1012/estilo. Objetivo: `staticcheck ./... | grep -c U1000` → 0.

---

## Mapa dependencias (resumen)
```
PR0 ─┬─> PR1 ─> PR2 ─> PR3
     │              └─> PR6 ─> PR7
     ├─> PR4 ──────────┘
     └─> PR5
```

## Definición de hecho (todo el conjunto)
- `go build ./...` OK, `go vet ./...` limpio.
- `staticcheck ./...` sin SA ni U1000; `govulncheck ./...` sin vulns alcanzables.
- `go test ./...` y `go test -race ./...` en verde.
- Demostrable: con director residente desactivado y sin cliente externo, Orquesta
  programa en idle y cierra sus goals; los goals de external-work también cierran;
  `status` distingue `running_live`/`running_stale` y nunca cuelga sin cuerpo.
