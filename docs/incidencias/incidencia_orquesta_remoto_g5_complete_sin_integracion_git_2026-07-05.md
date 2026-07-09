# Incidencia: G5 remoto completo sin integracion Git

Fecha: 2026-07-05.

ID inventario: `BUG-ORQ-20260705-191`.

## Resumen

En el servidor `srv1651826` Orquesta ejecuto el goal G5 del bot LLM del wizard
y escribio un `orquesta_goal_result.v0` con `status=complete`, pero los cambios
quedaron inicialmente en el worktree `pilot-remoto-1` sin commit ni integracion
en la rama principal `trabajo/plataforma-agentes`.

No parecia un fallo de compilacion ni de tests del goal. El fallo observado
estaba en la frontera entre entrega terminal del agente, revision/promocion Git
y sincronizacion remota.

## Evidencia original

- Servidor: `srv1651826`.
- Worktree principal: `/srv/orquesta-self/worktrees/orquesta`.
- Worktree de ejecucion: `/srv/orquesta-self/worktrees/pilot-remoto-1`.
- Goal: `goal-ref-task-autoprogramming-0cc9487d6778-g01`.
- Request: `request-ref-remoto-wizard-bot-llm-20260705-001`.
- Tarea: `task-wizard-bot-llm-g5`.

Resultado durable en el piloto:

```text
/srv/orquesta-self/worktrees/pilot-remoto-1/modulos/orquesta-web/docs/orquesta_goal_result_goal-ref-task-autoprogramming-0cc9487d6778-g01.json
status=complete
summary=Implementado G5 del bot del wizard...
commit_sha=null
```

Estado del piloto:

```text
## pilot-remoto-1
 M cmd/orquesta-server/config_file_v0.go
 M docs/autoprogramacion_orquesta_pendientes_2026-05-23.md
 M docs/diseno_wizard_programacion_2026-07-04.md
 M modulos/orquesta-web/nueva_app_wizard_bot_v0.go
 M modulos/orquesta-web/nueva_app_wizard_bot_v0_test.go
?? cmd/orquesta-server/wizard_bot_config_v0.go
?? cmd/orquesta-server/wizard_bot_llm_v0.go
?? cmd/orquesta-server/wizard_bot_llm_v0_test.go
?? modulos/orquesta-web/docs/checkpoint_started_goal-ref-task-autoprogramming-0cc9487d6778-g01.txt
?? modulos/orquesta-web/docs/orquesta_goal_result_goal-ref-task-autoprogramming-0cc9487d6778-g01.json
```

Estado del worktree principal:

```text
HEAD=885e76b0 docs: mision test de campo OPES real tras G5 (orden del operador)
git status limpio
```

El remoto Git del worktree principal no apuntaba a GitHub:

```text
origin /tmp/orquesta-self.bundle (fetch)
origin /tmp/orquesta-self.bundle (push)
git ls-remote origin trabajo/plataforma-agentes -> fatal: '/tmp/orquesta-self.bundle' does not appear to be a git repository
```

La API publica de Orquesta en `127.0.0.1:19071` devolvia cola vacia:

```text
POST /api/v0/autoprogramming/status
estado=ok
queue.count=0
terminal incluye request-ref-remoto-wizard-bot-llm-20260705-001 status=stopped
```

## Diagnostico

Habia dos problemas relacionados:

1. El goal produjo una entrega terminal de codigo, pero Orquesta no exigia ni
   conservaba un recibo de integracion Git antes de proyectar cola vacia.
2. El worktree principal remoto conservaba `origin` apuntando a un bundle
   temporal, asi que aunque hubiese commit local, ese worktree no podia hacer
   fetch/push directo a GitHub sin reconfiguracion o sin flujo externo de
   sincronizacion.

El problema no fue que GitHub rechazase un push de G5: no habia evidencia de que
se intentara empujar G5. La evidencia apuntaba a que G5 quedo sin commit en el
piloto y que la integracion manual/revisor no se ejecuto o no termino.

## Impacto

- Orquesta podia decir que no quedaba trabajo (`queue.count=0`) aunque quedasen
  cambios de codigo recuperables fuera de Git.
- El operador podia creer que G5 estaba integrado y pasar al test de campo OPES
  real sin tener el soporte LLM opt-in del wizard en la rama principal.
- Si se borraba o rotaba el worktree piloto, se podia perder una entrega
  completa que no estaba en Git.

## Cierre aplicado en checkpoint remoto

- `AutoprogrammingStagingEffectResultV0` declara `integration_status` e
  `integration_receipt_ref`.
- `orquesta-app-codex-stack` normaliza los efectos de promocion a
  `integrated`, `pending_integration` o `blocked_push` y conserva evidencia
  estable; `pending_push` no archiva ni cierra cola.
- `autoprogramming/status` publica `pending_integration` con accion
  `wait_for_integration_receipt` cuando una run esta cerrada causalmente pero
  el candidato de cola sigue no terminal.

## Evidencia de cierre

Tests focales:

- `TestAutoprogrammingStagingPromotionV0DeclaraReciboIntegracionV0`
- `TestCodexStackAutoprogrammingPromotionV0PendingPushExigeReciboIntegracionV0`
- `TestMCPAutoprogrammingStatusExecutorV0RunClosedEnColaPublicaPendingIntegrationV0`

Validacion requerida por el goal:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-web ./cmd/orquesta-server ./modulos/orquesta-autoprogramming ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp`
- `git diff --check`
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp`

## Residual operativo

El codigo de cierre existe en el checkpoint remoto, pero el servidor seguia sin
remote GitHub directo: el checkpoint tuvo que exportarse como bundle para
integrarlo desde un entorno con `origin` canonico. Queda pendiente validar en el
servidor remoto un smoke end-to-end con remote Git canonico o con protocolo
oficial de bundle/push, para que `blocked_push` no dependa de inspeccion manual.

Actualizacion 2026-07-09:

- El protocolo oficial queda en
  `docs/runbooks/protocolo_git_remoto_orquesta_2026-07-02.md`.
- El cierre remoto ya no puede basarse solo en `goal_result status=complete`,
  `queue.count=0` o cambios en un worktree piloto.
- Debe conservar `integration_receipt_ref` o dejar el estado como
  `pending_integration`/`blocked_push` con patch o bundle exportado, summary,
  write-set, pruebas y accion siguiente.
- Si el remoto vuelve a tener `origin` no canonico o sin credenciales, el fallo
  es operativo de integracion Git, no exito del goal.
