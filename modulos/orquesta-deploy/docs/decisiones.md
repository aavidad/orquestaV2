# Decisiones locales: orquesta-deploy

Las decisiones de este archivo solo afectan a `orquesta-deploy`. Si afectan a otro modulo, deben elevarse al director y registrarse en `../../CONTRATOS.md`.

## 2026-05-04 - DeploymentPlan v0 como contrato declarativo

Fecha: `2026-05-04`
Decision: Arrancar `DeploymentPlan v0` como contrato declarativo promovido a `v0 compartido`, con detalle canonico local y backlog inicial, sin scripts ni adaptadores reales.
Motivo: El contrato compartido necesita una frontera hexagonal local antes de tocar Docker, cloud, Kubernetes, tiendas o filesystem.
Alternativas:

- Crear scripts iniciales de deploy: descartado porque violaria la regla local de no crear scripts sin contrato ni smoke.
- Hardcodear un proveedor o runtime inicial: descartado porque el contrato v0 debe ser generico y sin secretos.

Impacto:

- `orquesta-deploy` puede preparar el siguiente corte con pruebas de contrato antes de implementar conectores.
- Los adaptadores reales quedan bloqueados hasta que tengan contrato, write-set y smoke.

Contratos afectados: `DeploymentPlan v0`
Estado: `aceptada_local`

Nota 2026-05-04: el director promovio `DeploymentPlan v0` a contrato compartido en `../../CONTRATOS.md`; DEP-001 deja de estar bloqueada por contrato global.

## 2026-05-04 - Matriz OS explicita

Fecha: `2026-05-04`
Decision: Todo `DeploymentPlan v0` debe declarar matriz por `linux`, `darwin`, `windows` cuando aplique.
Motivo: Multi-OS no debe quedar como promesa generica; cada target necesita aplicabilidad y motivo por sistema.
Alternativas:

- Documentar soporte multi-OS global: descartado por poco verificable.
- Postergar OS hasta implementacion: descartado porque la validacion de entorno depende de OS.

Impacto:

- Las pruebas de contrato deben comprobar presencia de matriz OS.
- Los targets pueden marcar `no_aplica`, pero deben explicar el motivo.

Contratos afectados: `DeploymentPlan v0`
Estado: `aceptada_local`

## 2026-05-04 - Contenedor solo bajo target o aceptacion

Fecha: `2026-05-04`
Decision: Docker/contenedor solo puede aparecer como requisito cuando el target lo pide o el usuario lo acepta.
Motivo: El deploy no debe imponer Docker como default tecnico.
Alternativas:

- Usar contenedor por defecto para homogeneizar entornos: descartado porque contradice las reglas locales.
- Dejarlo como recomendacion libre: descartado porque no seria verificable por contrato.

Impacto:

- `DeploymentPlan v0` incluye `aceptacion_contenedor`.
- El error publico `contenedor_no_aceptado` cubre planes que intenten requerir contenedor sin habilitacion.

Contratos afectados: `DeploymentPlan v0`
Estado: `aceptada_local`

## 2026-05-04 - Schema y fixtures canonicos DeploymentPlan v0

Fecha: `2026-05-04`
Decision: Publicar `deployment_plan_v0.schema.json` y tres fixtures canonicos para `contenedor`, `local` y target desconocido.
Motivo: DEP-004 necesita una verificacion contractual previa antes de cualquier adaptador real o archivo ejecutable.
Alternativas:

- Mantener la validacion solo documental: descartado porque el contrato compartido ya pide schemas/fixtures antes de scripts o adaptadores.
- Crear artefactos ejecutables de deploy: descartado porque v0 sigue siendo declarativo.

Impacto:

- Los consumidores pueden validar planes declarativos contra un schema estable.
- El fixture negativo fija que un target fuera de enum falla por `deploy_target`.

Contratos afectados: `DeploymentPlan v0`
Estado: `aceptada_local`

## 2026-05-04 - Harness Go relacional DeploymentPlan v0

Fecha: `2026-05-04`
Decision: Agregar un harness Go puro para decodificar JSON serializable y validar invariantes relacionales de `DeploymentPlanV0` que el schema no expresa completamente.
Motivo: El contrato necesita una verificacion local ejecutable antes de cualquier adaptador real de deploy.
Alternativas:

- Validar solo con JSON Schema: descartado porque algunas reglas cruzan campos, como target resuelto, aceptacion de contenedor y acciones previstas.
- Crear adaptadores o scripts de deploy para probar el flujo completo: descartado porque v0 sigue siendo declarativo.

Impacto:

- `orquesta-deploy` expone DTOs Go serializables y una funcion de validacion pura.
- Los adaptadores reales siguen bloqueados hasta contrato y smoke especificos.

Contratos afectados: `DeploymentPlan v0`
Estado: `aceptada_local`

## 2026-05-04 - Backlog de adaptadores futuros como conectores hexagonales

