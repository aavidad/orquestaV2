# Guía maestra para agentes de inventario y reconstrucción

Fecha: 2026-07-30.

Estado: procedimiento operativo para coordinar trabajo pequeño y verificable.
Esta guía no es el inventario histórico, no afirma que ese inventario esté
terminado, no crea capacidades, no acredita el producto y no tiene autoridad
superior a `AGENTS.md`, `product/roadmap.json`,
`product/capabilities.json`, `product/evidence/` ni
`docs/reconstruccion/ruta_total_100.md`.

Si esta guía discrepa con una de esas autoridades, se detiene únicamente el
conjunto de escritura afectado y se resuelve la contradicción en el catálogo
antes de programar.

## 1. Propósito

Esta guía permite que muchos agentes colaboren sin:

- perder conductas útiles de las versiones históricas;
- convertir todo hallazgo en trabajo obligatorio;
- repetir intentos fallidos sin entender su causa;
- importar el diseño o el entorno de ejecución antiguo;
- crear otro núcleo, ciclo de vida, planificador o almacén de estado;
- producir tareas tan grandes que el agente pierda contexto;
- producir fragmentación artificial que haga imposible comprender el sistema;
- confundir implementación, prueba local y acreditación;
- borrar trabajo, datos o evidencias antes de estudiarlos.

El resultado perseguido es una cadena comprobable:

```text
inventario completo
        ↓
caracterización semántica
        ↓
decisión de admisión
        ↓
tarea causal pequeña
        ↓
implementación y conexión reales
        ↓
revisión independiente
        ↓
ejercicio y acreditación
```

No se salta una etapa para acelerar. Sí se paralelizan elementos independientes
con conjuntos de escritura disjuntos.

## 2. Autoridad y fronteras

Cada agente debe conocer y respetar este orden:

1. `AGENTS.md` fija las fronteras generales.
2. `product/roadmap.json` fija capacidades, decisiones, dependencias y
   aceptación.
3. `product/capabilities.json` y `product/evidence/` fijan el estado
   acreditado de una revisión.
4. `docs/reconstruccion/ruta_total_100.md` fija el orden causal y los controles
   globales.
5. Los `AGENTS.md` locales gobiernan exclusivamente su ámbito y no pueden
   contradecir los puntos anteriores.
6. El legado solo aporta fuentes, caracterización, lecciones negativas y
   trazabilidad.

El árbol antiguo y sus copias se leen, pero no se modifican, importan, arrancan
ni se usan como dependencia. Una conducta útil se traduce primero a contrato,
prueba de caracterización o caso neutral y después se reimplementa detrás de la
arquitectura nueva.

Ningún agente puede:

- crear un identificador de capacidad o cambiar su estado mediante un
  documento;
- declarar equivalencia por nombres, número de funciones o parecido del
  código;
- copiar paquetes completos o crear puentes, dobles escrituras o alternativas
  hacia el entorno antiguo;
- publicar cambios Git, desplegar, publicar artefactos o mutar un sistema
  externo sin autorización explícita;
- borrar datos de un agente, candidato rechazado o fuente histórica pendiente
  de revisión;
- ampliar su conjunto de escritura porque haya descubierto trabajo relacionado.

## 3. Condición previa: el inventario debe cerrarse de verdad

Antes de convertir las carencias históricas en el plan de reconstrucción se
deben completar las oleadas y controles definidos en
`inventario_total_legacy_2026-07-30.md`.

Esto exige, como mínimo:

- manifiesto cerrado y revisado de todas las fuentes;
- censo físico reproducible;
- clasificación de código, datos, interfaces, configuración, migraciones,
  operaciones, pruebas, documentación e incidencias;
- deduplicación semántica que conserve todos los orígenes;
- mapeo de cada conducta contra las capacidades y evidencias de V2;
- disposición y razón revisadas para cada elemento;
- cero huecos sin resolver en los controles de omisiones;
- repetición independiente desde una copia limpia.

