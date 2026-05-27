# Arquitectura inicial: self-observability

Estado: historico/debug de una app generada. Este documento no marca cierre
productivo ni gobierna requisitos vivos del repo Orquesta; fija una primera
arquitectura del corte self-observability y deja trabajo posterior para agentes
de documentacion, programacion, pruebas, seguridad y revision.

Vigencia 2026-05-26: las piezas `manual_usuario`, `manual_desarrollador`,
`manual_sistemas_deploy`, `decisiones`, `pruebas_documentales` y `pendientes`
son tipos de artefacto esperados del proyecto objetivo. Cuando Orquesta genere
o modifique una app, esos nombres deben viajar como refs/rutas relativas de esa
app o como plantillas de `docs/plantillas_documentacion/`, no como documentos
raiz obligatorios del nucleo Orquesta.

## Alcance

La app objetivo expone observabilidad propia de Orquesta para solicitudes
llegadas por web, API y MCP. El alcance conocido es pequeno: dirigir una app
con arquitectura hexagonal, textos localizables y persistencia solo por puerto o
conector si una tarea futura la pide.

CONSULTA AL DIRECTOR: falta el alcance funcional concreto de la app final:
metricas exactas, pantallas, endpoints, eventos visibles, usuarios objetivo y
criterios de cierre productivo. Hasta recibirlo, este documento solo define una
base inicial.

## Entregables Minimos Historicos

Este corte documental cubre como plan inicial:

- `manual_usuario`
- `manual_desarrollador`
- `manual_sistemas_deploy`
- `decisiones`
- `pruebas_documentales`
- `pendientes`

En modo debug no se declara cierre productivo: solo queda preparada la primera
entrega accionable del director y el trabajo posterior para ejecucion real. Los
entregables anteriores pertenecen al proyecto generado de ese corte.

## Arquitectura

La solucion se organiza por hexagono:

- Dominio: conceptos estables de observabilidad, ejecucion, fases, tareas,
  agentes, entregas, revisiones, validaciones y pendientes.
- Aplicacion: casos de uso para consultar estado, abrir detalle, solicitar
  cambio, registrar evidencia compacta y preparar respuestas para cada canal.
- Puertos de entrada: web, API y MCP con DTOs pequenos, validacion de locale y
  errores traducibles.
- Puertos de salida: lectura de estado, escritura de eventos, entrega de
  artefactos y consulta de progreso. La persistencia concreta queda fuera del
  nucleo y se inyecta por conector.
- Presentacion: textos visibles por catalogo i18n. Enums, refs y estados
  internos no se traducen.

Reglas no negociables:

- Sin acceso directo desde web, API o MCP a detalles operativos internos.
- Sin persistencia concreta por defecto.
- Sin rutas locales, datos sensibles ni transcripciones completas en resultados
  publicos.
- Ficheros pequenos y responsabilidades separadas.
- Si se genera una app Go completa en una fase futura, debe tener modulo
  autonomo, entrypoint documentado e imports desde el modulo.

## Manual Usuario

Usuario previsto: operador o revisor que necesita saber que esta haciendo
Orquesta sin leer artefactos internos.

Flujos esperados:

- Ver resumen de una solicitud: estado, fase actual, tareas vivas, entregas y
  bloqueos.
- Abrir detalle de una tarea: objetivo, write-set declarado, pruebas requeridas,
  estado de revision y evidencia compacta.
- Ver avisos de calidad: falta de ACK, entrega sin prueba, exceso de tamano,
  solicitud de rework o consulta pendiente.
- Filtrar por app, run, fase, estado, severidad y fecha logica.
- Cambiar idioma visible entre al menos `es-ES` y `en-US` sin alterar refs ni
  estados internos.

Textos visibles: deben salir de catalogos i18n. Si falta una traduccion, se
debe mostrar una clave controlada y registrar pendiente documental.

## Manual Desarrollador

Paquetes sugeridos si se programa la app:

- `internal/domain/observability`: entidades y validaciones puras.
- `internal/application`: casos de uso y puertos.
- `internal/inbound/http`: entrada API.
- `internal/inbound/mcp`: entrada MCP.
- `internal/inbound/web`: view models o controladores web.
- `internal/outbound`: conectores concretos, siempre detras de puertos.
- `internal/i18n`: catalogos y resolucion de mensajes visibles.

Contratos principales:

- `RunSummary`: estado compacto de una solicitud.
- `TaskSummary`: objetivo, fase, write-set, pruebas y revision.
- `DeliverySummary`: entrega, evidencia compacta y resultado de review.
- `ObservationQuery`: filtros por app, run, fase, estado y locale.
- `LocalizedMessage`: clave, locale, parametros seguros y texto final.

Politica de errores:

- Errores de dominio con codigo estable.
- Traduccion en el borde visible.
- Evidencia compacta sin rutas locales ni datos sensibles.
- Ningun transporte debe inferir estado leyendo artefactos por su cuenta.

## Manual Sistemas Deploy

En modo debug solo se documenta la operacion esperada:

- Arranque opt-in por configuracion explicita del operador.
- Health check de lectura para confirmar que el canal responde.
- Configuracion de locales disponibles.
- Conectores de persistencia y progreso inyectados por puertos.
- Logs operativos con refs compactas y sin datos sensibles.
- Backup y retencion definidos por el conector elegido, no por el nucleo.

Pendiente para cierre productivo:

- Matriz de variables de configuracion.
- Runbook de arranque, parada, actualizacion y recuperacion.
- Politica de retencion y borrado.
- Prueba de despliegue en entorno representativo.

## Decisiones

- D-001: usar arquitectura hexagonal para separar dominio, aplicacion,
  transporte y conectores.
- D-002: aplicar i18n en todos los textos visibles desde el primer corte.
- D-003: no fijar persistencia concreta; se usaran puertos y conectores.
- D-004: documentar primero en modo debug y no cerrar como productivo.
- D-005: separar trabajo posterior en microtareas con write-set pequeno.
- D-006: emitir decisiones ejecutables solo hasta programacion; revision,
  validacion final y cierre quedan bloqueados hasta tener evidencias reales.

## Pruebas Documentales Historicas

Pruebas esperadas para esta fase del proyecto generado:

- `test -s docs/arquitectura.md`
- `test -s docs/plan_microtareas.md`
- Revision manual de que existen secciones: manual_usuario,
  manual_desarrollador, manual_sistemas_deploy, decisiones,
  pruebas_documentales y pendientes.
- Validacion de que las decisiones ejecutables no contienen evidencia extensa ni
  referencias humanas largas.

Estas pruebas no exigen crear `docs/manual_desarrollador.md`,
`docs/manual_sistemas_deploy.md`, `docs/pruebas_documentales.md` ni
`docs/pendientes.md` en el repo Orquesta vigente.

## Pendientes

- Confirmar alcance funcional exacto con el director.
- Definir catalogos i18n iniciales.
- Definir DTOs publicos para web, API y MCP.
- Crear manuales finales separados si el director decide pasar de debug a
  cierre documental completo.
- Crear plan de pruebas real de UI, API y MCP.
- Ejecutar las microtareas publicadas para producir entregables finales.
- Ejecutar revision final solo con evidencias reales de entrega.
