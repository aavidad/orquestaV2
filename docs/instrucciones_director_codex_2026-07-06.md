# Instrucciones del director para Codex - 2026-07-06

Autor: Claude (director/revisor). Fuente: `docs/auditoria_diseno_estructural_2026-07-06.md`
(leerla antes de empezar: cada tarea trae ahi la evidencia y el porque).

Cola ordenada por prioridad. Reglas de siempre: un commit por tarea, tests
requeridos verdes, `git diff --check`, actualizar bitacora al cerrar cada una.
Regla nueva vinculante (P5, aplicar desde YA en todas estas tareas): la
bateria requerida incluye los paquetes DEPENDIENTES transitivos del write-set,
no solo los tocados; minimo fijo al tocar `orquesta-director-*` u
`orquesta-state-file`: incluir `./modulos/orquesta-orchestration-core`.

## TAREA-D1 (P2, cierre): accepted nunca invisible

Un `prepare-run` con `accepted=true` debe ser siempre observable en
`autoprogramming/status` o reconciliarse solo. Reproducir el caso
`request-ref-remoto-telegram-nollm-runtime-20260705-001` (respuesta con
`workflow_task_refs` + bloque `continue` legacy y sin entrada en status),
corregir la proyeccion/reconciliacion, y test: todo accepted aparece en
status o produce entrada durable con reason code en un tick.
Ref: `docs/incidencias/incidencia_orquesta_telegram_nollm_accepted_invisible_2026-07-05.md`.

## TAREA-D2 (P1): contrato unico de presupuestos

Crear un punto unico de constantes de presupuesto de eventos/payload
(pagina, lectura total, maximo por run, payload scheduler) y hacer que
`orquesta-state-file`, `orquesta-orchestration-core`,
`orquesta-app-director-service` y `orquesta-director-scheduler` lo consuman.
Test de coherencia: pagina <= lectura total <= maximo store; snapshot
compactado <= payload scheduler. No cambiar los valores actuales, solo
unificarlos (hoy: 250/1000 pagina, 10000 lectura, 20000 store, 256KiB
scheduler).

## TAREA-D3 (P5): bateria requerida derivada de dependencias

El launcher goal-first calcula los paquetes afectados por dependencia
inversa del write-set (equivalente a `go list -deps` invertido) y los anade a
`required_test_results` del contrato del goal. Test: un goal que declare
write-set en `orquesta-director-tick-input` exige orchestration-core; uno en
`orquesta-web` no arrastra el mundo entero (limitar profundidad o lista de
modulos de plataforma).

## TAREA-D4 (P3): auditoria de emisores sin dedupe semantico

Inventariar todos los emisores de `AgentWorkAssessed`,
`DirectorQuestionRaised` y `ReplanDecisionRecorded` invocados desde bucles de
tick/supervision (fuera del drain ya corregido en ec07bf300). Para cada uno:
o clave semantica estable (patron de `liveAgentReconciliationSemanticDigestV0`)
o justificacion escrita de por que no puede duplicar. Test por emisor
corregido: dos ticks sin cambio causal no producen segundo evento.

## TAREA-D5 (P4): gate comun de compactacion de lanes

Extraer el gate de tamano del carril progress
(`progressLaneCompactionThresholdBytesV0`, 3c323763c) a un unico punto y
aplicarlo a `compactTickInputSnapshotForDeliveriesV0`,
`...ForPhaseArtifactsV0` y `...ForReviewGateV0`: con input pequeno el
snapshot queda intacto. OJO: hay tests actuales que fijan el filtrado
incondicional de esas lanes (p.ej. asserts de `Deliveries` filtradas con
input de 16KiB); hay que actualizarlos al nuevo contrato con-gate, y anadir
por cada lane el test espejo de
`TestBuildDirectorSchedulerTickInputV0CarrilProgressPequenoConservaSnapshotCompletoV0`.
Antes de tocar, correr orchestration-core en verde como linea base (regla P5).

## TAREA-D6 (P7): reason codes para estados intermedios

Sustituir las redacciones libres de los placeholders de progreso
("started", "checkpoint_started; implementacion pendiente",
"in_progress_checkpoint_materializado"...) por un reason code de catalogo
(p.ej. `checkpoint_started`) en el resultado durable, manteniendo el texto
libre solo como campo informativo. Actualizar observe/status para discriminar
por reason code. Test: un placeholder nunca se distingue por substring de
texto libre.

## TAREA-D7 (P8): runbook ejecutable de arranque/parada del servidor

Script unico (`scripts/orquesta_server_ctl.sh` o similar) con usuario fijo,
rutas del perfil remoto y verificacion post-arranque; el arranque falla con
reason code distinto para permiso-denegado vs fichero-corrupto (hoy
`codex_receipt_descriptor_file_store: read_failed` era un chown pendiente).

## Al terminar cada tarea

Anotar en `docs/bitacora_correccion_pericial_2026-07-03.md` seccion nueva con
refs de commit y tests. Claude revisa cada cierre (rol observador: diseno y
estructura; no programara salvo emergencia).
