# Decisiones locales: orquesta-core

Las decisiones de este archivo solo afectan a `orquesta-core`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## Plantilla

```text
Fecha:
Decision:
Motivo:
Alternativas:
Impacto:
Contratos afectados:
Estado:
```

```text
Fecha: 2026-05-27
Decision: Permitir que T198 referencie contratos core como fuentes canonicas de descriptors MCP sin acoplar core a MCP.
Motivo: FunctionContract y contratos core ya tienen owner y validacion local; el adaptador MCP debe evitar strings stale y conservar frontera hexagonal.
Alternativas: Importar MCP desde core; duplicar el shape en el transporte; dejar el resource como stale permanente.
Impacto: `orquesta-mcp` puede declarar `descriptor_source` hacia core, pero core no conoce transporte, HTTP, CLI ni runtime.
Contratos afectados: FunctionContractV0; MCPResourceDescriptorSourceV0.
Estado: aceptada_local
Revalidacion 2026-05-27: `agent-ref-task-autoprogramming-c3678e9bc306-g01`
mantiene esta decision y no acopla core al adaptador MCP.
```

```text
Fecha: 2026-05-04
Decision: El arranque de core expone un puerto de entrada provisional `RegistrarProyectoDesdeAppSpec v0`.
Motivo: El mapa global ya asigna a orquesta-core la responsabilidad de registrar proyecto y backlog, pero ProyectoPlan aun esta pendiente. Hace falta un borde pequeno para recibir AppSpecV0 validada y BacklogInicialPropuestoV0 desde factory sin bloquear a otros grupos.
Alternativas: Esperar a ProyectoPlan completo; hacer que factory cree tareas de core; persistir directamente desde factory.
Impacto: Core puede definir el caso de uso puro y su salida de borrador sin introducir DB, runtime ni adaptadores concretos.
Contratos afectados: RegistrarProyectoDesdeAppSpec v0, ProyectoPlanBorradorV0.
Estado: propuesta local
```

```text
Fecha: 2026-05-04
Decision: Core no revalida la AppSpec completa; solo exige evidencia/flag de validacion y compatibilidad minima con el backlog.
Motivo: AppSpecV0 pertenece a orquesta-factory. Duplicar su validacion en core generaria acoplamiento y divergencia de reglas.
Alternativas: Revalidar todo AppSpecV0 en core; aceptar cualquier payload de factory sin controles.
Impacto: Los errores de core se limitan a entrada ausente, version no soportada, falta de validacion, backlog vacio e incompatibilidad minima.
Contratos afectados: RegistrarProyectoDesdeAppSpecCommandV0.
Estado: propuesta local
```

```text
Fecha: 2026-05-04
Decision: El resultado `ProyectoPlanBorradorV0` no confirma persistencia ni lanzamiento de agentes.
Motivo: Persistencia y runtime son adaptadores/modulos separados segun AGENTS.md, README.md y contratos globales.
Alternativas: Guardar proyecto en DB desde core; emitir comandos runtime durante el registro; devolver un ProyectoPlan definitivo.
Impacto: La primera version queda testeable como dominio puro y lista para conectar despues a puertos de persistencia, observability y runtime.
Contratos afectados: RegistroProyectoAceptadoV0, ProyectoPlanBorradorV0, EventoDominioCoreV0.
Estado: propuesta local
```

```text
Fecha: 2026-05-04
Decision: Los eventos generados por el registro siguen siendo eventos de dominio locales y se mapean a `OrquestaEvent v0` solo por puerto de salida.
Motivo: `OrquestaEvent v0` ya esta promovido globalmente, pero core no debe publicar directamente ni conocer el sink. Core prepara un request compacto para un adaptador futuro.
Alternativas: Importar un paquete de observability inexistente; publicar directamente desde core; omitir eventos.
Impacto: Mantiene trazabilidad hacia observability sin acoplar core a sink, DB, bus, filesystem ni detalles de transporte.
Contratos afectados: EventoDominioCoreV0.
Estado: implementada en mapper local
```

