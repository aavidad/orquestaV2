# Orden de auditoría integral de Orquesta: diseño, bugs y acreditación

Fecha de emisión: 2026-07-13
Destinatario: agente auditor independiente
Ámbito: repositorio local `/home/alberto/Trabajo/orquesta` y su despliegue
Docker local
Naturaleza: auditoría de lectura y verificación; no es una declaración de
cierre ni una autorización para corregir código

## 1. Mandato

Audita Orquesta de arriba abajo para responder, con evidencia causal, estas
preguntas:

1. ¿Qué capacidades existen realmente y cuáles están solo declaradas,
   registradas, documentadas o probadas de manera insuficiente?
2. ¿Los fallos encontrados son bugs puntuales, síntomas de un problema
   sistémico de diseño, capacidades todavía ausentes, deuda operativa o falsas
   acreditaciones?
3. ¿Puede Orquesta crear hoy una aplicación completa, de forma gobernada,
   durable y observable, usando las mismas superficies públicas API/MCP que
   usará el operador?
4. ¿Qué parte del trabajo reciente es necesaria para cerrar invariantes reales
   y qué parte sería duplicación, sobreprogramación o ampliación accidental de
   alcance?
5. ¿Qué falta exactamente, en qué orden causal debe resolverse y qué evidencia
   demostraría el cierre de cada frente?

No aceptes como prueba una etiqueta de “terminado”, un checkbox, un commit de
documentación, un test unitario aislado, un resultado autorreportado por el
implementador ni la mera existencia de una tool. El objetivo de esta auditoría
es comprobar efectos completos y detectar las fronteras que los verdes
anteriores no cubrieron.

## 2. Contexto que debe tratarse como hipótesis hasta verificarlo

El repositorio contiene afirmaciones históricas de “núcleo cerrado”,
“conectores cerrados”, “plataforma terminada” y una estimación informal cercana
al 99 %. Esas afirmaciones eran cierres de alcance por hitos concretos, no una
demostración permanente de que todo el producto estuviera completo. Al menos
una acreditación fue revocada después:

- H0d se declaró cerrado porque una llamada `tools/call` escribía una línea
  durable en un JSONL.
- La revisión posterior comprobó que no había reader, claim, lease, consumo,
  replay ni ACK de entrega.
- Por tanto, se había acreditado admisión durable, no el efecto prometido:
  entregar una instrucción del operador al director o al goal.

También se documentaron otros casos donde pruebas más profundas o una app real
descubrieron fronteras que los tests anteriores no ejercitaban: bindings
registrados pero no cableados, evidencia independiente ignorada durante
promoción, identidad Git ausente en Docker, replay terminal que reobservaba un
workspace ya archivado, respuesta MCP demasiado grande y un fixture E2E de
Claude/Gemini que no reflejaba el store CAS productivo.

La explicación “el núcleo estaba al 99 % y ahora hay más tareas” no debe
resolverse por intuición. El auditor debe separar:

- el alcance que sí fue probado entonces;
- las garantías que se afirmaron sin prueba suficiente;
- las regresiones introducidas después;
- las capacidades de producto que nunca pertenecieron a aquel cierre;
- el endurecimiento legítimo exigido por uso real;
- y cualquier desarrollo nuevo que no sea necesario para el resultado final.

## 3. Estado observado al emitir esta orden

Este apartado es una fotografía, no una acreditación:

- El árbol local está sucio y contiene trabajo no integrado de 057, “intent
  manifest inmutable”, junto con cambios de cadencia ociosa del servidor.
- 057 intenta conservar la solicitud completa antes de su proyección limitada,
  ligarla a un goal mediante referencia y SHA-256, materializarla en el
  workspace efectivo y mantener esa identidad en Codex, Claude, Gemini y los
  reworks.
- La implementación y sus revisiones todavía están en curso. No debe darse por
  aceptada por estar presente en el árbol.
- El servidor local llegó a consumir CPU sin ejecutar un goal productivo. La
  hipótesis actual es polling excesivo de estados históricos/terminales, no un
  proceso zombi; hay cambios sin integrar destinados a establecer backoff
  ocioso. Deben medirse y revisarse.
- H0d está reabierto.
- 056, parada selectiva por goal/thread/generación, sigue pendiente.
- V1-B, credenciales con propietario, sigue pendiente.
- 058, Consejo de Sabios real, sigue pendiente y su contrato ha cambiado: el
  consejo debe ser opcional por política explícita, mientras que la revisión
  independiente del código sigue siendo un contrato separado.
- La creación completa de una app por API/MCP con los dos caminos de consejo
  todavía necesita una acreditación E2E reciente.

Trabajo en paralelo al emitir esta orden:

- integración y doble revisión adversarial del contrato 057, incluida la
  neutralidad Codex/Claude/Gemini y el fail-closed de rework;
- revisión separada del consumo ocioso y de la cadencia
  supervisor/observer;
- redacción de esta auditoría para que un agente nuevo pueda cuestionar tanto
  los bugs locales como el diseño transversal.

Observación puntual de runtime a las 2026-07-13T18:26:14+02:00:
`docker ps` no mostraba el contenedor canónico de orquesta-server, por lo que
el consumo anterior no podía volver a medirse en ese instante. Sí estaba vivo
`orquesta-hermes-gateway` con aproximadamente 0,13 % de CPU en una muestra.
No conviertas esa única muestra ni el estado detenido en prueba de que el bug
idle está corregido: hay que reconstruir, arrancar y ejecutar la matriz de la
Fase 11.

