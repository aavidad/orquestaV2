# Contratos locales: orquesta-deploy

Registra puertos, DTOs y eventos que `orquesta-deploy` expone o consume.

## DeploymentPlan v0

Nombre: `DeploymentPlan`
Tipo: `puerto_salida` / servicio de deploy
Version: `v0`
Propietario: `orquesta-deploy`
Estado compartido: `v0 compartido` en `../../CONTRATOS.md`; este archivo mantiene el detalle canonico local.
Consumidores:

- `orquesta-core`, como puerto para preparar entorno/deploy a partir de una app validada.

Schema canonico:

- `docs/schemas/deployment_plan_v0.schema.json`

Fixtures canonicos:

- `docs/fixtures/deployment_plan_v0/contenedor_valido.json`
- `docs/fixtures/deployment_plan_v0/local_valido.json`
- `docs/fixtures/deployment_plan_v0/paas_valido.json`
- `docs/fixtures/deployment_plan_v0/target_no_soportado_invalido.json`

Campos de entrada:

- `request_id`: identificador opaco de correlacion.
- `app_spec_id`: identificador opaco de la `AppSpecV0` validada.
- `app_spec_version`: debe ser `AppSpecV0`.
- `deploy_target`: valor de `AppSpecV0.Deploy.Target`.
- `sistemas_operativos`: matriz explicita de OS aplicables por target. Valores permitidos: `linux`, `darwin`, `windows`.
- `aceptacion_contenedor`: `true`, `false` o `no_aplica`; solo habilita Docker/contenedor cuando el target lo pide o el usuario lo acepta.
- `restricciones`: limites declarados por el consumidor, sin secretos ni proveedor hardcodeado.

Campos de salida:

- `plan_id`: identificador opaco del plan.
- `target_resuelto`: target normalizado.
- `matriz_os`: lista explicita de filas `{os, aplica, motivo, requisitos_previos}`.
- `validacion_entorno`: comprobaciones previstas antes de ejecutar deploy.
- `artefactos_previstos`: artefactos a producir o consumir por el deploy, sin rutas privadas ni secretos.
- `healthcheck`: comprobaciones previstas para considerar sano el despliegue.
- `rollback`: pasos previstos para volver al estado anterior o marcar reversibilidad no disponible con motivo.
- `acciones_previstas`: pasos declarativos ordenados; en v0 no son scripts ejecutables.
- `advertencias`: supuestos, degradaciones o decisiones pendientes.

Targets soportados v0:

- `sin_preferencia`: no elige proveedor; devuelve alternativas compatibles y preguntas si falta informacion.
- `local`: prepara plan para ejecucion local por OS aplicable.
- `contenedor`: prepara plan declarativo para contenedor; requiere target explicito o aceptacion de usuario.
- `paas`: prepara plan generico sin proveedor cloud hardcodeado.
- `serverless`: prepara plan generico sin proveedor cloud hardcodeado.
- `kubernetes`: prepara plan declarativo para cluster, sin asumir proveedor.
- `desktop`: prepara plan por OS de escritorio aplicable.
- `mobile_store`: prepara plan de publicacion en tienda sin credenciales ni integracion real.

Matriz multi-OS minima v0:

| Target | linux | darwin | windows |
| --- | --- | --- | --- |
| `sin_preferencia` | aplica si el consumidor lo permite | aplica si el consumidor lo permite | aplica si el consumidor lo permite |
| `local` | aplica | aplica | aplica |
| `contenedor` | aplica | aplica para build/desarrollo | aplica para build/desarrollo |
| `paas` | aplica si el proveedor futuro lo soporta | no_aplica salvo herramienta cliente | no_aplica salvo herramienta cliente |
| `serverless` | aplica si el proveedor futuro lo soporta | no_aplica salvo herramienta cliente | no_aplica salvo herramienta cliente |
| `kubernetes` | aplica | aplica para herramienta cliente | aplica para herramienta cliente |
| `desktop` | aplica | aplica | aplica |
| `mobile_store` | no_aplica salvo herramienta cliente | aplica para ecosistemas que lo requieran | no_aplica salvo herramienta cliente |

Invariantes:

- `DeploymentPlan v0` prepara entorno segun `AppSpecV0.Deploy.Target`; no decide arquitectura de producto.
- Docker/contenedor solo aparece como requisito cuando `deploy_target = contenedor` o `aceptacion_contenedor = true`.
- Todo plan incluye `validacion_entorno`, `healthcheck`, `rollback` y `artefactos_previstos`.
- La matriz OS debe ser explicita; no se aceptan promesas genericas de multi-OS.
- No contiene secretos, tokens, rutas HOME reales ni credenciales.
- No hardcodea proveedor cloud, tienda, DB, runtime, agente ni modelo.
- No crea scripts, manifiestos ni pipelines ejecutables en v0; solo contrato y backlog.
- Cualquier adaptador real de filesystem, Docker, cloud, CI/CD o tienda debe tener contrato y prueba de smoke antes de existir.

Errores:

- `deploy_target_requerido`
- `deploy_target_no_soportado`
- `matriz_os_incompleta`
- `contenedor_no_aceptado`
- `validacion_entorno_incompleta`
- `healthcheck_incompleto`
- `rollback_incompleto`
- `artefactos_previstos_incompletos`
- `secreto_detectado`
- `proveedor_hardcodeado`
- `script_sin_contrato`

Pruebas de contrato:

- Caso contrato: `contenedor_valido.json` valida contra `deployment_plan_v0.schema.json` y produce matriz OS explicita, validacion de entorno, healthcheck, rollback y artefactos previstos sin scripts reales.
- Caso smoke documental: `local_valido.json` valida contra `deployment_plan_v0.schema.json`, cubre `linux`, `darwin`, `windows` y no requiere contenedor cuando `aceptacion_contenedor = false`.
- Caso contrato negativo: `target_no_soportado_invalido.json` falla contra `deployment_plan_v0.schema.json` por `deploy_target` desconocido y no produce acciones ejecutables.

## LocalDeployAdapter v0

Nombre: `LocalDeployAdapter`
Tipo: conector hexagonal local de salida
Version: `v0`
Propietario: `orquesta-deploy`
Estado compartido: local; no se promueve a `../../CONTRATOS.md` porque no cambia el contrato entre modulos.
Consumidores:

- `orquesta-deploy`, como adaptador interno para preparar dry-run local desde un `DeploymentPlanV0` ya validado.

Entrada:

- `DeploymentPlanV0` con `schema_version = DeploymentPlanV0`.
- `deploy_target = local` y `target_resuelto = local`.
- `sistemas_operativos` y `matriz_os` con `linux`, `darwin`, `windows`.
- Cada fila de `matriz_os` debe aplicar con motivo y requisitos previos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback` y `acciones_previstas` completos.
- `rollback.reversible = true`; v0 no inventa politica por OS para acciones locales no reversibles.
- Ninguna accion prevista ni paso de rollback puede requerir contenedor.

Salida:

- `LocalDeployResultV0`, serializable, con `schema_version = LocalDeployResultV0`.
- `adapter = LocalDeployAdapterV0`.
- `mode = dry_run`.
- `status = preparado`.
- `request_id`, `plan_id` y `target_resuelto` copiados del plan.
- `os_preparados`: filas `{os, status, motivo, requisitos_previos}` derivadas de `matriz_os`.
- `evidencias`: comprobaciones compactas de contrato validado, target local, dry-run sin efectos y rollback reversible.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback`, `acciones_previstas` y `advertencias` copiadas del plan, sin materializar artefactos.

Invariantes:

- Es puro: no lee ni escribe filesystem productivo, no modifica HOME, PATH, servicios del sistema ni gestores globales.
- Es solo dry-run: no crea scripts, Dockerfile, compose, manifiestos, pipelines, IaC ni artefactos ejecutables.
- No consulta entorno real, no instala dependencias, no ejecuta comandos de usuario y no selecciona proveedor, runtime, DB, agente ni modelo.
- No acepta contenedor aunque `aceptacion_contenedor = true`; ese caso corresponde a otro conector.
- Si un plan local necesita acciones no reversibles por OS, queda bloqueado hasta versionar politica explicita en contrato.

Errores:

- Hereda errores de `DeploymentPlan v0` cuando el plan base es invalido.
- `local_deploy_target_no_local`
- `local_deploy_resultado_invalido`
- `local_deploy_rollback_no_reversible`
- `local_deploy_contenedor_no_puro`

Pruebas de contrato:

- Unit: `go test -count=1 .` prepara `LocalDeployResultV0` desde `local_valido.json`.
- Unit: rechaza target no local, matriz OS no aplicable, rollback no reversible y acciones que requieren contenedor.
- Smoke: `git diff --check -- .` confirma higiene del diff sin crear artefactos de deploy.

