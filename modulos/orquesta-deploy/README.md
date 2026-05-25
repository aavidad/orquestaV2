# orquesta-deploy

Responsabilidad: preparar contratos declarativos de entorno y despliegue para
Orquesta y apps generadas.

Incluye:

- Docker;
- compose;
- K8s;
- VM/systemd;
- serverless;
- edge;
- healthcheck;
- rollback;
- dependencias por sistema operativo.

Estado vigente: `DeploymentPlan v0` se expone como contrato de composicion y
dry-run por puerto. No decide arquitectura de producto y no ejecuta Docker,
Kubernetes, cloud, filesystem productivo ni secretos desde el nucleo.
