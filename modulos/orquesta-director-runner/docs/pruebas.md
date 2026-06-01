# Pruebas: orquesta-director-runner

## Ejecutadas Localmente

```sh
go test -count=1 ./modulos/orquesta-director-runner
```

Resultado: `ok` el 2026-06-01.

## Cobertura Contractual

- Scheduler en `waiting`: no llama al workflow.
- Scheduler en `commands_ready`: aplica comandos en orden.
- Presupuesto de comandos: si el plan trae mas comandos que `max_commands`,
  aplica solo el prefijo presupuestado sin exigir modo programming.
- Scheduler en `needs_director` con comando durable: aplica el comando previo y conserva `needs_director`.
- Comando con outbox: detiene el ciclo y devuelve outbox pendiente.
- `max_outbox` explicito: acumula varios outbox y conserva `outbox_pending`.
- Error workflow: detiene el ciclo y devuelve error publico.
- Progreso ajeno: se filtra antes del scheduler y no bloquea el ciclo.
- Tick preparado: el runner entrega al scheduler el snapshot/candidates ya
  construidos por capas superiores, sin crear un conector propio.
- Entrada incompleta: rechaza sin efectos.
- Event-store workflow: el adaptador persiste eventos por puerto, recarga el run
  por replay e ignora reaplicaciones idempotentes sin anexar duplicados.
- Integracion: scheduler real produce capacidad y workflow por event-store
  refleja evento/outbox al recargar por replay.
- DCR-007 no anade conector local: queda reconciliado fuera del runner y
  cubierto por la prueba de pass-through del tick preparado.
- DCR-009 no anade loop local: queda reclasificado fuera del runner. La parte
  multi-step neutral vive en `orquesta-director-cycle` y se cubre con
  `go test -count=1 ./modulos/orquesta-director-cycle`.
- T207 no anade cobertura local al runner: la verificacion focal vive en
  `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-orchestration-core`
  y la bateria transversal conserva `go test -count=1 ./modulos/orquesta-director-runner`.
