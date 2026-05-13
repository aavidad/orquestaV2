# Smoke: servidor temporal y continuidad de estado tras reinicio

Objetivo: validar que `cmd/orquesta-server` puede arrancar con estado temporal,
persistir una decision operativa de cola mediante conectores file-based, parar,
arrancar de nuevo con el mismo `state_dir` y recuperar esa informacion por API.

Este smoke no lanza agentes, no consume cuota Codex y no toca OPES.

## Comando

Desde la raiz del repo:

```bash
./scripts/smoke_orquesta_server_restart_state.sh
```

El script:

- compila un binario temporal de `cmd/orquesta-server`;
- arranca el servidor en `127.0.0.1:0` con `state_dir`, `project_dir` y
  `runtime_dir` temporales;
- llama a `POST /api/v0/runs/queue/priority` para guardar una prioridad de run;
- consulta ranking de cola por el mismo endpoint;
- para el proceso temporal con `SIGINT`;
- arranca otro proceso con el mismo `state_dir`;
- comprueba que el ranking conserva la run;
- actualiza la prioridad tras el reinicio y verifica que el conector sigue
  escribiendo.

## Controles

Variables utiles:

```bash
ORQUESTA_KEEP_SMOKE_DIR=1
ORQUESTA_SMOKE_ROOT=/tmp/orquesta-server-restart-state-manual
ORQUESTA_SMOKE_REQUEST_TIMEOUT_SECONDS=10
```

El script fuerza `ORQUESTA_OPES_BASE_URL=""` en el proceso temporal para evitar
puentes de dominio durante esta prueba.

## Criterio de exito

La salida debe incluir:

- `servidor before listo`;
- `rank_verificado=<run_ref> priority=80`;
- `servidor after listo`;
- otra verificacion `priority=80` tras reinicio;
- verificacion final `priority=95`;
- `agents_launched=false`;
- `opes_touched=false`.

## Alcance

Este smoke cierra la comprobacion basica de continuidad de estado operativo del
servidor residente sin gastar agentes. No sustituye a la prueba real larga de
autoprogramacion con director y workers reales, que debe ejecutarse despues en
un Orquesta temporal y aislado cuando no interfiera con trabajos activos.
