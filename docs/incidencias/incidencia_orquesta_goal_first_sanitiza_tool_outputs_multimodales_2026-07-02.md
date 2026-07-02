# Incidencia: goal-first no debe propagar payloads visuales en thread/read

Fecha: 2026-07-02

## Sintoma

En una ejecucion OPES goal-first, una llamada visual genero un
`data:image/png;base64,...` dentro del rollout. El historial operativo crecio
de forma desproporcionada y consumio contexto antes de producir la matriz o el
plan solicitado.

## Riesgo

Si `thread/read` devuelve turnos con payloads multimodales embebidos, Orquesta
puede volver a introducir binarios en observacion, diagnostico, busqueda de
marcadores y contexto de decisiones posteriores. Esto degrada coste y
observabilidad aunque el resultado terminal se sanee despues.

## Cierre aplicado

- La frontera de decodificacion RPC y WebSocket del app-server sanea respuestas
  `thread/read` antes de entregarlas al backend Goal.
- Los `data:*;base64,` se sustituyen por una proyeccion compacta con MIME,
  hash, bytes codificados, bytes decodificados y dimensiones cuando el formato
  permite leerlas.
- Los textos de item y `itemsView` que superan el presupuesto operativo se
  sustituyen por una referencia hash compacta; si hay marcador
  `ORQUESTA_GOAL_RESULT_V0`, se conserva la ventana del marcador.
- El trabajo no se rechaza ni se descarta: la salida queda representada como
  evidencia compacta recuperable.

## Evidencia

Tests:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestCodexAppServerThreadReadSanitizaDataURIMultimodalV0|TestDecodeCodexAppServerRPCResponseV0SanitizaThreadReadMultimodalV0|TestServerCodexAppServerGoalBackendV0SanitizaResultadoDurableConBase64V0'
go test -count=1 ./cmd/orquesta-server
```
