# Incidente de autoprogramacion real

Fecha: 2026-05-12.

## Prueba

Se lanzo Orquesta como servidor residente sobre una worktree aislada:

- worktree: `/home/alberto/Trabajo/orquesta-autoprogramacion-20260512-224047`;
- run: `run-spec-orquesta-req-orquesta-f54f00d4`;
- objetivo pedido: exponer por REST/gateway la validacion de
  autoprogramacion ya existente como tool MCP;
- modelo configurado: `gpt-5.5` con razonamiento `xhigh`;
- agentes iniciales arrancados por Orquesta: `director`, `api`, `i18n` y
  `calidad`.

## Lo que funciono

- Orquesta recibio la orden por `POST /api/v0/apps/director`.
- Arranco cuatro agentes reales Codex en paralelo desde el servidor.
- Persistio registros de proceso, receipts, outbox y run en conectores
  file-based.
- `POST /api/v0/director/stats` mostro proceso, modelo y capacidad cuando se
  pidio `include_process_refs=true`.
- El director genero `director_decisions.json` y el supervisor paso a fase
  `programacion`.
- El operador pudo pedir `stop` por `POST /api/v0/runs/control`.

## Lo que fallo

- El director se desvio del objetivo concreto y propuso microtareas historicas
  ya cerradas o no relacionadas directamente con el endpoint REST pedido.
- El primer agente de programacion modifico solo comentarios en `go.mod` y
  `cmd/orquesta-server/*`, sin cerrar ninguna capacidad real.
- El ACK del agente de programacion marco `failed`, pero el run seguia
  proyectando agentes en vuelo hasta que otro ciclo lo materializara.
- La web de estadisticas no pedia por defecto proceso, uso y cuota; por tanto
  podia ocultar datos necesarios para decisiones humanas o del director.
- La fuente de progreso no emitia agentes en estado normal de trabajo si no
  habia politica de presupuesto activa.

## Decision

La salida de esta prueba no se integra en `master`.

Se corrigen en el proyecto principal tres huecos acotados:

- endpoint REST `POST /api/v0/autoprogramming/validate-request`;
- estadisticas web con proceso, progreso y uso por defecto;
- presupuesto de progreso por defecto en `orquesta-server`.

Queda como trabajo posterior resolver de raiz:

- materializar `stop_requested` en el run visible aunque no haya nueva ejecucion
  de producto;
- validar que el director principal mantenga el objetivo actual y no reutilice
  tareas historicas fuera del alcance;
- impedir que agentes de brainstorming/documentacion hagan cambios fuera de su
  write-set efectivo;
- rechazar microtareas de programacion que no implementen una capacidad
  verificable.
