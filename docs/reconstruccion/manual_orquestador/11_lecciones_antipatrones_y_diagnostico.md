# 11. Lecciones, antipatrones y diagnóstico

> **Responsabilidad:** convertir fallos históricos repetidos en reglas de
> diagnóstico y pruebas de no regresión.
>
> **Alcance:** autoridad, agentes, capacidad, pruebas, configuración,
> seguridad, procesos, contexto, tamaño, legado e interfaces.
>
> **No acredita:** este inventario de lecciones no demuestra que todas estén
> implementadas ni sustituye el registro estructurado de incidencias.

## Al terminar este capítulo, el lector sabrá…

- localizar la primera divergencia causal en lugar de perseguir el último error;
- reconocer antipatrones de autoridad, agentes, capacidad, pruebas y operación;
- convertir un fallo histórico en invariante y prueba de no regresión;
- distinguir una solución causal de un parche narrativo o un falso verde;
- diagnosticar sin destruir procesos, entornos ni evidencia.

## Índice del capítulo

- [Vocabulario](#0-vocabulario), [finalidad](#1-para-qué-sirve-este-capítulo), [estado actual](#2-estado-actual-honesto) y [método](#3-método-universal-de-diagnóstico).
- [Autoridad fragmentada](#4-autoridad-fragmentada), [parada narrativa](#5-parada-narrativa), [agentes estáticos](#6-agentes-y-perfiles-estáticos) y [cuota inventada](#7-cuota-inventada).
- [Reintento ciego](#8-reintento-ciego), [falsos verdes](#9-pruebas-contaminadas-y-falsos-verdes), [configuración](#10-configuración-dispersa) y [secretos](#11-secretos-mezclados-con-configuración-o-contexto).
- [Conjunto de escritura](#12-conjunto-de-escritura-parcial), [procesos huérfanos](#13-procesos-huérfanos), [contexto perdido](#14-contexto-perdido) y [aplicación mastodóntica](#15-aplicación-mastodóntica).
- [Fragmentación vacía](#16-fragmentación-vacía), [legado como ejecución](#17-legado-como-entorno-de-ejecución), [interfaz como autoridad](#18-interfaz-como-autoridad) y [limpieza agresiva](#19-limpieza-agresiva-que-borra-evidencia).
- [Señales](#20-señales-derivadas), [plantilla](#21-plantilla-de-prueba-de-lección), [consultas](#22-lecciones-consultadas) y [fuentes](#23-fuentes-vigentes).

## 0. Vocabulario

- lección: relación demostrada entre síntoma, causa arquitectónica, invariante y
  prueba que impide la regresión.
- antipatrón: solución aparente que se repite y empeora la corrección,
  operación o mantenibilidad.
- autoridad: componente autorizado para decidir y persistir un hecho.
- proyección: vista derivada y reconstruible; informa, pero no manda.
- parada narrativa: texto, marcador o estado secundario que afirma «parado»
  sin demostrar la detención causal del recurso exacto.
- falso verde: comprobación satisfactoria que no prueba la conducta prometida.
- cerca (`fence`; la técnica aparece como `fencing`): ordinal que invalida a un
  propietario o escritor anterior.
- ciclo de vida (`lifecycle`): secuencia autoritativa de estados de una entidad.
- arrendamiento (`lease`): concesión temporal y exclusiva sobre un recurso.
- buzón causal (`mailbox`): intercambio durable y ordenado ligado a identidades
  y ejecución exactas.
- recibo (`receipt`): registro estructurado de un intento, resultado o efecto.
- huella criptográfica (`digest` o `hash`): resumen verificable de un sujeto
  inmutable.
- prueba de lección: prueba que reproduce la causa del fallo y falla si se
  reintroduce el mecanismo defectuoso.

## 1. Para qué sirve este capítulo

Una incidencia no aporta una lección por el mero hecho de estar documentada.
Debe poder responder:

```text
síntoma observado:
causa demostrada:
invariante vulnerado:
diagnóstico reproducible:
solución que elimina la causa:
prueba que falla con el enfoque antiguo:
capacidad y evidencia:
```

Las lecciones se aplican a cualquier orquestador, no solo a los nombres
históricos de Orquesta. El código antiguo se consulta en solo lectura para
extraer conducta y fallos; nunca se convierte en dependencia.

## 2. Estado actual honesto

| Área | Estado | Hecho comprobado |
|---|---|---|
| Registro canónico de capacidades y trazabilidad | ACREDITADO | V03 conserva tareas, incidencias, disposiciones y pruebas sin inferir cierre histórico. |
| Configuración central y prohibición de variables dispersas | ACREDITADO | `OPS-06` está acreditada en V07. |
| Presupuestos, efectos y falta temporal de cuota | ACREDITADO | V15 evita convertir falta temporal definitivamente no aplicada en fallo terminal. |
| Pruebas independientes | ACREDITADO | V17–V19 separan atestación, dos revisiones y Consejo. |
| Control y parada selectiva actuales | PARCIAL | V14 y V22 están acreditadas para su alcance estrecho; la parada de la capacidad elástica completa sigue pendiente. |
| Progreso honesto operativo | PENDIENTE | `ORC-23` pertenece a V32. |
| Vigilante operativo completo | PENDIENTE | `ORC-25`/`OPS-18` pertenecen a V32. |
| Agentes elásticos, mensajes vivos y capacidad externa | PENDIENTE | V38 es canónica, planificada y propietaria solo de `ORC-28`. `ORC-15` sigue en V27 y `OPS-16`/`OPS-17` en V32 como requisitos no propios que V38 no reasigna ni acredita. |
| Pruebas de mutación selectivas globales | PENDIENTE | `EVD-10` pertenece al cierre V34. |
| Inventario físico total del legado | PENDIENTE | El universo lógico está expandido, pero la compuerta física continúa bloqueada («NO-GO», es decir, no apta para promoción). |

Que una protección esté acreditada en una vertical no demuestra que cubra una
capacidad posterior más amplia. El alcance siempre se lee en el contrato.

La revisión actual contiene 257 capacidades, 38 verticales y 38 contratos.
V1–V22 están acreditadas y V23–V38 planificadas. En V38, A prueba el núcleo
elástico neutral sin KVM ni Firecracker; B exige activación explícita de
Firecracker, sin sustitución automática y con una microVM por agente; C ejecuta
1/5/10/16/20 físicamente sobre el mismo candidato. Solo A+B+C acreditan V38.
V23 no depende de Firecracker y V38/Firecracker no dependen de cerrar V23.

## 3. Método universal de diagnóstico

El diagnóstico empieza por la primera divergencia causal, no por el último
mensaje de error.

```text
diagnosticar(incidencia):
    identificar_sujeto_revision_y_configuracion()
    identificar_objetivo_trabajo_ejecucion_y_generacion()
    leer_estado_autoritativo_sin_mutar()
    derivar_proyeccion_esperada()
    comparar_con_procesos_agentes_y_efectos_reales()
    localizar_primera_frontera_donde_esperado_y_real_divergen()
    reproducir_con_entrada_minima()
    formular_causa_e_invariante()
    diseñar_prueba_que_falle_con_la_causa()
    solo_despues_proponer_solucion()
```

Preguntas obligatorias:

- ¿quién podía escribir el hecho?;
- ¿qué revisión esperaba?;
- ¿qué identidad y generación tenía?;
- ¿qué efecto pudo aplicarse aunque faltara recibo?;
- ¿qué observación está caducada?;
- ¿qué dato es autoridad y cuál proyección?;
- ¿qué prueba distinguiría causa de coincidencia?;

## 4. Autoridad fragmentada

### Síntoma

Una superficie muestra un objetivo activo, otra lo considera inactivo y una
tercera permite apagar o cerrar. Tras reiniciar, la contradicción cambia.

### Causa

Varias colas, tablas, marcadores, controladores o interfaces mantienen estados
propios y toman decisiones de ciclo de vida. Ninguna puede reconstruir la
decisión completa.

### Diagnóstico

1. enumerar todos los escritores del estado contradictorio;
2. localizar transiciones que no pasan por aplicación;
3. comparar identidad, generación y revisión;
4. comprobar si evento o proyección se usa como autoridad;
5. reconstruir el resultado únicamente desde `Goal`, trabajos y recibos.

### Solución

Un solo `Goal`, un escritor determinista en aplicación, un planificador y una
fuente activa. Eventos, interfaces y telemetría son proyecciones. Estado,
evento y bandeja transaccional cambian atómicamente.

### Prueba de lección

Crear estados secundarios contradictorios y demostrar que:

- no pueden cerrar ni reabrir;
- la consulta canónica sigue la fuente autoritativa;
- el reinicio produce la misma proyección;
- una mutación que acepte la proyección como escritor hace fallar la prueba.

## 5. Parada narrativa

### Síntoma

El sistema dice «detenido», pero el proceso sigue vivo; o mata un proceso y el
trabajo continúa marcado como en ejecución, se relanza o pierde su punto de
control.

### Causa

Se confunden orden, admisión, señal, salida del proceso y cierre del trabajo.
Un texto, marca, PID suelto o código HTTP se toma como prueba completa.

### Diagnóstico

- identificar orden de parada, objetivo exacto, modo y autorización;
- comprobar identidad externa, grupo de procesos y cerca;
- localizar confirmación cooperativa y posible escalado;
- comprobar recibo del controlador;
- cotejar trabajo, ejecución y proceso después de reiniciar.

### Solución

Separar:

```text
solicitud -> admisión -> intento cooperativo -> verificación
          -> escalado autorizado -> verificación -> conservación -> recibo
```

El cierre de la ejecución se decide después de verificar el efecto exacto.

### Prueba de lección

Un agente resistente a la primera señal, junto a dos agentes hermanos. Detener
solo el elegido, conservar los demás y repetir tras reinicio. Una mera cadena
que diga «detenido» nunca satisface la prueba.

## 6. Agentes y perfiles estáticos

### Síntoma

Solo trabajan tres, seis o un número configurado de agentes aunque haya más
trabajos independientes y recursos. Al agotarse un perfil, todo el objetivo se
atasca.

### Causa

La lista de perfiles creada al arrancar se confunde con capacidad del
orquestador. No existe reconciliación entre demanda completa y agentes reales,
ni alta y baja en caliente.

### Diagnóstico

1. contar trabajos listos sin aplicar límite físico;
2. contar ejecuciones durables en espera;
3. revisar perfiles configurados y plazas observadas;
4. comprobar si el planificador recorta demanda;
5. comprobar si se puede añadir capacidad sin reiniciar.

### Solución

Persistir toda la demanda, observar capacidad, reservarla atómicamente y
reconciliar estado deseado y real. Los límites pertenecen a proveedor,
presupuesto, anfitrión o sistema operativo y se exponen.

### Prueba de lección

Ejecutar por separado las dos series canónicas aún no acreditadas:

- demanda lógica exacta 1/16/70/500: con demanda quinientos y capacidad física
  veinte deben existir quinientas ejecuciones durables, veinte reservas como
  máximo y progreso por oleadas sin pérdida;
- pasos físicos exactos 1/5/10/16/20: cada escalón demuestra simultaneidad
  medida, carreras y recuperación sin convertir 70 o 500 en cifras físicas.

Los tamaños 1 y 16 aparecen en ambas series, pero prueban sujetos distintos y
no autorizan una lista única de siete cifras.

## 7. Cuota inventada

### Síntoma

El sistema muestra cuota disponible, agotada o cero sin fuente verificable.
Rota agentes por una frase de consola o presupone capacidad ilimitada cuando
el proveedor no informa.

### Causa

Se mezclan presupuesto interno, uso observado, límite contractual y
disponibilidad. El estado desconocido se interpreta como si fuera un valor
medido.

### Diagnóstico

- buscar fuente, instante y caducidad de la observación;
- separar plazas, fichas, dinero, procesos y recursos físicos;
- revisar reservas vivas;
- comprobar si la decisión se derivó de texto libre;
- comparar proveedor observado con perfil ejecutado.

### Solución

Una observación estructurada con calidad disponible, agotada, desconocida,
caducada o inaccesible. El proveedor observa y la aplicación decide.
Desconocida no equivale a cero ni a ilimitada.

### Prueba de lección

Inyectar observación desconocida, caducada y contradictoria. Ninguna puede
crear capacidad falsa, consumir un intento ni cerrar el trabajo. Una frase
«cuota superada» en la salida normal no cambia el estado.

## 8. Reintento ciego

### Síntoma

Tras un tiempo agotado o reinicio aparecen dos agentes, dos efectos o ciclos de
corrección que repiten el mismo trabajo y coste sin información nueva.

### Causa

No se distingue:

- definitivamente no aplicado;
- aplicado;
- aplicación desconocida.

La clave de idempotencia cambia o no existe una consulta de reconciliación.

### Diagnóstico

1. comparar intenciones, intentos y recibos;
2. agrupar por clave de idempotencia y ejecución;
3. buscar identidad externa recuperable;
4. localizar caídas entre efecto y recibo;
5. comprobar si la corrección aporta evidencia nueva.

### Solución

Reintentar solo lo definitivamente no aplicado. Consultar y poner en
cuarentena lo desconocido. Un relevo o corrección necesita un punto de control
y contexto nuevo; repetir la misma orden no es replanificar.

### Prueba de lección

Aplicar el efecto y perder el recibo. Tras varios reinicios debe seguir
existiendo un único efecto externo. Una corrección sin nueva evidencia queda
bloqueada en lugar de lanzar otro agente.

## 9. Pruebas contaminadas y falsos verdes

### Síntoma

La prueba pasa, pero no ejecutó casos, usó archivos del anfitrión, heredó
secretos, consultó la misma implementación o acreditó otro árbol.

### Causa

El ejecutor no está aislado, el contrato acepta salida textual, omisiones o
código cero sin trabajo real, o el sujeto no está sellado.

### Diagnóstico

- verificar orden exacta, código de salida y casos ejecutados;
- comprobar si aparece el mensaje de herramienta `no tests to run`, es decir,
  que no hay pruebas que ejecutar, o una omisión;
- inspeccionar montajes, entorno y red;
- comparar árbol, diferencia, pruebas y configuración;
- verificar huellas de salida y recibo;
- confirmar que el atestador es independiente.

### Solución

Pruebas requeridas estructuradas, aislamiento, entorno permitido mínimo,
sujeto inmutable, salida sellada y validación semántica del ejecutor. Un acuse
o texto `PASS` no basta.

### Prueba de lección

Casos negativos:

- orden válida que ejecuta cero pruebas;
- binario que imprime `PASS` y sale con error;
- salida de otro árbol;
- secreto heredado;
- enlace a archivo anfitrión;
- etiqueta que omite la prueba física.

Todos deben fallar cerrados sin fabricar evidencia.

## 10. Configuración dispersa

### Síntoma

Dos instalaciones se comportan distinto con el mismo TOML; una variable no
documentada cruza a un hijo; cambiar un valor no aparece en la configuración
efectiva.

### Causa

Lecturas directas de entorno, valores por defecto, nombres alternativos y
filtros por prefijo fuera del registro.

### Diagnóstico

```bash
rg -n 'os\\.(Getenv|LookupEnv|Environ)' --glob '*.go' \
  --glob '!vendor/**' --glob '!third_party/**'
```

Después se comparan cada clave encontrada, el registro, el cargador, los
getters generados y la configuración efectiva.

### Solución

Una definición en `config/registry.json`, TOML como entrada humana,
estructuras tipificadas por adaptador y proyección efectiva redactada. Un
nombre alternativo tiene fecha y prueba de retirada.

### Prueba de lección

Una variable con prefijo permitido pero clave no registrada no debe cruzar al
proceso hijo ni cambiar conducta. El validador arquitectónico debe detectar
lecturas en todos los argumentos, no solo el primero.

## 11. Secretos mezclados con configuración o contexto

### Síntoma

Una ficha aparece en un registro, instrucción, artefacto, proceso, configuración
efectiva o traspaso. Una credencial de un agente funciona para otro proyecto.

### Causa

Se pasan valores en claro y variables globales en vez de referencias,
propietario, alcance, versión y concesión mínima.

### Diagnóstico

- rastrear desde `credential_ref` hasta el consumidor;
- revisar alcance, dueño, versión, caducidad y revocación;
- inspeccionar proyecciones y artefactos con detectores de fuga;
- comprobar entorno exacto del hijo sin mostrar valores;
- probar acceso cruzado negativo.

### Solución

`CredentialStore`, referencias opacas, concesiones mínimas y efímeras,
listas permitidas exactas y saneamiento antes de persistir. Los errores no
repiten el secreto.

### Prueba de lección

Inyectar marcadores secretos en cada entrada posible. Ninguno debe aparecer en
estado, salida, trazas o artefactos. Revocar una concesión y comprobar que el
agente antiguo no puede reutilizarla.

## 12. Conjunto de escritura parcial

### Síntoma

El agente entrega un cambio aparentemente correcto, pero omite ficheros
necesarios, modifica rutas no declaradas o una revisión mira solo parte de la
diferencia.

### Causa

El conjunto de escritura se trata como recomendación o se calcula después de
trabajar. La evidencia no vincula la lista exacta, incluidos modos y
eliminaciones.

### Diagnóstico

- comparar conjunto declarado con `git diff --name-status`;
- buscar cambios no versionados;
- comprobar eliminaciones y modos;
- verificar archivos generados y migraciones;
- cotejar lista del candidato con el recibo.

### Solución

Conjunto cerrado antes de lanzar, espacio de trabajo aislado, escritura
rechazada fuera de alcance y revisión sobre el cambio completo. Una
dependencia nueva crea otra tarea o ampliación explícita.

### Prueba de lección

Modificar un fichero permitido y otro prohibido. El resultado debe quedar
pendiente y el destino sin integrar. Omitir una eliminación o cambiar un modo
debe alterar la huella y anular la revisión.

## 13. Procesos huérfanos

### Síntoma

Después de cerrar quedan agentes, ayudantes, zócalos, grupos de control,
montajes o bloqueos. Un reinicio adopta un PID reutilizado.

### Causa

No existe registro durable de propiedad o la limpieza usa nombres y patrones.
El padre termina sin recoger descendientes ni conservar evidencia.

### Diagnóstico

- inventariar PID, padre, usuario, inicio e identidad externa;
- comprobar grupo de control y bloqueo propietario;
- cotejar registro durable y procesos;
- verificar reutilización de PID;
- buscar recursos propios sin vínculo causal.

### Solución

Identidad durable, grupo por ejecución, bloqueo exclusivo, adopción con varios
atributos y parada exacta. La limpieza solo actúa sobre recursos propios
verificados y conserva los desconocidos.

### Prueba de lección

Crear procesos hermanos y reutilizar un PID simulado. Reiniciar y detener uno
no debe afectar al resto ni adoptar al impostor. El cierre exige cero recursos
propios residuales.

## 14. Contexto perdido

### Síntoma

Un agente sucesor repite análisis, contradice decisiones o no sabe qué cambios
y pruebas dejó el anterior. El operador necesita leer una transcripción
completa.

### Causa

El contexto vive en una sesión efímera o texto sin referencias, y no hay
secuencia, confirmación ni punto de control durable.

### Diagnóstico

- localizar último mensaje admitido, consumido y confirmado;
- comprobar remitente y destinatario exactos;
- buscar referencias de artefactos y cambio;
- cotejar generación y ejecución;
- distinguir falta de entrega de falta de aplicación.

### Solución

Traspaso compacto con hechos, decisiones, pruebas, bloqueos, artefactos y
referencias causales. Buzón secuenciado e idempotente. Los contenidos grandes
viajan por referencia.

### Prueba de lección

Caer entre admisión, entrega, consumo y confirmación. El sucesor exacto recibe
una vez; un sucesor distinto no suplanta al destinatario. Un mensaje grande no
entra completo en el contexto.

## 15. Aplicación mastodóntica

### Síntoma

Un fichero o gestor decide ciclo de vida, proveedores, configuración,
persistencia, interfaz y limpieza. Cualquier corrección afecta a todo.

### Causa

Se confunde monolito modular con clase o paquete único. No hay presupuestos de
responsabilidad, complejidad, escritores y bucles.

### Diagnóstico

- enumerar motivos de cambio de la pieza;
- contar dependencias, escritores, bucles y almacenes;
- trazar el recorrido para diagnosticar un fallo;
- comprobar si sus pruebas requieren preparar todo el sistema;
- identificar decisiones que pertenecen a otras capas.

### Solución

Dividir por responsabilidad cohesionada y dirección hexagonal. Mantener un
binario y ciclo de vida únicos no obliga a mezclar módulos.

### Prueba de lección

Una incidencia de adaptador debe corregirse sin modificar dominio ni interfaz.
Los controles arquitectónicos detectan dependencias hacia dentro incorrectas y
el presupuesto exige justificar piezas sobredimensionadas.

## 16. Fragmentación vacía

### Síntoma

Para entender una operación hay que saltar por decenas de archivos de una
constante, estructura o envoltorio sin decisión. Los nombres aparentan
arquitectura, pero no hay fronteras.

### Causa

Se equipara «pequeño» con «un fichero por símbolo». Cada función obtiene una
interfaz o servicio aunque tenga un consumidor y ninguna sustitución real.

### Diagnóstico

- buscar paquetes sin decisión propia;
- contar interfaces de una implementación;
- seguir envoltorios que solo reenvían argumentos;
- comprobar consumidores reales;
- localizar duplicación que la abstracción afirma retirar.

### Solución

Agrupar por cohesión. Crear puerto solo para una frontera externa o dos
consumidores reales; retirar envoltorios sin política. Código generado queda
separado, pero no crea reglas.

### Prueba de lección

El control de arquitectura y presupuesto rechaza un paquete nuevo sin frontera,
consumidor o duplicación nombrada. Una tarea pequeña sigue pudiendo localizarse
y probarse desde un único módulo coherente.

## 17. Legado como entorno de ejecución

### Síntoma

Una función nueva importa un módulo antiguo, consulta su base, arranca su
servidor o cae hacia él cuando falla el producto nuevo.

### Causa

La migración busca velocidad mediante puente, escritura doble o ruta
alternativa permanente. Así sobreviven dos autoridades.

### Diagnóstico

- buscar importaciones y órdenes hacia rutas antiguas;
- revisar composición y configuración;
- comprobar bases, zócalos y procesos compartidos;
- inspeccionar pruebas que necesiten el árbol antiguo;
- localizar escrituras dobles y rutas alternativas.

### Solución

Caracterizar conducta, escribir contrato neutral y reimplementar detrás de los
puertos nuevos. La copia antigua solo se lee para inventario. La retirada exige
equivalencia y ausencia de consumidores.

### Prueba de lección

Controles de importaciones, raíces, procesos y configuración deben fallar ante
cualquier dependencia nueva. Las pruebas del producto pasan con el legado
inaccesible.

## 18. Interfaz como autoridad

### Síntoma

Cambiar una pantalla, respuesta HTTP o estado local modifica el ciclo de vida;
dos superficies aplican reglas distintas; ocultar un botón equivale a
autorización.

### Causa

La interfaz implementa política, guarda estado privado o escribe directamente
en adaptadores. No comparte el registro de comandos y consultas.

### Diagnóstico

- comparar HTTP, MCP, línea de órdenes y web;
- buscar escrituras fuera de aplicación;
- comprobar autenticación y autorización en cada camino;
- localizar estados locales que deciden cierre;
- comparar códigos, internacionalización y recibos.

### Solución

Un registro de comandos, mismos casos de uso y autorización en aplicación. La
interfaz presenta y solicita; no decide. Las consultas son de solo lectura.

### Prueba de lección

La misma orden por todas las superficies produce igual semántica y código. Un
actor sin permiso se rechaza aunque manipule la interfaz. Cambiar una
proyección no altera el `Goal`.

## 19. Limpieza agresiva que borra evidencia

### Síntoma

Desaparece un entorno, registro o cambio justo cuando se necesita estudiar una
incidencia. Un script considera «antiguo» equivalente a «seguro para borrar».

### Causa

Parada, desmontaje, retención y eliminación se mezclan. No se conocen
identidad, referencias, procesos vivos ni disposición.

### Diagnóstico

- buscar orden y principal que autorizó la retirada;
- comprobar inventario y huellas anteriores;
- localizar referencias, bloqueos y procesos;
- revisar previsualización y recibo;
- comprobar si la ruta era propia y exacta.

### Solución

Conservar por defecto. Sellar e inventariar como conservado pendiente de
revisión (`preserved_pending_review`).
Retirar mediante otro efecto autorizado, exacto e idempotente.

### Prueba de lección

Objeto desconocido, referenciado o vivo nunca se borra. La previsualización no
escribe. Interrumpir una retirada no amplía su alcance y conserva un recibo.

## 20. Señales derivadas

Varios síntomas suelen apuntar a las mismas causas:

| Señal | Buscar primero |
|---|---|
| «Funciona hasta reiniciar» | Estado en memoria, recibo perdido o cerca ausente |
| «Solo falla en paralelo» | Revisión esperada, reserva, propietario y escritura compartida |
| «Hay que reiniciar para que avance» | Conciliación ausente o bucle privado bloqueado |
| «La prueba pasa localmente» | Aislamiento, sujeto, omisión y dependencia ambiental |
| «La interfaz dice otra cosa» | Proyección caducada o política duplicada |
| «No sabemos si se aplicó» | Identidad externa, idempotencia y cuarentena |
| «No se puede limpiar» | Propiedad no registrada y retención sin clasificar |
| «Otro agente debe empezar de cero» | Punto de control y traspaso inexistentes |
| «Hay demasiados módulos» | Fragmentación nominal sin fronteras |
| «Todo está en un gestor» | Responsabilidades y autoridades mezcladas |

## 21. Plantilla de prueba de lección

```text
referencia_incidencia:
identificadores_capacidad:
síntoma reproducido:
causa arquitectónica:
invariante:
precondiciones:
acción:
resultado defectuoso esperado en la mutación:
resultado correcto:
frontera de reinicio o carrera:
negativo de seguridad:
orden focal:
evidencia:
```

La prueba se enlaza al registro de incidencias. Cerrar la incidencia requiere
ejecutarla sobre el sujeto corregido; citar una prueba histórica no basta.

## 22. Lecciones consultadas

Las consultas obligatorias para `GOV-16`, `ORC-10`, `ORC-15`, `ORC-23`,
`ORC-25`, `ORC-28`, `OPS-06`, `OPS-16`, `OPS-17`, `OPS-18`, `OPS-19`,
`EVD-04`, `EVD-06` y `EVD-10` no encontraron coincidencias automáticas para
la ruta de este capítulo. El hueco queda registrado.

Las fuentes históricas se consultaron bajo
`/home/alberto/Trabajo/orquestaV2-legacy-consulta`, en solo lectura. Entre las
fuentes principales:

- `docs/analisis_fallos_estructurales_orquesta_2026-07-10.md`
- `docs/diseno_control_activo_agentes.md`
- `docs/diseno_pools_capacidad.md`
- `docs/runbooks/incidencia_codex_quota_auto_replan_2026-06-13.md`
- `docs/incidencias/incidencia_orquesta_goal_first_lifecycle_status_shutdown_readiness_2026-07-02.md`
- `docs/incidencias/incidencia_orquesta_limpieza_config_metricas_falsas_2026-07-11.md`
- `docs/incidencias/incidencia_orquesta_retencion_runtime_codex_waves_2026-07-10.md`
- `docs/incidencias/incidencia_orquesta_codebase_memory_huerfanos_watchdog_2026-07-02.md`
- `modulos/orquesta-server/self_watchdog_policy_v0.go`
- `modulos/orquesta-state-file/agent_process_registry_v0.go`

## 23. Fuentes vigentes

- `AGENTS.md`
- `product/roadmap.json`
- `product/traceability/README.md`
- `product/traceability/rebuild_bugs.jsonl`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/reconstruccion/ruta_total_100.md`
- `docs/reconstruccion/inventario_total_legacy_2026-07-30.md`
- `docs/reconstruccion/GUIA_MAESTRA_AGENTES_INVENTARIO_Y_RECONSTRUCCION.md`

Los registros estructurados conservan estado. Este capítulo explica cómo
pensar, diagnosticar y escribir la prueba que evita repetir el fallo.
