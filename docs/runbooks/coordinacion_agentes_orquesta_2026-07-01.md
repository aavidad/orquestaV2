# Coordinacion de agentes Orquesta

Fecha: 2026-07-01.

Este runbook fija la coordinacion operativa cuando hay mas de un agente
trabajando sobre Orquesta, especialmente un Codex local y otro Codex remoto.
No sustituye a `AGENTS.md`: lo concreta para evitar solapes de write-set,
commits divergentes y arreglos duplicados.

## Regla principal

Un bug, un propietario, un write-set.

Antes de editar, el agente debe declarar:

- bug o incidencia que va a cerrar;
- propietario: `local`, `remoto` u Orquesta;
- write-set concreto;
- pruebas focales esperadas;
- si necesitara tocar inventario de bugs.

Ningun otro agente edita ese write-set hasta que el propietario publique
commit, bundle/patch o bloqueo explicito.

## Roles vigentes

Mientras exista un Codex remoto persistente en el servidor aislado:

- Remoto: implementa bugs asignados y deja commit local, pruebas, patch/bundle y
  resumen en `/home/berserk/orquesta-inbox` o `/home/berserk/orquesta-logs`.
- Local: observa OPES/local, recoge incidencias nuevas, revisa resultados del
  remoto, integra commits utiles, resuelve conflictos y empuja la rama canonica.
- Orquesta: puede ejecutar trabajo cuando el runtime lo permita, pero no debe
  tocar produccion ni ampliar write-set sin contrato causal.

Si el remoto esta en prompt o sin tarea asignada, queda en espera. No debe
elegir un frente nuevo por su cuenta si hay bugs abiertos solapados con el
trabajo local.

## Handoff minimo

Cada entrega de un agente debe incluir:

- `HEAD` y rama;
- bug cerrado o estado `bloqueado`;
- archivos tocados;
- pruebas ejecutadas;
- riesgos residuales;
- patch/bundle si el trabajo viene de otro worktree;
- confirmacion de que no toco produccion, OPES productivo, root/sudo ni puertos
  externos.

## Sincronizacion

Antes de asignar un bug:

```bash
git status --short --branch
git rev-parse HEAD
```

En remoto aislado:

```bash
cd /srv/orquesta-self/runtime/audit-225a6752-next
git status --short --branch
git rev-parse HEAD
```

Si los `HEAD` no coinciden, se sincroniza antes de editar. Si hay cambios sin
commit, el propietario debe cerrar o documentar su estado antes de que otro
agente toque el mismo write-set.

## Inventario

Todo bug nuevo va primero a `docs/inventario_bugs_orquesta_2026-06-30.md` con:

- area;
- sintoma;
- evidencia;
- hipotesis arquitectonica;
- estado;
- accion.

No se programa una incidencia como parche aislado hasta revisar si repite un eje
arquitectonico ya abierto.

## Codebase y CPU

`codebase-memory-mcp` solo se usa para consultas concretas de grafo. No se usa
por defecto en subagentes ni como busqueda amplia.

Despues de usarlo, revisar procesos:

```bash
pgrep -af codebase-memory-mcp || true
```

Si quedan procesos sin consulta activa, cerrarlos de forma cooperativa y dejar
incidencia si el patron se repite.

## Proximo frente tras este corte

Estado de base tras `879c5125`:

- local, GitHub y remoto aislado deben quedar sincronizados en el `HEAD`
  canonico antes de editar; como minimo debe incluir `879c5125`;
- BUG-059, BUG-060, BUG-061, BUG-062 y BUG-063 estan cerrados;
- BUG-064 cierra solo el contrato puro OPES de calidad de tema;
- bugs abiertos principales: BUG-055, BUG-058, BUG-065, BUG-066 y BUG-069;
- el fichero
  `TAREA_OPES_ORQUESTA_GRUPO_B_GOAL_FIRST_OBSERVE_Y_QA_MINIMOS_2026-07-01.md`
  pertenece a la sesion OPES/local que lo esta alimentando y no debe
  reescribirse desde el remoto salvo handoff explicito.

Propietario recomendado para el siguiente tramo: remoto, pero solo despues de
confirmar `HEAD` canonico y declarar write-set. Frente recomendado:
integracion residual de BUG-058/BUG-067, no repetir BUG-064. Write-set inicial:

- `modulos/orquesta-opes-director`;
- wiring OPES/goal-first estrictamente necesario en `modulos/orquesta-app-codex-stack`
  o `modulos/orquesta-opes-*`;
- tests focales del modulo tocado;
- `docs/inventario_bugs_orquesta_2026-06-30.md` solo para actualizar estado.

BUG-065 y BUG-066 quedan como frentes separados de lifecycle/observacion
goal-first y shutdown; BUG-069 es estado operacional unico. Cada uno requiere
write-set propio antes de tocar servidor, MCP, runtime o web.

El local queda como supervisor/integrador: recoge incidencias de OPES, revisa
patches/bundles del remoto, hace push canonico y no edita el write-set remoto
mientras ese frente este activo. BUG-055 queda separado para una segunda tarea
con write-set propio de discovery/runtime; no mezclarlo con BUG-058 salvo que el
analisis demuestre una misma causa arquitectonica.
