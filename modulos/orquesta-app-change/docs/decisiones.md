# Decisiones: orquesta-app-change

```text
Fecha: 2026-05-10
Decision: El cambio de app exige persistencia y notificacion al director por
puertos separados.
Motivo: aceptar un cambio sin durabilidad o sin que el director lo vea repetiria
el fallo de orquestacion manual oculta.
Impacto: el caso de uso falla de forma publica si faltan puertos; la DB y el
canal concreto son conectores externos.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Un cambio concreto puede alimentar una fuente de decisiones del
director, pero solo si trae write-set y criterios de aceptacion.
Motivo: convertir texto libre sin limites en microtareas repetiria el fallo de
v1/v2: alcance inventado y bucles de arreglo. Si faltan datos, debe quedar como
consulta al director.
Impacto: `AppChangeRecordStorePortV0` separa guardado/listado; el stack compone
una fuente de decisiones que respeta la cadena vote -> decision -> contrato ->
microtarea.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: La primera integracion del cambio entra como pregunta durable al
director.
Motivo: el usuario puede pedir cambios en lenguaje natural, pero Orquesta no
debe decidir manualmente microtareas desde el transporte. El director debe
recibir una senal compacta y decidir fases/agentes.
Impacto: el stack traduce `AppChangeRequestV0` a `AskDirector` por puertos del
workflow y guarda el outbox de director. La replanificacion completa queda como
APP-CHANGE-003.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El cambio a mitad de ejecucion puede entrar como
`AppChangeIntentEventV0`, pero se convierte inmediatamente a
`AppChangeRequestV0`.
Motivo: formulario, API y MCP necesitan registrar "quiero modificar X" como
senal idempotente; duplicar otro flujo de cambio romperia el contrato ya
aceptado por store, notificador y fuente de director.
Impacto: el evento solo normaliza refs y metadatos. La validacion, persistencia
y notificacion siguen pasando por el caso de uso existente.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-13
Decision: `external_work` expone `job_ref` como referencia opaca opcional del
trabajo externo.
Motivo: el retorno de artefactos necesita un job estable y no debe inferirlo
desde `work_refs` ni desde prefijos de una app concreta como OPES.
Alternativas: usar el primer `work_ref`; crear reglas por dominio externo;
consultar internals de la app propietaria.
Impacto: si la app externa ya creo el job, envia `job_ref`; Orquesta lo
normaliza y lo conserva como ref de trabajo para trazabilidad y devolucion de
artefactos.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-13
Decision: `external_work` puede transportar `input_fields` opcionales usando
`orquesta-domain-work.DomainWorkFieldV0`.
Motivo: OPES y otras apps externas necesitan pasar contexto de dominio acotado
al trabajo externo sin inflar el contexto global ni convertirlo en JSON libre.
Alternativas: duplicar un DTO de campos en app-change; meter todo en
`user_intent`; ampliar refs globales con datos de dominio.
Impacto: app-change normaliza esos campos con el contrato de domain-work,
rechaza nombres no compactos y los conserva como `InputFields` al construir
`DomainWorkJobRequestV0`. La semantica de cada campo sigue perteneciendo al
dominio externo.
Estado: aceptada localmente.
```

```text
Fecha: 2026-05-13
Decision: Las apps externas declaran trabajo de dominio mediante
`external_work` opaco, no integrando su nucleo en Orquesta.
Motivo: OPES u otra app de dominio debe seguir siendo propietaria de sus
temarios, fuentes, bloques o artefactos. Orquesta solo necesita saber que existe
un trabajo externo acotado y verificable para convertirlo en microtareas de
agentes.
Alternativas: importar la app externa como modulo de Orquesta; meter rutas,
endpoints o jobs concretos en el core; tratarlo como texto libre sin contrato.
Impacto: `AppChangeRequestV0` y `AppChangeIntentEventV0` aceptan
`external_work` con refs compactas. La interpretacion concreta queda en
adaptadores/contratos externos; el modulo no conoce REST, MCP, DB, OPES,
runtime ni proveedor.
Estado: aceptada localmente.
```
