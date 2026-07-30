# Bloqueo de propiedad de la vista estable y la reserva física

Fecha: 2026-07-30.

Estado: contradicción de autoridad abierta; ejecución real en **NO-GO**.

## Encargo y frontera

```text
capability IDs: candidatas STG-17, EVD-08 y EVD-10; ninguna capacidad decidida
invariante: un contrato de aceptación no se convierte en capacidad ni propietario
autoridad que escribe: ninguna mientras el roadmap no resuelva el hueco
puertos afectados: ninguno
adaptadores afectados: ninguno
write-set: este bloqueo y su prueba estructural
dependencias causales: plan condicionado -> hueco de propiedad -> decisión atómica futura en roadmap
código antiguo que permitirá retirar: ninguno antes del censo acreditado
test de contrato: estructura vigente del roadmap, separación de recibos y NO-GO
negativo/mutación: atribuir propiedad o autorizar autenticidad física, vista, reserva o montaje
E2E/compuerta: no ejecutado; ninguna raíz histórica ni infraestructura se consulta
presupuesto: documento <= 220 líneas; prueba <= 190 líneas; cero efectos
```

Las consultas locales de lecciones para `GOV-16` y `STG-02` no devolvieron
patrones. El hueco no permite resolver la propiedad por similitud.

## Contradicción registrada

El plan del censo exige una vista estable, cercado, salida segura, reserva y
recibos antes de abrir una fuente. El roadmap vigente no asigna expresamente
la propiedad de esa infraestructura a una capacidad.

`AC-V34-CUTOVER-ACCREDITATION` es un **contrato de aceptación planificado**, no
una capacidad. Sus cuatro aserciones actuales tampoco nombran vista estable,
cercado, salida física ni reserva. Por tanto, no puede ser declarado
propietario ni usado como autorización implícita.

Las capacidades vigentes más cercanas son:

| Capacidad | Estado y contexto | Encaje posible, no decidido |
|---|---|---|
| `STG-17` | `declared`, `cutover_accreditation` | coordinación E2E, validación final, adquisición y restitución puntuales |
| `EVD-08` | `declared`, `cutover_accreditation` | evidencia de integración, conflictos y selección explícita |
| `EVD-10` | `declared`, `cutover_accreditation` | mutaciones de invariantes de cercado, reserva, recibos y recuperación |

Las tres apuntan al contrato V34, pero esa coincidencia no decide propiedad.
El ratchet vigente es `ninguna capacidad decidida`.
`GOV-16` solo gobierna el ledger canónico; `STG-02` y `WorkspaceManager`
gobiernan espacios de trabajo ordinarios; el arrendamiento del Director
autoriza propuestas sobre un Goal. Ninguno adquiere por ello montajes,
instantáneas, exclusión de escritores o capacidad física.

## Recomendación pendiente de autoridad

La recomendación más pequeña es que una futura decisión atómica del roadmap:

1. asigne a `STG-17` la coordinación puntual de adquisición, uso, liberación y
   restitución, sin convertirla en servicio residente;
2. limite `EVD-08` a evidencia de reconciliación y selección ante conflictos;
3. limite `EVD-10` a negativos y mutaciones de invariantes críticos;
4. amplíe las aserciones de `AC-V34-CUTOVER-ACCREDITATION` para exigir la
   misma composición, candidato inmutable, recibos y cero residuos;
5. declare si esas capacidades bastan o si el catálogo necesita otra decisión,
   sin crearla desde este documento.

Es una recomendación, no una decisión. El roadmap está ocupado por cambios V38
ajenos a este conjunto de escritura. Cuando quede libre, la resolución debe
hacerse allí en un único cambio revisable antes de programar. Este documento
no sustituye el roadmap ni acredita estado de producto.

## Alternativa de seguridad que no se reinterpreta

La alternativa sellada para impedir que la salida entre en una fuente es
exactamente:

```text
dispositivo distinto O vista externa cercada
```

No se endurece a exigir ambas ramas ni se debilita a una comparación léxica.
La estabilidad de la fuente sigue siendo una compuerta propia. La rama elegida
debe quedar acreditada durante toda la ventana correspondiente. En el estado
actual no está acreditada ninguna y permanece el **NO-GO**.

## Seis recibos de infraestructura

Estos seis recibos pertenecerían a la futura composición de infraestructura.
Se enumeran para impedir omisiones, no para autorizar su emisión:

