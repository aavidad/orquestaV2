# Pruebas: orquesta-director-runner

## Ejecutadas Localmente

```sh
go test -count=1 ./modulos/orquesta-director-runner
```

Resultado: `ok` el 2026-05-06.

## Cobertura Contractual

- Scheduler en `waiting`: no llama al workflow.
- Scheduler en `commands_ready`: aplica comandos en orden.
- Scheduler en `needs_director` con comando durable: aplica el comando previo y conserva `needs_director`.
- Comando con outbox: detiene el ciclo y devuelve outbox pendiente.
- `max_outbox` explicito: acumula varios outbox y conserva `outbox_pending`.
- Error workflow: detiene el ciclo y devuelve error publico.
- Progreso ajeno: se filtra antes del scheduler y no bloquea el ciclo.
- Entrada incompleta: rechaza sin efectos.
- Integracion: scheduler real produce capacidad y workflow en memoria refleja evento/outbox.
- T207 no anade cobertura local al runner: la verificacion focal vive en
  `go test -count=1 ./modulos/orquesta-runtime ./modulos/orquesta-orchestration-core`
  y la bateria transversal conserva `go test -count=1 ./modulos/orquesta-director-runner`.
