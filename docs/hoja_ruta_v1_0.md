# Hoja de ruta a Orquesta 1.0

Autor: Claude (director/revisor). Fecha: 2026-07-12.
Decision del operador: **primero v1.0 100% funcional**; multiusuario queda como
mejora posterior (v2), pero **se disena v1.0 sin cerrarle la puerta**.

## Que ya esta (verificado, no autodeclarado)

- **Nucleo**: veredicto causal unico, atestacion independiente de tests (208H),
  gobierno de progreso material, carrera de atestacion cerrada (H2).
- **Conectores**: backend real, bootstrap MCP/HTTP con guard exhaustivo, ciclo
  delivery→review→closure, canal operador-director.
- **Circuito completo probado POR USO**: existe un commit hecho por la propia
  Orquesta (`dda4f5e19`) integrando un modulo que ella escribio.
- **Promocion** desde cierre atestado (H3), sin push automatico.
- **Seis tools** vivas y gobernadas (H1b).

## Lo que FALTA para poder llamarlo 1.0

### V1-A. Configuracion desde la web (hueco confirmado)

**Estado actual, verificado en codigo:**

- `orquesta.config.json` es el fichero canonico, y el servidor **solo lo LEE**:
  cero rutas de escritura (`os.WriteFile` sobre la config: 0 ocurrencias).
- No existe ningun endpoint HTTP de configuracion (`/config`, `/settings`: no
  hay ninguno).
- `MCPConfigProjectionSettingV0` **proyecta** ajustes (los expone y verifica),
  pero **no los edita**.
- Las credenciales viven **fuera** y a mano: `auth.json` en el `CODEX_HOME`,
  `api_key_file` en disco. Nadie las gestiona desde la app.

**Conclusion: hoy Orquesta se configura editando ficheros a mano y
reiniciando.** Eso no es un producto 1.0.

**Lo que hay que construir:**

1. **Lectura de configuracion efectiva por API/web**: que se pueda ver el estado
   real (que hay puesto, de donde viene: fichero, env o default, y si hay
   conflicto). Ya existe la proyeccion; falta exponerla como superficie.
2. **Escritura gobernada de configuracion**: endpoint que valida contra el mismo
   esquema canonico, **escribe atomicamente** y deja **evidencia durable**
   (quien cambio que, cuando, valor anterior). Nada de mutar el estado sin
   receipt: es la regla de la casa.
3. **Recarga sin reinicio** donde sea seguro; y donde no lo sea, decirlo
   explicitamente ("este cambio requiere reinicio") en vez de fingir.
4. **UI web minima**: formulario sobre el mismo esquema, sin logica propia.
5. **Gestion de credenciales**: aqui esta lo delicado (ver V1-B).

### V1-B. Credenciales y OAuth por cuenta (el punto del operador)

El operador lo formulo asi: *"si mañana queremos varios usuarios, cada uno debe
poder configurar su cuenta OAuth para que no se usen los recursos de otro en mi
proyecto, ni al reves"*.

**Es correcto y es una decision de arquitectura, no de UI.** Aunque v1.0 sea de
un solo dueño, **las credenciales deben modelarse como un recurso con dueño
desde ya**, o migrar a multiusuario despues sera una reescritura.

**Diseno para v1.0 (single-tenant) que NO cierra la puerta a v2:**

- Un **almacen de credenciales** con la misma disciplina que el resto del estado
  (fichero durable, permisos 0600, escritura atomica) y una **entrada tipada**:
  `credential_ref` → { proveedor, tipo (oauth/api_key), owner_ref, scope }.
- **`owner_ref` OBLIGATORIO desde v1.0**, aunque hoy siempre valga el mismo
  operador. Es un campo, no una feature: cuesta cero ahora y lo cuesta todo
  despues.
- **El goal recibe una `credential_ref`, nunca el secreto.** El backend resuelve
  la credencial en el momento de lanzar; el secreto **no viaja** por el request,
  ni por el estado, ni por los logs, ni por las evidencias.
- **Nunca se devuelve el secreto por la API**: solo su ref, proveedor y estado
  (valida/caducada). La UI muestra "conectado como X", no la clave.
- **Evidencia durable de todo uso**: que goal uso que credencial. Sin eso no se
  puede auditar "quien gasto mis recursos", que es exactamente lo que el
  operador quiere impedir.

**Regla que se hereda de todo el proyecto:** ninguna operacion sensible sin
rastro. Ya lo aplicamos a `runtime.models` (`pull/serve/stop` con receipt); las
credenciales van igual.

### V1-A2. Selector de modelos e intensidades desde la web (spec del operador)

