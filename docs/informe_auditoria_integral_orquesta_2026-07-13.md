# Informe de auditoría integral de Orquesta

Fecha: 2026-07-13
Tipo de cierre: parcial, con fronteras exactas
Revisión base: `7752ca62d1d3d5e8650a90b1297211e3aee579d2`
Árbol inspeccionado: dirty tree concurrente; huella previa al informe
`0548a8961b8663077dd06f23409c04775799bc4ee56bf4aca10af6ec9528a70c`
Orden: [orden_auditoria_integral_diseno_y_bugs_2026-07-13.md](orden_auditoria_integral_diseno_y_bugs_2026-07-13.md)

## 1. Resumen ejecutivo

Sí hay problemas estructurales. Los huecos no son una colección de fallos
independientes. Orquesta mantiene capacidades valiosas y una base de pruebas
amplia, pero su superficie de producto ha crecido más rápido que su modelo de
autoridad y acreditación. El resultado es que registrar una tool, escribir un
mensaje, persistir un estado o pasar un unitario se ha usado varias veces como
prueba de que el efecto productivo completo existe.

No se ha demostrado en esta auditoría que la revisión actual pueda crear una
app real por API/MCP y cerrar todo el ciclo prometido. Tampoco se ha demostrado
lo contrario: existen rutas reales, smokes históricos y llamadas públicas que
alcanzan handlers. El veredicto correcto es **incompleto** hasta ejecutar dos
E2E sobre la misma revisión desplegada, con intención íntegra, credenciales,
intervención, stop selectivo, política de consejo, dos revisiones y cierre
causal.

No se detectó un P0. Se confirman varios P1:

- H0d no entrega la orden al goal: solo la añade a un mailbox.
- 056, stop selectivo preservando otro goal, sigue ausente.
- V1-B, credenciales con propietario y uso por referencia, sigue ausente.
- 057 está bien orientado, pero no está acreditado y omite la guarda contra
  hardlinks en los nuevos stores seguros.
- la identidad de votante del consejo puede proceder del artefacto producido
  por el propio agente, en vez del launch acreditado;
- el shutdown autoriza por palabras contenidas en `requested_by`, una regla
  falsificable y que además rechazó al operador real;
- la composición no llegó a limpiar todos los procesos por su API de shutdown;
- la configuración sigue dispersa a escala de repositorio, aunque el guard
  local de `cmd/orquesta-server` esté verde.

La simplificación aconsejada no consiste en quitar causalidad, CAS, leases,
outbox, workspaces aislados, manifests, credenciales por referencia ni revisión
independiente. Esos mecanismos son necesarios. Hay que simplificar las
autoridades duplicadas, generaciones antiguas, forks por provider, loops de
observación, catálogos de configuración y documentación acumulativa.

La arquitectura objetivo puede expresarse así:

```text
intención pública
    -> manifest canónico inmutable
    -> agregado Goal único
    -> adaptador fino Codex | Claude | Gemini
    -> receipts causales y artefactos
    -> atestaciones independientes
    -> cierre/promoción
```

Todo lo demás debe ser un adaptador o una proyección, no una segunda autoridad.

## 2. Alcance, fotografía y método

La auditoría fue de lectura y verificación; no modificó código. Se preservó el
dirty tree concurrente. Durante la inspección cambió de 40 ficheros tracked
modificados y 15 untracked a una fotografía posterior distinta, por lo que sus
cambios se tratan como trabajo en curso y no como versión acreditada.

Se revisaron las fuentes vigentes, el mapa de generaciones, la matriz de
smokes, la hoja de ruta de conectores, el inventario de bugs, el código de
composición, goal, providers, MCP, shutdown y configuración. Se levantó la
composición local aislada ya existente y se dirigieron cuatro frentes de
auditoría mediante la superficie pública de Orquesta. Tres entregas quedaron
aceptadas; una generó artefacto pero no pudo cerrar por infraestructura de
atestación. Esta última solo se usa como indicio corroborado con lectura local.

No se usó `codebase-memory-mcp`: las consultas eran de strings, Markdown,
configuración y código acotado. Al finalizar no quedaron procesos de esa
herramienta ni `orquesta-server run` del host. El contenedor de auditoría quedó
de nuevo parado.

### 2.1 Magnitud observada

| Superficie | Medición |
|---|---:|
| Paquetes Go | 114 |
| Directorios de módulo | 105 |
| Directorios con nombre de director | 18 |
| Ficheros Go tracked | 3.801 |
| Líneas Go aproximadas | 669.877 |
| Líneas de producción | 352.132 |
| Líneas de tests | 317.745 |
| `cmd/orquesta-server` | 477 ficheros / 99.225 líneas |
| Módulos de director | 77.080 líneas |
| Runtimes Codex/Claude/Gemini | 45.109 líneas |
| Documentos Markdown | 484 / 128.323 líneas |
| Ficheros Go de 500 líneas o más | 188 |
| Ficheros Go de 1.000 líneas o más | 44 |
| Tools MCP públicas | 48 |
| Bindings opcionales de `MCPTransportBindingsV0` | 44 |
| Variables `ORQUESTA_*` contadas por el guard actual | 513 |

Estas cifras no demuestran por sí solas un mal diseño. Sí aumentan mucho el
coste de saber qué ruta manda y de detectar wiring incompleto.

## 3. Mapa arquitectónico top-down

### 3.1 Capas que deben mantenerse

- Contratos y workflow puros: `orquesta-core-workflow`, `orquesta-goal`,
  `orquesta-domain-work` y puertos neutrales.
- Composición: `orquesta-app-codex-stack` y `cmd/orquesta-server`.
- Persistencia concreta: `orquesta-state-file`, `orquesta-run-file` y stores
  locales inyectados.
- Providers: adaptadores Codex, Claude y Gemini.
- Frontera pública: HTTP/MCP y tools tipadas.
- Dominios externos: OPES y otras apps solo detrás de contratos y refs opacas.

### 3.2 Generaciones que hoy compiten

| Generación/ruta | Papel real | Decisión recomendada |
|---|---|---|
| Goal-first | Ruta productiva vigente | Conservar y convertir en única autoridad |
| `legacy_director_loop` | Compatibilidad/mantenimiento | Congelar; cero features; retirar tras paridad |
| Director V2 tick/scheduler | Espina pura/offline | Conservar solo si alimenta Goal o experimentación explícita |
| Plan/Run/WorkflowTask/Wait legacy | Estado histórico y smokes | Convertir en proyecciones o deprecar, no autoridad paralela |

### 3.3 Autoridades actuales y objetivo

| Concepto | Autoridades/representaciones actuales | Autoridad objetivo |
|---|---|---|
| Intención | request pública, GoalSpec compactado, nuevos manifests | manifest canónico inmutable |
| Identidad | refs de request/run/task/goal/agent/provider | identidad de Goal + launches acreditados |
| Workspace | rutas lógicas, índices y routers por provider | asignación única propiedad del Goal |
| Estado | Goal, Run, PlanState, WorkflowTask, waits, colas | agregado Goal; resto proyecciones |
| Evidencia | ACK, delivery, receipts, self-result, tests | receipts tipados ligados a generación/árbol |
| Cierre | cierre de task/run/plan/goal/batch | cierre Goal y promoción derivados causalmente |
| Configuración | JSON, env registry, policies, effective mapping | registro único + snapshot efectivo |

