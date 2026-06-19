# Decisiones: orquesta-app-planner

```text
Fecha: 2026-05-14
Decision: La fase de cada unidad se deriva de su rol, no de una constante de
programacion.
Motivo: Orquesta debe saber si esta programando, documentando, integrando o
revisando sin usar una variable global. La app es multitarea: el tipo de
peticion vive en `request_kind`, pero la ejecucion real se gobierna por
`phase_id` de cada unidad.
Impacto: `documentacion` genera `phase_id=documentacion`; `integracion` y
`entorno`, `phase_id=integracion`; `revision`, `phase_id=revision`; el resto
queda en `programacion`. El planner valida fases contra el catalogo del core.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Separar planificacion de app real del nucleo y de fabricaapp legacy.
Motivo: la prueba con Codex real mostro que una app completa como unica tarea
tarda demasiado y puede no entregar ACK. La solucion es planificar microtareas
pequenas, no subir timeout ni relajar validadores.
Alternativas:
  - Reutilizar directamente fabricaapp/service.go: descartado por tamano y por
    mezclar demasiadas decisiones historicas.
  - Meter el planner en orquestacionnucleoapp: descartado porque el nucleo no
    debe saber que es una app Go concreta.
Impacto: este modulo genera plan y candidatos por puerto; el core sigue puro.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: El plan de app Go usa contrato de producto compilable, no solo
artefactos por write-set.
Motivo: una entrega con ACKs, docs y paquetes internos puede parecer progreso
real aunque no sea ejecutable como aplicacion. El plan debe cerrar desde el
principio los invariantes minimos de Go: modulo, entrypoint, imports de modulo
y pruebas.
Impacto: bootstrap exige `go.mod`; la unidad API escribe
`internal/adapters/http`, `internal/app/bootstrap` y `cmd/server`; apps grandes
usan el mismo corte de adaptador HTTP + composicion; todas las unidades
conservan `go test ./...` como required test por contrato runtime. Persistencia
y deploy siguen siendo puertos/conectores, sin proveedor DB hardcodeado.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Introducir `scale=large` para apps grandes.
Motivo: una app grande no puede entrar como seis tareas genericas. Necesita
arquitectura previa, contratos, puertos de persistencia, i18n, deploy,
integracion, documentacion y revision como cortes separados para mantener
contextos pequenos y agentes paralelizables.
Impacto: BuildGoAPIWebMicrotaskPlanV0 mantiene `standard` para pruebas pequenas
y usa `large` cuando se solicita explicitamente o cuando AppSpec indica
persistencia, pruebas altas o deploy no local. DB y deploy siguen como
conectores, sin proveedor hardcodeado.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: El planner consume AppSpecV0 validada, no un formulario paralelo.
Motivo: REST, MCP y web ya convergen en factory. Crear otra entrada para lanzar
apps repetiria el fallo de v1/v2: decisiones duplicadas y caminos no probados.
Impacto: AppPlanRequestFromAppSpecV0 valida el contrato de factory y solo
traduce superficies a un plan de microtareas.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Generar contexto local de app desde la microtarea bootstrap.
Motivo: cada Codex que trabaje en un directorio debe encontrar reglas locales
pequenas al arrancar. Si el contexto solo vive en la sesion principal, se repite
el fallo de contextos grandes y dependientes de una conversacion.
Impacto: bootstrap crea AGENTS.md y docs locales minimos de la app generada; las
fases siguientes leen contratos compactos y write-set cerrado.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Resolver contrato, evidencia y contexto por puertos separados.
Motivo: el planificador puede saber que unidad se lanza, pero no debe elegir
proveedor real, modelo, HOME ni credenciales. Esos datos quedan en los puertos
de capacidad, reserva y runtime.
Impacto: AppPlanFunctionContractResolverV0, AppPlanLaunchEvidenceResolverV0 y
AppPlanContextBundleResolverV0 permiten construir RuntimeLaunchRequestV0 sin
acoplar el planner a adaptadores concretos.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Filtrar olas por deliveries registradas.
Motivo: WorksetClaim.depends_on expresa dependencias cerradas dentro de un plan
de claims, pero el scheduler no debe recibir tareas futuras si sus entregas
previas aun no existen.
Impacto: el provider solo pasa la ola lista. La ola completa viaja una vez en
`WorkClaims`; cada candidato conserva solo su claim local para evitar payloads
O(n2) y contextos grandes.
Estado: aceptada.
```
