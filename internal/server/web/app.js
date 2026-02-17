// ========================================
// Loom Dashboard — Client-side JavaScript
// ========================================

(function () {
    'use strict';

    const MAX_LOG_ENTRIES = 1000;
    const STATS_POLL_INTERVAL = 1000;
    const WS_RECONNECT_DELAY = 2000;

    // --- State ---
    const activeFilters = new Set(['INFO', 'WARN', 'ERROR', 'FATAL', 'DEBUG']);
    let ws = null;
    let statsTimer = null;

    // --- DOM refs ---
    const logContainer = document.getElementById('log-container');
    const logEmpty = document.getElementById('log-empty');
    const statusDot = document.querySelector('.status-dot');
    const statusText = document.getElementById('status-text');
    const autoscrollCheckbox = document.getElementById('autoscroll');

    // --- WebSocket ---
    function connectWebSocket() {
        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${location.host}/ws`;

        ws = new WebSocket(url);

        ws.onopen = () => {
            setConnectionStatus('connected', 'Connected');
        };

        ws.onmessage = (event) => {
            try {
                const entry = JSON.parse(event.data);
                addLogEntry(entry);
            } catch (e) {
                console.error('Failed to parse log entry:', e);
            }
        };

        ws.onclose = () => {
            setConnectionStatus('disconnected', 'Disconnected');
            setTimeout(connectWebSocket, WS_RECONNECT_DELAY);
        };

        ws.onerror = () => {
            setConnectionStatus('disconnected', 'Error');
        };
    }

    function setConnectionStatus(state, text) {
        statusDot.className = `status-dot status-dot--${state}`;
        statusText.textContent = text;
    }

    // --- Log Entries ---
    function addLogEntry(entry) {
        // Hide empty state.
        if (logEmpty) logEmpty.style.display = 'none';

        const el = document.createElement('div');
        el.className = `log-entry log-entry--${entry.level}`;
        el.dataset.level = entry.level;

        // Check if visible based on current filters.
        if (!activeFilters.has(entry.level)) {
            el.classList.add('log-entry--hidden');
        }

        const time = formatTime(entry.timestamp);
        const source = entry.source ? entry.source.split(/[/\\]/).pop() : '';

        el.innerHTML = `
            <span class="log-entry__time">${time}</span>
            <span class="log-entry__level log-entry__level--${entry.level}">${entry.level.padEnd(5)}</span>
            <span class="log-entry__source" title="${escapeHtml(entry.source)}">${escapeHtml(source)}</span>
            <span class="log-entry__message">${escapeHtml(entry.message)}</span>
        `;

        logContainer.appendChild(el);

        // Trim old entries.
        while (logContainer.children.length > MAX_LOG_ENTRIES + 1) {
            const first = logContainer.querySelector('.log-entry');
            if (first) first.remove();
        }

        // Auto-scroll.
        if (autoscrollCheckbox.checked) {
            logContainer.scrollTop = logContainer.scrollHeight;
        }
    }

    function formatTime(timestamp) {
        try {
            const d = new Date(timestamp);
            return d.toLocaleTimeString('en-GB', { hour12: false });
        } catch {
            return '--:--:--';
        }
    }

    function escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    // --- Filters ---
    window.toggleFilter = function (level) {
        if (level === 'all') {
            // Toggle all on/off.
            const allActive = activeFilters.size === 5;
            activeFilters.clear();
            if (!allActive) {
                ['INFO', 'WARN', 'ERROR', 'FATAL', 'DEBUG'].forEach(l => activeFilters.add(l));
            }
        } else {
            if (activeFilters.has(level)) {
                activeFilters.delete(level);
            } else {
                activeFilters.add(level);
            }
        }
        updateFilterUI();
        applyFilters();
    };

    function updateFilterUI() {
        document.querySelectorAll('.filter-btn[data-level]').forEach(btn => {
            const level = btn.dataset.level;
            if (level === 'all') {
                btn.classList.toggle('filter-btn--active', activeFilters.size === 5);
            } else {
                btn.classList.toggle('filter-btn--active', activeFilters.has(level));
            }
        });
    }

    function applyFilters() {
        document.querySelectorAll('.log-entry').forEach(el => {
            const level = el.dataset.level;
            if (activeFilters.has(level)) {
                el.classList.remove('log-entry--hidden');
            } else {
                el.classList.add('log-entry--hidden');
            }
        });
    }

    window.clearLogs = function () {
        document.querySelectorAll('.log-entry').forEach(el => el.remove());
        if (logEmpty) logEmpty.style.display = 'flex';
    };

    // --- Stats Polling ---
    function pollStats() {
        fetch('/api/stats')
            .then(res => res.json())
            .then(data => {
                document.getElementById('stat-eps').textContent = data.eps.toFixed(1);
                document.getElementById('stat-total').textContent = formatNumber(data.total_events);
                document.getElementById('stat-errors').textContent = formatNumber(data.level_counts?.ERROR || 0);
                document.getElementById('stat-warnings').textContent = formatNumber(data.level_counts?.WARN || 0);
                document.getElementById('stat-files').textContent = data.files_watched;
                document.getElementById('stat-uptime').textContent = data.uptime;
            })
            .catch(() => { }); // Silently ignore on disconnect.
    }

    function formatNumber(n) {
        if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
        if (n >= 1000) return (n / 1000).toFixed(1) + 'K';
        return n.toString();
    }

    // --- Init ---
    connectWebSocket();
    statsTimer = setInterval(pollStats, STATS_POLL_INTERVAL);
    pollStats();
})();
