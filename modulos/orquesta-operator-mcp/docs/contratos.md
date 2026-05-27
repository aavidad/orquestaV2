# Contratos: orquesta-operator-mcp

## OperatorMCPCapabilitiesV0

Tipo: resource MCP puro.

Campos:

- `resource_uri`: `orquesta://operator/capabilities/v0`.
- `tools`: lista compacta de tools publicas.
- `opaque_refs`: refs que el cliente debe tratar como tokens opacos.
- `guardrails`: limites publicos para no depender de internos.

Invariantes:

- Una IA descubre capacidades aqui antes de invocar tools.
- El resource no menciona DB, tablas, event-store, HOME, OAuth, proveedor, modelo ni rutas.
- Cada tool declara el conector abstracto que resolvera la operacion.

## OperatorStatusQueryV0

Tipo: tool MCP puro.

Entrada:

- `request_ref`: id opaco de la peticion.
- `subject_ref`: run, flujo, tarea o sistema como ref opaca.
- `status_connector_ref`: conector opaco que sabe consultar estado.
- `include_sections`: secciones compactas solicitadas.

Salida:

- estado compacto para IA;
- `evidence_refs` opacas;
- errores publicos si el conector rechaza la consulta.

Invariantes:

- El tool no lee estado productivo por si mismo.
- El tool no requiere conocer modulos internos ni nombres de tipos Go.
- El resultado debe ser compacto, sin dumps ni datos sensibles.
- Alias razonables de conector como `state`, `estado` o `health` se normalizan
  en adaptadores; refs sensibles o con rutas se rechazan.

## OperatorSupervisedBurstV0

Tipo: tool MCP puro.

Entrada:

- `run_ref`: ref opaca del run.
- `burst_connector_ref`: conector opaco de burst supervisado.
- `max_steps`: presupuesto entero acotado.
- `supervision_ref`: politica o contexto de supervision como ref opaca.

Salida:

- `burst_ref`: ref opaca de la invocacion;
- `executed_steps`;
- `final_action`;
- `trace_refs` compactas.

Invariantes:

- El tool invoca un conector; no ejecuta scheduler, outbox, runtime ni supervisor real.
- `max_steps` debe ser positivo y acotado.
- No hay sleeps, polling, daemons ni reintentos por tiempo.
- Alias razonables como `supervised_burst` o `burst_supervisado` se normalizan
  en adaptadores externos o simulados.

## OperatorPendingOutboxV0

Tipo: tool MCP puro.

Entrada:

- `request_ref`: id opaco de la peticion.
- `subject_ref`: run, flujo o sistema como ref opaca.
- `outbox_connector_ref`: conector opaco que sabe listar pendientes.
- `limit`: maximo acotado de mensajes.
- `include_kinds`: tipos publicos opcionales, sin nombres internos.

Salida:

- `pending_count`;
- `items` con `message_ref`, `kind` y `target_ref` opacos o publicos;
- `watermark_ref`;
- `evidence_refs`.

Invariantes:

- El tool lista por conector; no lee colas, tablas, ficheros ni procesos.
- El resultado es compacto y no contiene payloads completos.
- Si no hay conector operativo, debe devolver error publico, no ejecutar otro flujo.
- Alias publicos como `question`, `consulta`, `pending_outbox` o
  `outbox_pending` se normalizan sin exponer nombres internos.

## OperatorDirectedQueryV0

Tipo: tool MCP puro.

Entrada:

- `query_ref`: ref opaca de consulta.
- `target_ref`: ref opaca del destinatario logico.
- `query_connector_ref`: conector opaco que enruta la consulta.
- `question`: texto compacto sin secretos.

Salida:

- `accepted`: booleano;
- `answer_ref` opcional;
- `next_action`;
- errores publicos.

Invariantes:

- Las consultas cruzan modulos por conectores y refs opacas.
- No se incluyen prompts completos, credenciales, HOME, rutas ni transcripts.
- Si falta contexto, la respuesta publica pide `CONSULTA AL DIRECTOR`.
- Texto con marcadores sensibles se rechaza con error publico compacto.

## Adaptador MCP puro en orquesta-mcp

Tipo: adaptador inbound fino.

Implementacion publicada:

- `orquesta.operator.operations.v0` como resource compacto.
- `ExecuteMCPOperatorStatusToolV0`.
- `ExecuteMCPOperatorBurstToolV0`.
- `ExecuteMCPOperatorOutboxToolV0`.
- `ExecuteMCPOperatorDirectedQueryToolV0`.
- `OperatorMCPConnectorV0` como puerto agregado opcional para estado, burst,
  outbox y consulta dirigida.
- `NewOperatorMCPSimulatedConnectorV0` como conector offline para pruebas y
  demos sin runtime real.

Invariantes:

- El adaptador vive en `orquesta-mcp` y consume solo puertos publicos `OperatorMCP*PortV0`.
- Si falta puerto, devuelve error publico `operator_mcp_port_unavailable`.
- Si el conector rechaza una ref declarada, devuelve
  `operator_mcp_connector_unavailable` sin caer a internals.
- No ejecuta scheduler, DB, outbox, runtime, filesystem, red, HOME ni servidor MCP real.
- El transporte MCP real queda como conector opt-in posterior.
- Hermes, OpenClaw u otro operador real debe implementarse fuera como adaptador
  externo sobre `OperatorMCPConnectorV0`; este modulo no importa esos productos.

## Transporte MCP opt-in v0

Tipo: frontera hexagonal externa.

Contrato consumidor:

- `orquesta-mcp` registra `orquesta.operator.operations.v0` y sus tools en `TransportPortV0`.
- El transporte real debe vivir fuera de `orquesta-operator-mcp` y `orquesta-mcp` como adaptador opt-in.
- Los handlers operativos reciben solo DTOs publicos y llaman a `OperatorMCP*PortV0` inyectados.
- Si se inyecta `OperatorMCPConnectorV0`, `orquesta-mcp` lo usa como fallback
  para los cuatro tools cuando no hay puerto especifico por tool.
- Los adaptadores externos que llamen operadores reales deben publicar
  cancelacion y deadline como errores estables (`operator_mcp_cancelled` y
  `operator_mcp_timeout`) sin filtrar detalles de transporte.

Invariantes:

- Este modulo no implementa red productiva ni servidor MCP real.
- Este modulo no lee DB, outbox, runtime, event-store, filesystem productivo, HOME, OAuth, proveedor ni modelo.
- La IA invoca capacidades por refs opacas y conectores, no por nombres internos de modulo, tabla, proceso o tipo privado.
- La falta de conector se expresa como error publico, no como fallback a internals.