### 3.4 Frontera Docker observada

La composición usada estaba aislada, sin montar OPES, HOME del host ni socket
Docker, con usuario no root y puertos locales. Esto es evidencia positiva de
frontera. No acredita por sí solo lifecycle, secretos o limpieza. La API de
shutdown no consiguió retirar todo el backend compartido y fue necesario parar
el contenedor desde fuera.

## 4. Inventario de capacidades y tools

### 4.1 Tools publicadas

`tools/list` devolvió estas 48 tools:

1. `orquesta.agents.list.v0`
2. `orquesta.app_vcs.v0`
3. `orquesta.apps.arrancar_director.v0`
4. `orquesta.apps.ejecutar_orquestacion.v0`
5. `orquesta.apps.observe_director_goal.v0`
6. `orquesta.apps.preparar_orquestacion.v0`
7. `orquesta.apps.request_change.v0`
8. `orquesta.apps.solicitar_nueva.v0`
9. `orquesta.autoprogramming.observe_active_goals.v0`
10. `orquesta.autoprogramming.observe_goal.v0`
11. `orquesta.autoprogramming.prepare_run.v0`
12. `orquesta.self_improvement.propose.v0`
13. `orquesta.status.v0`
14. `orquesta.supervise.v0`
15. `orquesta.validate_request.v0`
16. `orquesta.codebase.query.v0`
17. `orquesta.codebase.status.v0`
18. `orquesta.core_workflow.handle_command.v0`
19. `orquesta.council.convene.v0`
20. `orquesta.data.profile.v0`
21. `orquesta.director.bootstrap_appspec.v0`
22. `orquesta.director.human_work.review_plan.v0`
23. `orquesta.director.stats.v0`
24. `orquesta.director_agent.apply_decision.v0`
25. `orquesta.director_supervisor.briefing.v0`
26. `orquesta.document.text.extract.v0`
27. `orquesta.document_plan.expand.v0`
28. `orquesta.domain_work.v0`
29. `orquesta.external_work.dry_run.v0`
30. `orquesta.external_work.run.v0`
31. `orquesta.nueva_app.wizard.bot.v0`
32. `orquesta.nueva_app.wizard.v0`
33. `orquesta.workspace_timeline.v0`
34. `orquesta.operator.command.v0`
35. `orquesta.operator.directed_query.v0`
36. `orquesta.operator.director.message.v0`
37. `orquesta.operator.outbox.pending.v0`
38. `orquesta.operator.status.query.v0`
39. `orquesta.operator.supervised_burst.v0`
40. `orquesta.projects.list.v0`
41. `orquesta.run_queue.priority.v0`
42. `orquesta.runs.control.v0`
43. `orquesta.runs.supervisor.v0`
44. `orquesta.runtime.models.v0`
45. `orquesta.server.shutdown.v0`
46. `orquesta.status.v1`
47. `orquesta.tasks.list.v0`
48. `orquesta.tool.capabilities.list.v0`

### 4.2 Estado funcional por grupo

| Grupo | Declarado | Wiring/efecto observado | Acreditación |
|---|---:|---|---|
| Goal-first/autoprogramación | Sí | Goals reales y observación | Parcial; 057 y cierre final pendientes |
| Apps/director legacy | Sí | Handler vivo, exige modo legacy | Compatibilidad, no promesa productiva |
| Operador/mailbox | Sí | Append y ACK sintetizado | Contradicho como delivery H0d |
| Consejo/review | Sí | Código y stores presentes | Identidad/política/E2E incompletos |
| Domain/external work | Sí | Validación y acciones vivas | Parcial; hay drift de schema MCP |
| Runtime/modelos | Sí | Handler consultado | Respuesta sin modelos en composición auditada |
| Shutdown | Sí | Handler vivo | Auth defectuosa y cleanup incompleto |
| Catálogo de capacidades | Sí | Handler devuelve lista vacía | No sirve aún como verdad productiva |
| Estado/listados/colas | Sí | Proyecciones vivas | Mezclan historial/terminal con “activos” |
| Resto de utilidades | Sí | No se mutaron una a una | Evidencia indirecta; no acreditadas por registro |

### 4.3 Las seis tools disputadas

| Tool | Resultado público | Veredicto |
|---|---|---|
| `apps.solicitar_nueva` | Llegó al validador y devolvió campos requeridos | Cableada/ejecutada; efecto completo no acreditado |
| `apps.ejecutar_orquestacion` | Devolvió `legacy_director_loop_required` | Viva, pero solo preview legacy explícito |
| `director_agent.apply_decision` | Validador tipado rechazó campos inválidos | Cableada; mutación productiva no acreditada |
| `domain_work` | `create_job` alcanzó validación; acción externa evaluable existe | Viva; schema público incompleto |
| `runtime.models` | `ok`, provider Ollama, sin modelos | Cableada; disponibilidad de modelos no acreditada |
| `tool.capabilities.list` | `ok`, catálogo vacío | Cableada sin datos útiles en esta composición |

No deben llamarse “muertas”, pero tampoco “productivas” solo porque respondan.

## 5. Matriz requisito → evidencia