Antes de auditar, captura y registra:

- `git rev-parse HEAD`;
- `git status --short --untracked-files=all`;
- diff y lista de ficheros sin commit, sin modificarlos;
- commit embebido en la imagen Docker local;
- versión efectiva del servidor y de los CLIs proveedores;
- contenedores locales activos y uso de CPU/memoria;
- tests o procesos que estuvieran ejecutándose.

No mezcles evidencia del HEAD limpio con evidencia del árbol sucio ni con una
imagen Docker construida desde otro commit.

## 4. Restricciones operativas

1. Trabaja exclusivamente en local. No uses el host remoto
   `uso.dipgra.cloud` ni ningún otro remoto sin una orden nueva del operador.
2. El despliegue que se audita es Docker local.
3. Controla Orquesta por sus superficies públicas API/MCP, como lo hará el
   operador. Se permite inspeccionar contenedores y logs locales para
   diagnóstico, pero no dirigir el dominio escribiendo a pelo en stores,
   workspaces o sesiones internas.
4. No uses SSH como sustituto de API/MCP.
5. No hagas push, deploy a producción ni promoción remota.
6. No imprimas ni copies tokens, `auth.json`, secretos, `.env` completos ni el
   contenido de homes de agentes.
7. Los contenedores no deben acceder al home del host, al socket Docker, a
   OPES, a producción ni a redes privadas. El acceso al directorio gobernado de
   Orquesta y el egress público explícito pueden formar parte del perfil
   autorizado.
8. Preserva el árbol sucio. La primera pasada es de lectura. Si necesitas una
   mutación para demostrar que un test caza el fallo, hazla en un worktree
   efímero y aislado, registra el diff, revierte solo tu mutación y no mezcles
   resultados concurrentes.
9. No hagas commits ni arreglos durante la auditoría salvo autorización
   posterior. Un hallazgo no debe desaparecer antes de quedar documentado.
10. Usa `rg` y lecturas de fichero para strings, Markdown y configuración. No
    indexes el repositorio ni levantes instancias adicionales de
    codebase-memory-mcp sin permiso. Si ya hubiera procesos de ese servicio,
    comprueba que no queden instancias ociosas.
11. Evita suites pesadas simultáneas sobre el mismo worktree. La concurrencia
    de pruebas no debe contaminar la causalidad.
12. No cambies modelos, aliases, routing, seguridad, envs o ratchets para
    conseguir un verde.

## 5. Taxonomía obligatoria

Cada hallazgo debe llevar exactamente una clasificación primaria y, si aplica,
una secundaria.

### 5.1 Bug puntual

El contrato correcto ya existe, hay una autoridad clara y la mayor parte del
flujo lo respeta, pero una implementación local diverge. Debe poder señalarse
una frontera concreta y una reparación acotada que no requiera redefinir el
modelo.

Ejemplos orientativos, no conclusiones automáticas:

- una factory omite inyectar un puerto que ya es obligatorio;
- una comparación usa la referencia equivocada;
- un replay terminal vuelve a consultar un backend archivado;
- un nombre físico puede colisionar aunque la identidad lógica sea válida.

### 5.2 Fallo sistémico de diseño

El sistema carece de una autoridad única o de un invariante transversal, y
varias capas producen estados aparentemente válidos pero incompatibles. No
basta contar bugs similares: demuestra la causa compartida, las rutas afectadas
y por qué parches locales volverían a divergir.

Indicadores:

- dos o más fuentes de verdad para identidad, estado, workspace, evidencia o
  configuración;
- cada provider implementa de manera distinta una garantía que debería ser
  neutral;
- un estado durable no dispone de una máquina de estados, lease o CAS;
- registered, wired, executed, observed, reviewed y delivered se confunden
  entre sí;
- la seguridad depende de una env o convención que no forma parte del contrato
  tipado.

### 5.3 Capacidad ausente

La interfaz o la documentación prometen un resultado, pero no existe ruta
productiva completa que lo realice. No lo llames bug local si faltan puertos,
consumidores, estados o acciones enteras. H0d debe evaluarse bajo esta
categoría hasta que una ruta causal completa demuestre lo contrario.

### 5.4 Deuda operativa

La semántica principal puede ser correcta, pero operación, recursos,
diagnóstico, limpieza, despliegue o recuperación son insuficientes. Ejemplos:
polling ocioso, procesos residuales, logs sin límites, ausencia de backoff,
worktrees huérfanos o un preflight incapaz de explicar qué dependencia falta.

### 5.5 Falsa acreditación

Existe una declaración de cierre cuya evidencia no prueba el efecto declarado,
o lo contradice. Registra por separado:

- qué se afirmó;
- qué evidencia se usó;
- qué parte sí demostraba;
- qué parte faltaba;
- y qué prueba posterior la refutó.

Una falsa acreditación no implica necesariamente mala implementación; puede ser
un defecto del contrato de prueba o del proceso de revisión.

### 5.6 Regresión

Una capacidad estuvo demostrada sobre una revisión identificable y una revisión
posterior la rompe. Exige dos evidencias comparables: verde causal previo y rojo
actual sobre el mismo contrato. No llames regresión a una garantía que nunca se
probó.

### 5.7 Sobreprogramación o duplicación

