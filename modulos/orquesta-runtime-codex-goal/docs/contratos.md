# Contratos

## CodexGoalStartPacketV0

Paquete publico para iniciar un Codex Goal desde un `GoalWorkSpecV0`.

Contiene:

- `goal_ref`;
- refs de request/proyecto y tipo de trabajo cuando existan;
- `objective`;
- prompt compacto;
- refs de contexto;
- reglas;
- write-set;
- tests requeridos;
- criterios de aceptacion;
- contratos de artefactos;
- evidencias de entrada;
- presupuesto operativo;
- politica de cierre;
- politica de rework.

El prompt no debe superar 32 KiB y el paquete JSON completo no debe superar
64 KiB. Si se excede ese limite, el launcher devuelve `invalid` con
`codex_goal_prompt_too_large` o `codex_goal_start_packet_too_large` antes de
llamar al puerto real.

El paquete incluye `direction_contract` como contrato operativo compacto para
el proveedor: materializar checkpoint temprano dentro del `write_set`, evitar
pegar salidas largas, mantener final/ACK compacto y preservar evidencias
durables. La frontera app-server inyecta ese mismo contrato en `turn/start`
aunque el `prompt` legacy venga incompleto, con `max_text_bytes=16384` y
`thread_read_max_bytes=256 KiB` como limites de ingestion/lectura. El
app-server tambien materializa `checkpoint_started.txt` antes de `turn/start`
cuando hay `write_set` autorizado.

Estos limites no equivalen a un corte duro previo a herramientas internas del
proveedor: Orquesta puede acotar `thread/read`, sanear outputs ya generados y
pedir/replanificar contexto estrecho, pero no puede impedir por este contrato
que una herramienta interna produzca stdout gigante antes de que el backend lo
exponga. Ese cierre requiere soporte explicito del runtime/proveedor o que el
app-server medie la ejecucion real de herramientas.

No contiene HOME, token, OAuth, proveedor, modelo, comando de runtime/local,
ruta absoluta ni transcript. La excepcion deliberada son los comandos de tests
requeridos que ya viajan en `required_tests[].command` dentro del contrato
neutral `GoalWorkSpecV0`: el adaptador los preserva para que el goal pueda
reportar evidencia acotada de esos tests. Esos comandos no definen como se
arranca Codex Goal ni autorizan a meter un backend local en este modulo.

## CodexGoalStarterPortV0

Puerto de composicion que crea el goal real. El adaptador no implementa llamadas
directas a herramientas internas; solo define la frontera.

La composicion `cmd/orquesta-server` aporta implementacion opt-in normal con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`: usa `codex app-server`, no
`codex exec`, y mantiene el transporte fuera de este modulo. `app_server_tmux`
lanza `codex app-server --listen unix://<socket>` dentro de una sesion `tmux`
opaca y habla WebSocket sobre el Unix socket privado. `app_server_proxy` queda
solo como diagnostico breakglass explicito, no como ruta de self-programming.
Para no competir con las sqlite de la sesion Codex principal, la composicion proyecta
`auth.json` y `config.toml` a un `CODEX_HOME` aislado bajo el runtime del
app-server. Si el transporte no esta disponible, la
composicion puede devolver `IssueCode` compacto; el launcher neutral lo conserva
en el `GoalLaunchReceiptV0` invalidado para que el operador vea la causa real.
Si por un fallo de frontera el backend devuelve solo `error` y deja
`IssueCode` vacio, el launcher infiere un codigo compacto para familias
operativas conocidas (`ResetStdio`, tmux salido, permisos/bwrap, auth/cuota,
socket o comando ausente) sin copiar rutas, HOME ni stderr completo al receipt.

## CodexGoalObservationRequestV0

Paquete publico para observar un Codex Goal ya lanzado. Contiene `goal_ref` y,
si existe, `external_goal_ref`. Ambas refs se validan como opacas antes de
llamar al puerto real.

## CodexGoalObserverPortV0

Puerto de composicion que observa el goal real y devuelve estado, resumen,
artefactos, checklist, tests, receipts, planes de rework y evidencias. El
adaptador convierte esa respuesta a `GoalWorkResultV0`, valida refs/status y
rechaza observaciones cuyo `goal_ref` no coincida con el pedido.

En los backends app-server, la observacion usa la `external_goal_ref` persistida
como `threadId`, consulta `thread/goal/get` y, si el goal queda terminal,
consulta `thread/read` con `includeTurns=true`. La respuesta final del agente
debe incluir el marcador o, como fallback durable bajo el write-set,
`docs/orquesta_goal_result_v0.json` con `goal_ref` coincidente:

```text
ORQUESTA_GOAL_RESULT_V0 {"goal_ref":"...","summary":"...","artifact_refs":[],"artifact_paths":[],"materialized_artifacts":[],"checklist":{"expected_refs":[],"completed_refs":[],"missing_refs":[],"evidence_refs":[]},"required_test_results":[],"domain_receipt_refs":[],"rework_plan_refs":[],"evidence_refs":[]}
```

Solo esas refs estructuradas se fusionan como artefactos/evidencias de cierre.
Si faltan, Orquesta conserva el estado observado pero no inventa refs para
aceptar cierre. El archivo durable tiene prioridad sobre un marcador textual
incompleto o invalido.
`materialized_artifacts` es el canal canonico para declarar trabajo parcial o no
publicable: cada item debe incluir path/tipo/estado/evidencias/issues. Si algun
estado no es `valid`, el resultado no debe declararse `complete`; debe quedar
`blocked` con `checklist.missing_refs` y `rework_plan_refs` accionables.
Si el backend falla al observar y devuelve `IssueCode`, el observer neutral lo
conserva en el `GoalWorkResultV0` invalidado.
Si falla sin `IssueCode`, aplica el mismo fallback de clasificacion compacta
que el launcher.

## CodexGoalLauncherV0

Implementa `GoalWorkLauncherPortV0` usando un `CodexGoalStarterPortV0`.

## CodexGoalObserverV0

Implementa `GoalWorkObservationPortV0` usando un `CodexGoalObserverPortV0`.
`complete` observado no equivale a cierre aceptado; el cierre sigue pasando por
`GoalWorkClosureValidatorPortV0` o un validador de composicion equivalente.
