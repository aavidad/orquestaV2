# Decisiones locales: orquesta-factory

Las decisiones de este archivo solo afectan a `orquesta-factory`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## Registro

```text
Fecha: 2026-05-04
Decision: AppSpec v0 sera el DTO canonico local para pedir una nueva app y generar un backlog inicial propuesto.
Motivo: orquesta-web necesita una forma publica, versionada y validable de solicitar una app sin conocer internos de orquesta-factory ni activar runtime o DB.
Alternativas: aceptar texto libre sin contrato; crear tareas directas desde UI; delegar la peticion a orquesta-core desde el inicio.
Impacto: se documentan SolicitarNuevaApp v0, AppSpecRequestV0, AppSpecV0 y BacklogInicialPropuestoV0 como contratos locales de orquesta-factory.
Contratos afectados: docs/contratos.md; ../contratos/contratos_iniciales.md como fuente inicial, sin modificar en esta tarea.
Estado: propuesta local registrada
```

```text
Fecha: 2026-05-04
Decision: El backlog inicial de factory sera una propuesta, no un ProyectoPlan definitivo.
Motivo: ProyectoPlan pertenece a orquesta-core segun contratos_iniciales.md; factory puede proponer microtareas y contratos requeridos, pero no debe decidir el plan global ni el runtime.
Alternativas: hacer que factory genere ProyectoPlan completo; esperar a core antes de documentar AppSpec; crear tareas persistidas desde web.
Impacto: BacklogInicialPropuestoV0 queda limitado a fases, microtareas, riesgos, preguntas y contratos requeridos.
Contratos afectados: docs/contratos.md
Estado: propuesta local registrada
```

```text
Fecha: 2026-05-04
Decision: DB, runtime, filesystem, LLM, cache, cola y deploy solo aparecen como conectores o restricciones en AppSpec v0.
Motivo: las reglas del modulo prohiben mezclar wizard con runtime, crear tareas directas en DB desde UI y generar proyectos sin contratos verificables.
Alternativas: permitir seleccion de proveedor concreto en la spec; incluir SQL/tablas en datos; incluir arranque de agentes como parte del wizard.
Impacto: los campos de AppSpec v0 describen necesidades funcionales y conectores esperados, no implementacion de adaptadores.
Contratos afectados: AppSpecRequestV0, AppSpecV0, BacklogInicialPropuestoV0
Estado: propuesta local registrada
```

```text
Fecha: 2026-05-04
Decision: Promover un resumen minimo de SolicitarNuevaApp v0 a modulos/CONTRATOS.md para consumo de orquesta-web, orquesta-mcp y orquesta-cli.
Motivo: orquesta-web necesita un contrato compartido para avanzar sin conocer internals de orquesta-factory.
Alternativas: mantener el contrato solo como local de factory; duplicar todo el detalle en el contrato global; esperar a implementar schemas.
Impacto: modulos/CONTRATOS.md registra propietario, consumidores, DTOs, errores publicos, invariantes y enlace al detalle local.
Contratos afectados: SolicitarNuevaApp v0, AppSpecRequestV0, AppSpecV0, BacklogInicialPropuestoV0, AppSpecV0Validada
Estado: aceptada por el director y registrada
```

```text
Fecha: 2026-05-04
Decision: SolicitarNuevaAppV0 se implementa como caso de uso puro y determinista cuando recibe reloj explicito.
Motivo: factory debe poder validarse sin web, DB, runtime, filesystem, LLM ni adaptadores. El timestamp se inyecta para que los tests sean estables.
Alternativas: generar AppSpecV0 desde handler REST; usar tiempo global sin control; persistir la request como parte del caso de uso.
Impacto: appspec_usecase_v0.go normaliza AppSpecRequestV0 a AppSpecV0 y los adaptadores futuros solo deben llamar al puerto.
Contratos afectados: SolicitarNuevaApp v0, AppSpecRequestV0, AppSpecV0
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: Los tags JSON Go de AppSpecRequestV0 deben seguir el schema canonico, aunque los structs internos usen nombres Go distintos.
Motivo: orquesta-web, orquesta-mcp y orquesta-cli validaran contra schema; el caso de uso no puede rechazar campos validos como documentacion.usuario, agentes.revision_humana o calidad.accesibilidad.
Alternativas: cambiar el schema a nombres ingleses; aceptar duplicados; dejar el ajuste al adaptador REST.
Impacto: appspec_request_v0.go queda alineado con el schema de request; las opciones booleanas usan punteros para distinguir default de negacion explicita del usuario.
Contratos afectados: AppSpecRequestV0
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: GenerarBacklogInicialPropuestoV0 solo acepta AppSpecV0 con validation.estado=valida.
Motivo: FTY-005 pide generar backlog desde una AppSpecV0 valida; una spec provisional puede contener preguntas que cambien alcance o contratos y debe resolverse antes de proponer microtareas cerradas.
Alternativas: aceptar specs provisionales y marcar el backlog como provisional; generar solo fases sin microtareas; delegar la validacion a core.
Impacto: backlog_v0.go devuelve app_spec_invalida para specs no validas y mantiene preguntas abiertas como bloqueos/riesgos cuando una spec valida las conserva.
Contratos afectados: BacklogInicialPropuestoV0, AppSpecV0
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: El adaptador REST v0 de SolicitarNuevaApp devuelve 405 para metodos distintos de POST.
Motivo: el contrato REST canonico fija POST /api/v0/apps/spec y 405 permite comunicar metodo no permitido con Allow: POST sin inventar reglas de dominio.
Alternativas: devolver 400 para todo error de transporte; aceptar metodos adicionales; delegar el metodo a un router externo.
Impacto: appspec_http_v0.go mantiene un handler inbound fino con error publico app_spec_invalida para metodo incorrecto, sin stack ni cuerpo privado.
Contratos afectados: SolicitarNuevaApp v0 transporte REST
Estado: aceptada
```