Un cambio añade otra ruta, autoridad o abstracción sin ser necesario para
satisfacer un invariante de producto, o solapa una capacidad válida existente.
Para clasificarlo así debes localizar la alternativa existente, demostrar que
cubre el contrato y explicar el coste de mantener ambas. No confundas con
sobreprogramación el endurecimiento necesario para cerrar una pérdida real de
intención, identidad, causalidad, aislamiento o evidencia.

## 6. Escala de severidad y radio de impacto

Asigna severidad por consecuencia, no por tamaño del diff:

| Nivel | Criterio |
|---|---|
| P0 | Puede ejecutar fuera de autoridad, exponer secretos/host, corromper estado durable de forma amplia, promover código no acreditado o impedir operación segura global. |
| P1 | Rompe un flujo principal de app/goals, pierde intención o instrucciones, mezcla workspaces/identidades, permite falso cierre o bloquea recuperación causal. |
| P2 | Rompe una variante/provider, degrada observabilidad o replay, consume recursos de forma relevante o requiere intervención manual evitable. |
| P3 | Deuda localizada, diagnóstico deficiente o inconsistencia sin pérdida actual demostrada. |
| P4 | Limpieza, claridad o mantenibilidad sin efecto funcional demostrado. |

Registra además:

- superficies: API, MCP, runtime, Docker, store, workspace, provider, web;
- providers: Codex, Claude, Gemini u otros;
- estados: launch, running, rework, review, closure, replay, restart;
- persistencia: memoria, fichero, evento, receipt, Git;
- concurrencia: un goal, varios goals, reinicio, carrera, timeout;
- datos: intención, código, credenciales, identidad, evidencia;
- radio: local a una tool, paquete, backend, todos los goals o toda la
  plataforma.

## 7. Jerarquía de evidencia

Usa, de mayor a menor fuerza:

1. E2E vivo por API/MCP contra Docker local, con estado durable y efecto
   observable.
2. Test de composición que usa factories, stores y transportes productivos.
3. Test causal de paquete con puertos reales o dobles fieles al contrato.
4. Prueba de mutación que rompe el eslabón concreto y vuelve rojo el test.
5. Revisión estática de callers, factories, puertos e invariantes.
6. Test unitario aislado.
7. Receipt autorreportado por el implementador.
8. Documentación, checkbox, comentario o nombre de commit.

Los niveles inferiores orientan; no sustituyen a los superiores cuando el
requisito promete integración o efecto real.

Toda evidencia debe incluir:

- revisión exacta;
- árbol limpio o hash del diff sucio;
- comando o llamada API/MCP reproducible;
- código de salida/estado HTTP/JSON-RPC;
- refs durables relevantes;
- resultado esperado y observado;
- prueba negativa cuando sea material;
- y ausencia de secretos en la captura.

## 8. Método top-down

### Fase 0 — Congelar la fotografía

Documenta HEAD, dirty tree, imagen Docker, procesos, configuración pública
efectiva y estado de la cola. Identifica qué trabajo pertenece a 057, qué
pertenece al arreglo de CPU y qué cambios son ajenos. No atribuyas un test
verde a una revisión diferente.

### Fase 1 — Reconstruir la promesa del producto

Extrae requisitos de:

- README y documentación de arquitectura vigente;
- `CODEX_LEEME.md`;
- `docs/hoja_ruta_cierre_conectores_2026-07-12.md`;
- planes de v1.0, contratos H0a–H0d, 056–058 y H1/H4/H5;
- registros MCP/HTTP y tipos públicos;
- invariantes de dominio y validadores;
- tests que se presentan como acreditación.

Cuando dos documentos se contradigan, no elijas el más optimista. Registra la
contradicción y busca la autoridad más reciente más evidencia ejecutable.

Produce un inventario de capacidades con estos estados:

- especificada;
- implementada;
- cableada;
- ejercitada;
- observable;
- durable/replayable;
- acreditada causalmente;
- revocada;
- pendiente;
- obsoleta/solapada con justificación.

### Fase 2 — Dibujar los flujos de extremo a extremo

Para cada flujo principal, traza:

`entrada pública → validación → aplicación → dominio → puerto → adaptador →
store/runtime → observación → evidencia → review → closure → replay`

Flujos mínimos:

1. crear/preparar una app;
2. lanzar y observar un goal;
3. rework y successor;
4. detener un goal específico;
5. mensaje operador→director/goal;
6. delivery→revisión→atestación→cierre;
7. consejo previo, cuando la política lo requiera;
8. bypass explícito del consejo;
9. credencial por referencia y propietario;
10. promoción/integración/archivo;
11. restart y replay;
12. estado ocioso del servidor.

En cada flecha pregunta:

- ¿quién es la autoridad?
- ¿qué identidad viaja?
- ¿qué se persiste antes del efecto?
- ¿qué ACK acredita admisión y cuál acredita ejecución/entrega?
- ¿qué ocurre ante timeout, cancelación, crash o replay?
- ¿cómo se evita que una capa reconstruya la verdad por heurísticas?

### Fase 3 — Inventario de tools, rutas y conectores

Enumera desde el transporte vivo todas las tools/resources y contrástalas con
los bindings declarados y las factories productivas. Para cada tool verifica:

- aparece en `tools/list`;
- valida el input público;
- el puerto no es nil;
- llega a un consumidor productivo;
- produce un efecto o un error tipado útil;
- persiste y expone la evidencia prometida;
- replay no duplica el efecto;
- restart conserva la verdad;
- una mutación del binding o consumidor deja rojo el smoke.

Revalida expresamente las seis tools que en una revisión histórica se
reportaron como muertas; no asumas que siguen rotas ni que ya están arregladas:

