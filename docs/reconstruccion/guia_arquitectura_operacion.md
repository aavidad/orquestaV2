# Guía de arquitectura y operación del corte mínimo

Fecha: 2026-07-14

Esta guía describe el producto reconstruido. No describe ni reactiva el
runtime legacy.

## Arquitectura efectiva

```text
cliente MCP oficial
        |
Bearer local + MCP Streamable HTTP (loopback)
        |
internal/interfaces/mcp
        |
internal/application.Orchestrator  <- único escritor de lifecycle
        |
internal/goal                      <- Goal y WorkItem son la autoridad
        |
        +-- application.StateRepository ---- SQLite
        +-- application.AgentLauncher/Observer --- Codex
        +-- application.ArtifactStore ------------ filesystem CAS
        +-- application.Clock/IDGenerator -------- sistema local
```

`cmd/orquesta` solo interpreta `serve`, `version` y señales. `internal/bootstrap`
es la raíz de composición: carga una configuración tipada, abre adaptadores,
inyecta puertos, publica MCP y ejecuta el scheduler. Dominio y aplicación no
importan SQLite, filesystem, MCP, Codex ni módulos legacy.

El flujo acreditado es:

```text
IntentManifest -> AppSpec confirmado -> Goal + WorkItem -> Execution
                                      -> Artifact + Attestation
                                      -> WorkItem terminal -> Goal terminal
```

`IntentManifest` conserva la entrada exacta. `AppSpec` contiene la interpretación
confirmada, su hash y su generación causal. Una modificación no reescribe el
Goal anterior: crea un nuevo Intent, un AppSpec N+1 y un Goal sucesor pendiente;
el padre terminal y sus evidencias permanecen inmutables.

Las transiciones se deciden en dominio/aplicación y se persisten mediante
operaciones atómicas del repositorio. El adaptador de agente solo lanza y
observa. Repetir una ejecución con la misma causalidad es idempotente; cambiar
su payload bajo la misma identidad causal es conflicto. Eventos, acciones y
registros de ejecución son auditoría o proyecciones, no otra autoridad.

## Frontera operativa vigente

- un operador local: `actor:local-owner`;
- un proyecto inicial: `project:default`;
- HTTP limitado a una dirección de loopback y protegido por un token Bearer
  local persistente;
- SQLite local, blobs inmutables content-addressed en filesystem y Codex como
  primer proveedor;
- release acreditada en Linux; la limpieza de descendientes usa process groups
  Unix. Windows requiere un adaptador Job Object antes de prometer paridad;
- identidad y proyecto siguen viajando como refs opacas, aunque hoy tengan un
  único valor;
- errores públicos y de puertos usan códigos estables; no exponen stderr del
  proveedor;
- MCP publica `orquesta.goals.create`, `orquesta.goals.amend`,
  `orquesta.goals.get`, `orquesta.goals.list`, `orquesta.artifacts.read` y
  `orquesta.system.status`. Create/amend exigen `confirm:true`; actor, proyecto,
  refs generadas y tiempos proceden de dependencias de confianza del servidor,
  no del cliente.

No se debe abrir el listener a red pública: el token local evita que usuarios y
procesos sin acceso al fichero invoquen MCP, pero no aísla procesos que corren
con el mismo usuario del sistema operativo. Este corte tampoco incorpora
autenticación remota, autorización multiusuario ni TLS.

## Tres superficies de configuración y un almacén secreto

La precedencia de resolución es `default < TOML < alias de entorno declarado`.
No se aceptan claves desconocidas, duplicadas o valores con tipo incorrecto.
Registro, TOML y proyección efectiva son configuración; el token local es
material secreto separado y no una cuarta fuente de variables.

### 1. Registro canónico JSON

[`config/registry.json`](../../config/registry.json) es la única definición de
claves. Para cada clave registra tipo, default, sensibilidad, alcance, alias de
entorno, necesidad de reinicio y, cuando corresponde, límites o valores
permitidos. El código tipado generado debe permanecer sincronizado con su
revisión.

No es configuración operativa. Añadir o cambiar una clave es un cambio de
producto revisado: primero se modifica el registro, después se regenera y se
prueban accesores y ejemplo. Ningún adaptador lee variables de entorno ad hoc.

### 2. TOML no secreto del operador

[`config/orquesta.toml.example`](../../config/orquesta.toml.example) contiene
la superficie humana completa. El operador copia el ejemplo y cambia rutas,
límites y selección de adaptadores sin introducir secretos. Las rutas relativas
se resuelven desde el directorio de trabajo del proceso; en una instalación
estable conviene usar rutas absolutas, privadas y distintas para estado,
artefactos y trabajo de Codex.

`runtime.codex.credential_ref` es una referencia, nunca el secreto. El corte
actual aún no inyecta un resolvedor de credenciales y rechaza una referencia no
vacía con `bootstrap.credential_resolver_unavailable`. La autenticación local de
Codex llega únicamente por el entorno hijo expresamente permitido, por ejemplo
`CODEX_HOME`; no se copia el entorno completo del servidor.

