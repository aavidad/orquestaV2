# Smoke OPES External Work Con Agente Real

## Vigencia

Runbook historico de 2026-05-13. No describe la ruta normal vigente. Para
reejecucion con efectos, usar OPES temporal confirmado, Orquesta goal-first con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`, observacion por goal y sin loop
legacy salvo opt-in historico explicito.

Objetivo: validar el recorrido productivo minimo OPES -> Orquesta -> agente
real -> artefacto OPES sin usar DB, ficheros internos ni subagentes manuales.

## Alcance

Este smoke no sustituye al smoke directo `domain-work`. Prueba otra frontera:

1. OPES crea `topic` y `chapter` reales.
2. OPES crea un job externo `draft_content_block`.
3. Orquesta recibe el trabajo por `/api/v0/external-work/run`.
4. En la ruta vigente, Orquesta crea un contenedor goal-first y lanza Codex Goal
   por `app_server_tmux`.
5. La observacion por goal transforma evidencias/artefactos causales en
   `submit_artifact`.
6. OPES recibe un artefacto y materializa bloque si el dominio lo permite.

## Ejecucion

OPES debe estar levantado y exponer su API publica:

```bash
ORQUESTA_OPES_AGENT_SMOKE_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
OPES_BASE_URL=http://127.0.0.1:18082 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_REASONING_EFFORT=xhigh \
ORQUESTA_OPES_AGENT_SMOKE_TIMEOUT_SECONDS=900 \
scripts/smoke_opes_external_work_agent_real.sh
```

El script conserva evidencias por defecto bajo:

```text
/tmp/orquesta-opes-agent-smoke/<SMOKE_ID>/out
```

## Ejecucion Validada

El 2026-05-13 se ejecuto contra OPES en `http://127.0.0.1:18080` con
`ORQUESTA_CODEX_MODEL=gpt-5.5` y `ORQUESTA_CODEX_REASONING_EFFORT=xhigh`.

Resultado:

- `smoke_id`: `20260513T200104Z`
- `job_ref`: `8f7b3cf58599efd0e2c88561c1dfe220`
- `run_ref`: `run-external-work-opes-8f7b3cf58599efd0e2c88561c1dfe220-opes-job-8f7b3cf58599efd0e2c88561c1dfe220`
- `stats_response_final.json`: `tasks_total=1`, `tasks_delivered=1`,
  `agents_started=1`, `agents_delivered=1`, `agents_failed=0`.
- OPES recibio 1 artefacto y materializo 1 bloque en
  `pendiente_revision`.

Incidencia detectada y corregida despues de la ejecucion: el comando
`orquesta-server` usaba defaults de debug para progreso Codex
(`8` ticks de `stalled` con intervalo de 2s). El agente estaba trabajando y
termino bien, pero las estadisticas intermedias lo marcaban como
`needs_attention`. La composicion productiva queda ajustada a 300 ticks por
defecto y 10 minutos sin actividad.

## Evidencias Esperadas

- `external_work_run_response.json` contiene `run_ref` y
  `director_question_ref`.
- `opes_job_response.json` contiene `job.id` y `execution_mode=external`.
- `stats_response_final.json` muestra el job externo con proceso, progreso y
  uso si el agente llego a arrancar.
- `opes_artifacts_final.json` o `opes_blocks_final.json` contiene al menos un
  resultado.
- `summary.txt` incluye `run_ref`, `job_ref`, conteo de artefactos y bloques.

## Guardas

- El smoke exige `ORQUESTA_OPES_AGENT_SMOKE_CONFIRM=1` porque consume cuota real.
- Orquesta arranca un servidor temporal y lo apaga por
  `/api/v0/server/shutdown`.
- El smoke no llama a `/api/v0/apps/director`; si aparece un director inicial
  en runtime, la prueba debe considerarse fallida aunque OPES reciba artefacto.
- El paquete enviado al agente incluye temario, esquema, objetivo, contexto
  vecino, fuentes y longitud esperada; no se manda un parrafo aislado.
- Las refs siguen siendo opacas y compactas.
- Si no aparece artefacto antes del timeout, el fallo es valido: hay que leer
  `stats_response_*.json`, logs del servidor y runtime del agente conservados.

## No Objetivos

- No prueba el ensamblado completo de un tema de 50 folios.
- No evalua calidad academica final; solo valida orquestacion real y entrega.
- No selecciona modelos dentro del core: modelo y esfuerzo entran por entorno.