Un listado de funciones no es el inventario completo. Tampoco lo son una matriz
de módulos, una colección de documentos o un recuento de incidencias.

Mientras el inventario esté abierto se permiten tareas de:

- descubrimiento;
- censo;
- caracterización;
- reproducción neutral de una conducta;
- análisis de intentos y causas;
- mejora de los verificadores del propio inventario.

No se crean automáticamente tareas de producto por cada hallazgo. Las
decisiones de admisión y la ejecución de las prioridades registradas comienzan
después de cerrar y contrarrevisar el inventario.

## 4. Unidad de inventario

Cada elemento debe conservar al menos:

```text
referencia y revisión de origen:
resumen criptográfico:
tipo y familia:
conducta observada:
usuarios o consumidores:
invariantes:
fallos y lecciones negativas:
capacidades candidatas:
contratos de aceptación candidatos:
referencias V2:
evidencias V2:
intentos históricos relacionados:
disposición:
razón:
revisión independiente:
```

La unidad de deduplicación es la conducta, no el fichero. Varias funciones
pueden representar una sola conducta y una función puede contener varias.
Cuando se deduplica, no se pierde la procedencia de ninguna versión.

El agente de inventario no decide que una conducta deba implementarse. Entrega
hechos, fuentes, incertidumbres y una propuesta de clasificación.

## 5. Caracterización antes de decidir

Una conducta solo está caracterizada cuando otra persona puede responder, sin
leer toda la versión antigua:

- qué problema resolvía;
- para quién lo resolvía;
- qué entrada aceptaba y qué salida producía;
- qué estado podía leer o escribir;
- qué autoridad tomaba la decisión;
- qué permisos, secretos o efectos implicaba;
- cómo respondía a caída, repetición, concurrencia y reinicio;
- qué aspecto funcionó de verdad;
- qué aspecto fue simulado, parcial o nunca llegó a funcionar;
- qué incidencia o prueba demuestra cada afirmación;
- qué comportamiento debe conservarse y cuál debe evitarse.

La caracterización preferida es un contrato o caso neutral pequeño. No debe
arrastrar tipos, rutas, bases de datos, nombres de proveedores ni ciclos de vida
del legado.

Toda afirmación no demostrada se marca como incertidumbre. La ausencia de una
prueba histórica no demuestra que una conducta no existiera, y la existencia de
una prueba no demuestra que la composición real funcionara.

## 6. Filtro obligatorio de admisión

Cada conducta caracterizada pasa, en este orden, por cinco preguntas.

La aplicación de revisión es una superficie de ayuda sobre estos mismos
registros. Debe mostrar fuentes exactas, incertidumbres, intentos fallidos,
capacidades candidatas, pruebas y decisiones anteriores. Puede producir una
propuesta de disposición, pero no modificar por sí sola el catálogo, acreditar
una capacidad ni crear una tarea ejecutable. Cada decisión conserva actor,
proyecto, revisión esperada, fecha, razón y revisión independiente. Una
propuesta incompleta o en conflicto permanece sin aplicar.

### 6.1 ¿Es útil ahora?

Debe existir una necesidad vigente, un usuario o consumidor identificable y un
resultado distinguible. Se rechazan:

- código muerto o puramente accidental;
- duplicados sin semántica adicional;
- atajos creados solo para sostener otra autoridad antigua;
- opciones ya rechazadas explícitamente en el catálogo;
- complejidad que no resuelva una necesidad actual.

### 6.2 ¿Cabe en el producto canónico?

Debe mapearse a una capacidad y aceptación existentes. Si introduce una
obligación nueva, primero se registra una decisión en el catálogo mediante su
procedimiento autoritativo. Hasta entonces es candidata, no tarea de
implementación.

### 6.3 ¿Cumple todas las reglas del proyecto?

El cumplimiento es conjunto, no una selección. Entre otras reglas, debe
respetar:

