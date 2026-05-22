# Decisiones: orquesta-app-director-intake

```text
Fecha: 2026-05-09
Decision: La entrada canonica de app arranca por director.
Motivo: evita que REST, MCP, web u operador definan manualmente slices,
agentes y fases, que fue uno de los fallos de v1/v2.
Impacto: el director queda como pieza intercambiable; el nucleo solo ejecuta
comandos y outbox por puertos.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: `director_task` no entra en `Run.Tasks` durante el intake.
Motivo: `Run.Tasks` es la proyeccion de microtareas aceptadas por el workflow;
la tarea directora inicial solo sirve para pedir brainstorming/capacidad/agente.
Impacto: las estadisticas iniciales deben verse por `brainstorms` y agentes,
no por `tasks_total`. Las microtareas aparecen cuando el director publica
decisiones mediante el contrato de workflow.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: La autonomia alta prepara un equipo pequeno de directores por areas.
Motivo: una app mixta necesita pensar arquitectura, web, interfaz publica y
persistencia sin convertir el brainstorm en una tarea monolitica ni dejar que
REST/MCP/web elijan manualmente agentes.
Impacto: `director_tasks` conserva el plan completo, pero el provider alimenta
el scheduler en lotes acotados.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: El provider emite una cohorte inicial pequena de hasta cuatro work
candidates por tick.
Motivo: el director, web, API y persistencia deben poder arrancar juntos antes
de que el scheduler espere entregas de agentes vivos.
Impacto: se mantiene paralelismo real por batch con un limite pequeno y
explicito; payloads mayores siguen fuera de contrato.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Los summaries de arranque de agente son neutros.
Motivo: el outbox de launcher no debe transportar detalles tecnicos de
transporte, proveedor, HOME, credenciales, runtime ni base de datos.
Impacto: esos detalles viven en contratos/documentos de trabajo, no en el
payload compacto que arranca agentes.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Las refs de director se derivan de `spec_id`, no solo del slug de la
app.
Motivo: dos solicitudes con el mismo nombre visible generaban los mismos
task_ref, agent_request_id y outbox ids; la segunda podia quedar como "ok" sin
producir nuevos agentes.
Impacto: las tareas de director siguen siendo legibles, pero quedan aisladas
por especificacion validada y no colisionan entre runs de apps con el mismo
nombre.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Este modulo no sustituye al planner de prueba.
Motivo: el planner sirve para smoke tests de app pequena; el intake de director
es la ruta productiva para apps completas.
Impacto: podemos probar ambos caminos sin mezclar responsabilidades.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Los summaries neutros de director transportan `request_kind` y
`execution_mode`.
Motivo: el launcher compacto no admite detalles internos ni nuevos campos de
negocio, pero el agente director necesita saber los minimos de cierre. La
informacion debe viajar sin proveedor, modelo, HOME, runtime ni DB.
Impacto: cada tarea de director conserva payload pequeno y permitido por el
workflow, y el stack puede construir instrucciones distintas para app completa,
documentacion, analisis o debug.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-22
Decision: El summary del director transporta contexto funcional compacto de
`AppSpecV0`.
Motivo: un smoke real de `crear_app_completa` dejo al director sin objetivo de
producto suficiente y este creo solo una microtarea documental con `CONSULTA AL
DIRECTOR`. El rail era demasiado estrecho: el contrato ya tenia nombre,
objetivo, tipo, plataformas y necesidades de datos.
Impacto: `directorTaskSummaryV0` sigue sin transportar proveedor, modelo, HOME,
runtime ni persistencia concreta, pero anade `app`, `objetivo`, `descripcion`,
`tipo`, `plataformas` y una necesidad funcional compacta. El stack Codex puede
construir un prompt accionable sin tocar el nucleo.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-15
Decision: El director principal en modo normal usa capacidad xhigh y write-set
global documental cuando la peticion exige entregables completos.
Motivo: la prueba real "director total" mostro que Orquesta arrancaba agentes
reales, pero el director quedaba tratado como worker acotado: solo podia crear
`docs/arquitectura.md` y `docs/plan_microtareas.md`. Eso impedia que hiciera
lo que hace un director humano: producir manuales, decisiones, pruebas y
pendientes cuando la peticion lo exige.
Impacto: `primaryDirectorTaskAreaV0` usa xhigh salvo debug y anade
`docs/manual_usuario.md`, `docs/manual_desarrollador.md`,
`docs/manual_sistemas_deploy.md`, `docs/decisiones.md`, `docs/pruebas.md` y
`docs/pendientes.md` para `documentar_app`, `crear_app_completa` y
`planificar_app`. Los directores especializados mantienen write-set pequeno.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-10
Decision: El wizard de intake devuelve preguntas compactas con claves i18n y
solo crea `AppSpecV0` cuando `orquesta-factory` valida el borrador.
Motivo: formularios, API y MCP necesitan una entrada progresiva, pero no deben
duplicar validaciones ni traducir texto visible dentro del caso de uso.
Impacto: los adaptadores pueden presentar la siguiente pregunta y reenviar
respuestas por campo; al completar el contrato se reutiliza el intake de
director existente.
Estado: aceptada localmente.
```
