# Decision de arquitectura: el estado durable sigue en ficheros (2026-07-12)

Decidido por el operador tras revision del estado del arte por el revisor.

## La pregunta

¿Deberia Orquesta guardar su estado en una base de datos (con vectores u otras
tecnicas) en vez de en ficheros, como hacen otras herramientas del sector?

## Lo que hace el sector (verificado, julio 2026)

Nadie usa "una base de datos para todo". El patron es por CAPAS:

| Capa | Contenido | Almacen habitual |
|---|---|---|
| Estado durable del workflow | runs, pasos, checkpoints | BD (Postgres/DynamoDB) o motor durable (Temporal) |
| Memoria semantica | conocimiento recuperable | Vector DB |
| Estado estructurado | usuarios, logs, metricas | Relacional |
| Artefactos | ficheros generados, evidencias | Ficheros / object storage |

Dos matices decisivos:

1. **Checkpointing != ejecucion durable.** LangGraph guarda estado entre nodos,
   no DENTRO de un nodo: protege de fallos de aplicacion, no de infraestructura.
   Es exactamente el fallo H2 de Orquesta (claim de atestacion que no sobrevive
   a que el proceso muera a media operacion). **Una BD no lo arregla sola**: hace
   falta lease/expiracion, que es codigo, no motor.
2. **Los vectores son inadecuados para estado que cambia.** Un vector store no
   sabe que guarda, no distingue una preferencia de una frase suelta y no tiene
   noción de tiempo ni de contradiccion: produce **errores silenciosos de
   recuperacion**. Para hechos que cambian (si un test paso, quien atesto que)
   es la peor opcion posible. "Error silencioso" es la palabra prohibida de este
   proyecto.

## Decision

**El estado durable del nucleo sigue en ficheros** (`orquesta-state-file`:
JSON/JSONL bajo StateDir, escritura atomica y CAS). Razones:

- El estado de Orquesta es **evidencia**, no datos de aplicacion: receipts,
  atestaciones, refs. Su auditabilidad directa (`cat` sobre el receipt) es un
  activo central para un sistema cuyo problema es **no fiarse de lo que dicen
  los agentes**. Esa propiedad se ha usado en real para desmontar sospechas de
  falso verde.
- La concurrencia real es baja (unos pocos goals en paralelo): no es el problema
  que una BD resuelve.
- Migrar el store **con H2 abierto** seria mover los cimientos con una grieta
  activa.

## Mejora futura (registrada, no planificada)

- **Vectores**: no para el estado, sino para memoria de codigo (busqueda
  semantica sobre el repo). Aditivo; ya parcialmente cubierto por el broker de
  contexto.
- **Postgres**: para las apps que Orquesta genera (`orquesta-persistence` existe
  justo para eso), no para Orquesta.
- **Motor de ejecucion durable** (estilo Temporal): seria la respuesta seria a
  H2/H3 a nivel infraestructura. Implica reescribir el corazon; se evalua solo
  con el sistema sano y sin frentes abiertos.

## Consecuencia inmediata

El problema real no es DONDE se guarda el estado, sino que:

- **H2**: faltan locks con lease/expiracion (carrera de atestacion).
- **H3**: el cierre Goal-first no dispara promocion.

Una base de datos no regala ninguna de las dos si el codigo no las pide.
Se arreglan primero; la migracion queda como mejora futura.
