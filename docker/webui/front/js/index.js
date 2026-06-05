// ---- Helpers ----
function api(url, opts = {}) {
    return fetch(url, {
        method: opts.method || 'GET',
        headers: { 'Content-Type': 'application/json' },
        body: opts.body ? JSON.stringify(opts.body) : undefined,
    }).then(r => r.json());
}

// ---- Status ----
async function refreshStatus() {
    try {
        const data = await api('/status');
        const dot = document.getElementById('statusDot');
        const text = document.getElementById('statusText');
        const btn = document.getElementById('lockBtn');

        text.textContent = data.status || 'Unknown';
        dot.className = 'dot';
        switch (data.status) {
            case 'RUNNING':
                break; // green (default)
            case 'LOADING':
            case 'LAUNCHING':
                dot.classList.add('warning');
                break;
            default:
                if (data.status) dot.classList.add('unhealthy');
        }

        if (data.locked) {
            btn.textContent = 'Unblock inference';
            btn.className = 'btn btn-lock';
        } else {
            btn.textContent = 'Lock inference';
            btn.className = 'btn btn-unlock';
        }

        updateModelButtons(data.current_model);
        updateRunningModel(data.current_model);
    } catch (_) {
        document.getElementById('statusText').textContent = 'Error';
        document.getElementById('statusDot').className = 'dot unhealthy';
    }
}

async function toggleLock() {
    const btn = document.getElementById('lockBtn');
    const isLocked = btn.textContent === 'Unblock inference';
    const endpoint = isLocked ? '/set-lock/off' : '/set-lock/on';

    btn.textContent = isLocked ? 'Unblocking…' : 'Blocking…';
    btn.style.pointerEvents = 'none';

    try {
        await api(endpoint, { method: 'POST' });
        await refreshStatus();
        btn.style.pointerEvents = '';
    } catch (_) {
        btn.textContent = isLocked ? 'Unblock inference' : 'Lock inference';
        btn.style.pointerEvents = '';
    }
}

// ---- Running Model ----
function updateRunningModel(currentModel) {
    const el = document.getElementById('runningModel');
    if (currentModel && currentModel.length > 0) {
        el.innerHTML = `<span class="name">${escapeHtml(currentModel)}</span>`;
        el.onclick = () => {
            const url = window.location.href.replace(/:\d{2,5}\b/, ':8080');
            window.open(url, '_blank');
        };
        el.style.cursor = 'pointer';
    } else {
        el.innerHTML = '<span class="none">No model loaded</span>';
        el.onclick = null;
        el.style.cursor = '';
    }
}

// ---- Model Table ----
async function refreshModels() {
    try {
        const data = await api('/models');
        const tbody = document.getElementById('modelTableBody');
        tbody.innerHTML = '';

        if (!data.models || data.models.length === 0) {
            tbody.innerHTML = '<tr><td colspan="2" style="text-align:center;color:var(--text-muted);">No models available</td></tr>';
            return;
        }

        data.models.forEach(m => {
            const tr = document.createElement('tr');

            const tdName = document.createElement('td');
            const nameEl = document.createElement('span');
            nameEl.className = 'model-name';
            nameEl.textContent = m.name;
            if (m.content && window.innerWidth > 480) {
                const tooltip = document.createElement('div');
                tooltip.className = 'model-tooltip';
                const pre = document.createElement('pre');
                pre.textContent = m.content;
                tooltip.appendChild(pre);
                nameEl.appendChild(tooltip);

                nameEl.addEventListener('mouseenter', () => {
                    const rect = nameEl.getBoundingClientRect();
                    const spaceAbove = rect.top;
                    const spaceBelow = window.innerHeight - rect.bottom;
                    const showBelow = spaceBelow > spaceAbove;
                    tooltip.classList.remove('arrow-top', 'arrow-bottom');
                    tooltip.classList.add(showBelow ? 'arrow-bottom' : 'arrow-top');
                    tooltip.style.bottom = showBelow ? 'auto' : '100%';
                    tooltip.style.top = showBelow ? '100%' : 'auto';
                });
            }
            tdName.appendChild(nameEl);

            const tdAction = document.createElement('td');
            const btn = document.createElement('button');
            btn.dataset.name = m.name;
            btn.className = 'btn';
            btn.textContent = 'Select Preset';
            btn.onclick = () => loadModel(btn, m.name);

            tdAction.appendChild(btn);
            tr.appendChild(tdName);
            tr.appendChild(tdAction);
            tbody.appendChild(tr);
        });
    } catch (_) {
        document.getElementById('modelTableBody').innerHTML =
            '<tr><td colspan="2" style="text-align:center;color:var(--danger);">Error loading models</td></tr>';
    }
}

function updateModelButtons(currentModel) {
    const buttons = document.querySelectorAll('#modelTableBody .btn');
    buttons.forEach(btn => {
        if (btn.dataset.name === currentModel) {
            btn.className = 'btn running';
            btn.textContent = 'Selected';
            btn.onclick = null;
        } else {
            btn.className = 'btn';
            btn.textContent = 'Select Preset';
            btn.onclick = () => loadModel(btn, btn.dataset.name);
        }
    });
}

async function loadModel(btn, name) {
    btn.classList.add('loading');
    btn.textContent = 'Loading…';

    try {
        await api('/load-model', {
            method: 'POST',
            body: { name }
        });
        btn.classList.remove('loading');
        btn.textContent = 'Loading…';
        await refreshStatus();
    } catch (_) {
        btn.classList.remove('loading');
        btn.textContent = 'Select Preset';
    }
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

// ---- Init ----
refreshStatus();
refreshModels();

// Poll every 3s
setInterval(refreshStatus, 3000);
