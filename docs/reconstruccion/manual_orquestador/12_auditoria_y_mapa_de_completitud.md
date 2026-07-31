# 12. Auditoría y mapa de completitud

> **Responsabilidad:** definir una auditoría reproducible que mida la
> completitud del orquestador y convierta huecos viables en trabajo causal.
>
> **Alcance:** catálogo, evidencias, censo, código, operación, seguridad,
> agentes, Firecracker, superficies, tareas, traspaso y cierre global.
>
> **No acredita:** ejecutar parcialmente este procedimiento o completar una
> tabla no acredita el producto ni autoriza el censo físico histórico.

## Al terminar este capítulo, el lector sabrá…

- recorrer el universo canónico sin copiarlo como segunda autoridad;
- construir y verificar una matriz completa de requisito y evidencia;
- auditar código, operación, seguridad, agentes, Firecracker e interfaces;
- clasificar huecos y convertir solo los viables en microtareas causales;
- preparar un traspaso reproducible y evaluar la condición global del 100 %.

## Índice del capítulo

- [Vocabulario](#0-vocabulario), [principio de autoridad](#1-principio-auditar-sin-crear-otra-autoridad), [estado actual](#2-estado-actual-honesto-al-redactar-este-capítulo) y [fuentes](#3-autoridades-y-fuentes).
- [Estados](#4-estados-que-no-deben-confundirse), [registro](#5-registro-de-una-ejecución-de-auditoría), [procedimiento](#6-pseudocódigo-general) y [preparación](#7-oleada-0-congelar-método-y-alcance).
- [Catálogo](#8-oleada-1-catálogo-y-dependencias), [matriz](#9-oleada-2-matriz-requisito-evidencia), [producto y Git](#10-oleada-3-producto-nuevo-y-git) y [seguridad](#11-oleada-4-configuración-identidad-y-seguridad).
- [Operación](#12-oleada-5-procesos-disco-y-operación), [agentes](#13-oleada-6-agentes-y-capacidad), [Firecracker](#14-oleada-7-firecracker), [interfaces](#15-oleada-8-superficies-e-internacionalización) y [legado](#16-oleada-9-legado-y-censo).
- [Filtro](#17-filtro-obligatorio-de-cinco-preguntas), [microtareas](#18-convertir-huecos-viables-en-microtareas), [paralelismo](#19-paralelismo-seguro) y [pruebas del auditor](#20-pruebas-del-propio-auditor).
- [Diagnóstico](#21-diagnóstico-de-discrepancias), [traspaso](#22-traspaso-de-auditoría), [100 %](#23-condición-global-del-100-), [consultas](#24-lecciones-consultadas) y [fuentes vigentes](#25-fuentes-vigentes).

## 0. Vocabulario

- universo canónico: las 257 capacidades definidas en
  `product/roadmap.json`.
- completitud: cobertura demostrada de todas las obligaciones aplicables, sin
  reducir el universo para obtener cero.
- censo físico: enumeración reproducible de sujetos y recursos bajo una vista
  estable.
- censo semántico: clasificación de la conducta, consumidores, autoridad,
  invariantes y disposición.
- disposición: decisión trazable sobre reimplementar, conservar, rechazar,
  deduplicar o mantener en estudio.
- matriz requisito-evidencia: relación entre una obligación, su contrato, su
  implementación y la prueba que la acredita.
- hueco: obligación sin implementación, conexión, ejercicio, evidencia o
  decisión suficiente.
- compuerta: condición que debe cumplirse antes de avanzar.
- conjunto de escritura: rutas exclusivas que una tarea puede modificar.
- ciclo de vida (`lifecycle`): secuencia autoritativa de estados de una entidad.
- arrendamiento (`lease`): concesión temporal y exclusiva sobre un recurso.
- cerca (`fence`): ordinal que invalida a un propietario anterior.
- buzón causal (`mailbox`): intercambio durable y ordenado ligado a identidades
  y ejecución exactas.
- recibo (`receipt`): registro estructurado de un intento, resultado o efecto.
- huella criptográfica (`digest` o `hash`): resumen verificable de un sujeto
  inmutable.
- 100 %: estado global definido únicamente por la sección 9 de
  `docs/reconstruccion/ruta_total_100.md`.

## 1. Principio: auditar sin crear otra autoridad

Este capítulo no enumera las 257 capacidades. Las recorre desde el archivo
canónico y genera proyecciones temporales. Una lista copiada aquí quedaría
obsoleta y competiría con la autoridad.

La cadena es:

```text
catálogo canónico
 -> proyección de auditoría fechada
 -> matriz de huecos
 -> filtro de admisión
 -> microtareas
 -> evidencia
 -> estado canónico actualizado por su procedimiento
```

La proyección puede borrarse y regenerarse. No cambia estados ni decisiones.

## 2. Estado actual honesto al redactar este capítulo

La revisión actual presenta un único corte canónico:

| Código de auditoría | Sujeto | Capacidades | Verticales | Contratos | Ejecutables | Planificados | Autoridad |
|---|---|---:|---:|---:|---:|---:|---|
| `base_versionada_257_38_38` | `HEAD:product/roadmap.json` | 257 | 38 | 38 | 22 | 16 | Base guardada en Git y adoptada por la revisión actual. |

El corte conserva 81 capacidades acreditadas, 176 declaradas, 249 decisiones
de aceptación, tres condicionadas y cinco de rechazo. V38 es canónica y
prioritaria, pero su contrato sigue planificado: no puede presentarse como
implementada, conectada, ejercitada ni acreditada.

Estos números son observaciones reproducibles, no una tabla que deba mantenerse
a mano. Se recalculan antes de cada informe y se vinculan al sujeto exacto.

Otros hechos:

- V01–V22 tienen recibos acreditados;
- V23–V38 tienen contratos planificados y no están acreditadas;
- V38 posee solo `ORC-28`; `ORC-15` pertenece a V27 y
  `OPS-16`/`OPS-17` a V32, y V38 conserva esas tres como requisitos no
  propios sin reasignarlas ni acreditarlas;
- V23 no depende de Firecracker, KVM ni microVM, y V38/Firecracker no dependen
  de cerrar V23;
- `product/capabilities.json` describe el corte funcional mínimo y su versión,
  no reemplaza el catálogo total;
- V03 acredita los registros estructurados de trazabilidad, pero no la
  equivalencia total del legado;
- el universo físico expandido contiene 382 sujetos potenciales;
- el registro de raíces tiene `closed: false`;
- existen 27 decisiones físicas de fuente pendientes;
- la compuerta de censo físico está bloqueada («NO-GO», es decir, no apta para
  promoción);
- no existe todavía vista estable, mapeo físico real, recibos completos ni
  salida autorizada para ejecutar el censo.

Por tanto, la afirmación correcta es:

```text
producto parcial con capacidades acreditadas;
inventario histórico lógico avanzado;
censo físico y equivalencia completa pendientes;
Orquesta no está al 100 %.
```

## 3. Autoridades y fuentes

Orden de lectura:

1. `AGENTS.md`;
2. `HEAD:product/roadmap.json` como base versionada adoptada;
3. `product/capabilities.json` y `product/evidence/`;
4. `docs/reconstruccion/ruta_total_100.md`;
5. trazabilidad estructurada de `product/traceability/`;
6. contratos y código;
7. documentación explicativa;
8. legado en solo lectura.

Si dos fuentes autoritativas discrepan, se detiene solo el conjunto afectado y
se resuelve en el catálogo. La auditoría no elige una de forma silenciosa.

## 4. Estados que no deben confundirse

### 4.1 Capacidad

```text
declarada -> implementada -> conectada -> ejercitada -> acreditada
```

Los identificadores persistidos son, respectivamente, `declared`,
`implemented`, `wired`, `exercised` y `accredited`. Solo el último estado
cuenta como terminado.

### 4.2 Elemento inventariado

```text
descubierto -> clasificado -> mapeado -> caracterizado -> revisado
```

Los identificadores persistidos son `discovered`, `classified`, `mapped`,
`characterized` y `reviewed`. Aunque llegue al último, no acredita producto.

### 4.3 Decisión

- aceptación (`accept`): debe acreditarse para la meta total;
- rechazo (`reject`): conserva razón y trazabilidad, sin exigir implementación;
- condicionada (`conditional`): debe resolverse por el criterio previsto;
  abierta bloquea el cierre total.

### 4.4 Contrato

- ejecutable: tiene orden y sujeto verificables;
- planificado: no puede producir un verde de producto.

### 4.5 Disposición histórica

Los registros emplean identificadores estables no traducibles. Su significado
es:

| Significado | Identificador |
|---|---|
| reimplementado y acreditado | `reimplemented_and_accredited` |
| reimplementado, aún no acreditado | `reimplemented_not_accredited` |
| planificado | `planned` |
| candidato que requiere decisión del catálogo | `candidate_requires_roadmap_decision` |
| solo evidencia histórica | `historical_evidence_only` |
| duplicado con origen trazado | `duplicate_with_traced_origin` |
| tercero | `third_party` |
| rechazado con motivo | `rejected_with_reason` |
| hueco sin resolver | `unresolved_gap` |

Solo la primera demuestra equivalencia de una conducta admitida.

## 5. Registro de una ejecución de auditoría

```text
referencia_auditoria:
inicio:
fin:
referencia_operador:
confirmacion_git_fuente:
arbol_fuente:
huella_estado_espacio_trabajo:
huella_catalogo:
huella_capacidades:
huella_indice_evidencias:
huella_trazabilidad:
raices_permitidas:
raices_excluidas_con_motivo:
ordenes:
versiones_herramientas:
presupuestos:
hallazgos:
P0:
P1:
P2:
incertidumbres:
recibos:
revision_independiente:
```

Los datos sensibles y rutas privadas se sustituyen por referencias opacas. La
auditoría registra qué no pudo observar.

## 6. Pseudocódigo general

```text
auditar(revision):
    exigir_arbol_conocido_y_preservar_cambios_ajenos()
    autoridades = cargar_fuentes_canónicas(revision)
    validar_forma_y_huellas(autoridades)

    universo = cargar_257_desde_catalogo()
    capacidades = evaluar_estados_y_dependencias(universo)
    evidencias = validar_recibos_del_mismo_sujeto(capacidades)
    arquitectura = auditar_codigo_y_composicion()
    operacion = auditar_estado_vivo_solo_lectura()
    legado = cargar_censos_sellados_sin_abrir_fuentes_no_autorizadas()

    requisitos = unir_sin_duplicar(
        capacidades,
        contratos,
        superficies,
        inventario_semantico,
        incidencias,
        operacion
    )
    matriz = construir_matriz_requisito_evidencia(requisitos)
    huecos = filas_no_demostradas(matriz)

    para cada hueco:
        decision = aplicar_filtro_de_cinco_preguntas(hueco)
        si decision == admitido:
            proponer_microtareas_causales(hueco)
        si decision == estudio:
            conservar_sin_tarea(hueco)
        si decision == rechazado:
            registrar_razon_y_procedencia(hueco)

    evaluar_compuerta_global_sin_modificarla()
    sellar_informe_y_solicitar_revision_independiente()
```

## 7. Oleada 0: congelar método y alcance

Antes de explorar:

- fijar revisión;
- declarar raíces permitidas;
- declarar exclusiones;
- fijar esquemas y algoritmos de huella;
- separar auditoría lógica, física y operativa;
- presupuestar tiempo, disco, procesos y herramientas;
- consultar lecciones;
- preparar casos adversariales;
- declarar que la observación no escribe producto.

La auditoría se detiene si:

- la fuente cambia sin poder sellarla;
- falta permiso;
- una ruta conduce fuera del alcance;
- el presupuesto se agota;
- se descubre información sensible no prevista;
- se requeriría detener, montar, copiar o borrar sin autorización.

## 8. Oleada 1: catálogo y dependencias

### 8.1 Comprobaciones

- exactamente 257 identificadores únicos;
- exactamente 257 capacidades, 38 verticales y 38 contratos;
- exactamente 22 contratos ejecutables y 16 planificados;
- V38 canónica, planificada y propietaria solo de `ORC-28`;
- `ORC-15` en V27 y `OPS-16`/`OPS-17` en V32;
- vocabularios canónicos;
- decisión válida por capacidad;
- propietario de contexto;
- dependencias existentes y sin ciclos;
- aceptación por capacidad aceptada;
- vertical propietaria;
- alias sin crear capacidad duplicada;
- estado compatible con evidencia;
- ninguna decisión condicionada sin resolución para el cierre total.

### 8.2 Comandos de solo lectura

Base versionada:

```bash
git show HEAD:product/roadmap.json |
  jq '{catalog_size,
       capabilities:(.capability_entries|length),
       verticals:(.verticals|length),
       contracts:(.acceptance_contracts|length),
       executable:([.acceptance_contracts[]
         | select(.status=="executable")]|length),
       planned:([.acceptance_contracts[]
         | select(.status=="planned")]|length)}'
```

Propuesta local:

```bash
jq '{catalog_size,
     capabilities:(.capability_entries|length),
     verticals:(.verticals|length),
     contracts:(.acceptance_contracts|length),
     executable:([.acceptance_contracts[]
       | select(.status=="executable")]|length),
     planned:([.acceptance_contracts[]
       | select(.status=="planned")]|length)}' \
  product/roadmap.json
```

```bash
jq -r '.capability_entries
  | group_by(.status)[]
  | [.[0].status, length]
  | @tsv' product/roadmap.json
```

```bash
jq -r '.capability_entries
  | group_by(.decision)[]
  | [.[0].decision, length]
  | @tsv' product/roadmap.json
```

Prueba canónica:

```bash
go test -mod=vendor -count=1 . \
  -run '^TestProductRoadmapIsExhaustiveAndCausal$'
```

Una consulta `jq` ayuda a inspeccionar; la prueba mantiene invariantes más
ricos.

## 9. Oleada 2: matriz requisito-evidencia

Cada fila contiene:

```text
referencia_requisito:
referencia_fuente:
identificadores_capacidad:
decision:
vertical:
dependencias:
contratos_aceptacion:
referencias_implementacion:
referencias_conexion:
referencias_ejercicio:
referencias_evidencia:
huella_sujeto:
huella_binario_o_imagen:
huella_configuracion_efectiva:
pruebas_negativas:
pruebas_fisicas_reinicio_y_carrera:
referencias_revision:
referencias_disposicion_historica:
estado:
tipo_hueco:
```

No se rellena el estado acreditado (`accredited`) por inferencia. Se valida:

- el recibo existe;
- su contrato coincide;
- la capacidad lo referencia;
- el sujeto es exacto;
- la salida tiene huella;
- la ejecución fue satisfactoria;
- revisiones miraron el mismo sujeto;
- la composición prometida fue la ejercitada;
- la evidencia no precede al candidato;
- el cambio posterior invalida la aprobación.

### 9.1 Tipos de hueco

- decisión;
- dependencia;
- contrato;
- implementación;
- conexión;
- ejercicio;
- evidencia;
- revisión;
- composición física;
- seguridad;
- operación;
- retirada;
- inventario;
- incertidumbre.

El tipo decide la tarea; no todos los huecos son «programar».

## 10. Oleada 3: producto nuevo y Git

### 10.1 Estado Git

```bash
git status --short --branch
git diff --name-status
git diff --check
git ls-files -z
git worktree list --porcelain
```

Se conserva:

- rama y revisión;
- cambios versionados y no versionados;
- espacios de trabajo;
- submódulos o dependencias registradas;
- ficheros ignorados reclamados por operación;
- confirmaciones candidatas y sujetos de evidencia.

`git ls-files` no enumera procesos, temporales, ignorados ni fuentes externas.
No cierra un censo físico.

### 10.2 Arquitectura

Auditar:

- un único `Goal`;
- un escritor de ciclo de vida;
- un planificador;
- una fuente activa por despliegue;
- un registro de comandos;
- un registro de configuración;
- dirección hexagonal;
- cero importaciones, puentes y escrituras dobles al legado;
- composición fina;
- cero paquete sin frontera;
- presupuestos de tamaño, bucles y almacenes.

Pruebas:

- guardas de importaciones;
- raíces productivas;
- variables de entorno;
- registros;
- contratos de repositorio;
- reinicio y repetición.

### 10.3 V31 y varios anfitriones

La base transaccional para concesiones de afinidad está descrita en
[el contrato ejecutable objetivo para V31 multianfitrión del capítulo
03](03_estado_transacciones_y_recuperacion.md#18-contrato-ejecutable-objetivo-para-v31-multianfitrión).

El contrato `AC-V31-POSTGRES-S3-MULTIHOST` está planificado en la base
versionada; no está ejecutado ni acreditado. La auditoría de V31 debe demostrar:

- PostgreSQL y SQLite pasan el mismo contrato de repositorio sin cambiar
  dominio ni crear otro escritor;
- el almacén compatible con S3 y el sistema de archivos pasan el mismo contrato
  de artefactos, incluida idempotencia y aislamiento entre proyectos;
- reclamación, renovación y escritura usan transacción, revisión esperada y
  cerca válida entre anfitriones;
- la hora del servidor de estado decide la vigencia; el reloj de un anfitrión
  no la amplía;
- afinidad y colocación del anfitrión son concesiones recuperables, no otro
  planificador;
- un anfitrión caído pierde escritura y parada cuando otro obtiene una cerca
  mayor;
- la recuperación conserva la misma ejecución y no duplica efectos;
- los espacios de trabajo clonables se reconstruyen por referencias selladas,
  sin compartir escritura local;
- colaboración positiva e intentos cruzados negativos se ejercitan con varios
  usuarios y anfitriones reales;
- caída, partición, entrega tardía y reinicio no dejan reclamaciones dobles ni
  artefactos parcialmente publicados.

Una prueba con dos procesos sobre SQLite local no acredita V31. Se necesita la
composición PostgreSQL, objetos compartidos y anfitriones separados prometida
por su contrato.

## 11. Oleada 4: configuración, identidad y seguridad

### 11.1 Configuración

```bash
rg -n 'os\\.(Getenv|LookupEnv|Environ)' \
  --glob '*.go' --glob '!vendor/**' --glob '!third_party/**'
```

Comprobar:

- toda clave vive en el registro;
- TOML es entrada;
- JSON efectivo es salida redactada;
- nombres alternativos tienen retirada;
- adaptadores reciben estructuras acotadas;
- los hijos reciben lista permitida exacta;
- ningún secreto aparece en la proyección.

### 11.2 Identidad y autorización

- principal autenticado;
- proyecto explícito;
- referencias opacas;
- autorización en aplicación;
- aislamiento negativo entre proyectos;
- colaboración positiva dentro del proyecto;
- agentes como principales de servicio acotados;
- revocación y reinicio.

### 11.3 Secretos y efectos

- referencias en lugar de valores;
- propietario, alcance, versión y caducidad;
- rotación y revocación;
- detectores de fuga;
- intención, aprobación, intento y recibo separados;
- idempotencia;
- efecto desconocido en cuarentena;
- cero publicación, envío o despliegue no autorizado.

## 12. Oleada 5: procesos, disco y operación

### 12.1 Procesos

Lectura que evita mostrar argumentos potencialmente sensibles:

```bash
ps -eo pid=,ppid=,uid=,stat=,comm=
```

Comparar con el registro de propiedad:

- procesos principales;
- agentes e hijos;
- lanzadores;
- atestadores;
- grupos de control;
- zócalos;
- montajes;
- arrendamientos;
- bloqueos.

No se mata por nombre. Un proceso sin identidad se conserva como incidencia.

### 12.2 Disco

```bash
df -h .
df -i .
```

`du` solo se usa sobre raíces explícitas, acotadas y autorizadas. En fuentes
históricas móviles, incluso leer puede alterar metadatos; se respeta la
compuerta de vista estable.

Auditar:

- estado;
- artefactos;
- espacios de trabajo;
- entornos de agentes;
- cachés;
- temporales;
- copias;
- imágenes y descargas;
- políticas de retención.

No se borra nada durante la auditoría.

### 12.3 Continuidad

- copia reciente y verificada;
- restauración real en destino nuevo;
- migraciones;
- actualización;
- retroceso;
- salud y disponibilidad;
- vigilancia cooperativa;
- parada;
- cero recursos propios;
- traspaso.

## 13. Oleada 6: agentes y capacidad

Matriz:

| Requisito | Evidencia mínima |
|---|---|
| Demanda completa | Todos los trabajos listos tienen ejecución durable |
| Capacidad viva | Observación con fuente, instante, caducidad y calidad |
| Reserva | Operación atómica con arrendamiento y cerca |
| Arranque paralelo | Pasos físicos reales sin techo oculto |
| Mensajes | Orden, admisión, aplicación y confirmación separadas |
| Salud | Proceso, canal, latido y progreso estructurados |
| Parada | Identidad exacta, cooperación, escalado y recibo |
| Reinicio | Adopción sin duplicados ni PID reutilizado |
| Conservación | Inventario sellado y cero borrado automático |

Auditar como series distintas:

- demanda lógica exacta 1/16/70/500: todas las ejecuciones existen y
  esperan o progresan sin pérdida;
- pasos físicos exactos 1/5/10/16/20: cada cifra demuestra
  simultaneidad sobre recursos medidos.

Los tamaños 1 y 16 repetidos no fusionan ambas series. Ni 70 ni 500 son pasos
físicos. Su presencia en el contrato no acredita su ejecución.

La lista de perfiles configurada no demuestra elasticidad. Un contador
derivado no demuestra cuota de proveedor.

## 14. Oleada 7: Firecracker

Comprobar por separado:

1. atestador de pruebas con Bubblewrap;
2. candidato Firecracker del atestador;
3. entorno Firecracker de agentes.

Uno no acredita otro.

Matriz del agente:

- compuerta A: núcleo elástico neutral sin KVM ni Firecracker;
- compuerta B: Firecracker activado expresamente, sin sustitución automática y
  con una microVM por agente;
- compuerta C: ola física 1/5/10/16/20 sobre el mismo candidato de A y B;
- acreditación prohibida salvo que A+B+C pasen sobre ese candidato;
- un agente por microVM;
- adaptador detrás de puertos;
- lanzador externo mínimo;
- KVM y activos medidos;
- sistema raíz específico;
- grupo de control;
- CID con arrendamiento y cerca;
- red solo `vsock`;
- representante de salida con lista permitida;
- credenciales efímeras;
- lote de entrada y salida sellado;
- parada, inventario y conservación;
- recuperación;
- cohortes físicas.

Comprobaciones de solo lectura pueden registrar si `/dev/kvm` existe y su modo,
pero no abrirlo ni lanzar una microVM sin prueba autorizada. La presencia de
binarios, imágenes o dispositivos no acredita conexión ni aislamiento.

La auditoría comprueba además que V23 no haya adquirido una dependencia de
Firecracker y que V38 no haya adquirido una dependencia de cierre sobre V23.
La independencia es bidireccional y no reduce las dependencias causales propias
de cada vertical.

## 15. Oleada 8: superficies e internacionalización

Para cada comando:

| Dimensión | HTTP | MCP | Línea de órdenes | Web |
|---|---|---|---|---|
| Esquema | mismo registro | mismo registro | mismo registro | mismo registro |
| Autorización | aplicación | aplicación | aplicación | aplicación |
| Idempotencia | igual | igual | igual | igual |
| Código máquina | estable | estable | estable | estable |
| Recibo | mismo hecho | mismo hecho | mismo hecho | mismo hecho |
| Texto humano | catálogo | catálogo | catálogo | catálogo |

Auditar:

- paridad de comandos y consultas;
- paginación y límites;
- autenticación;
- rescate gobernado;
- accesibilidad;
- navegación por teclado;
- estados vacíos, error y carga;
- español predeterminado y reserva;
- paridad de claves;
- BCP-47;
- plurales, fechas, números, moneda y zona horaria;
- plantillas de instrucciones, avisos y errores.

La interfaz no escribe ciclo de vida directamente.

## 16. Oleada 9: legado y censo

### 16.1 Universos

- producto nuevo;
- copia histórica de consulta;
- otras raíces históricas registradas;
- ramas, paquetes y espacios de trabajo;
- requisitos, incidencias y pruebas;
- estado operativo autorizado;
- consumidores externos.

Cada sujeto conserva origen, revisión, huella, familia, conducta, capacidades,
disposición y revisión independiente.

### 16.2 Estado del censo físico

El universo lógico expandido de 382 sujetos no autoriza abrirlos. La ejecución
real requiere:

- vista estable y cercada;
- mapeo 1:1 y 1:N;
- reobservación de los 382;
- límites y reserva;
- recibos e idempotencia;
- salida privada separada;
- revisión independiente;
- orden expresa del operador.

Mientras falte una compuerta, se mantiene «NO-GO».

### 16.3 Censo semántico

No basta un fichero o una función. Para cada conducta:

- problema y consumidor;
- entrada y salida;
- autoridad y estado;
- permisos, secretos y efectos;
- caída, repetición, concurrencia y reinicio;
- parte útil;
- intento fallido;
- invariante;
- prueba neutral;
- disposición.

Deduplicar por significado conserva todos los orígenes.

## 17. Filtro obligatorio de cinco preguntas

Cada hueco o conducta histórica pasa en orden:

### 17.1 ¿Es útil ahora?

Debe existir consumidor y resultado distinguible. Código accidental, duplicado
sin semántica y soporte de una autoridad antigua se descartan con razón.

### 17.2 ¿Cabe en el producto canónico?

Debe mapear una capacidad existente. Un requisito nuevo necesita decisión de
catálogo antes de convertirse en tarea.

### 17.3 ¿Cumple todas las reglas?

Arquitectura, autoridad, configuración, secretos, identidad, efectos,
internacionalización, seguridad, operación y cero dependencia del legado.
Fallar una regla rechaza el enfoque.

### 17.4 ¿Es viable y operable?

Debe existir camino razonado para implementar, probar, desplegar, observar,
recuperar y retirar, con costes conocidos.

### 17.5 ¿La solución está suficientemente fundada?

Se comprende la causa, existe mecanismo distinto, aceptación ejecutable,
negativos, dependencias y tamaño acotable. Si hay duda sustancial, queda en
estudio sin tarea.

Pseudocódigo:

```text
filtrar(hueco):
    si no util_ahora(hueco):
        devolver rechazado_con_razon
    si no mapea_producto(hueco):
        devolver requiere_decision_catalogo
    si no cumple_todas_las_reglas(hueco):
        devolver rechazado_con_razon
    si no viabilidad_demostrada(hueco):
        devolver estudio_sin_tarea
    si no solucion_fundada(hueco):
        devolver estudio_sin_tarea
    devolver admitido
```

## 18. Convertir huecos viables en microtareas

Una tarea restaura un invariante, no «porta una función antigua».

```text
fuentes:
identificadores de capacidades:
invariante:
autoridad:
contrato:
puertos:
adaptadores:
conjunto de escritura:
dependencias:
prueba focal:
negativos y mutaciones:
carrera y reinicio:
extremo a extremo o física:
presupuesto:
retirada:
```

División normal:

1. decisión de catálogo, si falta;
2. caracterización;
3. contrato neutral;
4. aplicación;
5. persistencia;
6. adaptador;
7. composición;
8. recuperación y negativos;
9. superficie e internacionalización;
10. ejercicio real;
11. revisión y evidencia;
12. retirada.

Se fusionan pasos pequeños inseparables. Se dividen si mezclan autoridades,
superan presupuesto o dificultan diagnóstico.

## 19. Paralelismo seguro

Dos tareas solo progresan en paralelo si:

- sus dependencias ya están satisfechas;
- sus conjuntos de escritura son disjuntos;
- no comparten una migración o composición sin congelar;
- no compiten por el mismo recurso exclusivo;
- sus salidas no dependen entre sí;
- existe responsable y fecha de reserva.

```text
programar_tareas(candidatas):
    disponibles = filtrar_dependencias_acreditadas(candidatas)
    seleccion = []
    rutas_reservadas = conjunto_vacio()

    para cada tarea en ordenar_causalmente(disponibles):
        si interseccion(tarea.rutas, rutas_reservadas) es vacia:
            seleccion.agregar(tarea)
            rutas_reservadas.unir(tarea.rutas)
        si no:
            dejar_en_espera(tarea, causa="conjunto ocupado")

    devolver seleccion
```

Se serializan especialmente:

- `product/roadmap.json`;
- migraciones compartidas;
- composición;
- índices de evidencia;
- catálogos de internacionalización compartidos.

Un agente no amplía rutas por su cuenta. Conserva el hallazgo y solicita una
tarea hija.

## 20. Pruebas del propio auditor

El auditor necesita casos adversariales:

- capacidad duplicada;
- dependencia inexistente o ciclo;
- estado acreditado sin evidencia;
- recibo de otro sujeto;
- contrato planificado presentado como ejecutado;
- fuente nueva no clasificada;
- extensión desconocida;
- ruta con espacios;
- enlace simbólico;
- archivo ignorado reclamado;
- elemento ilegible;
- exclusión sin razón;
- universo reducido para obtener cero;
- tarea histórica sin capacidad;
- incidencia sin prueba de lección;
- proceso propio sin propietario;
- clave de configuración fuera del registro;
- texto público sin catálogo;
- comando presente solo en una superficie;
- Firecracker de atestación presentado como agente;
- capacidad desconocida presentada como cero;
- entorno conservado borrado.
- anfitrión con cerca antigua que consigue escribir, detener o renovar;
- artefacto S3 visible antes de publicar su huella y recibo;
- afinidad de anfitrión convertida en planificador o cola privada.

La contrarrevisión repite el censo lógico desde una copia limpia y compara
huellas, conteos y disposiciones.

## 21. Diagnóstico de discrepancias

| Síntoma | Posible causa | Acción |
|---|---|---|
| El total ya no es 257 | Catálogo alterado o lectura equivocada | Ejecutar prueba canónica y detener cambios de catálogo |
| Hay recibo, pero estado declarado | Evidencia no vinculada o promoción ausente | Validar contrato, sujeto y capacidad |
| Estado acreditado sin recibo válido | Falso verde o evidencia caducada | Abrir incidencia y rectificar por procedimiento |
| Censo da cero huecos demasiado pronto | Universo o exclusiones reducidos | Reconciliar con fuente física y casos adversariales |
| Git limpio, disco con gigabytes | Ignorados, entornos o fuentes externas | Auditar raíces operativas autorizadas |
| Hay más procesos que ejecuciones | Huérfanos o registro incompleto | Cotejar identidad sin matar por patrón |
| HTTP y MCP difieren | Registro duplicado o política en interfaz | Comparar caso de uso y autorización |
| Firecracker aparece «implementado» | Plan o atestador confundidos con agente | Exigir composición y prueba física exactas |
| Traducción parcial | Claves duplicadas o sin paridad | Ejecutar matriz de catálogos y reserva |
| La misma tarea reaparece | Causa no entendida o evidencia no conectada | Aplicar filtro y prueba de lección |

## 22. Traspaso de auditoría

```text
referencia_auditoria:
revisión y huellas:
alcance observado:
alcance no observado:
conteos recalculados:
estado de compuertas:
matriz de huecos:
P0/P1:
incertidumbres:
microtareas admitidas:
elementos en estudio:
elementos rechazados:
conjuntos reservados:
procesos y recursos vivos:
pruebas y recibos:
siguiente dependencia:
acciones prohibidas pendientes de autorización:
```

El traspaso usa referencias, no copia artefactos grandes ni secretos. Otro
auditor debe poder recomputar los conteos.

## 23. Condición global del 100 %

La única condición exacta es que **todos** los predicados de la sección 9 de
`docs/reconstruccion/ruta_total_100.md` pasen sobre una misma revisión y
publicación. Este capítulo no los replica como otra lista autoritativa.

Agrupados para diagnóstico, exigen:

- universo canónico, decisiones y aceptación sin huecos;
- toda capacidad aceptada y necesaria para el corte acreditada;
- P0 y P1 a cero;
- censo y disposiciones históricas completos;
- tareas e incidencias históricas con capacidad o lección;
- un ciclo de vida, escritor, planificador, estado, registro de comandos y
  registro de configuración;
- cero importación, puente, escritor, arrendamiento o recibo pendiente del legado;
- contratos de almacenamiento, identidad, proveedores y despliegue en varios
  anfitriones;
- paridad pública, internacionalización y accesibilidad;
- aplicaciones y consumidores externos acreditados;
- negativos y mutaciones críticas;
- instalación, actualización, retroceso, restauración y parada satisfactorias;
- cero procesos y recursos propios residuales;
- sujeto inmutable de fuente, binario o imagen y configuración efectiva.

Si un solo predicado falla:

```text
estado = producto_parcial
porcentaje_100 = prohibido
```

Puede publicarse una fracción mecánica de capacidades acreditadas entre
aceptadas, indicando los identificadores usados, el denominador y el objetivo
de publicación. No representa esfuerzo restante.

## 24. Lecciones consultadas

Las consultas obligatorias para `GOV-16`, `ORC-23`, `ORC-28`, `OPS-06`,
`OPS-11`, `OPS-16`, `OPS-17`, `OPS-19`, `EVD-04`, `EVD-06` y `EVD-10` no
encontraron coincidencias automáticas para la ruta de este capítulo. Se
registra el hueco.

## 25. Fuentes vigentes

- `AGENTS.md`
- `product/roadmap.json`
- `product/capabilities.json`
- `product/evidence/`
- `product/traceability/README.md`
- `product/traceability/pending_sources.json`
- `product/traceability/legacy_source_roots_2026-07-30.json`
- `product/traceability/legacy_physical_subject_universe_2026-07-30.json`
- `docs/reconstruccion/ruta_total_100.md`
- `docs/reconstruccion/inventario_total_legacy_2026-07-30.md`
- `docs/reconstruccion/estado_compuerta_censo_fisico_2026-07-30.md`
- `docs/reconstruccion/GUIA_MAESTRA_AGENTES_INVENTARIO_Y_RECONSTRUCCION.md`
- `acceptance/README.md`
- `product_roadmap_test.go`

La auditoría produce hechos y trabajo candidato. Solo el catálogo y la
evidencia de una publicación exacta pueden declarar cierre.
