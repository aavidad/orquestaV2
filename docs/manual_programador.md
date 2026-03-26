# Orquesta: Manual del Programador

Este manual está dirigido a desarrolladores que desean integrar nuevos agentes en Orquesta, extender sus funcionalidades o consumir sus APIs.

## 1. Arquitectura del Agente
Orquesta es **LLM-agnostic**. Un agente se integra mediante:
- **Runtime Mailbox:** El canal de comunicación principal via archivos o API.
- **Checkpoints:** El agente debe emitir señales periódicas de su estado para permitir el *Time Travel Debugging*.

## 2. Desarrollo con el Servidor de Orquesta
La vía operativa oficial es el servicio HTTP local de Orquesta.
- **URL por defecto:** `http://127.0.0.1:16543`
- **Política:** CLI, web y automatismos deben delegar por defecto en el servidor/daemon. El modo local queda solo para recuperación explícita.
- **MCP:** sigue siendo una línea arquitectónica abierta (`OP-088`), no el punto principal de integración para desarrollo diario.

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
- **Servidor primero:** si la operación existe por API/daemon, no la implementes ni la uses en local.
- **Evita el Acceso Directo a SQL:** usa servicios, CLI o API de Orquesta; el acceso directo a persistencia queda solo para diagnóstico o recuperación excepcional.
- **Contratos (OP-073):** Si cambias una interfaz Go pública, debes crear una OP antes de mergear.
- **Voto tardío:** `propuestasapp.Service.VoteDetail` permite a un agente votar en propuestas cerradas si no tiene posición definitiva registrada. El consenso no se re-evalúa. Ver [voto_tardio_agentes.md](voto_tardio_agentes.md).

---
*Documentación para Desarrolladores - v1.0*
