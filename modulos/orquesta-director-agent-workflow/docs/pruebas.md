# Pruebas locales

```bash
go test -count=1 ./modulos/orquesta-director-agent-workflow
```

Cobertura:

- la idempotencia de cada comando traducido usa `command_ref`, aunque el payload
  referencie una decision aceptada anterior;
- traduce `request_brainstorm` a `RequestBrainstorm`;
- traduce `open_phase`, `request_vote` y `accept_decision`;
- traduce `publish_function_contract` a `PublishFunctionContract`;
- traduce `create_microtask` a `CreateMicrotask`;
- traduce `ask_director` y `ask_user` a `AskDirector`, conservando `target_group=user` en `ask_user`;
- traduce `request_capacity` a `RequestCapacity`;
- traduce `request_agent` a `RequestAgent`;
- traduce y aplica `close_task` como `CloseTask` entre `accept_review` y
  `open_phase validacion_final`;
- traduce `request_rework` y `record_replan_decision`;
- traduce `register_final_validation` y `close_run` a comandos publicos de cierre;
- construye `DirectorAgentCompactStatsV0` desde `OrchestrationRunV0` publico;
- aplica una decision de brainstorm a un run por puertos;
- encadena director desde brainstorming hasta planificacion y microtarea sin comandos manuales intermedios;
- aplica contrato y microtarea sobre un run en planificacion con decision aceptada;
- guarda la microtarea completa en `TaskStore` para que el scheduler pueda programarla despues;
- rechaza `create_microtask` sin `TaskStore`;
- rechaza `propose_autonomous_plan_team` como comando workflow hasta que exista comando publico del core;
- rechaza decisiones invalidas;
- exige `occurred_at`;
- no importa legacy, DB ni conectores operativos.
