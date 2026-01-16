const ITEM_HEIGHT = 80;
const BUFFER = 10;

const state = {
    logs: [],
    isLoading: false,
    params: {
        limit: 1000,
        search: '',
        level: '',
        session_id: '',
        subscriber_type: 'file' // Default,
    },
    scroll: {
        scrollTop: 0,
        containerHeight: 0
    }
};

const elems = {
    container: null,
    list: null,
    spacer: null,
    dialog: null,
    dialogBody: null
};

document.addEventListener('DOMContentLoaded', () => {
    elems.container = document.getElementById('scroll-container');
    elems.list = document.getElementById('log-list');
    elems.spacer = document.getElementById('virtual-spacer');
    elems.dialog = document.getElementById('dialog-overlay');
    elems.dialogBody = document.getElementById('dialog-body');

    // Bind Controls
    document.getElementById('btn-refresh').addEventListener('click', fetchLogs);

    ['search', 'level', 'session_id', 'client_id', 'limit', 'subscriber_type', 'context', 'before_context', 'after_context'].forEach(id => {
        const el = document.getElementById(id);
        if (!el) return;
        el.addEventListener('change', (e) => {
            state.params[id] = e.target.value;
        });
        // Allow Enter key for text inputs
        if (el.tagName === 'INPUT') {
            el.addEventListener('keydown', (e) => {
                state.params[id] = e.target.value;
                if (e.key === 'Enter') fetchLogs();
            });
        }
    });

    // Close Dialog
    document.getElementById('btn-close').addEventListener('click', () => {
        elems.dialog.classList.remove('open');
    });

    // Scroll Handler
    elems.container.addEventListener('scroll', onScroll);
    window.addEventListener('resize', onResize);

    // Initial Load
    onResize();
    fetchLogs();
});

async function fetchLogs() {
    state.isLoading = true;
    renderStatus("Loading...");

    // Filter empty params
    const activeParams = {};
    Object.keys(state.params).forEach(key => {
        if (state.params[key]) activeParams[key] = state.params[key];
    });

    const qs = new URLSearchParams(activeParams).toString();
    try {
        console.info('Fetching logs', state.params);
        const res = await fetch(`/v1/logs?${qs}`);
        if (!res.ok) throw new Error(await res.text());
        const data = await res.json();

        console.info('Received resonse', Array.isArray(data) ? data.length : 'No results');

        state.logs = data || [];

        console.info('Rendering virtual list');

        elems.spacer.style.height = `${state.logs.length * ITEM_HEIGHT}px`;
        renderVirtualList();


        // If no logs found
        if (state.logs.length === 0) {
            console.info('No logs to render');
            renderStatus("No logs found.");
        }

        console.info('Rendering completed');
        renderStatus("Done.");
    } catch (err) {
        console.error(err);
        renderStatus("Something went wrong");
        elems.list.innerHTML = `<div style="padding:20px; color:var(--err-color)">Error: ${err.message}</div>`;
    } finally {
        console.info('Loading completed');
        state.isLoading = false;
        renderVirtualList();
    }
}

function renderStatus(msg) {
    elems.list.innerHTML = `<div style="padding:20px; text-align:center">${msg}</div>`;
}

function onScroll(e) {
    state.scroll.scrollTop = e.target.scrollTop;
    requestAnimationFrame(renderVirtualList);
}

function onResize() {
    state.scroll.containerHeight = elems.container.clientHeight;
    requestAnimationFrame(renderVirtualList);
}

function renderVirtualList() {
    if (state.isLoading || state.logs.length === 0) return;

    const { scrollTop, containerHeight } = state.scroll;
    const startIndex = Math.max(0, Math.floor(scrollTop / ITEM_HEIGHT) - BUFFER);
    const endIndex = Math.min(state.logs.length, Math.ceil((scrollTop + containerHeight) / ITEM_HEIGHT) + BUFFER);

    let html = '';

    for (let i = startIndex; i < endIndex; i++) {
        const log = state.logs[i];
        const top = i * ITEM_HEIGHT;
        const timeStr = new Date(log.time).toLocaleString(undefined, {
            hour12: true,
            year: 'numeric',
            month: 'numeric',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
        const displayMsg = log.message || log.error || 'No message';
        const level = log.level || 'INFO';

        // Escape HTML
        const safeMsg = displayMsg.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");

        html += `
            <div class="log-item" style="transform: translateY(${top}px)" onclick="showDetails(${i})">
                <div class="log-ts">${timeStr}</div>
                <div><span class="log-lvl ${level}">${level}</span></div>
                <div class="log-msg">${safeMsg}</div>
            </div>
        `;
    }

    elems.list.innerHTML = html;
}

window.showDetails = function (index) {
    const log = state.logs[index];
    const fullJson = JSON.stringify(log, null, 2);
    const timeStr = new Date(log.time).toLocaleString(undefined, {
        year: 'numeric',
        month: 'numeric',
        day: 'numeric',
        hour: 'numeric',
        minute: 'numeric',
        second: 'numeric',
        fractionalSecondDigits: 3,
        hour12: true
    });

    // 1. Grid Metadata
    let content = `
    <div class="meta-grid">
        <div class="meta-item">
            <span class="meta-label">Time</span>
            <span class="meta-value">${timeStr}</span>
            <span class="meta-value" style="font-size:11px; opacity:0.6">${log.time}</span>
        </div>
        <div class="meta-item">
            <span class="meta-label">Level</span>
            <span class="meta-value"><span class="log-lvl ${log.level}">${log.level}</span></span>
        </div>
        <div class="meta-item">
            <span class="meta-label">Session ID</span>
            <input class="meta-value" readonly value="${log.session_id || '-'}" onclick="this.select()" />
        </div>
        <div class="meta-item">
            <span class="meta-label">Client ID</span>
            <input class="meta-value" readonly value="${log.client_id || '-'}" onclick="this.select()" />
        </div>
        <div class="meta-item">
            <span class="meta-label">Source</span>
            <span class="meta-value">${log.source || '-'}</span>
        </div>
    </div>
    `;

    // 2. Message
    content += `
    <div class="section">
        <div class="section-label">Message <span></span></div>
        <div class="code-block">${log.message || ''}</div>
    </div>
    `;

    // 3. Error
    if (log.error) {
        content += `
        <div class="section">
            <div class="section-label" style="color:var(--err-color)">Error</div>
            <div class="code-block" style="border-color:var(--err-color)">${log.error}</div>
        </div>
        `;
    }

    // 4. Stacktrace
    if (log.stacktrace) {
        content += `
        <div class="section">
            <div class="section-label">Stacktrace</div>
            <div class="code-block">${log.stacktrace}</div>
        </div>
        `;
    }

    // 5. Full JSON
    content += `
    <div class="section">
        <div class="section-label">Raw JSON</div>
        <div class="code-block">${fullJson}</div>
    </div>
    `;

    elems.dialogBody.innerHTML = content;

    // Copy button handler logic
    const btnCopy = document.getElementById('btn-copy');
    btnCopy.onclick = () => {
        navigator.clipboard.writeText(log.message || log.error || JSON.stringify(log));
        const originalText = btnCopy.textContent;
        btnCopy.textContent = "Copied!";
        setTimeout(() => btnCopy.textContent = originalText, 2000);
    };

    elems.dialog.classList.add('open');
};
