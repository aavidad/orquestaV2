# Pruebas locales: orquesta-deploy

Registra pruebas obligatorias del modulo.

## DeploymentPlan v0 - contenedor valido

Caso: `DeploymentPlan v0` con `deploy_target = contenedor`
Tipo: `contract`
Comando: `npx --yes ajv-cli@5.0.0 validate -s docs/schemas/deployment_plan_v0.schema.json -d docs/fixtures/deployment_plan_v0/contenedor_valido.json --spec=draft7 --all-errors`
Evidencia esperada:

- El plan contiene `target_resuelto = contenedor`.
- La matriz OS declara `linux`, `darwin`, `windows` con aplicabilidad o motivo.
- Incluye `validacion_entorno`, `healthcheck`, `rollback` y `artefactos_previstos`.
- No contiene secretos, proveedor cloud hardcodeado, runtime/agente, DB ni scripts ejecutables.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- El schema reduce riesgos de forma contractual, pero no sustituye una auditoria de seguridad.

## DeploymentPlan v0 - local valido

Caso: `DeploymentPlan v0` con `deploy_target = local`
Tipo: `smoke`
Comando: `npx --yes ajv-cli@5.0.0 validate -s docs/schemas/deployment_plan_v0.schema.json -d docs/fixtures/deployment_plan_v0/local_valido.json --spec=draft7 --all-errors`
Evidencia esperada:

- El plan declara preparacion local por OS aplicable: `linux`, `darwin`, `windows`.
- Docker/contenedor no aparece como requisito salvo `aceptacion_contenedor = true`.
- Incluye validacion de entorno previa y healthcheck posterior.
- Incluye rollback o motivo explicito si alguna accion no es reversible.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- La semantica exacta por OS seguira necesitando pruebas de adaptador cuando exista implementacion.

## DeploymentPlan v0 - target no soportado

Caso: `DeploymentPlan v0` con target desconocido
Tipo: `contract`
Comando: `npx --yes ajv-cli@5.0.0 validate -s docs/schemas/deployment_plan_v0.schema.json -d docs/fixtures/deployment_plan_v0/target_no_soportado_invalido.json --spec=draft7 --all-errors`
Evidencia esperada:

- Falla por `/deploy_target` al usar un valor fuera de targets v0; el consumidor debe mapearlo a `deploy_target_no_soportado`.
- No genera `acciones_previstas` ejecutables.
- No propone Docker, proveedor cloud, runtime, agente, DB ni artefactos reales.

Ultima ejecucion: `2026-05-04: fallo esperado por enum de /deploy_target`
Riesgos:

- El schema detecta el target desconocido; el mapeo a error publico queda para el adaptador futuro.

## DeploymentPlan v0 - sintaxis JSON

Caso: Schema y fixtures tienen JSON valido
Tipo: `smoke`
Comando: `jq empty docs/schemas/*.schema.json docs/fixtures/deployment_plan_v0/*.json`
Evidencia esperada:

- `jq` no informa errores de parseo.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- Solo valida sintaxis JSON, no semantica.

## DeploymentPlan v0 - harness Go relacional

Caso: Decode/validate puro de `DeploymentPlanV0`
Tipo: `contract`
Comando: `go test -count=1 ./modulos/orquesta-deploy`
Evidencia esperada:

- `contenedor_valido.json` y `local_valido.json` se decodifican y validan como OK.
- Los casos inline rechazan matrices OS incompletas, target resuelto incompatible, contenedor no aceptado, acciones no declarativas y secciones obligatorias vacias.
- No usa DB, filesystem productivo, contenedor, proveedor, scripts ni adaptadores reales.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- El harness cubre invariantes relacionales de contrato, pero no sustituye validacion JSON Schema ni pruebas de adaptadores futuros.

## DEP-006 - LocalDeployAdapter v0 dry-run

Caso: Preparacion local pura desde `DeploymentPlanV0`
Tipo: `unit`
Comando: `go test -count=1 .`
Evidencia esperada:

- `local_valido.json` se transforma en `LocalDeployResultV0` con `adapter = LocalDeployAdapterV0`, `mode = dry_run` y `status = preparado`.
- El resultado conserva `linux`, `darwin`, `windows`, validacion de entorno, artefactos previstos, healthcheck, rollback, acciones previstas y advertencias sin aliasar slices del plan.
- El resultado es serializable a JSON y no materializa artefactos ni toca filesystem productivo.
- Se rechazan target no local, matriz OS no aplicable, rollback no reversible y acciones que requieren contenedor.
- Los planes base invalidos siguen devolviendo errores de `DeploymentPlan v0`.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- El adaptador es dry-run puro; no valida dependencias reales del sistema ni prueba un despliegue local efectivo.
- La politica por OS para acciones no reversibles sigue bloqueada porque `DeploymentPlanV0` solo expresa rollback global.

## DEP-007 - ContainerDeployAdapter v0 dry-run

Caso: Preparacion declarativa de contenedor desde `DeploymentPlanV0`
Tipo: `unit`
Comando: `go test -count=1 .`
Evidencia esperada:

- `contenedor_valido.json` se transforma en `ContainerDeployResultV0` con `adapter = ContainerDeployAdapterV0`, `mode = dry_run` y `status = preparado`.
- El resultado conserva `linux`, `darwin`, `windows`, validacion de entorno, artefactos previstos, healthcheck, rollback, acciones previstas y advertencias sin aliasar slices del plan.
- El resultado mantiene al menos una accion prevista con `requiere_contenedor = true`.
- El resultado es serializable a JSON y no construye imagenes, no publica artefactos y no toca filesystem productivo.
- Se rechazan target no contenedor, rollback no reversible y resultados con target resuelto incorrecto.
- Los casos base de contenedor no aceptado o sin accion de contenedor siguen devolviendo errores de `DeploymentPlan v0`.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- El adaptador es dry-run puro; no valida daemon, runtime, socket, permisos ni registry reales.
- La deteccion de runtime real y la construccion de imagenes siguen bloqueadas para microtareas posteriores con contrato y smoke propios.

## DEP-008 - KubernetesDeployAdapter v0 dry-run

Caso: Preparacion declarativa de kubernetes desde `DeploymentPlanV0`
Tipo: `unit`
Comando: `go test -count=1 .`
Evidencia esperada:

- `kubernetes_valido.json` se transforma en `KubernetesDeployResultV0` con `adapter = KubernetesDeployAdapterV0`, `mode = dry_run` y `status = preparado`.
- El resultado conserva `linux`, `darwin`, `windows`, validacion de entorno, artefactos previstos, healthcheck, rollback, acciones previstas y advertencias sin aliasar slices del plan.
- La matriz distingue al menos dos filas de herramienta cliente para `darwin` y `windows` mediante motivo declarativo.
- El resultado mantiene al menos una accion `publicar_declarativo`.
- El resultado es serializable a JSON y no escribe manifests, charts, overlays, kubeconfig ni toca cluster o filesystem productivo.
- Se rechazan target no kubernetes, ausencia de `publicar_declarativo`, rollback no reversible y matriz sin filas cliente.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- El adaptador es dry-run puro; no valida kubeconfig, contexto, cluster, namespace, CRDs ni providers reales.
- La escritura de manifests, Helm/Kustomize y la aplicacion real de recursos siguen bloqueadas para microtareas posteriores.

## DEP-009 - familias PaaS y serverless

Caso: Contrato documental de familias genericas de deploy gestionado
Tipo: `contract`
Comando: revision documental de `docs/contratos.md`, `docs/pruebas.md`, `docs/tareas.md` y `docs/decisiones.md`
Evidencia esperada:

- `PaaSDeployAdapter v0` y `ServerlessDeployAdapter v0` quedan fijados como familias genericas, no como proveedores concretos.
- Se documentan invariantes de dry-run futuro, errores publicos previstos y prohibiciones de credenciales, SDKs y runtime real.
- No se crea codigo, IaC, archivos de plataforma, naming de servicios ni dependencias de proveedor.

Ultima ejecucion: `2026-05-04: revision documental OK`
Riesgos:

- La familia queda util para backlog y contratos, pero no valida todavia un provider fake ni un DTO ejecutable.

## DEP-011 - PaaSDeployAdapter v0 dry-run

Caso: Preparacion declarativa PaaS desde `DeploymentPlanV0`
Tipo: `unit`
Comando: `go test -count=1 .`
Evidencia esperada:

- `paas_valido.json` se transforma en `PaaSDeployResultV0` con `adapter = PaaSDeployAdapterV0`, `mode = dry_run` y `status = preparado`.
- El resultado conserva `linux`, `darwin`, `windows`, validacion de entorno, artefactos previstos, healthcheck, rollback, acciones previstas y advertencias sin aliasar slices del plan.
- La matriz mantiene filas cliente para `darwin` y `windows`.
- El plan declara proveedor como restriccion opaca o pendiente, sin proveedor real.
- El resultado mantiene al menos una accion `publicar_declarativo`.
- El resultado es serializable a JSON y no usa SDK, no escribe IaC, no provisiona servicios y no toca filesystem productivo.
- Se rechazan target no paas, matriz sin filas cliente, proveedor no declarado, ausencia de `publicar_declarativo`, rollback no reversible y acciones que requieren contenedor.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- El adaptador es dry-run puro; no valida proveedor, runtime gestionado, region, cuenta, SDK ni plataforma real.
- Elegir proveedor, formato IaC o despliegue real sigue bloqueado para microtareas posteriores.

## DEP-010 - DesktopDeployAdapter v0 dry-run

Caso: Preparacion declarativa de desktop desde `DeploymentPlanV0`
Tipo: `unit`
Comando: `go test -count=1 .`
Evidencia esperada:

- `desktop_valido.json` se transforma en `DesktopDeployResultV0` con `adapter = DesktopDeployAdapterV0`, `mode = dry_run` y `status = preparado`.
- El resultado conserva `linux`, `darwin`, `windows`, validacion de entorno, artefactos previstos, healthcheck, rollback, acciones previstas y advertencias sin aliasar slices del plan.
- El resultado mantiene al menos un artefacto previsto con `tipo = paquete_desktop`.
- El resultado es serializable a JSON y no empaqueta binarios, no firma y no toca filesystem productivo.
- Se rechazan target no desktop, OS no aplicable, rollback no reversible, ausencia de `paquete_desktop` y acciones que requieren contenedor.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- El adaptador es dry-run puro; no valida toolchains reales, instaladores, formatos nativos ni distribucion desktop efectiva.
- Firmado, notarizacion y empaquetado real siguen bloqueados para microtareas posteriores.

## DEP-010 - MobileStoreDeployAdapter v0 dry-run

Caso: Preparacion declarativa de mobile_store desde `DeploymentPlanV0`
Tipo: `unit`
Comando: `go test -count=1 .`
Evidencia esperada:

- `mobile_store_valido.json` se transforma en `MobileStoreDeployResultV0` con `adapter = MobileStoreDeployAdapterV0`, `mode = dry_run` y `status = preparado`.
- El resultado conserva `linux`, `darwin`, `windows`, validacion de entorno, artefactos previstos, healthcheck, rollback, acciones previstas y advertencias sin aliasar slices del plan.
- La matriz mantiene `darwin` aplicable y `linux`/`windows` como filas cliente para revision.
- El resultado mantiene al menos un artefacto `metadata_publicacion` y una accion `solicitar_decision`.
- El resultado es serializable a JSON y no firma, no notariza, no reserva nombres y no toca tienda o filesystem productivo.
- Se rechazan target no mobile_store, matriz invalida, falta de metadata, ausencia de `solicitar_decision`, rollback no reversible y acciones que requieren contenedor.