| ID | Requisito | Estado previo | Evidencia observada | Veredicto |
|---|---|---|---|---|
| CAP-APP-01 | Crear app completa por API/MCP | Afirmado | Faltan dos E2E actuales y completos | incompleto |
| H0A-01 | Goal real aislado | Acreditado histórico | Composición real y goals de auditoría | evidencia indirecta |
| H0B-01 | Binding público cableado | Acreditado con mutación | 48 tools y seis llamadas; no mutación actual por grupo | evidencia indirecta |
| H0C-01 | Delivery→review→closure causal | Acreditado histórico | Goals aceptados; no mismo E2E final de app | evidencia indirecta |
| H0D-01 | Operador→goal entregado | Declarado cerrado/revocado | Store append sin consumidor/claim/delivery | contradicho |
| 056-01 | Stop selectivo preserva otro goal | Pendiente | El adaptador exige 056 para preservar artefactos | evidencia ausente |
| 057-01 | Intención íntegra e inmutable | En curso | Manifest/routers nuevos; hardlink y E2E faltan | incompleto |
| 057-02 | Mismo workspace en lifecycle | En curso | Índices/routers por provider y focales verdes | incompleto |
| 057-03 | Codex/Claude/Gemini equivalentes | En curso | Implementaciones separadas, no E2E común | incompleto |
| 058-01 | Consejo `auto|required|skip` | Pendiente | Código parcial; política/receipt de bypass incompletos | incompleto |
| REV-02 | Dos reviews independientes | Parcial | Fakes y stores; no dos launches reales ligados a commit | incompleto |
| V1B-01 | Credencial con dueño | Pendiente | Campos `credential_ref`, no store/uso/revocación E2E | evidencia ausente |
| CFG-01 | Una superficie canónica | Declarado en reglas | Registry local, 513 nombres y lecturas fuera del server | contradicho |
| CFG-02 | Snapshot efectivo/redactado | Parcial | Código efectivo extenso; no autoridad única repo-wide | incompleto |
| CFG-03 | Escritura gobernada | Pendiente V1-A | JSON de runtime esencialmente read-only | evidencia ausente |
| OPS-CPU-01 | Idle sin polling inútil | En curso | CPU cuasi-idle 28–91%; dirty incluye backoff | incompleto |
| OPS-SD-01 | Shutdown limpia composición | Afirmado | `backend_still_running`; parada externa necesaria | contradicho |
| SEC-AUTH-01 | Auth por identidad acreditada | Implícito | `requested_by` se decide por substrings | contradicho |
| SEC-FILE-01 | Stores resisten symlink/FIFO/hardlink | En curso | Symlink/FIFO cubiertos; `Nlink == 1` ausente | incompleto |
| TOOL-01 | `apps.solicitar_nueva` viva | Disputado | Handler/validador alcanzado | probado para wiring |
| TOOL-02 | `apps.ejecutar_orquestacion` viva | Disputado | Handler legacy alcanzado | probado para wiring |
| TOOL-03 | `apply_decision` viva | Disputado | Handler/validador alcanzado | probado para wiring |
| TOOL-04 | `domain_work` viva | Disputado | Handler vivo; schema no expone acción válida | incompleto |
| TOOL-05 | `runtime.models` viva | Disputado | Handler vivo, lista vacía | probado para wiring |
| TOOL-06 | catálogo de tools útil | Disputado | Handler vivo, catálogo vacío | incompleto |
| DOC-01 | Fuente vigente no contradictoria | Implícito | H0d figura abierto y cerrado en la misma hoja | contradicho |
| OPS-ACT-01 | “Activos” solo devuelve activos | Implícito | Respuesta incluye terminales/invalid/blocked históricos | contradicho |

## 6. Catálogo de hallazgos

### [AUD-001] Conviven varias autoridades para intención, estado y cierre

- **Clasificación primaria:** diseño sistémico
- **Clasificación secundaria:** sobreprogramación
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** HEAD `7752ca6`, dirty tree concurrente
- **Superficies afectadas:** goal-first, director legacy, V2, stores, estado público
- **Providers/estados afectados:** todos; todo el lifecycle
- **Promesa o invariante:** una decisión productiva tiene una autoridad inequívoca
- **Comportamiento observado:** Goal, Run, PlanState, WorkflowTask, waits, colas y batches pueden representar vida y cierre
- **Reproducción mínima:** inventariar tipos/stores y seguir sus factories desde `cmd/orquesta-server`
- **Evidencia positiva:** las rutas poseen pruebas y smokes propios
- **Prueba negativa o mutación:** falta una prueba que impida que dos autoridades discrepen tras replay
- **Causa inmediata:** generaciones preservadas como rutas activas
- **Causa raíz:** evolución acumulativa sin presupuesto de autoridades ni sunset ejecutado
- **Por qué no es otra categoría:** no se arregla en un handler aislado
- **Radio de impacto:** repositorio y operación completos
- **Riesgo de seguridad/datos:** cierre o acción sobre generación/workspace equivocados
- **Alternativas existentes y solape:** goal-first ya puede ser la autoridad principal
- **Corrección mínima que restaura el invariante:** impedir features nuevas en legacy y etiquetar cada proyección
- **Corrección estructural recomendada:** Goal único; legacy solo adaptador temporal
- **Tests que impedirían regresión:** replay cruzado que exige una sola generación/estado final
- **Dependencias/orden causal:** estabilizar 057 antes de migrar intención extensa
- **Criterio verificable de cierre:** ningún flujo productivo escribe dos autoridades de lifecycle

### [AUD-002] H0d acredita append como si fuera entrega al goal

- **Clasificación primaria:** falsa acreditación
- **Clasificación secundaria:** capacidad ausente
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** reproducido
- **Revisión y árbol:** imagen local basada en commit anterior y código HEAD
- **Superficies afectadas:** `operator.director.message`, mailbox, intervención en vuelo
- **Providers/estados afectados:** goals activos y backends no interactivos
- **Promesa o invariante:** una orden aceptada debe alcanzar el turn o successor y producir ACK causal
- **Comportamiento observado:** se añade JSONL y se sintetizan status/ack/response; no existe reader/claim/lease/delivery
- **Reproducción mínima:** llamar la tool y revisar `operator_director_mailbox_v0.go`
- **Evidencia positiva:** el mensaje queda durable
- **Prueba negativa o mutación:** no hay prompt/turn del goal que contenga la instrucción
- **Causa inmediata:** productor sin consumidor
- **Causa raíz:** confusión entre admisión, persistencia y efecto
- **Por qué no es otra categoría:** la documentación cerró una capacidad que no existe completa
- **Radio de impacto:** toda corrección del operador en vuelo
- **Riesgo de seguridad/datos:** falsa sensación de control operacional
- **Alternativas existentes y solape:** turn interactivo o successor/bundle según backend
- **Corrección mínima que restaura el invariante:** consumer con claim/lease, generación y ACK de delivery
- **Corrección estructural recomendada:** puerto neutral de intervención integrado en Goal
- **Tests que impedirían regresión:** tools/call→prompt→receipt, race, restart y replay
- **Dependencias/orden causal:** después de 057; antes de E2E final
- **Criterio verificable de cierre:** el agente activo recibe una marca única y replay no redelivery

### [AUD-003] 057 no acredita todavía la intención/workspace y omite hardlinks

- **Clasificación primaria:** regresión
- **Clasificación secundaria:** capacidad incompleta
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** dirty tree 057 no integrado
- **Superficies afectadas:** manifest, store seguro, workspace routers
- **Providers/estados afectados:** Codex, Claude y Gemini; launch/observe/attest
- **Promesa o invariante:** intención completa e inmutable y mismo workspace físico
- **Comportamiento observado:** el manifest conserva la request y hay checks de symlink/FIFO/permisos, pero la lectura no exige `Nlink == 1`
- **Reproducción mínima:** revisar `readIntentManifestAtV0` y tests de hardlink
- **Evidencia positiva:** focales y carreras pasan; captura precede al compactado
- **Prueba negativa o mutación:** falta hardlink y un E2E común de 20 invariantes para tres providers
- **Causa inmediata:** primitiva segura replicada/incompleta
- **Causa raíz:** seguridad de filesystem implementada por frente/provider, no como contrato compartido
- **Por qué no es otra categoría:** repara una pérdida real, pero aún no restaura todo el invariante
- **Radio de impacto:** todos los goals nuevos con workspace materializado
- **Riesgo de seguridad/datos:** sustitución/alias de artefacto local y atestación sobre árbol incorrecto
- **Alternativas existentes y solape:** consolidar file store seguro común
- **Corrección mínima que restaura el invariante:** validar link count y añadir ataques negativos
- **Corrección estructural recomendada:** store neutral compartido; routers provider finos
- **Tests que impedirían regresión:** matriz 057 completa y misma suite parametrizada por provider
- **Dependencias/orden causal:** primero en la cola funcional
- **Criterio verificable de cierre:** 20 invariantes verdes en la misma revisión desplegada

