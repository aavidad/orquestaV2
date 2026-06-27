# Principio: Orquesta piensa mediante director

Fecha: 2026-05-14.

## Decision

Orquesta es el plano de juicio del sistema. Cualquier app externa o modulo de
dominio puede pedir trabajo, aportar reglas y validar resultados, pero no debe
asumir capacidad de juicio si no esta usando un agente director de Orquesta.

Actualizacion 2026-06-25: si la composicion dispone de Codex Goal o runtime con
goal persistente, el Director operativo interno vive dentro del goal. Orquesta
conserva juicio exterior: compila `GoalWorkSpecV0`, lanza/observa por adaptador
y acepta o bloquea el cierre por evidencias. El loop Director/agentes queda como
compatibilidad para rutas sin goal persistente.

Esto aplica igual a:

- crear documentacion o temarios;
- programar apps nuevas;
- modificar, refactorizar o migrar apps existentes;
- analizar seguridad;
- preparar deploy;
- revisar calidad o decidir rework.

## Reparto de responsabilidades

Orquesta conserva:

- arrancar director y agentes;
- decidir plan, fases, granularidad y paralelismo mediante director;
- compilar, lanzar, observar y validar `GoalWorkSpecV0` cuando la composicion
  goal-first aporta un director operativo persistente propio;
- interpretar intencion razonable de agentes y adaptar nombres, alias o formas
  equivalentes cuando no haya riesgo causal ni de seguridad;
- elegir perfil neutral de trabajo, capacidad, razonamiento, modelo, proveedor y
  runtime por conectores;
- pedir aclaraciones al usuario o a otro grupo cuando falte informacion;
- supervisar progreso, cuota, bloqueos, basura, bucles y cierre controlado;
- registrar entregas, artefactos, estadisticas y evidencias compactas.

La app de dominio conserva:

- datos, taxonomia, reglas de negocio y reglas de calidad;
- validadores deterministicos;
- persistencia y ensamblado propios;
- UI, API o MCP de su dominio;
- plantillas, categorias, niveles, tipos de documento o estructura propia;
- decision final de aceptar o rechazar un artefacto segun reglas objetivas.

La app de dominio no debe:

- decidir arquitectura, plan de agentes o estrategia de ejecucion sin director;
- elegir modelo, proveedor, HOME, DB, runtime ni cuotas;
- abrir subagentes por su cuenta si lo que necesita es juicio de Orquesta;
- copiar internals del core de Orquesta;
- esconder planificacion inteligente dentro de scripts o jobs deterministicos.

## Flujo canonico

1. La app externa describe el objetivo con contrato: `domain_ref`,
   `request_kind`, `work_kind`, restricciones, reglas y refs de contexto.
2. Orquesta arranca o consulta un director con capacidad adecuada.
3. El director decide plan, fases, tamano de tareas, agentes y criterios de
   cierre.
4. Orquesta ejecuta las tareas con agentes especializados y contextos acotados.
5. La app externa recibe artefactos y los valida/ensambla segun su dominio.
6. Si falta informacion, el director pregunta; la app externa no inventa juicio.

## OPES

OPES no planifica temarios con juicio propio. OPES debe conservar programas,
oposiciones, temas, categorias, niveles, fuentes, reglas pedagogicas,
validadores, carpetas y ensamblado final.

Cuando OPES necesite crear un temario o un tema complejo, debe pedir a Orquesta
un trabajo de planificacion documental, por ejemplo:

```text
domain_ref=opes
work_kind=plan_temario | plan_tema | plan_documento
request_kind=crear_documentacion | ampliar_documentacion | revisar_documentacion
```

Orquesta debe arrancar un director documental, normalmente con capacidad alta o
xhigh para temarios reales, y ese director decide:

- orden de creacion de temas;
- dependencias entre temas;
- cortes amplios o pequenos segun contexto disponible;
- autores, teorias, legislacion, esquemas, visuales y revisiones necesarias;
- agentes requeridos y criterios de calidad.

OPES valida despues que el plan y los artefactos cumplen sus reglas:
extension, estructura, comprension lectora, autores, teorias, legislacion,
esquemas, visuales, bibliografia, preguntas y formatos finales.

## Programacion y otros dominios

Para programar, la web/API/MCP no debe convertirse en planificador inteligente.
Debe enviar `AppSpecV0` o un contrato equivalente. Orquesta arranca director,
el director piensa arquitectura y fases, y Orquesta delega en agentes.

Para refactor, seguridad, deploy o migraciones se usa el mismo principio:
el dominio aporta contexto y restricciones; Orquesta, mediante director,
decide el plan operativo y la composicion de agentes.

Los perfiles de trabajo son neutrales: `code_study`, `implementation`,
`refactor`, `required_tests`, `documentation`, `review` y `domain_work`
describen intencion, criterios y capacidad esperada. Codex es solo un runtime
posible para ejecutarlos; OPES u otra app son conectores de dominio que aportan
datos, reglas y validadores.

## Invariantes

- Hexagonal siempre: dominio externo por puertos/conectores.
- i18n en toda superficie visible.
- DB siempre por conector; nunca SQLite/Postgres hardcodeado por defecto.
- Contextos pequenos por modulo, tarea y agente.
- Tareas adaptativas: pequenas por defecto, mas amplias cuando el dominio lo
  requiera y el contexto sea suficiente.
- Tolerancia semantica por defecto: `web_app`, `web_application`, rutas hijas,
  globs o nombres cercanos no deben romper el ciclo si la intencion es clara.
  El adaptador normaliza y el director corrige; solo se bloquean seguridad,
  causalidad, refs imposibles, datos sensibles o acciones no autorizadas.
- El director dirige; no hace todo el trabajo pesado.
- Las apps externas no se integran como internals de Orquesta salvo contratos
  genericos y conectores opt-in.
