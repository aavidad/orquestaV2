<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-050 — Orquestador jerárquico y presupuesto de sesión

## Motivo

El entorno real de trabajo no es uniforme.

Licencias actuales conocidas:

- `Codex`: 4 licencias
- `Claude`: 1 licencia
- `Android`: 1 licencia

La arquitectura objetivo debe aprovechar esa realidad en vez de ignorarla.
Ese inventario debe ser editable: aumentar o disminuir capacidad, cambiar planes y añadir proveedores futuros como `Grok`.

## Problema a resolver

Se necesita un modelo mixto:

1. Un orquestador superior decide reparto global entre proyectos, prioridades y fases.
2. Cada licencia o runtime puede abrir agentes hijos cuando convenga.
3. Todo ello sigue gobernado por una capa superior de trazabilidad, sesiones y estado.

Ademas, cada runtime tiene limites de sesion o de presupuesto:

- tiempo restante hasta siguiente sesion
- tiempo restante hasta limite semanal
- consumo de tokens o contexto operativo

Si no se controla eso, el trabajo puede quedar cortado en mitad de una tarea.

## Propuesta base

### 1. Orquestador de orquestadores

Orquesta debe actuar a dos niveles:

- `nivel superior`
  decide estrategia global del workspace
- `nivel de pool/licencia`
  decide cuantos agentes hijos conviene abrir dentro de cada runtime o licencia

Ejemplo:

- pool `codex`: capacidad 4
- pool `claude`: capacidad 1
- pool `android`: capacidad 1

El orquestador superior reparte trabajo entre pools.
Cada pool puede abrir uno o varios agentes hijos dentro de su capacidad.

### 2. Pools de capacidad

Debe existir una entidad tipo `pool` o `capacidad_runtime` con:

- `slug`
- `proveedor`
- `runtime`
- `plan`
- `es_de_pago`
- `capacidad_total`
- `capacidad_disponible`
- `politica_delegacion`
- `permite_hijos`
- `permite_sobrecoste`
- `fuente_telemetria`

No debe modelarse solo como “número de licencias”.
Debe servir también para:

- planes gratuitos
- planes de pago
- límites por ventana de tiempo
- límites semanales
- límites por créditos
- límites por modelo
- límites por equipo

### 3. Agentes hijos

Los agentes hijos no rompen el modelo de identidad estable.

Pueden representarse como:

- `Codex1`, `Codex2`, `Codex3`, `Codex4`
- `Claude1`
- `Android1`

Pero deben saber a que pool pertenecen y cuanto presupuesto les queda.

### 4. Presupuesto de sesión

Cada sesion debe poder guardar:

- `session_budget_seconds`
- `session_budget_remaining_seconds`
- `weekly_budget_seconds`
- `weekly_budget_remaining_seconds`
- `token_budget_remaining`
- `budget_source`
- `budget_checked_at`
- `reset_at`
- `window_kind`
- `provider_limit_snapshot_json`

No todos los runtimes expondran todos los campos.
El modelo debe aceptar datos parciales.

## Telemetria que la app debe recoger

La app debe soportar varias fuentes de información:

1. `CLI status`
   Cuando el runtime muestra tiempo restante o presupuesto en comandos o pantallas de estado.

2. `API o consola del proveedor`
   Cuando el proveedor expone límites, rate limits, consumo o pricing.

3. `Configuracion manual`
   Cuando no exista telemetría fiable, Orquesta debe permitir definir políticas fijas.

4. `Inferencia operativa`
   Cuando solo se pueda estimar a partir de hora de inicio, reinicios, consumo observado o eventos previos.

## Modelo recomendado para la app

### Pool de capacidad

- `id`
- `slug`
- `proveedor`
- `runtime`
- `plan`
- `es_de_pago`
- `capacidad_total`
- `capacidad_reservada`
- `permite_hijos`
- `permite_modelos_multiples`
- `politica_handoff`
- `fuente_telemetria`
- `metadata_json`

### Modelo habilitado en un pool

- `pool_id`
- `model_slug`
- `activo`
- `prioridad`
- `coste_relativo`
- `limite_conocido_json`

### Presupuesto de sesion

- `sesion_id`
- `pool_id`
- `model_slug`
- `window_kind` (`5h`, `weekly`, `monthly`, `rolling`, `unknown`)
- `window_started_at`
- `reset_at`
- `remaining_seconds`
- `remaining_messages`
- `remaining_tokens`
- `remaining_credits`
- `source`
- `raw_snapshot_json`

### 5. Handoff preventivo

Cuando el presupuesto restante baje por debajo de umbral:

- no se arranca trabajo largo nuevo
- se fuerza checkpoint
- se exige `resumen_continuidad`
- se guarda `external_session_id`
- se propone relevo a otro agente del mismo proyecto o del mismo pool

## Politica operativa deseada

### Acciones del supervisor

El supervisor superior debe poder:

- repartir slots entre proyectos
- reservar capacidad por pool
- detectar agotamiento de sesion
- iniciar handoff antes del corte
- reasignar un relevo

### Acciones del agente

El agente debe poder:

- seguir trabajando mientras tenga presupuesto seguro
- consultar su presupuesto restante
- guardar continuidad de forma obligatoria antes del limite
- ceder la tarea a otro agente si el tiempo restante no permite terminar bien

## Preguntas para votacion

1. ¿Conviene modelar `pools/licencias` como entidad explicita de primer nivel?
2. ¿Debe el orquestador superior repartir trabajo a pools y no directamente a licencias sueltas?
3. ¿Debe permitirse que un pool abra agentes hijos por debajo de su capacidad?
4. ¿Que campos minimos debe guardar el presupuesto de sesion?
5. ¿Que umbral dispara el handoff preventivo?
6. ¿Debe reasignarse primero dentro del mismo pool o al mejor agente disponible del proyecto?
7. ¿Debe modelarse la capacidad como pools ampliables con plan y telemetria, en vez de como licencias rigidas?

## Criterios de decision

- aprovechamiento real de licencias
- simplicidad operativa
- continuidad del trabajo
- trazabilidad
- seguridad
- capacidad de crecer a nuevos modelos y nuevos limites futuros

## Recomendacion inicial de Codex1

La mejor opcion parece:

- un orquestador superior global
- pools de capacidad por licencia/runtime
- agentes hijos dentro de cada pool
- control explicito de presupuesto de sesion
- handoff preventivo obligatorio antes de agotar ventana

Pero esta decision no debe cerrarse sin al menos dos votos no autores.