### [AUD-004] No existe stop selectivo acreditado que preserve otro goal

- **Clasificación primaria:** capacidad ausente
- **Clasificación secundaria:** deuda operativa
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** HEAD y dirty 057
- **Superficies afectadas:** lifecycle, stop, cleanup y preservación de artefactos
- **Providers/estados afectados:** recursos compartidos; goals concurrentes
- **Promesa o invariante:** Stop(B) no debe interrumpir Observe(A)
- **Comportamiento observado:** preservar artefactos falla cerrado con `requires_056_selective_stop_receipt`
- **Reproducción mínima:** activar la ruta de preservación sin receipt 056
- **Evidencia positiva:** el fallo cerrado evita afirmar una capacidad inexistente
- **Prueba negativa o mutación:** no hay E2E concurrente A/B
- **Causa inmediata:** falta puerto/receipt selectivo completo
- **Causa raíz:** lifecycle compartido evolucionó antes que ownership por goal
- **Por qué no es otra categoría:** no es regresión; sigue pendiente explícitamente
- **Radio de impacto:** toda ejecución concurrente y shutdown parcial
- **Riesgo de seguridad/datos:** pérdida de trabajo o residuos
- **Alternativas existentes y solape:** stop global no es alternativa equivalente
- **Corrección mínima que restaura el invariante:** ownership y stop por goal/generación
- **Corrección estructural recomendada:** process registry único asociado al agregado Goal
- **Tests que impedirían regresión:** Stop(B)+Observe(A), restart, idempotencia y cleanup
- **Dependencias/orden causal:** tras H0d/057; antes de credenciales y E2E final
- **Criterio verificable de cierre:** B termina, A sigue observable y el backend común continúa sano

### [AUD-005] V1-B declara referencias de credencial sin capacidad completa de credenciales

- **Clasificación primaria:** capacidad ausente
- **Clasificación secundaria:** seguridad
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** HEAD/dirty actual
- **Superficies afectadas:** configuración, providers, attestation, multiusuario
- **Providers/estados afectados:** todos
- **Promesa o invariante:** secreto fuera del Goal y uso autorizado por propietario/alcance
- **Comportamiento observado:** existen campos de referencia, no store/resolve/use/revoke E2E con receipt
- **Reproducción mínima:** trazar `credential_ref` hasta resolución y uso real
- **Evidencia positiva:** el contrato evita pedir el secreto como campo ordinario
- **Prueba negativa o mutación:** no hay owner mismatch, revocación ni leak test productivo
- **Causa inmediata:** modelo de datos adelantado a la implementación
- **Causa raíz:** declaración confundida con capacidad efectiva
- **Por qué no es otra categoría:** no se observó pérdida; falta el servicio prometido
- **Radio de impacto:** multiusuario y proveedores con secretos
- **Riesgo de seguridad/datos:** uso cruzado o propagación accidental de secretos
- **Alternativas existentes y solape:** secretos de proceso sirven para bootstrap, no para ownership
- **Corrección mínima que restaura el invariante:** store privado, resolver autorizado y receipt redacted
- **Corrección estructural recomendada:** puerto de credenciales por app/composición
- **Tests que impedirían regresión:** owner/scope/revoke/redaction/replay
- **Dependencias/orden causal:** después de 056; antes de UI V1-A y E2E final
- **Criterio verificable de cierre:** un goal usa el secreto sin contenerlo y otro propietario es rechazado

### [AUD-006] Consejo y doble review no prueban identidad independiente real

- **Clasificación primaria:** falsa acreditación
- **Clasificación secundaria:** capacidad incompleta
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** HEAD
- **Superficies afectadas:** council, voter source, final review
- **Providers/estados afectados:** agentes revisores y estados de promoción
- **Promesa o invariante:** votos/reviews de launches distintos ligados al mismo árbol
- **Comportamiento observado:** `voter_ref` puede leerse del JSON producido; tests de doble review usan refs fake
- **Reproducción mínima:** inspeccionar `resident_director_council_vote_source_v0.go`
- **Evidencia positiva:** stores, policy parcial y gates existen
- **Prueba negativa o mutación:** un artefacto puede declarar la identidad invitada sin prueba del launch
- **Causa inmediata:** identidad aceptada desde contenido no confiable
- **Causa raíz:** receipts no son aún la única fuente de identidad/evidencia
- **Por qué no es otra categoría:** hay infraestructura, pero no acredita la promesa
- **Radio de impacto:** consejo y promoción de apps
- **Riesgo de seguridad/datos:** suplantación de independencia
- **Alternativas existentes y solape:** identidad de launch/ACK ya puede ser fuente
- **Corrección mínima que restaura el invariante:** derivar voter/reviewer del receipt firmado por runtime
- **Corrección estructural recomendada:** evidence envelope común para consejo y review
- **Tests que impedirían regresión:** spoof negativo y dos launches reales sobre mismo commit
- **Dependencias/orden causal:** V1-B para identidad/credenciales; consejo no bloquea doble review
- **Criterio verificable de cierre:** política `auto|required|skip` y dos reviews reales con receipts distintos

### [AUD-007] Shutdown decide autorización por palabras controladas por el caller

- **Clasificación primaria:** bug puntual
- **Clasificación secundaria:** seguridad
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** reproducido
- **Revisión y árbol:** imagen local y HEAD
- **Superficies afectadas:** `server.shutdown`, auth de operador
- **Providers/estados afectados:** servidor y todos los backends residentes
- **Promesa o invariante:** autorización basada en identidad acreditada
- **Comportamiento observado:** el operador real fue rechazado; un `requested_by` con “director” fue aceptado
- **Reproducción mínima:** variar solo `requested_by` en la misma llamada autenticada
- **Evidencia positiva:** existe un gate de autorización
- **Prueba negativa o mutación:** una palabra elegida por el caller cambia la decisión
- **Causa inmediata:** matching de substrings en `auth_v0.go`
- **Causa raíz:** identidad operativa no propagada al puerto de shutdown
- **Por qué no es otra categoría:** el defecto se localiza en una decisión concreta
- **Radio de impacto:** disponibilidad e integridad del servicio
- **Riesgo de seguridad/datos:** denegación de servicio o bloqueo del operador legítimo
- **Alternativas existentes y solape:** principal autenticado/rol tipado de la composición
- **Corrección mínima que restaura el invariante:** ignorar `requested_by` para auth y usar principal/claims
- **Corrección estructural recomendada:** contexto de autorización común a tools con efectos
- **Tests que impedirían regresión:** spoof, operador válido y ausencia de rol
- **Dependencias/orden causal:** corregible en paralelo, integración serial si comparte `cmd`
- **Criterio verificable de cierre:** texto libre nunca concede ni retira permiso

### [AUD-008] La composición mantiene trabajo/residuos y no cierra por API