1. `orquesta.apps.ejecutar_orquestacion.v0`
2. `orquesta.apps.solicitar_nueva.v0`
3. `orquesta.director_agent.apply_decision.v0`
4. `orquesta.domain_work.v0`
5. `orquesta.runtime.models.v0`
6. `orquesta.tool.capabilities.list.v0`

Clasifica cada una como viva, parcialmente viva, ausente, solapada u obsoleta,
con un caller/efecto y una prueba negativa. “Registrada” nunca significa
“operativa”.

### Fase 4 — Autoridades, identidad y durabilidad

Busca fuentes de verdad paralelas para:

- request/run/goal/external-goal/thread/generation refs;
- estado del goal y versión CAS;
- manifest de intención y su SHA;
- workspace lógico y físico;
- result, closure, attestation y promotion receipts;
- identidad del autor, revisor, votante y operador;
- configuración y revisión efectiva;
- credenciales y propietario;
- commit, árbol y write-set.

Para cada identidad comprueba:

- construcción canónica e inyectiva;
- límites de longitud y caracteres;
- ausencia de colisiones por minúsculas, sustitución o truncado;
- persistencia antes de ejecutar;
- igualdad entre launch, process, observe, attestation y closure;
- rechazo fail-closed ante discrepancia;
- replay idempotente;
- comportamiento tras restart.

Una referencia lógica segura no implica que su nombre físico también lo sea.
Los filenames derivados de Claude/Gemini y cualquier otro provider deben
resistir colisiones y longitudes hostiles.

### Fase 5 — Auditoría específica de 057

057 no se acepta hasta demostrar todas estas propiedades:

1. La petición goal-first completa se captura antes de truncar objetivo,
   contexto o criterios.
2. Existe una representación canónica determinista, con validación estricta,
   límites explícitos y SHA-256.
3. El store es create-if-absent, durable, idempotente y fail-closed ante una
   ref existente con contenido distinto.
4. Lectura/escritura resisten symlinks, hardlinks, FIFOs, ficheros especiales,
   permisos inseguros, ancestros inseguros, tamaños hostiles y JSON con campos
   desconocidos/duplicados/trailing data.
5. `GoalSpec` propaga referencia y SHA sin romper solicitudes legacy.
6. Codex, Claude y Gemini materializan y ejecutan exactamente el mismo
   manifest verificado.
7. La ruta física usada por process coincide con el binding durable escrito al
   launch y con la usada por observe, watcher, fingerprint y attestation.
8. No hay una extracción Codex-only que deje a Claude/Gemini caer al workspace
   canónico.
9. `ProjectWorkDir`, workspace de promoción y workspace del agente no divergen
   por usar configuraciones distintas.
10. Los nombres físicos de goal/provider son inyectivos y acotados.
11. Rework usa un workspace sucesor distinto y nunca comparte ni copia un
    workspace que pueda seguir vivo.
12. Mientras no exista una autoridad acreditada de stop/terminal para la
    fuente, `PreserveArtifacts` falla cerrado con el error causal exacto
    `requires_056_selective_stop_receipt`. 057 no debe fabricar una garantía de
    transferencia que depende de 056.
13. Replay del mismo successor no pisa trabajo posterior ni crea otro
    workspace.
14. Un segundo successor incompatible falla cerrado.
15. Reworks legacy y App Director conservan compatibilidad o reciben un error
    explícito y probado; no se rompen silenciosamente.
16. El manifest materializado es de solo lectura y se verifica de nuevo antes
    de Start/turn.
17. Los límites del manifest dejan margen real para el envelope HTTP/MCP; un
    objeto válido en dominio debe ser transportable por la superficie que lo
    acepta.
18. Tests adversariales fuerzan `prepare=A, resolve=B` y exigen fallo antes de
    ejecutar.
19. Focales, carrera, repetición, paquetes consumidores y E2E de los tres
    providers quedan verdes en la misma revisión.
20. Dos revisiones independientes inspeccionan contrato y seguridad; ninguna
    puede ser el autor de la implementación.

La transferencia de artefactos, checkpoints o write-set preservado queda fuera
del cierre autónomo de 057 y se acredita junto con 056. Solo cuando exista un
receipt durable que pruebe que la fuente está selectivamente detenida o en un
terminal compatible podrá incorporarse snapshot/copia. Entonces serán
obligatorios confinamiento, rechazo de symlinks y ficheros especiales,
estabilidad ante carreras, límites de cantidad/tamaño, durabilidad,
idempotencia y replay que no pise trabajo posterior. Hasta ese momento, fallar
cerrado es el comportamiento correcto.

Hallazgos recientes que deben revalidarse, no copiarse sin comprobar:

- comparación de replay antes de reconstruir identidad;
- autorización débil del source workspace;
- rework compartiendo workspace con un proceso todavía vivo;
- intento de copiar/preservar desde un workspace potencialmente vivo, en vez
  de exigir `requires_056_selective_stop_receipt`;
- nombres Gemini/Claude no inyectivos;
- binding de file-control preparado en A pero process/observe resolviendo B;
- lookup/watcher/fingerprint especializados en Codex y ciegos para
  Claude/Gemini;
- divergencia entre `IdleSelfImprovementProjectWorkDir` y `ProjectWorkDir`;
- directorio vacío resuelto accidentalmente como CWD;
- límites incompatibles entre manifest y envelope.

Para cada uno indica si el árbol actual lo conserva, lo corrige por completo o
solo lo desplaza.

