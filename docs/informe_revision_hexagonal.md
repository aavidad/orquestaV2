/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

# Informe de revisión arquitectónica y de orquestación

## Contexto

Tarea #[394](orquesta tarea ver 394) / #[395](orquesta tarea ver 395) / #[396](orquesta tarea ver 396): revisar toda la aplicación en busca de brechas con el objetivo declarado (arquitectura hexagonal, servidor-primero, i18n, catálogo revisado) y definir el plan de remediación del servidor MCP.

## Avance inicial

- Se ha comprobado que la política hexagonal está recogida en `ARQUITECTURA.md` y amplificada por `OP-056`, con un núcleo sin dependencias directas de la persistencia y un servidor central como adaptador inbound que evita el acceso directo a SQLite.
- Se han identificado los comandos que ya obligan a delegar en `cmd/cliente_servidor.go`, lo cual permite cuantificar la cobertura del servidor en el segundo plano del CLI/API.
- Se ha comenzado el rastreo del vínculo entre servidor (cmd/server.go) y adaptadores (rpclocal) para documentar cómo se lanza `orquesta serve` y se ejecutan comandos remotos.
- Ya se han localizado documentos clave (`docs/uso_actual_app_orquesta.md`, `op_088_orquesta_servidor_mcp.md`) que subrayan la política servidor-primero y la necesidad de eliminar el acceso directo a la persistencia desde scripts o clientes locales.

## Hallazgos preliminares

