package orquestaweb

const opsDashboardHTMLChunk1V0 = `      if (!n) return '-';
      const units = ['B','KB','MB','GB','TB'];
      let v = n; let i = 0;
      while (v >= 1024 && i < units.length - 1) { v = v / 1024; i++; }
      return (i === 0 ? v.toFixed(0) : v.toFixed(1)) + ' ' + units[i];
    }
    function statusClass(value) {
      const v = String(value || '').toLowerCase();
      if (['ok','running','ready','completed','accepted','validada','validated'].includes(v)) return 'ok';
      if (v.includes('block') || v.includes('fail') || v.includes('error') || v.includes('stalled')) return 'bad';
      if (v.includes('pending') || v.includes('wait') || v.includes('unknown')) return 'warn';
      return 'info';
    }
    function statusPill(value) {
      const label = value || '-';
      return '<span class="status ' + statusClass(label) + '">' + esc(label) + '</span>';
    }
    function tableCell(label, content, attrs) {
      return '<td data-label="' + esc(label) + '"' + (attrs ? ' ' + attrs : '') + '>' + content + '</td>';
    }
    function formatTokens(value) {
      const n = Number(value || 0);
      if (!Number.isFinite(n) || n <= 0) return '-';
      if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M tok';
      if (n >= 1000) return (Math.round(n / 100) / 10) + 'k tok';
      return String(n) + ' tok';
    }
    function agentUsageLabel(agent) {
      const parts = [];
      if (agent.capacity_level) parts.push(agent.capacity_level);
      if (agent.quota_status) parts.push(agent.quota_status);
      const tokens = formatTokens(agent.total_tokens);
      if (tokens !== '-') parts.push(tokens);
      return parts.length ? parts.join(' · ') : '-';
    }
    function detailKey(runRef, agentRef, name) {
      return [runRef || '-', agentRef || '-', name || '-'].join('|');
    }
    function captureDetailOpenState() {
      document.querySelectorAll('#selected-detail details[data-detail-key]').forEach(function(node) {
        const key = node.getAttribute('data-detail-key') || '';
        if (!key) return;
        if (node.open) openDetailKeys.add(key); else openDetailKeys.delete(key);
      });
    }
    document.addEventListener('toggle', function(event) {
      const node = event.target;
      if (!node || node.tagName !== 'DETAILS') return;
      const key = node.getAttribute('data-detail-key') || '';
      if (!key) return;
      if (node.open) openDetailKeys.add(key); else openDetailKeys.delete(key);
    }, true);
    function idleReasonText(value) {
      const reason = String(value || '');
      switch (reason) {
        case 'capacity_pending_skips':
          return 'hay skips pendientes del supervisor; no lanza automejora nueva aunque la cola siga activa';
        case 'capacity_full':
          return 'cola en objetivo de capacidad; no se crean tareas nuevas';
        case 'capacity_cooldown':
          return 'esperando ventana de cooldown antes de crear mas tareas';
        case 'capacity_free':
          return 'capacidad libre para crear nuevas tareas';
        case 'idle_window_waiting':
          return 'esperando ventana de inactividad';
        default:
          return reason || '-';
      }
    }
    function bar(percent, bad) {
      const p = Math.max(0, Math.min(100, Number(percent || 0)));
      const cls = bad ? 'bad' : (p >= 90 ? 'ok' : (p >= 40 ? 'warn' : ''));
      return '<div class="bar" title="' + p + '%"><span class="' + cls + '" style="width:' + p + '%"></span></div>';
    }
    function validationState(run) {
      const status = String(run.status || '').toLowerCase();
      const refs = (run.evidence_refs || []).join(' ').toLowerCase();
      if (status.includes('blocked') || status.includes('failed') || status.includes('error')) return 'bloqueada';
      if (refs.includes('prepare-run-enqueued') || refs.includes('run-coordinator-executed') || refs.includes('validated')) return 'validada';
      return 'pendiente';
    }
    function titleFromRef(value) {
      let raw = String(value || '').trim();
      if (!raw) return 'Tarea sin nombre';
      raw = raw
        .replace(/^request-ref-autoprogramming-backlog-/i, '')
        .replace(/^request-ref-/i, '')
        .replace(/^run-ref-/i, '')
        .replace(/^task-ref-/i, '')
        .replace(/^agent-ref-task-autoprogramming-/i, 'autoprogramming-')
        .replace(/-[a-f0-9]{8,}$/i, '');
      raw = raw.replace(/[_-]+/g, ' ').replace(/\s+/g, ' ').trim();
      return raw.replace(/\bt(\d+)\b/gi, 'T$1');
    }
    function queueStableID(item) {
      const runRef = String((item || {}).run_ref || '');
      const appRef = String((item || {}).app_ref || '');
      if (runRef) return 'queue:' + runRef;
      if (appRef) return 'queue-app:' + appRef;
      return 'queue:sin-ref';
    }
    function queueTitle(item) {
      return (item && item.title) || titleFromRef((item || {}).run_ref);
    }
    function queueDetail(item) {
      if (item && item.detail) return item.detail;
      return ['run_ref=' + ((item || {}).run_ref || '-'), 'app_ref=' + ((item || {}).app_ref || '-'), 'status=' + ((item || {}).status || '-')].join('; ');
    }
    function taskIDFromRef(value) {
      const raw = String(value || '');
      const match = raw.match(/(?:^|-)t(\d+)(?:-|$)/i);
      if (match) return 'T' + match[1];
      const compact = raw.replace(/^request-ref-/, '').replace(/^run-ref-/, '').replace(/^task-ref-/, '');
      if (!compact) return 'SIN-ID';
      return compact.length <= 12 ? compact : compact.slice(0, 6) + '-' + compact.slice(-5);
    }
    function runTitle(run) {
      if (!run) return 'Tarea sin seleccionar';
      const runtimeAgent = runtimeAgentForRun(run.run_ref);
      const runtimeTask = ((runtimeAgent || {}).task || {});
      if (runtimeTask.title) return runtimeTask.title;
      if (runtimeTask.objective_summary) return runtimeTask.objective_summary;
      if (run.summary) return run.summary;
      if (run.task_title) return run.task_title;
      if (run.title) return run.title;
      return titleFromRef(run.run_ref || run.task_ref || run.agent_ref);
    }
    function taskTitleCell(title, subtitle, id) {
      return '<span class="task-title"><span class="task-id">' + esc(id || 'SIN-ID') + '</span>' + esc(title || '-') + '</span>' +
        '<span class="task-subtitle mono">' + esc(subtitle || '') + '</span>';
    }
    function runByRef(runRef) {
      return (lastSnapshot.runs || []).find(function(run) { return run.run_ref === runRef; }) ||
        (lastSnapshot.ranked || []).find(function(run) { return run.run_ref === runRef; }) ||
        (completedHistory || []).find(function(run) { return run.run_ref === runRef; }) ||
        null;
    }
    function runtimeAgentForRun(runRef) {
      const detail = runtimeDetails[runRef] || {};
      const agents = detail.agents || [];
      if (!agents.length) return null;
      if (selectedAgentRef) {
        const selected = agents.find(function(agent) { return agent.agent_ref === selectedAgentRef; });
        if (selected) return selected;
      }
      return agents[0];
    }
    function loadCompletedHistory() {
      try {
        const raw = localStorage.getItem('orquesta.ops.completed.v1');
        const parsed = raw ? JSON.parse(raw) : [];
        return Array.isArray(parsed) ? parsed.slice(0, 200) : [];
      } catch (_) {
        return [];
      }
    }
    function saveCompletedHistory() {
      try {
        localStorage.setItem('orquesta.ops.completed.v1', JSON.stringify(completedHistory.slice(0, 200)));
      } catch (_) {}
    }
    function loadStableOrders() {
      try {
        const raw = localStorage.getItem('orquesta.ops.stable_order.v1');
        const parsed = raw ? JSON.parse(raw) : {};
        return {
          runs: parsed && parsed.runs && typeof parsed.runs === 'object' ? parsed.runs : {},
          agents: parsed && parsed.agents && typeof parsed.agents === 'object' ? parsed.agents : {},
          queue: parsed && parsed.queue && typeof parsed.queue === 'object' ? parsed.queue : {}
        };
      } catch (_) {
        return {runs: {}, agents: {}, queue: {}};
      }
    }
    function loadPendingEnvEdits() {
      try {
        const raw = localStorage.getItem('orquesta.ops.pending_env.v1');
        const parsed = raw ? JSON.parse(raw) : {};
        return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {};
      } catch (_) {
        return {};
      }
    }
    function savePendingEnvEdits() {
      try {
        localStorage.setItem('orquesta.ops.pending_env.v1', JSON.stringify(pendingEnvEdits || {}));
      } catch (_) {}
    }
    function nextStableIndex(bucket) {
      return Object.keys(bucket || {}).reduce(function(max, key) {
        const value = Number(bucket[key]);
        return Number.isFinite(value) && value >= max ? value + 1 : max;
      }, 1);
    }
    function stableOrder(bucketName, key) {
      const bucket = stableOrders[bucketName] || {};
      stableOrders[bucketName] = bucket;
      const safeKey = String(key || '');
      if (!safeKey) return Number.MAX_SAFE_INTEGER;
      if (bucket[safeKey] == null) {
        bucket[safeKey] = stableCounters[bucketName]++;
        stableOrdersDirty = true;
      }
      return Number(bucket[safeKey]);
    }
    function stableSortByFirstSeen(items, bucketName, keyFn) {
      return (items || []).slice().sort(function(a, b) {
        const orderA = stableOrder(bucketName, keyFn(a));
        const orderB = stableOrder(bucketName, keyFn(b));
        if (orderA !== orderB) return orderA - orderB;
        const keyA = String(keyFn(a) || '');
        const keyB = String(keyFn(b) || '');
        return keyA.localeCompare(keyB);
      });
    }
    function saveStableOrders() {
      if (!stableOrdersDirty) return;
      stableOrdersDirty = false;
      try {
        localStorage.setItem('orquesta.ops.stable_order.v1', JSON.stringify(stableOrders));
      } catch (_) {}
    }
    async function fetchJSON(url, opts) {
      const res = await fetch(url, opts || {});
      if (!res.ok) throw new Error(url + ' -> HTTP ' + res.status);
      return await res.json();
    }
    async function fetchRunStats(runRef) {
      const body = {
        request_id: 'ops-' + Date.now() + '-' + Math.random().toString(16).slice(2),
        run_ref: runRef,
        include_process_refs: false,
        include_agent_progress: true,
        include_agent_usage: true
      };
      return await fetchJSON('/api/v0/director/stats', {
        method: 'POST',
        headers: {'content-type': 'application/json'},
        body: JSON.stringify(body)
      });
    }
    async function fetchRuntimeDetail(runRef) {
      return await fetchJSON('/api/v0/ops/agent-runtime-detail', {
        method: 'POST',
        headers: {'content-type': 'application/json'},
        body: JSON.stringify({
          request_id: 'ops-runtime-detail-' + Date.now(),
          run_ref: runRef,
          include_logs: true,
          max_file_bytes: 200000
        })
      });`
