#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 1 ]]; then
	printf 'uso: %s [DIRECTORIO_CONECTOR_CANONICO]\n' "${0##*/}" >&2
	exit 64
fi

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo_root=$(cd -- "$script_dir/.." && pwd -P)
vendor_dir="$repo_root/vendor/github.com/aavidad/agente_microvm/conectores/orquesta"
readonly module='github.com/aavidad/agente_microvm/conectores/orquesta'
readonly version='v0.0.0-20260902163835-f17173c661e8'
readonly revision='f17173c661e80789b731b11a112e7411f48ba83b'
readonly module_sum='h1:vuFER9h1mN83LoZiofxSnwNpSpWoHQiE/xMdo0MNoZg='
readonly go_mod_sum='h1:Q/wNodoAhLQTJweffqspI2QT4XZ/4OJiGnyqhNwbT+s='
readonly tree_sha256='a9bfe50a4f272f3a48aacc5aa97e44cacc2f1604003a9d1f75f646cad7831db8'
readonly -a files=(
	README.md
	cliente.go
	concesion.go
	concesion_egreso_codec.go
	contrato.go
	credencial_codex.go
	errores.go
	intermediacion.go
	json_estricto.go
	perfil.go
	validacion_contenido.go
)

if ! awk -v module="$module" -v version="$version" '
	$1 == module {
		matches++
		if ($2 != version) {
			invalid = 1
		}
	}
	$1 == "replace" && $2 == module { invalid = 1 }
	END { exit !(matches == 1 && invalid == 0) }
' "$repo_root/go.mod"; then
	printf 'estado=error causa=version_base_go_mod_invalida\n' >&2
	exit 1
fi

if ! awk -v module="$module" -v version="$version" \
	-v module_sum="$module_sum" -v go_mod_sum="$go_mod_sum" '
	$1 == module && $2 == version && $3 == module_sum { module_matches++ }
	$1 == module && $2 == version "/go.mod" && $3 == go_mod_sum { go_mod_matches++ }
	$1 == module && ($2 == version || $2 == version "/go.mod") { selected++ }
	END { exit !(module_matches == 1 && go_mod_matches == 1 && selected == 2) }
' "$repo_root/go.sum"; then
	printf 'estado=error causa=sello_modulo_publicado_invalido\n' >&2
	exit 1
fi

if ! awk -v module="$module" -v version="$version" '
	BEGIN {
		prefix = "# " module " "
		expected = prefix version
	}
	index($0, prefix) == 1 {
		headers++
		if ($0 == expected) {
			exact++
		}
	}
	$0 == module { package_lines++ }
	END { exit !(headers == 1 && exact == 1 && package_lines == 1) }
' "$repo_root/vendor/modules.txt"; then
	printf 'estado=error causa=version_base_vendor_modules_invalida\n' >&2
	exit 1
fi

observed_names=$(find "$vendor_dir" -mindepth 1 -maxdepth 1 -printf '%f\n' | sort)
expected_names=$(printf '%s\n' "${files[@]}" | sort)
if [[ "$observed_names" != "$expected_names" ]]; then
	printf 'estado=error causa=inventario_vendor_publicado_invalido\n' >&2
	exit 1
fi
for file in "${files[@]}"; do
	vendor_file="$vendor_dir/$file"
	if [[ ! -f "$vendor_file" || -L "$vendor_file" || "$(stat -c '%h' -- "$vendor_file")" != 1 ]]; then
		printf 'estado=error causa=archivo_vendor_invalido archivo=%s\n' "$file" >&2
		exit 1
	fi
done
actual_tree_sha256=$({
	for file in "${files[@]}"; do
		vendor_file="$vendor_dir/$file"
		printf '%s\t%s\t%s\n' "$file" "$(stat -c '%s' -- "$vendor_file")" \
			"$(sha256sum -- "$vendor_file" | cut -d' ' -f1)"
	done
} | sort | sha256sum | cut -d' ' -f1)
if [[ "$actual_tree_sha256" != "$tree_sha256" ]]; then
	printf 'estado=error causa=arbol_vendor_publicado_divergente\n' >&2
	exit 1
fi

if [[ $# -eq 1 ]]; then
	if [[ ! -d "$1" ]]; then
		printf 'estado=error causa=directorio_canonico_ausente\n' >&2
		exit 1
	fi
	canonical_dir=$(cd -- "$1" && pwd -P)
	for file in "${files[@]}"; do
		canonical_file="$canonical_dir/$file"
		vendor_file="$vendor_dir/$file"
		if [[ ! -f "$canonical_file" ]]; then
			printf 'estado=error causa=archivo_canonico_ausente archivo=%s\n' "$file" >&2
			exit 1
		fi
		if ! cmp --silent -- "$canonical_file" "$vendor_file"; then
			printf 'estado=error causa=archivo_canonico_divergente archivo=%s\n' "$file" >&2
			exit 1
		fi
	done
fi

printf 'estado=ok modulo=%s version=%s revision=%s arbol_sha256=%s overlay_local=false\n' \
	"$module" "$version" "$revision" "$tree_sha256"
