# 06. Seguridad, configuración, identidad y efectos

Responsabilidad: explica cómo impedir que configuración, credenciales,
identidad o efectos externos creen autoridades laterales o fugas.

Alcance: registro de configuración, almacén de credenciales, autenticación,
autorización, aislamiento entre proyectos y cadena de efectos.

No acredita: los controles descritos requieren pruebas contractuales y
composición real por adaptador.

## Al terminar este capítulo, el lector sabrá…

- separar configuración pública, referencias de credenciales y secretos;
- autenticar una identidad y autorizar cada operación en aplicación;
- impedir cruces de proyecto, rutas inseguras y fugas de información;
- gobernar cada efecto como intención, aprobación, intento y recibo.

El vocabulario compartido se define en el
[glosario técnico común](01_mision_limites_y_vocabulario.md#glosario-técnico-común).

## 1. Modelo de amenazas

El orquestador ejecuta código y comunica sistemas con permisos distintos. Debe
suponer:

- peticiones malformadas o repetidas;
- referencias de otro proyecto;
- credenciales caducadas, revocadas o sustituidas;
- rutas con enlaces, escapes o cambios entre comprobación y uso;
- procesos que sobreviven al padre;
- respuestas ambiguas tras un corte de red;
- agentes que inventan éxito;
- proveedores que aceptan una operación antes del tiempo límite;
- registros que contienen secretos;
- configuración que cambia entre preparación y ejecución;
- un trabajador con un arrendamiento antiguo que intenta escribir;
- evidencia reutilizada sobre otro candidato.

## 2. Registro único de configuración

La configuración parte de un esquema único que define:

- clave y referencia semántica;
- tipo y valor predeterminado;
- validadores locales y cruzados;
- sensibilidad;
- alcance;
- precedencia;
- alias temporales;
- necesidad de reinicio;
- documentación y proyección de interfaz.

Entradas:

1. fichero humano no sensible;
2. referencias a credenciales.

Salida:

3. configuración efectiva generada, redactada e inmutable.

Solo el cargador gobernado puede leer variables de entorno y solo para aplicar
las claves, alias y precedencias declarados en el registro de configuración.
Fuera del cargador y del registro quedan prohibidos `os.Getenv`,
`os.LookupEnv`, `os.Environ` y cualquier equivalente, así como descubrir claves
por prefijo. Ningún módulo puede inventar un valor predeterminado ni recibir el
registro global: recibe una estructura tipada y acotada.

## 3. Cambiar configuración

Secuencia obligatoria:

1. buscar semántica existente;
2. decidir reutilizar, reemplazar o crear;
3. modificar primero el registro;
4. regenerar accesores, esquema, documentación y superficies;
5. añadir validadores y negativos;
6. comprobar redacción de valores sensibles;
7. fijar si el cambio requiere reinicio;
8. retirar alias cuando ya no tengan consumidores.

Un alias es una migración temporal, no otra clave permanente.

## 4. Credenciales

El `CredentialStore` conserva:

- referencia opaca;
- propietario;
- alcance;
- versión;
- estado de rotación y revocación;
- adaptador sustituible.

La aplicación nunca persiste el secreto en:

- `Goal`;
- `WorkItem`;
- plantilla de instrucciones;
- evento;
- registro;
- artefacto;
- recibo;
- configuración efectiva;
- nombre de proceso o argumento visible.

El hijo recibe una lista exacta de variables, descriptores o montajes y solo el
material mínimo durante el tiempo necesario.

## 5. Identidad

La autenticación traduce una credencial a un principal. La autorización decide
si ese principal puede ejecutar un caso de uso sobre un proyecto y alcance
concretos.

La aplicación valida autorización antes de:

- comando o consulta;
- lectura de artefacto;
- cambio de configuración;
- creación o control de agentes;
- decisión del Director;
- aprobación de efecto;
- integración Git;
- publicación o despliegue;
- retirada de datos.

Los adaptadores OIDC, Active Directory, token local o LDAP no escriben el
ciclo de vida ni mantienen políticas distintas.

## 6. Aislamiento de proyecto

Debe probarse tanto la ruta permitida como la denegada:

- un actor de A no consulta B;
- una ejecución de A no recibe correo de B;
- un artefacto de A no se resuelve desde B;
- una credencial de A no se proyecta en B;
- un espacio de trabajo de A no se reutiliza en B;
- una reserva de A no se liquida desde B;
- una evidencia de A no acredita B.

La ausencia de datos en una respuesta no basta: debe existir una denegación
estable o una consulta confinada demostrable.

## 7. Cadena de efectos

Todo efecto externo separa:

```text
intención -> aprobación -> intento -> recibo
```

### 7.1 Intención

Declara sujeto, acción, alcance, política, presupuesto, objetivo inmutable y
clave causal.

### 7.2 Aprobación

Declara principal, decisión, razón, alcance, revisión y caducidad. No se infiere
de la confirmación de otro efecto.

### 7.3 Intento

Fija arrendamiento, cercado, identidad, clave idempotente, instante y parámetros
redactados antes de cruzar la frontera.

### 7.4 Recibo

Conserva resultado observado, referencia externa, uso, tiempo y relación exacta
con intención, aprobación e intento.

Un código HTTP correcto, un mensaje del agente o un acuse de admisión no
equivalen a recibo terminal.

## 8. Fallo ambiguo

Si la frontera puede haber aplicado el efecto:

- no se libera la reserva por intuición;
- no se crea un intento nuevo con otra clave;
- se observa o reconcilia el mismo efecto;
- se mantiene la escritura cercada;
- se conserva `unknown_applied` hasta obtener evidencia;
- el reinicio retoma la misma frontera causal.

Solo una prueba estructural de “definitivamente no aplicado” permite reintentar
sin reconciliación externa.

## 9. Sistema de ficheros

Para contenido sensible o autoritativo:

- abrir desde un directorio confiable;
- no seguir enlaces;
- comprobar tipo por descriptor;
- verificar propietario, modo y número de enlaces cuando corresponda;
- acotar tamaño antes y durante lectura;
- usar temporal privado;
- sincronizar fichero;
- renombrar atómicamente;
- sincronizar directorio;
- revalidar identidad si existe carrera posible;
- rechazar representación ambigua.

Una comprobación por ruta antes de abrir no protege contra sustitución.

## 10. Red

Todo conector declara:

- destinos permitidos;
- resolución y revalidación tras redirección;
- protocolos y certificados;
- tiempo límite y tamaño;
- intermediario de red y procedencia;
- credencial y alcance;
- política de repetición;
- coste y presupuesto;
- recibo y retirada.

La respuesta se trata como dato no confiable. El conector no decide política.

## 11. Pruebas mínimas

Cada frontera sensible incluye:

- acceso correcto;
- principal ausente;
- proyecto cruzado;
- referencia manipulada;
- secreto redactado;
- rotación y revocación;
- repetición idéntica;
- clave idempotente en conflicto;
- tiempo límite antes y después de aplicar;
- cercado obsoleto;
- enlace simbólico, enlace físico, permisos y salto de directorios;
- reinicio y recuperación;
- registro y artefacto sin material sensible.

## 12. Errores que deben evitarse

- variables de entorno leídas en adaptadores;
- configuración completa inyectada en todo el sistema;
- secretos copiados a espacios persistentes sin contrato;
- autorización solo en la interfaz HTTP;
- identidad inferida desde una ruta o plantilla de instrucciones;
- reintento ciego después del tiempo límite;
- recibo sintetizado por el núcleo;
- un filtro textual usado como seguridad;
- borrar datos porque una ejecución terminó;
- permitir que un complemento comparta la base interna.
