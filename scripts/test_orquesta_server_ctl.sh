#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$ROOT/scripts/orquesta_server_ctl.sh"
workdir="$(mktemp -d "${TMPDIR:-/tmp}/orquesta-server-ctl-test.XXXXXX")"
trap 'cleanup' EXIT

cleanup() {
  if [ -n "${fake_pid:-}" ] && kill -0 "$fake_pid" 2>/dev/null; then
    kill "$fake_pid" 2>/dev/null || true
  fi
  rm -rf "$workdir"
}

free_addr() {
  python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
host, port = s.getsockname()
s.close()
print(f"{host}:{port}")
PY
}

fake_bin="$workdir/orquesta-server-fake"
cat >"$fake_bin" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$@" >"$ORQUESTA_FAKE_ARGS_FILE"
python3 - <<'PY'
import http.server
import json
import os
import socketserver

host, port = os.environ["ORQUESTA_SERVER_ADDR"].rsplit(":", 1)

class Handler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path != "/api/status":
            self.send_response(404)
            self.end_headers()
            return
        body = json.dumps({"status": "running"}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args):
        pass

socketserver.TCPServer.allow_reuse_address = True
with socketserver.TCPServer((host, int(port)), Handler) as server:
    server.serve_forever()
PY
SH
chmod +x "$fake_bin"

run_start_case() {
  case_name="$1"
  expected_config="$2"
  root="$workdir/$case_name"
  mkdir -p "$root/state" "$root/project"
  args_file="$root/args.txt"
  addr="$(free_addr)"
  if [ "$expected_config" = "auto" ]; then
    expected_config="$root/project/orquesta.config.json"
    printf '%s\n' '{"schema_version":"orquesta_config.v0"}' >"$expected_config"
  fi

  ORQUESTA_CTL_HOME="$root" \
  ORQUESTA_CTL_BINARY="$fake_bin" \
  ORQUESTA_CTL_USER="$(id -un)" \
  ORQUESTA_CTL_ADDR="$addr" \
  ORQUESTA_CTL_WORKDIR="$root/project" \
  ORQUESTA_CTL_STARTUP_SLEEP="1" \
  ORQUESTA_FAKE_ARGS_FILE="$args_file" \
    bash "$script" start >/dev/null

  fake_pid="$(cat "$root/server.pid")"
  if [ "$expected_config" = "none" ]; then
    grep -qx 'run' "$args_file"
    if grep -q -- '--config' "$args_file"; then
      echo "config arg inesperado en $case_name" >&2
      exit 1
    fi
  else
    grep -qx 'run' "$args_file"
    grep -qx -- '--config' "$args_file"
    grep -qx "$expected_config" "$args_file"
  fi
  kill "$fake_pid"
  wait "$fake_pid" 2>/dev/null || true
  fake_pid=""
}

bash -n "$script"
run_start_case "without-config" "none"
run_start_case "with-auto-config" "auto"
