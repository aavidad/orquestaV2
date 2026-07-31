# 10. Operación, observabilidad y continuidad

> **Responsabilidad:** definir la operación segura, observable y recuperable
> de un orquestador en servicio.
>
> **Alcance:** composición, arranque, salud, disponibilidad, diagnóstico,
> parada, copias, restauración, actualización, retención e incidencias.
>
> **No acredita:** este capítulo especifica conducta y pruebas; no acredita la
> operación completa V32 ni la gestión elástica planificada V38.

## Al terminar este capítulo, el lector sabrá…

- distinguir proceso vivo, servicio saludable, disponibilidad y progreso real;
- arrancar y detener la composición en un orden causal y recuperable;
- diagnosticar atascos mediante estado, procesos y evidencia de solo lectura;
- diseñar copias, restauraciones, migraciones y retrocesos verificables;
- conservar recursos e incidencias y entregar un traspaso operativo completo.

## Índice del capítulo

- [Vocabulario](#0-vocabulario-mínimo), [resultado](#1-resultado-operativo), [estado actual](#2-estado-actual-honesto), [composición](#3-composición-segura) y [arranque](#4-arranque).
- [Observabilidad](#5-observabilidad-de-solo-lectura), [diagnóstico](#6-diagnóstico-de-atascos), [vigilante](#7-vigilante-cooperativo) y [parada](#8-parada-ordenada).
- [Cuotas](#9-cuotas-y-capacidad), [copias](#10-copias-verificables), [restauración](#11-restauración) y [actualización](#12-migración-actualización-y-retroceso).
- [Retención](#13-retención-sin-borrado-automático), [limpieza](#14-limpieza-de-recursos-propios), [incidencias](#15-incidencias) y [traspaso](#16-traspaso-operativo).
- [Pruebas](#17-pruebas-de-operación-y-continuidad), [diagnóstico abreviado](#18-diagnóstico-operativo-abreviado), [tareas](#19-plan-de-construcción-en-tareas-pequeñas), [lecciones](#20-lecciones-conservadas) y [fuentes](#21-fuentes-vigentes).

## 0. Vocabulario mínimo

- composición: lugar donde se eligen adaptadores concretos y se conectan con
  los puertos de aplicación.
- salud: capacidad de un componente para responder y conservar sus invariantes.
- disponibilidad: aptitud del servicio para admitir trabajo nuevo de forma
  segura; puede ser falsa aunque el proceso esté vivo.
- observabilidad: capacidad de explicar el estado mediante métricas, registros
  estructurados y trazas sin modificarlo.
- métrica: valor numérico agregado a lo largo del tiempo.
- traza: relación causal de una petición entre componentes.
- alerta: notificación derivada de una condición sostenida y accionable.
- vigilante cooperativo: observador conocido como `watchdog` en `OPS-18`; pide
  diagnóstico o parada gobernada, pero no decide contenido ni mata procesos
  por una señal aislada.
- copia: instantánea inmutable y verificable de una autoridad.
- restauración: creación de un destino inactivo nuevo a partir de una copia.
- migración: transformación versionada y comprobada de esquema o datos.
- retroceso: vuelta a una versión anterior compatible. El identificador
  histórico `rollback` aparece en algunas capacidades y órdenes.
- traspaso: resumen compacto y estructurado que permite continuar una
  incidencia o trabajo sin depender de la memoria de una persona.
- ciclo de vida (`lifecycle`): secuencia autoritativa de estados de una entidad.
- arrendamiento (`lease`): concesión temporal y exclusiva sobre un recurso.
- cerca (`fence`): ordinal que invalida a un propietario anterior.
- buzón causal (`mailbox`): intercambio durable y ordenado ligado a identidades
  y ejecución exactas.
- recibo (`receipt`): registro estructurado de un intento, resultado o efecto.
- huella criptográfica (`digest` o `hash`): resumen verificable de un sujeto
  inmutable.

## 1. Resultado operativo

Una Orquesta profesional debe:

- arrancar solo cuando sus dependencias esenciales son válidas;
- distinguir proceso vivo, servicio saludable y servicio disponible;
- exponer observación de solo lectura;
- diagnosticar atascos por hechos causales;
- controlar cuota y capacidad sin inventarlas;
- detener la admisión antes de apagar componentes;
- obtener puntos de control y cerrar recursos propios;
- conservar datos e incidencias sin borrado automático;
- crear, verificar y restaurar copias;
- actualizar y retroceder con sujeto y recibos;
- recuperarse de una caída sin duplicar trabajo;
- dejar un traspaso suficiente para otro operador.

Métricas, registros y trazas son proyecciones. No cierran, cancelan ni reabren
un `Goal`; tampoco sustituyen la fuente de estado.

## 2. Estado actual honesto

| Área | Estado | Situación comprobada |
|---|---|---|
| Composición única en `cmd/orquesta` e `internal/bootstrap` | ACREDITADO | V01–V22 usan un binario, una fuente de estado activa y los mismos casos de uso. |
| Construcción con liberación inversa al fallar | ACREDITADO | `Build` registra cierres y no publica una composición parcial. |
| Arranque, espera y parada básica | ACREDITADO | `Runtime.Start`, `Wait` y `Shutdown` coordinan servidor, planificador y adaptadores; existen pruebas de reinicio y de proceso de agente recogido. |
| Copia, verificación y restauración SQLite | ACREDITADO | V09 usa copia en línea, contenido inmutable y restauración en destino privado nuevo e inactivo. |
| Migraciones SQLite versionadas y transaccionales | ACREDITADO | `OPS-14` está cubierto por V09, incluida verificación del esquema restaurado. |
| Atestación, revisiones y recibos | ACREDITADO | V17–V19 ofrecen la base de evidencia para cambios operativos. |
| Diagnóstico de configuración interno | PARCIAL | Existe `config.Manager.Doctor`, pero la instalación y diagnóstico operativo completo V32 no están conectados ni acreditados. |
| Parada de todos los agentes elásticos | PARCIAL | Hay parada del adaptador compuesto y controles exactos. `OPS-16` pertenece a V32; V38 exige parada exacta como conducta no propia, sin reasignarla ni acreditarla. |
| Salud y disponibilidad públicas | PENDIENTE | `UI-10` pertenece a V24. No existe todavía la superficie operativa completa. |
| Métricas, registros estructurados y trazas | PENDIENTE | `OPS-19` pertenece a V32. |
| Vigilante cooperativo y perfil ocioso | PENDIENTE | `OPS-18` y su equivalente canónico `ORC-25` pertenecen a V32. |
| Instalación, actualización y retroceso | PENDIENTE | `OPS-15` y `OPS-21` pertenecen a V32. |
| Pruebas de humo reales y nocturnas | PENDIENTE | `OPS-20` pertenece a V32 y es verificación, no otro producto. |
| Retención de registros, entornos, cachés y árboles de trabajo | PENDIENTE | `OPS-17` pertenece a V32; V38 exige conservación como conducta no propia. No hay borrado automático autorizado. |
| Cuota y límites vivos de todos los proveedores | PENDIENTE | `AGT-12` continúa en V25 y `ORC-28` pertenece canónicamente a V38; ninguna de las dos está acreditada. |
| Operación completa | PENDIENTE | `AC-V32-OPERATIONS-TELEMETRY` continúa planificado. |

La copia V09 acreditada es una frontera técnica. Sus conexiones públicas,
automatización de calendario, almacenamiento remoto, actualización y
restauración operativa completa pertenecen a verticales posteriores. No deben
atribuirse a V09.

La revisión actual de `product/roadmap.json` contiene 257 capacidades, 38
verticales y 38 contratos. V1–V22 están acreditadas y V23–V38 continúan
planificadas. V38 es canónica, prioritaria y propietaria únicamente de
`ORC-28`; `ORC-15` sigue en V27 y `OPS-16`/`OPS-17` en V32.

### 2.1 Corte operativo de V38

V38 conserva tres compuertas: A prueba el núcleo elástico neutral sin KVM ni
Firecracker; B conecta Firecracker mediante activación explícita, sin
sustitución automática y con una microVM por agente; C ejecuta la ola física
1/5/10/16/20 sobre el mismo candidato que superó A y B. Solo A+B+C sobre ese
candidato permiten acreditar V38.

La demanda lógica 1/16/70/500 es distinta de la serie física. Ni este capítulo
ni una parada correcta acreditan por sí solos V38.

V23 no depende de Firecracker, KVM ni microVM; V38 y Firecracker tampoco
dependen de cerrar V23. Esta independencia bidireccional impide convertir una
incidencia de una vertical en bloqueo de la otra.

## 3. Composición segura

### 3.1 Regla

La composición conoce marcas y configuración; el núcleo no. El orden debe
permitir liberar en sentido inverso todo recurso abierto si una fase falla.

Secuencia recomendada:

```text
1. cargar registro y TOML
2. resolver credenciales por referencias
3. validar configuración efectiva redactada
4. comprobar identidad y permisos del proceso
5. comprobar almacenamiento y migraciones
6. abrir fuente de estado
7. abrir almacén de artefactos
8. abrir espacios de trabajo
9. abrir proveedores y atestador
10. construir casos de uso
11. construir superficies públicas
12. abrir escucha
13. publicar composición construida
```

Si falla el paso 9, se cierran 8, 7, 6 y los recursos anteriores que se hayan
abierto. No se empieza a servir tráfico.

### 3.2 Pseudocódigo

```text
construir(configuracion):
    cierres = pila_vacia()
    para cada fase en fases_de_composicion:
        recurso = fase.abrir(configuracion_tipificada)
        si recurso.error:
            cerrar_en_orden_inverso(cierres)
            devolver_error_sin_publicar()
        cierres.apilar(recurso.cerrar)

    servicio = ensamblar_recursos()
    servicio.adoptar(cierres)
    devolver servicio
```

Los adaptadores reciben estructuras tipificadas y acotadas, no el registro
global. Los secretos no entran en la configuración efectiva.

## 4. Arranque

### 4.1 Fases

1. validar binario o imagen y configuración;
2. verificar migraciones aplicadas;
3. reconciliar estado durable;
4. reclamar o poner en cuarentena recursos externos conocidos;
5. iniciar planificador sin admitir aún tráfico mutable;
6. iniciar consultas de observación;
7. comprobar dependencias esenciales;
8. publicar disponibilidad;
9. admitir comandos.

Un proceso no está disponible solo porque abrió un puerto.

### 4.2 Estados operativos

```text
construyendo
recuperando
disponible
degradado
drenando
detenido
fallido
```

Son estados de la instancia, no otro ciclo de vida del `Goal`.

### 4.3 Salud frente a disponibilidad

| Consulta | Pregunta | Puede hacer efectos |
|---|---|---|
| Vida | ¿responde el proceso? | No |
| Salud | ¿sus componentes esenciales conservan invariantes? | No |
| Disponibilidad | ¿puede admitir trabajo nuevo de forma segura? | No |
| Diagnóstico | ¿por qué no está disponible? | No |

La respuesta incluye:

- versión;
- huella de binario o imagen;
- revisión de esquema;
- huella de configuración efectiva redactada;
- estado de fuente de datos;
- estado del planificador;
- edad de la última reconciliación;
- dependencias degradadas;
- instante y caducidad de la observación.

No incluye secretos, rutas sensibles, órdenes internas ni texto de agentes.

## 5. Observabilidad de solo lectura

### 5.1 Invariante

Observar no:

- crea trabajo;
- renueva arrendamientos;
- confirma mensajes;
- cambia estado;
- ejecuta herramientas;
- detiene agentes;
- corrige automáticamente una incoherencia.

Si una consulta descubre una incidencia, devuelve hechos y un código estable.
La reparación se solicita mediante otro comando autorizado.

### 5.2 Registros estructurados

Cada registro incluye cuando proceda:

- código de acontecimiento;
- instante y nivel;
- `ProjectRef`, `GoalRef`, generación;
- `WorkItemRef`, `ExecutionRef`, intento;
- referencia de acción, efecto y recibo;
- componente y anfitrión;
- causa encadenada;
- traza;
- clasificación recuperable, definitiva o desconocida.

No incluye:

- secretos;
- plantillas de instrucciones completas;
- fichas de acceso;
- contenido grande;
- variables de entorno;
- datos personales innecesarios.

El texto humano usa el catálogo de internacionalización. El código máquina no
se traduce.

### 5.3 Métricas

Familias mínimas:

| Familia | Ejemplos |
|---|---|
| Admisión | peticiones aceptadas, rechazadas y latencia |
| Objetivos | activos, bloqueados, terminales y antigüedad |
| DAG | trabajos listos, esperando dependencias y en ejecución |
| Acciones | pendientes, reclamadas, caducadas y en cuarentena |
| Agentes | deseados, reales, arrancando, saludables y deteniéndose |
| Capacidad | plazas, reservas, cuota, calidad y edad de observación |
| Efectos | intenciones, aprobaciones, intentos, recibos y desconocidos |
| Evidencia | pruebas, revisiones, mutaciones y acreditaciones |
| Persistencia | latencia, conflictos de revisión, tamaño y migración |
| Continuidad | copias creadas, verificadas, restauradas y antigüedad |
| Recursos | CPU, memoria, procesos, descriptores, disco e inodos |

Las etiquetas tienen cardinalidad limitada. Las referencias individuales se
buscan en trazas y registros, no se convierten indiscriminadamente en
etiquetas de métricas.

### 5.4 Trazas

Una traza une:

```text
petición pública
 -> autorización
 -> comando de aplicación
 -> transacción
 -> acción de salida
 -> reclamación
 -> efecto
 -> recibo
 -> proyección pública
```

La traza ayuda a diagnosticar. La identidad causal durable sigue viviendo en
el estado.

### 5.5 Alertas

Una alerta debe:

- corresponder a una acción operativa conocida;
- requerir duración o repetición cuando una muestra aislada no sea grave;
- incluir alcance y referencias;
- evitar texto sensible;
- distinguir degradación de caída;
- cerrarse por evidencia, no por silencio.

Alertas iniciales:

- copia no verificada dentro del plazo;
- disco o inodos por debajo del margen;
- bandeja transaccional envejecida;
- resultado de efecto desconocido;
- arrendamientos caducados no reconciliados;
- servicio no disponible;
- cuota agotada o desconocida demasiado tiempo;
- proceso propio huérfano;
- conservación pendiente que supera la política;
- fallo de actualización o restauración.

## 6. Diagnóstico de atascos

### 6.1 Orden causal

No se empieza reiniciando. Se sigue la cadena:

```text
Goal
 -> WorkItem y dependencias
 -> Execution
 -> Action
 -> reclamación, arrendamiento y cerca
 -> capacidad y presupuesto
 -> agente o herramienta
 -> sesión y buzón
 -> artefactos y pruebas
 -> revisiones y decisión
 -> cierre
```

Reclamación (`claim`), arrendamiento (`lease`) y cerca (`fence`) son los
conceptos que aparecen con esos identificadores en el modelo actual.

### 6.2 Pseudocódigo

```text
diagnosticar(referencia_objetivo):
    objetivo = consultar_objetivo(referencia_objetivo)
    trabajos = consultar_trabajos_y_dependencias(referencia_objetivo)
    para cada trabajo no_terminal:
        ejecucion = consultar_ejecucion_vigente(trabajo)
        accion = consultar_accion(ejecucion)
        reclamacion = consultar_arrendamiento_y_cerca(accion)
        capacidad = consultar_observacion_y_reserva(ejecucion)
        externo = observar_identidad_externa(ejecucion)
        mensajes = consultar_buzon_sin_confirmar()
        evidencia = consultar_artefactos_pruebas_y_revisiones()
        explicar_primera_frontera_que_no_converge()
```

### 6.3 Tabla de síntomas

| Síntoma | Primera comprobación | No hacer |
|---|---|---|
| Objetivo activo sin acciones | Dependencias, generación y conjunto listo | Crear una cola paralela |
| Acción pendiente antigua | Arrendamiento, cerca y reclamador | Marcarla completada a mano |
| Agente visible sin ejecución | Vínculo causal e identidad externa | Adoptarlo por nombre |
| Ejecución sin progreso | Salud, último mensaje y herramienta activa | Matar por silencio |
| Pruebas aprobadas sin cierre | Revisiones, Consejo y sujeto exacto | Forzar estado terminal |
| Cuota aparentemente agotada | Fuente estructurada y caducidad | Buscar palabras en registros |
| Copia reciente pero inválida | Verificación, huella y esquema | Sobrescribir la anterior |
| Métrica contradice estado | Definición y fuente de la métrica | Corregir el estado desde la métrica |

## 7. Vigilante cooperativo

El vigilante observa recursos y progreso; no interpreta el contenido del
trabajo. Sus entradas:

- CPU y duración del uso alto;
- memoria, procesos y disco;
- trabajos y acciones activos;
- agentes registrados;
- trabajo asíncrono;
- parada en curso;
- último progreso estructurado;
- calidad y edad de la observación.

Política:

```text
evaluar_vigilante(observacion, politica):
    si politica.deshabilitada:
        devolver sin_accion
    si observacion.caducada:
        devolver pedir_nueva_observacion
    si recursos_dentro_de_limites:
        devolver saludable
    si existe_causa_operativa_o_progreso_reciente:
        devolver observar_sin_interrumpir
    si no_se_cumplio_ventana_sostenida:
        devolver observar
    devolver solicitar_diagnostico_y_parada_cooperativa
```

La salida es una solicitud gobernada, no una señal directa al proceso. Antes de
parar:

1. confirmar identidad;
2. impedir nuevas admisiones;
3. solicitar punto de control;
4. esperar plazo;
5. detener cooperativamente;
6. escalar solo con autorización y evidencia;
7. conservar.

Una CPU alta con trabajo real no es un bloqueo. Una CPU baja tampoco demuestra
que el sistema esté sano.

## 8. Parada ordenada

### 8.1 Secuencia objetivo

1. autenticar y autorizar la petición;
2. persistir el estado operativo `drenando` y cerrar la admisión de mutaciones
   nuevas;
3. mantener consultas, estado y adaptadores vivos mientras terminan o quedan
   clasificadas todas las órdenes y entregas de bandeja ya admitidas;
4. congelar nuevas asignaciones sin descartar acciones durables;
5. inventariar trabajo activo y pedir puntos de control;
6. detener cada agente exacto de forma cooperativa y escalar solo con
   autorización;
7. sincronizar cambios, artefactos, diarios y salidas;
8. inventariar y sellar cada entorno, y persistir su conservación pendiente de
   revisión;
9. detener planificador, despachadores y servidor cuando ya no producen
   trabajo;
10. cerrar representantes, proveedores, credenciales, espacios de trabajo,
    artefactos y demás almacenes no autoritativos, recogiendo sus recibos;
11. confirmar que la bandeja admitida quedó drenada o clasificada y persistir
    el punto de recuperación, recibo interno y traspaso mientras la fuente de
    estado sigue abierta;
12. cerrar la fuente de estado en último lugar;
13. después de la salida del proceso, hacer que un supervisor externo compruebe
    cero procesos, zócalos, montajes, grupos y bloqueos propios.

| Código | Invariante |
|---|---|
| `apagado_estado_al_final` | La fuente de estado es el último componente interno que se cierra; la comprobación de residuos ocurre después y desde fuera del proceso. |

El recibo interno demuestra el punto alcanzado antes de cerrar el estado. La
comprobación externa posterior produce otra observación o recibo fuera del
sujeto ya cerrado; no se inventa una verificación de cero residuos desde un
proceso que todavía sigue vivo.

### 8.2 Estado actual y deuda

La composición actual cancela el contexto, apaga servidor y adaptador de
agente, espera al planificador y cierra recursos. Existe prueba que comprueba
la recogida de un proceso propio.

Todavía faltan para `OPS-16` en V32. V38 reutiliza esta conducta como requisito
no propio, sin poseer ni acreditar `OPS-16`:

- inventario completo de todos los agentes dinámicos;
- punto de control por agente;
- parada individual cooperativa y forzada;
- sellado antes de desmontar;
- conservación pendiente de revisión (`preserved_pending_review`);
- verificación separada de demanda lógica y pasos físicos;
- recibo externo de cero residuos después de cerrar el proceso.

### 8.3 Tiempo agotado

Si vence el plazo:

- no se declara parada limpia;
- se conserva la lista de trabajo activo;
- se registra qué componente no cerró;
- se mantienen referencias saneadas;
- se aplica parada forzada solo a recursos propios exactos y autorizados;
- se persisten punto y traspaso mientras el estado siga abierto;
- se entrega al supervisor externo la comprobación pendiente.

## 9. Cuotas y capacidad

Una observación de capacidad contiene:

- proveedor y perfil;
- plazas totales, ocupadas, reservadas y disponibles;
- cuota y límites medibles;
- CPU, memoria, disco o procesos del anfitrión;
- instante, caducidad y fuente;
- calidad: disponible, agotada, desconocida, caducada o inaccesible;
- recibo.

Reglas:

- desconocida no significa cero ni ilimitada;
- la disponibilidad se deriva, no se edita;
- las reservas son atómicas;
- la aplicación decide; el proveedor observa;
- la falta temporal conserva la misma ejecución;
- el límite es visible y justificable;
- una métrica de cuota no sustituye su recibo de proveedor.

La interfaz de operación muestra demanda completa, capacidad admitida y espera.
Así se distingue atasco de contrapresión correcta.

## 10. Copias verificables

### 10.1 Qué existe

El puerto `StateRecovery` ofrece:

- `CreateBackup`;
- `VerifyBackup`;
- `RestoreBackup`.

Los identificadores se conservan por ser nombres del código; significan crear,
verificar y restaurar una copia.

El adaptador SQLite actual:

- usa la interfaz de copia en línea del motor;
- no copia directamente archivos WAL o SHM;
- publica por contenido;
- guarda manifiesto canónico;
- verifica huella física y huella lógica;
- comprueba integridad, claves externas, migraciones y agregados;
- excluye secretos y cuerpos de artefactos;
- bloquea rutas inseguras y raíces solapadas;
- restaura solo en un destino privado nuevo e inactivo;
- limpia etapas parciales propias ante fallo.

### 10.2 Conjunto de continuidad

La copia de estado no contiene todo el servicio. Deben existir políticas
coordinadas para:

- base de estado;
- almacén de artefactos;
- credenciales, con su mecanismo seguro propio;
- configuración explícita;
- activos de despliegue;
- índices reconstruibles;
- recibos externos.

Un manifiesto de continuidad vincula sus referencias y tiempos sin introducir
otra fuente de verdad.

### 10.3 Crear y verificar

```text
crear_copia():
    comprobar_presupuesto_y_destino_privado()
    recibo = estado.crear_copia_en_linea()
    verificacion = estado.verificar_copia(recibo.ref)
    exigir_misma_huella_y_esquema(recibo, verificacion)
    registrar_manifiesto_de_continuidad()
    replicar_segun_politica_sin_borrar_origen()
    devolver recibo
```

Una copia no verificada no cumple el objetivo de recuperación.

### 10.4 Frecuencia

La frecuencia se deriva de:

- pérdida de datos máxima tolerable;
- tiempo máximo de recuperación;
- volumen y ritmo de cambios;
- coste;
- requisitos legales;
- capacidad de verificar y restaurar.

No se copia con más frecuencia de la que se puede comprobar y conservar de
forma segura.

## 11. Restauración

### 11.1 Regla

Nunca se sobrescribe la fuente activa. La restauración crea un destino nuevo,
lo valida y solo después se estudia una promoción separada.

```text
restaurar(copia_ref, destino_nuevo):
    exigir_instancia_destino_inactiva()
    verificar_copia(copia_ref)
    recibo = crear_destino_desde_copia(copia_ref, destino_nuevo)
    verificar_integridad_esquema_agregados_y_referencias(destino_nuevo)
    huella_antes = calcular_huella(destino_nuevo)
    abrir_copia_desechable_solo_lectura(destino_nuevo)
    simular_recuperacion(
        adaptadores_de_efecto = denegar_todos,
        credenciales = ninguna,
        despachadores = apagados,
        escritura_de_simulacion = memoria_separada
    )
    ejecutar_pruebas_de_consulta_sin_escrituras()
    exigir_huella_igual(huella_antes, calcular_huella(destino_nuevo))
    devolver recibo_de_destino_no_promovido
```

La repetición durante una restauración es solo una simulación. No reclama
acciones, no renueva arrendamientos, no escribe en la copia desechable y no
invoca proveedores ni herramientas. Cualquier estado auxiliar de la simulación
vive en memoria o en un almacén separado que se descarta sin tocar el destino
restaurado.

Promover el destino requiere:

- autorización;
- ventana;
- configuración exacta;
- copia final del origen;
- comprobación de consumidores;
- conmutación atómica o controlada;
- observación;
- posibilidad de retroceso.

### 11.2 Pruebas periódicas

Una operación profesional restaura periódicamente en un entorno aislado y
mide:

- tiempo;
- pérdida real;
- integridad;
- migraciones;
- pendientes reclamables;
- terminales no repetidos;
- cero efectos externos y cero escrituras sobre la copia desechable;
- igualdad de la huella antes y después de la simulación;
- disponibilidad de artefactos y credenciales por sus políticas;
- limpieza del entorno de prueba.

## 12. Migración, actualización y retroceso

### 12.1 Migraciones

Cada migración:

- tiene versión y huella;
- avanza en una sola dirección;
- se ejecuta transaccionalmente cuando el motor lo permite;
- valida precondiciones;
- conserva datos y recibos;
- falla sin dejar autoridad doble;
- tiene prueba desde cada versión soportada;
- tiene copia verificada previa.

Una migración de configuración usa lectura única y transformación declarativa.
No deja dos formatos como entradas permanentes.

### 12.2 Actualización

Secuencia objetivo:

1. identificar versión actual y candidata;
2. sellar binario o imagen y configuración;
3. comprobar compatibilidad de esquema;
4. ejecutar diagnóstico previo;
5. crear y verificar copia;
6. drenar;
7. aplicar migración necesaria;
8. activar la versión candidata mediante subcomando `orquesta` o servicio
   externo del sistema operativo;
9. comprobar vida, salud y disponibilidad;
10. ejecutar prueba de humo real;
11. observar durante ventana definida;
12. emitir recibo.

No se crea un «guardián» residente en Go con otro ciclo de vida. La capacidad
`OPS-21` exige un subcomando del mismo binario o un servicio externo.

### 12.3 Retroceso

El retroceso de binario solo es seguro si el esquema y datos siguen siendo
compatibles. Si no:

1. drenar la versión nueva;
2. conservar su estado e incidencia;
3. restaurar la copia en otro destino;
4. verificar;
5. activar versión anterior contra ese destino;
6. comprobar consultas y no repetición;
7. emitir recibo.

Nunca se ejecutan migraciones inversas improvisadas sobre el estado activo.

## 13. Retención sin borrado automático

### 13.1 Clasificación

Cada objeto se clasifica como:

- estado autoritativo;
- evidencia;
- artefacto referenciado;
- entorno conservado;
- registro operativo;
- caché reconstruible;
- temporal propio;
- desconocido.

«Desconocido» se conserva.

### 13.2 Flujo de retirada

```text
evaluar_retencion(objeto):
    identidad = resolver_propietario_y_ejecucion(objeto)
    referencias = buscar_referencias_durables(objeto)
    procesos = verificar_que_no_hay_uso_vivo(objeto)
    politica = resolver_plazo_y_clasificacion(objeto)

    si identidad_falta o referencias_desconocidas o procesos_vivos:
        devolver conservar_bloqueado
    si plazo_no_cumplido:
        devolver conservar
    devolver candidato_a_retirada
```

Ser candidato no borra. La retirada:

- se previsualiza;
- exige confirmación exacta;
- usa rutas validadas;
- afecta solo recursos propios;
- es idempotente;
- produce recibo;
- nunca sigue enlaces;
- no acepta una raíz amplia ni una variable sin resolver.

Los entornos de agente terminados permanecen conservados pendientes de
revisión (`preserved_pending_review`) hasta que se estudien.

## 14. Limpieza de recursos propios

Al terminar una prueba, actualización o incidencia se inventarían:

- procesos e hijos;
- microVM;
- grupos de control;
- zócalos;
- puertos y escuchas;
- montajes;
- temporales;
- árboles de trabajo;
- cachés creadas;
- credenciales efímeras;
- arrendamientos;
- bloqueos.

Solo se retiran los que la operación creó y puede identificar exactamente. Si
un recurso puede contener evidencia o no tiene propietario demostrable, se
conserva y se registra.

Los temporales se crean en directorios privados y acotados. La limpieza nunca
apunta a la raíz del repositorio, directorio personal o ruta obtenida de texto
no validado.

## 15. Incidencias

### 15.1 Registro mínimo

```text
identificador:
fecha e instante:
severidad:
síntoma:
impacto:
alcance:
referencias causales:
versión y configuración:
hechos observados:
acciones realizadas:
datos conservados:
causa arquitectónica:
invariante roto:
capacidad:
prueba de reproducción:
corrección:
prueba de lección:
evidencia de cierre:
riesgos restantes:
```

Un P0 impide uso seguro o causa pérdida, fuga o efecto grave. Un P1 impide una
función esencial o deja riesgo alto. El cierre total exige P0 y P1 a cero.

### 15.2 Respuesta

1. limitar efectos sin destruir evidencia;
2. conservar estado, registros y entorno;
3. identificar sujeto y alcance;
4. reproducir en aislamiento;
5. relacionar con capacidad e invariante;
6. corregir con conjunto de escritura pequeño;
7. añadir prueba que falle con la causa;
8. revisar de forma independiente;
9. acreditar sobre sujeto nuevo;
10. documentar retirada o riesgo.

No se eleva un umbral, borra un proceso o reinicia indefinidamente para ocultar
la causa.

## 16. Traspaso operativo

Un traspaso compacto contiene:

- objetivo actual y prioridad;
- versión y estado Git;
- estado canónico de capacidades;
- procesos y recursos vivos;
- acciones y efectos pendientes;
- última copia verificada;
- cambios no integrados;
- pruebas ejecutadas con resultado;
- incidencias y bloqueos;
- decisiones y supuestos;
- conjuntos de escritura reservados;
- siguiente dependencia causal;
- órdenes seguras de reanudación.

No contiene secretos, transcripciones extensas ni resultados grandes. Usa
referencias y huellas.

```text
crear_traspaso():
    hechos = leer_estado_y_recursos_sin_mutar()
    refs = compactar_referencias_causales(hechos)
    devolver documento(
        objetivo,
        version,
        capacidades,
        recursos_vivos,
        pruebas,
        bloqueos,
        siguiente_accion,
        refs
    )
```

Un traspaso no marca tareas terminadas ni sustituye los registros estructurados.

## 17. Pruebas de operación y continuidad

### 17.1 Composición y arranque

- fallo en cada fase libera recursos anteriores;
- configuración o secreto inválidos impiden publicar disponibilidad;
- migración pendiente impide admisión;
- dos arranques no comparten una fuente exclusiva;
- reinicio adopta solo recursos de identidad coincidente;
- observación no produce escrituras.

### 17.2 Salud y observabilidad

- vida puede ser cierta con disponibilidad falsa;
- dependencia degradada aparece con causa;
- métricas no contienen secretos;
- cardinalidad queda limitada;
- traza conserva causalidad;
- consulta repetida no cambia revisiones;
- dato caducado no se presenta como actual.

### 17.3 Vigilante

- una muestra alta no detiene;
- progreso reciente evita interrupción;
- causa operativa explica consumo;
- observación caducada pide renovación;
- inactividad sostenida solicita parada cooperativa;
- la solicitud sin autorización no fuerza proceso;
- silencio textual no equivale a bloqueo.

### 17.4 Copia y restauración

- escritura concurrente queda totalmente antes o después de la copia;
- copia corrupta o truncada falla;
- huella, esquema o agregado alterados fallan;
- secretos y cuerpos de artefactos no aparecen en copia de estado;
- destino existente, activo, enlazado o inseguro falla;
- caída en cada frontera no publica parcial;
- la simulación no reclama, no ejecuta efectos y no obtiene credenciales;
- la huella de la copia desechable no cambia durante la simulación;
- toda escritura accidental contra la copia desechable falla;
- restauración periódica cumple tiempos.

### 17.5 Actualización y retroceso

- versión o huella falsa se rechaza;
- diagnóstico previo no muta;
- fallo de migración conserva versión anterior;
- versión nueva no disponible activa retroceso gobernado;
- esquema incompatible impide retroceso directo;
- copia restaurada en nuevo destino permite retorno;
- cada efecto tiene aprobación, idempotencia y recibo.

### 17.6 Parada y retención

- nuevas mutaciones se rechazan al drenar;
- consultas siguen disponibles mientras sea seguro;
- trabajo activo produce punto de control;
- una orden admitida termina o queda clasificada antes de cerrar sus
  adaptadores;
- el entorno se sincroniza, inventaría, sella y persiste antes de desmontar;
- proveedores y almacenes cierran antes que la fuente de estado;
- el punto y el recibo internos se persisten antes de cerrar el estado;
- la fuente de estado se cierra la última;
- la comprobación de cero residuos procede de un supervisor externo posterior;
- cierre repetido es idempotente;
- plazo agotado conserva referencias;
- cero procesos propios residuales;
- desconocido nunca se borra;
- retirada sin confirmación falla;
- un recurso ajeno nunca se toca.

### 17.7 Extremo a extremo operativo

La aceptación V32 debe demostrar en composición real:

- instalación;
- diagnóstico;
- actualización;
- retroceso;
- restauración;
- periodo ocioso;
- vigilancia cooperativa;
- métricas y trazas;
- vida, salud y disponibilidad;
- parada;
- suplantación rechazada;
- cero residuos.

La aceptación planificada V38 añade parada y conservación a dos series
separadas: demanda lógica exacta 1/16/70/500 y pasos físicos exactos
1/5/10/16/20. Estas conductas no reasignan ni acreditan `OPS-16` o `OPS-17`,
ni permiten acreditar V38 sin completar A+B+C sobre el mismo candidato.

## 18. Diagnóstico operativo abreviado

Antes de intervenir:

```bash
git status --short --branch
df -h .
```

Después, mediante consultas públicas o herramientas de solo lectura:

1. versión y configuración efectiva redactada;
2. vida, salud y disponibilidad;
3. estado de fuente y esquema;
4. demanda, acciones y arrendamientos;
5. capacidad y cuota;
6. procesos propios y agentes;
7. bandeja y efectos desconocidos;
8. copia verificada más reciente;
9. disco, inodos, memoria y descriptores;
10. incidencias abiertas.

No se usan órdenes destructivas como diagnóstico. Si el servicio no responde,
se inspeccionan sus recursos exactos y estado durable antes de reiniciar.

## 19. Plan de construcción en tareas pequeñas

1. definir puerto neutral de observación operativa y adaptador falso;
2. modelar vida, salud, disponibilidad, degradación y drenaje;
3. exponer consultas de solo lectura por el registro de comandos;
4. añadir registros estructurados con saneamiento;
5. añadir métricas de cardinalidad acotada;
6. añadir propagación de trazas;
7. crear alertas accionables;
8. implementar diagnóstico causal;
9. implementar vigilante cooperativo;
10. completar parada y recibo de cero residuos;
11. conectar copia, verificación y restauración a operación autorizada;
12. coordinar copia de artefactos, credenciales y configuración;
13. implementar instalación y diagnóstico previo;
14. implementar actualización por el mismo binario o servicio externo;
15. implementar retroceso compatible y por restauración;
16. implementar clasificación y previsualización de retención;
17. implementar retirada separada y recibos;
18. ejecutar pruebas periódicas de continuidad;
19. ejecutar aceptación V32;
20. completar las conductas operativas del contrato V38 sin atribuirle
    `ORC-15`, `OPS-16` ni `OPS-17`, y conservar la separación A/B/C.

Cada tarea declara capacidad, autoridad, contrato, conjunto de escritura,
pruebas, presupuesto y siguiente dependencia.

## 20. Lecciones conservadas

Las consultas obligatorias para `EVD-15`, `OPS-14`, `OPS-15`, `OPS-16`,
`OPS-17`, `OPS-18`, `OPS-19`, `OPS-21`, `ORC-25`, `AGT-12` y `CTX-02` no
encontraron coincidencias automáticas para la ruta de este capítulo. El hueco
queda documentado.

La lectura histórica en
`/home/alberto/Trabajo/orquestaV2-legacy-consulta`, siempre en solo lectura,
aporta estas lecciones:

- salud, disponibilidad y parada deben derivar de una proyección coherente;
- abrir un puerto no acredita disponibilidad estable;
- el vigilante debe considerar ventana sostenida, causa operativa y progreso;
- la parada congela nuevas asignaciones antes de coordinar trabajo activo;
- si el cierre vence, el traspaso conserva referencias del trabajo;
- un entorno sin registro de identidad no se puede borrar con seguridad;
- retención usa previsualización, confirmación y recibo;
- las métricas deben medir semántica, no coincidencias nominales;
- ninguna observación secundaria escribe el estado autoritativo.

Fuentes históricas principales:

- `docs/incidencias/incidencia_orquesta_goal_first_lifecycle_status_shutdown_readiness_2026-07-02.md`
- `docs/incidencias/incidencia_orquesta_shutdown_active_work_handoff_2026-07-02.md`
- `docs/incidencias/incidencia_orquesta_retencion_runtime_codex_waves_2026-07-10.md`
- `docs/incidencias/incidencia_orquesta_limpieza_config_metricas_falsas_2026-07-11.md`
- `modulos/orquesta-server/self_watchdog_policy_v0.go`
- `modulos/orquesta-server/shutdown_freeze_v0.go`
- `modulos/orquesta-server/supervisor_metrics_v0.go`

## 21. Fuentes vigentes

- `AGENTS.md`
- `product/roadmap.json`
- `product/capabilities.json`
- `product/evidence/v09_recovery_backup.json`
- `docs/reconstruccion/ruta_total_100.md`
- `docs/reconstruccion/LEEME_AGENTE_ORQUESTAV2.md`
- `docs/reconstruccion/handoff_continuacion_agente_2026-07-30.md`
- `internal/application/recovery.go`
- `internal/adapters/state/sqlite/recovery.go`
- `internal/adapters/state/sqlite/recovery_backup.go`
- `internal/adapters/state/sqlite/recovery_verify.go`
- `internal/adapters/state/sqlite/recovery_restore.go`
- `internal/bootstrap/runtime.go`
- `internal/bootstrap/shutdown_e2e_test.go`

Este capítulo define el objetivo operativo y la ruta de construcción. Solo los
contratos ejecutados y sus recibos pueden promover V32 o V38 después de que
esta última sea adoptada.
