<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Berserk (asistente técnico-operativo de Alberto)
Fecha: 2026-03-30
-->

# Informe operativo de estado, doctrina y prioridades — 2026-03-30

## Objetivo

Dejar una referencia operativa viva para no perder el norte mientras se sigue desarrollando Orquesta.

Este informe no sustituye a la visión, al roadmap ni a las OPs.
Su función es más táctica:

- resumir qué está decidido de verdad
- distinguir doctrina aprobada de implementación real
- señalar brechas activas
- priorizar el trabajo técnico
- servir como mapa de control para Alberto y para los agentes

---

## 1. Resumen ejecutivo

Orquesta ya no debe entenderse como un panel o un conjunto de scripts para lanzar agentes.
La dirección aprobada del producto es la de un **plano de control multiagente, server-first, con gobierno operativo, memoria, conectores, trazabilidad y control de ciclo de vida real** sobre agentes, proyectos y runtimes dentro del workspace `~/Trabajo`.

La app tiene una base funcional importante ya implementada:

- daemon/servidor operativo
- panel web funcional
- CLI y API amplias
- tareas, propuestas, votos y sesiones
- control plane inicial
- runtime orders, checkpoints y observabilidad pasiva
- reglas, skills y workflows persistentes
- gobierno Git, memoria, decisiones y documentación referenciada

Sin embargo, el estado actual sigue siendo **híbrido**.
La doctrina del sistema está más madura que su cierre operativo.
Las principales brechas no están tanto en la ausencia total de capacidades, sino en:

- caminos locales aún vivos junto al modelo server-first
- acoplamiento excesivo de `cmd/` con `db.*`
- continuidad/runtime/control plane todavía no suficientemente finos
- inconsistencia entre contratos aprobados e implementación real en algunos frentes (por ejemplo i18n)
- scripts/rutas heredadas que siguen contaminando el camino operativo principal

Conclusión operativa:

> La prioridad no debe ser abrir nuevos frentes por defecto, sino consolidar la base operativa y estructural ya aprobada para que Orquesta mande de verdad sobre los agentes y sobre sí misma.

---

## 2. Doctrina aprobada que debe considerarse vigente

### 2.1. Arquitectura hexagonal como gate

Doctrina:

- el núcleo no debe depender de SQLite, Codex, Claude, Gemini, Terminator ni adaptadores concretos
- la base de datos es adaptador, no núcleo
- CLI, web y API deben ser adaptadores de entrada finos
- la lógica de aplicación debe vivir en servicios/puertos

Implicación práctica:

- no se debe seguir metiendo lógica nueva en `db/` o en `cmd/` para “avanzar rápido”
- cualquier ampliación relevante debe pasar por capa de servicios y puertos

### 2.2. Server-first como operativa oficial

Doctrina:

- `orquesta serve` / daemon es la fuente de verdad
- CLI, web y API son clientes del servicio
- el modo local queda solo para recuperación explícita

Implicación práctica:

- no deben crearse nuevas rutas de negocio que operen normalmente contra la BD local
- scripts manuales y wrappers heredados deben salir del camino principal

### 2.3. Política AP-077 de acceso a persistencia

Doctrina:

- toda mutación normal debe pasar por la aplicación/servidor
- la lectura directa de persistencia solo se tolera excepcionalmente mientras falte cobertura real
- si una necesidad obliga a SQL directo, debe abrirse tarea para llevarlo a API/CLI

Implicación práctica:

- cero nuevas rutas de negocio manuales contra SQLite
- toda brecha observada se traduce en deuda técnica priorizable

### 2.4. i18n donde aplique

Doctrina:

- i18n no es ritual universal
- sí es obligatorio por defecto en superficies humanas visibles y operativas
- no debe imponerse artificialmente a utilidades técnicas sin retorno real
- las apps/docs con consumo humano deben nacer con contrato estructurado, fallback y política definida

Implicación práctica:

- no seguir metiendo texto visible hardcodeado en superficies que deberían respetar política i18n
- cualquier cierre del frente i18n debe alinear contrato generado y cargador real

### 2.5. Reglas, skills y workflows como gobierno vivo

Doctrina:

- reglas, skills y workflows deben servirse desde Orquesta
- deben ser editables, versionables y auditables
- pueden gobernarse por ámbito global, proyecto, rol o agente

Implicación práctica:

- evitar volver a ficheros dispersos como fuente principal de comportamiento
- usar catálogo persistente como capa viva de briefing