## ContainerDeployAdapter v0

Nombre: `ContainerDeployAdapter`
Tipo: conector hexagonal de salida para target `contenedor`
Version: `v0`
Propietario: `orquesta-deploy`
Estado compartido: local; no se promueve a `../../CONTRATOS.md` porque no cambia el contrato entre modulos.
Consumidores:

- `orquesta-deploy`, como adaptador interno para preparar dry-run declarativo de contenedor desde un `DeploymentPlanV0` ya validado.

Entrada:

- `DeploymentPlanV0` con `schema_version = DeploymentPlanV0`.
- `deploy_target = contenedor` y `target_resuelto = contenedor`.
- `sistemas_operativos` y `matriz_os` con `linux`, `darwin`, `windows`.
- `aceptacion_contenedor` distinta de `false`; el target explicito o la aceptacion habilitan el conector.
- Cada fila de `matriz_os` debe tener motivo y requisitos previos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback` y `acciones_previstas` completos.
- `rollback.reversible = true`.
- Debe existir al menos una accion prevista con `requiere_contenedor = true`.

Salida:

- `ContainerDeployResultV0`, serializable, con `schema_version = ContainerDeployResultV0`.
- `adapter = ContainerDeployAdapterV0`.
- `mode = dry_run`.
- `status = preparado`.
- `request_id`, `plan_id` y `target_resuelto` copiados del plan.
- `os_preparados`: filas `{os, status, motivo, requisitos_previos}` derivadas de `matriz_os`.
- `evidencias`: comprobaciones compactas de contrato validado, target contenedor, presencia de accion que requiere contenedor y dry-run sin efectos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback`, `acciones_previstas` y `advertencias` copiadas del plan, sin construir imagenes ni publicar artefactos.

Invariantes:

- Es puro: no lee ni escribe filesystem productivo, no modifica HOME, PATH, servicios del sistema ni gestores globales.
- Es solo dry-run: no crea Dockerfile, compose, imagenes, manifests, pipelines, IaC ni artefactos ejecutables.
- No consulta runtime real de contenedor, no ejecuta comandos de usuario, no usa sockets, no instala dependencias y no contacta registries.
- No selecciona proveedor, runtime, DB, agente ni modelo.
- Si un plan de contenedor no es reversible o no declara acciones de contenedor, queda bloqueado hasta versionar politica posterior.

Errores:

- Hereda errores de `DeploymentPlan v0` cuando el plan base es invalido.
- `container_deploy_target_invalido`
- `container_deploy_resultado_invalido`
- `container_deploy_contenedor_no_aceptado`
- `container_deploy_sin_accion_contenedor`
- `container_deploy_rollback_no_reversible`

Pruebas de contrato:

- Unit: `go test -count=1 .` prepara `ContainerDeployResultV0` desde `contenedor_valido.json`.
- Unit: rechaza target no contenedor, rollback no reversible y resultados con target resuelto incorrecto.
- Unit: los casos de aceptacion explicita y accion de contenedor ausente siguen fallando en el contrato base `DeploymentPlan v0`.
- Smoke: `git diff --check -- .` confirma higiene del diff sin crear artefactos de deploy.

## KubernetesDeployAdapter v0

Nombre: `KubernetesDeployAdapter`
Tipo: conector hexagonal de salida para target `kubernetes`
Version: `v0`
Propietario: `orquesta-deploy`
Estado compartido: local; no se promueve a `../../CONTRATOS.md` porque no cambia el contrato entre modulos.
Consumidores:

- `orquesta-deploy`, como adaptador interno para preparar dry-run declarativo de Kubernetes desde un `DeploymentPlanV0` ya validado.

Entrada:

- `DeploymentPlanV0` con `schema_version = DeploymentPlanV0`.
- `deploy_target = kubernetes` y `target_resuelto = kubernetes`.
- `sistemas_operativos` y `matriz_os` con `linux`, `darwin`, `windows`.
- `matriz_os` debe distinguir al menos un entorno de ejecucion y dos filas de herramienta cliente para revision/preparacion.
- Cada fila de `matriz_os` debe tener motivo y requisitos previos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback` y `acciones_previstas` completos.
- `rollback.reversible = true`.
- Debe existir al menos una accion prevista con `tipo = publicar_declarativo`.

Salida:

- `KubernetesDeployResultV0`, serializable, con `schema_version = KubernetesDeployResultV0`.
- `adapter = KubernetesDeployAdapterV0`.
- `mode = dry_run`.
- `status = preparado`.
- `request_id`, `plan_id` y `target_resuelto` copiados del plan.
- `os_preparados`: filas `{os, status, motivo, requisitos_previos}` derivadas de `matriz_os`.
- `evidencias`: comprobaciones compactas de contrato validado, target kubernetes, publicacion declarativa y dry-run sin cluster.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback`, `acciones_previstas` y `advertencias` copiadas del plan, sin escribir manifests ni charts.

Invariantes:

- Es puro: no lee kubeconfig real, no resuelve contexto actual, no consulta cluster, no usa SDK/provider ni toca filesystem productivo.
- Es solo dry-run: no crea manifests, charts, overlays, Helm values, pipelines, IaC ni artefactos ejecutables.
- No ejecuta comandos de usuario, no selecciona proveedor, namespace, cuenta cloud, DB, agente ni modelo.
- La matriz multi-OS distingue estacion cliente frente a entorno de ejecucion sin inferir herramientas concretas.
- Si el plan no conserva rollback reversible o no declara publicacion declarativa, queda bloqueado en v0.

Errores:

- Hereda errores de `DeploymentPlan v0` cuando el plan base es invalido.
- `kubernetes_deploy_target_invalido`
- `kubernetes_deploy_resultado_invalido`
- `kubernetes_deploy_rollback_no_reversible`
- `kubernetes_deploy_sin_publicacion`
- `kubernetes_deploy_matriz_cliente_invalida`

Pruebas de contrato:

- Unit: `go test -count=1 .` prepara `KubernetesDeployResultV0` desde `kubernetes_valido.json`.
- Unit: rechaza target no kubernetes, ausencia de `publicar_declarativo`, rollback no reversible y matriz sin filas cliente.
- Smoke: `git diff --check -- .` confirma higiene del diff sin crear artefactos de deploy.

## DesktopDeployAdapter v0

Nombre: `DesktopDeployAdapter`
Tipo: conector hexagonal de salida para target `desktop`
Version: `v0`
Propietario: `orquesta-deploy`
Estado compartido: local; no se promueve a `../../CONTRATOS.md` porque no cambia el contrato entre modulos.
Consumidores:

- `orquesta-deploy`, como adaptador interno para preparar dry-run declarativo de empaquetado desktop desde un `DeploymentPlanV0` ya validado.

Entrada:

- `DeploymentPlanV0` con `schema_version = DeploymentPlanV0`.
- `deploy_target = desktop` y `target_resuelto = desktop`.
- `sistemas_operativos` y `matriz_os` con `linux`, `darwin`, `windows`.
- Cada fila de `matriz_os` debe aplicar, con motivo y requisitos previos explicitos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback` y `acciones_previstas` completos.
- `rollback.reversible = true`.
- Debe existir al menos un artefacto previsto con `tipo = paquete_desktop`.
- Ninguna accion prevista ni paso de rollback puede requerir contenedor.

Salida:

- `DesktopDeployResultV0`, serializable, con `schema_version = DesktopDeployResultV0`.
- `adapter = DesktopDeployAdapterV0`.
- `mode = dry_run`.
- `status = preparado`.
- `request_id`, `plan_id` y `target_resuelto` copiados del plan.
- `os_preparados`: filas `{os, status, motivo, requisitos_previos}` derivadas de `matriz_os`.
- `evidencias`: comprobaciones compactas de contrato validado, target desktop, artefacto desktop previsto y dry-run sin efectos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback`, `acciones_previstas` y `advertencias` copiadas del plan, sin empaquetar binarios reales.

Invariantes:

- Es puro: no lee ni escribe filesystem productivo, no modifica HOME, PATH ni instaladores del sistema.
- Es solo dry-run: no crea instaladores, bundles, firmados, notarizaciones, pipelines ni artefactos ejecutables.
- No ejecuta comandos de usuario, no usa contenedor, no selecciona proveedor, runtime, DB, agente ni modelo.
- El target desktop exige matriz aplicable para `linux`, `darwin` y `windows`; no admite promesas vagas de multiplataforma.
- Si falta paquete desktop o rollback reversible, el plan queda bloqueado en v0.

Errores:

- Hereda errores de `DeploymentPlan v0` cuando el plan base es invalido.
- `desktop_deploy_target_invalido`
- `desktop_deploy_resultado_invalido`
- `desktop_deploy_rollback_no_reversible`
- `desktop_deploy_sin_paquete`
- `desktop_deploy_contenedor_no_puro`

Pruebas de contrato:

- Unit: `go test -count=1 .` prepara `DesktopDeployResultV0` desde `desktop_valido.json`.
- Unit: rechaza target no desktop, OS no aplicable, rollback no reversible, ausencia de `paquete_desktop` y acciones con contenedor.
- Smoke: `git diff --check -- .` confirma higiene del diff sin crear artefactos de deploy.

## MobileStoreDeployAdapter v0

Nombre: `MobileStoreDeployAdapter`
Tipo: conector hexagonal de salida para target `mobile_store`
Version: `v0`
Propietario: `orquesta-deploy`
Estado compartido: local; no se promueve a `../../CONTRATOS.md` porque no cambia el contrato entre modulos.
Consumidores:

- `orquesta-deploy`, como adaptador interno para preparar dry-run declarativo de publicacion mobile desde un `DeploymentPlanV0` ya validado.

Entrada:

- `DeploymentPlanV0` con `schema_version = DeploymentPlanV0`.
- `deploy_target = mobile_store` y `target_resuelto = mobile_store`.
- `sistemas_operativos` y `matriz_os` con `linux`, `darwin`, `windows`.
- `matriz_os` debe conservar `darwin` como fila aplicable y `linux`/`windows` como filas cliente o de revision.
- Cada fila de `matriz_os` debe tener motivo y requisitos previos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback` y `acciones_previstas` completos.
- `rollback.reversible = true`.
- Debe existir al menos un artefacto previsto con `tipo = metadata_publicacion`.
- Debe existir al menos una accion prevista con `tipo = solicitar_decision`.
- Ninguna accion prevista ni paso de rollback puede requerir contenedor.

Salida:

- `MobileStoreDeployResultV0`, serializable, con `schema_version = MobileStoreDeployResultV0`.
- `adapter = MobileStoreDeployAdapterV0`.
- `mode = dry_run`.
- `status = preparado`.
- `request_id`, `plan_id` y `target_resuelto` copiados del plan.
- `os_preparados`: filas `{os, status, motivo, requisitos_previos}` derivadas de `matriz_os`.
- `evidencias`: comprobaciones compactas de contrato validado, target mobile_store, metadata de publicacion y dry-run sin efectos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback`, `acciones_previstas` y `advertencias` copiadas del plan, sin firmar ni publicar.

Invariantes:

- Es puro: no lee credenciales, no toca filesystem productivo y no consulta tiendas reales.
- Es solo dry-run: no firma, no notariza, no reserva nombres, no sube paquetes y no crea pipelines de publicacion.
- No ejecuta comandos de usuario, no usa contenedor, no selecciona tienda, proveedor, cuenta, DB, agente ni modelo.
- El target mobile_store exige metadata de publicacion y escalado manual mediante `solicitar_decision` antes de cualquier integracion real.
- Si la matriz no distingue `darwin` aplicable y filas cliente para `linux`/`windows`, el plan queda bloqueado en v0.

Errores:

- Hereda errores de `DeploymentPlan v0` cuando el plan base es invalido.
- `mobile_store_deploy_target_invalido`
- `mobile_store_deploy_resultado_invalido`
- `mobile_store_deploy_rollback_no_reversible`
- `mobile_store_deploy_matriz_invalida`
- `mobile_store_deploy_sin_metadata`
- `mobile_store_deploy_sin_decision`
- `mobile_store_deploy_contenedor_no_puro`

Pruebas de contrato:

- Unit: `go test -count=1 .` prepara `MobileStoreDeployResultV0` desde `mobile_store_valido.json`.
- Unit: rechaza target no mobile_store, matriz sin filas cliente, falta de `metadata_publicacion`, ausencia de `solicitar_decision`, rollback no reversible y acciones con contenedor.
- Smoke: `git diff --check -- .` confirma higiene del diff sin crear artefactos de deploy.

## PaaSDeployAdapter v0

Nombre: `PaaSDeployAdapter`
Tipo: conector hexagonal de salida para familia `paas`
Version: `v0`
Propietario: `orquesta-deploy`
Estado compartido: local; no se promueve a `../../CONTRATOS.md` porque no cambia el contrato entre modulos.
Consumidores:

- `orquesta-deploy`, como adaptador interno para preparar dry-run declarativo PaaS desde un `DeploymentPlanV0` ya validado.

Entrada:

- `DeploymentPlanV0` con `schema_version = DeploymentPlanV0`.
- `deploy_target = paas` y `target_resuelto = paas`.
- `sistemas_operativos` y `matriz_os` con `linux`, `darwin`, `windows`.
- `matriz_os` debe distinguir `linux` como entorno de ejecucion generico y `darwin`/`windows` como filas de herramienta cliente o revision.
- Restricciones opacas para proveedor, cuenta, region y runtime; no valores reales.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback` y `acciones_previstas` completos.
- `rollback.reversible = true`.
- Debe existir al menos una accion prevista con `tipo = publicar_declarativo`.
- Ninguna accion prevista ni paso de rollback puede requerir contenedor.

Salida:

- `PaaSDeployResultV0`, serializable, con `schema_version = PaaSDeployResultV0`.
- `adapter = PaaSDeployAdapterV0`.
- `mode = dry_run`.
- `status = preparado`.
- `request_id`, `plan_id` y `target_resuelto` copiados del plan.
- `os_preparados`: filas `{os, status, motivo, requisitos_previos}` derivadas de `matriz_os`.
- `evidencias`: comprobaciones compactas de contrato validado, target paas, proveedor opaco y dry-run sin efectos.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback`, `acciones_previstas` y `advertencias` copiadas del plan, sin materializar artefactos.

Invariantes:

- V0 de familia, no de proveedor: no nombra servicios concretos ni SDKs.
- Es puro: no lee credenciales, no toca filesystem productivo, no consulta SDK, entorno real ni servicio gestionado.
- Es solo dry-run declarativo: sin credenciales reales, sin provisionar y sin publicar.
- El proveedor, runtime gestionado, naming y politicas de cuenta quedan fuera hasta nueva microtarea.
- Si falta proveedor opaco, publicacion declarativa o rollback reversible, el plan queda bloqueado en v0.

Errores:

- Hereda errores de `DeploymentPlan v0` cuando el plan base es invalido.
- `paas_deploy_target_invalido`
- `paas_deploy_resultado_invalido`
- `paas_deploy_rollback_no_reversible`
- `paas_deploy_proveedor_no_declarado`
- `paas_deploy_sin_publicacion`
- `paas_deploy_matriz_cliente_invalida`
- `paas_deploy_contenedor_no_puro`

Pruebas de contrato:

- Unit: `go test -count=1 .` prepara `PaaSDeployResultV0` desde `paas_valido.json`.
- Unit: rechaza target no paas, matriz sin filas cliente, proveedor no declarado, ausencia de `publicar_declarativo`, rollback no reversible y acciones con contenedor.
- Smoke: `git diff --check -- .` confirma higiene del diff sin crear artefactos de deploy.

## ServerlessDeployAdapter v0

Nombre: `ServerlessDeployAdapter`
Tipo: conector hexagonal de salida para familia `serverless`
Version: `v0`
Propietario: `orquesta-deploy`
Estado compartido: backlog documental local.
Consumidores:

- `orquesta-deploy`, cuando un `DeploymentPlanV0` resuelva `deploy_target = serverless`.

Entrada:

- `DeploymentPlanV0` validado con `deploy_target = serverless`.
- Restricciones opacas para proveedor, runtime, cuenta, region y triggers.
- `validacion_entorno`, `artefactos_previstos`, `healthcheck`, `rollback` y `acciones_previstas` completos.

Invariantes:

- V0 de familia, no de proveedor: no nombra servicios, runtimes ni empaquetadores concretos.
- Solo dry-run declarativo futuro: sin credenciales reales, sin desplegar funciones y sin configurar triggers.
- El runtime gestionado, politicas de empaquetado y naming operativo quedan fuera hasta nueva microtarea.

Errores publicos previstos:

- `serverless_deploy_target_invalido`
- `serverless_deploy_resultado_invalido`
- `serverless_deploy_runtime_no_declarado`

## Plantilla

```text
Nombre:
Tipo: puerto_entrada | puerto_salida | dto | evento | error
Version:
Propietario:
Consumidores:
Campos:
Invariantes:
Errores:
Pruebas de contrato:
```