```text
Fecha: 2026-05-04
Decision: appspec_usecase_v0.go queda como shell publico y el detalle se separa por DTOs, ensamblado, normalizacion, defaults, identidad y helpers de colecciones.
Motivo: el fichero estaba en zona roja de tamano y el siguiente trabajo sobre factory seria fragil si el caso de uso siguiera concentrando responsabilidades.
Alternativas: mantener el fichero monolitico hasta anadir mas funcionalidad; mover logica a otro modulo; cambiar contratos para simplificar el usecase.
Impacto: se conserva el contrato SolicitarNuevaApp v0 y se reduce el contexto necesario para depurar cada parte del flujo.
Contratos afectados: SolicitarNuevaApp v0, AppSpecRequestV0, AppSpecV0
Estado: aceptada_local
```

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
Fecha: 2026-05-10
Decision: `request_kind` y `execution_mode` son parte canonica de AppSpec.
Motivo: Orquesta debe saber si se le pide documentar, analizar, programar,
desplegar o crear una app completa; no puede inferir minimos de cierre desde
texto libre ni permitir entregas reducidas salvo debug explicito.
Alternativas: dejarlo como texto en objetivo; definir tipos solo en web; crear
un contrato separado por cada flujo.
Impacto: AppSpecRequestV0 acepta los campos opcionales con defaults
`crear_app_completa` y `normal`; AppSpecV0 los publica siempre, los schemas
canonicos los documentan y `ResolveRequestPolicyV0` devuelve minimos de cierre.
El modo `debug` es la unica via para recortar alcance.
Contratos afectados: AppSpecRequestV0, AppSpecV0, SolicitarNuevaApp v0.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-23
Decision: El transporte HTTP de AppSpec vive en `orquesta-factory-http`.
Motivo: `orquesta-factory` debe quedar como negocio puro reutilizable sin arrastrar `net/http` a consumidores neutrales.
Alternativas: mantener handler REST dentro de factory; crear wrappers de compatibilidad que conservaran el acoplamiento; mover el negocio al adaptador.
Impacto: `NewAppSpecHTTPHandlerV0`, path, clock y envelopes REST pasan al paquete adaptador, que importa `orquesta-factory` para DTOs y casos de uso. Los consumidores REST migran al adaptador `orquesta-factory-http` y `orquesta-factory` queda sin transporte HTTP.
Contratos afectados: SolicitarNuevaApp v0 transporte REST.
Estado: aceptada localmente.
```

```text
Fecha: 2026-06-27
Decision: `datos.storage[].tipo` usa un catalogo canonico de capacidades en factory.
Motivo: `/nueva-app` y la guia documental ya necesitaban distinguir ausencia de persistencia, objetos/blob, cache clave-valor, eventos de auditoria y storage mixto sin convertir el contrato en proveedores concretos como PostgreSQL.
Alternativas: dejar listas duplicadas en web/schema/validador; aceptar strings libres; introducir proveedores de DB como enums.
Impacto: `SupportedDataStorageTypesV0` es la fuente Go del validador; los schemas JSON se alinean; `sin_preferencia` y `sin_persistencia` no crean conectores `storage-*`; los proveedores concretos siguen rechazados o entran como restricciones/adaptadores externos.
Contratos afectados: AppSpecRequestV0, AppSpecV0, SolicitarNuevaApp v0.
Estado: aceptada localmente.
```
