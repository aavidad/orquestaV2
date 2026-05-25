# Informe operativo de cierre de autoprogramacion - 2026-05-25

Alcance de este corte: `cmd/orquesta-server`, `modulos/orquesta-mcp`,
`modulos/orquesta-runtime-codex`, `modulos/orquesta-runtime-worktree` y `docs`.

## Resultado

- La frontera HTTP de autoprogramacion en MCP ahora rechaza cuerpos JSON con
  trailing data o exceso de tamano con codigos publicos compactos.
- Los fallos del executor de `autoprogramming/supervise` ya no publican el
  texto crudo del error interno en HTTP ni en transporte MCP.
- `run-status` y `status` del binario servidor leen respuestas HTTP con limite y
  no incorporan bodies de error del peer al stderr.
- La supervision conserva `idempotency_key` como campo publico trazable en
  request/response. La idempotencia durable de efectos sigue perteneciendo a los
  puertos de cola, workflow, outbox y supervisor inyectados.

## Estado de cola y cierre

La consulta publica de autoprogramacion sigue separando cola y run:
`orquesta.autoprogramming.status.v0` consulta cola por
`orquesta.run_queue.priority.v0` y stats por `orquesta.director.stats.v0`.
Cuando la cola esta vacia o no visible, conserva diagnostico
`queue_empty_or_not_visible` y no recomienda `supervise`/`retry` como accion
segura. Cuando hay candidatos sin stats de run, conserva el aviso
`run_stats_required_for_safe_supervision`.

## Rework de revision

La correccion de rework no relanza otro agente padre sobre la tarea original.
El paquete de rework conserva el mismo write-set cerrado y trae contexto
`required=true`/`mode=ref_only` con accion `ack_evidence_required`; por tanto,
el cierre se valida con lectura local del paquete, documentos vigentes y
receipt explicito en el ACK, no con material inventado.

La prueba obligatoria reejecutada para este rework es:

```bash
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-worktree ./modulos/orquesta-mcp
```

## Bloqueos fuera de este write-set

No se cerro aqui el smoke residente amplio de presupuestos anidados
`run-supervisor -> director-supervisor -> director-supervised-burst`: requiere
editar `modulos/orquesta-server`, `modulos/orquesta-run-supervisor`,
`modulos/orquesta-orchestration-core`, `modulos/orquesta-director-supervisor` y
`modulos/orquesta-director-supervised-burst`, fuera del alcance cerrado de esta
tarea. Queda como pendiente verificable en T68/T69.
