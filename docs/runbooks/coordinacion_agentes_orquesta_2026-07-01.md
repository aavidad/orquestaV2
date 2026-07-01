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

Estado tras `137e19fd`:

- remoto y local sincronizados;
- BUG-057 cerrado e integrado;
- bugs abiertos principales: BUG-055, BUG-058, BUG-059 y BUG-060;
- siguiente frente recomendado: BUG-060, por ser el mas estrecho
  (`required_test_results` no debe pasar con `evidence_refs=[]`).

Propietario recomendado para BUG-060: remoto. El local debe limitarse a revisar,
integrar y vigilar nuevas incidencias OPES mientras el remoto tenga ese write-set.
