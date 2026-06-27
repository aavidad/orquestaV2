# Decisiones

## D-001 goal-first

Codex Goal u otro runtime con goal puede actuar como Director operativo dentro
del trabajo. Orquesta no duplica ese loop. Orquesta compila contrato, reglas,
contexto y criterios de cierre; despues valida evidencias.

## D-002 cierre externo al goal

Un goal `complete` no equivale automaticamente a cierre aceptado. Orquesta debe
validar tests requeridos, artefactos, receipts y politica de dominio antes de
cerrar.

## D-003 loop historico compatible

`app-director-service`, `OperationalDirectorPlanStateV0` y waits por ola/cohorte
siguen existiendo para compatibilidad y casos no migrados. El camino nuevo debe
entrar por `GoalWorkSpecV0` cuando el runtime soporte goal.

## D-004 lifecycle neutral

El ciclo minimo `launch -> persist -> observe -> validate closure -> persist`
vive en `orquesta-goal` como lifecycle neutral por puertos. Esto elimina
duplicacion entre composiciones goal-first sin importar Codex, OPES, HTTP,
filesystem, colas ni `core-workflow`.

`complete` sigue sin ser cierre aceptado: el lifecycle solo persiste resultado y
closure neutral. La composicion decide como reflejar `accepted`, `blocked` o
`needs_rework` en su run, cola o dominio.
