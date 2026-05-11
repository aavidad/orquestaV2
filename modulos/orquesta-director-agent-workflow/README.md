# orquesta-director-agent-workflow

Adaptador hexagonal entre una decision de un agente director externo y comandos
publicos de `orquesta-core-workflow`.

Responsabilidades:

- validar `DirectorAgentDecisionV0`;
- traducir `request_brainstorm` a `RequestBrainstorm`;
- construir metadata idempotente y compacta;
- aplicar una decision a un run mediante puertos inyectados;
- rechazar DTOs sin comando publico del workflow;
- mantener fuera runtime, DB, proveedor, modelo, HOME y credenciales.

No crea conectores por defecto. El caller inyecta store y sink.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-director-agent-workflow
```
