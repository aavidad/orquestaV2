# Progresión futura sin reescribir el núcleo

Fecha: 2026-07-14

Estado: razonamiento histórico posterior al corte mínimo. La secuencia y el
alcance vigentes están en `ruta_total_100.md` y `product/roadmap.json`.

El corte local no es un prototipo desechable. `Goal`, `WorkItem`, sus refs y las
operaciones de aplicación permanecen al añadir usuarios, PostgreSQL,
autenticación, RBAC o proveedores. La progresión sustituye o añade adaptadores
y composición; no crea una segunda autoridad de lifecycle.

## Reglas de evolución

1. `internal/goal` sigue siendo dominio puro.
2. `internal/application.Orchestrator` sigue siendo el único escritor de
   lifecycle.
3. ActorRef, ProjectRef, ExecutionRef y ArtifactRef siguen siendo opacas y
   causales; no codifican roles, proveedor o rutas.
4. HTTP, auth, DB, secretos, proveedores y conectores viven en adaptadores y se
   seleccionan en bootstrap.
5. Cada adaptador nuevo pasa la misma suite contractual antes de cutover.
6. Registro JSON, TOML no secreto y `effective_config` redactado conservan sus
   papeles; no aparecen variables o defaults fuera del registro canónico.

## Secuencia propuesta

### 1. Identidad multiusuario y autenticación

Sustituir `identity.LocalOwnerProvider` por un adaptador que traduzca una
identidad autenticada a `identity.Principal`. OIDC o Active Directory validan
tokens/sesiones en middleware de interfaz; el dominio recibe solo ActorRef y
ProjectRef validadas. Claims, tokens y cookies no entran en `Goal` ni en SQLite
como reglas de negocio.

Mientras solo exista autenticación local por Bearer, el listener sigue limitado
a loopback. Ese token protege el proceso local, pero no sustituye identidad por
usuario, RBAC, TLS ni sesiones remotas. El primer cutover multiusuario exige
pruebas de aislamiento entre dos actores y dos proyectos, expiración/revocación
y ausencia de credenciales en logs, artefactos y `effective_config`.

### 2. Autorización y RBAC

Añadir un puerto de política de autorización en la capa de aplicación y
consultarlo antes de cada comando/consulta. El adaptador mantiene usuarios,
roles y grants; la decisión durable conserva actor, proyecto, acción, recurso y
resultado para auditoría. Los handlers no consultan tablas de roles ni deciden
por strings. Las consultas y mutaciones del repositorio siguen acotadas por
ProjectRef, con pruebas negativas de acceso cruzado.

RBAC no añade estados a `Goal`: autoriza o rechaza la intención antes de que el
single writer la materialice.

### 3. PostgreSQL

Implementar otro adaptador de `application.StateRepository`. Debe conservar las
operaciones atómicas actuales: create idempotente, lectura scoped, claim con
lease, CAS por revisión, cuarentena y cierre con artefacto/atestación causal.
Migraciones, pool, DSN y health pertenecen al adaptador/composición, nunca al
dominio.

Antes del cutover:

- ejecutar la misma suite contractual de repositorio contra PostgreSQL;
- verificar concurrencia con varios workers y recuperación de leases;
- importar snapshots con invariantes y conteos de evidencia comprobados;
- hacer ensayo aislado y rollback documentado;
- seleccionar `sqlite` o `postgres` mediante una clave nueva añadida primero al
  registro canónico, sin aliases paralelos.

SQLite permanece como composición local soportada; PostgreSQL no obliga a
cambiar los aggregates.

### 4. Credenciales y secretos

Inyectar en bootstrap un resolvedor de `credential_ref` respaldado por el
secret store elegido. Los adaptadores reciben material efímero mínimo; TOML y
`effective_config` conservan solo referencias/redacción. Rotación y revocación
se prueban en la composición. No se añade lectura directa de entorno o fichero
de secretos dentro de un proveedor.

### 5. Más proveedores y conectores

Claude, Gemini, Ollama u otro runtime implementan `ports.AgentLauncher` y
`ports.AgentObserver`, incluida idempotencia causal, observación terminal
recuperable, límites y shutdown. Su selección entra en bootstrap por el
registro canónico. Ningún proveedor obtiene acceso al repositorio de lifecycle.

OPES, Forge u otras apps externas entran por conectores con contratos y refs
opacas. Aportan datos, validadores y publicación; Orquesta conserva dirección,
ejecución y evidencia. Un conector nunca comparte la DB interna ni convierte
paths de otro producto en ArtifactRef.

