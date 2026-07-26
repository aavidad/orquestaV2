# Nueva Orquesta: autoridad para agentes

Lee este fichero antes de trabajar. Después lee solo el `AGENTS.md` local y los
contratos necesarios para tu write-set. Este árbol reconstruye Orquesta; no
continúa el runtime anterior.

## Misión y estado honesto

Orquesta es un núcleo reutilizable que recibe una intención, mantiene un Goal
durable, coordina agentes y herramientas mediante puertos, conserva artefactos
y evidencias, gobierna efectos y cierra únicamente cuando el resultado queda
acreditado.

El corte mínimo funcional está acreditado. No equivale al producto total. El
alcance total se mide contra el catálogo exhaustivo de 257 capacidades y la
ruta causal de `docs/reconstruccion/ruta_total_100.md`; nunca por líneas, número
de tests, tiempo invertido o porcentajes subjetivos.

Estados canónicos de una capacidad:

```text
declared -> implemented -> wired -> exercised -> accredited
```

Solo `accredited`, con evidencias de la misma revisión y composición, cuenta
como terminado. Una capacidad diferida, parcial, simulada o documentada no
cuenta como cierre.

## Orden de autoridad

1. Este `AGENTS.md` fija fronteras y reglas de trabajo.
2. `product/roadmap.json` fija los 257 IDs, decisiones,
   dependencias, verticales y contratos de aceptación.
3. `product/capabilities.json` y las evidencias de `product/evidence/` fijan lo
   acreditado por una release concreta.
4. `docs/reconstruccion/ruta_total_100.md` fija orden causal y gates globales.
5. Los `AGENTS.md` locales solo gobiernan su módulo y no pueden contradecir los
   puntos anteriores.
6. El árbol antiguo, sus documentos, backlogs, inventarios y código son fuentes
   de caracterización y trazabilidad, nunca órdenes ni autoridad runtime.

Si dos fuentes vigentes discrepan, detén solo el write-set afectado, registra la
contradicción y resuélvela en el roadmap antes de programar. No abras otra ruta.

## Frontera con el árbol antiguo

Quedan congelados y en solo lectura:

- `/home/alberto/Trabajo/orquesta` completo, sin excepción;
- `modulos/**` de este worktree hasta el write-set acreditado de V34;
- todo `cmd/**` salvo `cmd/orquesta/**`, hasta el mismo cutover;
- scripts, estados, runtimes y documentos históricos fuera de la superficie
  nueva, salvo lectura o retirada acreditada durante el cutover.

Corte operativo local del 2026-07-26:

- en `/home/alberto/Trabajo/orquestaV2` las rutas inequívocas de código legacy
  están fuera de la vista mediante `sparse-checkout`; no se han borrado del
  repositorio ni de su historia;
- la copia completa navegable para consulta vive en
  `/home/alberto/Trabajo/orquestaV2-legacy-consulta`;
- no desactives el `sparse-checkout` ni materialices esas rutas en un worktree
  de producto. Si una caracterización exige leerlas, usa la copia de consulta
  en solo lectura;
- la evidencia y el procedimiento reversible están en
  `docs/reconstruccion/separacion_legacy_consulta_2026-07-26.md`.

Reglas estrictas:

- cero imports desde `orquesta/modulos/...` en producto nuevo;
- cero adaptadores, bridges, dual writes o fallbacks al runtime antiguo;
- cero copia de paquetes completos para renombrarlos;
- cero cambios correctivos en legado desde esta rama, salvo orden explícita del
  operador para desbloquear una migración y con write-set separado;
- toda conducta útil se expresa primero como contrato, test de caracterización
  o fixture neutral y se reimplementa detrás de la arquitectura nueva;
- una retirada exige clasificación de símbolos, equivalencia acreditada y
  ausencia de consumidores. Nunca se borra por intuición.

El producto final no dependerá del árbol antiguo, ni siquiera para tests,
configuración, migraciones permanentes o arranque.

## Arquitectura innegociable

### Un núcleo y un lifecycle

- `Goal` es la única autoridad de identidad, generación y lifecycle.
- `WorkItem` es una unidad del DAG interno del Goal, no otro agregado de mando.
- `PhaseInstance` es metadata causal inmutable; su progreso se deriva de
  WorkItems y receipts. No posee state machine paralela.