### Fase 6 — Lifecycle, concurrencia y replay

Audita todas las máquinas de estado y stores:

- transiciones válidas e inválidas;
- versión monotónica;
- CAS y tratamiento del ganador concurrente;
- leases con owner, expiración y reclaim;
- cancelación del contexto después de adquirir un claim;
- writers tardíos;
- doble launch/observe/closure;
- sucesor inmediato y cierre causal;
- restart entre cada par de transiciones;
- compatibilidad entre memoria y file store;
- limpieza selectiva sin apagar recursos compartidos;
- replay sin segundo commit, delivery, voto o promoción.

Usa `-race` y repeticiones cuando la prueba sea determinista. Una prueba de
concurrencia que ejecuta mutaciones simultáneas sobre el mismo worktree sin
aislamiento no acredita causalidad.

### Fase 7 — H0d, entrega causal del mailbox

El contrato mínimo a verificar es:

- scope explícito `run_ref + goal_ref`;
- `message_ref` idempotente;
- enqueue ACK distinto de delivery ACK;
- estados event-sourced
  `queued → claimed(lease CAS) → delivered`;
- orden estable;
- lease expirado recuperable;
- aislamiento entre runs/goals;
- replay sin duplicar turn ni prompt;
- entrega a goal app-server activo mediante turn de la misma generación;
- entrega a backend no interactivo mediante successor/bundle;
- cuerpo completo, sin copiarlo a `AcceptanceCriteria` limitados;
- E2E `MCP tools/call → queued → claimed → prompt real → delivery ACK`;
- restart y `-race -count=3`.

Si solo existe append, clasifica la capacidad como ausente/parcial, no como bug
de ACK.

### Fase 8 — 056, stop selectivo

Demuestra que detener B:

- identifica goal, thread y generación exactos;
- adquiere el lease correcto;
- impide nuevos turns de B;
- limpia solo sus recursos;
- conserva el backend/tmux compartido;
- permite que A siga ejecutándose y observable;
- es idempotente;
- funciona en carrera con observe/start/rework;
- deja receipt durable y estado terminal coherente;
- no confunde cancelación del request HTTP con stop durable.

La ausencia de esta capacidad no se suple apagando el servidor ni el daemon
compartido.

### Fase 9 — Consejo opcional y doble revisión obligatoria

No mezcles deliberación previa con revisión del código entregado.

#### Política del Consejo de Sabios

058 debe modelar una política tipada y durable:

- `auto`: el sistema decide convocar ante ambigüedad, riesgo o decisión
  arquitectónica/material según reglas visibles;
- `required`: el consejo es obligatorio;
- `skip_by_operator`: un humano con dirección clara lo omite explícitamente.

`skip_by_operator` exige identidad del operador, motivo, timestamp, spec/hash
afectado y receipt durable/reproducible. No puede ser un booleano implícito, un
default silencioso ni una inferencia por ausencia de configuración.

En el camino `required` o cuando `auto` convoca:

- miembros reales lanzados por Orquesta, no votos sintéticos;
- Sol, Terra y Luna como familias/participantes según routing vigente, sin
  inventar que el alias es una identidad acreditada;
- launch receipt, ACK, familia, rol y generación;
- propuesta, crítica, voto/disenso y decisión causal;
- identidad del votante derivada del launch, no declarada libremente por el
  caller;
- bloqueo/veto y rework versionados;
- un rework no sella el consejo para siempre;
- coste/cuota y degradación explícitos;
- menos diversidad o cuota insuficiente no degrada silenciosamente a una sola
  opinión.

#### Revisión independiente de la implementación

Aunque el consejo se omita, una app con código necesita como mínimo:

- autor separado de los revisores;
- al menos dos ojos/agentes independientes;
- una revisión primaria y otra adversarial, de identidades/familias
  acreditadas;
- receipts ligados a entrega, generación, commit/árbol y tests exactos;
- cualquiera puede bloquear y abrir rework;
- resolución durable de discrepancias;
- atestación independiente de tests separada de ambos reviews;
- cierre imposible si falta uno de los receipts requeridos.

El E2E final debe cubrir dos caminos completos:

1. app con requisitos claros y `skip_by_operator`;
2. app deliberada con consejo real Sol/Terra/Luna.

Ambos deben llegar por API/MCP hasta implementación, tests, dos reviews,
delivery, promoción y closure.

### Fase 10 — Credenciales, configuración y frontera Docker

Comprueba:

- `owner_ref` obligatorio para credenciales;
- autorización por propietario;
- al goal solo viaja `credential_ref`, nunca el secreto;
- resolución del secreto en la frontera mínima necesaria;
- receipt de qué goal usó qué referencia, sin valor secreto;
- revocación y replay;
- logs/prompts/resultados redaccionados;
- configuración canónica de una sola lectura por revisión;
- futura escritura gobernada posible, validada, atómica y con before/after
  receipt;
- sin duplicados semánticos de env;
- envs de seguridad con autorización explícita;
- contenedor no root, sin Docker socket, sin home/OPES/producción, sin red
  privada y con filesystem/mounts gobernados;
- internet permitido no implica acceso al host.

Busca especialmente rutas de fallback que conviertan una referencia ausente en
HOME, CWD, config global o credenciales personales.

### Fase 11 — Operación, CPU y residuos

Mide por separado:

- CPU de host;
- CPU por contenedor;
- CPU por proceso/thread;
- frecuencia y duración de ticks;
- número de goals activos, terminales e históricos inspeccionados;
- timers, polling, backoff y wakeups;
- sesiones tmux, app-server, procesos hijos y worktrees residuales.