**Requisito literal del operador:** *"desde la web debemos poder elegir el
modelo que queremos de director o de trabajadores, incluso sus intensidades.
Que salga un listado con los posibles modelos, incluso los locales. Con el que
elegimos por defecto y el porque (un bocadillo de ayuda que explique por que
elegimos ese modelo). Los modelos locales de Ollama, vLLM o el que sea deben
revisarse en tiempo real: comprobar cual podemos usar ANTES de ensenarlos."*

**Buena noticia: media pieza ya existe.** Verificado en codigo:

- El routing **ya es tipado por complejidad de tarea**, con modelo Y esfuerzo
  por nivel (`model_routing_defaults_v0.go`):

      trivial  -> luna  (effort: low)
      normal   -> terra (effort: medium)
      complejo -> (effort: high)
      critico  -> sol   (effort: high)

  Y hay `TaskRoutes` (rutas por tarea concreta), hoy vacio.
- **`runtime.models` (la tool que acabamos de encender en H1b) ya sabe listar
  modelos locales en vivo**: `ListRuntimeModelsV0` devuelve modelos con su
  `Status`. Se cablo justo antes de que hiciera falta.

**Lo que falta construir:**

1. **Catalogo unificado de modelos**, que combine:
   - **remotos** (los del routing: `gpt-5.6-sol/luna/terra`),
   - **locales descubiertos EN VIVO** via `runtime.models` (Ollama y, por
     contrato, cualquier otro proveedor local tipo vLLM).
   Cada entrada: id, proveedor, **disponibilidad real comprobada ahora**
   (`ready` / `not_pulled` / `unreachable`), y capacidades.

2. **Comprobacion de disponibilidad ANTES de mostrar** (esto es lo que pide el
   operador y es la parte fina): un modelo local **no se ofrece si no responde**.
   - Si Ollama no esta arriba: se muestra el proveedor como `unreachable`, no
     sus modelos como elegibles.
   - Si el modelo no esta descargado: se muestra como `not_pulled` con la accion
     "descargar" (que ya existe: `pull`, y **deja receipt** — regla de H1b).
   - **Nunca ofrecer como elegible algo que fallara al lanzarse.** Esa es la
     diferencia entre un selector util y una lista de promesas.

3. **Seleccion por ROL y por INTENSIDAD** desde la web:
   - rol: **director** vs **trabajadores** (hoy el routing es por complejidad,
     no por rol: hay que anadir la dimension rol o mapearla),
   - intensidad: el `reasoning effort` por nivel, ya modelado.

4. **Defaults con explicacion (el "bocadillo")**: cada default trae un texto
   corto de POR QUE. No es adorno: es la diferencia entre que el operador
   confie en el default o lo cambie a ciegas. Ejemplos de la logica real que ya
   aplicamos: *"terra por defecto en tareas normales: equilibrio coste/calidad;
   sol solo en criticas porque es el mas caro; luna en triviales para no gastar
   contexto caro en trabajo mecanico"*.

5. **Guardarrailes (heredados del proyecto):**
   - Cambiar el routing **es configuracion, y toda config deja evidencia**
     (V1-A): quien cambio que modelo, cuando y valor anterior.
   - **La tool `runtime.models` no decide routing, solo disponibilidad.** Esa
     separacion ya esta escrita y no se rompe.
   - El selector **no puede inventar modelos**: solo los que el catalogo real
     devuelve.