- **Clasificación primaria:** deuda operativa
- **Clasificación secundaria:** regresión
- **Severidad:** P1
- **Confianza:** media-alta
- **Estado:** reproducido
- **Revisión y árbol:** imagen base anterior; dirty tree contiene mitigaciones no desplegadas
- **Superficies afectadas:** supervisor, observers, app-server compartido, shutdown
- **Providers/estados afectados:** goals terminales/stale y backend Codex
- **Promesa o invariante:** sin trabajo activo, CPU baja; shutdown alcanza `shutdown_ready`
- **Comportamiento observado:** CPU cuasi-idle 28–91%, terminales en “active”; `backend_still_running`
- **Reproducción mínima:** levantar estado retenido, observar goals y solicitar shutdown público
- **Evidencia positiva:** bajo carga de cuatro goals se usaron varios cores, comportamiento esperado
- **Prueba negativa o mutación:** dirty backoff/coalescing tiene focales, no before/after desplegado
- **Causa inmediata:** loops/timers y reconciliación de estado stale; backend compartido no retirado
- **Causa raíz:** múltiples observadores y fuentes de lifecycle sin ownership único
- **Por qué no es otra categoría:** hay defecto operativo real y una raíz arquitectónica más amplia
- **Radio de impacto:** CPU, shutdown, despliegues y diagnóstico
- **Riesgo de seguridad/datos:** procesos huérfanos y acciones sobre estado antiguo
- **Alternativas existentes y solape:** wakeups/coalescing ya existen en dirty tree
- **Corrección mínima que restaura el invariante:** filtrar terminales, backoff y cleanup gobernado
- **Corrección estructural recomendada:** scheduler/observer residente único y dirigido por eventos
- **Tests que impedirían regresión:** perfil 10 min idle, stale restart y shutdown sin fallback
- **Dependencias/orden causal:** medir tras integrar 057/idle; 056 mejora ownership
- **Criterio verificable de cierre:** umbral CPU acordado y cero procesos propios tras API shutdown

### [AUD-009] El schema MCP de DomainWork omite una acción soportada

- **Clasificación primaria:** bug puntual
- **Clasificación secundaria:** falsa acreditación de catálogo
- **Severidad:** P2
- **Confianza:** alta
- **Estado:** reproducido
- **Revisión y árbol:** HEAD
- **Superficies afectadas:** schema MCP, `domain_work`, clientes estrictos
- **Providers/estados afectados:** cualquier consumidor MCP
- **Promesa o invariante:** schema público y handler aceptan las mismas acciones
- **Comportamiento observado:** handler soporta `evaluate_external_capabilities`; enum publica solo `create_job` y `submit_artifact`
- **Reproducción mínima:** comparar descripción/dispatcher con `mcp_transport_tool_input_schema_v0.go`
- **Evidencia positiva:** llamada directa sin validación cliente alcanzó la acción
- **Prueba negativa o mutación:** un cliente que aplica el schema la rechaza antes de enviarla
- **Causa inmediata:** schema manual desincronizado
- **Causa raíz:** declaración y capacidad no se generan desde un registro único
- **Por qué no es otra categoría:** es drift localizado con causa repetible
- **Radio de impacto:** interoperabilidad pública
- **Riesgo de seguridad/datos:** bajo; fallo de disponibilidad funcional
- **Alternativas existentes y solape:** generar schema/dispatcher/catálogo desde capability spec
- **Corrección mínima que restaura el invariante:** añadir acción y test de paridad
- **Corrección estructural recomendada:** registro único con estados declarada/wired/acreditada
- **Tests que impedirían regresión:** enumerar acciones y ejecutar fixture por cada una
- **Dependencias/orden causal:** independiente; evitar pisar 057 en integración
- **Criterio verificable de cierre:** ninguna acción del handler queda fuera del schema

### [AUD-010] La configuración está centralizada solo de forma parcial

- **Clasificación primaria:** diseño sistémico
- **Clasificación secundaria:** sobreprogramación
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** HEAD/dirty actual
- **Superficies afectadas:** server, guardian, runtimes, rails, providers
- **Providers/estados afectados:** startup, procesos hijos, configuración efectiva
- **Promesa o invariante:** una clave se declara, valida y resuelve una sola vez
- **Comportamiento observado:** guard local verde y `orquesta-config` existen, pero hay 172 call-sites de lectura de env en producción y 513 nombres contados
- **Reproducción mínima:** AST/`rg` repo-wide de `os.Getenv`, `os.LookupEnv` y catálogo
- **Evidencia positiva:** loader tipado, JSON, revision y effective config ya están iniciados
- **Prueba negativa o mutación:** el guard no cubre todo el repo y no demuestra precedencia/proyección única
- **Causa inmediata:** cada composición/adaptador conservó su bootstrap propio
- **Causa raíz:** centralización por capas paralelas, no por pipeline de resolución
- **Por qué no es otra categoría:** mover unas lecturas no elimina metadata y defaults duplicados
- **Radio de impacto:** todo despliegue y diagnóstico
- **Riesgo de seguridad/datos:** secretos en superficies incorrectas y configuración efectiva impredecible
- **Alternativas existentes y solape:** `orquesta-config`, registry y effective config son base reutilizable
- **Corrección mínima que restaura el invariante:** un ingress por ejecutable y guard repo-wide decreciente
- **Corrección estructural recomendada:** tres superficies: config pública, store privado y snapshot efectivo generado
- **Tests que impedirían regresión:** unknown/precedence/redaction/restart/daemon-equivalence
- **Dependencias/orden causal:** fundación paralelizable; V1-B antes de escritura UI
- **Criterio verificable de cierre:** cero lecturas ad hoc fuera de loaders canónicos

### [AUD-011] La seguridad de workspace está duplicada por provider

- **Clasificación primaria:** sobreprogramación
- **Clasificación secundaria:** diseño sistémico
- **Severidad:** P2
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** dirty 057
- **Superficies afectadas:** secure file y routers Claude/Gemini/Codex
- **Providers/estados afectados:** tres providers
- **Promesa o invariante:** mismo contrato neutral y mismas defensas
- **Comportamiento observado:** Claude/Gemini suman 714 líneas casi gemelas; similitud normalizada 0,913, con diferencias de validación/error
- **Reproducción mínima:** comparar las dos implementaciones secure-file normalizadas
- **Evidencia positiva:** cada provider obtiene cobertura focal
- **Prueba negativa o mutación:** hardlink falta en ambos y el drift ya es visible
- **Causa inmediata:** copy/fork de primitiva transversal
- **Causa raíz:** límites de adapter definidos demasiado arriba
- **Por qué no es otra categoría:** el código funciona en focales, pero multiplica coste/riesgo
- **Radio de impacto:** toda evolución de seguridad filesystem
- **Riesgo de seguridad/datos:** parche aplicado a un provider y omitido en otro
- **Alternativas existentes y solape:** store/validator neutral compartido
- **Corrección mínima que restaura el invariante:** extraer checks y suite parametrizada
- **Corrección estructural recomendada:** provider solo traduce launch/observe/stop
- **Tests que impedirían regresión:** misma batería de ataques para los tres adapters
- **Dependencias/orden causal:** consolidar después de acreditar el comportamiento 057
- **Criterio verificable de cierre:** una sola implementación de primitivas seguras

