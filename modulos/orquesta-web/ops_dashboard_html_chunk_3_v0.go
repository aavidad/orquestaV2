package orquestaweb

const opsDashboardHTMLChunk3V0 = `          currently_visible: true,
          last_seen_ms: nowMs,
          last_seen_at: now
        });
      });
      Object.keys(agentDisplayCache || {}).forEach(function(key) {
        if (!freshKeys.has(key)) agentDisplayCache[key].currently_visible = false;
      });
      const allowedRuns = new Set((runs || []).map(function(run) { return run.run_ref; }).filter(Boolean));
      const merged = Object.values(agentDisplayCache || {}).filter(function(agent) {
        if (!allowedRuns.has(agent.run_ref)) return false;
        if (agent.currently_visible !== false) return true;
        return Number(agent.last_seen_ms || 0) >= nowMs - graceMs;
      });
      return merged.slice(-500);
    }
    function agentPercent(agent, progress) {
      const status = String(agent.status || '').toLowerCase();
      const pstatus = String(progress.status || '').toLowerCase();
      if (status.includes('completed') || status.includes('delivered')) return 100;
      if (agent.needs_attention || pstatus.includes('stalled')) return 35;
      if (agent.in_flight && pstatus) return 65;
      if (agent.in_flight) return 45;
      return 0;
    }
    function filterValue(id) {
      const node = byId(id);
      return node ? String(node.value || '').toLowerCase().trim() : '';
    }
    function rowText(value) {
      return Object.keys(value || {}).map(function(key) {
        const item = value[key];
        return Array.isArray(item) ? item.join(' ') : String(item == null ? '' : item);
      }).join(' ').toLowerCase();
    }
    function matchesText(value, query) {
      if (!query) return true;
      return rowText(value).indexOf(query) >= 0;
    }
    function matchesStatus(status, filter) {
      const value = String(status || '').toLowerCase();
      if (!filter) return true;
      if (filter === 'running') return value.includes('running') || value.includes('ready') || value.includes('progress') || value.includes('wait');
      if (filter === 'completed') return value.includes('completed') || value.includes('closed') || value.includes('delivered');
      if (filter === 'blocked') return value.includes('block') || value.includes('fail') || value.includes('error') || value.includes('stalled');
      if (filter === 'open') return !matchesStatus(status, 'completed');
      if (filter === 'closed') return matchesStatus(status, 'completed');
      return value.includes(filter);
    }
    function renderCurrentSnapshot() {
      renderProjects(lastSnapshot.runs || []);
      renderTasks(lastSnapshot.tasks || []);
      renderAgents(lastSnapshot.agents || []);
      renderQueue(lastSnapshot.ranked || []);
      renderCompletedHistory();
    }
    function renderProjects(runs) {
      const query = filterValue('filter-runs');
      const status = filterValue('filter-status');
      const validation = filterValue('filter-validation');
      runs = (runs || []).filter(function(run) {
        return matchesText(run, query) &&
          matchesStatus(run.status || run.closure_status || '', status) &&
          (!validation || String(run.validation || '').toLowerCase().includes(validation));
      });
      if (!runs.length) {
        byId('projects-body').innerHTML = '<tr><td colspan="6" class="empty">Sin proyectos activos</td></tr>';
        return;
      }
      byId('projects-body').innerHTML = runs.map(function(run) {
        const percent = Number(run.percent_complete || 0);
        const bad = run.blocked || Number(run.stalled_agents || 0) > 0 || Number(run.agents_failed || 0) > 0;
        const selected = run.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(run.run_ref || '') + '"' + selected + ' onclick="selectRun(\'' + esc(run.run_ref || '') + '\')">' +
          '<td title="' + esc(runTitle(run)) + '">' + taskTitleCell(runTitle(run), run.app_ref || '-', run.task_id || taskIDFromRef(run.run_ref)) + '</td>' +
          '<td class="mono" title="' + esc(run.run_ref || '-') + '">' + esc(shortRef(run.run_ref || '-')) + '</td>' +
          '<td>' + statusPill(run.status || 'unknown') + '</td>' +
          '<td>' + bar(percent, bad) + '<span class="sub">' + percent + '% · ' + esc(run.current_phase || '-') + '</span></td>' +
          '<td>' + esc(String(run.agents_in_flight || 0)) + ' / ' + esc(String(run.agents_started || 0)) + '</td>' +
          '<td>' + statusPill(run.validation || 'pendiente') + '</td>' +
        '</tr>';
      }).join('');
    }
    function renderTasks(tasks) {
      const query = filterValue('filter-tasks');
      const status = filterValue('filter-task-status');
      const agentFilter = filterValue('filter-task-agent');
      tasks = (tasks || []).filter(function(task) {
        const hasAgent = !!task.agent_ref;
        return matchesText(task, query) &&
          matchesStatus(task.status || task.progress_status || '', status) &&
          (agentFilter !== 'with_agent' || hasAgent) &&
          (agentFilter !== 'without_agent' || !hasAgent);
      });
      if (!tasks.length) {
        byId('tasks-body').innerHTML = '<tr><td colspan="5" class="empty">Sin tareas que coincidan con los filtros</td></tr>';
        return;
      }
      byId('tasks-body').innerHTML = tasks.map(function(task) {
        const selected = task.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(task.run_ref || '') + '"' + selected + ' onclick="selectRun(\'' + jsArg(task.run_ref || '') + '\', \'' + jsArg(task.agent_ref || '') + '\')">' +
          '<td title="' + esc(task.task_ref || '-') + '">' + taskTitleCell(task.title || titleFromRef(task.task_ref), task.app_ref || shortRef(task.task_ref || ''), task.task_id || taskIDFromRef(task.task_ref)) + '</td>' +
          '<td class="mono" title="' + esc(task.run_ref || '-') + '">' + esc(shortRef(task.run_ref || '-')) + '</td>' +
          '<td>' + statusPill(task.status || 'unknown') + '</td>' +
          '<td class="mono" title="' + esc(task.agent_ref || '-') + '">' + esc(shortRef(task.agent_ref || '-')) + '</td>' +
          '<td>' + esc(task.progress_status || '-') + ' · ticks ' + esc(String(task.no_progress_ticks || 0)) + '</td>' +
        '</tr>';
      }).join('');
    }
    function renderAgents(agents) {
      const query = filterValue('filter-agents');
      const status = filterValue('filter-agent-status');
      const signal = filterValue('filter-agent-signal');
      agents = (agents || []).filter(function(agent) {
        return matchesText(agent, query) &&
          (status !== 'attention' ? matchesStatus(agent.status || agent.progress_status || '', status) : !!agent.needs_attention) &&
          (signal !== 'progressing' || String(agent.progress_status || '').toLowerCase().includes('progress')) &&
          (signal !== 'stalled' || Number(agent.no_progress_ticks || 0) > 0 || String(agent.progress_status || '').toLowerCase().includes('stall'));
      });
      if (!agents.length) {
        byId('agents-body').innerHTML = '<tr><td colspan="6" class="empty">Sin agentes observados</td></tr>';
        return;
      }
      byId('agents-body').innerHTML = agents.map(function(agent) {
        const selected = agent.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(agent.run_ref || '') + '" data-agent-ref="' + esc(agent.agent_ref || '') + '"' + selected + ' onclick="selectRun(\'' + esc(agent.run_ref || '') + '\', \'' + esc(agent.agent_ref || '') + '\')">' +
          '<td class="mono" title="' + esc(agent.agent_ref || '-') + '">' + esc(shortRef(agent.agent_ref || '-')) + '</td>' +
          '<td class="mono" title="' + esc(agent.run_ref || '-') + '">' + esc(shortRef(agent.run_ref || '-')) + '</td>' +
          '<td>' + statusPill(agent.status || 'unknown') + '</td>' +
          '<td title="' + esc(agent.task_ref || '-') + '">' + taskTitleCell(agent.task_title || titleFromRef(agent.task_ref), shortRef(agent.task_ref || ''), agent.task_id || taskIDFromRef(agent.task_ref)) + '</td>' +
          '<td>' + bar(agent.percent, agent.needs_attention) + '<span class="sub">' + agent.percent + '%</span></td>' +
          '<td>' + esc(agent.progress_status || '-') + ' · ticks ' + esc(String(agent.no_progress_ticks || 0)) + (agent.currently_visible === false ? ' · visto ' + esc(agent.last_seen_at || '-') : '') + '</td>' +
        '</tr>';
      }).join('');
    }
    function renderQueue(ranked) {
      if (!ranked.length) {
        byId('queue-body').innerHTML = '<tr><td colspan="7" class="empty">Sin tareas en cola</td></tr>';
        return;
      }
      byId('queue-body').innerHTML = ranked.map(function(item) {
        const title = queueTitle(item);
        const detail = queueDetail(item);
        const stableID = item.stable_id || queueStableID(item);
        const selected = item.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(item.run_ref || '') + '" data-stable-id="' + esc(stableID) + '"' + selected + ' onclick="selectRun(\'' + jsArg(item.run_ref || '') + '\')">' +
          '<td>' + esc(item.rank || '-') + '</td>' +
          '<td title="' + esc(detail) + '">' + taskTitleCell(title, detail, taskIDFromRef(item.run_ref)) + '</td>' +
          '<td title="' + esc(item.app_ref || '-') + '">' + esc(shortRef(item.app_ref || '-')) + '</td>' +
          '<td>' + statusPill(item.status || 'unknown') + '</td>' +
          '<td>' + esc(item.priority_score || 0) + '</td>' +
          '<td>' + statusPill(validationState(item)) + '</td>' +
          '<td><div class="row-actions">' +
            '<button class="small" type="button" title="Subir prioridad" onclick="event.stopPropagation(); setQueueRowPriority(\'' + jsArg(item.run_ref || '') + '\', 25)">↑</button>' +
            '<button class="small" type="button" title="Bajar prioridad" onclick="event.stopPropagation(); setQueueRowPriority(\'' + jsArg(item.run_ref || '') + '\', -25)">↓</button>' +
            '<button class="small" type="button" title="Pausar run" onclick="event.stopPropagation(); controlQueueRow(\'' + jsArg(item.run_ref || '') + '\', \'pause\')">Ⅱ</button>' +
            '<button class="small" type="button" title="Reanudar run" onclick="event.stopPropagation(); controlQueueRow(\'' + jsArg(item.run_ref || '') + '\', \'resume\')">▶</button>' +
          '</div></td>' +
        '</tr>';
      }).join('');
    }
    function selectRun(runRef, agentRef) {
      selectedRunRef = runRef || '';
      selectedAgentRef = agentRef || '';
      controlMessage = '';
      renderProjects(lastSnapshot.runs || []);
      renderTasks(lastSnapshot.tasks || []);
      renderAgents(lastSnapshot.agents || []);
      renderQueue(lastSnapshot.ranked || []);
      renderCompletedHistory();
      renderSelectedDetail();
      ensureRuntimeDetail(selectedRunRef);
    }
    function updateCompletedHistory(runs, ranked) {
      const activeRefs = new Set((ranked || []).map(function(item) { return item.run_ref; }).filter(Boolean));
      const knownRefs = new Set(completedHistory.map(function(item) { return item.run_ref; }));
      (runs || []).forEach(function(run) {
        if (!run.run_ref || knownRefs.has(run.run_ref)) return;
        const terminal = isTerminalRun(run);
        const disappeared = !activeRefs.has(run.run_ref) && Number(run.percent_complete || 0) >= 100;
        if (!terminal && !disappeared) return;
        completedHistory.unshift({
          run_ref: run.run_ref,
          app_ref: run.app_ref,
          task_id: run.task_id || taskIDFromRef(run.run_ref),
          title: runTitle(run),
          status: terminal || 'salio_de_cola',
          percent_complete: run.percent_complete || 0,
          observed_at: new Date().toLocaleString()
        });
        knownRefs.add(run.run_ref);
      });
      completedHistory = completedHistory.slice(0, 200);
      saveCompletedHistory();
    }
    function isTerminalRun(run) {
      const status = String(run.status || '').toLowerCase();
      const closure = String(run.closure_status || '').toLowerCase();
      if (closure === 'closed') return 'closed';
      if (status.includes('completed') || status.includes('closed')) return status;
      if (Number(run.tasks_total || 0) > 0 && Number(run.tasks_closed || 0) >= Number(run.tasks_total || 0)) return 'completed';
      return '';
    }
    function renderCompletedHistory() {
      if (!completedHistory.length) {
        byId('completed-body').innerHTML = '<tr><td colspan="6" class="empty">Aún no hay tareas completadas observadas por este panel</td></tr>';
        return;
      }
      byId('completed-body').innerHTML = completedHistory.slice(0, 80).map(function(item) {
        const selected = item.run_ref === selectedRunRef ? ' class="selected"' : '';
        return '<tr data-run-ref="' + esc(item.run_ref || '') + '"' + selected + ' onclick="selectRun(\'' + esc(item.run_ref || '') + '\')">' +
          '<td>' + taskTitleCell(item.title || titleFromRef(item.run_ref), item.app_ref || '-', item.task_id || taskIDFromRef(item.run_ref)) + '</td>' +
          '<td class="mono" title="' + esc(item.run_ref || '-') + '">' + esc(shortRef(item.run_ref || '-')) + '</td>' +
          '<td title="' + esc(item.app_ref || '-') + '">' + esc(shortRef(item.app_ref || '-')) + '</td>' +
          '<td>' + statusPill(item.status || 'completed') + '</td>' +
          '<td>' + bar(item.percent_complete || 100, false) + '<span class="sub">' + esc(String(item.percent_complete || 100)) + '%</span></td>' +
          '<td>' + esc(item.observed_at || '-') + '</td>' +
        '</tr>';
      }).join('');
    }
    function listInline(values) {
      const items = (values || []).filter(Boolean);
      if (!items.length) return '<span class="sub">-</span>';
      return items.map(function(value) { return '<div class="mono">' + esc(value) + '</div>'; }).join('');
    }
    function prettyJSON(value) {
      if (!value) return '';
      try { return JSON.stringify(value, null, 2); } catch (_) { return String(value); }
    }
    function detailsBlock(title, content, key) {
      if (!content) return '';
      const safeKey = String(key || title || '');
      const open = openDetailKeys.has(safeKey) ? ' open' : '';
      return '<details data-detail-key="' + esc(safeKey) + '"' + open + '><summary>' + esc(title) + '</summary><pre>' + esc(content) + '</pre></details>';
    }
    function runtimeFileContent(agent, fileName) {
      const file = ((agent || {}).files || []).find(function(item) { return item.name === fileName; });
      return file && file.exists ? file.content : '';
    }
    function runtimeDetailHTML(run) {
      const loading = runtimeDetailLoading[run.run_ref];
      const detail = runtimeDetails[run.run_ref];
      if (loading && !detail) return '<div class="detail-section"><div class="sub">Cargando detalle runtime...</div></div>';
      if (!detail) return '<div class="detail-section"><div class="sub">Detalle runtime pendiente.</div></div>';
      if (detail.estado === 'error') {
        const issue = ((detail.issues || [])[0] || {}).message || 'detalle runtime no disponible';
        return '<div class="detail-section"><div class="error">' + esc(issue) + '</div></div>';
      }
      const agent = runtimeAgentForRun(run.run_ref);
      if (!agent) return '<div class="detail-section"><div class="sub">Sin runtime de agente para este run.</div></div>';
      const packet = agent.agent_packet || {};
      const task = packet.task || {};
      const skills = agent.skills || {};
      const promptText = agent.prompt_text || runtimeFileContent(agent, 'agent_prompt.txt');
      const packetText = runtimeFileContent(agent, 'agent_packet.json') || prettyJSON(packet);
      const ackText = runtimeFileContent(agent, 'agent_ack.json') || prettyJSON(agent.ack);
      const logs = ['codex_last_message.txt', 'codex_stdout.log', 'codex_stderr.log', 'director_decisions.json'].map(function(name) {
        return detailsBlock(name, runtimeFileContent(agent, name), detailKey(run.run_ref, agent.agent_ref, name));
      }).join('');
      return '<div class="detail-section">' +
        '<div class="kv"><div class="k">Qué se consigue</div><div>' + esc(task.objective || task.title || runTitle(run)) + '</div></div>' +
        '<div class="kv"><div class="k">Agente seleccionado</div><div class="mono">' + esc(agent.agent_ref || '-') + '</div></div>' +
        '<div class="kv"><div class="k">Write-set</div><div>' + listInline(task.write_set || []) + '</div></div>' +
        '<div class="kv"><div class="k">Tests requeridos</div><div>' + listInline(task.required_tests || []) + '</div></div>' +
        '<div class="kv"><div class="k">Skills/policies</div><div>' +
          '<div>Caveman: ' + esc(skills.caveman_requested ? 'si' : 'no') + ' · compacto: ' + esc(skills.compact_protocol ? 'si' : 'no') + ' · subagentes max: ' + esc(skills.max_child_agents || '-') + '</div>' +
          listInline(skills.policies || []) +
        '</div></div>' +
        detailsBlock('Prompt exacto', promptText, detailKey(run.run_ref, agent.agent_ref, 'prompt')) +
        detailsBlock('agent_packet.json', packetText, detailKey(run.run_ref, agent.agent_ref, 'agent_packet.json')) +
        detailsBlock('agent_ack.json', ackText, detailKey(run.run_ref, agent.agent_ref, 'agent_ack.json')) +
        logs +
      '</div>';
    }
    function renderSelectedDetail() {
      captureDetailOpenState();
      const run = runByRef(selectedRunRef);
      if (!run) {
        byId('selected-detail').innerHTML = '<div class="empty">Selecciona una tarea o agente.</div>';
        return;
      }
      const runAgents = (lastSnapshot.agents || []).filter(function(agent) { return agent.run_ref === run.run_ref; });
      const tasks = run.tasks || [];
      const priority = run.priority_score == null ? 10 : run.priority_score;
      const isCompletedSnapshot = !!run.observed_at;
      const taskRows = tasks.length ? tasks.map(function(task) {
        return '<div class="kv"><div class="k">' + esc(task.status || '-') + '</div><div>' +
          '<div class="task-title">' + esc(task.summary || titleFromRef(task.task_ref)) + '</div>' +
          '<div class="task-subtitle mono">' + esc(task.task_ref || '-') + '</div>' +
          '</div></div>';
      }).join('') : '<div class="sub">Sin subtareas detalladas publicadas por stats.</div>';
      const agentRows = runAgents.length ? runAgents.map(function(agent) {
        return '<div class="kv"><div class="k">' + esc(agent.status || '-') + '</div><div>' +
          '<div class="task-title">' + esc(agent.task_title || titleFromRef(agent.agent_ref)) + '</div>' +
          '<div class="task-subtitle mono">' + esc(shortRef(agent.agent_ref || '-')) + '</div>' +`
