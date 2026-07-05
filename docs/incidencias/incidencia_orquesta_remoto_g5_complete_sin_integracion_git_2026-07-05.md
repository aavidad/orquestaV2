# Incidencia: G5 complete sin integracion Git

Fecha: 2026-07-05.

## Sintoma

Un goal de codigo podia devolver resultado durable `complete` y salir de la cola
operativa sin que existiera un recibo de integracion Git verificable. En remoto,
esto permitia declarar terminado el trabajo aunque el cambio no estuviera
integrado en la rama esperada o el push siguiera pendiente.

## Causa

La promocion de staging ya distinguia efectos incompletos como `pending_push`,
pero ese efecto no estaba elevado a contrato de cierre: faltaba un
`integration_receipt` obligatorio y `autoprogramming/status` no publicaba un
estado semantico claro cuando la run estaba cerrada causalmente pero seguia en
cola por integracion pendiente.

## Cierre aplicado

- `AutoprogrammingStagingEffectResultV0` declara `integration_status` e
  `integration_receipt_ref`.
- `orquesta-app-codex-stack` normaliza los efectos de promocion a
  `integrated`, `pending_integration` o `blocked_push` y conserva evidencia
  estable; `pending_push` no archiva ni cierra cola.
- `autoprogramming/status` publica `pending_integration` con accion
  `wait_for_integration_receipt` cuando una run esta cerrada causalmente pero
  el candidato de cola sigue no terminal.

## Evidencia

Tests focales:

- `TestAutoprogrammingStagingPromotionV0DeclaraReciboIntegracionV0`
- `TestCodexStackAutoprogrammingPromotionV0PendingPushExigeReciboIntegracionV0`
- `TestMCPAutoprogrammingStatusExecutorV0RunClosedEnColaPublicaPendingIntegrationV0`

Validacion requerida por el goal:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-web ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp`
- `git diff --check`
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp`
