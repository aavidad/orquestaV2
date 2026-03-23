# Orquesta: Manual del Programador

Este manual está dirigido a desarrolladores que desean integrar nuevos agentes en Orquesta, extender sus funcionalidades o consumir sus APIs.

## 1. Arquitectura del Agente
Orquesta es **LLM-agnostic**. Un agente se integra mediante:
- **Runtime Mailbox:** El canal de comunicación principal via archivos o API.
- **Checkpoints:** El agente debe emitir señales periódicas de su estado para permitir el *Time Travel Debugging*.

## 2. Desarrollo con el Servidor MCP (OP-088)
Puedes interactuar con Orquesta usando el protocolo **Model Context Protocol**.
- **Endpoint:** `http://localhost:3000/mcp`
- **Herramientas Útiles:**
  - `lista_propuestas`: Consulta el estado de la gobernanza desde tu propio código.
  - `enviar_evento`: Inyecta eventos personalizados en el flujo de trabajo.

## 3. El Protocolo A2UI (OP-094)
Para que tu agente muestre interfaces en el dashboard, debes enviar un mensaje con el siguiente formato:
```json
{
  "type": "a2ui_render",
  "component": "MetricCard",
  "props": {
    "label": "Eficiencia de Refactor",
    "value": "94%"
  }
}
```

## 4. Memoria de Entidades (Beads)
No guardes conocimiento crítico solo en el prompt. Persístelo en la tabla `entidades_memoria`:
- **Lectura:** El sistema te inyectará las perlas relevantes al inicio.
- **Escritura:** Usa la herramienta `verificar_entidad` para actualizar el conocimiento compartido.

## 5. Mejores Prácticas
- **Evita el Acceso Directo a SQL:** Usa siempre el adaptador de `pkg/db`.
- **Contratos (OP-073):** Si cambias una interfaz Go pública, debes crear una OP antes de mergear.

---
*Documentación para Desarrolladores - v1.0*