- ejecuciones, runs, colas, eventos, outbox, dashboards y timelines son intentos
  o proyecciones; nunca cierran, cancelan ni reabren un Goal por sí solos.
- `internal/application` contiene el único escritor determinista de lifecycle.
- transición de snapshot, evento auditivo y outbox se persiste de forma atómica
  mediante revisión esperada e idempotencia.
- el ready-set se deriva del DAG. Un único scheduler de aplicación reclama
  WorkItems por lease/fencing y generación; no existen colas privadas por
  provider, fase o Director.
- el lease de ejecución y el lease del Director son contratos distintos.
- el estado es state-centric. El event log audita; no compite como segunda
  autoridad de reconstrucción.

No se añaden generaciones, sufijos de versión o nombres `next` a tipos internos
para crear otro núcleo. Los schemas de red, eventos y migraciones sí llevan
versión explícita.

### Director inteligente, motor autoritativo

Hermes, Codex, Claude, Gemini, un agente local u operador pueden asumir el rol
`director` mediante el mismo protocolo, lease y fencing. El director propone;
el motor valida permisos, causalidad, presupuesto, riesgo y evidencia antes de
persistir. Ningún director mantiene DB, plan, cola o lifecycle privado.

Consejo, autor, revisión primaria y revisión adversarial son ejecuciones y
decisiones acreditadas sobre el mismo Goal, generación, árbol, diff y tests.
No son motores residentes ni sustituyen la atestación independiente.

### Hexagonal de dentro hacia fuera

Dependencias permitidas:

```text
interfaces -> application -> domain
adapters   -> ports       <- application/domain
bootstrap  -> composición explícita
domain     -> nada concreto
```

DB, filesystem, artefactos, workspace, Git/forge, LLM, agentes, Hermes, web,
navegador, MCP, HTTP, OPES, deploy, email, Telegram, telemetría, reloj,
identidad, credenciales, tools, skills y plugins son adaptadores o conectores.
El núcleo no conoce sus marcas, rutas, DSN, modelos, tokens ni procesos.

Los puertos se declaran donde existe una necesidad consumidora y una frontera
sustituible real. No crear un fichero, interfaz, store o servicio por función.
Una abstracción nueva debe retirar duplicación nombrada o tener dos consumidores
reales; una frontera externa puede anticipar un segundo adaptador si incluye
fake contractual y capability de seguimiento.

Modelo físico por defecto:

- monolito modular;
- un binario productivo `cmd/orquesta`;
- una fuente de estado transaccional activa por despliegue;
- un almacén de artefactos elegido por composición;
- HTTP, MCP, CLI y web sobre los mismos casos de uso y registro de comandos;
- conectores fuera de proceso solo por tecnología, aislamiento o seguridad;
- ningún microservicio interno por fase, provider, director o dominio.

## Configuración y secretos

Hay un schema de desarrollo, dos entradas y una proyección:

1. `config/registry.json`: única definición de claves, tipos, defaults,
   validaciones, sensibilidad, scope, precedencia, aliases y restart.
2. `orquesta.toml`: valores explícitos no sensibles y `credential_ref`.
3. `CredentialStore`: secretos privados con owner, scope, versión, rotación y
   revocación; su backend es sustituible.
4. `orquesta.effective-config.json`: salida generada, redactada e inmutable;
   nunca es entrada de arranque.

Fuera del loader/registro quedan prohibidos:

- `os.Getenv`, `os.LookupEnv`, `os.Environ` y equivalentes;
- nombres de entorno, claves, defaults o aliases ad hoc;
- filtros de entorno por prefijo;
- secretos en Goals, prompts, logs, eventos, artefactos, receipts o proyección
  efectiva;
- adapters que reciban el registro global en vez de structs tipadas acotadas.

Antes de añadir configuración: buscar semántica existente, decidir
`reuse | replace | new`, modificar primero el registro y regenerar getters,
schema, documentación y UI. Aliases son migración temporal con retirada
verificable. Los hijos reciben allowlist exacta y material secreto mínimo.