Ultima ejecucion: `2026-05-04: OK`
Riesgos:

- El adaptador es dry-run puro; no valida formatos de tienda, credenciales, nombres reservados ni publicacion efectiva.
- Firmado, notarizacion y subida real siguen bloqueados para microtareas posteriores.

## DEP-012 - self-programming remoto aislado

Caso: Contrato estatico del perfil `deploy/self-programming`
Tipo: `contract`
Comando: `go test -count=1 ./deploy/self-programming`
Evidencia esperada:

- `docker-compose.yml` publica solo `127.0.0.1:19039:19039`.
- Los binds del host salen solo de `/srv/orquesta-self` y no montan Docker socket, OPES productivo, `uso-app` ni rutas de temarios.
- El contenedor conserva usuario `10001:10001`, `read_only: true`, `no-new-privileges:true` y `cap_drop: ALL`.
- `orquesta-self.env.example` mantiene desactivados promocion, OPES/DomainWork productivos y fallbacks `stdio`/proxy; `app_server_tmux` sigue siendo obligatorio.
- El runbook documenta `sudo` solo para preparacion/operacion host del contenedor aislado y conserva el check de `docker inspect`.

Ultima ejecucion: `2026-07-02: OK`
Riesgos:

- Es una prueba estatica; no arranca Docker remoto ni sustituye `docker inspect` tras desplegar.

## DEP-003 - separacion de pruebas para adaptadores futuros

Caso: Backlog de adaptadores futuros de deploy
Tipo: `contract`
Comando: revision documental de `docs/tareas.md`, `docs/pruebas.md` y `docs/decisiones.md`
Evidencia esperada:

- Las pruebas de contrato/documentales permanecen como validaciones por defecto: schema, fixtures, harness puro y `git diff --check -- .`.
- Las unitarias futuras usan codigo puro, fakes o memoria; no invocan Docker, Kubernetes, cloud, tiendas, filesystem productivo ni scripts.
- Las smoke futuras son dry-run, usan directorios temporales, fake clients o entornos efimeros autorizados, y no requieren secretos.
- Las integraciones reales futuras quedan separadas, opt-in y saltadas por defecto; deben declarar contrato, write-set, entorno, limpieza y evidencia sin secretos.
- Ningun adaptador futuro puede crear Dockerfile, compose, manifiestos, pipelines, scripts shell ni IaC antes de tener contrato y smoke especificos.

Ultima ejecucion: `2026-05-04: revision documental OK`
Riesgos:

- El backlog no ejecuta integraciones reales; solo fija la puerta minima para implementarlas despues.

## DEP-003 - smoke documental

Caso: Backlog documental de conectores hexagonales futuros
Tipo: `smoke`
Comando: `git diff --check -- .`
Evidencia esperada:

- El diff no tiene errores de whitespace.
- Solo cambia documentacion local dentro del write-set de DEP-003: `docs/tareas.md`, `docs/pruebas.md` y `docs/decisiones.md`.
- El backlog cubre conectores `local`, `contenedor`, `kubernetes`, `paas`, `serverless`, `desktop` y `mobile_store`.
- Cada conector futuro declara contrato futuro, write-set, pruebas, bloqueos y prohibiciones.
- No se crean Dockerfile, compose, scripts, manifiestos, pipelines ni IaC.

Ultima ejecucion: `2026-05-04: git diff --check -- . OK`
Riesgos:

- `git diff --check -- .` valida higiene del diff; la semantica se revisa por contrato documental.

## Documentacion DEP-000

Caso: Corte documental sin scripts reales
Tipo: `smoke`
Comando: `git diff --check`
Evidencia esperada:

- El diff no tiene errores de whitespace.
- Solo cambia el write-set permitido para DEP-000.

Ultima ejecucion: `2026-05-04: git diff --check OK`
Riesgos:

- No valida semantica de contrato; solo higiene del diff.

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```
