# Incidencia: cuota Codex agotada sin replan automático

Fecha: 2026-06-13

## Contexto

Durante el cierre de A2 Informática en OPES, varios padres de Orquesta quedaron
sin `agent_ack.json` porque Codex devolvió límite de uso. El wrapper ya generaba
`codex_usage_accounting.json`, pero el supervisor de progreso no leía ese
informe al clasificar un proceso parado sin ACK.

Efecto observado: Orquesta veía el proceso como `no_ack` genérico, no como
`capacity_limited`; por tanto el operador tenía que relanzar manualmente cuando
volvía la cuota.

## Arreglo aplicado

Módulo: `modulos/orquesta-runtime-codex-delivery`.

Cambios:

- `progress_failure_context_v0.go` lee `codex_usage_accounting.json` desde el
  directorio del ACK.
- Si el informe estructurado tiene `quota.status = exhausted`, clasifica el
  fallo como `provider_quota_exhausted`.
- Esa clase emite `BudgetStatus = capacity_limited`, `DecisionRequired = true`
  y evidencias `evidence-ref-provider-quota-exhausted` y
  `evidence-ref-capacity-limited`.
- Se conserva la protección existente: no se clasifica cuota, capacidad o auth
  por texto libre de `stderr/stdout/last_message`, para evitar falsos positivos.

## Pruebas

Comandos ejecutados:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-director ./modulos/orquesta-director-scheduler ./modulos/orquesta-orchestration-core
```

Resultado: OK.

Cobertura añadida:

- Reporte parado sin ACK con `codex_usage_accounting.json` agotado produce
  `capacity_limited`.
- El origen de observaciones de progreso propaga esa señal al supervisor.
- Los tests previos de "no clasificar por texto libre" siguen pasando.

## Operativa

Cuando el servidor Orquesta en ejecución cargue este cambio, los agentes que
caigan por cuota agotada deben entrar en el flujo normal de capacidad limitada:
cierre del agente, replanificación compacta y relanzamiento cuando haya
capacidad disponible.

Si hay agentes activos en una instancia antigua, no reiniciar a mitad de trabajo:
esperar a que terminen o queden parados antes de sustituir el proceso.

## Aplicación OPES 2026-06-13

En A2 Informática se esperó a que terminaran los padres `retryquota` 052-060.
Después se reinició la instancia `127.0.0.1:8792` con el código nuevo. El
arranque requirió `ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop` porque el estado
persistido conservaba runs transitorios activos aunque ya no había procesos
reales. El endpoint `/api/v0/server/status` confirmó `startup_ready` y purga
lógica completada.

La prueba de integración real de cuota agotada queda como tarea separada:
`docs/tarea_codex_quota_auto_replan_2026-06-13.md`.
