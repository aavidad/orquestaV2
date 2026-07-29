# Runbook local: varias identidades con `local_token`

## Alcance

Este corte permite que un solo servidor local autentique varias identidades
humanas lógicas con credenciales distintas. No demuestra por sí solo que las
credenciales estén custodiadas por personas físicas diferentes: si comparten
usuario del sistema operativo, quien controle ese usuario puede leer ambas.
La garantía fuerte de dos personas requiere OIDC o custodios/usuarios del
sistema separados. No cambia el contrato anterior:

- `identity.local_actor` y `identity.local_token_path` siguen definiendo al
  propietario local primario;
- si `identity.local_principals_manifest_path` está vacío, el comportamiento es
  el mismo que antes;
- el manifest solo enlaza credencial, `principal_ref` y `actor_ref`; no contiene
  roles y nunca crea ni modifica membresías;
- OIDC continúa siendo un proveedor separado y no consume este manifest.

La separación crítica sigue siendo la del dominio: quien propone un efecto no
puede aprobar su propio efecto crítico. El aprobador independiente necesita hoy
`project_owner`; `reviewer` u `operator` no bastan para la guarda de autoridad
crítica. Queda como deuda explícita diseñar, acreditar y migrar a un rol más
estrecho para aprobación crítica, y cubrir la misma operación con dos humanos
OIDC.

## Preparar secretos y manifest

Usar una ruta privada propiedad del usuario efectivo que ejecuta Orquesta, modo
exacto `0700` y sin symlinks. El ejemplo usa una variable para no asumir una
ruta de sistema ni requerir `sudo`. El manifest debe ser un fichero regular
privado `0600`, del mismo usuario y con un solo enlace físico:

```bash
umask 077
ORQUESTA_LOCAL_SECRETS=/ruta/privada/orquesta/secrets
install -d -m 700 "$ORQUESTA_LOCAL_SECRETS"
install -m 600 /dev/null "$ORQUESTA_LOCAL_SECRETS/local-principals.json"
```

Contenido mínimo:

```json
{
  "schema_version": 1,
  "document_type": "orquesta.local_token_principals",
  "principals": [
    {
      "principal_ref": "actor:local-approver",
      "actor_ref": "actor:local-approver",
      "token_path": "local-approver.token"
    }
  ]
}
```

Reglas fail-closed:

- `principal_ref` debe ser exactamente igual a `actor_ref`, de tipo humano;
- cada ref, ruta y credencial debe ser única;
- `token_path` es relativo al directorio del manifest y no admite traversal,
  ruta absoluta, symlink ni hardlink;
- el documento completo, su número de entradas y cada token están acotados por
  la configuración canónica;
- primero se valida todo el manifest; una entrada estructural inválida no crea
  ninguno de sus tokens;
- el arranque solo lee tokens secundarios ya provisionados: si falta uno,
  falla sin crear credenciales ni dejar un conjunto parcial;
- nunca se genera una membresía desde el manifest.

Provisionar cada token antes del primer arranque sin mostrarlo ni copiarlo al
TOML. Este patrón publica un fichero completo y no sobrescribe uno existente;
si se interrumpe antes de retirar el enlace temporal, el token queda con dos
enlaces y Orquesta lo rechaza:

```bash
set -euo pipefail
token_path="$ORQUESTA_LOCAL_SECRETS/local-approver.token"
token_tmp="$(mktemp "$ORQUESTA_LOCAL_SECRETS/.local-approver.XXXXXX")"
trap 'rm -f -- "$token_tmp"' EXIT
head -c 32 /dev/urandom | base64 -w0 | tr '/+' '_-' | tr -d '=' >"$token_tmp"
test "$(wc -c <"$token_tmp")" -eq 43
chmod 0600 "$token_tmp"
ln -- "$token_tmp" "$token_path"
rm -- "$token_tmp"
trap - EXIT
```

Un token existente inválido bloquea el arranque; no se sustituye
silenciosamente.

## Configuración canónica

Configurar únicamente las claves del registro canónico:

```toml
[identity]
provider = "local_token"
local_actor = "actor:local-owner"
local_token_path = "/ruta/privada/orquesta/secrets/local-owner.token"
local_principals_manifest_path = "/ruta/privada/orquesta/secrets/local-principals.json"
local_principals_manifest_max_bytes = 65536
local_principals_manifest_max_entries = 32
```

Las dos últimas claves tienen límites canónicos en `config/registry.json`.
`effective_config.json` puede contener las rutas, pero nunca el material de las
credenciales. Cambiar el manifest requiere reiniciar el servidor para componer
de nuevo el proveedor.

## Conceder autoridad de forma explícita

Tras el primer arranque, el principal secundario ya puede autenticarse, pero
recibe `forbidden` hasta que el propietario le conceda una membresía durable. No
editar SQLite.

Ejemplo de concesión deliberada de `project_owner`:

```bash
orquesta command \
  --url http://127.0.0.1:8080 \
  --credential-file /ruta/privada/orquesta/secrets/local-owner.token \
  --max-credential-bytes 4096 \
  --max-response-bytes 65536 \
  --timeout 5s \
  --request-ref request:grant-local-approver \
  --project-ref project:default \
  --payload '{"target_principal_ref":"actor:local-approver","target_actor_ref":"actor:local-approver","target_kind":"human","target_method":"local_token","role":"project_owner","expected_revision":0}' \
  -- projects memberships grant
```

Conservar el `audit_ref` y el receipt devueltos. Un replay con el mismo
`request-ref` y payload debe devolver exactamente el mismo resultado, también
después de reiniciar.

Para aprobar un intent crítico, el principal lógico secundario usa su propio
fichero:

```bash
orquesta command \
  --url http://127.0.0.1:8080 \
  --credential-file /ruta/privada/orquesta/secrets/local-approver.token \
  --max-credential-bytes 4096 \
  --max-response-bytes 65536 \
  --timeout 5s \
  --request-ref request:approve-critical-intent \
  --project-ref project:default \
  --payload '{"goal_ref":"goal:...","intent_ref":"effect-intent:...","expected_intent_digest":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","decision":"approved","reason":"independent critical approval"}' \
  -- effects decide
```

El propietario que propuso el efecto debe seguir recibiendo `forbidden` al
intentar autoaprobarlo. Un token de ejecución con namespace reservado
`orqex1.` tampoco autentica como humano.

## Revocar

La revocación de autoridad es explícita y durable:

```bash
orquesta command \
  --url http://127.0.0.1:8080 \
  --credential-file /ruta/privada/orquesta/secrets/local-owner.token \
  --max-credential-bytes 4096 \
  --max-response-bytes 65536 \
  --timeout 5s \
  --request-ref request:revoke-local-approver \
  --project-ref project:default \
  --payload '{"target_principal_ref":"actor:local-approver","expected_revision":1}' \
  -- projects memberships revoke
```

Quitar una entrada del manifest y reiniciar corta su autenticación, pero no
reescribe historia ni sustituye la revocación de la membresía. Revocar primero,
guardar el receipt, retirar después la entrada y reiniciar.

## Evidencia automatizada

- `internal/adapters/auth/localtoken/localtoken_test.go` cubre permisos,
  owner, symlinks, hardlinks, traversal, duplicados, límites, identidad
  explícita, credenciales distintas y validación completa antes de crear
  tokens.
- `internal/bootstrap/localtoken_multi_e2e_test.go` usa dos CLIs reales contra
  un servidor y acredita denegación previa, grant con receipt, rechazo de
  autoaprobación, decisión por un principal lógico distinto, ejecución crítica
  y replay exacto tras restart. No acredita dos personas físicas.
