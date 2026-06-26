package orquestaweb

func runQueueHTMLV0() string {
	return operatorPageHTMLV0("Cola Orquesta", "Consulta cola y prioridad por contrato publico.", "run-queue-root", `
      <div class="row">
        <label>Cola <input id="queue-ref" value="global"></label>
        <label>Limite <input id="queue-limit" type="number" value="30"></label>
      </div>
      <button onclick="loadQueue()">Actualizar cola</button>
      <script>
        async function loadQueue() {
          const q = encodeURIComponent(document.getElementById('queue-ref').value || 'global');
          const l = encodeURIComponent(document.getElementById('queue-limit').value || '30');
          show(await getJSON('/run-queue?action=rank&queue_ref=' + q + '&limit=' + l));
        }
        loadQueue();
      </script>`)
}

func runControlHTMLV0() string {
	return operatorPageHTMLV0("Control de run", "Pausa, reanuda, para o cancela una run sin salir de la web.", "run-control-root", `
      <div class="row">
        <label>Run ref <input id="run-ref" placeholder="run-ref..."></label>
        <label>Accion <select id="action"><option value="pause">pause</option><option value="resume">resume</option><option value="stop">stop</option><option value="cancel">cancel</option></select></label>
      </div>
      <label>Motivo <input id="reason" value="operador web"></label>
      <label><input id="forced" type="checkbox"> Forzado</label>
      <button onclick="sendControl()">Aplicar control</button>
      <script>
        async function sendControl() {
          const id = 'web-run-control-' + Date.now().toString(36);
          show(await postJSON('/run-control', {
            request_id: id,
            correlation_id: id,
            locale: 'es-ES',
            action: document.getElementById('action').value,
            run_ref: document.getElementById('run-ref').value.trim(),
            requested_by: 'orquesta-web',
            reason: document.getElementById('reason').value.trim(),
            forced: document.getElementById('forced').checked
          }));
        }
      </script>`)
}

func directorStatsHTMLV0() string {
	return operatorPageHTMLV0("Stats de Director", "Consulta progreso, agentes, cierre y evidencias publicas de una run.", "director-stats-root", `
      <label>Run ref <input id="run-ref" placeholder="run-ref..."></label>
      <div class="row">
        <button onclick="loadStats()">Consultar stats</button>
        <button onclick="observeGoal()">Observar goal</button>
      </div>
      <script>
        async function loadStats() {
          const run = encodeURIComponent(document.getElementById('run-ref').value.trim());
          show(await getJSON('/director-stats?include_agent_progress=true&include_agent_usage=true&run_ref=' + run));
        }
        async function observeGoal() {
          const id = 'web-director-stats-goal-observe-' + Date.now().toString(36);
          const run = document.getElementById('run-ref').value.trim();
          show(await postJSON('`+WebDirectorStatsGoalObserveEndpointV0+`', {
            request_id: id,
            correlation_id: id,
            run_ref: run,
            requested_by: 'orquesta-web-director-stats'
          }));
        }
      </script>`)
}

func operatorPageHTMLV0(title string, subtitle string, rootID string, body string) string {
	return `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>` + title + `</title>
  <style>
    :root { color-scheme: dark; --bg:#101316; --panel:#181d22; --line:#34404a; --text:#eef4f8; --muted:#aab7c3; --accent:#7cd4fd; }
    * { box-sizing: border-box; }
    body { margin:0; min-height:100%; padding:22px; background:var(--bg); color:var(--text); font:14px/1.5 ui-sans-serif, system-ui, sans-serif; }
    main { max-width:1120px; margin:0 auto; display:grid; gap:14px; }
    header { display:flex; justify-content:space-between; gap:12px; align-items:flex-start; }
    h1 { margin:0; font-size:clamp(28px,4vw,48px); letter-spacing:-.04em; }
    a { color:var(--accent); }
    nav { display:flex; flex-wrap:wrap; gap:8px; }
    nav a, button { border:1px solid var(--line); background:#202832; color:var(--text); border-radius:8px; padding:8px 10px; text-decoration:none; cursor:pointer; }
    .panel { border:1px solid var(--line); background:var(--panel); border-radius:14px; padding:16px; }
    .row { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:10px; }
    label { display:grid; gap:5px; color:var(--muted); margin-bottom:10px; }
    input, select { border:1px solid var(--line); border-radius:8px; background:#0e1216; color:var(--text); padding:9px; font:inherit; }
    pre { margin:0; min-height:260px; max-height:560px; overflow:auto; border:1px solid #24303a; border-radius:10px; background:#090d11; padding:12px; white-space:pre-wrap; word-break:break-word; }
    .hint { color:var(--muted); }
    @media (max-width:760px){ body{padding:14px} header,.row{display:grid; grid-template-columns:1fr} }
  </style>
</head>
<body>
  <main id="` + rootID + `">
    <header>
      <div><h1>` + title + `</h1><p class="hint">` + subtitle + `</p></div>
      <nav><a href="/">Inicio</a><a href="/ops">Ops</a><a href="/autoprogramming">Autoprogramar</a></nav>
    </header>
    <section class="panel">` + body + `</section>
    <section class="panel"><pre id="output">{}</pre></section>
  </main>
  <script>
    const output = document.getElementById('output');
    function show(value) { output.textContent = JSON.stringify(value || {}, null, 2); }
    async function getJSON(url) {
      const response = await fetch(url, {headers:{'Accept':'application/json'}});
      return await response.json().catch(() => ({estado:'error', errores_publicos:[{code:'respuesta_no_json'}]}));
    }
    async function postJSON(url, payload) {
      const response = await fetch(url, {method:'POST', headers:{'Content-Type':'application/json','Accept':'application/json'}, body:JSON.stringify(payload)});
      return await response.json().catch(() => ({estado:'error', errores_publicos:[{code:'respuesta_no_json'}]}));
    }
  </script>
</body>
</html>
`
}
