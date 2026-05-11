# Contexto Codex: orquesta-operator-mcp

Lee primero este archivo y `README.md`. Despues lee solo los docs locales necesarios para tu microtarea.

## Reglas comunes

- Contexto pequeno: este modulo publica una fachada MCP para operadores IA/director.
- Hexagonal siempre: estado, burst supervisado y consultas entran por conectores.
- Ficheros pequenos, contratos compactos y pruebas locales de invariantes.
- No cruces `internal/`, structs privados ni detalles internos de otro modulo.
- Si necesitas informacion de otro grupo, emite una `CONSULTA AL DIRECTOR`.

## Alcance local

- Describir capacidades MCP operativas sin exponer internos.
- Definir tools/resources para consultar estado operativo.
- Definir tools/resources para invocar burst supervisado acotado.
- Definir tools/resources para registrar consultas y respuestas dirigidas.
- Normalizar envelopes, refs opacas, errores publicos y trazas compactas.

## Prohibido

- DB, SQL, filesystem productivo, red real, runtime real, OAuth, HOME, credenciales, proveedor o modelo.
- Leer tablas, event-store, outbox, procesos, PID, secretos, prompts o transcripts.
- Duplicar reglas del director, scheduler, supervisor, runtime, capacity u observability.
- Convertir nombres internos de modulo, tipo o tabla en contrato MCP publico.

## Entrega

Cada cambio debe incluir contrato local y prueba local si toca Go. La documentacion local debe explicar que una IA invoca capacidades por refs opacas y conectores, no por conocimiento interno.
