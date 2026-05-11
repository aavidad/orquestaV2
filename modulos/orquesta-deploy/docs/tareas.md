# Tareas locales: orquesta-deploy

Cada tarea debe ser pequena y cerrada.

## DEP-000 - Corte documental ejecutable de deploy

ID: `DEP-000`
Objetivo: Definir el alcance inicial de `orquesta-deploy` como conector hexagonal documental, sin scripts ni adaptadores reales.
Write-set:

- `docs/contratos.md`
- `docs/tareas.md`
- `docs/pruebas.md`
- `docs/decisiones.md`

Simbolo foco: `DeploymentPlan v0`
Contrato: `DeploymentPlan v0`
Validacion:

- `git diff --check`
- Revisar que el contrato local incluya target, matriz OS, validacion de entorno, healthcheck, rollback y artefactos previstos.

Bloqueos:

- No tocar `../../CONTRATOS.md` sin consulta al director.
- No crear scripts, manifiestos ni conectores reales.

Estado: `completada`

## DEP-001 - Contrato local DeploymentPlan v0

ID: `DEP-001`
Objetivo: Documentar `DeploymentPlan v0` como puerto de salida/servicio de deploy para preparar entorno segun `AppSpecV0.Deploy.Target`.
Write-set:

- `docs/contratos.md`
- `docs/pruebas.md`

Simbolo foco: `DeploymentPlan`
Contrato: `DeploymentPlan v0`
Validacion:

- Casos de contrato previstos para `contenedor`, `local` y target no soportado en `docs/pruebas.md`.
- Confirmar targets v0: `sin_preferencia`, `local`, `contenedor`, `paas`, `serverless`, `kubernetes`, `desktop`, `mobile_store`.

Bloqueos:

- Contrato ya promovido a `v0 compartido` en `../../CONTRATOS.md`.
- Cualquier cambio incompatible futuro requiere nueva version o `CONSULTA AL DIRECTOR`.

Estado: `completada`

## DEP-002 - Matriz OS y politica de contenedor

ID: `DEP-002`
Objetivo: Precisar la matriz multi-OS y la regla de Docker/contenedor solo bajo target explicito o aceptacion de usuario.
Write-set:

- `docs/contratos.md`
- `docs/decisiones.md`

Simbolo foco: `matriz_os`
Contrato: `DeploymentPlan v0`
Validacion:

- Revisar que cada target documentado tenga filas explicitas para `linux`, `darwin`, `windows`.
- Revisar que `contenedor_no_aceptado` exista como error publico.

Bloqueos:

- No inferir preferencias de contenedor desde otros modulos en cambios futuros.
- No elegir proveedor cloud ni runtime.

Estado: `completada`

## DEP-003 - Backlog de adaptadores futuros

ID: `DEP-003`
Objetivo: Preparar backlog inicial para adaptadores reales de deploy sin implementarlos todavia.
Write-set:

- `docs/tareas.md`
- `docs/pruebas.md`
- `docs/decisiones.md`

Simbolo foco: `acciones_previstas`
Contrato: `DeploymentPlan v0`
Validacion:

- Cada adaptador futuro debe declarar contrato, write-set y smoke antes de crear archivos ejecutables.
- `docs/pruebas.md` debe separar pruebas de contrato/documentales de integraciones reales futuras.
- `git diff --check -- .`

Bloqueos:

- No escribir Dockerfile, compose, manifiestos K8s, pipelines, scripts shell ni IaC en esta tarea.
- No incluir secretos ni nombres de proveedor.

Estado: `completada`

### Backlog DEP-003 - adaptadores futuros

Regla comun: estos items son backlog documental. No autorizan implementacion, archivos ejecutables, artefactos de deploy ni integraciones reales. Cada tarea futura debe arrancar con contrato local versionado, write-set cerrado y prueba de smoke documentada antes de crear cualquier conector o materializar artefactos.

Contrato futuro minimo por adaptador:

- Nombre versionado: `<Target>DeployAdapter v0` o contrato local equivalente.
- Tipo: conector hexagonal de salida de `orquesta-deploy`.
- Entrada: `DeploymentPlanV0` validado, target compatible, matriz OS explicita, validacion de entorno declarada y aceptacion de contenedor cuando aplique.
- Salida: resultado serializable con estado, evidencias compactas, artefactos materializados o previstos, healthcheck, rollback y advertencias publicas.
- Errores: codigos publicos sin secretos, sin rutas HOME reales, sin proveedor hardcodeado y sin detalles internos de otro modulo.
- Pruebas: contrato documental, unitarias puras cuando haya codigo y smoke en modo dry-run o entorno efimero antes de cualquier integracion real.
- Escalado: si el resultado pasa a ser contrato compartido o se elige proveedor/plataforma concreta, preparar `CONSULTA AL DIRECTOR`.