- arquitectura hexagonal de dentro hacia fuera;
- `Goal` como única autoridad de identidad, generación y ciclo de vida;
- un único escritor de aplicación y un único planificador;
- una sola fuente activa de estado por despliegue;
- monolito modular como forma física predeterminada;
- puertos solo para fronteras sustituibles o consumidores reales;
- registro único de configuración y registro único de comandos;
- secretos únicamente mediante referencias y `CredentialStore`;
- identidad, proyecto, ejecución y permisos explícitos;
- efectos mediante intención, aprobación, intento y justificante separados;
- internacionalización completa, español predeterminado y reserva, BCP-47,
  formatos y paridad de catálogos;
- interfaces HTTP, MCP, línea de comandos y web sobre los mismos casos de uso;
- aislamiento de proyecto, negativos de seguridad e idempotencia;
- artefactos y evidencias ligados a revisiones y resúmenes inmutables;
- cero dependencia, escritura doble o ruta alternativa hacia el legado.

Una conducta incompatible se rechaza aunque funcionara bien en el pasado. No se
crea una tarea para «adaptarla» si eso conserva la infracción.

### 6.4 ¿Es técnicamente viable y operable?

Debe haber una vía razonada para implementarla, probarla, operarla, recuperarla
y retirarla. También deben conocerse sus costes previsibles de complejidad,
procesador, memoria, disco, red, credenciales y mantenimiento.

Que una biblioteca o proveedor exista no demuestra viabilidad. Las fronteras
externas requieren versión, permisos, aislamiento, prueba contractual, prueba
real cuando se promete disponibilidad y procedimiento de retirada.

### 6.5 ¿La solución está suficientemente fundada?

Solo se crea una tarea cuando:

- se comprende el mecanismo del problema;
- la solución propuesta rompe la causa, no solo oculta el síntoma;
- existe un criterio de aceptación ejecutable;
- pueden escribirse negativos que hagan fallar el enfoque antiguo;
- las dependencias están disponibles o representadas como bloqueos explícitos;
- el tamaño puede dividirse en unidades cohesionadas;
- un revisor independiente puede refutar la propuesta.

Si cualquiera de estas condiciones genera una duda sustancial, el elemento
queda **en estudio, sin tarea de implementación**. No se rellena el plan con
trabajo especulativo.

## 7. Tratamiento de intentos históricos fallidos

Una idea que tuvo muchos intentos fallidos no se rechaza ni se admite por
inercia. Primero se prepara una ficha de caso:

```text
necesidad original:
versiones e intentos:
qué llegó a funcionar:
qué no llegó a funcionar:
síntomas:
causa o causas demostradas:
supuestos falsos:
invariantes vulnerados:
coste operativo observado:
por qué el siguiente enfoque sería distinto:
prueba capaz de reproducir el fallo:
prueba capaz de distinguir la solución:
incertidumbres restantes:
```

Después se aplica una de estas disposiciones:

- **admitido**: la causa está comprendida, hay solución diferente y fundada,
  cumple todas las reglas y tiene aceptación ejecutable;
- **en estudio sin tarea**: la necesidad puede tener sentido, pero la causa o
  la solución siguen siendo dudosas;
- **rechazado con razón**: contradice una decisión o regla, no aporta valor
  vigente o su coste y riesgo no son justificables;
- **evidencia histórica**: conserva una lección, pero no representa una función
  que deba volver;
- **duplicado trazado**: otra conducta admitida cubre completamente su
  semántica.

Reintentar con otro nombre, otra versión o una biblioteca distinta no cuenta
como solución diferente. La diferencia debe aparecer en el contrato, la
autoridad, el aislamiento o el mecanismo probado.

Un caso en estudio no bloquea el trabajo admitido salvo que sea una dependencia
causal real. Se conserva y puede reevaluarse cuando aparezca evidencia nueva.

## 8. Conversión de una conducta admitida en tareas

La unidad de trabajo no es «implementar la función histórica». Es restaurar una
invariante concreta dentro del diseño canónico.

La descomposición normal es:

