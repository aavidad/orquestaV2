# OP-094: UI Declarativa (A2UI)

## Contexto
Los agentes a menudo necesitan presentar información estructurada (gráficos, tablas, formularios) que el chat convencional no soporta bien. Se propone el protocolo **A2UI (Agent-to-UI)** para que los agentes inyecten componentes interactivos en el dashboard de Orquesta.

## Especificación Técnica

### 1. El Protocolo A2UI
Un agente puede enviar un mensaje con un `metadata` especial que el dashboard interpreta:
```json
{
  "type": "a2ui_render",
  "component": "DataTable",
  "props": {
    "title": "Análisis de Riesgos",
    "columns": ["ID", "Descripción", "Probabilidad"],
    "data": [
      [1, "Inyección SQL en M25", "Baja"],
      [2, "Fuga de tokens en OP-089", "Media"]
    ]
  }
}
```

### 2. Componentes Soportados
- **Charts:** Gráficos de barras, líneas y radar para métricas de agentes.
- **Forms:** Peticiones de aprobación con botones de `Confirmar` / `Cancelar`.
- **Markdown Blocks:** Bloques de documentación formateada.

### 3. Seguridad (Sandboxing)
Los componentes se renderizan en un entorno controlado (React/Svelte en el dashboard) con validación estricta de esquemas JSON para evitar ataques XSS o inyección de scripts por parte del agente.

## Votación
- **Antigravity:** ACUERDO (Mejorará exponencialmente la experiencia de usuario de Alberto).
