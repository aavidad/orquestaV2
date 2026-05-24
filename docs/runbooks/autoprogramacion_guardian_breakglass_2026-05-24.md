# Guardian break-glass de autoprogramacion

Estado: base implementada en `cmd/orquesta-guardian`.

## Objetivo

Evitar que una automejora rota deje a Orquesta sin capacidad de repararse. El
guardian vive fuera del servidor Orquesta y trabaja con binario candidato,
`last_good`, healthcheck temporal y paquete de reparacion durable.

## Flujo

1. Compila el candidato en un path temporal.
2. Ejecuta tests requeridos.
3. Arranca el candidato contra estado/runtime temporal y comprueba `/healthz`.
4. Si todo pasa, guarda el binario actual como `last_good` y promociona el
   candidato de forma atomica.
5. Si falla build, tests o healthcheck, no toca el binario vivo, escribe
   manifest y crea `repair_packet.json`.
6. Si se configura `--repair-command`, lo ejecuta con
   `ORQUESTA_GUARDIAN_REPAIR_PACKET` para lanzar un reparador externo opt-in.

## Ejemplo

```bash
go run ./cmd/orquesta-guardian check-promote \
  --project-dir /home/alberto/Trabajo/orquesta \
  --state-dir /home/alberto/Trabajo/.orquesta-control/orquesta/guardian \
  --current-bin /home/alberto/Trabajo/.orquesta-control/orquesta/autoprogramacion-servidor-2026-05-23/bin/orquesta-server-latest \
  --test-command 'go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server' \
  --test-command 'go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-runtime-worktree'
```

## Reparador externo opt-in

El guardian no mete Codex ni proveedor en el nucleo. Para reparar con agente hay
dos opciones opt-in.

Opcion generica: configurar `--repair-command`. Ese comando recibe:

- `ORQUESTA_GUARDIAN_REPAIR_PACKET`
- `ORQUESTA_GUARDIAN_FAILURE_PHASE`
- `ORQUESTA_GUARDIAN_PROJECT_DIR`

Opcion Codex: pasar `--repair-codex`. El guardian crea un prompt Markdown junto
al `repair_packet.json` y lanza un solo agente externo con:

```bash
<current-bin> codex-launch-wave --agents 1 --prompt-file <repair.md>
```

El binario usado debe ser el ultimo bueno o el binario vivo que todavia tenga el
comando `codex-launch-wave`.

## Shutdown

Existe shutdown controlado por `modulos/orquesta-server-shutdown` y endpoint
`POST /api/v0/server/shutdown`. Para un restart de guardian no debe usarse el
atajo `orquesta-server stop` como unica pieza, porque hoy solicita
`forced=true`.

`cmd/orquesta-guardian shutdown-server` pide shutdown cooperativo por defecto
(`forced=false`), espera `shutdown_ready=true` y despues puede senalizar el PID
si se pasa `--server-pid`. El `deadline` es la hora limite del cierre ordenado:
se calcula desde el inicio usando `--shutdown-timeout`. Hasta ese momento los
agentes pueden escribir checkpoint/handoff y drenar. Si vence el timeout y no
hay cierre suficiente, el guardian escala a `forced=true`. `--now` salta esa
espera y pide forzado desde el principio.

Autoridad: el shutdown se solicita como `orquesta-director`. Los agentes no
pueden iniciar shutdown; solo preparan checkpoint/ACK y el Director decide si
continua la espera o fuerza tras el deadline.

Ejemplo:

```bash
go run ./cmd/orquesta-guardian shutdown-server \
  --project-dir /home/alberto/Trabajo/orquesta \
  --state-dir /home/alberto/Trabajo/.orquesta-control/orquesta/guardian \
  --current-bin /home/alberto/Trabajo/.orquesta-control/orquesta/autoprogramacion-servidor-2026-05-23/bin/orquesta-server-latest \
  --server-addr 127.0.0.1:8090 \
  --server-pid 12345 \
  --shutdown-timeout 2m
```