| Identificador | Hecho separado |
|---|---|
| `recibo_autorizacion_operativa` | principal, alcance, ventana, caducidad, operaciones y aprobación |
| `recibo_quiescencia_cercado` | escritores inventariados, parada o exclusión, token y vigencia |
| `recibo_adquisicion_vista` | identidad, solo lectura, no-atime, montaje o instantánea y cercado |
| `recibo_aptitud_salida` | rama elegida, identidad, POSIX, propietario y permisos efectivos |
| `recibo_reserva_fisica` | cuota o preasignación real mínima de 44 GiB, margen y consumo; no `df` ni fichero disperso |
| `recibo_liberacion_restitucion` | liberación, restitución autorizada y cero recurso propio residual |

Los cinco primeros deben existir y validarse antes del mapeo. El sexto solo se
emite después de la revisión, al liberar y restituir. Ninguno se deduce de
otro. Intento, aprobación, llamada alcanzable, mensaje humano o recibo del
censo no sustituyen un recibo de infraestructura.

## Cuatro artefactos del censo

Los artefactos del censo continúan siendo otro conjunto, con los cuatro
dominios ya sellados por el plan:

| Dominio | Artefacto |
|---|---|
| `orquesta.physical-census-subject.v1` | manifiesto del sujeto |
| `orquesta.physical-census-attempt-receipt.v1` | recibo de intento |
| `orquesta.physical-census-batch-receipt.v1` | recibo de lote |
| `orquesta.physical-census-confirmation.v1` | confirmación de publicación |

Cada artefacto conserva su propia identidad y encuadre. Ninguno acredita,
infiere ni sustituye los seis recibos de infraestructura; tampoco uno de esos
seis recibos demuestra sujeto, intento, lote o confirmación del censo.

## Cadena causal bloqueada

```text
decisión_atómica_roadmap
  -> autorización y composición de infraestructura
  -> cinco recibos infraestructurales previos
  -> mapeo -> sujeto -> bruto -> recibo de intento -> confirmación
  -> recibo de lote -> revisión independiente del mismo candidato
  -> liberación y restitución -> sexto recibo infraestructural final
  -> evidencia V34
```

La primera flecha no existe todavía. No se puede saltar usando un documento,
un contrato planificado, una prueba unitaria o un recibo de otra capa.

## Frontera exacta de validación

Está permitido ahora bajo `GOV-16` un validador puro de los bytes V3, universo
y candidato aportados explícitamente. Solo puede certificar forma, ligadura por
bytes y coherencia interna. No resuelve propiedad ni modifica el estado de una
capacidad.

Ese validador puro consume bytes de artefactos y sus sellos; no consume
descriptores de fichero persistidos: un descriptor solo vale en el proceso que
lo mantiene abierto. Tampoco abre raíces, consulta montajes o espacio, detiene
escritores, reserva, monta, copia, borra, libera ni restituye.

Permanece bloqueado cualquier validador o adquiridor que afirme propiedad o
autenticidad física de vista, cercado, salida, reserva o recibos. Esas
afirmaciones requieren la futura decisión de capacidad, efectos autorizados,
observación física y recibos propios; no se derivan de la coherencia interna.

## Negativos de la futura decisión física

- contrato V34 presente sin capacidad propietaria decidida: cero efectos;
- recibo ausente, manipulado, duplicado, de otra revisión o sujeto: no cierre;
- pérdida de identidad, cercado, solo lectura, no-atime o rama de salida:
  aborto sin lectura ordinaria alternativa;
- menos de 44 GiB reales, `df`, fichero disperso o promesa humana: reserva no
  acreditada;
- artefacto de censo usado como recibo de infraestructura, o al contrario:
  rechazo;
- descriptor serializado o reabierto por nombre como identidad durable:
  rechazo;
- liberación anterior al sello de la copia: copia inválida;
- recurso propio vivo o restitución no acreditada: V34 no cierra.

## Estado y cierre

Permanece **NO-GO**: no hay decisión de capacidad en roadmap, autorización,
vista, rama de salida, reserva ni recibos. No se autoriza implementar la
composición física, adquirir, montar, reservar, afirmar autenticidad física ni
abrir raíces reales.
Este cierre físico no prohíbe el validador puro de bytes limitado por
`GOV-16`, que no abre esas compuertas.

```text
hecho: contradicción y candidatos registrados; validador lógico separado del físico
invariante restaurado: contrato de aceptación no equivale a capacidad propietaria
autoridad final: futura decisión atómica de product/roadmap.json
tests/negativos/mutaciones/E2E: prueba estructural; E2E no ejecutado
recibos y revisión acreditada: ninguno; seis infraestructurales y cuatro del censo siguen separados
código o decisión retirados: atribución de propiedad y autorización prematura de un validador físico
legacy retirado o bloqueo de retirada: toda retirada sigue bloqueada
LOC netas y complejidad: documento y prueba sin efectos
riesgos/P0/P1: todo efecto o afirmación física antes de resolver el roadmap continúa bloqueado
siguiente dependencia causal: liberar write-set V38 y resolver capacidad/aserciones en roadmap
```
