# Capacidades declaradas que NUNCA se ejecutan (2026-07-12)

Auditoria del revisor a peticion del operador: *"busca todas las funciones que
he pedido como esta [el consejo de decision] y que nunca se ejecutan"*.

**Metodo (mecanico, no a ojo):** para cada modulo, comprobar si la composicion
real (`cmd/orquesta-server` + `orquesta-app-codex-stack`) lo importa. Si nadie
lo importa, **no puede ejecutarse jamas**, por muy completo que este su codigo.

## Resultado: 7 capacidades muertas

| Capacidad | Que es | Server | Stack | Tool MCP | Veredicto |
|---|---|:--:|:--:|:--:|---|
| **Consejo de decision** (`resident_director_council`) | Grupo de discusion/votacion entre agentes | — | declarado | — | **VoteSource nunca construido**; solo se invoca desde el briefing del director residente, y no forma parte del ciclo del goal |
| **`orquesta-document-extraction`** | Extraer campos tipados de documentos | 0 | 0 | NINGUNA | Muerta: nadie la importa |
| **`orquesta-data-ingestion`** | Ingesta de datos (CSV/JSON) con receipts | 0 | 0 | NINGUNA | Muerta: nadie la importa |
| **`orquesta-presentation-extraction`** | Extraccion de presentaciones | 0 | 0 | NINGUNA | Muerta: nadie la importa |
| **`orquesta-autonomy-program`** | Programa padre durable con nodos goal/tarea (DAG) | 0 | 0 | NINGUNA | Muerta: nadie la importa |
| **`orquesta-document-plan-expander`** | Convierte plan documental en jobs de dominio | 0 | 0 | NINGUNA | Muerta: nadie la importa |
| **`orquesta-work-profiles`** | — | 0 | 0 | — | Placeholder historico, **sin codigo ejecutable** (0 ficheros .go) |

Ademas, sin cablear pero de infraestructura (no son "features pedidas"):
`orquesta-cli`, `orquesta-domain-work-sql`, `orquesta-domain-work-memory`,
`orquesta-run-memory`, `orquesta-agent-process-registry-memory`,
`orquesta-native-smoke-tool` (el que creo Orquesta hoy),
`orquesta-runtime-required-test` (este SI se usa: via wiring opt-in).

## El patron, que ya es sistemico

Es **la misma enfermedad** que llevamos todo el dia destapando:

- **H1b**: seis tools registradas en MCP y con el puerto a `nil` → anunciadas y
  muertas.
- **H4**: 99 funciones inalcanzables, incluidas **validaciones** que creiamos
  activas (`ValidateStrictEventSequenceV0`, presupuesto de payload, leases).
- **H5**: el consejo de decision, declarado y jamas cableado.
- **Y ahora**: **cinco modulos de capacidad completos** (extraccion documental,
  ingesta de datos, presentaciones, programa de autonomia, expander de planes)
  que **la composicion ni siquiera importa**.

**Orquesta tiene mas cosas declaradas que enchufadas.** El codigo existe, tiene
tests propios, pasa el build... y **no esta en ningun camino de ejecucion**.

## Por que importa (y no es cosmetico)

1. **El operador pidio estas capacidades y no existen en la practica.** Creer
   que se tienen y no tenerlas es peor que no tenerlas.
2. **Da falsa sensacion de completitud**: el repo parece rico, el producto es
   mas pobre.
3. **Deuda que crece**: cada modulo muerto se mantiene, se compila, se testea y
   ocupa presupuesto de envs/contexto, sin devolver nada.
4. **Y una lección para el proceso**: nuestros tests verdes no detectan esto.
   Un modulo con 100% de cobertura y cero consumidores **pasa todos los guards**.

## Que hacer con cada una (decision del operador, una por una)

Igual que en H4, hay tres salidas y **ninguna es "dejarlo como esta"**:

- **CABLEAR** — la capacidad es buena y el operador la quiere: se conecta al
  ciclo real (tool MCP + endpoint + wiring) y se prueba con un run de verdad.
- **RETIRAR** — no se va a usar: se saca del arbol (git conserva la historia).
- **CONGELAR CON MOTIVO** — se conserva a proposito para un frente futuro, y
  **se documenta en su README que hoy NO esta enchufada**, para que nadie crea
  que existe.

Lo que **no** puede seguir pasando es que haya capacidades enteras en el limbo,
sin que nadie sepa que estan muertas.
