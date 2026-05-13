# Tareas: orquesta-app-change

## APP-CHANGE-001

Objetivo: crear contrato y caso de uso para registrar cambios sobre una app
existente.

Estado: hecho.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-change`

## APP-CHANGE-002

Objetivo: conectar el contrato a MCP/REST/web mediante adaptadores finos.

Estado: hecho.

Validacion:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-web ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway`

## APP-CHANGE-003

Objetivo: convertir el cambio aceptado en replanificacion completa de trabajo
por parte del director.

Estado: pendiente.
Estado implementado parcial: hecho para solicitudes concretas con
`allowed_write_set` y `acceptance_criteria`.

Entrada:

- el cambio ya queda registrado como pregunta/evento del workflow en el stack;
- `orquesta-app-change-director-source` convierte el cambio en respuesta al
  director, votacion, decision aceptada, contrato funcional, microtarea y vuelta
  a programacion;
- si faltan write-set o criterios, no inventa trabajo y deja la consulta para el
  director.

## APP-CHANGE-004

Objetivo: aceptar un cambio a mitad de ejecucion como evento externo y
convertirlo en `AppChangeRequestV0` sin duplicar el servicio existente.

Estado: hecho.

Validacion:

- `TestReceiveAppChangeIntentEventV0RegistraSolicitudDeCambio`;
- `TestReceiveAppChangeIntentEventV0DerivaChangeRefDesdeEvento`;
- `TestReceiveAppChangeIntentEventV0InvalidoNoTocaPuertos`.

## APP-CHANGE-005

Objetivo: permitir que una app externa declare trabajo de dominio acotado sin
integrarse dentro de Orquesta.

Estado: hecho.

Contrato:

- `AppChangeRequestV0.external_work` transporta solo refs opacas de proyecto,
  interfaz y trabajo externo;
- no se aceptan refs no compactas;
- el evento externo conserva `external_work` al normalizar a solicitud.

Validacion:

- `TestRequestAppChangeV0RechazaExternalWorkNoCompacto`;
- `TestReceiveAppChangeIntentEventV0RegistraSolicitudDeCambio`.
