# OP-094: UI Declarativa (A2UI)

## Estado actual

Fecha: 2026-03-23

Preparación técnica ya realizada, pero todavía no activada en la operativa normal.

Hecho:

- contrato A2UI aislado en `internal/a2ui`
- validación estricta de envelopes y `props`
- componentes soportados en esta fase:
  - `DataTable`
  - `Chart`
  - `ApprovalForm`
  - `MarkdownBlock`
- adaptador dormido para encapsular mensajes A2UI en `runtime_mailbox`
- tests unitarios del contrato y del envelope de mailbox

Rutas de código:

- `internal/a2ui/a2ui.go`
- `internal/a2ui/a2ui_test.go`

Ya activado parcialmente:

- `serve` muestra mensajes A2UI en modo lectura dentro del detalle de runtime
- no se ha abierto todavía una vía de edición o respuesta humana específica sobre esos componentes
- sigue siendo un canal complementario; no sustituye el flujo normal de tareas, propuestas o mailbox

Envelope de mailbox preparado:

- `kind`: `a2ui_render`
- `version`: `a2ui.mailbox.v1`
- helper canónico de emisión: `BuildMailboxMessage(fromAgente, toAgente, req)`

Ejemplo canónico de payload en `runtime_mailbox`:

```json
{
  "from_agente": "Codex3",
  "to_agente": "alberto",
  "kind": "a2ui_render",
  "payload": "{\"version\":\"a2ui.mailbox.v1\",\"request\":{\"type\":\"a2ui_render\",\"component\":\"MarkdownBlock\",\"props\":{\"title\":\"Nota\",\"markdown\":\"hola\"}}}"
}
```

Objetivo de esta fase:

- dejar cerrado el contrato y la validación para poder integrar después el lector y el renderizado sin abrir ahora una superficie inestable en `serve` o en la API caliente

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
