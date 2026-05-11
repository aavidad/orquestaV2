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
