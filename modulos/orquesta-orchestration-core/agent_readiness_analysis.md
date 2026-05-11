# Agent Readiness V0

## Corte implementado

`ExternalProcessAgentLauncherV0` acepta un puerto opcional
`AgentReadinessProbePortV0`. Si el puerto no se inyecta, el comportamiento queda
igual que antes. Si se inyecta, el launcher espera una respuesta `ready` despues
de `LaunchExternalAgentProcessV0` y antes de devolver `AgentLaunchResultV0`.

El probe recibe solo refs opacas: `run_id`, `agent_request_id`, `process_ref`,
`launch_ref`, `readiness_ref` y evidencias. No recibe proveedor, HOME, OAuth,
tokens ni rutas reales.

## Workflow

`orquesta-core-workflow` no expone un comando/evento `AgentReady`. El contrato
actual registra `AgentStarted` con `launch_ref`, `ack_ref` y `readiness_ref`
opacas, y documenta que el core no resuelve esas refs. Por eso no se fuerza un
evento nuevo de workflow en este corte.

## Prueba local

La prueba focal usa un readiness probe fake y un runtime fake que devuelve un
`ProcessRuntimeSnapshotV0` publico. No arranca procesos largos ni depende de
adaptadores concretos.
