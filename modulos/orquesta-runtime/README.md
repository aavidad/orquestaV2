# orquesta-runtime

Responsabilidad: ciclo de vida de agentes y conectores runtime.

Incluye:

- launch;
- resume;
- mailbox delivery;
- readiness;
- progress heartbeat compacto;
- handles;
- checkpoints;
- lifecycle hooks;
- conectores CLI/MCP/API/local/remoto;
- conector de proceso local controlado para pruebas e2e;
- multi-HOME;
- identidad operativa.
- contrato experimental opt-in de rotacion por handoff durable
  (`RuntimeSessionRotationRequestV0`).

No decide negocio. Ejecuta ordenes del core y devuelve evidencia.