### 2.6. Control real del ciclo de vida de agentes

Doctrina:

- identidad estable por agente
- continuidad reanudable
- runtime handles, runtime orders, mailbox, checkpoints, watchdog y handoff
- worktrees y ramas aisladas cuando haga falta

Implicación práctica:

- Orquesta debe controlar de verdad arranque, pausa, continuidad y relevo
- el sistema no debe depender de recordar manualmente por dónde iba cada agente

### 2.7. Gobierno Git y trazabilidad

Doctrina:

- commit no equivale a terminado
- push no equivale a integrado
- ramas, merges, worktrees y revisión deben estar gobernados por Orquesta

Implicación práctica:

- mantener visibilidad de ramas, dirty state, merges y decisiones de integración

### 2.8. Memoria y conocimiento persistente

Doctrina:

- memoria de proyecto
- memoria de tarea
- fuentes
- hallazgos
- decisiones
- documentación externa referenciada

Implicación práctica:

- el conocimiento no debe perderse entre sesiones ni agentes
- el sistema debe favorecer continuidad real, no reinicios desde cero

---

## 3. Estado real de la implementación

## 3.1. Lo que ya funciona o tiene base sólida

### Operativa base

- `orquesta serve` arranca correctamente
- healthcheck local operativo
- panel web funcional
- CLI usable
- API amplia

### Dominio funcional visible

- tareas
- propuestas
- votos
- sesiones
- proyectos
- conectores
- reglas / skills / workflows
- memoria / decisiones / documentación referenciada

### Runtime / control plane (base ya existente)

- runtime orders
- runtime handles
- checkpoints
- observabilidad pasiva
- mailbox
- watchdog
- control de agentes expuesto por CLI/API/web

### Skills y catálogo

- catálogo base implementado
- versionado y auditoría
- metadata normalizada
- anti-duplicado funcional
- superficie CLI/API/web ya bastante trabajada

### Server-first parcial

- existe daemon oficial
- existe gating y delegación parcial al servidor
- ya hay dirección clara de cliente fino

---

## 3.2. Brechas activas más importantes

### Brecha A — `cmd/` sigue demasiado acoplado a `db.*`

Síntoma:

- demasiada lógica de negocio en adaptadores CLI/API/web
- demasiadas referencias directas a `db.*`

Riesgo:

- dificulta cumplir hexagonalidad
- mantiene al servidor como centralización táctica, no como orquestador canónico limpio

### Brecha B — server-first aún no está completamente cerrado

Síntoma:

- persisten rutas con `ensureLocalDB()`
- el modo local todavía contamina flujos de negocio
- scripts manuales siguen existiendo en el camino operativo

Riesgo:

- contradice AP-077
- duplica caminos de verdad
- dificulta depuración y trazabilidad

### Brecha C — runtime / continuidad / control plane todavía híbridos

Síntoma:

- órdenes runtime y runtime handles existen, pero no siempre derivan en ciclo de vida real limpio
- acumulación de mailbox/watchdog antiguos en algunos agentes
- sesiones/handles degradados o fallidos
- todavía hay mezcla entre scripts manuales y server-first

Riesgo:

- el plano de control existe pero no siempre gobierna con limpieza
- el sistema parece más cerca de “medio controlado” que de “control operativo pleno”

### Brecha D — i18n con contrato desalineado

Síntoma:

- el esqueleto i18n y el cargador no comparten exactamente el mismo formato operativo

Riesgo:

- el contrato aprobado no puede materializarse de forma coherente en todas las superficies

### Brecha E — seguridad declarada más fuerte que la operativa viva

Síntoma:

- la política/documentación de seguridad es más ambiciosa que la implementación demostrada
- hay validaciones de seguridad, pero no todos los bloques declarados parecen cerrados en práctica

Riesgo:

- falsa sensación de cierre en capas sensibles

### Brecha F — rutas y scripts heredados

Síntoma:

- existencia de rutas antiguas, wrappers y documentación con deriva
- restos de entornos anteriores (`/home/alberto/...`) y posibles inconsistencias de launcher

Riesgo:

- rompe arranques, continuidad y claridad operativa

---

## 4. Ejes de producto identificados

## 4.1. Núcleo de plataforma

Incluye:

- daemon único
- control plane
- API/CLI/web como clientes
- conectores
- runtime y sesiones
- persistencia mediada

Estado:

- muy avanzado en intención
- funcional en buena parte
- aún no completamente consolidado

## 4.2. Arquitectura y gobernanza técnica

