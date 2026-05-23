# Errores de rail observados 2026-05-23

Objetivo: acumular errores reales de rails/validadores para convertirlos en
casos de prueba rapidos antes de compilar servidor o ejecutar Orquesta completa.

Comando rapido:

```bash
./scripts/test_rails_fast.sh
```

La matriz `TestRailRecordConcurrencyGateExternalMatrixV0` incluye tambien una
tabla generada con los terminos historicamente problematicos:
`secret`, `secreto`, `token`, `password`, `credential`, `credencial`,
`api_key`, `access_token`, `refresh_token`, `client_secret`, `oauth`,
`transcript`, `prompt`, `completion`, `raw_text`, `full_text`, `runtime`,
`sql`, `dsn`, `connection`, `conexion`, `table`, `tabla`, `provider`,
`proveedor`, `model`, `home`, `tmux`, `docker` y `git`.

## Formato

```text
ID:
Fecha:
Sintoma:
Campo:
Payload minimo:
Decision:
Test:
Estado:
```

## Casos

```text
ID: RAIL-20260523-001
Fecha: 2026-05-23
Sintoma: `RecordConcurrencyGate: detalle_prohibido`.
Campo: `payload.evidence_refs` / `payload.summary`.
Payload minimo: `secrets_policy`, `oauth-client-secret-policy-doc`, `credential`.
Decision: abrir filtro global por palabras en `RecordConcurrencyGate`; revisar en futuro con sanitizador/clasificador por campo.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/secrets-policy-reference`.
Estado: cubierto
```

```text
ID: RAIL-20260523-002
Fecha: 2026-05-23
Sintoma: `RecordConcurrencyGate: detalle_prohibido` al crear evento.
Campo: `ConcurrencyGateRecorded.payload`.
Payload minimo: evento de gate con refs opacas que contienen `secrets_policy`.
Decision: excluir `ConcurrencyGateRecorded` del filtro global de eventos; mantener validacion estructural del evento.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/secrets-policy-reference`.
Estado: cubierto
```

```text
ID: RAIL-20260523-003
Fecha: 2026-05-23
Sintoma: `director_cycle_step_invalido: step.run.command_effects: run invalido`.
Campo: `run.command_effects` y `run.concurrency_gates`.
Payload minimo: `request-ref-app-completion-loop` dentro de subject refs/proyecciones del gate.
Decision: no pasar `concurrency_gates` ni `command_effects` por filtro global de palabras; son proyecciones/efectos estructurados con refs opacas y hashes.
Test: `TestRailRecordConcurrencyGateExternalMatrixV0/completion-run-ref`.
Estado: cubierto
```

```text
ID: FLAKE-20260523-001
Fecha: 2026-05-23
Sintoma: `go test -count=1 ./...` fallo una vez en `cmd/orquesta-server`: stdout no contenia prompt ejecutado para un agente fake recursivo.
Campo: `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`.
Payload minimo: pendiente de aislar.
Decision: tratar como prioridad de fiabilidad; investigar no determinismo en stdout/orden/concurrencia/estado temporal.
Test: repeticion focal `go test -count=1 ./cmd/orquesta-server -run TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje -v`.
Estado: registrado, pendiente de diagnostico
```

## Candidatos pendientes de convertir en matriz

```text
ID: RAIL-CAND-CORE-EVENTS-001
Origen: revision paralela 2026-05-23.
Casos: `RunBlocked.summary` con `completion`, `token budget`, `prompt policy ref`
o `transcript policy ref`.
Decision pendiente: permitir refs/politicas opacas pero seguir rechazando
contenido crudo (`access_token=...`, keys `prompt`, `raw_text`, `transcript`).
Test futuro: `TestRailEventPayloadExternalMatrixV0`.
```

```text
ID: RAIL-CAND-WORKFLOW-TASK-001
Origen: revision paralela 2026-05-23.
Casos: `WorkflowTaskV0` y `CreateMicrotask` con write-set/context refs de
`modulos/orquesta-runtime`, `provider interface`, `modelado de dominio`,
`git diff` y `token budget`.
Decision pendiente: permitir referencias opacas de arquitectura/repo; reservar
rechazo para secretos efectivos o paths inseguros.
Test futuro: `TestRailWorkflowTaskExternalMatrixV0`.
```

```text
ID: RAIL-CAND-CONTEXT-001
Origen: revision paralela 2026-05-23.
Casos: `ContextBundleRequestV0` con `task_kind=web_application`,
`phase=programming`, `capacity_level=normal` y
`write_set=modulos/orquesta-runtime`.
Decision pendiente: normalizar alias razonables en adaptador/director; no
rechazar el modulo real `orquesta-runtime` por palabra.
Test futuro: matriz rapida en `orquesta-context`.
```

```text
ID: RAIL-CAND-DIRECTOR-AGENT-001
Origen: revision paralela 2026-05-23.
Casos: `DirectorAgentDecisionV0` con `summary` que menciona `codex adapter`,
`runtime`, `model`, `provider`, y `record_review_result` con
`accepted_with_notes`.
Decision pendiente: distinguir alias reparables y refs opacas de proveedor real;
normalizar estados equivalentes fuera del core.
Test futuro: matriz rapida en `orquesta-director-agent`.
```

```text
ID: RAIL-CAND-RUNTIME-001
Origen: revision paralela 2026-05-23.
Casos: rol con espacios (`frontend engineer`), `runtime_kind=local`,
`capacity=normal`, `provider_ref=openai`, `model_ref=gpt-5`.
Decision pendiente: normalizar alias de rol/runtime/capacidad; mantener
proveedor/modelo concretos como refs opacas o adaptador opt-in.
Test futuro: matriz rapida en `orquesta-runtime`.
```

```text
ID: FLAKE-CAND-SERVER-001
Origen: revision paralela 2026-05-23.
Casos: stress de `TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje`
con `-count=50 -shuffle=on`, bloque `codex_wave` con procesos fake y `-race`
en `orquesta-runtime`.
Decision pendiente: crear harness rapido de flakes y esperar cierre real de
stdout/ficheros en vez de asumir que `last_message` implica stdout completo.
Test futuro: `./scripts/test_flakes_fast.sh`.
```