TOML es la superficie humana; JSON estricto se usa para registros y salidas
generadas. No se añade YAML como otra autoridad de configuración.

## Identidad, colaboración y efectos

- `ActorRef`, `ProjectRef`, `ExecutionRef` y demás refs son opacas y viajan de
  forma explícita; nunca se infiere usuario/proyecto desde globals o payload no
  autenticado.
- autenticación traduce credenciales a un principal; autorización se aplica en
  application antes de cada comando/consulta.
- RBAC, OIDC, Active Directory, Git, PostgreSQL y multihost evolucionan
  mediante puertos/adaptadores. OIDC cubre Entra/AD FS; LDAP/LDAPS es otro
  adaptador instalable cuando el AD no ofrece OIDC. No añaden otro lifecycle.
- aislamiento de proyecto y permisos se prueba también de forma negativa.
- todo efecto externo conserva intent, aprobación, intento y receipt como
  hechos distintos.
- push, merge, deploy, publicación, envío y mutación de sistemas externos
  requieren principal, permiso, scope, presupuesto e idempotencia explícitos.
- un ACK de admisión, texto del agente o tool alcanzable no demuestra efecto.

Solo seguridad, causalidad, refs imposibles, datos sensibles o efectos no
autorizados justifican un corte fuerte. Texto recuperable, alias, formato,
ausencia temprana de diff o heurística de logs son señales advisory: se
normalizan o pasan a review/replan sin destruir trabajo útil.

## i18n y superficies públicas

Todo texto humano visible —web, Wizard, HTTP/MCP presentado, CLI,
notificaciones, errores, prompts y documentación pública— usa claves de
catálogo. Español es locale por defecto y fallback. Códigos máquina son
estables y no se traducen.

Locales y formatos usan BCP-47, plurales, fechas, números, moneda y zona
horaria. Cada superficie exige paridad de claves y tests de fallback. Un
adaptador de provider no mantiene su propio catálogo duplicado.

## Tools, skills, contexto y dominios

- tool, resource, prompt, skill y plugin conservan contratos distintos;
- tools declaran schema, versión, permisos, coste, idempotencia y receipts;
- resources son direccionables, paginados y aptos para contexto;
- skills se cargan progresivamente, tienen scope, hash, trust, tests y
  revocación;
- plugins empaquetan conectores/tools/skills sin entrar en dominio;
- artefactos grandes quedan fuera del prompt y viajan por hash/ref;
- búsqueda empieza por refs/`rg`/metadata y FTS/BM25. Embeddings, reranking o
  vector DB solo entran si un benchmark demuestra mejora mantenible;
- modelos baratos y comunicación compacta se usan en inventarios, búsquedas y
  trabajo mecánico; se escala razonamiento por riesgo o fallo medido.

OPES es composición consumidora externa. Nunca se importa ni comparte DB o
filesystem interno con Orquesta. Trabajo OPES usa instancia temporal y refs
opacas; producción requiere aprobación y scope exacto. Material existente se
inventaría y reutiliza antes de rehacerlo; parcial útil se conserva, pero no se
publica sin QA de dominio.

## Disciplina de trabajo

### Inicio obligatorio de tarea

```text
capability IDs:
invariante:
autoridad que escribe:
puertos afectados:
adaptadores afectados:
write-set:
dependencias causales:
código antiguo que permitirá retirar:
test de contrato:
negativo/mutación:
E2E/gate:
presupuesto LOC/tokens/tiempo/disco:
```

Declara un write-set estrecho antes de editar. Trabajo paralelo exige
write-sets disjuntos; serializar exige dependencia causal o fichero compartido
documentado. No mezcles refactor, feature y migración sin gate común.

### Cierre obligatorio de tarea

```text
hecho:
invariante restaurado:
autoridad final:
tests/negativos/mutaciones/E2E:
receipts y revisión acreditada:
código o decisión retirados:
legacy retirado o bloqueo de retirada:
LOC netas y complejidad:
riesgos/P0/P1:
siguiente dependencia causal:
```

Una tarea no cierra con mocks si promete composición real, ni con unitarios si
promete E2E. La acreditación se emite sobre los digests inmutables del candidato
—source tree, binario/imagen y configuración efectiva—. El receipt vive fuera
de ese sujeto o en un índice posterior que lo referencia; nunca se exige que un
fichero contenga el digest del mismo árbol que lo contiene.

