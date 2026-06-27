# Pruebas locales de Orquesta - 2026-05-25

## Alcance

Este runbook documenta una comprobacion local acotada para confirmar que la
composicion actual de Orquesta puede preparar una tarea de autoprogramacion,
entregar contexto reducido a Codex y arrancar un agente del runtime cuando el
operador ya tiene autenticacion valida.

La prueba es un rail de composicion, no una regla del nucleo. No cambia codigo
productivo, no lee secretos, no convierte refs opacas en rutas ni nombres Git y
no toca OPES.

Refs preservadas como opacas en este corte:

- `worktree-ref-orquesta-local-auth-probe`
- `branch-ref-orquesta-local-auth-probe`
- `task-ref-self-improvement-5d9b27efd87a`

## Frontera

- Orquesta coordina la run, el director, la cola, los agentes y las evidencias.
- El runtime Codex, su autenticacion, su directorio de estado y su proveedor son
  configuracion opt-in del adaptador de composicion.
- El operador valida que las variables del proceso apuntan a una instalacion
  temporal o local autorizada; este runbook no publica valores ni rutas reales.
- Si falta autenticacion, cuota, runtime o puerto de servidor, la prueba debe
  bloquearse como configuracion externa pendiente, no como fallo del core.

## Preparacion segura

1. Arrancar el servidor local temporal o residente con el stack Codex opt-in.
2. Confirmar liveness en `http://127.0.0.1:8787/healthz` y readiness
   operativa en `http://127.0.0.1:8787/api/v0/server/readiness` antes de
   lanzar trabajo externo o automejora.
3. Preparar una solicitud de autoprogramacion de bajo impacto con write-set
   documental acotado y tests requeridos explicitos.
4. Transportar `worktree_ref` y `branch_ref` como strings opacos; no usarlos
   para derivar carpetas, ramas Git ni comandos de shell.
5. No pegar `auth.json`, tokens, cabeceras `Authorization`, rutas privadas,
   prompts ni transcripts en logs o artefactos.

## Comprobacion recomendada

Usar solo API publica del servidor:

```bash
curl -sS -X POST http://127.0.0.1:8787/api/v0/autoprogramming/prepare-run \
  -H 'Content-Type: application/json' \
  --data @prepare-local-auth-probe.json

curl -sS -X POST http://127.0.0.1:8787/api/v0/autoprogramming/goal/observe \
  -H 'Content-Type: application/json' \
  --data '{"run_ref":"RUN_REF"}'

curl -sS -X POST http://127.0.0.1:8787/api/v0/autoprogramming/status \
  -H 'Content-Type: application/json' \
  --data '{"run_ref":"RUN_REF"}'
```

Para runs legacy no migrados puede usarse `autoprogramming/supervise` como
diagnostico/compatibilidad, pero no como avance normal cuando `prepare-run`
devuelve `director_execution_mode=goal_first` o `goal_ref`.
La creacion de nuevos runs legacy desde `prepare-run` requiere
`ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=1` y
`director_execution_mode=legacy_director_loop` en el payload; sin opt-in debe
devolver `autoprogramming_legacy_director_loop_opt_in_required`, y con opt-in
pero sin modo explicito debe devolver
`autoprogramming_legacy_director_mode_required`.

El payload `prepare-local-auth-probe.json` debe declarar:

- `director_execution_mode=legacy_director_loop` solo si se esta probando
  compatibilidad legacy;
- `project_ref` de prueba local;
- `worktree_ref=worktree-ref-orquesta-local-auth-probe`;
- `branch_ref=branch-ref-orquesta-local-auth-probe`;
- write-set documental estrecho;
- tests requeridos que el agente pueda ejecutar sin tocar OPES ni servicios
  productivos;
- reglas compactas para no imprimir secretos ni rutas privadas.

## Criterios de exito

- `prepare-run` devuelve `run_ref`, `workflow_task_refs` y, si aplica,
  `wait_agent_refs` sin exponer detalles privados en la rama legacy. En la rama
  Goal-first devuelve `goal_specs[]` sin `run_ref` legacy ni `continue`.
- El pulso de `supervise` intenta arrancar o continuar solo la run acotada.
- El estado publico muestra agente pendiente, entregado o bloqueado con razon
  reparable.
- Si Codex no esta autenticado, el bloqueo queda atribuido a configuracion del
  proveedor o runtime, con evidencia compacta y sin relanzar trabajos amplios.
- Si Codex arranca, la entrega conserva write-set, tests requeridos y refs
  opacas; el cierre posterior depende de review y evidencias causales.

## Evidencia minima

Registrar solo referencias compactas:

- `run_ref`, `task_ref`, `agent_ref` y `ack_ref`;
- estado publico de `status` sin dumps internos;
- tests requeridos ejecutados con comando exacto, status y exit code;
- razon de bloqueo si el runtime no puede autenticarse o arrancar.

No registrar:

- valores de entorno;
- rutas reales de directorios de usuario;
- contenido de autenticacion;
- prompts, completions o transcripts completos;
- salidas largas de comandos.

## Validacion local del corte documental

Este corte se valida con:

```bash
go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp ./cmd/orquesta-server
```

Ademas, el paquete de agente exige resolver el contexto `required ref_only`
mediante lectura local, consulta al director o evidencia explicita. Para esta
tarea se resuelve por lectura local de `agent_packet.json` y se declara en el
ACK como evidencia requerida, porque la entrada venia con
`required_ref_action=ack_evidence_required`.
