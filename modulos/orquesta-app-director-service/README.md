# orquesta-app-director-service

Servicio de aplicacion para arrancar una solicitud de app por director.

Es la ruta operativa preferente para `AppSpecV0` cuando una app nueva requiere
juicio del Director V2, plan-state, waits acotados, review/tests/cierre o
recursion gobernada. Los entrypoints de `app-runner` quedan como
preview/compatibilidad.

Flujo:

1. valida `AppSpecRequestV0` con factory;
2. prepara intake de director;
3. guarda el run inicial por `RunStorePortV0`;
4. compone observadores opcionales de entregas, progreso, leases y replans;
5. ejecuta el loop progresivo del nucleo con dispatchers inyectados;
6. devuelve un resultado compacto para REST/MCP/web.

El resultado incluye `director_tasks` para que la superficie publica pueda
mostrar el equipo preparado sin conocer scheduler, outbox ni decisiones internas.

No crea persistencia ni runtime por defecto. Todo efecto se inyecta. Si se
inyectan fuentes de observacion, el servicio las adapta a candidates del nucleo
sin que REST/MCP/web conozcan scheduler, outbox ni comandos internos.

Mapa local:

- `docs/mapa.md`: indice conceptual del modulo, ciclo de entrada/continuacion,
  grupos de ficheros y reglas para ampliar sin mezclar adaptadores.
- `docs/contratos.md`: contratos publicos `StartAppDirectorV0` y
  `ContinueAppDirectorV0`.
- `docs/tareas.md`: backlog local y evidencias de cierre por tarea.
- `docs/pruebas.md`: matriz de pruebas focales del servicio.
- `docs/decisiones.md`: decisiones arquitectonicas aceptadas o historicas.

Validacion:

```bash
go test -count=1 ./modulos/orquesta-app-director-service
```

## Uso operativo

Casos sanos:

- arranque normal: la request trae `AppSpecV0`, el servicio abre run/intake y
  espera decisiones del director por puertos inyectados;
- plan operativo directo: la composicion trae `OperationalDirectorPlanV0`
  `ready` y contratos funcionales explicitos; el servicio publica la cadena
  causal de bootstrap, materializa `launch_subagents`, persiste `PlanState` si
  hay store y entra al loop con scope de ola/cohorte;
- reentrada: el caller usa `operational_director_plan_ref` o filtros
  `wait_wave_ref`/`wait_cohort_ref`/`wait_parent_task_ref`; el servicio resuelve
  `WaitAgentRefs` con `DirectorTaskStore` y no espera agentes vivos ajenos.

Supuestos practicos:

- `DirectorTaskStore`, `OperationalPlanStateStore`, `WaitStateStore`,
  `EventReader`, `RequiredTestEvidenceStore`, `RequiredTestRunner` y
  `OperationalClosureSource` son puertos de composicion; si faltan, el servicio
  bloquea o reintenta con razon durable en vez de inventar estado.
- `WaitAgentRefs` explicitas tienen compatibilidad legacy, pero si existen refs
  de ola/cohorte/parent se prefieren para acotar pending, wait e ingesta.
- `domain_work` no inventa tests de programacion: el cierre de dominios externos
  depende de artefactos, validacion y refs opacas aportadas por la composicion.

Errores frecuentes:

- usar `wait_agent_refs` vacio esperando que signifique "scope actual": vacio es
  compatibilidad de run completo; usar `wait_wave_ref`, `wait_cohort_ref` o
  `operational_director_plan_ref` para reentradas del Director Operativo;
- pedir filtros de wait sin `DirectorTaskStore`: no hay forma segura de derivar
  agentes desde metadata de `WorkflowTaskV0`;
- llamar cierre desde steps previos a `replan_or_close/running`: el servicio no
  debe invocar `OperationalClosureSource` mientras review/tests/outbox no hayan
  dejado evidencia causal suficiente.
