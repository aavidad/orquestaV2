# Incidencia: goal-first OPES sin checkpoint durable por tema

Fecha: 2026-07-02.

## Sintoma

En varias olas OPES los agentes goal-first producian artefactos utiles y QA
local suficiente, pero algunos goals seguian activos o reescribiendo tras una
entrega aparentemente terminal. El registro por tema podia ver artefacto y
receipt, pero no una evidencia estructurada de que el goal hubiera dejado un
checkpoint durable por ese tema.

## Causa

`update_topic_registry` ya publicaba `settlement_status` por tema, pero no
diferenciaba rutas legacy de rutas `goal_first` que necesitan checkpoint o
resultado durable antes de declarar `settled_text` o `settled_final`.

## Mitigacion

El director OPES ahora proyecta un contrato de lifecycle:

- `goal_first_lifecycle_status`
- `goal_first_lifecycle_reason`
- `goal_first_lifecycle_contract`
- `goal_first_checkpoint_refs`
- `goal_first_heartbeat_refs`

Si un artefacto OPES viene de `goal_first` y pretende asentarse como texto o
paquete final sin checkpoint durable, el registro queda `not_settled` con
`pending_refs=goal-first-topic-checkpoint-required` y rework causal de
Director. Con checkpoint presente, conserva el settlement normal.

El mismo cambio alinea `orquesta-opes-director` con el cierre estricto del
stack: el manifest final OPES exige tambien evidencias y QA de banco de
preguntas publicable y paquete tutor.

## Evidencia

- `TestProduceOPESCausalJobsV0GoalFirstTextoQAPassSinCheckpointNoAsientaTemaV0`
- `TestProduceOPESCausalJobsV0GoalFirstTextoQAPassConCheckpointAsientaTemaV0`
- `TestProduceOPESCausalJobsV0PaqueteFinalSinBancoYTutorNoLiberaRegistroV0`
- `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-topic-registry ./modulos/orquesta-app-codex-stack`

## Alcance

Avance sobre `BUG-ORQ-20260701-058` y `BUG-ORQ-20260701-066`. No cierra los
residuales de runtime: enforcement fuerte de write-set/checkpoint temprano,
reconciliacion tras cortes externos y smoke OPES temporal largo.
