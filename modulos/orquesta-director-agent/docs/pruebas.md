# Pruebas locales

Comando local:

```bash
go test -count=1 ./modulos/orquesta-director-agent
```

Suite T15 reconciliada:

```bash
go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-core-workflow ./modulos/orquesta-context ./modulos/orquesta-director-agent ./cmd/orquesta-server
```

Evidencia esperada: el DTO acepta refs opacas y vocabulario operativo, conserva
context_refs compactas y rechaza proveedor/modelo reales, rutas privadas,
prompts/transcripts crudos, secretos o payloads largos.

Cobertura local:

- acepta `request_brainstorm` compacto;
- acepta `open_phase`, `request_vote` y `accept_decision`;
- acepta `publish_function_contract` compacto;
- acepta `create_microtask` con contrato funcional explicito;
- acepta `create_microtask` con `write_set` amplio de tarea vertical real sin
  relajar refs/evidencias compactas;
- acepta `create_microtask` con linaje recursivo neutral y refs compactas;
- acepta `create_microtask` con `context_refs` opacas compactas;
- acepta microtareas documentales paralelas en `programacion` sin
  `required_tests` cuando el `write_set` es puramente documental;
- acepta `work_profile_kind` opcional en microtareas y unidades de equipo;
- acepta `ask_director` y `ask_user` compactos;
- acepta `request_capacity` y `request_agent` compactos;
- acepta `propose_autonomous_plan_team` compacto;
- acepta `request_rework` y `record_replan_decision` compactos;
- acepta `close_task` entre `accept_review` y `open_phase validacion_final`;
- acepta `register_final_validation` y `close_run` compactos;
- acepta `DirectorAgentCompactStatsV0` con contadores y refs pendientes;
- rechaza proveedor/modelo y payloads largos;
- rechaza microtareas sin contrato funcional.
- rechaza microtareas de programacion que tocan codigo sin `required_tests`.
- rechaza linaje recursivo incoherente en microtareas.
- rechaza `context_refs` no compactas o con detalles prohibidos.
- rechaza plan/equipo con detalle operativo o asignaciones inexistentes.
- rechaza cierre con fase incoherente o evidencias con rutas.
- rechaza `close_task` sin evidencias o fuera de `revision`.
- rechaza status de revision no soportado.
- rechaza stats con rutas o detalle operativo.

Prueba real opt-in:

```bash
ORQUESTA_PRUEBA_REAL_OPT_IN=1 \
ORQUESTA_PRUEBA_REAL_DIRECTOR_COMMAND_PATH=/abs/director-cli-wrapper \
ORQUESTA_PRUEBA_REAL_DIRECTOR_WORKDIR=/abs/workdir \
./scripts/probar_director_agente_real_orquesta_v2.sh
```

Una ejecucion solo cuenta como real si el agente externo lo lanza Orquesta y el ACK/artifact correlaciona con `AgentStartPacketV0`.