### [AUD-012] La documentación vigente contiene cierres contradictorios

- **Clasificación primaria:** falsa acreditación
- **Clasificación secundaria:** deuda operativa
- **Severidad:** P2
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** docs del 2026-07-12/13
- **Superficies afectadas:** hoja de ruta, rolling log, decisiones de agentes
- **Providers/estados afectados:** planificación completa
- **Promesa o invariante:** una fuente vigente determina estado y evidencia
- **Comportamiento observado:** H0d se reabre al inicio y aparece cerrado después en la misma hoja
- **Reproducción mínima:** leer las secciones iniciales y de cierre de `hoja_ruta_cierre_conectores_2026-07-12.md`
- **Evidencia positiva:** los documentos conservan historial útil
- **Prueba negativa o mutación:** un lector por secciones obtiene dos estados opuestos
- **Causa inmediata:** documentos append-only usados también como snapshot vigente
- **Causa raíz:** no hay ledger generado de capacidades/evidencias
- **Por qué no es otra categoría:** el texto sostuvo una acreditación falsa, no es solo estilo
- **Radio de impacto:** cualquier agente que seleccione backlog o declare cierre
- **Riesgo de seguridad/datos:** indirecto; trabajo omitido o repetido
- **Alternativas existentes y solape:** inventario y matriz pueden ser ledger único
- **Corrección mínima que restaura el invariante:** cabecera vigente inequívoca y enlaces históricos
- **Corrección estructural recomendada:** estado generado desde receipts de acreditación
- **Tests que impedirían regresión:** linter de estados duplicados/refs sin evidencia
- **Dependencias/orden causal:** inmediata y paralela
- **Criterio verificable de cierre:** una consulta produce un estado único por capacidad/revisión

### [AUD-013] Falta la acreditación final de creación de app

- **Clasificación primaria:** capacidad ausente
- **Clasificación secundaria:** falsa acreditación histórica
- **Severidad:** P1
- **Confianza:** alta
- **Estado:** demostrado estáticamente
- **Revisión y árbol:** revisión actual no desplegada como release acreditada
- **Superficies afectadas:** API/MCP, nueva app, providers, reviews, cierre
- **Providers/estados afectados:** al menos dos políticas de consejo y todos los gates prometidos
- **Promesa o invariante:** crear dos apps reales hasta closure en la misma revisión
- **Comportamiento observado:** hay smokes parciales/históricos, no los dos E2E finales pedidos
- **Reproducción mínima:** buscar receipts de ambos escenarios ligados al mismo commit/image
- **Evidencia positiva:** Goal-first y varias piezas ejecutan trabajo real
- **Prueba negativa o mutación:** faltan escenarios `skip_by_operator` y `required` completos
- **Causa inmediata:** la acreditación fue por frentes, no por producto integrado
- **Causa raíz:** cierre de checklist estrecho extrapolado a producto completo
- **Por qué no es otra categoría:** la capacidad puede existir parcialmente; lo ausente es su prueba integral prometida
- **Radio de impacto:** afirmación comercial/operativa central
- **Riesgo de seguridad/datos:** ejecutar sin garantías integrales
- **Alternativas existentes y solape:** reutilizar todos los smokes focales como preparación
- **Corrección mínima que restaura el invariante:** ejecutar los dos E2E tras cerrar P1 previos
- **Corrección estructural recomendada:** release gate generado por capability ledger
- **Tests que impedirían regresión:** ambos E2E y mutaciones críticas sobre imagen inmutable
- **Dependencias/orden causal:** último tramo
- **Criterio verificable de cierre:** dos receipts finales revisados independientemente y reproducibles

## 7. Causas raíz transversales

### 7.1 Declaración, admisión, efecto y acreditación no son estados separados

H0d y los bindings/catálogos comparten una causa demostrada: se ha promovido
como capacidad algo que solo estaba registrado, cableado o persistido. La
solución es un ledger con cinco estados explícitos:

```text
declarada -> implementada -> cableada -> ejercitada -> acreditada
```

Solo `acreditada` debe alimentar promesas de producto.

### 7.2 La intención se compactó antes de tener una autoridad durable

057 tiene una causa inmediata diferente: una proyección compacta se convirtió
en sustituto de la request original. Comparte con H0d la debilidad del proceso
de acreditación —faltaba un E2E que probase el efecto—, pero no el mismo bug de
código.

### 7.3 Las generaciones se añadieron sin retirar autoridad a las anteriores

El objetivo de compatibilidad fue razonable, pero cada generación retuvo
estado, loops, factories y documentación. La migración debe medir y reducir
autoridades, no solo añadir rutas verdes.

### 7.4 Las preocupaciones transversales se implementaron dentro de providers

Filesystem seguro, workspace, prompts, config y lifecycle se bifurcan. Esto
produce muchos tests y líneas sin aumentar capacidad de producto en la misma
proporción, y facilita drift de seguridad.

### 7.5 La documentación acumulativa sustituyó al estado generado

El rolling log conserva historia pero no puede ser simultáneamente la foto
actual. Las contradicciones de H0d muestran que los agentes necesitan una
fuente de verdad más pequeña y verificable.

## 8. Falsas acreditaciones

| Afirmación | Evidencia usada | Por qué no alcanza | Estado correcto |
|---|---|---|---|
| H0d cerrado | append + ACK sintetizado | no existe consumidor ni turn | revocado |
| Tool productiva | aparece en `tools/list` o devuelve 200 | no demuestra efecto | wiring probado, efecto pendiente |
| Consejo independiente | dos refs/respuestas | identidad puede venir del artefacto/fake | incompleto |
| 057 seguro | focales filesystem verdes | no atacan hardlink ni E2E común | en curso |
| Idle corregido | tests de backoff | cambio dirty no desplegado/medido | en curso |
| Config centralizada | guard local verde | scope no cubre repo ni guardian/providers | parcial |
| Producto al 99 % | checklist histórico estrecho | no incluía cola actual ni E2E final | no usar porcentaje |

## 9. Pruebas, llamadas y mutaciones ejecutadas

### 9.1 Pruebas locales sobre el dirty tree

Pasaron:

- `git diff --check`;
- guard focal `TestEnvVarsBudgetMEJ106V0` —informó 513 nombres;
- paquetes `orquesta-autoprogramming`, `orquesta-goal`, runtimes Goal Codex,
  Claude y Gemini;
- `orquesta-mcp`, `orquesta-server`, `orquesta-app-codex-stack`;
- `cmd/orquesta-server` en 71,580 s;
- carreras focales con `-race -count=3` para manifest/workspace, providers y
  coalescing/idle.

No se usa `go test ./...` como acreditación global de este informe: no quedó
una salida final completa y el propio mandato exige E2E/mutaciones específicas.

### 9.2 Evidencia pública

- `tools/list`: 48 tools.
- Seis llamadas disputadas alcanzaron sus handlers; sus efectos se califican
  por separado en §4.3.
- Cuatro goals de auditoría se lanzaron por
  `orquesta.autoprogramming.prepare_run.v0`; tres cerraron aceptados y uno
  quedó sin cierre de atestación.