1. decisión o corrección de catálogo, si falta;
2. contrato neutral y prueba de caracterización;
3. modelo de dominio o necesidad consumidora;
4. puerto, únicamente si hay frontera real;
5. adaptador contractual;
6. composición explícita;
7. recuperación, repetición, concurrencia y negativos;
8. superficie pública e internacionalización;
9. prueba de extremo a extremo real;
10. revisión primaria y adversarial;
11. atestación y evidencia de la misma revisión;
12. retirada o clasificación final de la conducta antigua.

No todas las conductas necesitan doce tareas. Se fusionan pasos cuando el
conjunto siga siendo pequeño, causal y verificable. Se dividen cuando mezclar
pasos o responsabilidades dificulte comprender, revisar o revertir el cambio.

Cada tarea tiene una única razón de existir y un único resultado principal.
Una corrección, una función nueva y una migración solo comparten tarea si un
mismo control indivisible las acredita.

## 9. Encargo obligatorio para un agente

El coordinador entrega un paquete compacto:

```text
excepción de coordinación: arranque provisional directo | Orquesta acreditada
objetivo exacto:
capability IDs:
contrato de aceptación:
invariante:
autoridad que escribe:
fuentes y lecciones necesarias:
puertos afectados:
adaptadores afectados:
conjunto de escritura exclusivo:
dependencias causales acreditadas:
código o decisión antigua que permitirá retirar:
prueba contractual:
pruebas negativas, de carrera y reinicio:
prueba de extremo a extremo o control:
presupuesto de tamaño, complejidad, tiempo y disco:
acciones prohibidas:
forma de entrega:
```

Antes de editar, el agente:

1. lee `AGENTS.md`, el `AGENTS.md` local y solo los contratos necesarios;
2. comprueba el estado vivo y el estado Git;
3. ejecuta `scripts/consultar_lecciones_legacy.sh` con capacidad, ruta y
   operación;
4. confirma que su conjunto de escritura no se solapa;
5. comunica cualquier contradicción antes de tocar la parte afectada.

Una consulta sin coincidencias se registra como hueco de conocimiento, no como
permiso para improvisar.

## 10. Presupuestos para tareas rápidas

Los límites canónicos de una capacidad o registro prevalecen. Si no existe un
límite más estricto, los valores siguientes son señales obligatorias de
división, no nuevos contratos de producto:

- una invariante principal;
- una capacidad o un grupo inseparable expresamente justificado;
- de uno a seis ficheros manuales;
- hasta 300 líneas netas manuales;
- una orden focal de pruebas durante la autoría;
- una sesión de trabajo objetivo inferior a una hora;
- un paquete de contexto que pueda leerse completo al empezar, sin incluir
  artefactos grandes;
- cero cambios incidentales fuera del conjunto declarado.

Una migración, código generado o caso indivisible puede superar un umbral solo
con razón previa, presupuesto explícito y revisión más estricta. El exceso no
autoriza a esconder varias responsabilidades en una sola tarea.

El contexto del agente debe contener referencias exactas, resúmenes y pequeñas
porciones relevantes. Los artefactos grandes viajan por referencia y resumen,
no pegados a la orden. Si para explicar una tarea hay que resumir gran parte
del producto, la tarea todavía no está suficientemente dividida.

## 11. Monolito modular sin piezas mastodónticas

El producto es un monolito modular, pero eso no permite aplicaciones, módulos,
ficheros, servicios, gestores o funciones mastodónticos.

Cada tarea declara y revisa presupuestos para:

- líneas netas manuales;
- longitud y responsabilidad de funciones;
- complejidad ciclomática;
- dependencias entrantes y salientes;
- número de autoridades escritoras;
- bucles de control o segundo plano;
- almacenes de estado;
- comandos y superficies públicas;
- coste de comprender y depurar un fallo futuro.

Si una pieza supera su presupuesto, se divide por responsabilidad cohesionada
antes de continuar. La división debe conservar una dirección de dependencias
clara y no crear otro escritor, bucle o almacén.