Prueba como mínimo:

1. cola vacía durante una ventana suficiente;
2. muchos estados terminales sin goals activos;
3. un goal activo;
4. transición activo→terminal;
5. wakeup por nuevo trabajo después de backoff;
6. restart del servidor.

El arreglo debe:

- excluir o indexar estados terminales para no recorrerlos continuamente;
- aplicar backoff o espera dirigida en idle;
- despertar con latencia acotada al llegar trabajo;
- no ralentizar observación activa;
- no crear goroutines/timers acumulativos;
- exponer métricas suficientes para explicar consumo;
- mantener CPU ociosa cercana al baseline medido, no a un umbral inventado.

Un proceso vivo que hace polling inútil es deuda/bug operativo; no lo llames
zombi si responde y ejecuta su loop diseñado.

### Fase 12 — Seguridad

Revisa amenazas y no solo happy paths:

- path traversal y canonicalización;
- symlink/hardlink/TOCTOU;
- ficheros especiales y bloqueo por FIFO;
- permisos y ancestros;
- límites de tamaño/recursión/cantidad;
- JSON estricto;
- inyección en shell/argv/env/prompts;
- refs colisionables;
- replay conflictivo;
- ejecución fuera de write-set;
- cruce entre runs/workspaces;
- identidad del caller/votante/operador;
- secretos en logs/receipts;
- sandbox opt-out;
- egress a redes privadas/metadata;
- Docker socket y mounts host;
- promoción de código no revisado.

Toda relajación de seguridad debe ser explícita, aislada, opt-in, probada y
visible en el título/receipt del cambio. “Está dentro de Docker” no elimina la
necesidad de demostrar la frontera del contenedor.

### Fase 13 — Pruebas de verdad y mutación

Para cada gate crítico pregunta qué mutación debería dejarlo rojo. Con worktrees
aislados, prueba al menos:

- binding declarado pero puerto nil;
- consumidor no llamado;
- ACK de admisión sin efecto;
- evidencia autorreportada sin atestación;
- reviewer igual al autor;
- segundo review ausente;
- voter_ref inventado;
- manifest alterado después de persistir;
- workspace resuelto distinto del binding;
- provider alternativo cayendo al canonical;
- CAS ganador incompatible;
- lease expirado/no reclamado;
- replay duplicando efecto;
- test requerido omitido;
- commit/árbol distintos de los revisados;
- consejo omitido sin receipt de operador;
- stop B apagando A;
- polling de terminales sin backoff.

Una prueba es fuerte si la mutación rompe exactamente el eslabón y falla por la
razón esperada. Un rojo por compilación, fixture roto, falta de red o mutaciones
concurrentes no acredita el gate.

Orden recomendado:

1. focales de la función/contrato;
2. paquete completo;
3. consumidores y factories;
4. `-race` y repeticiones;
5. E2E de composición;
6. suite global offline/vendor solo sobre una revisión estable;
7. smoke vivo API/MCP en Docker local.

No uses un `go test ./...` verde para ocultar que el E2E material no existe.

## 9. Matriz requisito → evidencia

Entrega una tabla completa. Empieza con esta semilla y amplíala:

| ID | Requisito | Estado previo declarado | Evidencia necesaria | Evidencia actual | Veredicto |
|---|---|---|---|---|---|
| CAP-APP-01 | Crear una app completa por API/MCP | Afirmado históricamente | Dos E2E vivos, consejo skip y required, hasta closure | Por recopilar | No demostrado hasta auditoría |
| H0A-01 | Goal real por app_server_tmux aislado | Acreditado históricamente | Smoke real, receipt, cleanup, frontera Docker | Por revalidar regresión | Pendiente |
| H0B-01 | Binding MCP/HTTP realmente cableado | Acreditado con mutación | tools/list + llamada por grupo + mutación de puerto | Por revalidar | Pendiente |
| H0C-01 | Delivery→review→closure causal | Acreditado con mutación | E2E real, ACK, gate, atestación, closure | Por revalidar | Pendiente |
| H0D-01 | Operador→goal entregado causalmente | Cierre revocado | tools/call→prompt→delivery ACK + replay/race | Evidencia histórica solo de append | Abierto |
| 056-01 | Stop selectivo preserva otro goal | Pendiente | Stop(B) concurrente con Observe(A), sin shutdown compartido | Por implementar/verificar | Abierto |
| 057-01 | Intención completa, inmutable y multibackend | En curso | Matriz de 20 invariantes de Fase 5 | Dirty tree no acreditado | En curso |
| 058-01 | Consejo con política auto/required/skip | Pendiente | E2E de convocatoria y bypass durable | Por implementar/verificar | Abierto |
| REV-02 | Dos reviews independientes de app | Parcial/histórico | Dos launch/ACK/receipts ligados al mismo commit | Por verificar | Abierto |
| V1B-01 | Credencial con dueño y secreto no propagado | Pendiente | tests de auth, redacción, uso y revocación | Por implementar/verificar | Abierto |
| OPS-CPU-01 | Idle sin polling inútil | Bug observado | perfil antes/después y tests de backoff/wakeup | Cambios sin integrar | En curso |
| TOOL-01..06 | Seis tools históricamente reportadas muertas | Disputado | llamada pública + efecto + mutación por tool | Por revalidar | Pendiente |

Valores permitidos para veredicto:

