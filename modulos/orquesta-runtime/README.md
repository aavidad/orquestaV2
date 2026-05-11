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

No decide negocio. Ejecuta ordenes del core y devuelve evidencia.
