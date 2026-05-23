# orquesta-mcp

Responsabilidad: superficie MCP para que una IA gobierne Orquesta.

Incluye:

- resources;
- prompts;
- tools;
- proyecciones compactas;
- validacion de entrada/salida.

MCP es adaptador inbound. No es el cerebro del sistema.

Estado vigente:

- `orquesta.autoprogramming.prepare_run.v0` prepara un run continuable desde
  una request de autoprogramacion por executor inyectado. No arranca agentes por
  si mismo; la composicion debe supervisar despues con `run_ref` explicito.
- `orquesta.domain_work.v0` es el tool generico para que una IA cree trabajos de
  dominio y entregue artefactos sin conocer OPES, DB ni runtime.
- `/api/v0/autoprogramming/prepare-run` es el bridge HTTP local del executor de
  preparacion cuando una composicion lo inyecta.
- `/api/v0/autoprogramming/status` y `/api/v0/autoprogramming/supervise`
  exponen gestion fina de autoprogramacion sobre puertos inyectados de cola,
  stats y supervisor. No ejecutan runtime ni leen estado concreto por si mismos.
- `/api/v0/domain-work` es el bridge HTTP local del mismo executor.
- OPES se conecta hoy inyectando su cliente REST como adaptador de dominio; si se
  usa MCPO o servidor MCP real, debe envolver estos tools como transporte opt-in,
  no duplicar logica en el nucleo.
