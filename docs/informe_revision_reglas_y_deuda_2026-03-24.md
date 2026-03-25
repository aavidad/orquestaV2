<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Informe de revisión: reglas, deuda técnica y limpieza

Fecha: 2026-03-24
Revisión: Codex3

## Objetivo

Revisar la app de arriba a abajo para detectar:

- desviaciones respecto a reglas activas del proyecto
- deuda técnica que choque con la dirección de producto
- código redundante o probablemente muerto
- mejoras claras y ordenadas para saneamiento

## Reglas contrastadas

La revisión se ha centrado especialmente en estas reglas ya activas en el repo:

- cliente fino para web y futuros clientes sobre Orquesta
- i18n real y multilenguaje donde aplique
- evitar acoplar lógica o presentación directamente a SQLite / `db`
- reutilización por capas según complejidad
- evitar duplicación de superficies y caminos alternativos sin necesidad

Referencias de política y contexto:

- `docs/politica_arquitectura_tipos_proyecto_es.md`
- `docs/politica_arquitectura_tipos_proyecto_en.md`
- `docs/politica_multilenguaje_por_defecto.md`
- `docs/diseno_app_escritorio_orquesta.md`

## Validación técnica ejecutada

- `go vet ./...` → OK
- varias baterías focalizadas de `cmd`, `planocontrol` e `internal/rpclocal` → OK durante la sesión
- batería focalizada final de `./cmd` sobre i18n web, dashboard, agentes, gitgov y time-travel → OK
- `go test ./agentesapp ./lenguajeapp ./runtimesapp ./gitgobernanza ./operacionesapp ./propuestasapp ./panelapp ./cmd -count=1` → OK
- `go test ./cmd -count=1 -timeout=180s` → OK
- `go test ./cmd -run 'TestCmdNoUsaSQLDirectoNiAperturasFueraDeExcepcionesControladas$' -count=1` → OK
- `go test ./db -run 'Test(TomarMuestraGenericProcessProcesoActual|RegistrarMuestraGenericProcessPersisteYActualizaRuntime)$' -count=1` → OK
- `go test ./db -count=1 -timeout=300s` → OK
- `go test ./... -count=1 -timeout=600s` → OK

## Progreso aplicado durante la revisión

Ya se ha corregido una parte sustancial del Bloque 1:

- resolución real de idioma por request en web mediante query, cookie y `Accept-Language`
- `Content-Language` dinámico
- `<html lang="...">` dinámico
- `webRender` con funciones de plantilla ligadas al idioma resuelto

Superficies ya barridas y validadas con tests focalizados:

- `cmd/serve.go` para dashboard, tareas y propuestas
- `cmd/lenguaje_web.go`
- `cmd/ops_web.go`
- `cmd/gitgov_web.go`
- `cmd/proyectos_web.go`
- `cmd/gobernanza_web.go`
- `cmd/agentes_web.go`

Cobertura añadida o ampliada:

- `cmd/serve_i18n_test.go`
- `cmd/lenguaje_web_test.go`
- `cmd/ops_web_test.go`
- `cmd/gitgov_web_test.go`
- `cmd/proyectos_web_test.go`
- `cmd/gobernanza_web_test.go`
- `cmd/agentes_web_test.go`

Conclusión de estado:

- el incumplimiento de i18n web ya no es estructural en la infraestructura común
- el bloque visible principal de la web ya está cubierto también en `cmd/serve.go`
- los flashes visibles y los literales duros detectados en la web revisada quedaron ya normalizados o sustituidos por claves
- el siguiente frente importante deja de ser interfaz y pasa a ser consolidación de capa fina
- el endpoint `/api/status` vuelve a incluir el detalle de votos de propuestas abiertas
- la telemetría `generic_process` dejó de escanear todo `/proc` para contar hijos y usa una lectura acotada de `/proc/<pid>/task/<pid>/children`
- los tests de `cmd` con BD temporal ya no heredan `DSN/driver/backend` del entorno y quedan aislados sobre SQLite temporal real
- el naming y la retención de respaldos vuelven a depender del backend activo mediante helpers canónicos de `db`
- los helpers de tests de `db` también quedaron unificados y aislados del entorno global de almacenamiento
- los tests antiguos de `db` que abrían sesión sin registrar agente quedaron alineados con la regla actual