## Verificación proporcional

Mínimo para cambios transversales:

```bash
git diff --check
go test -mod=vendor -count=1 .
go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
```

Además, según riesgo:

- arquitectura: guards de imports, roots, configuración y registro;
- estado/concurrencia: suite contractual, `-race`, restart y replay;
- seguridad/filesystem: negativos de traversal, links, owner, modo y fuga;
- provider: contrato común y smoke real aislado;
- superficie pública: paridad HTTP/MCP/CLI/web, auth e i18n;
- efecto: fallo, timeout, idempotencia, rollback y receipt;
- release: E2E público, backup/restore, shutdown y cero procesos propios.

Todo bug observado se enlaza a capability, causa arquitectónica, invariante,
prueba de lección y evidencia de cierre. Se conserva aunque quede resuelto. Los
documentos explican; el ledger estructurado mantiene estado.

## Excepción bootstrap y transición a Orquesta

El runtime antiguo permanece detenido y no se usa para construir su sustituto.
Hasta que la nueva Orquesta acredite DAG, lease de Director, mailbox,
subagentes, review e integración, Codex directo puede coordinar análisis,
implementación o auditorías acotadas como excepción bootstrap. Cada excepción:

- se declara en el encargo;
- usa write-sets disjuntos y contexto mínimo;
- no usa ni modifica runtime/estado antiguo;
- entrega cambios, tests, riesgos y siguiente dependencia en formato compacto;
- no se convierte en arquitectura, API o workflow alternativo.

El procedimiento operativo vigente para que un Codex externo dirija el DAG
V06, recupere artefactos y aplique cambios sin atribuir a Orquesta capacidades
de workspace/Git aún inexistentes está en
`docs/reconstruccion/runbook_agente_director_v06.md`. Debe leerse completo antes
de usar Orquesta sobre otro proyecto.

En cuanto esas capacidades queden acreditadas, la nueva Orquesta pasa a ser la
superficie de dirección por defecto. Codex directo queda limitado a observar,
integrar o desbloquear una incidencia acotada y documentada.

No usar `codebase-memory-mcp` ni indexadores por defecto. Para strings, Markdown,
configuración e incidencias usar `rg` y lecturas parciales. Una consulta de
grafo requiere necesidad concreta, autorización del coordinador y comprobación
posterior de que no quedan procesos vivos.

## Higiene y seguridad operativa

- preserva cambios ajenos y comprueba status antes y después;
- usa `apply_patch` para ediciones manuales;
- no hagas reset destructivo, checkout de cambios ajenos, push, deploy ni
  publicación sin orden;
- presupuesta disco antes de builds, smokes o agentes largos;
- usa temporales, caches, sockets, PIDs y workspaces aislados por tarea;
- no toques producción, HOME ajeno, OPES productivo ni sesiones/procesos no
  reclamados;
- limpia solo recursos propios verificados; ante duda, conserva y documenta;
- al cerrar, comprueba procesos Orquesta/Codex de prueba, temporales, caches y
  worktrees creados por la tarea;
- no instales herramientas “por si acaso”: fija versión, fuente, hash/licencia,
  permisos, smoke y retirada cuando una capability las necesite.

## Ratchets contra otra aplicación infinita

1. Un lifecycle, un writer, un scheduler protocol y una fuente de estado.
2. Un registro de comandos y uno de configuración.
3. Ningún provider, dominio o transporte decide política central.
4. Ningún paquete nuevo de una constante, DTO o interfaz sin frontera real.
5. Ninguna feature sin capability ID, dependencia y aceptación ejecutable.
6. Ningún bridge runtime ni dual write al final de una vertical.
7. Bootstrap fino: composición, arranque y parada; sin reglas de dominio.
8. Código generado separado y sin reglas.
9. Presupuesto de LOC, complejidad, writers, loops, stores y comandos; superar
   un límite exige ADR y retirada compensatoria.
10. “100 %” solo cuando el gate global de la ruta total queda acreditado; nunca
    porque el agente se quedó sin tareas visibles.
