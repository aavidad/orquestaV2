# Incidencia: G5 remoto completo sin integracion Git

Fecha: 2026-07-05.

ID inventario: `BUG-ORQ-20260705-191`.

## Resumen

En el servidor `srv1651826` Orquesta ejecuto el goal G5 del bot LLM del wizard
y escribio un `orquesta_goal_result.v0` con `status=complete`, pero los cambios
quedaron en el worktree `pilot-remoto-1` sin commit ni integracion en la rama
principal `trabajo/plataforma-agentes`.

No parece un fallo de compilacion ni de tests del goal. El fallo observado esta
en la frontera entre entrega terminal del agente, revision/promocion Git y
sincronizacion remota.

## Evidencia

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

El remoto Git del worktree principal no apunta a GitHub:

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

Hay dos problemas relacionados:

1. El goal produjo una entrega terminal de codigo, pero Orquesta no exige ni
   conserva un recibo de integracion Git antes de proyectar cola vacia.
2. El worktree principal remoto conserva `origin` apuntando a un bundle temporal,
   asi que aunque hubiese commit local, ese worktree no podria hacer fetch/push
   directo a GitHub sin reconfiguracion o sin el flujo externo de sincronizacion.

El problema no es que GitHub rechazase un push de G5: no hay evidencia de que se
intentara empujar G5. La evidencia apunta a que G5 quedo sin commit en el piloto
y que la integracion manual/revisor no se ejecuto o no termino.

## Impacto

- Orquesta puede decir que no queda trabajo (`queue.count=0`) aunque queden
  cambios de codigo recuperables fuera de Git.
- El operador puede creer que G5 esta integrado y pasar al test de campo OPES
  real sin tener el soporte LLM opt-in del wizard en la rama principal.
- Si se borra o rota el worktree piloto, se pierde una entrega completa que no
  estaba en Git.

## Criterio de arreglo

- Un goal de codigo no debe cerrar operacionalmente como trabajo terminado si
  falta un `integration_receipt` o equivalente cuando la tarea exige integracion.
- El recibo debe declarar como minimo:
  - `commit_sha`;
  - rama destino;
  - worktree/ref origen;
  - estado de push (`pushed`, `pending_push`, `blocked_push`, `not_required`);
  - revisor o mecanismo de promocion;
  - pruebas ejecutadas tras integrar.
- Si el agente escribe `status=complete` pero no hay commit/integracion, las
  superficies publicas deben proyectar `pending_integration` o `blocked_push`,
  no cola vacia cerrada.
- El servidor remoto debe validar en preflight que el remote Git canonico esta
  disponible o declarar explicitamente `remote_push_unavailable`.

## Tests esperados

- Test unitario/status: resultado durable `complete` con cambios de codigo y
  `commit_sha` ausente se proyecta como `pending_integration`.
- Test de promocion: si `git ls-remote` falla para el remote canonico, el run
  queda `blocked_push` con accion clara.
- Smoke remoto: goal escribe cambios en worktree piloto, integracion falta, y
  `/api/v0/autoprogramming/status` conserva el run visible hasta que exista
  recibo de integracion o bloqueo explicito.

## Workaround operativo

Antes de lanzar el test OPES real tras G5, recuperar o integrar manualmente el
diff de `/srv/orquesta-self/worktrees/pilot-remoto-1`, ejecutar las pruebas G5
en el worktree principal, crear commit y sincronizar con GitHub desde un entorno
con remote correcto.