Prohibiciones comunes:

- No escribir Dockerfile, compose, manifiestos K8s, pipelines, scripts shell ni IaC desde DEP-003.
- No incluir secretos, tokens, credenciales, rutas absolutas reales, nombres de cuenta, proveedor cloud hardcodeado, tienda concreta, DB, runtime, agente ni modelo.
- No mutar entornos reales por defecto; cualquier integracion real futura debe ser opt-in, aislada y saltada por defecto.
- No cruzar `internal/` ni structs privados de otros modulos.

#### DEP-006 - Backlog conector local

ID: `DEP-006`
Objetivo: Preparar un conector hexagonal local puro para dry-run por OS aplicable, sin asumir runtime, instaladores globales ni filesystem productivo.
Write-set:

- `docs/contratos.md`
- `docs/pruebas.md`
- `docs/decisiones.md`
- `docs/tareas.md`
- `local_deploy_adapter_v0.go`
- `local_deploy_adapter_v0_test.go`

Simbolo foco: `LocalDeployAdapter`
Contrato: `LocalDeployAdapter v0`, consumidor de `DeploymentPlanV0` con `deploy_target = local`.
Pruebas:

- Contract: fixture local valida que `linux`, `darwin`, `windows` tengan requisitos y motivos.
- Unit: validacion pura de precondiciones por OS sin tocar HOME real ni filesystem productivo.
- Unit: preparacion de `LocalDeployResultV0` serializable en modo `dry_run`, sin materializar artefactos.
- Unit: rechazo de target no local, OS no aplicable, rollback no reversible y acciones que requieren contenedor.
- Smoke: `git diff --check -- .`.

Bloqueos:

- Deploy local real sigue bloqueado hasta tarea posterior con contrato, entorno efimero y smoke opt-in.
- Las acciones no reversibles por OS siguen bloqueadas: `DeploymentPlanV0` solo expresa rollback global, no politica por OS.
- Cualquier materializacion de artefactos requiere contrato nuevo.

Prohibiciones:

- No modificar PATH, servicios del sistema, HOME real ni gestores globales de paquetes.
- No crear scripts shell ni wrappers ejecutables sin contrato y smoke previo.
- No crear Dockerfile, compose, manifiestos, pipelines ni IaC.

Estado: `completada`

#### DEP-007 - Backlog conector contenedor

ID: `DEP-007`
Objetivo: Implementar un conector hexagonal pequeno y puro para target `contenedor`, limitado a `dry_run` declarativo desde planes aceptados por target explicito o `aceptacion_contenedor = true`.
Write-set:

- `docs/contratos.md`
- `docs/pruebas.md`
- `docs/decisiones.md`
- `container_deploy_adapter_v0.go`
- `container_deploy_adapter_v0_test.go`

Simbolo foco: `ContainerDeployAdapterV0`
Contrato: `ContainerDeployAdapter v0`, consumidor de `DeploymentPlanV0` con `deploy_target = contenedor` o aceptacion explicita.
Pruebas:

- Unit: preparar `ContainerDeployResultV0` serializable en modo `dry_run` desde `contenedor_valido.json`.
- Unit: rechazar target no contenedor, rollback no reversible y resultado con target resuelto incorrecto.
- Contract: los casos de contenedor no aceptado o sin accion que requiera contenedor siguen rechazados por `DeploymentPlan v0`.
- Smoke: `git diff --check -- modulos/orquesta-deploy`.

Bloqueos:

- Crear Dockerfile, compose o imagen real requiere microtarea posterior con contrato, write-set y smoke propios.
- Publicar imagen o usar registry requiere decision y credenciales por referencias opacas.

Prohibiciones:

- No escribir Dockerfile ni compose desde este backlog.
- No hardcodear registry, proveedor, credenciales, etiquetas privadas ni rutas de socket.
- No consultar runtime real de contenedor, daemon, socket, permisos del host ni filesystem productivo.

Estado: `completada`

#### DEP-008 - Backlog conector Kubernetes

ID: `DEP-008`
Objetivo: Preparar un conector hexagonal futuro para target `kubernetes`, sin asumir proveedor, cluster, namespace ni contexto local.
Write-set futuro:

- `docs/contratos.md`
- `docs/pruebas.md`
- `docs/decisiones.md`
- Codigo y tests del conector Kubernetes en rutas exactas que declare la tarea futura.

Simbolo foco: `KubernetesDeployAdapter`
Contrato: `KubernetesDeployAdapter v0`, consumidor de `DeploymentPlanV0` con `deploy_target = kubernetes`.
Validacion:

- Unit: `go test -count=1 .` prepara `KubernetesDeployResultV0` desde `kubernetes_valido.json`.
- Unit: rechaza target no kubernetes, falta de `publicar_declarativo`, rollback no reversible y matriz sin filas cliente.
- Smoke: `git diff --check -- .` confirma higiene del diff sin crear manifiestos ni artefactos de despliegue.