Incluye:

- hexagonalidad
- política arquitectónica por tipo de proyecto
- selección de lenguaje
- contratos obligatorios
- gates

Estado:

- doctrinalmente fuerte
- parcialmente aplicado
- con deuda estructural todavía importante

## 4.3. Runtime y autonomía de agentes

Incluye:

- control de ciclo de vida
- runtime orders
- watchdog
- handoff
- nudge
- observabilidad
- arbitraje y discordia

Estado:

- base real existente
- todavía no suficientemente redonda en operación viva

## 4.4. Conocimiento y memoria

Incluye:

- memoria de entidades
- decisiones
- historial de votaciones
- documentación externa
- deriva

Estado:

- bastante maduro en dirección e implementación parcial/alta

## 4.5. UX operativa

Incluye:

- panel web
- dashboard runtime
- A2UI
- futura app de escritorio
- experiencia de control en tiempo real

Estado:

- panel actual funcional
- A2UI y UX más rica todavía abiertas o parciales

## 4.6. Producto expandido

Incluye:

- análisis de apps externas
- asistente guiado
- informe final
- exportación
- empaquetado
- despliegue remoto
- premium / fábrica de clones

Estado:

- doctrinalmente aprobado en muchas piezas
- desigual en madurez según frente

---

## 5. Prioridades operativas recomendadas

## Prioridad 1 — Consolidar la base operativa del orquestador

### Objetivo

Hacer que Orquesta controle de verdad el ciclo de vida de agentes y que el camino server-first sea el principal sin ambigüedad.

### Subfrentes

1. cerrar el dispatcher real de `runtime_orders`
2. revisar continuidad y bootstrap de reanudación
3. sacar scripts manuales del camino principal
4. corregir coherencia entre proyecto actual, continuidad y lanzamiento desatendido
5. añadir humo E2E con runtime vivo real

### Tareas asociadas ya existentes

- `#344`
- `#405`
- `#406`
- `#407`
- `#408`

---

## Prioridad 2 — Reducir la deuda estructural hexagonal y server-first

### Objetivo

Reducir la mezcla de adaptadores y persistencia para que el servidor sea realmente el plano de control canónico.

### Subfrentes

1. extraer lógica de `cmd/` a servicios/puertos
2. reducir referencias directas a `db.*`
3. cerrar los huecos de `ensureLocalDB()` en flujos de negocio
4. terminar de llevar web/API al patrón de cliente fino
5. cerrar cobertura CLI/API de persistencia y diagnóstico

### Tareas asociadas ya existentes

- `#224`
- `#226`
- `#227`
- `#244`
- `#337`

---

## Prioridad 3 — Alinear contratos aprobados con implementación real

### Objetivo

Evitar que la política viva del proyecto diverja de la base técnica operativa.

### Subfrentes

1. alinear contrato i18n con cargador real
2. depurar documentación técnica con deriva
3. revisar rutas hardcodeadas/legadas
4. limpiar scripts y launcher desalineados
5. alinear manuales con la realidad server-first

---

## Prioridad 4 — Después, consolidar capacidades diferenciales abiertas

### Objetivo

Subir de una base operativa correcta a una plataforma claramente superior.

### Frentes abiertos principales

- handoff fuerte (`OP-089`)
- nudge/probing (`OP-090`)
- memoria de entidades más fuerte (`OP-091`)
- time travel (`OP-092`)
- refinería (`OP-093`)
- A2UI (`OP-094`)
- orquestación mixta / arbitraje (`OP-095`)
- MCP más redondo (`OP-088`)
- premium / fábrica de clones (`OP-115`)

---

## 6. Frentes abiertos que no deben olvidarse

## 6.1. i18n

No es cosmética. Es parte del contrato del producto cuando aplica.

## 6.2. Hexagonalidad

No es “refactor bonito”. Es gate arquitectónico.

## 6.3. Persistencia mediada

No es preferencia. Es política operativa aprobada.

## 6.4. Skills/reglas/workflows

No son adornos. Son parte del sistema de gobierno vivo del comportamiento de los agentes.

## 6.5. Deploy / packaging / análisis / asistente guiado

Son parte real de la ambición de producto y no deben perderse, aunque no sean la prioridad estructural inmediata.

---

## 7. Riesgos a vigilar

### Riesgo 1 — Abrir más frentes antes de cerrar la base

Consecuencia:

- aumentará la deuda híbrida
- el control plane seguirá siendo irregular
- las nuevas funciones se apoyarán sobre base inestable

