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

1. **Instalarla y configurarla sin editar ficheros a mano** (V1-A).
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
4. **V1-C** — status honesto.
5. Etiquetar **v1.0**.
