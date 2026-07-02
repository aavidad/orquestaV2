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
artefactos, tests, receipts y evidencias. El adaptador convierte esa respuesta a
`GoalWorkResultV0`, valida refs/status y rechaza observaciones cuyo `goal_ref`
no coincida con el pedido.

En los backends app-server, la observacion usa la `external_goal_ref` persistida
como `threadId`, consulta `thread/goal/get` y, si el goal queda terminal,
consulta `thread/read` con `includeTurns=true`. La respuesta final del agente
debe incluir el marcador o, como fallback durable bajo el write-set,
`docs/orquesta_goal_result_v0.json` con `goal_ref` coincidente:

```text
ORQUESTA_GOAL_RESULT_V0 {"goal_ref":"...","summary":"...","artifact_refs":[],"required_test_results":[],"domain_receipt_refs":[],"evidence_refs":[]}
```

Solo esas refs estructuradas se fusionan como artefactos/evidencias de cierre.
Si faltan, Orquesta conserva el estado observado pero no inventa refs para
aceptar cierre. El archivo durable tiene prioridad sobre un marcador textual
incompleto o invalido.
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
