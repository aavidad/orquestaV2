# Pruebas

Comando:

```bash
go test ./modulos/orquesta-director-agent-file-source -count=1
```

Cobertura inicial:
- lee sobre versionado y devuelve decisiones validadas;
- filtra descriptores de otro run;
- rechaza decisiones invalidas;
- normaliza `create_microtask.phase_id=programacion` a
  `planificacion_microtareas` si esa fase venia de la tarea objetivo;
- normaliza `command_type=create_microtask.v0` a `create_microtask` si el
  payload canonico esta presente;
- rechaza `create_microtask.v0` sin payload canonico;
- tolera comas finales de JSON emitido por agentes y valida despues el contrato;
- normaliza payloads planos de agente para `request_vote`, `accept_decision` y
  `publish_function_contract`;
- filtra decisiones de otro run aunque el descriptor no lleve `run_id`;
- pasa al proveedor las proyecciones compactas `phase_artifacts` y `deliveries`
  del run para preservar causalidad en conectores externos;
- lector OS respeta limite maximo.
