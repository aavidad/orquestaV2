# Corte supervisor Codex director - 2026-05-18

## Objetivo

El objetivo de este corte es acotar la pieza que falta para que Orquesta pueda
hacer lo que hasta ahora hacia el operador manualmente: arrancar un agente Codex
por el flujo normal y empujar el ciclo con `sigue` hasta que termine.

Este corte no reabre OPES ni mueve reglas al core. Es generico para programacion,
trabajo de dominio y cualquier composicion que use Codex como runtime exterior.

## Implementado

- `modulos/orquesta-app-codex-stack/codex_supervisor_v0.go`
  - `CodexSupervisorAgentLifecyclePortV0`;
  - alias compatible `CodexSupervisorRuntimePortV0`;
  - `SuperviseCodexV0`;
  - primer tick `LaunchV0(ctx)`;
  - ticks posteriores `ContinueV0(ctx, "sigue")`;
  - parada por `done`, `failed`, contexto, error de runtime o `max_ticks`;
  - historial compacto de ticks y snapshots.
- `modulos/orquesta-app-codex-stack/codex_supervisor_v0_test.go`
  - fija que `pending`/`stopped` no son terminales;
  - fija que el supervisor sigue empujando con `sigue`;
  - fija corte por `max_ticks`.
- `modulos/orquesta-app-codex-stack/codex_supervisor_stack_lifecycle_v0.go`
  - adapta `CodexSupervisorAgentLifecyclePortV0` al stack real;
  - con `RunRef`, `LaunchV0` y `ContinueV0` reentran por `DrainRunV0`;
  - sin `RunRef`, reentran por `RunGlobalSupervisorV0`;
  - traduce estados del loop del nucleo a snapshot Codex supervisor;
  - conserva evidencia compacta del drain/supervisor.
- `modulos/orquesta-app-codex-stack/codex_supervisor_stack_lifecycle_v0_test.go`
  - prueba `SuperviseCodexV0` sobre una run existente con tick `launch` y tick
    `continue`;
  - fija que no se relanzan agentes al empujar `sigue`;
  - prueba el camino de supervisor global ya existente;
  - fija el mapeo de estados del nucleo.
- Superficie publica neutral:
  - `orquesta.runs.supervisor.v0` en `orquesta-mcp`;
  - `POST /api/v0/runs/supervise`;
  - gateway neutral sin referencias a Codex;
  - executor `CodexStackRunSupervisorExecutorV0` solo en
    `orquesta-app-codex-stack`.

## Manejo real de agentes

Orquesta no gestiona agentes por stdin. El flujo real es:

- el director o replan emite outbox `LaunchRuntimeAgent`;
- `agentBatchDispatcherV0` usa `ExternalProcessAgentBatchExecutorV0`;
- el batch executor resuelve spec Codex, lanza proceso, registra `AgentStarted` y
  `AgentProcessRegistryRecordV0`;
- `CodexReceiptRecordingSpecResolverV0` registra descriptor ACK por agente;
- `CodexDeliveryObservationSourceV0` lee `agent_ack.json` y devuelve
  observaciones;
- `DrainRunV0` aplica observaciones y reentra `ContinueAppDirectorV0`;
- progreso parado/lento entra por `ProgressSupervisionCandidateProviderV0`;
- `AssessmentReplanSourceV0` puede pedir `replace_agent`, que acaba de nuevo en
  outbox `LaunchRuntimeAgent`.

Por eso `ContinueV0("sigue")` queda como puerto de borde de ciclo de agente. El
adaptador de stack ya avanza supervisor/drain/replan/outbox usando esas piezas,
sin crear un canal paralelo.

## Lo que no esta cerrado

- No existe todavia smoke real de recursion padre/hijo/nieto con parent refs,
  presupuesto global, profundidad/fanout y review causal completa.
- El supervisor global del servidor ya reentra runs por intervalo. La API
  `POST /api/v0/runs/supervise` permite pedir una pasada acotada desde la app sin
  cambiar el nucleo ni crear canal paralelo.
- Desde el 2026-05-22, `POST /api/v0/autoprogramming/prepare-run` encola la run
  preparada en la cola global del stack Codex. Por tanto, el supervisor
  residente o una pasada de `/api/v0/runs/supervise` sin `run_ref` ya cubren el
  camino `prepare-run -> cola global -> supervisor` con Codex fake en el smoke
  `AUTOPROGRAMMING-SUPERVISED-FAKE`.

## Validacion

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexSupervisor(StackLifecycle|RuntimeState)|TestCodexSupervisorV0'
go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway
go test -count=1 ./modulos/orquesta-app-codex-stack
```

## Siguiente corte recomendado

El siguiente corte no debe reimplementar agentes. Debe usar
`POST /api/v0/runs/supervise` / `CodexSupervisorStackLifecycleV0` y cerrar el
smoke real:

1. `LaunchV0` crea/encola la run por el flujo existente o recibe `RunRef`.
2. `ContinueV0("sigue")` ejecuta el adaptador de stack.
3. Si el tick decide otro agente, este sale por outbox `LaunchRuntimeAgent`.
4. La evidencia se observa por ACK/progress existentes.
5. El smoke opt-in prueba que Orquesta arranca, continua y termina sin que el
   operador pulse `sigue`.