- `observe_active_goals` incluyó historial terminal/invalid/blocked.
- shutdown público reprodujo autorización defectuosa y cleanup incompleto.

### 9.3 Mutaciones todavía necesarias

1. H0d: quitar el consumer debe impedir receipt de delivery.
2. 057: hardlink/symlink/FIFO/collision/restart y cambio de workspace deben
   dejar el gate rojo en los tres providers.
3. 056: Stop(B) no puede alterar generation/PID/observación de A.
4. V1-B: owner, scope o revocación incorrectos deben impedir uso.
5. Consejo/review: identidad declarada en contenido no puede sustituir receipt.
6. MCP: cualquier acción del handler ausente del schema debe fallar el build.
7. Config: lectura `os.Getenv` nueva fuera del loader debe fallar el guard.
8. Shutdown: texto libre nunca modifica autorización.
9. E2E app: retirar cualquiera de dos reviews debe impedir promoción.

## 10. Operación, CPU y residuos

| Momento | CPU observada | Interpretación |
|---|---:|---|
| Arranque previo a auditoría | hasta 165 % | reconciliación/estado retenido |
| Cuatro goals activos | alrededor de 402 % | carga esperable |
| Tras goals terminales | 28–91 %, luego ~27 % | cuasi-idle con estado stale; demasiado alto |

La medida cuasi-idle no es una prueba limpia de “cero trabajo”: el shutdown
descubrió rework antiguo y app-server compartido vivo. Precisamente por eso es
un hallazgo operativo: la superficie “activos” y el lifecycle no permiten
distinguir o retirar limpiamente ese residuo. Los cambios dirty de coalescing y
cadencia son razonables, pero necesitan un before/after desplegado durante una
ventana fija, con perfiles y lista de procesos.

## 11. Sobreprogramación y simplificación

| Frente | Necesidad real | Solape/coste | Decisión |
|---|---|---|---|
| Manifest 057 | Evitar pérdida de intención | Justificado | conservar y completar |
| Secure files por provider | Evitar ataques filesystem | 714 líneas casi duplicadas | consolidar después de acreditar |
| Routers/índices por provider | Mismo workspace | riesgo de varias autoridades | consolidar alrededor de Goal |
| Goal-first | Ejecución productiva | ruta vigente | conservar |
| Director legacy | Compatibilidad | duplica estado/loops | congelar y deprecar |
| V2 offline | Núcleo puro experimental | puede ser cuarta ruta | mantener fuera de promesa o integrar en Goal |
| 48 tools/44 bindings | Superficie pública amplia | wiring/schema manual | generar desde capability registry |
| Loops residentes | Observación/reconciliación | timers y proyecciones repetidas | unificar por eventos/wakeup |
| Config registry/policies/effective | Tipado y auditoría | metadata repartida en miles de líneas | un registro de specs + snapshot |
| Docs rolling | Historia | contradice snapshot | archivar como log; generar estado vigente |

El dirty 057 añade al menos 2.619 líneas en sus piezas principales. No es razón
para descartarlo: corrige una pérdida real. Sí obliga a consolidar después de
demostrar el contrato, especialmente las dos implementaciones secure-file con
91 % de similitud. Primero se fija el comportamiento; después se extrae la
primitiva común sin cambiar receipts.

## 12. Configuración y variables de entorno

### 12.1 Qué existe

`orquesta-config`, el loader tipado, `orquesta.config.json`, los registries y
`effective_config` son una base útil. El problema es que la centralización se
ha hecho añadiendo capas, mientras adaptadores y ejecutables siguen leyendo env
por su cuenta.

Se contaron 172 líneas de llamadas directas de entorno en producción:

| Área | Call-sites aproximados |
|---|---:|
| `cmd/orquesta-server` | 127 en 38 ficheros |
| `cmd/orquesta-guardian` | 34 en 4 ficheros |
| runtime Codex app-server | 5 |
| rails | 4 |
| runtime worktree | 1 |
| runtime Codex delivery | 1 |

El guard actual permite 513 nombres y su baseline AST se concentra en el
server. Sirve de ratchet local, no demuestra cumplimiento repo-wide.

### 12.2 Los tres ficheros/superficies correctos

La tercera pieza que probablemente se recordaba es la **configuración
efectiva**. No conviene crear tres autoridades editables ni dos `.env` de
strings. Conviene tener estas tres superficies:

1. **`orquesta.config.json`**: configuración no secreta, tipada y editable.
   Puede contener `credential_ref`, nunca el secreto.
2. **`orquesta.credentials.json` o store privado equivalente**: material
   secreto, modo `0600`, fuera de Git, con `credential_ref`, provider, tipo,
   `owner_ref`, scope, rotación y revocación. En producción puede sustituirse
   por un secret manager detrás del mismo puerto.
3. **`effective_config.json`**: snapshot generado, inmutable y redactado de lo
   realmente aplicado. Incluye origen `file|env|default`, revisión/hash y si
   cada cambio exige restart. Nunca se edita a mano.

Los nombres son recomendados; el contrato importa más que el nombre. Los tres
no son equivalentes: solo el primero es configuración ordinaria editable, el
segundo es un almacén restringido y el tercero es evidencia generada.

### 12.3 Pipeline único

```text
ConfigKeySpec registry
       + orquesta.config.json
       + env de bootstrap/aliases temporales
       + CredentialStore
       -> LoadConfig(...)
       -> ResolvedConfig tipado
       -> proceso y adaptadores
       -> effective_config.json redactado
```

Cada ejecutable puede tener un único ingress (`server`, `guardian`), pero los
módulos no deben leer env. El registro `ConfigKeySpec` debe contener path JSON,
env/alias, tipo, default, validación, sensibilidad, alcance, restart y
proyección efectiva. No deben mantenerse esa metadata en registry, policies y
mapper independientes.

La migración completa y los tests se detallan en la guía asociada.

## 13. Backlog causal priorizado

| Orden | Trabajo | Write-set estimado | Riesgo | Paralelo | Evidencia de cierre |
|---:|---|---|---|---|---|
| 0A | Corregir auth shutdown | shutdown/auth + tests | P1 seguridad | Sí, workspace aislado | spoof/role/E2E shutdown |
| 0B | Corregir schema DomainWork | MCP schema + tests | P2 público | Sí | paridad schema-handler |
| 0C | Fijar ledger documental | docs/generador | P2 operación | Sí | un estado por capability |
| 1 | Completar/acreditar 057 | manifest/workspace/providers | P1 | Por providers, integración serial | matriz 20 invariantes |
| 2 | Implementar H0d | mailbox/intervention/Goal | P1 | Tras contrato 057 | delivery/replay/race |
| 3 | Implementar 056 | process ownership/stop | P1 | No con lifecycle compartido | A/B concurrente |
| 4A | Registro/config pipeline | `orquesta-config`, ingress | P1 diseño | Sí por ejecutable | guard repo-wide/precedencia |
| 4B | Unificar observer/idle | resident server loops | P1 operación | Sí, tras snapshot | perfil before/after |
| 5 | V1-B credentials | store/port/providers | P1 seguridad | Parcial | owner/scope/revoke/leak |
| 6 | V1-A/A2 escritura config | API/MCP/config CAS | P2 producto | Tras V1-B | write/receipt/effective |
| 7A | 058 consejo | council/identity/policy | P1 integridad | Sí | auto/required/skip E2E |
| 7B | Dos reviews reales | review/runtime/receipts | P1 integridad | Sí, no depende de convocar consejo | dos launches mismo tree |
| 8 | Consolidar providers y retirar legacy | shared store/adapters | P2 simplificación | Por módulos | paridad + caída LOC/rutas |
| 9 | Dos E2E finales de app | composición inmutable | gate release | No | receipts completos |