También se prohíbe el extremo contrario: no se crea un paquete o fichero por
cada constante, estructura de transporte, interfaz o función. Una abstracción
nueva debe representar una frontera real, retirar duplicación nombrada o tener
consumidores reales según las reglas del proyecto.

La pregunta de depuración obligatoria es:

> Ante un fallo dentro de seis meses, ¿puede una persona localizar la autoridad,
> reproducir el caso y cambiar una sola responsabilidad sin recorrer un fichero
> dios ni saltar por decenas de envoltorios vacíos?

Si la respuesta no está demostrada por estructura, nombres, contratos y
pruebas, la tarea no cierra.

Al cierre, un revisor registra de forma explícita:

- tamaño neto y tamaño de las piezas principales;
- funciones que superen el límite local y su justificación;
- complejidad y dependencias nuevas;
- escritores, bucles y almacenes añadidos o retirados;
- fragmentación creada o evitada;
- ruta concreta de diagnóstico del fallo principal.

## 12. Cabeceras, manifiestos y documentación cercana al código

Toda aplicación creada o modificada por Orquesta debe disponer desde su
creación de un manifiesto o cabecera principal, en el idioma y catálogo que
correspondan a la aplicación, que describa:

- propósito y usuarios;
- alcance y exclusiones;
- arquitectura y módulos;
- autoridades escritoras;
- datos, permisos, secretos y efectos;
- arranque, diagnóstico, recuperación y parada;
- contratos y pruebas que la acreditan.

Todo módulo, herramienta, adaptador y fichero relevante lleva una cabecera
breve en castellano que permita conocer con rapidez:

- su responsabilidad;
- lo que deliberadamente no hace;
- su autoridad escritora, o que no escribe;
- los contratos de los que depende;
- la prueba que acredita su conducta.

En Go, la documentación de paquete cumple esa función para el módulo. Los
comentarios de símbolos exportados siguen las convenciones del lenguaje y una
cabecera específica de fichero se añade solo cuando expresa una frontera o
restricción real que no quede clara de otra forma. Se evitan comentarios que se
limiten a repetir el nombre del fichero o la función.

Las plantillas de aplicaciones, la revisión automática y la aceptación deben
comprobar estos manifiestos y cabeceras donde resulte razonablemente mecánico.
La ausencia de la información exigible impide el cierre.

Esta regla no autoriza texto humano incrustado fuera del sistema de
internacionalización. La aplicación respeta su idioma predeterminado, reserva,
catálogos y formatos. Si el catálogo vigente todavía no contiene la capacidad
o aceptación necesaria para hacer exigible esta regla, se resuelve primero esa
decisión; no se introduce como autoridad paralela desde esta guía.

## 13. Conjuntos de escritura y paralelismo

Dos agentes pueden trabajar en paralelo únicamente cuando sus conjuntos de
escritura son disjuntos y sus salidas no dependen causalmente una de otra.

El coordinador:

- reserva cada ruta antes de lanzar al agente;
- serializa catálogo, composición, migraciones compartidas e índices de
  evidencia cuando haya un único fichero;
- asigna contratos y adaptadores distintos en paralelo solo si su interfaz ya
  está congelada;
- evita que varios agentes editen el mismo archivo para después intentar
  reconciliar cambios incompatibles;
- mantiene visible quién posee cada conjunto, desde qué revisión y hasta qué
  entrega.

El agente no añade rutas «necesarias» por su cuenta. Si descubre una dependencia
no declarada, preserva el hallazgo y solicita una tarea hija o una ampliación
explícita.

Los cambios ajenos presentes en el árbol se conservan. Nunca se usan
restablecimientos destructivos ni se deshace trabajo que no pertenece al
encargo.

## 14. División y rescate de una tarea atascada

Una tarea se considera candidata a división cuando ocurre cualquiera de estas
señales:

- el agente necesita releer repetidamente contexto que no cabe en su encargo;
- descubre más de una invariante o responsabilidad;
- supera el presupuesto sin haber producido una prueba focal;
- espera una dependencia que puede separarse;
- repite dos veces el mismo enfoque sin nueva evidencia;
- modifica una interfaz compartida no congelada;
- no puede distinguir si el fallo pertenece al contrato, la composición o el
  entorno;
- la revisión produce varios defectos independientes.

El rescate no borra ni sobrescribe el trabajo. El agente entrega:

```text
estado exacto:
revisión base:
cambios y rutas:
pruebas ejecutadas y resultados:
hipótesis confirmadas:
hipótesis descartadas:
bloqueo concreto:
artefactos y resúmenes:
división propuesta:
datos que deben preservarse:
```

El coordinador congela el espacio de trabajo y divide por frontera causal:
contrato, dominio, adaptador, composición, recuperación, superficie o
evidencia. Un nuevo agente recibe solo una hija. No se relanza la misma orden
sin estudiar la entrega anterior.

Un bloqueo recuperable produce espera o replanificación. No consume un intento
del agente ni se convierte en fallo terminal por falta temporal de cuota,
máquina, credencial o revisor.

## 15. Entrega y cierre de cada tarea

La entrega mínima es:

```text
hecho:
invariante restaurado:
autoridad final:
ficheros modificados:
pruebas, negativos, mutaciones y extremo a extremo:
resultados exactos:
justificantes y revisión acreditada:
código o decisión retirados:
legado retirado o bloqueo de retirada:
líneas netas y complejidad:
escritores, bucles, almacenes y dependencias:
cabeceras y manifiestos comprobados:
riesgos P0/P1:
trabajo no realizado:
siguiente dependencia causal:
```

El agente inspecciona su diferencia y ejecuta `git diff --check`. Las pruebas
focales sirven para autoría; el cierre exige la verificación proporcional de
`AGENTS.md` y la aceptación de la capacidad.

Una tarea no queda cerrada porque:

- el código compile;
- pasen pruebas unitarias con dobles;
- exista un documento o un esquema;
- el agente afirme que terminó;
- el cambio sea alcanzable desde una herramienta;
- haya un acuse de recibo sin artefacto ni efecto;
- se haya creado una confirmación Git.

Implementación, conexión, ejercicio y acreditación son estados distintos.

## 16. Revisión, integración y acreditación

Autor, revisión primaria y revisión adversarial deben observar la misma
generación, árbol, diferencia y pruebas. Ninguna revisión sustituye a la otra
ni al Consejo cuando la política lo exija.

La revisión comprueba:

- semántica y aceptación;
- única autoridad y única fuente de estado;
- dependencias hexagonales;
- configuración, identidad, secretos y efectos;
- concurrencia, reinicio, repetición y parada;
- seguridad e internacionalización;
- tamaño, complejidad y facilidad de diagnóstico;
- cabeceras, manifiestos y documentación pública;
- evidencia negativa de que el enfoque fallido no reaparece;
- ausencia de dependencia o ruta alternativa hacia el legado.

La integración es explícita y usa la revisión base y la comparación esperadas.
Un conflicto no se resuelve ocultando cambios ajenos.

La acreditación se emite sobre resúmenes inmutables del árbol fuente,
binario o imagen y configuración efectiva. El justificante vive fuera del
sujeto que acredita o en un índice posterior que lo referencia.

## 17. Coordinación provisional y transición a Orquesta

Hasta que Orquesta acredite la gestión elástica de agentes y el entorno físico
Firecracker para agentes, la coordinación directa de Codex se limita a la
excepción provisional de arranque prevista en `AGENTS.md`:

- encargos acotados;
- conjuntos de escritura disjuntos;
- contexto mínimo;
- lectura del estado vivo antes de actuar;
- ningún uso ni modificación del entorno antiguo;
- entrega compacta con cambios, pruebas, riesgos y siguiente dependencia.

Esta coordinación provisional no se convierte en otra arquitectura, cola,
interfaz o ciclo de vida.