1. **Inmadurez en la capa de servicios/domino**: `cmd/api.go` define handlers que llaman directamente a funciones de `db.*` para consultar y mutar `Proyectos`, `Tareas`, `Propuestas` y `Sesiones`. Ese acoplamiento sitúa la base de datos dentro del adaptador inbound, en vez de través de puertos/servicios internos (`internal/…`). Para cumplir la regla de "Interfaces primero" y el principio de "la base de datos es un adaptador", hay que extraer esa lógica a servicios (por ejemplo `internal/controlplane` o `internal/api`) con interfaces, y dejar que los handlers solo invocen esos ports.
2. **Persistencia multi-tenant pendiente**: No se ha implementado aún el patrón `tenantDB(ctx)` recomendado por las reglas (#81000, #81037 y `db/historial.go`). La base de datos actual no separa instancias por ayuntamiento ni resuelve conexiones desde el contexto (no se detecta ningún `tenantDB` activo fuera de la documentación de migraciones). Esto compromete la política de aislamiento y debe priorizarse en el plan de corrección.
3. **Fuga hacia el modo local/script**: Aunque `orquesta serve` y `cmd/cliente_servidor.go` introducen un plan de servidor, el equipo sigue utilizando `scripts/inicio_agente.sh` (ver `docs/uso_actual_app_orquesta.md`) y muchos comandos (`cmd/tarea.go`, `cmd/sesion.go`, `cmd/worktree.go`, `cmd/agente.go`, etc.) llaman a `ensureLocalDB()` cuando el API no responde. Ese wrapper lanza los comandos CLI locales, lo cual no garantiza que se pase por MCP ni por el control plane y lo mantiene fuera del servidor. Hay que diseñar un reemplazo que use el API/daemon o eliminarlo cuando el servidor cubra el ciclo completo.
4. **Cobertura de idiomas incompleta**: El esqueleto actual en `i18n/` solo entrega `es.json` y `en.json`. La política exige un pack inicial con `es`, `en`, `de`, `fr`, `it`, `zh`, `gl`, `eu`, `ca` y `val`. Falta crear los ficheros de esos idiomas, el fallback, y un mecanismo de selección por usuario/despliegue antes de seguir añadiendo textos o UI.
5. **Documentación del cliente Server-first pendiente**: `docs/uso_actual_app_orquesta.md` todavía describe el uso mixto CLI/Web/API y el manual promueve la combinación local. La política `docs/politica_arquitectura_tipos_proyecto_es.md` ya exige delegar por defecto en `orquesta serve` y solo permitir `--local`/`ORQUESTA_FORCE_LOCAL` en casos de recuperación, por lo que hay que actualizar la documentación (y posiblemente guías de flux) para reflejar que la única vía productiva es el daemon + API + control plane, y no generar nuevos flujos locales.
6. **Enlaces directos a repositorios desde los adaptadores de coordinación**: `cmd/worktree.go` y `cmd/coordination_support.go` invocan `db.Coordination*Repository()` y `db.GetProyecto` directamente. Esa lógica debería vivir detrás de puertos del `coordinacion.Service`, expuestos únicamente vía servidor/daemon, para no romper la capa de servicio ni dar acceso directo a la persistencia desde comandos locales como `orquesta worktree`.
7. **Política AP-077 no satisfecha por flujo actual**: La [Política de Acceso a Persistencia (AP-077)](docs/politica_acceso_persistencia_es.md) exige que toda mutación/lectura ocurra vía Orquesta y que el acceso directo se convierta en excepción residual. Hoy todavía se usan variantes locales (scripts, worktrees, sesiones manuales) que leen/escriben sin pasar por el servidor MCP. Hay que cerrar esos huecos antes de imponer el bloqueo técnico total que la política prevé.
8. **Gating server-first incompleto en el arranque**: `cmd/root.go` fuerza `ORQUESTA_REQUIRE_SERVER=1` y delega por defecto al servidor, pero `initDB()` sigue abriendo la BD local cuando `shouldPreferAPIClient(...)` no aplica. El modo local debe quedar reservado a recuperación explícita con `--local` o `ORQUESTA_FORCE_LOCAL=1`, nunca como flujo normal de negocio. Los tests en `cmd/root_gating_test.go` deben seguir protegiendo ese contrato.
9. **Contrato i18n e implementación incompatibles**: `i18n/project_skeleton.go` y `docs/plantillas_i18n/README_es.md` definen un esqueleto por idioma y dominio (`i18n/<lang>/<domain>.json`), pero `i18n/bundle.go` solo carga ficheros planos `i18n/*.json` y no recorre subdirectorios por idioma ni dominios. Ahora mismo el contrato aprobado y el cargador en producción no hablan el mismo formato. Antes de ampliar idiomas o UI, hay que unificar ambos lados.
10. **Control activo con diseño correcto pero integración incompleta**: `coordinacion.Service` ya introduce un servicio orientado a puertos para locks y worktrees, y el diseño de `docs/diseno_control_activo_agentes.md` define claramente el camino de órdenes y handles. Sin embargo, `cmd/agente_control_lifecycle.go` sigue validando agentes, resolviendo proyectos/handles y encolando `runtime_orders` mediante `db.*` desde el propio adaptador CLI/API. Hay una base hexagonal, pero aún no se usa de forma consistente.
11. **La web está a medio camino hacia cliente fino**: según `docs/informe_revision_reglas_y_deuda_2026-03-24.md`, buena parte de la superficie principal ya fue reencaminada a `tareasapp`, `propuestasapp`, `operacionesapp`, `lenguajeapp`, `agentesapp`, `panelapp`, `runtimesapp` y `gitgobernanza`. La deuda que sigue abierta en `cmd/serve.go` ya no es “toda la web contra db”, sino mezcla residual de tipos/filtros `db.*`, enums del dominio (`db.EstadoTarea`, `db.PrioridadTarea`, `db.PosicionVoto`) y algunos handlers de control que aún llaman a lógica local como `encolarControlAgenteLocal(...)`.
12. **La deuda server-first es cuantificable**: dentro de `cmd/` hay 50 puntos que todavía llaman a `ensureLocalDB()`. No es una excepción residual; sigue siendo un patrón extendido en tareas, sesiones, agentes, worktrees, propuestas, logs, auditoría, exportación y runtime. Conviene tratarlo como programa de refactorización, no como ajuste menor.
13. **El acoplamiento de `cmd/` con `db` es estructural**: 103 ficheros `.go` dentro de `cmd/` referencian `db.*` directamente. Esto confirma que la capa adaptadora está sobredimensionada y absorbe lógica de aplicación de forma transversal, no solo en unos pocos comandos heredados.
14. **Incumplimiento de cabeceras GPLv3 en código de producción**: al menos `tareasapp/service.go`, `runtimesapp/service.go`, `cmd/status_service.go`, `db/backend_postgres.go`, `db/backend_mysql.go` y `db/coordinacion_backend.go` no incorporan la cabecera GPLv3 exigida por las reglas activas. Si los tests quedan fuera del criterio de licencia, la revisión debe centrarse en producción y en archivos nuevos no-test antes del merge global.
15. **Abstracción de backend no equivale a multi-tenant**: `db/backend_postgres.go` y `db/backend_mysql.go` abren el camino a soportar varios motores, pero siguen operando sobre una conexión/configuración global y no implementan resolución por tenant ni `tenantDB(ctx)`. La portabilidad de motor está avanzando, pero el aislamiento multi-tenant sigue pendiente.
16. **Separación de coordinación en progreso, aún dentro de `db`**: `db/coordinacion_backend.go` es un paso correcto porque devuelve interfaces `coordinacion.*Repository`, pero la implementación sigue residiendo en `db` y no está aún desacoplada de la conexión global ni del futuro backend por tenant. Es una base útil para la refactorización, no el estado final.
17. **Deriva documental en el manual del programador**: `docs/manual_programador.md` ya no refleja el estado real del sistema. Sigue hablando del adaptador `pkg/db` (que no existe en el repositorio actual) y sitúa MCP en `http://localhost:3000/mcp`, mientras el código define `defaultServerURL = http://127.0.0.1:16543` en `cmd/server_defaults.go`. Esta desalineación puede provocar integraciones equivocadas y debe corregirse junto al bloque server-first.
18. **Directorios duplicados vacíos que añaden ruido de capa**: existen carpetas inglesas vacías como `capacityapp`, `taskapp`, `sessionapp`, `proposalapp`, `importapp` y `configapp`, mientras la implementación real vive en `capacidadapp`, `tareasapp`, `sesionesapp`, `propuestasapp`, `importacionapp` y `configuracionapp`. No bloquean la ejecución, pero sí inducen ambigüedad sobre qué paquete es la fuente real y pueden generar trabajo en la ruta equivocada.
19. **Regla de módulos opcionales sin implementación visible**: la exigencia de “módulos opcionales vía `MODULES`” aparece en `db/schema.go` y `db/historial.go`, pero no se observan usos reales de `MODULES` en `cmd/`, `internal/` o `db/`. A día de hoy no hay evidencia de instanciación condicional por módulo en el runtime principal, así que esta regla debe considerarse pendiente.
20. **TLS/mTLS validado pero no aplicado en el servidor unificado**: `cmd/serve_compat.go` introduce `validateServeSecurity(...)` y exige `tls-client-ca` para exposición remota, pero `cmd/servidor_unificado.go` arranca el servicio con `http.ListenAndServe(...)`, sin `ListenAndServeTLS`, sin `http.Server` con `TLSConfig` y sin `ClientAuth`. En la práctica, la validación previa no se traduce en transporte seguro real para el servidor HTTP principal.
21. **JWT RS256 y Argon2id no aparecen como implementación viva**: las reglas activas de seguridad están sembradas en `db/schema.go`, pero el rastreo en `cmd/`, `db/` e `internal/` no muestra autenticación JWT RS256 ni gestión de contraseñas con Argon2id para el servidor operativo. Lo que sí existe en `internal/rpclocal` es un token técnico por cabecera (`X-Orquesta-Token`) para el RPC local, útil como control de canal interno, pero no equivalente a autenticación de usuarios/agentes ni a las exigencias de seguridad aprobadas.
22. **RBAC parcial y no sistemático por endpoint**: hay comprobaciones de rol y permisos en zonas concretas como `db/catalogo_versionado.go`, y muchas consultas filtran por `rol`/`habilitado`, pero no se observa una capa uniforme de autorización del tipo “usuario → grupos AD → roles internos → permisos → acción” aplicada de forma transversal a cada endpoint. La autorización actual es más bien puntual y orientada a tablas o casos concretos, no un RBAC completo de plataforma.
23. **Auditoría obligatoria implementada de forma plana**: `db.Audit(...)` inserta directamente en `audit_log`, y la tabla definida en `db/schema.go` solo contiene `agente`, `accion`, `entidad`, `detalle` y `created_at`. No aparecen campos ni lógica para SHA-256 encadenado, retención a 5 años o cifrado AES-256-GCM at rest; lo que existe es una detección textual defensiva de patrones sospechosos antes de insertar. La plataforma sí audita muchas mutaciones, pero no con el modelo de integridad/cifrado exigido por las reglas activas.
24. **La regla multi-tenant de seguridad sigue siendo declarativa**: el rastreo en `cmd/`, `db/` e `internal/` no devuelve usos de `tenantDB(ctx)` ni una implantación activa de `tenant_id` como frontera o de BD aislada por ayuntamiento. La regla está recogida en `db/schema.go` y `db/historial.go`, pero hoy no hay evidencia de enforcement operativo en el servidor.

## Próximos pasos

1. Trazar las zonas donde la lógica (p. ej. `cmd/api.go`, `cmd/*`) aún manipula `db.*` directamente y evaluar si puede moverse a servicios dentro de `internal/`.
2. Verificar la cobertura actual del servidor: qué endpoints y operaciones aún se ejecutan localmente o exigen `scripts/inicio_agente.sh`.
3. Registrar hallazgos en este informe y proponer acciones concretas (mover componentes a adaptadores, reforzar i18n, eliminar dependencias directas con la DB).
4. Contrastar las reglas de seguridad declaradas (TLS, JWT, Argon2id, RBAC, auditoría y multi-tenant) con la implementación viva para priorizar qué bloque debe abordarse antes del merge global.

## Plan de corrección (primeros bloques)

1. **Delimitar los puertos del control plane**: Definir interfaces (puertos) dentro de `internal/controlplane` o `internal/api` que agrupen operaciones sobre proyectos, tareas, sesiones, reglas y propuestas; los handlers en `cmd/api.go` solo deberían inyectar esas interfaces y delegar en implementaciones situadas en `internal/db` o `internal/controlplane`. Esto permite desacoplar la lógica del adaptador HTTP y cumplir la regla de “interfaces primero” (OP-056). Las especificaciones técnicas de `docs/diseno_control_activo_agentes.md` y `docs/runbook_control_plane_agentes.md` servirán de base para el comportamiento esperado del control plane desde el servidor.
2. **Implementar `tenantDB(ctx)` y aislar las bases por ayuntamiento**: Crear un paquete `internal/multitenant` que resuelva la conexión por contexto y sustituya los `db.DB` globales actuales, reajustando los repositorios (`db/*.go`) para usar ese constructor. Priorizar esto antes de cerrar el módulo del servidor MCP (tarea #395).
3. **Cerrar el modo local a favor del servidor MCP**: Eliminar o adaptar `scripts/inicio_agente.sh` para que hable exclusivamente con `http://localhost:8080` y el control plane del daemon; configurar `ORQUESTA_REQUIRE_SERVER=1` como opción por defecto y documentar que `--local` es solo para recuperación explícita. Esta acción apoya la política “Servidor primero; local solo recuperación”.
4. **Llegar al pack i18n completo**: Crear los ficheros `de.json`, `fr.json`, `it.json`, `zh.json`, `gl.json`, `eu.json`, `ca.json`, `val.json` en `i18n/`, con claves base compartidas por proyecto y un `bundle.go` que los envuelva con fallback a `es`. También hay que permitir la selección de idioma por configuración de usuario/despliegue y referenciarlo en los comandos `tr` de `cmd/serve.go` y `cmd/api.go`. El contrato del esqueleto i18n ya está detallado en `docs/plantillas_i18n/README_es.md` y en `i18n/project_skeleton.go`, por lo que la tarea consiste en materializar esos templates y enlazar la configuración con la política `docs/politica_multilenguaje_por_defecto.md`.
5. **Actualizar documentación y manuales**: Revisar `docs/uso_actual_app_orquesta.md` y otros manuales para dejar claro que la vía oficial actual es el servidor MCP/daemon. Documentar en `docs/informe_revision_hexagonal.md` los cambios propuestos y la relación con las tareas #394/395/396.
6. **Cerrar la brecha de seguridad declarativa vs. operativa**: separar el token técnico de `internal/rpclocal` de la autenticación real del servidor, introducir transporte TLS/mTLS efectivo en el servidor HTTP principal, diseñar una capa de autorización transversal y redefinir la auditoría para que soporte integridad encadenada y cifrado en reposo antes de considerar cerrado el camino server-first.

## Brechas específicas para que el servidor orqueste agentes

1. **El control de agentes sigue encapsulado en helpers locales**: tanto `apiHandlerAgenteControl(...)` como el comando `agente control` terminan en `encolarControlAgenteLocal(...)` dentro de `cmd/agente_control_lifecycle.go`. Aunque se expone una ruta HTTP, la lógica real de validación, resolución de proyecto/handle y encolado de órdenes sigue residiendo en el adaptador `cmd/`, no en un servicio del control plane reutilizable.
2. **El control plane depende de `dbAutomationService`, no de puertos de aplicación**: `cmd/controlplane_support.go` construye el runner con una implementación que llama directamente a `db.CheckReanimaciones`, `db.PlanificarTareasAutomaticamente`, `db.ProcesarRuntimeOrdersBatch`, `db.ProcesarRefineriaBatch`, `db.ProcesarHandoffsBatch` y `db.Audit`. Eso hace que el “orquestador” actual sea básicamente un coordinador local sobre `db.*`, no una capa independiente sobre interfaces.
3. **La autonomía de agentes usa rutas locales del mismo adaptador**: `procesarAutonomiaSesionActiva(...)` y `reactivarAgenteTrasReanimacion(...)` vuelven a `encolarControlAgenteLocal(...)` y a consultas directas sobre `db.*`. La automatización existe, pero todavía no está desacoplada del mismo paquete `cmd/` que debería limitarse a entrada/salida.
4. **Runtime y mailbox siguen anclados a `db.*`**: `cmd/runtime.go` lista handles, órdenes, checkpoints, mailbox, runtimes y además crea/enfila órdenes directamente con `db.EncolarRuntimeOrder(...)`, `db.CrearRuntimeCheckpoint(...)`, `db.EnviarRuntimeMailbox(...)`, etc. Mientras esto siga así, el control operativo del runtime no pertenece realmente al servidor como dominio orquestador, sino al CLI/API como fachada con acceso a persistencia.
5. **El servidor unificado es local por scope, no un control plane desacoplado**: `cmd/server.go` levanta un daemon local apoyado en `internal/rpclocal`, autoarranca si hace falta y persiste un statefile por scope. Eso sirve para centralizar el workspace actual, pero todavía responde al patrón “servidor local del CLI” más que a un servicio de orquestación con límites nítidos, seguridad fuerte y puertos de dominio estables.
6. **La política server-first depende de reachability y fallback, no de cierre estructural**: `shouldPreferAPIClient(...)` solo usa la API si el servidor está alcanzable, y el CLI aún conserva caminos de recuperación/local. Operativamente ayuda, pero desde el punto de vista de orquestación aún no hay cierre definitivo del acceso directo en comandos críticos (`tarea`, `agente`, `runtime`, `propuesta`, `sesion`, `worktree`).
7. **Falta un servicio explícito de orquestación de agentes**: hoy existen piezas (`controlplane`, `runtime`, `handoff`, `autonomía`, `mailbox`), pero no un `AgentOrchestrationService` o puerto equivalente que concentre lifecycle, asignación de runtime, handoff, pausas, reanudación, nudges y resolución de estado. Esa ausencia explica por qué la lógica está repartida entre `cmd/agente.go`, `cmd/agente_control_lifecycle.go`, `cmd/runtime.go` y `cmd/controlplane_support.go`.

### Consecuencia práctica

Para que el servidor “se haga cargo de los agentes” de verdad no basta con exponer más endpoints. Hay que mover el ciclo de vida del agente, las órdenes runtime, los nudges de autonomía, el mailbox y el handoff a un servicio de aplicación/control plane con interfaces propias, y dejar que CLI, web y API solo invoquen ese servicio. Mientras `cmd/` siga creando órdenes y resolviendo estado directamente contra `db.*`, el servidor seguirá siendo una centralización táctica, no el orquestador canónico del sistema.

### Forma mínima recomendada del servicio de orquestación

Tomando como referencia el patrón ya visible en `coordinacion.Service`, el siguiente corte razonable sería introducir un servicio explícito de orquestación de agentes, por ejemplo `orquestacionagentes.Service`, con dependencias inyectadas por puerto:

1. `AgentRepository`
   Para obtener estado del agente, pausas, reanimación y rol.
2. `ProjectRepository`
   Para resolver proyecto activo, slug y contexto del agente.
3. `SessionRepository`
   Para sesión activa, heartbeat y contexto de continuidad.
4. `RuntimeRepository`
   Para handles, runtime instances, checkpoints y mailbox.
5. `RuntimeOrderRepository`
   Para encolar, listar y reconciliar órdenes.
6. `TaskRepository`
   Para decidir si el agente tiene trabajo arrancable o requiere nudge.
7. `ProposalRepository`
   Para pendientes de voto y señalización de gobernanza.
8. `AuditRepository`
   Para registrar eventos de control plane sin depender de `db.Audit(...)` desde adaptadores.

Con esa base, los casos de uso mínimos deberían salir de `cmd/`:

1. `EncolarAccionAgente(...)`
   Sustituye a `encolarControlAgenteLocal(...)`.
2. `ProcesarAutonomiaSesionActiva(...)`
   Sustituye la lógica de `cmd/controlplane_support.go`.
3. `ReactivarAgente(...)`
   Centraliza la reanimación y el relanzamiento.
4. `EnviarNudge(...)`
   Centraliza mailbox/autonomía.
5. `ResolverHandleActivo(...)`
   Evita que CLI/API resuelvan estado runtime directamente.
6. `PrepararHandoff(...)` y `ReanudarHandoff(...)`
   Unifica el ciclo de handoff vivo y checkpoint.

CLI, web y API deberían limitarse entonces a:

1. validar entrada/salida,
2. invocar el servicio,
3. serializar la respuesta.

Ese es el salto que falta para que el servidor deje de ser “un CLI con daemon local” y pase a ser realmente el orquestador del sistema.

El “Inventario de pendientes para orquestación autónoma” (`docs/inventario_pendientes_orquestacion_autonoma_2026-03-24.md`) ya señala los pasos críticos (cubrir el daemon oficial, trasladar scripts a cliente fino, validar runtime vivo y cerrar la web/i18n). El plan anterior se apoya en esa hoja de ruta para no duplicar esfuerzos y para alinear las tareas #395 y #396 con los asuntos priorizados (#361, #362, #357, #244, #363, #364).

### Esquema preliminar para los puertos

1. `internal/controlplane/ProjectPort` → métodos para `Listar`, `Descubrir` y `Get`.
2. `internal/controlplane/TaskPort` → `Listar`, `Tomar`, `Completar`, `Registrar`, `Notas`.
3. `internal/controlplane/SessionPort` → `Iniciar`, `Guardar`, `Continuar`, `Finalizar`, `Summary`.
4. `internal/controlplane/ProposalPort` → `Listar`, `Votar`, `Cerrar`, `Reabrir`.
5. `internal/controlplane/CoordinationPort` → `Worktree`, `Lock`, `Runtime` (ejecutando `coordinacion.Service`).
6. Cada puerto se implementa sobre repositorios `db.*` (repos por entidad) que, a su vez, reciben la conexión resuelta por `internal/multitenant/tenantDB(ctx)` y exponen solo operaciones sin exponer `*sql.DB` al núcleo.
7. Los handlers `cmd/api.go`, `cmd/worktree.go`, `cmd/agente_control_lifecycle.go`, etc., inyectan estas interfaces y no hablan con `db.*` directamente; cuando el servidor está activo, la API y la CLI deben usar los clientes HTTP (ver `cmd/cliente_servidor.go` y `cargarWorktreesDesdeAPI`).

## Priorización sugerida

1. Corregir la incompatibilidad i18n entre `project_skeleton` y `bundle`, porque ahora mismo la política aprobada no puede materializarse de forma consistente.
2. Sacar `cmd/api.go`, `cmd/tarea.go`, `cmd/worktree.go` y `cmd/agente*.go` del acceso directo a `db.*`, empezando por puertos de tareas, sesiones y coordinación.
3. Endurecer el camino server-first: mantener el fallback local solo para diagnóstico/recuperación y cerrar el uso normal de `ensureLocalDB()` en comandos de negocio.
4. Introducir `tenantDB(ctx)` y repositorios multi-tenant antes de consolidar el servidor MCP como camino oficial único.
5. Corregir la brecha de seguridad operativa: hoy el servidor unificado arranca en HTTP sin TLS real, no hay evidencia de JWT RS256/Argon2id ni de auditoría encadenada, y el RBAC es parcial. Este bloque ya afecta a reglas activas, no solo a calidad interna.
6. Actualizar manuales y runbooks para que dejen de recomendar wrappers locales como flujo diario.

## Riesgo para merge global

- **Crítico**: incompatibilidad i18n entre generación y carga; uso extendido de `db.*` en `cmd/`; fallback local aún operativo en rutas de negocio; servidor HTTP principal sin TLS efectivo pese a existir regla activa de TLS obligatorio.
- **Alto**: ausencia de `tenantDB(ctx)` y multi-tenant real; no hay evidencia de JWT RS256/Argon2id operativos; auditoría plana sin SHA-256 encadenado ni AES-256-GCM; control activo y runtime todavía parcialmente fuera de puertos/servicios; documentación todavía mezclada entre flujo local y server-first.
- **Medio**: cabeceras GPL pendientes en producción; limpieza residual de enums/tipos `db.*` en web y API; ampliación de tests de frontera.
- **Bajo**: ajustes cosméticos o residuales de documentación una vez cerrado el camino oficial del servidor.

## Bloques paralelizables recomendados

1. **Bloque i18n**
   Ficheros frontera: `i18n/`, `docs/plantillas_i18n/`, `cmd/serve*_test.go`, `cmd/lenguaje*_test.go`.
   Objetivo: unificar `project_skeleton` y `bundle`, fijar fallback real y asegurar que el contrato generado es consumible.

2. **Bloque server-first / fallback local**
   Ficheros frontera: `cmd/root.go`, `cmd/cliente_servidor.go`, `cmd/server.go`, `cmd/root_gating_test.go`, `cmd/cliente_servidor_test.go`.
   Objetivo: cerrar el uso normal de `ensureLocalDB()` y reservar local solo para recuperación explícita.

3. **Bloque API / CLI de tareas-sesiones-propuestas**
   Ficheros frontera: `cmd/api.go`, `cmd/tarea.go`, `cmd/sesion.go`, `cmd/propuesta.go`, `tareasapp/`, `sesionesapp/`, `propuestasapp/`.
   Objetivo: reencaminar endpoints y comandos a servicios `*app`, dejando `cmd/` como adaptador.

4. **Bloque runtime / control plane**
   Ficheros frontera: `cmd/runtime.go`, `cmd/agente.go`, `cmd/agente_control_lifecycle.go`, `cmd/worktree.go`, `coordinacion/`, `runtimesapp/`, `agentesapp/`.
   Objetivo: mover lifecycle, runtime orders, handoff y worktrees detrás de servicios/puertos ya existentes.

5. **Bloque multi-tenant / backend**
   Ficheros frontera: `db/backend_*.go`, `db/coordinacion_backend.go`, `db/schema_backend.go`, `storage/`, futura capa `internal/multitenant/`.
   Objetivo: introducir `tenantDB(ctx)` y separar soporte multi-motor de soporte multi-tenant real.

6. **Bloque documentación y cumplimiento**
   Ficheros frontera: `docs/manual_programador.md`, `docs/uso_actual_app_orquesta.md`, `docs/informe_revision_hexagonal.md`, cabeceras GPL en producción.
   Objetivo: alinear manuales con el código actual y cerrar ruido de licencia/rutas duplicadas antes del merge.

## Base ya aprovechable

- `coordinacion.Service` ya ofrece una forma razonable de encapsular reglas de aplicación para locks/worktrees.
- `cmd/cliente_servidor.go` ya delimita qué comandos deben ser server-first y da un punto central para endurecer la política.
- `i18n/project_skeleton.go` ya incorpora el pack inicial de idiomas aprobado y el contrato por dominio.
- Existen servicios de aplicación reutilizables en `tareasapp`, `sesionesapp`, `runtimesapp` y `agentesapp` con patrón `Service + Store`, lo que reduce el coste de mover `cmd/` hacia adaptadores más finos.
- `db/coordinacion_backend.go` y los backends `postgres/mysql` muestran que ya hay piezas orientadas a interfaz y a soporte multi-motor, aunque todavía no resuelvan el tenant.
- También existen tests útiles para esta dirección: `cmd/architecture_test.go` intenta impedir SQL directo en `cmd/`, `cmd/cliente_servidor_test.go` valida la cobertura server-first, y `tareasapp/service_test.go` comprueba la capa `Service + Store` con stubs.
- `docs/app_layer_packages_estado_2026-03-25.md` confirma que `agentesapp`, `lenguajeapp` y `runtimesapp` no son ruido, sino parte útil de la migración por capas ya en marcha.

La refactorización no parte de cero. La deuda principal es hacer que los adaptadores `cmd/` y `api` dejen de saltarse esas capas y que el cargador i18n consuma el mismo contrato que genera el esqueleto.

## Matiz importante

La arquitectura actual es híbrida, no puramente monolítica:

- Hay zonas todavía muy acopladas a `db.*` dentro de `cmd/`.
- Pero también hay paquetes `*app` que ya separan una capa de servicio y un `Store`.
- Incluso dentro de `cmd/` empiezan a aparecer extracciones parciales como `statusService` y el uso de `capacidadService`, que pueden tomarse como patrón de transición.
- La deuda más rentable de corregir no es “crear servicios desde cero”, sino obligar a que CLI, API y web usen consistentemente esos servicios y, en una segunda fase, desacoplar esos `Store` de tipos/filtros concretos de `db`.

## Hotspots de deuda

- `cmd/api.go` concentra el mayor volumen de referencias directas a `db.*` dentro de `cmd/` y debe ser el primer candidato a extracción de puertos.
- Después aparecen `cmd/runtime.go`, `cmd/agente.go`, `cmd/tarea.go`, `cmd/serve.go`, `cmd/propuesta.go` y `cmd/catalogo_briefing.go`, todos con lógica de negocio mezclada con acceso a persistencia.
- `cmd/serve.go` no solo presenta tipos basados en `db.*`; también construye filtros, estados y lecturas de tareas/propuestas/runtimes desde el propio adaptador web.
- `cmd/runtime.go` sigue ofreciendo lecturas y mutaciones del control plane con acceso directo a `db.*` cuando falla el API; eso lo convierte en otro punto prioritario del cierre server-first.
- `cmd/agente.go` mezcla briefing, sesiones, tareas, propuestas, handoff, pausas y configuración directamente contra `db.*`, por lo que es un hotspot crítico del control plane.

## Ranking cuantificado de deuda en `cmd/`

- Referencias directas a `db.*` en producción dentro de `cmd/`: **801**.
- Llamadas a `ensureLocalDB()` en producción dentro de `cmd/`: **50**.

### Top ficheros por referencias directas a `db.*` (producción)

1. `cmd/api.go` → 209
2. `cmd/cliente_servidor.go` → 50
3. `cmd/runtime.go` → 40
4. `cmd/agente.go` → 40
5. `cmd/cliente_servidor_recursos.go` → 37
6. `cmd/tarea.go` → 36
7. `cmd/serve.go` → 31
8. `cmd/mcp.go` → 28
9. `cmd/controlplane_support.go` → 26
10. `cmd/propuesta.go` → 22
11. `cmd/catalogo_briefing.go` → 22

### Top ficheros por uso de `ensureLocalDB()` (producción)

1. `cmd/tarea.go` → 14
2. `cmd/propuesta.go` → 8
3. `cmd/agente.go` → 8
4. `cmd/sesion.go` → 5
5. `cmd/memoria.go` → 3
6. `cmd/worktree.go` → 2
7. `cmd/exportar.go` → 2

Estas métricas refuerzan la estrategia de corte propuesta: el primer frente de refactorización debe entrar por `cmd/api.go`, `cmd/tarea.go`, `cmd/agente.go`, `cmd/runtime.go`, `cmd/propuesta.go` y `cmd/sesion.go`. Son los adaptadores con mayor retorno inmediato tanto para reducir acoplamiento hexagonal como para cerrar la fuga al modo local.

## Estado real del bloque i18n

- En el árbol `i18n/` solo existen hoy `es.json` y `en.json`, además de `bundle.go` y `project_skeleton.go`.
- `i18n/bundle.go` recorre únicamente ficheros `*.json` en la raíz de `i18n/`, interpreta el nombre del fichero como idioma y construye un único diccionario plano por idioma.
- `i18n/project_skeleton.go`, en cambio, materializa un contrato distinto: `i18n/config.json` más carpetas por idioma (`i18n/<lang>/`) y un fichero por dominio (`<domain>.json`), con el pack base `es`, `en`, `de`, `fr`, `it`, `zh`, `gl`, `eu`, `ca` y `val`.
- La brecha no es solo de idiomas faltantes; también es de formato y de cargador. Incluso si se añadieran mañana los idiomas pendientes, el `bundle` actual no podría consumir el contrato aprobado sin una refactorización explícita.

## Secuencia práctica de ataque

1. Extraer primero `TaskPort`, `SessionPort` y `ProposalPort`, porque son los más reutilizados por CLI, API y web.
2. Mover después `CoordinationPort` y `RuntimePort` para despejar `cmd/agente.go`, `cmd/agente_control_lifecycle.go`, `cmd/worktree.go` y `cmd/runtime.go`.
3. Refactorizar `cmd/api.go` y `cmd/serve.go` para que queden como adaptadores finos sobre esos puertos.
4. Refactorizar `cmd/runtime.go` y `cmd/agente*.go` para que runtime control, handoff y lifecycle queden detrás del control plane.
5. Endurecer el modo server-first una vez que la cobertura real del cliente HTTP sustituya a `ensureLocalDB()` en los comandos de negocio.

## Patrón de migración recomendado para `cmd/api.go`

1. Mantener el archivo como router y adaptador HTTP, no como capa de aplicación.
2. Extraer endpoint por endpoint hacia servicios existentes (`tareasapp`, `sesionesapp`, `runtimesapp`, `agentesapp`, `capacidadapp`) o servicios nuevos donde falten.
3. Usar el enfoque ya visible en `statusService`: introducir interfaces pequeñas y reemplazar llamadas directas a `db.*` por dependencias inyectables.
4. Dejar para una segunda fase la limpieza profunda de tipos `db.*` en firmas de `Store`; primero hay que sacar la lógica de negocio de `cmd/`.

## Mapa resumido por capa

- `cmd/`: adaptadores sobredimensionados; concentran la mayor parte del bypass a `db.*` y del fallback local.
- `*app`: capa de aplicación útil pero todavía fina; ya encapsula casos de uso en varios dominios (`tareasapp`, `sesionesapp`, `runtimesapp`, `agentesapp`, `lenguajeapp`, `capacidadapp`).
- `coordinacion/`: mejor aproximación actual a puertos explícitos en dominio/aplicación.
- `db/`: sigue siendo tanto adaptador de persistencia como contenedor de demasiada lógica y tipos compartidos.
- `i18n/`: buena dirección en generación de esqueleto, pero contrato de almacenamiento incompatible con el cargador activo.
- `docs/`: valiosos para orientar la migración, pero con deriva puntual en manuales operativos/técnicos.
- Directorios `*app` en inglés vacíos: ruido estructural que conviene limpiar o documentar para evitar duplicados accidentales.

## Matriz rápida de cumplimiento de reglas

- **Arquitectura hexagonal / Clean Architecture**: parcial.
  Hay servicios `*app` y puertos en `coordinacion/`, pero `cmd/` sigue concentrando demasiado acceso a `db.*`.
- **Servidor primero; local solo recuperación**: parcial.
  La política existe y el gating está implantado, pero siguen abiertas 50 rutas con `ensureLocalDB()` y scripts/manuales locales.
- **i18n obligatorio antes de nuevos textos/UI**: parcial con brecha crítica.
  Existe esqueleto i18n, pero `project_skeleton` y `bundle` no comparten formato.
- **Multi-tenant aislado con `tenantDB(ctx)`**: incumplido.
  Solo hay referencia documental; no aparece implementación operativa.
- **TLS obligatorio en todos los endpoints**: incumplido.
  `validateServeSecurity(...)` existe, pero el servidor unificado arranca con `http.ListenAndServe(...)` sin TLS/mTLS efectivo.
- **JWT RS256 + Argon2id**: incumplido o no evidenciado.
  No aparecen implementaciones vivas de autenticación JWT RS256 ni contraseñas Argon2id en el servidor operativo.
- **RBAC completo**: parcial.
  Hay comprobaciones puntuales por rol/permisos, pero no una cadena uniforme de autorización aplicada a cada endpoint.
- **Auditoría SHA-256 encadenada + AES-256-GCM at rest**: incumplido.
  La auditoría existe, pero `audit_log` es plano y `db.Audit(...)` solo realiza inserciones directas sin encadenado ni cifrado visible.
- **Módulos opcionales vía `MODULES`**: incumplido o no evidenciado.
  No se detecta uso real de `MODULES` en el código.
- **Conectores independientes del núcleo**: parcial, con buena base.
  `runtimeagente/driver.go` y `planocontrol/runner.go` ya modelan transporte, driver y automatización sin casarse con un terminal concreto. Sin embargo, scripts como `runtime_connector.sh`, `agente_console.sh` y `launch_agentes.sh`, junto con backends `tmux`/Terminator/PTy, siguen formando parte de la operativa real y el control activo todavía cae en parte a `db.*` y lógica local.
- **Cabecera GPLv3 en producción**: parcial.
  El repositorio la usa extensamente, pero hay ficheros de producción recientes sin cabecera.

## Situación de pruebas respecto a la arquitectura

- Los tests existentes ya capturan parte de la intención hexagonal y server-first.
- Aun así, no bastan para frenar la deriva actual, porque conviven con 103 ficheros de `cmd/` que siguen usando `db.*` y con 50 rutas que pueden caer a `ensureLocalDB()`.
- En particular, `cmd/architecture_test.go` protege contra SQL directo y aperturas de BD fuera de excepciones controladas, pero no prohíbe el uso generalizado de `db.*` desde `cmd/`. Por eso puede pasar ese test y, aun así, mantenerse un acoplamiento fuerte entre adaptadores y persistencia.
- También hay tests puntuales útiles de seguridad, como `cmd/compat_test.go`, pero hoy solo validan la función previa `validateServeSecurity(...)`; no garantizan que el servidor unificado levante realmente con TLS/mTLS ni que exista autenticación/autorización acorde a las reglas activas.
- La siguiente mejora útil en pruebas no es “más tests genéricos”, sino tests de frontera: que CLI/web/API no llamen a `db.*` directamente cuando existe servicio `*app`, y que el contrato i18n generado por `project_skeleton` sea consumible por `bundle`.

## Pruebas hexagonales que faltan

1. **Guardarraíl `cmd -> db` más estricto**: ampliar `cmd/architecture_test.go` para prohibir referencias generales a `db.*` en adaptadores donde ya exista servicio `*app` o puerto equivalente, no solo SQL directo y `db.Open/Close`.
2. **Cobertura de cliente fino real**: añadir tests que fallen si comandos server-first (`tarea`, `sesion`, `propuesta`, `runtime`, `worktree`, `agente control`) vuelven a ejecutar lógica local de negocio cuando el servidor está disponible.
3. **Arranque seguro del servidor**: crear tests sobre el arranque real de `orquesta serve` o del servidor unificado para verificar que, si se solicita exposición remota, la ruta efectiva usa TLS/mTLS y no solo `validateServeSecurity(...)`.
4. **Contrato i18n extremo a extremo**: probar que el formato generado por `i18n/project_skeleton.go` es exactamente consumible por `i18n/bundle.go`, incluyendo fallback a `es` y selección por idioma.
5. **Puertos de aplicación con stubs dobles**: replicar el patrón de `tareasapp/service_test.go` en `sesionesapp`, `propuestasapp`, `agentesapp`, `runtimesapp` y coordinación, reforzando que la lógica de aplicación se valida sin tocar persistencia real.
6. **Protección del modo local**: tests que demuestren que `--local` solo habilita recuperación explícita, no el flujo normal de negocio.
7. **Autorización por endpoint**: cuando exista la capa RBAC real, añadir pruebas por endpoint/handler para demostrar que la autorización vive en el borde de entrada y no dispersa en consultas ad hoc.
