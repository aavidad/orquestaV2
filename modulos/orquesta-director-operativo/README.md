# orquesta-director-operativo

Contrato puro para el primer corte del Director Operativo V1.

El objetivo es modelar lo que hoy hace bien un Codex coordinador externo:
mantener un plan vivo, dividir trabajo, lanzar subagentes, esperar, revisar,
replanificar y cerrar con pruebas. Este modulo solo define el plan operativo y
sus guardas. No arranca agentes ni conoce Codex, OPES, runtime, DB, web ni MCP.

Debe servir para:

- autoprogramacion de Orquesta;
- trabajos de dominio externos como OPES;
- futuros dominios que entren por contratos y refs opacas.

Tambien modela delegacion recursiva gobernada: un agente puede solicitar
subagentes, pero solo bajo presupuesto del director, profundidad maxima,
fanout maximo, parent/child refs y review posterior. No existe spawn libre.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-director-operativo
```

Estado tras el primer corte:

- este modulo llega hasta contrato, validacion y proyeccion a olas;
- la materializacion inicial vive en
  `modulos/orquesta-orchestration-core/operational_director_materializer_v0.go`;
- hoy solo se materializan items `launch_subagents` listos como workflow tasks y
  comandos `CreateMicrotask`;
- quedan pendientes wait por cohorte/ola, review/rework/replan/cierre durable y
  recursion Codex real con parent/child refs.