- probado;
- contradicho;
- incompleto;
- evidencia indirecta;
- evidencia ausente;
- no aplica con justificación.

No uses “parece”, “probablemente” o “no vi errores” como cierre.

## 10. Patrones de falso verde que debes buscar

1. **Registrado ≠ cableado.** Tool presente con dispatcher/puerto nil.
2. **Cableado ≠ ejecutado.** La llamada llega a un stub o rama sin efecto.
3. **Append ≠ entrega.** El store contiene un mensaje que nadie consume.
4. **ACK ≠ efecto.** Receipt de admisión presentado como ejecución.
5. **Self-result ≠ acreditación.** El autor declara tests pasados sin refs
   independientes.
6. **Test fixture ≠ producción.** Fake/memory store no implementa CAS, lease,
   restart o wiring real.
7. **Unitario ≠ composición.** La función funciona, la factory no la inyecta.
8. **Una provider ≠ todas.** Codex verde mientras Claude/Gemini usan otra ruta.
9. **Ruta lógica ≠ workspace físico.** Launch, process, observe y attestation
   miran checkouts distintos.
10. **Estado terminal ≠ replay correcto.** Reiniciar vuelve a ejecutar,
    observar o promover.
11. **Dos nombres ≠ dos identidades.** Sanitización/truncado produce colisión.
12. **Dos respuestas ≠ dos revisiones.** Mismo agente/familia o un agregador
    se presenta como independencia.
13. **Consejo registrado ≠ consejo real.** Votos sintéticos o caller-controlled
    sin launch/ACK.
14. **Documentación ≠ estado.** Checkbox posterior contradicho por tests o
    runtime.
15. **200 ≠ éxito sano.** Recuperación degradada oculta un error original.
16. **MCP 0 activos ≠ 0 historial.** Proyección de cola confundida con ausencia
    global.
17. **Docker ≠ sandbox demostrado.** La frontera depende de mounts/red/env no
    auditados.
18. **Suite global verde ≠ requisito cubierto.** Ningún test activa la rama
    crítica.
19. **Cierre de frente ≠ producto completo.** Alcance histórico más estrecho
    que la afirmación posterior.
20. **Diff grande ≠ progreso.** Nueva infraestructura sin consumidor ni E2E.

## 11. Auditoría de sobreprogramación

Para cada módulo o abstracción añadida desde el cierre histórico:

1. identifica el requisito exacto que satisface;
2. enlaza el bug o riesgo reproducible que la motivó;
3. busca una capacidad previa equivalente;
4. cuenta autoridades antes y después;
5. comprueba si el cambio reduce o aumenta rutas/provider forks;
6. exige un test que falle sin el cambio;
7. mide su alcance y coste operativo;
8. decide: conservar, consolidar, completar, deprecar o borrar.

Aplica especialmente este análisis a 057. La pérdida observada de intención
completa justifica una solución; no justifica automáticamente cada abstracción
del dirty tree. El criterio correcto es el mínimo conjunto de mecanismos que
garantice la intención canónica y el mismo workspace en todo el lifecycle, no
el mínimo diff ni la arquitectura más grande.

Una capacidad sin caller debe recibir una de estas resoluciones explícitas:

- conectar porque es válida y necesaria;
- consolidar con la autoridad vigente;
- marcar deprecación con migración;
- eliminar porque está solapada/obsoleta/inválida;
- mantener experimental fuera de la promesa pública.

No conectes todo ciegamente: una ruta muerta puede ser evidencia de diseño
abandonado, no una orden para multiplicar superficies.

## 12. Formato obligatorio de cada hallazgo

Usa esta plantilla:

### [ID] Título factual

- **Clasificación primaria:** bug puntual / diseño sistémico / capacidad
  ausente / deuda operativa / falsa acreditación / regresión /
  sobreprogramación
- **Clasificación secundaria:** opcional
- **Severidad:** P0–P4
- **Confianza:** alta / media / baja
- **Estado:** reproducido / demostrado estáticamente / hipótesis / descartado
- **Revisión y árbol:** HEAD, dirty hash o imagen
- **Superficies afectadas:**
- **Providers/estados afectados:**
- **Promesa o invariante:**
- **Comportamiento observado:**
- **Reproducción mínima:** comando o llamada pública, sin secretos
- **Evidencia positiva:**
- **Prueba negativa o mutación:**
- **Causa inmediata:**
- **Causa raíz:** si está demostrada; si no, “no determinada”
- **Por qué no es otra categoría:** por ejemplo, por qué es diseño y no bug
  local
- **Radio de impacto:**
- **Riesgo de seguridad/datos:**
- **Alternativas existentes y solape:**
- **Corrección mínima que restaura el invariante:**
- **Corrección estructural recomendada:** solo si difiere
- **Tests que impedirían regresión:**
- **Dependencias/orden causal:**
- **Criterio verificable de cierre:**

Separa hechos, inferencias e hipótesis. Una lectura de código puede demostrar un
caller ausente, pero no siempre demuestra el comportamiento runtime.

## 13. Entregables

El auditor debe producir:

1. **Resumen ejecutivo**: qué funciona, qué no, riesgos P0/P1 y si la
   afirmación “Orquesta puede crear una app” está probada, contradicha o
   pendiente.
2. **Mapa arquitectónico top-down**: autoridades, puertos, adaptadores, stores,
   providers y fronteras Docker.