Progreso adicional de capa aplicado durante la revisión:

- `cmd/serve.go` ya usa `tareasapp` para listar, crear, leer y accionar tareas
- `cmd/serve.go` ya usa `propuestasapp` para listar, leer detalle, crear, votar, cerrar, reabrir y reparar votos pendientes
- `cmd/ops_web.go` ya usa `operacionesapp` también para sesiones de inspección y detalle, no solo para asignaciones
- `cmd/lenguaje_web.go` ya usa `lenguajeapp`
- `cmd/agentes_web.go` ya usa `agentesapp`
- `cmd/serve.go` ya usa `panelapp` y `runtimesapp` para dashboard, runtimes y time-travel
- `cmd/gitgov_web.go` ya usa `gitgobernanza` también para detalle de worktree y lock
- `cmd/exportar_web.go` ya usa `operacionesapp` para auditoría en vez de ir directo a `db`
- `cmd/servidor_unificado.go` ya no abre la BD directamente; la apertura queda reencaminada a un helper en `cmd/server.go`, respetando el guardarraíl de arquitectura
- se eliminó la capa duplicada de wrappers “API web” no montados en `ops`, `gitgov`, `proyectos` y `gobernanza`

Conclusión:

- la deuda de “web contra db directo” sigue existiendo, pero ya no es tan homogénea ni tan extendida como al inicio de la revisión

## Estado final de hallazgos

### 1. I18n web visible

Impacto:

- el núcleo visible ya está barrido y cubierto con tests
- la deuda restante ya no está en el motor i18n ni en las pantallas principales
- lo que queda son residuos puntuales y mensajes no modelados como claves

Conclusión:

- la resolución por request funciona
- la superficie visible revisada quedó sin literales duros detectables en plantillas
- no queda un incumplimiento visible abierto en la web revisada

### 2. Cliente fino en web

Impacto:

- la web sigue leyendo y mutando estado contra `db` de forma directa
- esto dificulta cumplir la regla de “cliente fino sobre API/servicios expuestos”
- mezcla presentación y acceso a persistencia, reforzando el acoplamiento interno

Ejemplos representativos:

- `cmd/ops_web.go:85-104`
  la vista de sesiones resuelve proyecto y lista sesiones directamente con `db.GetProyecto` y `db.ListarSesionesInspeccion`
- `cmd/ops_web.go:118-124`
  el detalle de sesión llama directamente a `db.GetSesionInspeccionByID`
- `cmd/lenguaje_web.go:59-96`
  la vista `/lenguaje` consume `db.GetLanguagePolicy`, `db.ListLanguageMatrixEntries` y `db.ResolveLanguage`
- `cmd/gitgov_web.go:141-150`
  detalle GitGov usa directamente `db.SQLiteWorktreeRepository{}` y `db.SQLiteLockRepository{}`
- `cmd/agentes_web.go:174-205`
  el panel agrega agentes, asignaciones, sesiones, runtimes, handles, órdenes, mailbox y checkpoints llamando a `db.*` de forma directa
- `cmd/serve.go:371-807`
  el panel principal clásico sigue operando masivamente contra `db.*` para listar, crear y mutar tareas/propuestas/runtimes

Conclusión:

- la dirección del producto y varias tareas recientes van hacia cliente fino y single-writer, pero la capa web aún conserva mucha lógica de acceso directo

Estado actualizado:

- esta observación ya no aplica a tareas, propuestas, sesiones operativas, lenguaje, agentes ni al bloque principal de dashboard/runtimes/time-travel
- en la capa web revisada ya no quedan llamadas operativas `db.Función(...)`; solo sobreviven conversiones de enums y uso de tipos/filtros `db.*`
- la deuda fuerte inicial de “web contra db directo” queda cerrada en la superficie principal revisada