Bloqueos:

- Crear manifiestos K8s, Helm/Kustomize o aplicar recursos requiere contrato especifico posterior.
- Elegir proveedor gestionado, naming de namespaces o convenciones de cluster requiere `CONSULTA AL DIRECTOR` si afecta a otros modulos.

Prohibiciones:

- No escribir manifiestos, charts, kustomizations ni pipelines desde DEP-003.
- No leer kubeconfig real, contexto actual, namespace por defecto ni secretos de cluster sin contrato explicito.

Estado: `completada`

#### DEP-009 - Backlog conectores PaaS y serverless

ID: `DEP-009`
Objetivo: Preparar conectores hexagonales futuros para `paas` y `serverless` como familias genericas, sin seleccionar proveedor ni runtime.
Write-set futuro:

- `docs/contratos.md`
- `docs/pruebas.md`
- `docs/decisiones.md`
- Codigo y tests de conectores PaaS/serverless en rutas exactas que declare la tarea futura.

Simbolo foco: `PaaSDeployAdapter` / `ServerlessDeployAdapter`
Contrato: `PaaSDeployAdapter v0` y `ServerlessDeployAdapter v0`, consumidores de `DeploymentPlanV0` con target compatible.
Validacion:

- Contract: revision documental de invariantes, errores publicos previstos y prohibiciones por familia.
- Smoke: `git diff --check -- .` confirma que el cierre de backlog no introduce SDK, provider, IaC ni runtime real.

Bloqueos:

- Elegir proveedor, region, cuenta, runtime gestionado o formato IaC requiere decision posterior y, si cruza modulos, `CONSULTA AL DIRECTOR`.
- Cualquier despliegue real necesita credenciales por referencia opaca y pruebas opt-in saltadas por defecto.

Prohibiciones:

- No escribir IaC, pipelines, archivos de plataforma, configuracion de proveedor ni nombres de servicios concretos desde DEP-003.
- No almacenar secretos, tokens, IDs de cuenta, regiones reales ni nombres de proyecto proveedor.

Estado: `completada_documental`

#### DEP-010 - Backlog conectores desktop y mobile store

ID: `DEP-010`
Objetivo: Preparar conectores hexagonales futuros para `desktop` y `mobile_store` solo cuando `AppSpecV0.Deploy.Target` lo requiera.
Write-set:

- `docs/contratos.md`
- `docs/pruebas.md`
- `docs/decisiones.md`
- `docs/tareas.md`
- `docs/fixtures/deployment_plan_v0/desktop_valido.json`
- `docs/fixtures/deployment_plan_v0/mobile_store_valido.json`
- `deployment_plan_contract_v0_test.go`
- `desktop_deploy_adapter_v0.go`
- `desktop_deploy_adapter_v0_test.go`
- `mobile_store_deploy_adapter_v0.go`
- `mobile_store_deploy_adapter_v0_test.go`

Simbolo foco: `DesktopDeployAdapter` / `MobileStoreDeployAdapter`
Contrato: `DesktopDeployAdapter v0` y `MobileStoreDeployAdapter v0`, consumidores de `DeploymentPlanV0` con target compatible.
Pruebas:

- Contract: `desktop_valido.json` y `mobile_store_valido.json` validan como `DeploymentPlanV0` y quedan cubiertos por `deployment_plan_contract_v0_test.go`.
- Unit: `go test -count=1 ./modulos/orquesta-deploy` prepara `DesktopDeployResultV0` y `MobileStoreDeployResultV0` en modo `dry_run`.
- Unit: desktop rechaza target no desktop, OS no aplicable, rollback no reversible, ausencia de `paquete_desktop` y acciones con contenedor.
- Unit: mobile_store rechaza target no mobile_store, matriz invalida, falta de `metadata_publicacion`, ausencia de `solicitar_decision`, rollback no reversible y acciones con contenedor.
- Smoke: `git diff --check -- modulos/orquesta-deploy`.

Bloqueos:

- Firmado, notarizacion, subida a tienda o integracion con tiendas requiere contrato posterior y credenciales por referencia opaca.
- Definir tiendas concretas o politicas de publicacion compartidas puede requerir `CONSULTA AL DIRECTOR`.

Prohibiciones:

- No escribir perfiles de firma, certificados, credenciales de tienda, pipelines ni manifests de publicacion desde DEP-003.
- No subir paquetes, registrar apps ni reservar nombres en servicios externos por defecto.

Estado: `completada`

#### DEP-011 - Conector PaaS dry-run puro

ID: `DEP-011`
Objetivo: Implementar un conector hexagonal pequeno y puro para target `paas`, limitado a `dry_run` declarativo sin seleccionar proveedor, runtime, region, cuenta, SDK ni IaC.
Write-set:

- `docs/contratos.md`
- `docs/pruebas.md`
- `docs/decisiones.md`
- `docs/tareas.md`
- `docs/fixtures/deployment_plan_v0/paas_valido.json`
- `deployment_plan_contract_v0_test.go`
- `paas_deploy_adapter_v0.go`
- `paas_deploy_adapter_v0_test.go`

Simbolo foco: `PaaSDeployAdapterV0`
Contrato: `PaaSDeployAdapter v0`, consumidor de `DeploymentPlanV0` con `deploy_target = paas`.
Pruebas:

- Contract: `paas_valido.json` valida como `DeploymentPlanV0` y queda cubierto por `deployment_plan_contract_v0_test.go`.
- Unit: `go test -count=1 .` prepara `PaaSDeployResultV0` en modo `dry_run`.
- Unit: rechaza target no paas, matriz sin filas cliente, proveedor no declarado como restriccion opaca, ausencia de `publicar_declarativo`, rollback no reversible y acciones con contenedor.
- Smoke: `git diff --check -- .`.

Bloqueos:

- Elegir proveedor, region, cuenta, runtime gestionado, formato IaC, SDK o archivo de plataforma requiere microtarea posterior.
- Cualquier despliegue real necesita credenciales por referencia opaca y pruebas opt-in saltadas por defecto.

Prohibiciones:

- No escribir IaC, pipelines, archivos de plataforma, configuracion de proveedor ni nombres de servicios concretos.
- No consultar SDK, entorno real, credenciales, cuentas, regiones, servicios gestionados ni filesystem productivo.

Estado: `completada`

## DEP-004 - Schema y fixtures DeploymentPlan v0

ID: `DEP-004`
Objetivo: Definir schema JSON y fixtures minimos de `DeploymentPlan v0` sin scripts, Dockerfiles, compose, manifiestos, pipelines ni adaptadores reales.
Write-set:

- `docs/tareas.md`
- `docs/pruebas.md`
- `docs/contratos.md`
- `docs/decisiones.md`
- `docs/schemas/deployment_plan_v0.schema.json`
- `docs/fixtures/deployment_plan_v0/contenedor_valido.json`
- `docs/fixtures/deployment_plan_v0/local_valido.json`
- `docs/fixtures/deployment_plan_v0/target_no_soportado_invalido.json`

Simbolo foco: `DeploymentPlanV0`
Contrato: `DeploymentPlan v0`
Validacion:

- `jq empty docs/schemas/*.schema.json docs/fixtures/deployment_plan_v0/*.json`
- `npx --yes ajv-cli@5.0.0 validate -s docs/schemas/deployment_plan_v0.schema.json -d docs/fixtures/deployment_plan_v0/contenedor_valido.json --spec=draft7 --all-errors`
- `npx --yes ajv-cli@5.0.0 validate -s docs/schemas/deployment_plan_v0.schema.json -d docs/fixtures/deployment_plan_v0/local_valido.json --spec=draft7 --all-errors`
- `npx --yes ajv-cli@5.0.0 validate -s docs/schemas/deployment_plan_v0.schema.json -d docs/fixtures/deployment_plan_v0/target_no_soportado_invalido.json --spec=draft7 --all-errors` debe fallar por `/deploy_target`.
- `git diff --check`

Bloqueos:

- No escribir Dockerfile, compose, manifiestos K8s, pipelines, scripts shell ni IaC.
- No incluir secretos, rutas HOME reales, proveedor cloud hardcodeado, DB, runtime, agente ni modelo.

Estado: `completada`

## DEP-005 - Harness Go relacional DeploymentPlan v0

ID: `DEP-005`
Objetivo: Implementar decode/validate puro en Go para invariantes relacionales de `DeploymentPlanV0`, sin scripts ni adaptadores reales.
Write-set:

- `deployment_plan_contract_v0.go`
- `deployment_plan_contract_v0_test.go`
- `docs/tareas.md`
- `docs/pruebas.md`
- `docs/decisiones.md`

Simbolo foco: `DeploymentPlanV0`
Contrato: `DeploymentPlan v0`
Validacion:

- `go test -count=1 ./modulos/orquesta-deploy`
- `git diff --check`

Bloqueos:

- No escribir Dockerfile, compose, manifiestos K8s, pipelines, scripts shell ni IaC.
- No crear adaptadores reales de DB, filesystem productivo, contenedor, proveedor cloud, CI/CD ni tienda.
- No modificar schema, fixtures canonicos ni contrato global en esta tarea.

Estado: `completada`

## Plantilla

```text
ID:
Objetivo:
Write-set:
Simbolo foco:
Contrato:
Validacion:
Bloqueos:
Estado:
```
