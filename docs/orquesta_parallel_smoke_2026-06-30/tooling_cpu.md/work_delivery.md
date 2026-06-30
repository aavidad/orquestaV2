# Tooling CPU: codebase-memory-mcp y subagentes

Refs: `req-orquesta-parallel-tooling-cpu-20260630T143204Z`,
`job-ref-orquesta-parallel-tooling-cpu-20260630T143204Z`,
`BUG-ORQ-20260630-032`.

Alcance: auditoria doc-only. No se uso `codebase-memory-mcp`, indexadores,
servidores, OPES productivo ni temarios. La lectura fue acotada con `sed`,
`rg`, `ps` y contratos locales.

## Evidencia revisada

- `docs/inventario_bugs_orquesta_2026-06-30.md` ya registra
  `BUG-ORQ-20260630-032`: varias instancias de `codebase-memory-mcp` quedaron
  vivas tras sesiones/subagentes y consumian CPU sin trabajo activo.
- `scripts/bootstrap_agent_tooling.sh` ya mitiga la causa inmediata: Codebase
  queda opt-in, no se instala/indexa por defecto y la norma persistente prohibe
  usarlo por defecto en subagentes o trabajo paralelo amplio.
- `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md` T259 demuestra que
  el self-watchdog existente gobierna CPU del servidor Orquesta por
  composicion, correlando CPU alta con progreso causal antes de parar.
- Comprobacion de proceso por `comm=codebase-memory-mcp`: sin procesos vivos en
  esta sesion al momento de escribir este informe.

## Diagnostico

El fallo pendiente no es de busqueda de codigo, sino de ciclo de vida de
herramientas auxiliares. En trabajo paralelo, cada agente o subagente puede
abrir su propio MCP/indexador sin que Orquesta tenga una identidad causal de esa
herramienta, lease, heartbeat, ultimo uso ni release durable. Si el padre acaba,
queda bloqueado o pierde la sesion de transporte, el proceso auxiliar puede
seguir consumiendo CPU aunque ya no exista trabajo causal observable.

La mitigacion documental reduce la probabilidad, pero no cierra el bug: una
herramienta opt-in autorizada todavia necesita ownership, TTL, renovacion y
parada cooperativa. El self-watchdog actual del servidor no debe barrer
procesos por nombre ni matar recursos ajenos; por tanto necesita un registro de
herramientas propias antes de actuar sobre MCPs o indexadores.

## Propuesta watchdog/lease

1. Crear un contrato de composicion `AuxToolLeaseV0` para herramientas
   auxiliares, fuera del core puro. Campos minimos: `lease_ref`, `tool_ref`,
   `owner_kind`, `owner_ref`, `run_ref`, `goal_ref`, `purpose_ref`,
   `started_at`, `ttl_seconds`, `renewed_at`, `last_request_at`,
   `process_ref` opaco, `cpu_percent_window`, `status`, `action_policy` y
   `evidence_refs`. No publicar rutas locales, HOME, tokens ni argumentos
   completos.
2. Lanzar MCPs/indexadores solo mediante wrapper de composicion que primero
   cree el lease, inyecte `ORQUESTA_AUX_TOOL_LEASE_REF`, renueve en cada
   peticion real y haga release al cerrar. Los subagentes quedan en
   `default_deny` para `codebase-memory-mcp`; solo pueden usarlo con lease
   explicito y justificacion acotada.
3. Anadir `AuxToolWatchdogV0` junto al self-watchdog del servidor: lista leases
   registrados, correlaciona CPU alta con `last_request_at`, heartbeat y trabajo
   causal activo, y publica diagnostico `aux_tool_idle_high_cpu` cuando no hay
   progreso.
4. Politica de accion: primero alertar y marcar health degradado; despues
   pedir parada cooperativa al wrapper; solo tras gracia configurable parar el
   proceso propio registrado por `process_ref`. Prohibido `pkill`, barridos por
   nombre o matar procesos no registrados por Orquesta.
5. Configuracion canonica por composicion: herramientas permitidas, maximo por
   run/goal/agente, TTL, ventana de CPU, ventana sin uso y accion final. Los
   defaults deben conservar Codebase como opt-in y no habilitar indexado por
   defecto.

## Pruebas propuestas

- Unit fake metrics: lease activo con CPU alta y peticion reciente no dispara
  parada.
- Unit fake metrics: lease vencido, sin peticiones ni trabajo causal, CPU alta
  sostenida produce `aux_tool_idle_high_cpu` y solicitud de parada cooperativa.
- Unit frontera: el contrato puro de lease no importa Codex, OPES, DB concreta,
  HOME, OAuth, proveedor ni filesystem real.
- Integracion opt-in: subagente sin lease no puede arrancar
  `codebase-memory-mcp`; subagente con lease explicito arranca por wrapper,
  renueva y libera; si se queda huerfano, el watchdog solo detiene el proceso
  registrado.

Conclusion: mantener la norma opt-in actual y cerrar el pendiente con un
registro de leases de herramientas auxiliares mas un watchdog de composicion.
Eso ataca la multiplicacion por subagentes sin convertir una heuristica de
contenido en rail, y sin contaminar el nucleo generico con procesos reales.