### 6. Escala, proyecciones y web

PostgreSQL acreditado permite claims/CAS compartidos, pero no basta para
multihost. Antes de ejecutar varios schedulers/workers se requieren además
afinidad `worker_ref/host_ref`, fencing y recuperación de host, artefactos
S3-compatible y workspaces clonables o forge remoto. Métricas, audit log,
búsqueda y dashboard son proyecciones reconstruibles; no cierran Goals.

Una web administrativa consume los mismos comandos/queries de aplicación que
MCP y atraviesa auth/RBAC. No lee o escribe tablas directamente. La UI puede
añadirse, sustituirse o retirarse sin migrar el aggregate.

## Estrategia de migración por etapa

Para cada etapa:

1. definir el puerto o ampliar el contrato sin importar el adaptador;
2. añadir conformance tests compartidos y pruebas negativas de frontera;
3. implementar el adaptador con write-set aislado;
4. ejecutarlo en instancia temporal o read-only cuando corresponda;
5. comparar IDs, revisiones, terminalidad y evidencia, no textos incidentales;
6. activar mediante configuración canónica con rollback explícito;
7. retirar compatibilidad solo tras evidencia equivalente.

No se recomienda dual-write de lifecycle: crearía dos escritores. Para una
migración de estado, se detienen nuevas mutaciones, se exporta/importa y se
valida antes de cambiar el adaptador activo.

## Catálogo posterior preservado

El catálogo funcional quedó integrado de forma autocontenida en
`product/roadmap.json`, con sus 257 IDs, cuatro hashes de fuente, decisiones,
dependencias y contratos planificados. Ese ledger sustituye la práctica de
copiar capacidad por capacidad desde el árbol antiguo; no copia implementación
legacy.

Los siguientes bloques siguen expresamente pendientes:

- Goal enriquecido con DAG, plantillas de fases, pausa/reanudación/cancelación,
  replan y progreso derivado, sin crear otro lifecycle;
- Director como rol con lease intercambiable entre Hermes, Codex, otros
  agentes u operador; Consejo y revisiones independientes como políticas de
  ejecuciones acreditadas, no como nuevos motores;
- presupuestos, permisos, efectos, receipts, mailbox, mensajería y stop
  selectivo por identidad causal;
- Wizard, `AppSpec`, fábrica de aplicaciones y las fases de investigación,
  síntesis, arquitectura, planificación, programación, documentación, pruebas,
  revisión, integración, E2E, deploy, cierre y aprendizaje;
- registro y SDK neutrales de tools, resources, skills, rulepacks y plugins,
  con carga progresiva, permisos, versiones, tests y revocación;
- broker de contexto y RAG documental/código, handoffs compactos, modo caveman,
  caching y routing barato medidos. FTS/BM25 es el inicio local; embeddings,
  reranking o una base vectorial separada solo entran tras un benchmark que
  demuestre beneficio;
- workspaces, Git/worktrees y adaptadores Forge para colaboración, además de
  API HTTP, web administrativa, streaming, CLI y notificaciones;
- secretos, PostgreSQL, artefactos S3-compatible, backup/restore, migración,
  telemetría, watchdog de CPU ociosa, instalación, actualización y deploy;
- conectores de agentes y modelos posteriores a Codex, investigación web,
  documentos, datos, medios, shell/navegador gobernados y dominios externos;
- OPES como composición consumidora completa, nunca como lógica del núcleo ni
  compartiendo su base de datos o filesystem.

Toda aplicación generada o modificada conservará el mismo gate: monolito
modular por defecto, hexagonal puro, i18n desde el primer texto, configuración
centralizada, accesibilidad, seguridad, pruebas, documentación y despliegue
proporcionales. Una capacidad solo pasa de `deferred` a `accepted` cuando tiene
contrato, wiring y evidencia E2E; no por existir en un documento.

## Qué se reutiliza del Orquesta anterior

Se reutiliza experiencia convertida en contratos pequeños: refs opacas,
goal-first, single writer, idempotencia, revisiones/CAS, leases, cierre causal
con artefactos y atestaciones, puertos de proveedores, smokes reales y parada
cooperativa.

No se reutilizan imports, runtime ni estado legacy. Tampoco se copian el
control-plane antiguo, una DB o filesystem compartidos, vocabulario versionado
como autoridad alternativa, defaults dispersos o ownership de lifecycle por
proveedores. El árbol anterior permanece referencia de solo lectura. Si una
conducta merece sobrevivir, se expresa primero como contrato o test nuevo y se
implementa detrás de las fronteras actuales.