`identity.local_token_path` también es solo una ruta no secreta. El secreto
real vive en ese fichero separado: Orquesta lo crea una sola vez con 256 bits
aleatorios, directorio `0700` y fichero `0600`, y lo reutiliza tras reinicios.
No se copia al TOML ni a `effective_config`, no se incluye en argumentos o
entorno hijo de Codex y nunca debe aparecer en logs. Esto no impide que un
proceso del mismo usuario del sistema lea ficheros accesibles a ese usuario:
el token local no es un sandbox de secretos. Un destino existente inseguro,
inválido o enlazado por symlink hace fallar el arranque en vez de ser
sustituido.

`runtime.codex.max_concurrent_executions` limita procesos activos (70 por
defecto) sin perder Goals: cuando no hay slot, la acción permanece durable en
cola y no consume el presupuesto de observación. La demora de drenaje de pipes
tras salir Codex también está centralizada como
`runtime.codex.process_pipe_drain_delay`; no hay timeouts ad hoc en el adapter.

### 3. `effective_config` redactado

En cada arranque se escribe la proyección resuelta indicada por
`config.effective_path`. Incluye revisión del registro, hash del snapshot,
origen efectivo de cada valor y metadata operativa. Todo valor sensible aparece
como `[REDACTED]`. La escritura es atómica y el fichero queda con modo de
propietario `0600`.

Es evidencia diagnóstica de salida, no una entrada reutilizable. El loader no
acepta ese JSON como configuración. `config.effective_max_existing_bytes`
acota tanto la proyección nueva como la lectura del documento anterior, para
que una configuración válida no pueda escribir un fichero que el siguiente
arranque rechace.

### Por qué no YAML

Hay una fuente de verdad para máquinas (el registro JSON) y una superficie
humana sencilla (TOML). TOML representa directamente las secciones y tipos del
registro. Añadir YAML crearía una segunda sintaxis humana, más reglas de
coerción, duplicados, aliases y otra dependencia de parser sin aportar una
capacidad del corte. No se afirma que YAML sea inseguro: se evita para reducir
ambigüedad y deriva. El guard arquitectónico impide dependencias y ficheros de
configuración YAML en el producto nuevo.

## Arranque

1. Copiar el ejemplo a una ruta privada y ajustar al menos las rutas de estado,
   artefactos, trabajo y token local. Los árboles operativos no pueden
   solaparse; el token debe vivir en un directorio privado propio.
2. Mantener el listener en loopback y asegurar que
   `runtime.codex.timeout < scheduler.execution_timeout`.
3. Comprobar que el comando `codex` configurado existe y que el entorno
   allowlisted contiene solo lo necesario.
4. Arrancar desde la raíz del repositorio:

```bash
go run ./cmd/orquesta serve --config /ruta/privada/orquesta.toml
```

Sin `--config`, se aplican defaults y aliases de entorno declarados. El endpoint
por defecto es `http://127.0.0.1:8080/mcp`. Debe consumirse con un cliente MCP
Streamable HTTP que añada `Authorization: Bearer <contenido-del-token>`, no como
una API REST informal. Una petición sin token o con uno incorrecto recibe
`401` antes de llegar a MCP. `orquesta.system.status` informa si el repositorio
está listo y cuántos Goals y acciones quedan.

SQLite crea su directorio con modo privado si no existe; si ya existe y permite
acceso a grupo u otros, el arranque falla. El almacén de artefactos y el work
root de Codex aplican la misma política de aislamiento. No se deben reutilizar
rutas del producto legacy.

## Reinicio y parada

Un reinicio usa el mismo TOML y los mismos roots. El repositorio restaura el
aggregate y evidencia con revisiones exactas. Trabajo ya terminal no se vuelve
a lanzar; una observación terminal del adaptador Codex también es recuperable
por `ExecutionRef`.

`SIGINT` y `SIGTERM` inician parada cooperativa. El bootstrap cancela el
scheduler, deja de aceptar HTTP, detiene procesos propios del agente, espera su
recolección y cierra listener, artifact store y SQLite dentro de
`server.shutdown_timeout`. En Linux el adaptador Codex mata también su grupo de
procesos y falla la ejecución si no puede acreditar esa limpieza. No se debe
terminar la operación borrando roots o matando procesos sin dejar que actúe
esta secuencia.

## Diagnóstico acotado

- `bootstrap.listen_must_be_loopback`: se intentó una dirección no local;
- `bootstrap.runtime_paths_overlap`: dos roots operativos se solapan;
- `bootstrap.execution_timeout_invalid`: el timeout de Codex no es menor que
  el de ejecución;
- `bootstrap.credential_resolver_unavailable`: se configuró una referencia de
  credencial antes de instalar su adaptador;
- errores `localtoken.*`: revisar ruta, propietario y modos del directorio y
  fichero; no regenerar ni imprimir el token como reparación automática;
- fallos `*.permissions` o `*.directory_not_private`: corregir ownership/modo,
  no relajar el adaptador;
- errores de config: contrastar el TOML con registro y `effective_config`; no
  añadir aliases nuevos fuera del registro.

Los detalles del proveedor se conservan solo para diagnóstico local acotado;
las interfaces retornan códigos tipados y no el stderr bruto.