```text
Fecha: 2026-05-04
Decision: Promocionar `RegistrarProyectoDesdeAppSpec v0` a contrato global minimo en `modulos/CONTRATOS.md`.
Motivo: factory necesita un borde estable para entregar `AppSpecV0` validada y `BacklogInicialPropuestoV0` sin conocer internals de core ni activar DB/runtime.
Alternativas: dejarlo solo como contrato local; esperar al ProyectoPlan definitivo; permitir que factory registre proyecto por su cuenta.
Impacto: El contrato global publica nombre, propietario, consumidores, DTOs, errores e invariantes. El detalle permanece en `orquesta-core/docs/contratos.md`.
Contratos afectados: RegistrarProyectoDesdeAppSpec v0, ProyectoPlanBorradorV0.
Estado: aceptada por el director y registrada
```

```text
Fecha: 2026-05-04
Decision: Formalizar `FunctionContract v0` como contrato local canonico de microprogramacion.
Motivo: DB v1 contiene `especificaciones_funcion` con objetivo/titulo, archivo y simbolo foco, pre/postcondiciones, dependencias, tests, write-set y formato de salida. Esa forma encaja con la regla V2 de microtareas pequenas, pero la base historica no debe convertirse en runtime ni canon de ids.
Alternativas: Importar filas de DB v1 como tareas vivas; dejar FunctionContract pendiente hasta tener implementacion; definir solo una plantilla informal.
Impacto: Core puede validar contratos de funcion, write-set y evidencia sin acoplarse a SQLite, runtime ni proveedores. Los demas modulos consumen el resumen global desde `modulos/CONTRATOS.md`.
Contratos afectados: FunctionContract v0, FunctionContractErrorV0.
Estado: aceptada_director; resumen minimo promovido a `modulos/CONTRATOS.md`.
```

```text
Fecha: 2026-05-04
Decision: Implementar `RegistrarProyectoDesdeAppSpecV0` como funcion pura y determinista, sin puerto de persistencia ni adaptadores.
Motivo: El siguiente corte core debe arrancar implementacion con DTOs publicos de factory, pero persistence, runtime, capacidad, agentes y observability aun no forman parte del alcance.
Alternativas: Definir una interfaz de repositorio desde el primer corte; devolver solo errores sin plan; duplicar validacion completa de factory en core.
Impacto: Core valida schema/estado/spec_id, mapea fases y microtareas a un `ProyectoPlanBorradorV0` serializable y genera solo un evento de dominio local devuelto como dato.
Contratos afectados: RegistrarProyectoDesdeAppSpec v0, ProyectoPlanBorradorV0, EventoDominioCoreV0.
Estado: implementada
```

```text
Fecha: 2026-05-04
Decision: Alinear los DTOs locales de salida con los schemas globales promovidos para observability y persistence.
Motivo: La revision de direccion detecto que `OrquestaEventV0` local no tenia el shape requerido por `orquesta_event_v0.schema.json` y que persistence usaba versiones no canonicas.
Alternativas: Mantener DTOs locales divergentes y exigir que el adaptador los traduzca completamente; importar paquetes de observability/persistence.
Impacto: Core produce un OrquestaEventV0 compacto con event_id, severity, outcome, correlation, subject, producer y privacy. Persistence usa `PersistenceRepositoryV0` y `ProyectoPlanBorradorV0`; el adaptador de salida sigue siendo responsable de construir el envelope global con `unidad_trabajo`.
Contratos afectados: OrquestaEventPublisherPortV0, PersistenceRepositoryPortV0.
Estado: implementada
```

```text
Fecha: 2026-05-04
Decision: Definir puertos de salida locales para `PersistenceRepository v0`, `RuntimeLaunchRequest v0`, `OrquestaEvent v0` y `GovernanceCatalog v0`.
Motivo: Los cuatro contratos ya estan promovidos en `modulos/CONTRATOS.md` y core necesita un borde de consumo sin implementar DB, runtime, event sink ni governance real.
Alternativas: Mantenerlos solo como pendientes documentales; importar paquetes de otros modulos antes de que existan; implementar adaptadores reales en core.
Impacto: Core expone interfaces y DTOs publicos locales, mappers puros para persistence/observability y proyecciones pequenas para runtime/governance. Los adaptadores futuros traducen al contrato global de cada modulo.
Contratos afectados: PersistenceRepositoryPortV0, RuntimeLauncherPortV0, OrquestaEventPublisherPortV0, GovernanceCatalogPortV0, FunctionContractV0.
Estado: implementada
```