Fecha: `2026-05-04`
Decision: Registrar los adaptadores futuros de deploy como backlog local de conectores hexagonales, separados por targets `local`, `contenedor`, `kubernetes`, `paas`, `serverless`, `desktop` y `mobile_store`.
Motivo: El modulo necesita una secuencia verificable antes de crear artefactos ejecutables o tocar entornos reales.
Alternativas:

- Implementar primero adaptadores reales y documentar despues: descartado porque violaria el bloqueo de contrato y smoke previo.
- Crear un adaptador generico para todos los targets: descartado porque ocultaria diferencias de OS, proveedor, permisos, rollback y healthcheck.
- Elegir un proveedor inicial para cloud o tiendas: descartado porque seria una decision transversal y no pertenece a DEP-003.

Impacto:

- Cada adaptador futuro debe declarar contrato local versionado, write-set cerrado y smoke dry-run antes de existir como codigo o artefacto.
- Las integraciones reales futuras quedan separadas de pruebas contract/documentales y deben ser opt-in, aisladas y saltadas por defecto.
- Dockerfile, compose, manifiestos K8s, pipelines, scripts shell e IaC siguen prohibidos en DEP-003.
- Cualquier eleccion de proveedor, tienda, runtime gestionado o contrato compartido requiere microtarea posterior y, si afecta a otros modulos, `CONSULTA AL DIRECTOR`.

Contratos afectados: `DeploymentPlan v0`; futuros contratos locales `<Target>DeployAdapter v0`.
Estado: `aceptada_local`

## 2026-05-04 - LocalDeployAdapter v0 solo dry-run puro

Fecha: `2026-05-04`
Decision: Implementar `LocalDeployAdapterV0` como conector local puro que prepara `LocalDeployResultV0` en modo `dry_run` desde un `DeploymentPlanV0` local validado.
Motivo: DEP-006 puede avanzar sin tocar filesystem productivo ni crear artefactos de deploy si el adaptador se limita a validar precondiciones y devolver un resultado serializable.
Alternativas:

- Ejecutar preparacion local real: descartado porque requeriria politica de entorno, materializacion de artefactos y smoke opt-in.
- Aceptar acciones que requieren contenedor en target local: descartado porque este conector es local puro y el contenedor pertenece a otro adaptador.
- Inventar campos por OS para acciones no reversibles: descartado porque `DeploymentPlanV0` solo tiene rollback global; la politica por OS necesita contrato posterior.

Impacto:

- El adaptador rechaza target no local, OS no aplicable, rollback no reversible y acciones que requieren contenedor.
- El resultado dry-run copia validacion de entorno, artefactos previstos, healthcheck, rollback y acciones previstas sin materializarlos.
- Deploy local real, artefactos materializados y politica de no reversibilidad por OS quedan bloqueados para tareas futuras.

Contratos afectados: `DeploymentPlan v0`; `LocalDeployAdapter v0`.
Estado: `aceptada_local`

## 2026-05-04 - ContainerDeployAdapter v0 solo dry-run declarativo

Fecha: `2026-05-04`
Decision: Implementar `ContainerDeployAdapterV0` como conector puro que prepara `ContainerDeployResultV0` en modo `dry_run` desde un `DeploymentPlanV0` de target `contenedor` ya validado.
Motivo: DEP-007 puede avanzar sin crear Dockerfile, compose, imagenes ni runtime real si el adaptador se limita a validar precondiciones y devolver un resultado serializable.
Alternativas:

- Construir una imagen real durante la preparacion: descartado porque requeriria runtime, permisos, limpieza y smoke opt-in.
- Consultar daemon, socket o registry para validar el plan: descartado porque este slice debe ser puro y no tocar entorno real.
- Permitir planes de contenedor sin ninguna accion que requiera contenedor: descartado porque vaciaria el sentido del target y degradaria la trazabilidad del contrato.

Impacto:

- El adaptador rechaza target no contenedor, rollback no reversible y resultados inconsistentes.
- Los casos de contenedor no aceptado o sin accion que requiera contenedor siguen frenados por `DeploymentPlan v0`.
- El resultado dry-run copia validacion de entorno, artefactos previstos, healthcheck, rollback y acciones previstas sin materializarlos.
- La deteccion de runtime real, construccion de imagenes, uso de sockets y publicacion en registry quedan bloqueados para tareas futuras.

Contratos afectados: `DeploymentPlan v0`; `ContainerDeployAdapter v0`.
Estado: `aceptada_local`

## 2026-05-04 - KubernetesDeployAdapter v0 solo dry-run declarativo

Fecha: `2026-05-04`
Decision: Implementar `KubernetesDeployAdapterV0` como conector puro que prepara `KubernetesDeployResultV0` en modo `dry_run` desde un `DeploymentPlanV0` de target `kubernetes` ya validado.
Motivo: DEP-008 puede avanzar sin kubeconfig, cluster, manifests ni proveedor si el slice se limita a validar precondiciones y devolver un resultado serializable.
Alternativas:

- Leer kubeconfig o contexto actual para validar el plan: descartado porque romperia el aislamiento y meteria dependencias del entorno.
- Generar manifests, Helm o Kustomize en este slice: descartado porque requiere contrato adicional, write-set de artefactos y smoke separado.
- Resolver proveedor, namespace o naming operativo desde aqui: descartado porque eso cruza modulos y endurece demasiado pronto el contrato.

