# Paquetes App-Layer: utilidad y estado

Fecha: 2026-03-25  
Responsable: Codex2  
Tarea Orquesta: `#379`

## Resumen

Se han revisado los paquetes no pusheados `agentesapp`, `lenguajeapp` y `runtimesapp` para determinar si eran ruido o código útil.

Conclusión:

- `agentesapp`: útil. Contiene agregación de estado operativo de agentes para panel/detalle.
- `lenguajeapp`: útil. Encapsula política, matriz y resolución de lenguaje como capa de aplicación fina.
- `runtimesapp`: útil y además necesario. `HEAD` ya referencia este paquete desde `cmd/mcp.go`.

## Qué se ha dejado hecho

- comentarios de paquete (`doc.go`) en los tres paquetes
- tests propios de paquete:
  - `agentesapp`: agregación de filas y detalle con deduplicación de mailbox
  - `lenguajeapp`: normalización de entradas y delegación a store
  - `runtimesapp`: delegación de consultas de observabilidad/runtime
- verificación adicional de integración en `cmd`

## Qué no se ha hecho en esta tarea

- no se ha tocado la integración local sucia de `cmd/agentes_web.go` y `cmd/lenguaje_web.go`
- no se han arrastrado cambios ajenos mezclados en `cmd/`
- no se ha reordenado arquitectura fuera de estos tres paquetes

## Verificación

- `go test ./agentesapp ./lenguajeapp ./runtimesapp`
- `go test ./cmd -run 'Test(WebAgentesPanelMuestraEstadoVivo|WebLenguajePaginaMuestraPoliticaMatrizYResolucion|MCP)'`

## Observación importante

`runtimesapp` no debe quedarse sin commit: `cmd/mcp.go` en `HEAD` ya lo importa.