3. **Inventario completo de capacidades/tools** con estado y caller/efecto.
4. **Matriz requisito→evidencia** completa, no solo la semilla.
5. **Catálogo de hallazgos** con la plantilla anterior.
6. **Análisis transversal de causa raíz**: agrupa síntomas solo cuando haya una
   causa compartida demostrada.
7. **Informe de falsas acreditaciones** y por qué sus pruebas no alcanzaban.
8. **Plan de mutaciones y E2E** con resultados reales.
9. **Informe de operación/CPU/residuos** con mediciones antes/después.
10. **Análisis de sobreprogramación/solape** por frente y recomendación de
    conservar/consolidar/completar/deprecar/eliminar.
11. **Backlog causal priorizado**, con dependencias, write-set estimado,
    riesgo, pruebas de cierre y si puede paralelizarse sin compartir
    workspaces.
12. **Veredicto final independiente**, que puede ser:
    - cierre demostrado;
    - cierre parcial con fronteras exactas;
    - cierre contradicho;
    - no verificable por evidencia insuficiente.

No entregues solo una lista de bugs. El valor principal es saber si comparten
una causa de diseño y qué prueba impide volver a declarar un falso cierre.

## 14. Cola conocida que debe contrastarse

La cola vigente propuesta, sujeta a la auditoría, es:

1. cerrar 057;
2. entrega causal H0d;
3. 056 stop selectivo;
4. V1-B credenciales con propietario;
5. V1-A/V1-A2, escritura gobernada de configuración y catálogo/routing;
6. 058 Consejo opcional con identidad real;
7. V1-C/H4 y capacidades válidas todavía sin conexión;
8. acreditación final de app por API/MCP;
9. arreglos operativos como CPU idle, ordenados por severidad y dependencia.

El auditor debe decir si este orden es correcto. Puede reordenarlo solo
aportando una dependencia causal o un riesgo superior. En particular:

- 057 precede trabajos cuya intención extensa se perdería sin él;
- H0d permite corregir goals en vuelo;
- 056 evita apagar recursos compartidos al gobernar procesos;
- credenciales deben cerrarse antes de afirmar multiusuario seguro;
- el consejo depende de identidad real, pero es opcional por política;
- la doble revisión de código no depende de que el consejo se convoque;
- el E2E final no puede preceder a las capacidades que pretende acreditar.

## 15. Preguntas que el veredicto debe responder sin ambigüedad

1. ¿Qué significaba exactamente el “99 %” y qué no incluía?
2. ¿Cuántos cierres históricos siguen acreditados al reejecutar sus pruebas
   causales actuales?
3. ¿Cuántos se revocan por falta de efecto o por regresión?
4. ¿Existe una causa sistémica común detrás de H0d, 057 y los bindings muertos,
   o son fronteras distintas? Demuéstralo.
5. ¿Hay una autoridad única para intención, identidad, workspace, estado,
   evidencia y cierre?
6. ¿Codex, Claude y Gemini recorren el mismo contrato neutral?
7. ¿Puede el operador intervenir un goal en vuelo?
8. ¿Puede detener solo un goal?
9. ¿Puede omitir el consejo explícitamente sin omitir las dos revisiones?
10. ¿Los votos y reviews proceden de agentes realmente lanzados?
11. ¿Los secretos permanecen fuera de prompts, resultados y receipts?
12. ¿El servidor queda realmente ocioso cuando no hay trabajo?
13. ¿Las seis tools históricamente muertas tienen efecto productivo?
14. ¿Qué cambios del dirty tree son necesarios y cuáles sobran o se solapan?
15. ¿Puede crearse una app real hoy por API/MCP? Si no, ¿cuál es el bloqueo
    mínimo exacto?
16. ¿Qué evidencia concreta permitiría declarar Orquesta terminada sin volver a
    depender de documentación optimista?

## 16. Condición de cierre de la auditoría

La auditoría termina cuando cada requisito público relevante tiene un
veredicto y una evidencia proporcional a su alcance; no cuando deja de
encontrar bugs obvios.

No declares Orquesta terminada salvo que:

- no queden P0/P1 abiertos;
- las capacidades prometidas estén cableadas y ejercitadas;
- el lifecycle sea durable, idempotente y causal en restart/replay;
- Codex, Claude y Gemini mantengan intención/workspace/identidad;
- H0d y 056 tengan E2E real;
- consejo `auto|required|skip_by_operator` y doble revisión separada estén
  demostrados;
- credenciales y frontera Docker estén acreditadas;
- CPU idle y residuos estén bajo control medido;
- los dos E2E de creación de app pasen por API/MCP local;
- las mutaciones críticas dejen rojos los gates correspondientes;
- y una segunda revisión independiente confirme el resultado sobre la misma
  revisión desplegada.

Si falta una sola de esas pruebas, entrega un cierre parcial honesto con el
bloqueo exacto. No reduzcas la definición de terminado para ajustarla a lo que
ya existe.

## Addendum de autoridad del operador del 2026-07-13

La orden operativa posterior prevalece para la ejecución inmediata: se detiene
la paridad Claude/Gemini/Ollama/local y se conserva aislada; 057 se cierra con
contratos provider-neutral y adaptador Codex, sin abrir otros frentes salvo
bloqueos directos del E2E Codex. El resultado acreditado es
`request-ref-e2e-057-004` sobre `388c1bd09f`: observe en vuelo `running`, cierre
aceptado, test independiente, promoción/commit canónicos, replay idempotente y
shutdown público limpio. Esto cambia el alcance de ejecución, no convierte la
paridad suspendida en evidencia disponible.