### 3. Superficies duplicadas en handlers “API web”

Impacto:

- añade ruido al código
- crea caminos alternativos difíciles de mantener
- los tests pueden cubrir funciones que no estén montadas en el wiring real

Candidatos claros:

- `cmd/ops_web.go:182-236`
  `webHandlerAPIAsignaciones`, `webHandlerAPISesiones`, `webHandlerAPIExportEstado`, `webHandlerAPIExportAudit`
- `cmd/gitgov_web.go:158-238`
  `webHandlerAPIWorktrees`, `webHandlerAPILocks`, `webHandlerAPIMerges`, `webWriteJSON`, `listGitMerges`

Conclusión:

- tras revisar el wiring real, esos wrappers no estaban montados
- se eliminaron junto con sus tests asociados, dejando una sola API real y una sola web real

## Deuda adicional observada

### 4. Mezcla inconsistente de servicios y acceso directo a `db`

Hay zonas de web que sí usan servicios intermedios:

- `cmd/proyectos_web.go` usa `memoriaproyecto.NewService(...)`
- `cmd/ops_web.go` usa parcialmente `operacionesapp.NewService(...)`
- `cmd/gobernanza_web.go` usa `gobernanzaapp.NewService(...)`

Pero esas mismas áreas vuelven a caer en lecturas directas a `db` en otros handlers.

Conclusión:

- la dirección por capas ya existe y durante esta revisión se ha extendido bastante
- lo que queda es residual y se concentra en composición, tipos compartidos y posibles mejoras futuras, no en la superficie principal

### 5. La superficie web crece más rápido que su disciplina de contrato

Síntomas:

- vistas nuevas añadidas con rapidez
- detalle operativo muy rico
- parte del contenido aún no está plenamente i18n-izado
- parte del acceso sigue yendo directo a persistencia en vez de a contratos estables

Conclusión:

- el producto está creciendo, pero la disciplina arquitectónica no está cerrando al mismo ritmo

## Limpieza / código candidato a simplificación

### Candidato A: wrappers API web redundantes

Estado: resuelto durante esta revisión

Acción aplicada:

- revisión del wiring real
- eliminación de wrappers no montados y de tests que solo cubrían esa superficie muerta

### Candidato B: unificar acceso web a través de servicios / read-models

Prioridad: alta

Archivos afectados:

- `cmd/gobernanza_web.go`
- `cmd/proyectos_web.go`
- helpers residuales de `cmd/serve.go`

Acción sugerida:

- seguir extrayendo servicios read-only y comandos de aplicación donde aún no existan
- dejar a la web como adaptador de presentación
- revisar si algunos módulos ya suficientemente encapsulados pueden quedarse como están sin nueva capa intermedia

### Candidato C: barrido de cadenas duras en plantillas web

Estado: prácticamente resuelto en la superficie revisada

Acción aplicada:

- sustitución de literales visibles por claves
- normalización de flashes `ok/err` detectados
- mantenimiento de resolución real de idioma por request

## Propuesta de orden de arreglo

### Bloque 1. I18n real en web

Estado: resuelto en la superficie principal revisada

### Bloque 2. Consolidación de cliente fino

Estado: resuelto en la superficie principal revisada

### Bloque 3. Limpieza de superficies duplicadas

Estado: resuelto en la capa “API web” no montada

## Recomendación práctica inmediata

Siguiente posible trabajo, ya no de saneamiento urgente:

1. decidir si merece la pena añadir más capas de aplicación en módulos que ya están razonablemente encapsulados
2. completar una validación global de `go test ./...` con más tiempo
3. convertir parte de estos guardarraíles en CI si todavía no lo están

Actualización final:

- la validación global `go test ./... -count=1 -timeout=600s` ya cerró en verde
- el saneamiento de esta revisión queda completo en código, tests e informe