El orden conocido era básicamente correcto. Se añaden 0A por seguridad y 0B
por contrato público, ambos paralelizables sin cambiar la dependencia principal
057→H0d→056→V1-B→E2E. Config foundation e idle pueden avanzar en paralelo con
write-sets separados, pero la UI/escritura de secretos no precede a V1-B.

## 14. Respuestas explícitas a las preguntas de la orden

1. **“99 %”**: describía el cierre de un checklist histórico más estrecho, no
   el producto actual. No incluía H0d efectivo, 056, V1-B, la política 058
   actual, dos E2E finales, acreditación 057 ni operación idle desplegada.
2. **Cierres que siguen acreditados**: no existe un denominador normalizado que
   permita un total honesto. En la semilla, H0b y H0c conservan evidencia
   histórica/indirecta; no fueron reejecutados integralmente sobre una misma
   release actual.
3. **Cierres revocados**: uno confirmado en el alcance semilla, H0d. 057, 056,
   V1-B y 058 no se revocan: nunca estuvieron cerrados con la definición actual.
4. **Causa común**: H0d y bindings comparten la confusión
   registrado/admitido/efectivo. 057 tiene otra causa de código —proyección
   lossy antes de captura durable—, pero el mismo fallo de acreditación permitió
   que escapase.
5. **Autoridad única**: no. Hay demasiadas representaciones para intención,
   identidad, workspace, estado, evidencia y cierre.
6. **Contrato neutral multibackend**: existe a nivel de tipos, pero las
   implementaciones workspace/secure-file están bifurcadas y no hay un E2E
   común que acredite equivalencia Codex/Claude/Gemini.
7. **Intervenir un goal en vuelo**: no de forma causal acreditada; H0d solo
   persiste la orden.
8. **Detener un goal**: no está demostrado preservando otro goal; 056 abierto.
9. **Omitir consejo sin omitir reviews**: es el diseño deseado, no una
   capacidad E2E acreditada todavía.
10. **Identidad real de votos/reviews**: no siempre; hay identidad derivada del
    artefacto y tests con refs fake.
11. **Secretos fuera de prompts/resultados/receipts**: la intención de diseño es
    correcta, pero V1-B no permite acreditarlo extremo a extremo.
12. **Servidor ocioso**: no demostrado; la composición auditada mantuvo carga y
    residuo stale. Los cambios dirty necesitan medición desplegada.
13. **Seis tools**: todas alcanzan handler o responden; no todas tienen efecto
    productivo acreditado. DomainWork tiene drift de schema y el catálogo está
    vacío.
14. **Dirty tree necesario/sobrante**: manifest íntegro, propagación de ref/SHA,
    workspace estable y backoff son necesarios. Las implementaciones seguras
    duplicadas por provider deben consolidarse tras fijar comportamiento; no se
    justifica una autoridad de workspace adicional por provider.
15. **Crear app hoy**: hay piezas funcionales, pero la promesa completa no está
    demostrada en la revisión actual. El bloqueo mínimo de acreditación es
    cerrar P1 causales y ejecutar los dos E2E finales sobre la misma imagen.
16. **Evidencia de terminado**: cero P0/P1, capability ledger acreditado,
    matriz 057, H0d/056, V1-B, consejo y doble review reales, perfil idle,
    shutdown limpio, dos E2E de app y revisión independiente sobre el mismo
    commit/imagen.

## 15. Veredicto final

**Cierre parcial con fronteras exactas.** Orquesta no es una base fallida ni
requiere empezar de cero. Tiene un núcleo útil, separación por puertos, stores,
causalidad y mucha cobertura. El problema de fondo es una arquitectura
acumulativa: demasiadas rutas y representaciones, más un sistema de acreditación
que no distingue con suficiente dureza presencia, wiring, efecto y cierre.

La prioridad no debe ser producir más infraestructura. Debe ser:

1. cerrar los P1 con efectos observables;
2. fijar Goal como autoridad única;
3. consolidar primitivas transversales y configuración;
4. retirar rutas y loops duplicados con evidencia de paridad;
5. hacer que la promesa pública se genere solo desde receipts acreditados.

La guía ejecutable para agentes está en
[guia_agentes_reparacion_y_simplificacion_orquesta_2026-07-13.md](guia_agentes_reparacion_y_simplificacion_orquesta_2026-07-13.md).

## 16. Corte posterior: cierre operativo 057 Codex

La orden posterior del operador redujo expresamente el alcance: detener la
paridad Claude/Gemini/Ollama/local, conservarla aislada y cerrar 057 solo con
contratos provider-neutral y adaptador Codex. Este corte no reescribe el
veredicto amplio de la auditoría ni presenta un provider como todos los demás.

Resultado sobre `trabajo/plataforma-agentes`:

- `ae398d4f11`, `f371eded80` y `60bfe9a435` integran el contrato neutral, el
  binding durable previo al trabajo y autoridades Codex concurrentes;
- `c69b09b710` deja los workspaces privados antes del checkout y hace
  persistible el fallo pre-binding sin relajar autoridades ya ligadas;
- `388c1bd09f` elimina la doble validación de generación tmux y conserva los
  conflictos reales como observaciones `running` reintentables;
- la paridad alterna permanece fuera de la rama, en snapshot/stash aislado, y
  no debe mezclarse ni ampliarse hasta una nueva orden explícita.

Evidencia real final:

- binario exacto `388c1bd09f`, SHA-256
  `b0a3ae16146db2f919d2ea845683395bc620cad70262c79913e51f1370fd745c`;
- MCP `request-ref-e2e-057-004`, Goal
  `goal-ref-task-autoprogramming-74d17f9aaa0d-g01`;
- primera observación en vuelo: `estado=ok`, `goal_status=running`,
  `recommended_action=observe_later`;
- cierre: run `cerrada`, Goal `complete`, `closure_accepted=true`,
  `recommended_action=no_action_closed`, test requerido acreditado;
- integración canónica única: commit
  `528170925320eab5e64055b810959b1bd894bbf9`, archive e integration receipt
  durables;
- replay: un commit, un archive y un integration receipt; sin rework ni
  `closure_issues`;
- shutdown público: `shutdown_ready=true`, `cleanup_completed` y cero procesos
  propios del servidor/app-server.

Por tanto, 057 queda cerrado en el alcance operativo vigente Codex. La matriz
multibackend original queda suspendida, no falsamente cerrada.
