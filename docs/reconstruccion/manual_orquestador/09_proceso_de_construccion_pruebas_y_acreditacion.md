# 09. Proceso de construcción, pruebas y acreditación

> **Responsabilidad:** definir el proceso reproducible para construir,
> continuar, probar, revisar y acreditar un orquestador.
>
> **Alcance:** catálogo, cortes verticales, microtareas, contratos, pruebas,
> sujeto inmutable, recibos, integración y cierre.
>
> **No acredita:** este capítulo es una guía de trabajo; escribirlo o seguirlo
> parcialmente no acredita ninguna capacidad.

## Al terminar este capítulo, el lector sabrá…

- convertir una capacidad canónica en un corte vertical y microtareas causales;
- diseñar contratos y pruebas que detecten falsos resultados satisfactorios;
- continuar trabajo existente sin perder contexto ni pisar cambios ajenos;
- sellar un sujeto, revisarlo de forma independiente y emitir evidencia válida;
- integrar confirmaciones pequeñas y cerrar tareas con criterios verificables.

## Índice del capítulo

- [Vocabulario](#0-vocabulario-mínimo), [resultado](#1-resultado-que-debe-producir-el-proceso), [estado actual](#2-estado-actual-honesto-de-orquesta) y [catálogo](#3-antes-de-construir-catálogo-primero).
- [Continuación](#4-continuar-un-trabajo-existente), [cortes y microtareas](#5-cortes-verticales-y-microtareas) y [contratos](#6-diseñar-el-contrato-antes-del-código).
- [Falsos verdes](#7-falsos-verdes), [escalera de pruebas](#8-escalera-de-pruebas), [mutación](#9-pruebas-de-mutación) y [revisión](#10-revisión-independiente).
- [Sujeto inmutable](#11-sujeto-inmutable), [recibos](#12-recibos-y-evidencia), [integración](#13-integración-y-confirmaciones-pequeñas) y [órdenes seguras](#14-comandos-seguros).
- [Cierre](#15-cierre-de-microtarea), [diagnóstico](#16-diagnóstico-del-proceso), [lecciones](#17-lecciones-conservadas) y [fuentes](#18-fuentes-vigentes).

## 0. Vocabulario mínimo

- capacidad: conducta concreta que el producto debe ofrecer, identificada en el
  catálogo y acompañada de aceptación ejecutable.
- vertical: corte causal pequeño que atraviesa las capas necesarias para
  ofrecer una conducta real.
- conjunto de escritura: lista cerrada de ficheros que una tarea puede
  modificar.
- contrato: entrada, salida, invariantes, errores y pruebas comunes de una
  frontera.
- adaptador falso contractual: implementación de prueba que obedece el mismo
  contrato que un adaptador real y permite desarrollar la aplicación sin
  fingir una composición física.
- falso verde: resultado que parece satisfactorio, pero no demuestra la
  conducta prometida.
- prueba extremo a extremo: prueba que entra por una superficie pública,
  atraviesa la composición real y verifica el resultado observable. En algunos
  identificadores se abrevia `E2E`.
- prueba de mutación: modificación automática y deliberada del programa para
  comprobar que una prueba falla cuando se rompe un invariante.
- confirmación Git: objeto inmutable conocido habitualmente como `commit`.
- ciclo de vida (`lifecycle`): secuencia autoritativa de estados de una entidad.
- arrendamiento (`lease`): concesión temporal y exclusiva sobre un recurso.
- cerca (`fence`): ordinal que invalida a un propietario anterior.
- buzón causal (`mailbox`): intercambio durable y ordenado ligado a identidades
  y ejecución exactas.
- recibo (`receipt`): registro estructurado de una ejecución o efecto, con
  identidad, tiempo, resultado y huellas.
- huella criptográfica (`digest` o `hash`): resumen verificable del contenido
  de un sujeto.
- sujeto candidato: conjunto exacto e inmutable de código, binario o imagen y
  configuración que se somete a prueba y revisión.
- acreditación: estado final que solo se concede cuando el contrato se ha
  implementado, conectado, ejercitado y demostrado sobre el mismo sujeto.

## 1. Resultado que debe producir el proceso

El proceso de construcción debe permitir dos usos:

1. crear un orquestador desde cero sin improvisar otra arquitectura;
2. continuar un orquestador existente sin perder decisiones, repetir trabajo
   ni declarar terminadas capacidades incompletas.

El resultado no se mide por líneas, tiempo, número de agentes, confirmaciones
Git ni porcentaje subjetivo. Se mide por capacidades canónicas en esta
secuencia:

```text
declarada -> implementada -> conectada -> ejercitada -> acreditada
```

Solo «acreditada» cuenta como terminada. Una capacidad documentada, simulada,
compilada o probada únicamente con un adaptador falso sigue sin acreditar si su
contrato exige composición real.

El proceso debe conservar siempre:

- una autoridad única del producto;
- dependencias causales visibles;
- tareas pequeñas y revisables;
- evidencia reproducible;
- errores y falsos verdes como lecciones;
- posibilidad de reanudar desde el estado vivo;
- cero acciones externas no autorizadas.

## 2. Estado actual honesto de Orquesta

La revisión actual de `product/roadmap.json` contiene 257 capacidades, 38
verticales y 38 contratos: 22 ejecutables y 16 planificados. V1–V22 están
acreditadas; V23–V38 permanecen planificadas. Un contrato planificado forma
parte del catálogo, pero no demuestra implementación, conexión, ejercicio ni
acreditación.

| Área | Estado | Situación comprobada |
|---|---|---|
| Gobierno del catálogo de 257 capacidades y 38 verticales | ACREDITADO | `GOV-16` fija identidades y estados; no significa que las 38 verticales estén acreditadas. |
| Contrato canónico V38 | PENDIENTE | V38 está adoptada como vertical planificada y prioritaria, pero todavía no está implementada, conectada, ejercitada ni acreditada. |
| Estados canónicos de capacidad | ACREDITADO | `GOV-16` impide confundir declaración, implementación y acreditación. |
| Verticales V01–V22 | ACREDITADO | Tienen contratos ejecutables y recibos de su revisión; no implican que el producto total esté terminado. |
| Atestación de pruebas requeridas | ACREDITADO | V17 vincula pruebas, cambio, árbol, generación y espacio de trabajo mediante un atestador independiente. |
| Autor, revisión principal y revisión adversarial | ACREDITADO | V18 exige tres lanzamientos distintos sobre la misma generación, árbol, diferencia y pruebas. |
| Consejo condicionado por política | ACREDITADO | V19 está separado de las dos revisiones y no las sustituye. |
| Protocolo de evidencia V3 sobre fuente Git sellada | ACREDITADO | Los recibos vigentes vinculan confirmación, árbol, descriptor contractual, salida y huella del candidato. |
| Plan inicial público con roles, oleadas y conjuntos de escritura | PENDIENTE | `WIZ-24` pertenece a V23 y no está acreditado. |
| Prueba de mutación selectiva del producto nuevo | PENDIENTE | `EVD-10` pertenece a V34. El piloto histórico no acredita esta capacidad. |
| Validación final de todas las superficies | PENDIENTE | `STG-17` pertenece a V34. |
| Sujeto completo de publicación: fuente, binario o imagen y configuración efectiva | PENDIENTE | La regla está definida, pero el cierre global V34 todavía no está acreditado. |
| Todas las capacidades aceptadas acreditadas, P0/P1 cero y legado retirado | PENDIENTE | Es el criterio global V34; ningún capítulo del manual lo reemplaza. |

El estado se consulta sin mezclar sujetos:

1. `product/roadmap.json` fija el catálogo, las decisiones y los contratos
   adoptados de la revisión;
2. `product/capabilities.json` y `product/evidence/` fijan la acreditación de
   una publicación concreta;
3. `docs/reconstruccion/ruta_total_100.md` ordena causalmente lo adoptado.

El texto narrativo ayuda a interpretar, pero no modifica ese estado.

### 2.1 Orden acreditador de V38

V38 posee solo `ORC-28`. `ORC-15` continúa en V27 y `OPS-16`/`OPS-17` en
V32; continuidad de mensajes, parada exacta y conservación son conductas
acotadas de V38, no una reasignación ni una nueva acreditación de esas
capacidades.

Su aceptación conserva tres compuertas:

| Compuerta | Sujeto | Resultado permitido |
|---|---|---|
| A | Núcleo elástico neutral, sin exigir KVM ni Firecracker. | Evidencia de A; no acredita V38. |
| B | Firecracker activado expresamente, sin sustitución automática y con una microVM por agente. | Evidencia de B; no acredita V38 por sí sola. |
| C | Ola física 1/5/10/16/20 del mismo candidato que superó A y B. | Permite acreditar V38 solo después de A y B. |

Solo A+B+C del mismo candidato acreditan V38. Las demandas lógicas
1/16/70/500 y los pasos físicos 1/5/10/16/20 son series distintas.

V23 no depende de Firecracker, KVM ni microVM, y V38/Firecracker no dependen de
cerrar V23. La independencia es bidireccional: ninguna tarea ni fallo de una
vertical puede usarse como compuerta inventada de la otra.

Invariante transversal que debe conservar cualquier adaptador de efectos:

| Código | Regla | Prueba mínima |
|---|---|---|
| `intento_durable_antes_del_efecto` | La intención, la aprobación, el intento y su clave idempotente se persisten antes de llamar al sistema externo. | Caer antes y después de la llamada; recuperar por identidad externa e idempotencia sin repetir un efecto desconocido. |

## 3. Antes de construir: catálogo primero

### 3.1 No empezar por paquetes

El error inicial más costoso es crear componentes porque «parecen necesarios».
Primero se responde:

- ¿qué intención humana se quiere satisfacer?;
- ¿qué capacidad canónica representa esa conducta?;
- ¿está aceptada, rechazada o condicionada?;
- ¿qué vertical la posee?;
- ¿qué dependencias deben existir antes?;
- ¿qué contrato ejecutable demuestra el resultado?;
- ¿qué código antiguo podrá retirarse cuando se acredite?;
- ¿qué riesgo y presupuesto tiene?

Solo entonces se eligen tipos, puertos, adaptadores y ficheros.

Una interfaz nueva debe responder a una necesidad consumidora y a una frontera
sustituible real. Un paquete nuevo no se justifica por tener nombre propio.

### 3.2 Entrada estructurada de una tarea

Toda tarea comienza con:

```text
capacidad:
invariante:
autoridad que escribe:
puertos afectados:
adaptadores afectados:
conjunto de escritura:
dependencias causales:
código antiguo que permitirá retirar:
prueba de contrato:
prueba negativa o de mutación:
prueba extremo a extremo o criterio físico:
presupuesto de líneas, fichas, tiempo y disco:
```

La ausencia de alguno de estos datos no siempre bloquea el análisis, pero sí
impide empezar una modificación cuyo alcance no pueda acotarse.

### 3.3 Pseudocódigo de selección

```text
elegir_siguiente_tarea():
    estado = leer_catalogo_capacidades_y_evidencias()
    pendientes = capacidades_aceptadas_no_acreditadas(estado)
    candidatas = []

    para cada capacidad en pendientes:
        si todas_sus_dependencias_estan_acreditadas:
            candidatas.agregar(capacidad)

    ordenar(candidatas, por_secuencia_causal_y_prioridad_operador)
    capacidad = primera(candidatas)
    vertical = menor_conducta_ejecutable_que_la_demuestra(capacidad)
    devolver dividir_en_microtareas_con_conjuntos_disjuntos(vertical)
```

No se elige una tarea porque un fichero sea cómodo de modificar. Se elige por
dependencia causal y aceptación.

## 4. Continuar un trabajo existente

Antes de diseñar o editar:

1. leer el `AGENTS.md` raíz;
2. leer el corte breve y el traspaso vigente;
3. consultar `git status --short --branch`;
4. identificar cambios ajenos y no tocarlos;
5. leer el estado de la capacidad y su evidencia;
6. leer el `AGENTS.md` local del módulo;
7. ejecutar la consulta de lecciones por capacidad, ruta y operación;
8. inspeccionar únicamente los contratos y consumidores necesarios;
9. declarar el conjunto de escritura;
10. ejecutar una prueba focal que establezca el estado inicial.

Pseudocódigo:

```text
reanudar(tarea):
    vivo = leer_estado_vivo()
    arbol = leer_estado_git_sin_modificar()
    si existe_contradiccion_de_autoridad(tarea, vivo):
        registrar_y_resolver_en_catalogo()
        detener_solo_este_conjunto_de_escritura()

    proteger_cambios_ajenos(arbol)
    lecciones = consultar_lecciones(tarea.capacidad, tarea.ruta, tarea.operacion)
    contrato = leer_aceptacion_y_consumidores(tarea)
    base = ejecutar_prueba_focal(contrato)
    devolver contexto_minimo(vivo, lecciones, contrato, base)
```

No se repite una auditoría completa si ya existe un inventario trazable. Sí se
vuelve a comprobar el hecho concreto que puede haber cambiado.

## 5. Cortes verticales y microtareas

### 5.1 Qué es un corte vertical

Un corte útil ofrece una conducta demostrable atravesando solo lo necesario:

```text
contrato -> aplicación -> puerto -> adaptador -> composición -> prueba pública
```

No todas las tareas necesitan todas esas piezas. Una tarea de dominio puro
puede cerrar con propiedades y unitarias. Una promesa de efecto físico no.

Un corte horizontal como «crear todas las interfaces» genera estructuras sin
consumidores y aplaza los fallos de composición. Es preferible implementar una
conducta pequeña completa.

### 5.2 Cómo dividir

Una microtarea debe:

- tener una sola responsabilidad;
- caber en un conjunto de escritura pequeño;
- retirar o sustituir una duplicación concreta;
- entregar una prueba focal rápida;
- dejar una salida consumible por la siguiente tarea;
- expresar su bloqueo sin inventar un camino alternativo.

Ejemplo:

```text
1. contrato neutral y adaptador falso;
2. modelo y validación de aplicación;
3. persistencia y migración;
4. adaptador real;
5. composición;
6. prueba de concurrencia y reinicio;
7. prueba extremo a extremo;
8. revisión y acreditación.
```

Si dos tareas editan el mismo fichero, se serializan o se rediseña la frontera.
La supuesta rapidez de dos agentes no compensa una integración confusa.

### 5.3 Presupuesto y tamaño

Antes de implementar se fijan límites:

- líneas productivas netas;
- complejidad y número de escritores;
- nuevos bucles, almacenes y comandos;
- ficheros tocados;
- tiempo de prueba;
- fichas de modelo;
- disco y procesos;
- efectos externos.

Superar un límite obliga a parar, explicar la causa y dividir o retirar código.
No se amplía silenciosamente.

## 6. Diseñar el contrato antes del código

Un contrato suficiente define:

- entradas válidas y su identidad;
- salida y códigos de error;
- invariantes;
- idempotencia;
- autorización;
- presupuesto;
- límites;
- concurrencia;
- recuperación tras caída;
- recibos;
- comportamiento de al menos dos adaptadores, uno de ellos falso;
- prueba negativa.

El adaptador falso no devuelve siempre éxito. Debe poder simular:

- demora y cancelación;
- falta temporal;
- rechazo definitivo;
- aplicación desconocida;
- respuesta duplicada;
- identidad obsoleta;
- corrupción;
- caída entre fronteras durables.

Así se comprueba la política de aplicación sin trasladarla al proveedor.

## 7. Falsos verdes

Ninguno de estos hechos acredita por sí solo:

- el código compila;
- una prueba unitaria pasa;
- el agente escribió «terminado»;
- el proveedor aceptó una petición;
- existe un archivo con forma de artefacto;
- una función es alcanzable;
- una interfaz tiene una implementación falsa;
- se ejecutó una prueba distinta a la del contrato;
- una prueba fue omitida por etiqueta, plataforma o condición;
- un proceso terminó con salida aparentemente correcta, pero código no cero;
- un recibo pertenece a otra confirmación Git;
- una revisión miró otro árbol, diferencia o configuración;
- una métrica se llama «duplicación» sin comparar semántica;
- un documento afirma que la capacidad existe.

Para cada riesgo se escribe una prueba que falle si se sustituye la evidencia
real por ese atajo.

### 7.1 Comprobación contra falsos verdes

```text
validar_resultado(contrato, ejecucion, sujeto):
    exigir(ejecucion.codigo_salida == 0)
    exigir(ejecucion.orden == contrato.orden_exacta)
    exigir(ejecucion.sujeto_huella == sujeto.huella)
    exigir(ejecucion.no_fue_omitida)
    exigir(salida_regular_y_huella_valida)
    exigir(recibo_esquema_estricto)
    exigir(revision_independiente_mismo_sujeto)
    devolver verdadero
```

## 8. Escalera de pruebas

La clase de riesgo decide qué escalones son obligatorios.

### 8.1 Unitarias y propiedades

Comprueban reglas puras:

- transiciones válidas;
- invariantes de referencias;
- derivaciones deterministas;
- serialización canónica;
- límites y normalización;
- propiedades para muchos valores generados.

Son rápidas y se ejecutan durante la autoría. No prueban cableado.

### 8.2 Contrato común

La misma batería se ejecuta contra cada adaptador:

- memoria o falso;
- SQLite y PostgreSQL;
- sistema de archivos y almacén remoto;
- proveedor local y proveedores externos.

Impide que una marca cambie la semántica del núcleo.

### 8.3 Integración

Comprueba dos o más componentes reales:

- repositorio y migraciones;
- aplicación y bandeja transaccional;
- configuración y composición;
- espacio de trabajo y Git;
- artefactos y atestador.

Debe usar temporales privados, límites y limpieza de recursos propios.

### 8.4 Carreras

El detector `-race` de Go busca accesos concurrentes inseguros. Además deben
existir pruebas deterministas para:

- dos reclamaciones de la misma acción;
- cerca antigua contra nueva;
- escritura y copia concurrentes;
- cierre mientras llega un resultado;
- dos revisiones del mismo papel;
- publicación idempotente;
- lectura durante una migración permitida.

Una pasada sin aviso de `-race` no demuestra por sí sola idempotencia ni
atomicidad.

### 8.5 Reinicio y repetición

Se inyectan caídas:

- antes y después de cada escritura durable;
- antes y después de un efecto;
- después de aplicar el efecto y antes del recibo;
- durante publicación;
- durante parada;
- durante copia o restauración.

Después se reconstruye desde estado y se exige:

- cero efecto duplicado;
- terminales no reejecutados;
- pendientes reclamables;
- cercas antiguas rechazadas;
- resultado desconocido en cuarentena;
- misma identidad causal.

### 8.6 Seguridad

Pruebas negativas según la frontera:

- referencias de otro proyecto;
- actor sin permiso;
- secreto en registro o artefacto;
- ruta transversal;
- enlace simbólico o duro;
- propietario o modo incorrectos;
- archivo especial;
- expansión excesiva;
- suplantación de proceso o proveedor;
- efecto sin aprobación;
- salida de red prohibida.

### 8.7 Extremo a extremo

Una prueba extremo a extremo entra por HTTP, MCP, línea de órdenes o web y
atraviesa la composición seleccionada. Debe comprobar conducta, no solo estado
HTTP.

Por ejemplo:

```text
intención pública
 -> Goal durable
 -> trabajo reclamado
 -> agente real
 -> cambio
 -> pruebas independientes
 -> dos revisiones
 -> integración
 -> cierre
 -> reinicio sin repetición
```

### 8.8 Física

Se exige cuando la promesa depende de:

- un proveedor real;
- KVM o una microVM;
- red;
- sistema operativo;
- navegador;
- dispositivo;
- servicio de identidad;
- base de datos remota;
- despliegue.

Una simulación lógica puede preparar la prueba, pero no sustituirla.

## 9. Pruebas de mutación

No se aplican indiscriminadamente a todo el repositorio. Se eligen invariantes
críticos:

- cierre terminal;
- autorización;
- comparación de revisión;
- cerca;
- idempotencia;
- selección de sujeto;
- validación de rutas;
- preservación de recibos;
- no repetición tras reinicio.

Procedimiento:

1. seleccionar paquete e invariante;
2. registrar herramienta, versión y configuración;
3. ejecutar sobre una copia aislada;
4. clasificar mutantes supervivientes;
5. añadir pruebas que maten los graves;
6. repetir;
7. conservar resultado y huella.

Un porcentaje global no decide el cierre. Un mutante de autorización
superviviente es grave aunque la puntuación total sea alta.

El piloto histórico enseñó a mantener alcance cerrado, activación explícita y
clasificación de supervivientes. No se copia su dependencia ni se atribuye al
producto nuevo hasta que `EVD-10` tenga composición y evidencia propias.

## 10. Revisión independiente

### 10.1 Papeles distintos

Todo cambio de código tiene:

- autor;
- revisor principal;
- revisor adversarial.

En Orquesta acreditada son tres lanzamientos diferentes. Los revisores reciben
el mismo sujeto:

- generación;
- árbol;
- diferencia;
- pruebas;
- configuración relevante;
- huellas.

Cambiar cualquiera invalida decisiones anteriores.

### 10.2 Qué revisa cada papel

El principal comprueba:

- correspondencia con la capacidad;
- claridad y mantenibilidad;
- contratos y casos normales;
- pruebas suficientes;
- integración.

El adversarial intenta romper:

- autoridad;
- causalidad;
- idempotencia;
- concurrencia;
- seguridad;
- recuperación;
- evidencia;
- retirada.

Una crítica válida produce una tarea causal de corrección y una nueva revisión
del sujeto resultante. No se edita el dictamen para convertirlo en aprobación.

### 10.3 Consejo

El Consejo es una decisión colegiada condicionada por política. Resuelve
alternativas, riesgo, disenso y veto; no sustituye la revisión principal ni la
adversarial. Omitirlo por política autorizada conserva principal, motivo,
fecha y huella.

## 11. Sujeto inmutable

### 11.1 Qué se sella

Para una capacidad de código:

- confirmación Git exacta;
- árbol;
- lista ordenada de ficheros candidatos;
- modos y eliminaciones;
- descriptor contractual;
- configuración de prueba;
- herramientas y versiones.

Para una publicación:

- árbol fuente;
- binario o imagen;
- configuración efectiva redactada;
- migraciones;
- manifiestos de activos.

La identidad se calcula con un esquema versionado que encuadra longitud, ruta,
modo y contenido. Concatenar textos sin encuadre permite ambigüedades.

### 11.2 Fuente limpia

La ejecución acreditadora ocurre sobre una copia Git separada, limpia y
desacoplada de la rama de autoría. El recibo vigente conserva:

- confirmación y árbol;
- estado limpio;
- huella del estado Git vacío;
- huella del descriptor;
- huella del conjunto candidato.

Los cambios posteriores del árbol de trabajo no reescriben una evidencia
histórica.

### 11.3 Evitar la autorreferencia

El recibo y la salida se crean después de ejecutar las pruebas. No forman parte
del sujeto cuya huella declaran. Viven fuera de él o en una confirmación
posterior que lo referencia.

Nunca se exige que un archivo contenga la huella del árbol que incluye ese
mismo archivo.

## 12. Recibos y evidencia

### 12.1 Estructura mínima

```text
schema_version
evidence_kind
contract
result
executed_at
execution:
  argv
  source_git_commit_oid
  source_worktree_state
  source_status_porcelain_sha256
  exit_code
  combined_output_path
  combined_output_sha256
  tool_versions
contract_descriptor_sha256
candidate_digest_algorithm
candidate_sha256
sealed_source:
  identity_kind
  git_commit_oid
  git_tree_oid
  contract_descriptor_blob_oid
  sha256
```

Los nombres anteriores son identificadores literales del esquema. En
castellano, el primer bloque declara versión, clase de evidencia, contrato,
resultado e instante; `execution` describe la ejecución observada:
`argv` es el vector de argumentos, los campos `source_*` identifican la
confirmación y el estado de la copia de trabajo de origen, `exit_code` es el
código de salida, `combined_output_*` localiza y verifica la salida conjunta y
`tool_versions` enumera las versiones de herramientas. Los campos
`contract_descriptor_*` y `candidate_*` verifican el descriptor y el candidato;
`sealed_source` identifica la fuente sellada mediante la clase de identidad, la
confirmación, el árbol, el objeto del descriptor y sus huellas criptográficas.

El esquema vigente V3 llama `fixture_sha256` a la huella del descriptor de
aceptación. Se conserva ese identificador literal por compatibilidad; la
explicación humana usa «descriptor contractual».

### 12.2 Secuencia de acreditación

```text
acreditar(capacidad, candidato):
    contrato = leer_contrato_canónico(capacidad)
    sujeto = sellar_candidato(candidato, contrato)
    exigir_copia_git_limpia_y_separada(sujeto)

    ejecucion = atestador.ejecutar(contrato.orden_exacta, sujeto)
    salida = sellar_salida(ejecucion)
    recibo = construir_recibo(contrato, sujeto, ejecucion, salida)
    validar_esquema_huellas_tiempos_y_ascendencia(recibo)

    revisiones = obtener_revisiones_independientes(sujeto, salida)
    exigir_aprobaciones_vigentes(revisiones)
    exigir_consejo_si_politica_lo_ordena(sujeto)

    registrar_evidencia(capacidad, recibo)
    promover_estado(capacidad, "accredited")
```

La promoción ocurre al final. Un recibo existente no acredita una capacidad
que no esté vinculada explícitamente en el catálogo.

### 12.3 Fallos que invalidan el recibo

- sujeto sucio;
- confirmación abreviada o inexistente;
- descriptor distinto;
- lista candidata desordenada, duplicada o con rutas ascendentes;
- salida ausente, enlace o huella errónea;
- código de salida no cero;
- ejecución anterior al sujeto sellado;
- resultado `PASS` escrito a mano;
- herramienta o versión desconocida;
- recibo de otro contrato;
- revisión de otro sujeto;
- candidato cambiado después de probar.

## 13. Integración y confirmaciones pequeñas

La integración es explícita. Antes:

1. revisar `git status`;
2. inspeccionar la diferencia completa del conjunto de escritura;
3. ejecutar `git diff --check`;
4. ejecutar pruebas focales;
5. ejecutar pruebas proporcionales;
6. confirmar que no se incluyen cambios ajenos;
7. comprobar temporales y procesos propios;
8. registrar riesgos y dependencia siguiente.

Cada confirmación Git:

- tiene una sola responsabilidad;
- usa mensaje breve en castellano;
- deja el árbol compilable cuando sea razonable;
- incluye su prueba o contrato;
- no mezcla reestructuración, función y migración sin aceptación común.

No se envía a un repositorio remoto, se despliega ni se publica sin orden
expresa y efecto gobernado.

## 14. Comandos seguros

### 14.1 Inspección previa

```bash
git status --short --branch
git diff --name-only
git diff --stat
df -h .
```

No se ejecuta una batería costosa antes de conocer el conjunto de escritura y
el presupuesto.

### 14.2 Durante la autoría

```bash
go test -mod=vendor -count=1 ./ruta/del/paquete
go test -mod=vendor -count=1 ./ruta/del/paquete -run '^PruebaFocal$'
git diff --check
```

Se usan argumentos directos. No se construyen órdenes de consola a partir de
texto de usuario o de un agente.

### 14.3 Cierre transversal

Cuando el riesgo lo exige:

```bash
git diff --check
go test -mod=vendor -count=1 .
go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
```

Para concurrencia:

```bash
go test -mod=vendor -race -count=1 ./rutas/afectadas
```

El contrato de cada vertical puede ordenar pruebas adicionales. Se ejecuta
exactamente sobre su sujeto sellado y con tiempo límite explícito.

No se usan para «limpiar» el trabajo:

- `git reset --hard`;
- restauración destructiva de ficheros ajenos;
- borrados recursivos amplios;
- órdenes sobre rutas derivadas de variables no validadas.

## 15. Cierre de microtarea

La entrega conserva:

```text
hecho:
invariante restaurado:
autoridad final:
pruebas, negativos, mutaciones y extremo a extremo:
recibos y revisión acreditada:
código o decisión retirados:
legado retirado o bloqueo de retirada:
líneas netas y complejidad:
riesgos P0/P1:
siguiente dependencia causal:
```

Si falta una prueba física prometida, se dice «pendiente». Si una dependencia
externa no está disponible, se dice «bloqueada». No se cambia a «terminada»
para vaciar una lista.

## 16. Diagnóstico del proceso

| Síntoma | Causa probable | Comprobación |
|---|---|---|
| Muchas tareas abiertas, poco resultado | Cortes horizontales o dependencias ignoradas | Relacionar cada tarea con capacidad, contrato y consumidor |
| La misma función se rehace varias veces | Inventario o lección no consultados | Buscar semántica y decisiones antes de editar |
| Pruebas verdes, composición rota | Solo unitarias o adaptadores falsos | Ejecutar integración y extremo a extremo reales |
| La revisión aprueba un cambio distinto | Sujeto no sellado o huellas incompletas | Comparar generación, árbol, diferencia y pruebas |
| Un fallo reaparece tras reinicio | Frontera de caída no probada | Inyectar caída antes y después de la escritura o efecto |
| El recibo no puede reproducirse | Orden, herramienta o salida no vinculadas | Validar esquema V3 y huellas |
| Dos agentes pisan cambios | Conjuntos de escritura solapados | Serializar o dividir por frontera |
| Crecen paquetes y almacenes | Abstracciones sin consumidor o autoridad duplicada | Aplicar presupuesto y retirar duplicación |
| Se declara un porcentaje sin base | Progreso narrativo | Contar capacidades acreditadas y bloqueos exactos |

La primera respuesta es observar y reconstruir la causalidad, no añadir otra
capa que esconda el fallo.

## 17. Lecciones conservadas

Las consultas obligatorias para `GOV-16`, `STG-13`, `STG-14`, `EVD-04`,
`EVD-06`, `EVD-10` y `STG-17` no encontraron coincidencias automáticas para
la ruta de este capítulo. El hueco se registra y no autoriza a omitir la
consulta en tareas posteriores.

La lectura histórica en
`/home/alberto/Trabajo/orquestaV2-legacy-consulta`, siempre en solo lectura,
aporta estas lecciones:

- una herramienta de mutación debe tener alcance cerrado, activación expresa y
  clasificación de supervivientes;
- una puntuación alta no compensa un mutante crítico;
- una métrica que agrupa nombres parecidos no demuestra código duplicado;
- una prueba obligatoria inválida debe rechazarse antes de lanzar al autor;
- repetir un trabajo sin información nueva crea ciclos costosos;
- una prueba focal y una observación real valen más que un estado narrativo;
- fallos y falsos verdes deben conservar capacidad, causa, invariante y prueba.

Fuentes principales:

- `docs/runbooks/mutation_testing_piloto_2026-07-04.md`
- `docs/incidencias/incidencia_orquesta_limpieza_config_metricas_falsas_2026-07-11.md`
- `docs/incidencias/incidencia_orquesta_race_harness_shutdown_2026-07-11.md`
- `modulos/orquesta-state-file/required_test_evidence_store_v0.go`

## 18. Fuentes vigentes

- `AGENTS.md`
- `product/roadmap.json`
- `product/capabilities.json`
- `product/evidence/`
- `docs/reconstruccion/ruta_total_100.md`
- `docs/reconstruccion/LEEME_AGENTE_ORQUESTAV2.md`
- `docs/reconstruccion/handoff_continuacion_agente_2026-07-30.md`
- `acceptance/evidence_protocol_test.go`
- `acceptance/evidence_support_test.go`
- `acceptance/v17_test_attestor_test.go`
- `acceptance/v18_independent_reviews_test.go`

Este capítulo define cómo trabajar. El catálogo y los recibos ejecutados
deciden qué está terminado.