### Riesgo 2 — Confundir “hay una UI” con “la plataforma ya gobierna”

Consecuencia:

- sensación falsa de cierre operativo
- más runtime zombie y más caminos laterales

### Riesgo 3 — Seguir usando caminos manuales por comodidad

Consecuencia:

- contradicción con AP-077 y server-first
- pérdida de trazabilidad
- duplicación de verdad operativa

### Riesgo 4 — Tratar i18n o hexagonalidad como adornos postergables sin límite

Consecuencia:

- deuda estructural creciente
- incumplimiento de doctrina aprobada
- re-trabajo posterior mucho más caro

---

## 8. Recomendación operativa de trabajo para Berserk y agentes

### Regla 1

Antes de tocar código, identificar si el problema cae en:

- base operativa
- deuda estructural
- alineación de contratos
- feature diferencial

### Regla 2

Si un problema afecta a runtime, continuidad, arranque, persistencia mediada o server-first, debe tratarse como prioridad alta.

### Regla 3

Si un frente ya tiene tarea abierta en la BD, reutilizarla como marco de trabajo en vez de inventar una tarea paralela salvo necesidad real.

### Regla 4

Si se detecta una brecha operativa que obliga a workaround manual, debe abrirse o reutilizarse tarea para eliminar ese workaround.

### Regla 5

Cuando una capacidad nueva compita con el cierre de la base, priorizar primero la base salvo decisión explícita de Alberto.

---

## 9. Lista operativa de siguientes pasos recomendados

### Paso 1

Auditar técnicamente los frentes:

- `#344`
- `#405`
- `#406`
- `#407`
- `#408`

Objetivo: saber cuáles están de verdad medio resueltos, cuáles solo están modelados, y cuál conviene atacar primero.

### Paso 2

Inventariar y corregir incoherencias de launcher/rutas/scripts ligadas al runtime real de agentes.

### Paso 3

Cruzar el estado del control plane con los agentes reales (Codex1..Codex6, etc.) para distinguir:

- runtime vivo
- handle fallido
- cola vieja
- continuidad recuperable
- basura operativa

### Paso 4

Atacar el primer frente técnico prioritario con ejecución real:

- arreglo directo por Berserk si es pequeño y claro
- orden a agente vía Orquesta si conviene delegar

---

## 10. Criterio de éxito operativo a corto plazo

Se considerará que Orquesta entra en un estado operativo mucho más sano cuando se cumplan estos mínimos:

1. el daemon sea el camino real dominante
2. el arranque/continuidad de agentes no dependa de scripts heredados
3. `runtime_orders` gobierne ciclo de vida real, no solo estado modelado
4. las sesiones y handoffs se puedan seguir sin pérdida de contexto
5. la web/CLI/API operen como clientes finos de un control plane real
6. las brechas de persistencia local queden reducidas al modo recuperación

---

## 11. Política de continuidad operativa

Mientras Alberto no indique lo contrario, Berserk debe operar sobre Orquesta en modo continuidad.

### Regla operativa

- no quedarse solo en diagnóstico cuando ya exista trabajo ejecutable
- encadenar análisis, implementación, delegación, validación y siguiente paso
- usar Orquesta para delegar trabajo de programación cuando compense
- priorizar la consolidación real de la app sobre respuestas parciales o puramente teóricas
- mantener alineación con la doctrina aprobada del proyecto
- trabajar en modo loop: revisar periódicamente tareas, agentes, bloqueos y progreso sin esperar nuevas instrucciones humanas
- supervisar en silencio mientras no exista bloqueo real

### Cuándo sí debe detenerse

- si hace falta una decisión humana real
- si la acción es destructiva, irreversible o de riesgo alto
- si faltan credenciales, permisos o accesos imprescindibles
- si hay riesgo claro de romper datos, repos o continuidad operativa
- si Alberto pide pausa, auditoría o cambio de prioridad

Esta política convierte el desarrollo de Orquesta en un trabajo continuo y no en una secuencia de diagnósticos aislados.

## 12. Nota final

Este informe debe tratarse como documento operativo vivo.

No reemplaza:

- `orquesta_v1_vision.md`
- `orquesta_v1_roadmap.md`
- las OP aprobadas
- las tareas de la BD

Su valor está en recordar, durante el desarrollo diario, qué es lo esencial, qué está pendiente de cerrar y qué no conviene olvidar mientras se sigue ampliando el producto.
