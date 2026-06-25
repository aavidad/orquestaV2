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