Impacto:

- El adaptador rechaza target no kubernetes, rollback no reversible, ausencia de `publicar_declarativo` y matrices que no distinguen filas cliente.
- El resultado dry-run copia validacion de entorno, artefactos previstos, healthcheck, rollback y acciones previstas sin materializarlos.
- Cluster real, kubeconfig, manifests, Helm/Kustomize y convenciones de proveedor quedan bloqueados para tareas futuras.

Contratos afectados: `DeploymentPlan v0`; `KubernetesDeployAdapter v0`.
Estado: `aceptada_local`

## 2026-05-04 - PaaS y serverless quedan como familias documentales

Fecha: `2026-05-04`
Decision: Cerrar DEP-009 solo a nivel documental/contractual, dejando `PaaSDeployAdapter v0` y `ServerlessDeployAdapter v0` como familias genericas sin proveedor concreto.
Motivo: El backlog necesita una frontera clara para deploy gestionado, pero no conviene fijar SDK, runtime, region, naming ni IaC antes de saber que proveedor va a sobrevivir.
Alternativas:

- Elegir ya un proveedor PaaS o serverless: descartado porque generaria deuda temprana y acoplaria otros modulos.
- Dejar DEP-009 totalmente vacio: descartado porque impide planificar tareas futuras y no captura prohibiciones importantes.

Impacto:

- Quedan definidos invariantes, errores publicos previstos y prohibiciones por familia sin prometer implementacion falsa.
- Cualquier avance ejecutable posterior requerira nueva microtarea, write-set propio y smoke opt-in.
- Se evita filtrar credenciales, cuentas, regiones, nombres de servicio y runtimes concretos al contrato base.

Contratos afectados: `PaaSDeployAdapter v0`; `ServerlessDeployAdapter v0`.
Estado: `aceptada_local`

## 2026-05-04 - Desktop y mobile_store cierran como dry-run puros

Fecha: `2026-05-04`
Decision: Cerrar DEP-010 con dos adaptadores pequenos y puros, `DesktopDeployAdapterV0` y `MobileStoreDeployAdapterV0`, ambos en modo `dry_run`.
Motivo: El backlog de deploy quedaba incompleto sin targets de escritorio y publicacion mobile, pero no conviene introducir firmado, notarizacion, paquetes reales ni tiendas antes de tener reglas mas estables.
Alternativas:

- Cerrar DEP-010 solo documentalmente: descartado porque ya habia patron contrastado para slices puros y el modulo podia soportar un cierre ejecutable pequeno.
- Implementar solo `desktop`: descartado porque dejaria `mobile_store` como hueco sin invariantes minimos ni fixtures.
- Integrar firmado, notarizacion o subida a tienda: descartado porque exigiria credenciales, proveedores y smoke opt-in fuera del alcance.

Impacto:

- `orquesta-deploy` cubre ya todos los targets v0 con DTOs o adaptadores verificables dentro del modulo.
- `desktop` exige artefacto `paquete_desktop`, matriz aplicable en `linux`, `darwin` y `windows`, y prohbe contenedor.
- `mobile_store` exige `metadata_publicacion`, una accion `solicitar_decision`, `darwin` aplicable y filas cliente para `linux` y `windows`.
- Firmado, notarizacion, publicacion real, tiendas concretas y politicas compartidas siguen bloqueados para microtareas posteriores.

Contratos afectados: `DeploymentPlan v0`; `DesktopDeployAdapter v0`; `MobileStoreDeployAdapter v0`.
Estado: `aceptada_local`

## 2026-05-04 - PaaSDeployAdapter v0 solo dry-run declarativo

Fecha: `2026-05-04`
Decision: Implementar `PaaSDeployAdapterV0` como conector puro que prepara `PaaSDeployResultV0` en modo `dry_run` desde un `DeploymentPlanV0` de target `paas` ya validado.
Motivo: DEP-011 puede materializar el contrato documental de PaaS sin elegir proveedor, runtime, region, cuenta, SDK ni formato IaC.
Alternativas:

- Elegir un proveedor PaaS inicial: descartado porque cruzaria decisiones de cuenta, runtime y credenciales.
- Generar archivos de plataforma o IaC: descartado porque requiere contrato y smoke propios.
- Consultar SDK o entorno real para validar disponibilidad: descartado porque rompe el caracter puro del slice.

Impacto:

- El adaptador rechaza target no paas, matriz sin filas cliente, proveedor no declarado, ausencia de `publicar_declarativo`, rollback no reversible y acciones con contenedor.
- El resultado dry-run copia validacion de entorno, artefactos previstos, healthcheck, rollback y acciones previstas sin materializarlos.
- Proveedor, runtime gestionado, region, cuenta, SDK, IaC y despliegue real quedan bloqueados para microtareas posteriores.

Contratos afectados: `DeploymentPlan v0`; `PaaSDeployAdapter v0`.
Estado: `aceptada_local`

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