```text
Fecha: 2026-05-04
Decision: Responder a CLI con operaciones candidatas de solo lectura `listar FunctionContractV0` y `ver FunctionContractV0`; `registrar FunctionContractV0` queda bloqueada.
Motivo: CLI puede necesitar inspeccionar contratos de funcion, pero registrar contratos implica mutacion, idempotencia, auditoria y eventos. Esas garantias dependen de cerrar OrchestrationRun, CommandHandler y Outbox, que no pertenecen a este corte.
Alternativas: Implementar comandos CLI ya; permitir registrar como documento local; esperar a scheduler/workflow antes de documentar cualquier borde.
Impacto: Core documenta un borde candidato consultable sin implementar scheduler, workflow, runtime ni adaptadores. La promocion global de listar/ver queda para decision del director si CLI los implementa como contrato inter-modulo.
Contratos afectados: FunctionContract v0, ListarFunctionContractsV0, VerFunctionContractV0, RegistrarFunctionContractV0.
Estado: propuesta local no promovida
```

```text
Fecha: 2026-05-04
Decision: Dividir `puertos_salida_v0.go` por familias mecanicas de puertos de salida.
Motivo: El archivo preservado concentraba persistence, observability, runtime, governance y FunctionContract en una zona roja de 411 lineas, dificultando cortes pequenos sin cambiar contratos.
Alternativas: Mantener el archivo monolitico; crear subpaquetes; renombrar DTOs por adaptador.
Impacto: Los nombres publicos, interfaces, errores y mappers se conservan en el mismo paquete; solo cambia la ubicacion fisica por fichero.
Contratos afectados: PersistenceRepositoryPortV0, OrquestaEventPublisherPortV0, RuntimeLauncherPortV0, GovernanceCatalogPortV0, FunctionContractV0.
Estado: implementada
```

```text
Fecha: 2026-05-04
Decision: Dividir `registrar_proyecto_appspec_v0.go` por responsabilidades mecanicas de contrato publico, validacion, ensamblado y helpers.
Motivo: Era el ultimo fichero Go de orquesta-core por encima de 300 lineas y mezclaba DTOs, validacion, construccion del borrador/evento y utilidades deterministas del caso de uso.
Alternativas: Mantener el archivo monolitico; crear subpaquetes; renombrar DTOs o helpers.
Impacto: Los nombres publicos, errores, invariantes y comportamiento de `RegistrarProyectoDesdeAppSpecV0` se conservan en el mismo paquete; solo cambia la ubicacion fisica de funciones privadas.
Contratos afectados: RegistrarProyectoDesdeAppSpec v0, ProyectoPlanBorradorV0, EventoDominioCoreV0.
Estado: implementada
```

```text
Fecha: 2026-05-04
Decision: Cerrar CORE-001, CORE-002 y CORE-003 como completadas sin tocar codigo Go.
Motivo: La implementacion pura, el mapeo y las pruebas de contrato ya existen en RegistrarProyectoDesdeAppSpecV0; faltaba dejar fixtures documentales compactos y actualizar el estado de las tareas iniciales.
Alternativas: Reabrir implementacion Go; esperar a ProyectoPlan global definitivo; crear un schema ejecutable nuevo en este corte.
Impacto: Los fixtures de `docs/fixtures/registrar_proyecto_appspec_v0/` fijan comandos ok/error, y `go test -count=1 ./modulos/orquesta-core` cubre OK, errores principales e idempotencia sin DB, runtime, proveedor ni rutas HOME.
Contratos afectados: RegistrarProyectoDesdeAppSpec v0, RegistrarProyectoDesdeAppSpecCommandV0, ProyectoPlanBorradorV0.
Estado: implementada
```