La transición a Orquesta como superficie principal solo se realiza cuando la
misma revisión acredite, según el catálogo y la ruta causal:

- cálculo de toda la demanda sin techo global oculto;
- incorporación y retirada dinámica de capacidad;
- envío, confirmación y recuperación de órdenes;
- cuota y disponibilidad observables y recuperables;
- arranque y parada individual con identidad exacta;
- conservación `preserved_pending_review` sin borrado automático;
- reinicio sin duplicar agentes ni aceptar escrituras caducadas;
- una microVM Firecracker aislada por agente;
- credenciales efímeras, sistema de archivos específico e intermediario por
  vsock;
- cero red IP directa del invitado;
- sellado, inventario y recuperación del entorno;
- pruebas físicas progresivas y cierre sin procesos o recursos propios.

La progresión física recomendada es una microVM, después cinco agentes, diez y
veinte. Las cohortes lógicas de 70 y 500 deben demostrar que no existe un techo
codificado; una ola física mayor solo se ejecuta con presupuesto, máquina y
autorización reales.

Hasta que ese control pase, no se atribuye a Orquesta una elasticidad o
autogestión que todavía no tenga. Después, Orquesta dirige por defecto las
tareas restantes; Codex directo queda para observación, integración o
desbloqueo acotado y documentado.

## 18. Orden de reconstrucción después del inventario

Una vez cerrado y contrarrevisado el inventario, el trabajo admitido se ordena
por dependencias, con esta prioridad operativa:

1. gestión elástica y durable de agentes;
2. órdenes posteriores, reanudación, cuota y recuperación;
3. ejecución física Firecracker, una microVM por agente;
4. pruebas progresivas de cohortes, conservación y cierre limpio;
5. transición de la coordinación a Orquesta;
6. resto de tareas admitidas, ordenadas por el DAG canónico;
7. controles globales, contrarrevisión y acreditación total.

La prioridad no autoriza a reabrir capacidades acreditadas sin una incidencia
demostrada ni a saltarse dependencias del catálogo.

Para cientos de tareas, el coordinador mantiene una cola causal, no una lista
plana. Solo se lanzan tareas cuyo conjunto de dependencias esté acreditado y
cuyo conjunto de escritura esté libre. La ausencia temporal de un agente no
debe dejar el objetivo detenido: se prepara capacidad nueva cuando la política,
la cuota y los recursos lo autoricen.

## 19. Controles contra autoengaño

Antes de declarar terminada una capacidad o el producto se intenta refutar el
cierre:

- se busca una conducta histórica aplicable sin disposición;
- se elige una muestra adversarial de cada familia;
- se repite el censo desde una copia limpia;
- se prueba caída, reinicio, repetición, escritura caducada y falta temporal de
  capacidad;
- se comprueba que las interfaces públicas comparten semántica, autorización e
  internacionalización;
- se comprueba que no sobreviven procesos, sockets, grupos de control,
  arrendamientos o credenciales propios;
- se revisan tamaños, complejidad, escritores, bucles, almacenes y dependencias;
- se verifica que cada aplicación y pieza relevante tenga la información de
  responsabilidad y diagnóstico exigida;
- se comprueba que la evidencia corresponde a la misma revisión y composición.

Un cero obtenido reduciendo el universo, excluyendo una fuente difícil o
marcando como tarea todo lo dudoso no es válido.

## 20. Estado honesto

Mientras el inventario no supere sus controles, debe decirse:

```text
inventario en curso;
fuentes y conductas todavía pendientes;
brechas confirmadas, pero cobertura total no demostrada.
```

Mientras una conducta admitida no esté acreditada:

```text
capacidad parcial o pendiente;
implementación, conexión, ejercicio o evidencia todavía insuficientes.
```

Solo el control global de `ruta_total_100.md`, sobre una única revisión y con
todos los elementos aplicables acreditados, permite afirmar que Orquesta está
terminada al 100 %.