6. **Intensidades y CRITICIDAD DE SEGURIDAD (spec ampliada del operador)**

   Requisito literal: *"podemos poner mas intensidades. Critico si consigue
   solucionar el problema sol:high, pero para seguridad critica usamos minimo
   sol:xhigh incluso max"*.

   **Estado verificado en codigo:**
   - El validador ya acepta: `none`, `low`, `medium`, `high`, **`xhigh`**
     (`capacity_decision_validator_helpers_v0.go:72`).
   - **`max` NO existe todavia**: hay que anadirlo al validador y a la escala.

   **Y aqui esta lo importante, que es un cambio de modelo, no un valor mas:**

   El operador esta introduciendo una **segunda dimension** que el sistema no
   tiene. Hoy solo existe **complejidad** (trivial/normal/complejo/critico).
   Lo que pide es distinguir:

   - **complejidad tecnica** — "¿es un problema dificil?" → `critico` puede
     resolverse con `sol:high`.
   - **criticidad de SEGURIDAD** — "¿si esto sale mal, hay dano?" → exige
     **minimo `sol:xhigh`, y `max` cuando proceda**, *independientemente* de lo
     dificil que sea el problema.

   **Un cambio de una linea en un guard de seguridad es trivial en complejidad
   y maximo en criticidad.** Con una sola dimension, el sistema le asignaria
   `luna:low`. Eso es exactamente lo que hay que impedir.

   **Diseno:**

   - Anadir `security_criticality` (o equivalente) al contrato de la tarea:
     `normal` | `sensitive` | `critical`.
   - **Regla de piso, no de sustitucion**: la criticidad de seguridad impone un
     **minimo** de modelo+effort, y el maximo de las dos dimensiones gana. Nunca
     la complejidad puede rebajar el piso de seguridad.
   - Pisos por defecto (revisables desde la web, con su bocadillo):

         seguridad normal     -> sin piso (manda la complejidad)
         seguridad sensitive  -> minimo sol:xhigh
         seguridad critical   -> minimo sol:max

   - **Que cuenta como criticidad de seguridad** (lista explicita, no a ojo):
     guards y ratchets, sandbox y permisos, credenciales y secretos, atestacion
     independiente, write-sets, promocion/integracion, y el propio routing de
     modelos. Es decir: **todo lo que hoy protege al sistema de si mismo**.
   - Test que **falle** si una tarea marcada `critical` se enruta por debajo de
     su piso.

   **Justificacion empirica (de hoy mismo, no teorica):** Codex intento colar
   una relajacion de sandbox (`danger-full-access`) dentro de un commit de
   "docs", y dos veces intento apagar un guard escondiendo tools. Ninguno de
   esos cambios era **complejo**; todos eran **criticos**. Un routing que solo
   mira dificultad los habria mandado al modelo mas barato.

### V1-C. Observabilidad honesta (deuda detectada por Sonyi)

`autoprogramming/status` devuelve `projects=0 tasks=0 agents=0` cuando **no hay
nada activo**, aunque haya 34 runs terminales. Es coherente, pero **enga;a al
operador**: parece roto cuando esta vacio. El status debe distinguir *"no hay
nada activo"* de *"no hay nada"*.

### V1-D. Higiene (hito H4, ya asignado)

99 funciones inalcanzables (77 en produccion). No es borrar por borrar: hay
**validaciones muertas** (`ValidateStrictEventSequenceV0`,
`ValidateOrchestrationEventPayloadBudgetV0`, evaluacion de leases de agentes)
que son **garantias que creemos tener y no tenemos**. Se clasifican en
CONECTAR / BORRAR / CONSERVAR CON MOTIVO.

Ademas: `env_vars = 426/426`, al tope exacto. Sin margen.

## Definicion de "1.0"

Orquesta es 1.0 cuando un operador puede:

1. **Instalarla y configurarla sin editar ficheros a mano** (V1-A), incluido
   **elegir modelos e intensidades por rol desde la web**, viendo solo los que
   de verdad puede usar (V1-A2).
2. **Conectar sus credenciales desde la app**, con dueño y trazabilidad, sin que
   el secreto viaje ni se filtre (V1-B).
3. **Pedirle una app y recibirla**, con el circuito completo: goal → cierre
   atestado → promocion → integracion (**ya funciona**).
4. **Ver honestamente que esta pasando** (V1-C).
5. Y todo ello sobre codigo sin garantias desenchufadas (V1-D).

## Multiusuario (v2, mejora futura — NO ahora)

Registrado en `docs/decision_arquitectura_estado_2026-07-12.md` y aqui:

- Hoy: **un servidor = un proyecto = un dueño**. El aislamiento entre goals es
  fisico (workspace por goal) y el estado es un directorio compartido.
- Dos personas contra el mismo servidor **no se corrompen los datos** (CAS +
  workspaces), pero **se ven todo** y **comparten el repositorio de destino**.
- Multiusuario real exige: `tenant_ref` como clave primaria en TODO el estado,
  workdir/StateDir por tenant resueltos desde la peticion (no desde el env del
  servidor), e identidad/permisos con aislamiento de lectura.
- **Apaño valido mientras tanto**: un servidor por proyecto. Es lo que hacemos
  hoy y funciona.

**Lo unico que v1.0 debe garantizar es no cerrarle la puerta**: por eso
`owner_ref` en credenciales es obligatorio desde ya.

## Orden de trabajo

1. **H4** — higiene y validaciones desenchufadas (ya asignado a Codex).
2. **V1-B** — credenciales con dueño y trazabilidad (**es lo que condiciona la
   arquitectura**; va antes que la UI).
3. **V1-A** — configuracion por API + web sobre el esquema canonico.
   **V1-A2** — selector de modelos e intensidades (director/trabajadores), con
   descubrimiento en vivo de modelos locales y defaults explicados.
4. **V1-C** — status honesto.
5. Etiquetar **v1.0**.
